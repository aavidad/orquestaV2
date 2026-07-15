package local

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"orquesta/internal/credentials"
)

const (
	testCredential = credentials.CredentialRef("credential:test")
	testOwner      = credentials.OwnerRef("owner:test")
	testScope      = credentials.ScopeRef("scope:test")
	testPurpose    = credentials.PurposeRef("purpose:test")
)

func TestCASAndIdempotencyAcrossStoreInstances(t *testing.T) {
	path := testStorePath(t)
	first := testOpen(t, path, nil)
	second := testOpen(t, path, nil)
	t.Cleanup(func() { _ = first.Close(); _ = second.Close() })
	created := testCreate(t, first, testCredential, "request:create", []byte("initial-credential-value"))
	replayed, err := first.Create(context.Background(), testCreateRequest(t, testCredential, "request:create", []byte("initial-credential-value")))
	if err != nil || !replayed.Replayed || replayed.Metadata.Version != created.Metadata.Version {
		t.Fatalf("create replay = %+v, %v", replayed, err)
	}
	_, err = first.Create(context.Background(), testCreateRequest(t, testCredential, "request:create", []byte("different-value")))
	if !credentials.HasErrorCode(err, credentials.ErrorIdempotencyConflict) {
		t.Fatalf("changed replay = %v", err)
	}

	starts := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for index, store := range []*Store{first, second} {
		go func(index int, store *Store) {
			ready.Done()
			<-starts
			secret := testSecret(t, []byte{byte('a' + index), '-', 'r', 'o', 't', 'a', 't', 'e', 'd'})
			_, err := store.Rotate(context.Background(), credentials.RotateRequest{
				ActorRef: "actor:test", RequestRef: "request:rotate:" + string(rune('a'+index)),
				CredentialRef: testCredential, OwnerRef: testOwner, ExpectedVersion: 1, Material: secret,
			})
			results <- err
		}(index, store)
	}
	ready.Wait()
	close(starts)
	succeeded, conflicted := 0, 0
	for range 2 {
		switch err := <-results; {
		case err == nil:
			succeeded++
		case credentials.HasErrorCode(err, credentials.ErrorVersionConflict):
			conflicted++
		default:
			t.Fatalf("rotate = %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("CAS succeeded=%d conflicted=%d", succeeded, conflicted)
	}
}

func TestUseIsReentrantAndRevocationDominatesReplay(t *testing.T) {
	store := testOpen(t, testStorePath(t), nil)
	t.Cleanup(func() { _ = store.Close() })
	secondRef := credentials.CredentialRef("credential:second")
	testCreate(t, store, testCredential, "request:create:first", []byte("first-secret"))
	testCreate(t, store, secondRef, "request:create:second", []byte("second-secret"))

	output, receipts, err := credentials.WithChildEnvironment(context.Background(), store, credentials.ChildEnvironmentRequest{
		ActorRef: "actor:test", RequestRef: "request:child", OwnerRef: testOwner,
		ScopeRef: testScope, PurposeRef: testPurpose, PublicEnvironment: map[string]string{"PATH": "/bin"},
		Bindings: []credentials.ChildBinding{{EnvironmentName: "FIRST", CredentialRef: testCredential}, {EnvironmentName: "SECOND", CredentialRef: secondRef}},
	}, func(environment []string) (credentials.ChildOutput, error) {
		joined := []byte{}
		for _, entry := range environment {
			joined = append(joined, entry...)
			joined = append(joined, '\n')
		}
		if !bytes.Contains(joined, []byte("FIRST=first-secret")) || !bytes.Contains(joined, []byte("SECOND=second-secret")) {
			return credentials.ChildOutput{}, errors.New("missing exact bindings")
		}
		return credentials.ChildOutput{Result: []byte("ok")}, nil
	})
	if err != nil || string(output.Result) != "ok" || len(receipts) != 2 {
		t.Fatalf("nested use = output=%+v receipts=%+v err=%v", output, receipts, err)
	}
	useReplay := testUseRequest(testCredential, "request:use-replay", 0)
	var retained credentials.Secret
	firstReceipt, err := store.Use(context.Background(), useReplay, func(secret credentials.Secret) error { retained = secret; return nil })
	if err != nil || bytes.Contains(retained.Bytes(), []byte("first-secret")) {
		t.Fatalf("callback-scoped secret survived: receipt=%+v err=%v", firstReceipt, err)
	}
	secondReceipt, err := store.Use(context.Background(), useReplay, func(secret credentials.Secret) error { return nil })
	if err != nil || secondReceipt != firstReceipt {
		t.Fatalf("use replay = %+v, %v", secondReceipt, err)
	}
	useReplay.Version = 1
	called := false
	if _, err := store.Use(context.Background(), useReplay, func(credentials.Secret) error { called = true; return nil }); !credentials.HasErrorCode(err, credentials.ErrorIdempotencyConflict) || called {
		t.Fatalf("changed use replay err=%v called=%v", err, called)
	}

	use := testUseRequest(testCredential, "request:use-before-revoke", 1)
	if _, err := store.Use(context.Background(), use, func(credentials.Secret) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Revoke(context.Background(), credentials.RevokeRequest{
		ActorRef: "actor:test", RequestRef: "request:revoke", CredentialRef: testCredential,
		OwnerRef: testOwner, ExpectedVersion: 1, Reason: "test",
	}); err != nil {
		t.Fatal(err)
	}
	called = false
	_, err = store.Use(context.Background(), use, func(credentials.Secret) error { called = true; return nil })
	if !credentials.HasErrorCode(err, credentials.ErrorRevoked) || called {
		t.Fatalf("revoked replay err=%v called=%v", err, called)
	}
}

func TestUseNeverReturnsCredentialMaterialFromConsumerErrors(t *testing.T) {
	store := testOpen(t, testStorePath(t), nil)
	t.Cleanup(func() { _ = store.Close() })
	material := []byte("consumer-error-secret")
	testCreate(t, store, testCredential, "request:create-error-secret", material)
	receipt, err := store.Use(context.Background(), testUseRequest(testCredential, "request:use-error-secret", 1), func(secret credentials.Secret) error {
		message := string(secret.Bytes())
		secret.Destroy()
		return errors.New(message)
	})
	if receipt.CredentialRef != testCredential || !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || bytes.Contains([]byte(err.Error()), material) {
		t.Fatalf("unsafe consumer result receipt=%+v err=%v", receipt, err)
	}

	sentinel := errors.New("ordinary consumer failure")
	_, err = store.Use(context.Background(), testUseRequest(testCredential, "request:use-safe-error", 1), func(credentials.Secret) error { return sentinel })
	if !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) || errors.Is(err, sentinel) || errors.Unwrap(err) != nil {
		t.Fatalf("consumer cause escaped stable error: %v", err)
	}

	var contaminatedCause error
	_, err = store.Use(context.Background(), testUseRequest(testCredential, "request:use-typed-error", 1), func(secret credentials.Secret) error {
		contaminatedCause = errors.New(string(secret.Bytes()))
		return credentials.WrapError(credentials.ErrorChildEnvironment, "consumer", contaminatedCause)
	})
	if !credentials.HasErrorCode(err, credentials.ErrorChildEnvironment) || errors.Is(err, contaminatedCause) || errors.Unwrap(err) != nil ||
		bytes.Contains([]byte(err.Error()), material) {
		t.Fatalf("typed consumer cause escaped or was retained: %v", err)
	}

	var hiddenCause error
	_, err = store.Use(context.Background(), testUseRequest(testCredential, "request:use-hidden-cause", 1), func(secret credentials.Secret) error {
		hiddenCause = errors.New(string(secret.Bytes()))
		return hiddenCauseError{cause: hiddenCause}
	})
	if !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) || errors.Is(err, hiddenCause) || errors.Unwrap(err) != nil ||
		bytes.Contains([]byte(err.Error()), material) {
		t.Fatalf("hidden consumer cause escaped or was retained: %v", err)
	}

	for name, sentinel := range map[string]error{"canceled": context.Canceled, "deadline": context.DeadlineExceeded} {
		t.Run(name, func(t *testing.T) {
			_, err := store.Use(context.Background(), testUseRequest(testCredential, "request:use-"+name, 1), func(credentials.Secret) error {
				return sentinel
			})
			if err != sentinel || errors.Unwrap(err) != nil {
				t.Fatalf("context sentinel changed or gained a cause: %v", err)
			}
		})
	}
}

type hiddenCauseError struct{ cause error }

func (err hiddenCauseError) Error() string { return "safe callback failure" }
func (err hiddenCauseError) Unwrap() error { return err.cause }

func TestDurableProjectionGateIsAtomicAcrossCredentialLifecycle(t *testing.T) {
	t.Run("create_raw_and_encoded_metadata", func(t *testing.T) {
		material := []byte("durable-projection-secret")
		projections := map[string]string{
			"raw":    string(material),
			"base64": base64.StdEncoding.EncodeToString(material),
			"hex":    hex.EncodeToString(material),
		}
		for name, projection := range projections {
			t.Run(name, func(t *testing.T) {
				path := testStorePath(t)
				store := testOpen(t, path, nil)
				request := testCreateRequest(t, testCredential, "request:create-leaking-metadata", material)
				defer request.Material.Destroy()
				request.OwnerRef = credentials.OwnerRef(projection)
				result, err := store.Create(context.Background(), request)
				if !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || !zeroMutationResult(result) {
					t.Fatalf("leaking create result=%+v err=%v", result, err)
				}
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("failed create mutated store: %v", err)
				}

				safe := testCreateRequest(t, testCredential, "request:create-safe-metadata", material)
				defer safe.Material.Destroy()
				if _, err := store.Create(context.Background(), safe); err != nil {
					t.Fatalf("authorized material JSON was leak-gated: %v", err)
				}
				_ = store.Close()
				reopened := testOpen(t, path, nil)
				defer reopened.Close()
				assertUseMaterial(t, reopened, testCredential, "request:use-safe-create", 1, material)
			})
		}
	})

	t.Run("use_ledger_and_receipt", func(t *testing.T) {
		path := testStorePath(t)
		store := testOpen(t, path, nil)
		material := []byte("request:durable-use-leak")
		testCreate(t, store, testCredential, "request:create-use-gate", material)
		before := testStoreSnapshot(t, path)
		called := false
		receipt, err := store.Use(context.Background(), testUseRequest(testCredential, string(material), 1), func(credentials.Secret) error {
			called = true
			return nil
		})
		if !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || receipt.CredentialRef != "" || called {
			t.Fatalf("leaking use receipt=%+v called=%v err=%v", receipt, called, err)
		}
		assertStoreSnapshot(t, path, before)
		_ = store.Close()
		reopened := testOpen(t, path, nil)
		defer reopened.Close()
		assertUseMaterial(t, reopened, testCredential, "request:use-after-gate", 1, material)
	})

	t.Run("rotate_retains_old_material_guard", func(t *testing.T) {
		path := testStorePath(t)
		store := testOpen(t, path, nil)
		oldMaterial := []byte("request:durable-rotate-leak")
		newMaterial := []byte("safe-rotated-material")
		testCreate(t, store, testCredential, "request:create-rotate-gate", oldMaterial)
		before := testStoreSnapshot(t, path)
		request := credentials.RotateRequest{
			ActorRef: "actor:test", RequestRef: string(oldMaterial), CredentialRef: testCredential,
			OwnerRef: testOwner, ExpectedVersion: 1, Material: testSecret(t, newMaterial),
		}
		defer request.Material.Destroy()
		result, err := store.Rotate(context.Background(), request)
		if !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || !zeroMutationResult(result) {
			t.Fatalf("leaking rotate result=%+v err=%v", result, err)
		}
		assertStoreSnapshot(t, path, before)
		_ = store.Close()
		reopened := testOpen(t, path, nil)
		assertUseMaterial(t, reopened, testCredential, "request:use-after-rotate-gate", 1, oldMaterial)
		safeRequest := credentials.RotateRequest{
			ActorRef: "actor:test", RequestRef: "request:rotate-safe", CredentialRef: testCredential,
			OwnerRef: testOwner, ExpectedVersion: 1, Material: testSecret(t, newMaterial),
		}
		defer safeRequest.Material.Destroy()
		if _, err := reopened.Rotate(context.Background(), safeRequest); err != nil {
			t.Fatal(err)
		}
		_ = reopened.Close()
		reopened = testOpen(t, path, nil)
		defer reopened.Close()
		assertUseMaterial(t, reopened, testCredential, "request:use-after-safe-rotate", 2, newMaterial)
	})

	t.Run("revoke_retains_removed_material_guard", func(t *testing.T) {
		path := testStorePath(t)
		store := testOpen(t, path, nil)
		material := []byte("request:durable-revoke-leak")
		testCreate(t, store, testCredential, "request:create-revoke-gate", material)
		before := testStoreSnapshot(t, path)
		result, err := store.Revoke(context.Background(), credentials.RevokeRequest{
			ActorRef: "actor:test", RequestRef: string(material), CredentialRef: testCredential,
			OwnerRef: testOwner, ExpectedVersion: 1, Reason: "test",
		})
		if !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || !zeroMutationResult(result) {
			t.Fatalf("leaking revoke result=%+v err=%v", result, err)
		}
		assertStoreSnapshot(t, path, before)
		_ = store.Close()
		reopened := testOpen(t, path, nil)
		assertUseMaterial(t, reopened, testCredential, "request:use-after-revoke-gate", 1, material)
		if _, err := reopened.Revoke(context.Background(), credentials.RevokeRequest{
			ActorRef: "actor:test", RequestRef: "request:revoke-safe", CredentialRef: testCredential,
			OwnerRef: testOwner, ExpectedVersion: 1, Reason: "test",
		}); err != nil {
			t.Fatal(err)
		}
		_ = reopened.Close()
		reopened = testOpen(t, path, nil)
		defer reopened.Close()
		called := false
		_, err = reopened.Use(context.Background(), testUseRequest(testCredential, "request:use-after-safe-revoke", 1), func(credentials.Secret) error {
			called = true
			return nil
		})
		if !credentials.HasErrorCode(err, credentials.ErrorRevoked) || called {
			t.Fatalf("revoke was not durable: called=%v err=%v", called, err)
		}
	})

	t.Run("cross_record_encoded_material", func(t *testing.T) {
		path := testStorePath(t)
		store := testOpen(t, path, nil)
		material := []byte("cross-record-durable-secret")
		testCreate(t, store, testCredential, "request:create-first-record", material)
		before := testStoreSnapshot(t, path)
		secondRef := credentials.CredentialRef("credential:second")
		request := testCreateRequest(t, secondRef, "request:create-second-record", []byte("safe-second-material"))
		defer request.Material.Destroy()
		request.OwnerRef = credentials.OwnerRef(base64.StdEncoding.EncodeToString(material))
		result, err := store.Create(context.Background(), request)
		if !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || !zeroMutationResult(result) {
			t.Fatalf("cross-record leak result=%+v err=%v", result, err)
		}
		assertStoreSnapshot(t, path, before)
		_ = store.Close()
		reopened := testOpen(t, path, nil)
		defer reopened.Close()
		assertUseMaterial(t, reopened, testCredential, "request:use-first-after-cross-gate", 1, material)
		called := false
		_, err = reopened.Use(context.Background(), testUseRequest(secondRef, "request:use-missing-second", 1), func(credentials.Secret) error {
			called = true
			return nil
		})
		if !credentials.HasErrorCode(err, credentials.ErrorNotFound) || called {
			t.Fatalf("failed cross-record create mutated store: called=%v err=%v", called, err)
		}
	})

	t.Run("open_rejects_contaminated_projection", func(t *testing.T) {
		path := testStorePath(t)
		store := testOpen(t, path, nil)
		material := []byte("owner:contaminated-secret")
		testCreate(t, store, testCredential, "request:create-contamination", material)
		_ = store.Close()

		content := testStoreSnapshot(t, path)
		defer clear(content)
		var doc document
		if err := json.Unmarshal(content, &doc); err != nil {
			t.Fatal(err)
		}
		defer clearDocument(&doc)
		contaminatedOwner := credentials.OwnerRef(string(material))
		item := doc.Records[testCredential]
		item.Metadata.OwnerRef = contaminatedOwner
		doc.Records[testCredential] = item
		for key, entry := range doc.Ledger {
			entry.Result.OwnerRef = contaminatedOwner
			entry.Receipt.OwnerRef = contaminatedOwner
			doc.Ledger[key] = entry
		}
		payload, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		payload = append(payload, '\n')
		defer clear(payload)
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(testOptions(path, nil)); !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) {
			t.Fatalf("material-bearing durable projection reopened: %v", err)
		}
	})
}

