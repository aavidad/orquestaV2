//go:build linux

package firecrackeraudit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

const testLimit = 2048

const emptyDigestLength = len(EmptyDigest)

func TestEmptyDigestIsCanonicalCompileTimeConstant(t *testing.T) {
	if emptyDigestLength != 64 || EmptyDigest != Digest(nil) {
		t.Fatalf("EmptyDigest=%q", EmptyDigest)
	}
}

func TestThirtyTwoStreamsRunConcurrentlyWithoutCollision(t *testing.T) {
	root := privateTempDir(t)
	store := newTestStore(t, root, 4)
	t.Cleanup(func() { _ = store.Close() })

	type result struct {
		ref  string
		head string
		path string
		err  error
	}
	results := make(chan result, 32)
	var ready sync.WaitGroup
	ready.Add(32)
	start := make(chan struct{})
	for index := 0; index < 32; index++ {
		go func(index int) {
			ref := Digest([]byte(fmt.Sprintf("stream-%02d", index)))
			operation := Digest([]byte(fmt.Sprintf("operation-%02d", index)))
			subject := Digest([]byte(fmt.Sprintf("subject-%02d", index)))
			ready.Done()
			<-start
			stream, err := store.Begin(ref)
			if err == nil {
				err = stream.Append(startEvent(operation, subject))
			}
			if err == nil {
				err = stream.Append(successEvent(operation, subject, []byte{byte(index)}))
			}
			var head, path string
			if err == nil {
				head, path, err = stream.Close(OutcomeSucceeded)
			}
			results <- result{ref: ref, head: head, path: path, err: err}
		}(index)
	}
	ready.Wait()
	close(start)

	paths := make(map[string]struct{}, 32)
	for index := 0; index < 32; index++ {
		got := <-results
		if got.err != nil {
			t.Fatalf("stream %s: %v", got.ref, got.err)
		}
		if filepath.Base(got.path) != got.ref+finalSuffix {
			t.Fatalf("path=%q ref=%s", got.path, got.ref)
		}
		if _, duplicate := paths[got.path]; duplicate {
			t.Fatalf("duplicate path %q", got.path)
		}
		paths[got.path] = struct{}{}
		content, err := os.ReadFile(got.path)
		if err != nil {
			t.Fatal(err)
		}
		verified, err := Verify(bytes.NewReader(content), got.ref, testLimit, 4)
		if err != nil {
			t.Fatal(err)
		}
		if verified.Events != 2 || verified.Pending || verified.HeadDigest != got.head {
			t.Fatalf("verification=%+v head=%s", verified, got.head)
		}
		assertModeAndLinks(t, got.path, 0o400, 1)
	}
}