func TestPrivateFileRecoveryAndMaterialRemoval(t *testing.T) {
	for _, point := range []string{FailpointAfterReplacementSync, FailpointAfterStoreRename, FailpointAfterStoreDirectorySync} {
		t.Run(point, func(t *testing.T) {
			path := testStorePath(t)
			seed := testOpen(t, path, nil)
			testCreate(t, seed, testCredential, "request:create", []byte("initial-secret"))
			_ = seed.Close()
			failed := testOpen(t, path, func(current string) error {
				if current == point {
					return errors.New("stop")
				}
				return nil
			})
			rotated := []byte("rotated-secret")
			request := credentials.RotateRequest{ActorRef: "actor:test", RequestRef: "request:rotate", CredentialRef: testCredential,
				OwnerRef: testOwner, ExpectedVersion: 1, Material: testSecret(t, rotated)}
			if _, err := failed.Rotate(context.Background(), request); !credentials.HasErrorCode(err, credentials.ErrorStoreIO) {
				t.Fatalf("failpoint error = %v", err)
			}
			_ = failed.Close()
			recovered := testOpen(t, path, nil)
			result, err := recovered.Rotate(context.Background(), request)
			if err != nil || result.Metadata.Version != 2 || !result.Replayed {
				t.Fatalf("recovery = %+v, %v", result, err)
			}
			if _, err := recovered.Revoke(context.Background(), credentials.RevokeRequest{ActorRef: "actor:test", RequestRef: "request:revoke",
				CredentialRef: testCredential, OwnerRef: testOwner, ExpectedVersion: 2, Reason: "remove"}); err != nil {
				t.Fatal(err)
			}
			_ = recovered.Close()
			content, err := os.ReadFile(path)
			if err != nil || bytes.Contains(content, rotated) || bytes.Contains(content, []byte(base64.StdEncoding.EncodeToString(rotated))) {
				t.Fatalf("revoked material remains: err=%v payload=%s", err, content)
			}
		})
	}

	t.Run("unsafe_parent_and_hardlink", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Chmod(root, 0o750); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(testOptions(filepath.Join(root, "credentials.json"), nil)); !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) {
			t.Fatalf("unsafe parent = %v", err)
		}
		if err := os.Chmod(root, 0o700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "credentials.json")
		store := testOpen(t, path, nil)
		testCreate(t, store, testCredential, "request:create", []byte("material"))
		_ = store.Close()
		if err := os.Link(path, path+".copy"); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(testOptions(path, nil)); !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) {
			t.Fatalf("hardlink = %v", err)
		}
	})
}

func TestLockAcquisitionHonorsContextCancellation(t *testing.T) {
	path := testStorePath(t)
	store := testOpen(t, path, nil)
	t.Cleanup(func() { _ = store.Close() })
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = directory.Close() })
	if err := syscall.Flock(int(directory.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(directory.Fd()), syscall.LOCK_UN)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = store.Create(ctx, testCreateRequest(t, testCredential, "request:lock-timeout", []byte("lock-secret")))
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("blocked lock ignored context: elapsed=%s err=%v", time.Since(started), err)
	}
}

func TestCredentialDocumentMaterialBuffersAreCleared(t *testing.T) {
	oldMaterial := []byte("old-document-secret")
	doc := document{Records: map[credentials.CredentialRef]record{
		testCredential: {Material: oldMaterial},
	}}
	clearDocument(&doc)
	if !bytes.Equal(oldMaterial, make([]byte, len(oldMaterial))) || doc.Records[testCredential].Material != nil {
		t.Fatalf("document material was not cleared: %q", oldMaterial)
	}

	replacedMaterial := []byte("replace-this-secret")
	target := record{Material: replacedMaterial}
	replaceRecordMaterial(&target, []byte("new-secret"))
	if !bytes.Equal(replacedMaterial, make([]byte, len(replacedMaterial))) || string(target.Material) != "new-secret" {
		t.Fatalf("replaced material retained old bytes: old=%q new=%q", replacedMaterial, target.Material)
	}
	clearDocument(&document{Records: map[credentials.CredentialRef]record{testCredential: target}})
}