func TestStoreCloseDoesNotInvalidateActiveCleanup(t *testing.T) {
	root := privateTempDir(t)
	store := newTestStore(t, root, 4)
	ref := Digest([]byte("active-cleanup"))
	stream, err := store.Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin(Digest([]byte("after-close"))); ErrorCodeOf(err) != CodeNotOpen {
		t.Fatalf("begin after store close err=%v", err)
	}
	operation := Digest([]byte("cleanup"))
	subject := Digest([]byte("microvm"))
	if err := stream.Append(startEvent(operation, subject)); err != nil {
		t.Fatal(err)
	}
	if err := stream.Append(Event{
		OperationRef: operation,
		SubjectRef:   subject,
		Transition:   Failed,
		Diagnostic: Diagnostic{
			Code: DiagnosticCleanupFailed, Digest: EmptyDigest,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := stream.Close(OutcomeFailed); err != nil {
		t.Fatal(err)
	}
}

func TestSameRefAdmitsOneBeginAndCloseNeverClobbers(t *testing.T) {
	root := privateTempDir(t)
	store := newTestStore(t, root, 4)
	t.Cleanup(func() { _ = store.Close() })
	ref := Digest([]byte("same-ref"))

	type beginResult struct {
		stream *Stream
		err    error
	}
	results := make(chan beginResult, 16)
	var wait sync.WaitGroup
	for index := 0; index < 16; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			stream, err := store.Begin(ref)
			results <- beginResult{stream: stream, err: err}
		}()
	}
	wait.Wait()
	close(results)
	var winner *Stream
	for result := range results {
		if result.err == nil {
			if winner != nil {
				t.Fatal("more than one Begin succeeded")
			}
			winner = result.stream
			continue
		}
		if ErrorCodeOf(result.err) != CodeInUse {
			t.Fatalf("begin err=%v", result.err)
		}
	}
	if winner == nil {
		t.Fatal("no Begin succeeded")
	}
	operation := Digest([]byte("operation"))
	subject := Digest([]byte("subject"))
	if err := winner.Append(startEvent(operation, subject)); err != nil {
		t.Fatal(err)
	}
	if err := winner.Append(successEvent(operation, subject, nil)); err != nil {
		t.Fatal(err)
	}

	finalPath := filepath.Join(root, ref+finalSuffix)
	sentinel := []byte("must-not-be-replaced")
	if err := os.WriteFile(finalPath, sentinel, 0o400); err != nil {
		t.Fatal(err)
	}
	if _, _, err := winner.Close(OutcomeSucceeded); ErrorCodeOf(err) != CodeFinalExists {
		t.Fatalf("close collision err=%v", err)
	}
	got, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(got, sentinel) {
		t.Fatalf("final clobbered err=%v content=%q", err, got)
	}
	assertModeAndLinks(t, filepath.Join(root, ref+openSuffix), 0o600, 1)
	if err := winner.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverOpenContinuesValidChainAndRejectsPartial(t *testing.T) {
	root := privateTempDir(t)
	store := newTestStore(t, root, 4)
	t.Cleanup(func() { _ = store.Close() })
	ref := Digest([]byte("recover-valid"))
	operation := Digest([]byte("operation"))
	subject := Digest([]byte("subject"))
	stream, err := store.Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Append(startEvent(operation, subject)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecoverOpen(ref); ErrorCodeOf(err) != CodeInUse {
		t.Fatalf("concurrent recovery err=%v", err)
	}
	if err := stream.Release(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Release(); err != nil {
		t.Fatalf("idempotent release err=%v", err)
	}

	recovered, err := store.RecoverOpen(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.Append(successEvent(operation, subject, nil)); err != nil {
		t.Fatal(err)
	}
	head, finalPath, err := recovered.Close(OutcomeSucceeded)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(bytes.NewReader(content), ref, testLimit, 4)
	if err != nil || verified.HeadDigest != head {
		t.Fatalf("verify=%+v err=%v", verified, err)
	}

	partialRef := Digest([]byte("recover-partial"))
	partialStream, err := store.Begin(partialRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := partialStream.Append(startEvent(operation, subject)); err != nil {
		t.Fatal(err)
	}
	if err := partialStream.Release(); err != nil {
		t.Fatal(err)
	}
	openPath := filepath.Join(root, partialRef+openSuffix)
	file, err := os.OpenFile(openPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("{")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(openPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecoverOpen(partialRef); ErrorCodeOf(err) != CodePartial {
		t.Fatalf("recover partial err=%v", err)
	}
	after, err := os.ReadFile(openPath)
	if err != nil || !bytes.Equal(after, before) {
		t.Fatalf("partial changed err=%v", err)
	}
}

func TestCrashRecoveryEmptyOpenAndChmodFrontier(t *testing.T) {
	root := privateTempDir(t)
	store := newTestStore(t, root, 4)
	t.Cleanup(func() { _ = store.Close() })

	emptyRef := Digest([]byte("empty-open"))
	empty, err := store.Begin(emptyRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := empty.Release(); err != nil {
		t.Fatal(err)
	}
	recoveredEmpty, err := store.RecoverOpen(emptyRef)
	if err != nil {
		t.Fatal(err)
	}
	operation := Digest([]byte("operation"))
	subject := Digest([]byte("subject"))
	if err := recoveredEmpty.Append(startEvent(operation, subject)); err != nil {
		t.Fatal(err)
	}
	if err := recoveredEmpty.Append(successEvent(operation, subject, nil)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := recoveredEmpty.Close(OutcomeSucceeded); err != nil {
		t.Fatal(err)
	}

	chmodRef := Digest([]byte("chmod-frontier"))
	chmodStream, err := store.Begin(chmodRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := chmodStream.Append(startEvent(operation, subject)); err != nil {
		t.Fatal(err)
	}
	if err := chmodStream.Append(successEvent(operation, subject, nil)); err != nil {
		t.Fatal(err)
	}
	chmodStream.afterChmodBeforeRename = func() error { return errors.New("injected-after-chmod") }
	if _, _, err := chmodStream.Close(OutcomeSucceeded); ErrorCodeOf(err) != CodeIO {
		t.Fatalf("injected chmod frontier err=%v", err)
	}
	openPath := filepath.Join(root, chmodRef+openSuffix)
	assertModeAndLinks(t, openPath, 0o400, 1)
	if _, _, err := chmodStream.Close(OutcomeSucceeded); ErrorCodeOf(err) != CodeBlocked {
		t.Fatalf("blocked close err=%v", err)
	}
	if err := chmodStream.Append(Event{
		OperationRef: operation,
		SubjectRef:   subject,
		Transition:   Failed,
		Diagnostic:   Diagnostic{Code: DiagnosticCleanupFailed, Digest: EmptyDigest},
	}); ErrorCodeOf(err) != CodeBlocked {
		t.Fatalf("blocked cleanup bypass err=%v", err)
	}
	if _, err := store.RecoverOpen(chmodRef); ErrorCodeOf(err) != CodeInUse {
		t.Fatalf("live chmod frontier lease err=%v", err)
	}
	if err := chmodStream.Release(); err != nil {
		t.Fatal(err)
	}
	recoveredChmod, err := store.RecoverOpen(chmodRef)
	if err != nil {
		t.Fatal(err)
	}
	assertModeAndLinks(t, openPath, 0o600, 1)
	if _, _, err := recoveredChmod.Close(OutcomeSucceeded); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverFinalCompletesRenameCrashDurably(t *testing.T) {
	root := privateTempDir(t)
	store := newTestStore(t, root, 4)
	t.Cleanup(func() { _ = store.Close() })
	ref := Digest([]byte("rename-frontier"))
	operation := Digest([]byte("operation"))
	subject := Digest([]byte("subject"))
	stream, err := store.Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Append(startEvent(operation, subject)); err != nil {
		t.Fatal(err)
	}
	if err := stream.Append(successEvent(operation, subject, nil)); err != nil {
		t.Fatal(err)
	}
	stream.afterRenameBeforeSync = func() error { return errors.New("injected-after-rename") }
	if _, _, err := stream.Close(OutcomeSucceeded); ErrorCodeOf(err) != CodeIO {
		t.Fatalf("injected rename frontier err=%v", err)
	}
	if _, _, err := stream.Close(OutcomeSucceeded); ErrorCodeOf(err) != CodeBlocked {
		t.Fatalf("blocked close err=%v", err)
	}
	if _, err := store.RecoverFinal(ref); ErrorCodeOf(err) != CodeInUse {
		t.Fatalf("live final lease err=%v", err)
	}
	if err := stream.Release(); err != nil {
		t.Fatal(err)
	}
	final, err := store.RecoverFinal(ref)
	if err != nil {
		t.Fatal(err)
	}
	if final.Outcome != OutcomeSucceeded || final.Events != 2 ||
		!validHex(final.HeadDigest) || filepath.Base(final.Path) != ref+finalSuffix {
		t.Fatalf("final=%+v", final)
	}
	replayed, err := store.RecoverFinal(ref)
	if err != nil || replayed != final {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	if _, err := store.RecoverOpen(ref); ErrorCodeOf(err) != CodeFinalExists {
		t.Fatalf("recover open after final err=%v", err)
	}
}

func TestSchemaRedactionTamperAndFilesystemAttacks(t *testing.T) {
	t.Run("schema-redaction-tamper", func(t *testing.T) {
		root := privateTempDir(t)
		store := newTestStore(t, root, 4)
		ref := Digest([]byte("schema"))
		operation := Digest([]byte("operation"))
		subject := Digest([]byte("subject"))
		stream, err := store.Begin(ref)
		if err != nil {
			t.Fatal(err)
		}
		if err := stream.Append(startEvent(operation, subject)); err != nil {
			t.Fatal(err)
		}
		if err := stream.Append(Event{
			OperationRef: operation,
			SubjectRef:   subject,
			Transition:   Failed,
			Diagnostic: Diagnostic{
				Code: DiagnosticCode("/tmp/free text"), Digest: EmptyDigest,
			},
		}); ErrorCodeOf(err) != CodeEvent {
			t.Fatalf("free diagnostic err=%v", err)
		}
		if err := stream.Append(successEvent(operation, subject, []byte("bounded"))); err != nil {
			t.Fatal(err)
		}
		_, path, err := stream.Close(OutcomeSucceeded)
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range [][]byte{
			[]byte(root), []byte(`"path"`), []byte(`"pid"`), []byte(`"uid"`),
			[]byte(`"gid"`), []byte(`"nonce"`), []byte(`"args"`), []byte(`"env"`),
		} {
			if bytes.Contains(content, forbidden) {
				t.Fatalf("forbidden data %q in %s", forbidden, content)
			}
		}
		tampered := append([]byte(nil), content...)
		index := bytes.Index(tampered, []byte(`"seq":2`))
		tampered[index+6] = '3'
		if _, err := Verify(bytes.NewReader(tampered), ref, testLimit, 4); ErrorCodeOf(err) != CodeIntegrity {
			t.Fatalf("tamper err=%v", err)
		}
		lines := bytes.Split(bytes.TrimSuffix(content, []byte{'\n'}), []byte{'\n'})
		var object map[string]any
		if err := json.Unmarshal(lines[0], &object); err != nil {
			t.Fatal(err)
		}
		object["path"] = "/forbidden"
		lines[0], _ = json.Marshal(object)
		unknown := append(bytes.Join(lines, []byte{'\n'}), '\n')
		if _, err := Verify(bytes.NewReader(unknown), ref, testLimit, 4); ErrorCodeOf(err) != CodeIntegrity {
			t.Fatalf("unknown field err=%v", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		root := privateTempDir(t)
		store := newTestStore(t, root, 2)
		ref := Digest([]byte("symlink"))
		target := filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, ref+openSuffix)); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RecoverOpen(ref); ErrorCodeOf(err) != CodeSecurity {
			t.Fatalf("symlink err=%v", err)
		}
	})

	t.Run("hardlink", func(t *testing.T) {
		root, store, ref := validOpenFile(t, "hardlink")
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Link(filepath.Join(root, ref+openSuffix), link); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RecoverOpen(ref); ErrorCodeOf(err) != CodeSecurity {
			t.Fatalf("hardlink err=%v", err)
		}
	})

	t.Run("mode", func(t *testing.T) {
		root, store, ref := validOpenFile(t, "mode")
		if err := os.Chmod(filepath.Join(root, ref+openSuffix), 0o640); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RecoverOpen(ref); ErrorCodeOf(err) != CodeSecurity {
			t.Fatalf("mode err=%v", err)
		}
	})

	t.Run("root-symlink", func(t *testing.T) {
		target := privateTempDir(t)
		link := filepath.Join(t.TempDir(), "root")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := New(link, uint32(os.Geteuid()), testLimit, 2); ErrorCodeOf(err) != CodeSecurity {
			t.Fatalf("root symlink err=%v", err)
		}
	})
}

func newTestStore(t *testing.T, root string, maxEvents uint64) *Store {
	t.Helper()
	store, err := New(root, uint32(os.Geteuid()), testLimit, maxEvents)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func privateTempDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func startEvent(operation, subject string) Event {
	return Event{
		OperationRef: operation,
		SubjectRef:   subject,
		Transition:   Started,
		Diagnostic:   Diagnostic{Code: DiagnosticNone, Digest: EmptyDigest},
	}
}

func successEvent(operation, subject string, output []byte) Event {
	return Event{
		OperationRef: operation,
		SubjectRef:   subject,
		Transition:   Succeeded,
		Diagnostic: Diagnostic{
			Code: DiagnosticNone, Bytes: uint64(len(output)), Digest: Digest(output),
		},
	}
}

func validOpenFile(t *testing.T, label string) (string, *Store, string) {
	t.Helper()
	root := privateTempDir(t)
	store := newTestStore(t, root, 2)
	ref := Digest([]byte(label))
	stream, err := store.Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Append(startEvent(Digest([]byte("op")), Digest([]byte("subject")))); err != nil {
		t.Fatal(err)
	}
	if err := stream.Release(); err != nil {
		t.Fatal(err)
	}
	return root, store, ref
}

func assertModeAndLinks(t *testing.T, path string, mode os.FileMode, links uint64) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode().Perm() != mode || stat.Nlink != links {
		t.Fatalf("mode=%v links=%d", info.Mode(), stat.Nlink)
	}
}