func TestCredentialReadFailuresClearPartialMaterial(t *testing.T) {
	material := []byte("partial-read-secret")
	path := testStorePath(t)
	if err := os.WriteFile(path, material, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	fs := &fileSystem{owner: uint32(os.Geteuid()), maximum: 1 << 20}
	for _, test := range []struct {
		name  string
		stats []scriptedStat
		err   error
	}{
		{name: "partial_read", stats: []scriptedStat{{info: info}}, err: errors.New("partial")},
		{name: "final_stat", stats: []scriptedStat{{info: info}, {err: errors.New("fstat")}}, err: io.EOF},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := &scriptedReadable{material: append([]byte(nil), material...), readErr: test.err, stats: test.stats}
			content, found, err := fs.readOpened(file, info)
			if content != nil || found || !errors.Is(err, errUnsafeFile) || !bytes.Equal(file.observed, make([]byte, len(file.observed))) {
				t.Fatalf("unsafe read retained material: found=%v content=%q observed=%q err=%v", found, content, file.observed, err)
			}
		})
	}
}

type scriptedStat struct {
	info os.FileInfo
	err  error
}

type scriptedReadable struct {
	material []byte
	readErr  error
	stats    []scriptedStat
	observed []byte
	statCall int
}

func (file *scriptedReadable) Read(target []byte) (int, error) {
	if len(file.material) == 0 {
		return 0, file.readErr
	}
	read := copy(target, file.material)
	file.material = file.material[read:]
	file.observed = target[:read]
	return read, file.readErr
}

func (file *scriptedReadable) Stat() (os.FileInfo, error) {
	if file.statCall >= len(file.stats) {
		return nil, errors.New("unexpected stat")
	}
	result := file.stats[file.statCall]
	file.statCall++
	return result.info, result.err
}

func TestCredentialRequestFingerprintHasStableVersionedVector(t *testing.T) {
	const want = "sha256:14f46224bf2016af0639dd209f47bea5b55ae686b1dbdcb66961c9dd33a2aeeb"
	if got := requestFingerprint("rotate", "actor:test", "request:test", "credential:test"); got != want {
		t.Fatalf("credential request fingerprint=%q want=%q", got, want)
	}
}

func testStorePath(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "credentials.json")
}

func testOptions(path string, failpoint func(string) error) Options {
	return Options{Path: path, OwnerUID: os.Geteuid(), MaxStoreBytes: 1 << 20,
		Now: func() time.Time { return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC) }, Failpoint: failpoint}
}

func testOpen(t *testing.T, path string, failpoint func(string) error) *Store {
	t.Helper()
	store, err := Open(testOptions(path, failpoint))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func testSecret(t *testing.T, material []byte) credentials.Secret {
	t.Helper()
	secret, err := credentials.NewSecret(material)
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

func testCreateRequest(t *testing.T, ref credentials.CredentialRef, request string, material []byte) credentials.CreateRequest {
	t.Helper()
	return credentials.CreateRequest{ActorRef: "actor:test", RequestRef: request, CredentialRef: ref, OwnerRef: testOwner,
		ScopeRefs: []credentials.ScopeRef{testScope}, PurposeRef: testPurpose, Material: testSecret(t, material)}
}

func testCreate(t *testing.T, store *Store, ref credentials.CredentialRef, request string, material []byte) credentials.MutationResult {
	t.Helper()
	result, err := store.Create(context.Background(), testCreateRequest(t, ref, request, material))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func testUseRequest(ref credentials.CredentialRef, request string, version credentials.Version) credentials.UseRequest {
	return credentials.UseRequest{ActorRef: "actor:test", RequestRef: request, CredentialRef: ref, OwnerRef: testOwner,
		ScopeRef: testScope, PurposeRef: testPurpose, Version: version}
}

func zeroMutationResult(result credentials.MutationResult) bool {
	return result.Metadata.CredentialRef == "" && result.Receipt.CredentialRef == "" && !result.Replayed
}

func testStoreSnapshot(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func assertStoreSnapshot(t *testing.T, path string, want []byte) {
	t.Helper()
	got := testStoreSnapshot(t, path)
	defer clear(got)
	if !bytes.Equal(got, want) {
		t.Fatalf("credential store mutated after rejected projection")
	}
}

func assertUseMaterial(t *testing.T, store *Store, ref credentials.CredentialRef, request string, version credentials.Version, want []byte) {
	t.Helper()
	called := 0
	_, err := store.Use(context.Background(), testUseRequest(ref, request, version), func(secret credentials.Secret) error {
		called++
		got := secret.Bytes()
		defer clear(got)
		if !bytes.Equal(got, want) {
			return errors.New("unexpected credential material")
		}
		return nil
	})
	if err != nil || called != 1 {
		t.Fatalf("use material called=%d err=%v", called, err)
	}
}
