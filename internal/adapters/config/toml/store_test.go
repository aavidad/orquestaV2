package toml

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"orquesta/internal/config"
)

const (
	testMaxSource  = int64(64 << 10)
	testMaxReceipt = int64(128 << 10)
)

func TestStoreCommitReadReplayCASAndDefensiveCopies(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	empty, err := store.Read(context.Background())
	if err != nil {
		t.Fatalf("read empty: %v cause=%v", err, errors.Unwrap(err))
	}
	if len(empty.Content) != 0 || empty.Revision != revisionFor(nil) {
		t.Fatalf("empty document = %#v", empty)
	}

	request := testCommitRequest(empty.Revision, []byte("[server]\nlisten = \"127.0.0.1:9000\"\n"), "actor:owner", "request:first")
	request.ChangedKeys = []config.Key{"server.listen", "api.locale"}
	request.PendingRestartKeys = []config.Key{"server.listen"}
	result, err := store.Commit(context.Background(), request)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if result.Replayed || result.Document.Revision != revisionFor(request.Replacement) ||
		!bytes.Equal(result.Document.Content, request.Replacement) ||
		!strings.HasPrefix(result.Receipt.ReceiptRef, receiptRefPrefix) ||
		result.Receipt.BeforeRevision != empty.Revision || result.Receipt.AfterRevision != result.Document.Revision {
		t.Fatalf("commit result = %#v", result)
	}
	wantChanged := []config.Key{"api.locale", "server.listen"}
	if !reflect.DeepEqual(result.Receipt.ChangedKeys, wantChanged) ||
		!reflect.DeepEqual(result.Receipt.PendingRestartKeys, []config.Key{"server.listen"}) {
		t.Fatalf("normalized keys = %#v/%#v", result.Receipt.ChangedKeys, result.Receipt.PendingRestartKeys)
	}
	assertMode(t, path, 0o600)
	assertMode(t, store.receiptPath(request.ActorRef, request.RequestRef), 0o400)
	assertAbsent(t, store.pendingPath)
	assertAbsent(t, store.pendingStagePath)
	assertAbsent(t, store.nextPath)

	replayRequest := cloneRequest(request)
	replayed, err := store.Commit(context.Background(), replayRequest)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !replayed.Replayed || !reflect.DeepEqual(replayed.Document, result.Document) || !reflect.DeepEqual(replayed.Receipt, result.Receipt) {
		t.Fatalf("replay = %#v, want %#v", replayed, result)
	}
	retryAtAnotherTime := cloneRequest(replayRequest)
	retryAtAnotherTime.ChangedAt = retryAtAnotherTime.ChangedAt.Add(time.Hour)
	replayedLater, err := store.Commit(context.Background(), retryAtAnotherTime)
	if err != nil || !replayedLater.Replayed || !reflect.DeepEqual(replayedLater, replayed) {
		t.Fatalf("later replay = %#v, %v; want original %#v", replayedLater, err, replayed)
	}

	request.Replacement[0] = 'X'
	request.ChangedKeys[0] = "tampered.key"
	result.Document.Content[0] = 'Y'
	result.Receipt.ChangedKeys[0] = "tampered.key"
	stored, err := store.Read(context.Background())
	if err != nil {
		t.Fatalf("read committed: %v", err)
	}
	if !bytes.Equal(stored.Content, replayRequest.Replacement) {
		t.Fatalf("caller mutation escaped: %q", stored.Content)
	}

	derivedReplayDifferences := []struct {
		name   string
		mutate func(*config.CommitRequest)
	}{
		{name: "replacement", mutate: func(request *config.CommitRequest) { request.Replacement = []byte("api.locale = \"different\"\n") }},
		{name: "expected revision", mutate: func(request *config.CommitRequest) { request.ExpectedRevision = revisionFor([]byte("different")) }},
		{name: "changed keys", mutate: func(request *config.CommitRequest) { request.ChangedKeys = []config.Key{"server.listen"} }},
		{name: "pending keys", mutate: func(request *config.CommitRequest) {
			request.PendingRestartKeys = []config.Key{"api.locale", "server.listen"}
		}},
	}
	for _, test := range derivedReplayDifferences {
		t.Run("replay ignores derived "+test.name, func(t *testing.T) {
			candidate := cloneRequest(replayRequest)
			test.mutate(&candidate)
			got, err := store.Commit(context.Background(), candidate)
			if err != nil || !got.Replayed || !reflect.DeepEqual(got, replayed) {
				t.Fatalf("derived replay = %#v, %v; want %#v", got, err, replayed)
			}
		})
	}
	conflicting := cloneRequest(replayRequest)
	conflicting.Fingerprint = string(revisionFor([]byte("different")))
	_, err = store.Commit(context.Background(), conflicting)
	assertStoreCode(t, err, config.DocumentStoreReplayConflict)

	stale := testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:owner", "request:stale")
	_, err = store.Commit(context.Background(), stale)
	assertStoreCode(t, err, config.DocumentStoreRevisionConflict)
}

func TestStoreSerializesConcurrentCASAcrossInstances(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	first := openTestStore(t, path, nil)
	second := openTestStore(t, path, nil)
	empty, err := first.Read(context.Background())
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	requests := []config.CommitRequest{
		testCommitRequest(empty.Revision, []byte("api.locale = \"es\"\n"), "actor:a", "request:a"),
		testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:b", "request:b"),
	}
	stores := []*Store{first, second}
	start := make(chan struct{})
	errorsByWriter := make([]error, len(stores))
	var wait sync.WaitGroup
	for index := range stores {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, errorsByWriter[index] = stores[index].Commit(context.Background(), requests[index])
		}(index)
	}
	close(start)
	wait.Wait()

	successes, conflicts := 0, 0
	for _, err := range errorsByWriter {
		switch {
		case err == nil:
			successes++
		case config.IsDocumentStoreError(err, config.DocumentStoreRevisionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected writer error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes/conflicts = %d/%d, errors=%v", successes, conflicts, errorsByWriter)
	}
	document, err := first.Read(context.Background())
	if err != nil {
		t.Fatalf("read winner: %v", err)
	}
	if !bytes.Equal(document.Content, requests[0].Replacement) && !bytes.Equal(document.Content, requests[1].Replacement) {
		t.Fatalf("unexpected winner content: %q", document.Content)
	}
}

func TestStoreReplayAfterStateAdvanceReturnsCurrentDocumentAndOriginalReceipt(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	empty, _ := store.Read(context.Background())
	firstRequest := testCommitRequest(empty.Revision, []byte("a = 1\n"), "actor:a", "request:first")
	first, err := store.Commit(context.Background(), firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	secondRequest := testCommitRequest(first.Document.Revision, []byte("a = 1\nb = 2\n"), "actor:a", "request:second")
	second, err := store.Commit(context.Background(), secondRequest)
	if err != nil {
		t.Fatal(err)
	}

	retry := cloneRequest(firstRequest)
	retry.Replacement = []byte("derived state is intentionally different\n")
	retry.PendingRestartKeys = []config.Key{"server.listen"}
	replayed, err := store.Commit(context.Background(), retry)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Document, second.Document) ||
		!reflect.DeepEqual(replayed.Receipt, first.Receipt) {
		t.Fatalf("advanced replay = %#v, %v; current=%#v receipt=%#v", replayed, err, second.Document, first.Receipt)
	}
	entries, err := os.ReadDir(store.receiptDirectory)
	if err != nil || len(entries) != 2 {
		t.Fatalf("advanced replay receipt count = %d, %v", len(entries), err)
	}
}

func TestStoreReplayOnlyCannotBecomeWriteAfterABA(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	empty, _ := store.Read(context.Background())

	probe := testCommitRequest(empty.Revision, []byte("must not be written\n"), "actor:a", "request:absent")
	probe.ReplayOnly = true
	probe.ChangedAt = time.Time{}
	probe.Replacement = nil
	probe.ChangedKeys = nil
	probe.PendingRestartKeys = nil
	_, err := store.Commit(context.Background(), probe)
	assertStoreCode(t, err, config.DocumentStoreRevisionConflict)

	after, err := store.Read(context.Background())
	if err != nil || !reflect.DeepEqual(after, empty) {
		t.Fatalf("replay-only mutated source: %#v, %v; want %#v", after, err, empty)
	}
	assertAbsent(t, path)
	assertAbsent(t, store.nextPath)
	assertAbsent(t, store.pendingPath)
}

func TestStoreReplayOnlyReturnsMatchingReceiptAndCurrentDocument(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	empty, _ := store.Read(context.Background())
	request := testCommitRequest(empty.Revision, []byte("a = 1\n"), "actor:a", "request:first")
	committed, err := store.Commit(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}

	probe := config.CommitRequest{
		ExpectedRevision: empty.Revision,
		ReplayOnly:       true,
		ActorRef:         request.ActorRef,
		RequestRef:       request.RequestRef,
		Fingerprint:      request.Fingerprint,
	}
	replayed, err := store.Commit(context.Background(), probe)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Document, committed.Document) ||
		!reflect.DeepEqual(replayed.Receipt, committed.Receipt) {
		t.Fatalf("replay-only = %#v, %v; want %#v", replayed, err, committed)
	}
}

func TestStoreRecoversEveryDurableCrashBoundary(t *testing.T) {
	points := []string{
		FailpointAfterIntentSync,
		FailpointAfterSourceRename,
		FailpointAfterSourceDirectory,
		FailpointAfterReceiptSeal,
		FailpointAfterReceiptRename,
		FailpointAfterReceiptDirectory,
	}
	for _, point := range points {
		t.Run(string(point), func(t *testing.T) {
			path := filepath.Join(privateTempDir(t), "orquesta.toml")
			failed := false
			store := openTestStore(t, path, func(observed string) error {
				if observed == point && !failed {
					failed = true
					return errors.New("simulated_crash")
				}
				return nil
			})
			empty, err := store.Read(context.Background())
			if err != nil {
				t.Fatalf("read empty: %v", err)
			}
			request := testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:crash", "request:"+string(point))
			_, err = store.Commit(context.Background(), request)
			assertStoreCode(t, err, config.DocumentStoreIO)
			if !failed {
				t.Fatalf("failpoint %s was not reached", point)
			}

			recoveredStore := openTestStore(t, path, nil)
			recovered, err := recoveredStore.Read(context.Background())
			if err != nil {
				t.Fatalf("recover read: %v", err)
			}
			if recovered.Revision != revisionFor(request.Replacement) || !bytes.Equal(recovered.Content, request.Replacement) {
				t.Fatalf("recovered document = %#v", recovered)
			}
			replay, err := recoveredStore.Commit(context.Background(), request)
			if err != nil || !replay.Replayed || replay.Receipt.AfterRevision != recovered.Revision {
				t.Fatalf("replay after recovery = %#v, %v", replay, err)
			}
			assertMode(t, recoveredStore.receiptPath(request.ActorRef, request.RequestRef), 0o400)
			assertAbsent(t, recoveredStore.pendingPath)
			assertAbsent(t, recoveredStore.pendingStagePath)
			assertAbsent(t, recoveredStore.nextPath)
		})
	}
}

func TestStoreDiscardsUnpublishedIntentStageAndReplacement(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	if err := ensureReceiptDirectory(store.receiptDirectory); err != nil {
		t.Fatalf("ensure receipt directory: %v", err)
	}
	writeFile(t, store.pendingStagePath, []byte("partial unpublished intent"), 0o600)
	writeFile(t, store.nextPath, []byte("unpublished replacement"), 0o600)
	document, err := store.Read(context.Background())
	if err != nil {
		t.Fatalf("read after unpublished stage: %v", err)
	}
	if document.Revision != revisionFor(nil) || len(document.Content) != 0 {
		t.Fatalf("unpublished stage changed document: %#v", document)
	}
	assertAbsent(t, store.pendingStagePath)
	assertAbsent(t, store.nextPath)
}

func TestStoreDiscardsOrphanReplacementWithoutReceipt(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, func(point string) error {
		if point == FailpointAfterReplacementSync {
			return errors.New("simulated_crash")
		}
		return nil
	})
	empty, err := store.Read(context.Background())
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	request := testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:orphan", "request:orphan")
	_, err = store.Commit(context.Background(), request)
	assertStoreCode(t, err, config.DocumentStoreIO)
	assertAbsent(t, store.pendingPath)
	if _, err := os.Lstat(store.nextPath); err != nil {
		t.Fatalf("orphan replacement missing before recovery: %v", err)
	}

	recoveredStore := openTestStore(t, path, nil)
	recovered, err := recoveredStore.Read(context.Background())
	if err != nil {
		t.Fatalf("recover orphan: %v", err)
	}
	if recovered.Revision != empty.Revision || len(recovered.Content) != 0 {
		t.Fatalf("orphan was acknowledged: %#v", recovered)
	}
	assertAbsent(t, recoveredStore.nextPath)
	assertAbsent(t, recoveredStore.receiptPath(request.ActorRef, request.RequestRef))
	committed, err := recoveredStore.Commit(context.Background(), request)
	if err != nil || committed.Replayed {
		t.Fatalf("fresh retry after orphan = %#v, %v", committed, err)
	}
}

func TestStoreRejectsUnsafeFilesystemObjectsAndBounds(t *testing.T) {
	tests := []struct {
		name string
		make func(*testing.T, string)
		code config.DocumentStoreErrorCode
	}{
		{
			name: "symlink source",
			make: func(t *testing.T, path string) {
				target := path + ".target"
				writeFile(t, target, []byte("safe"), 0o600)
				if err := os.Symlink(target, path); err != nil {
					t.Skipf("symlink unsupported: %v", err)
				}
			},
			code: config.DocumentStoreSourceInvalid,
		},
		{
			name: "hardlink source",
			make: func(t *testing.T, path string) {
				original := path + ".original"
				writeFile(t, original, []byte("safe"), 0o600)
				if err := os.Link(original, path); err != nil {
					t.Skipf("hardlink unsupported: %v", err)
				}
			},
			code: config.DocumentStoreSourceInvalid,
		},
		{
			name: "fifo source",
			make: func(t *testing.T, path string) {
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Skipf("fifo unsupported: %v", err)
				}
			},
			code: config.DocumentStoreSourceInvalid,
		},
		{
			name: "broad source mode",
			make: func(t *testing.T, path string) { writeFile(t, path, []byte("safe"), 0o644) },
			code: config.DocumentStoreSourceInvalid,
		},
		{
			name: "read-only source mode",
			make: func(t *testing.T, path string) { writeFile(t, path, []byte("safe"), 0o400) },
			code: config.DocumentStoreSourceInvalid,
		},
		{
			name: "source directory",
			make: func(t *testing.T, path string) {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatalf("mkdir source: %v", err)
				}
			},
			code: config.DocumentStoreSourceInvalid,
		},
		{
			name: "oversize source",
			make: func(t *testing.T, path string) { writeFile(t, path, bytes.Repeat([]byte("x"), 129), 0o600) },
			code: config.DocumentStoreSourceTooLarge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(privateTempDir(t), "orquesta.toml")
			test.make(t, path)
			store, err := Open(Options{Path: path, MaxSourceBytes: 128, MaxReceiptBytes: testMaxReceipt})
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			_, err = store.Read(context.Background())
			assertStoreCode(t, err, test.code)
		})
	}

	t.Run("symlink lock", func(t *testing.T) {
		path := filepath.Join(privateTempDir(t), "orquesta.toml")
		target := path + ".lock-target"
		writeFile(t, target, nil, 0o600)
		if err := os.Symlink(target, path+".lock"); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		store := openTestStore(t, path, nil)
		_, err := store.Read(context.Background())
		assertStoreCode(t, err, config.DocumentStoreSourceInvalid)
	})

	t.Run("symlink receipt directory", func(t *testing.T) {
		root := privateTempDir(t)
		path := filepath.Join(root, "orquesta.toml")
		target := filepath.Join(root, "receipt-target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatalf("mkdir receipt target: %v", err)
		}
		if err := os.Symlink(target, path+".receipts"); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		store := openTestStore(t, path, nil)
		_, err := store.Read(context.Background())
		assertStoreCode(t, err, config.DocumentStoreSourceInvalid)
	})

	t.Run("symlink parent component", func(t *testing.T) {
		root := privateTempDir(t)
		realDirectory := filepath.Join(root, "real")
		if err := os.Mkdir(realDirectory, 0o700); err != nil {
			t.Fatalf("mkdir real: %v", err)
		}
		linkedDirectory := filepath.Join(root, "linked")
		if err := os.Symlink(realDirectory, linkedDirectory); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		store := openTestStore(t, filepath.Join(linkedDirectory, "orquesta.toml"), nil)
		_, err := store.Read(context.Background())
		assertStoreCode(t, err, config.DocumentStoreSourceInvalid)
	})

	t.Run("non-private parent mode", func(t *testing.T) {
		root := privateTempDir(t)
		directory := filepath.Join(root, "broad")
		if err := os.Mkdir(directory, 0o755); err != nil {
			t.Fatalf("mkdir broad: %v", err)
		}
		if err := os.Chmod(directory, 0o755); err != nil {
			t.Fatalf("chmod broad: %v", err)
		}
		store := openTestStore(t, filepath.Join(directory, "orquesta.toml"), nil)
		_, err := store.Read(context.Background())
		assertStoreCode(t, err, config.DocumentStoreSourceInvalid)
	})
}

func TestStoreRejectsTamperedReceiptAndDivergentPendingIntent(t *testing.T) {
	t.Run("receipt mode", func(t *testing.T) {
		path := filepath.Join(privateTempDir(t), "orquesta.toml")
		store := openTestStore(t, path, nil)
		empty, _ := store.Read(context.Background())
		request := testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:receipt", "request:receipt")
		if _, err := store.Commit(context.Background(), request); err != nil {
			t.Fatalf("commit: %v", err)
		}
		receiptPath := store.receiptPath(request.ActorRef, request.RequestRef)
		if err := os.Chmod(receiptPath, 0o600); err != nil {
			t.Fatalf("chmod receipt: %v", err)
		}
		_, err := store.Commit(context.Background(), request)
		assertStoreCode(t, err, config.DocumentStoreSourceInvalid)
	})

	t.Run("external source divergence", func(t *testing.T) {
		path := filepath.Join(privateTempDir(t), "orquesta.toml")
		store := openTestStore(t, path, func(point string) error {
			if point == FailpointAfterIntentSync {
				return errors.New("simulated_crash")
			}
			return nil
		})
		empty, _ := store.Read(context.Background())
		request := testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:pending", "request:pending")
		_, err := store.Commit(context.Background(), request)
		assertStoreCode(t, err, config.DocumentStoreIO)
		writeFile(t, path, []byte("api.locale = \"external\"\n"), 0o600)
		recovering := openTestStore(t, path, nil)
		_, err = recovering.Read(context.Background())
		assertStoreCode(t, err, config.DocumentStoreRevisionConflict)
		content, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(content, []byte("api.locale = \"external\"\n")) {
			t.Fatalf("divergent source overwritten: %q %v", content, err)
		}
	})
}

func TestStoreValidatesSourceRequirementRequestAndReceiptLimit(t *testing.T) {
	emptyStore, err := Open(Options{MaxSourceBytes: testMaxSource, MaxReceiptBytes: testMaxReceipt})
	if err != nil {
		t.Fatalf("open defaults-only: %v", err)
	}
	document, err := emptyStore.Read(context.Background())
	if err != nil || document.Revision != revisionFor(nil) || len(document.Content) != 0 {
		t.Fatalf("defaults-only read = %#v, %v", document, err)
	}
	_, err = emptyStore.Commit(context.Background(), testCommitRequest(document.Revision, []byte("x = 1\n"), "actor:a", "request:a"))
	assertStoreCode(t, err, config.DocumentStoreSourceRequired)

	missingPath := filepath.Join(privateTempDir(t), "missing.toml")
	_, err = Open(Options{
		Path: missingPath, MaxSourceBytes: testMaxSource, MaxReceiptBytes: testMaxReceipt, RequireExisting: true,
	})
	assertStoreCode(t, err, config.DocumentStoreSourceRequired)
	assertAbsent(t, missingPath+".lock")
	writeFile(t, missingPath, []byte{}, 0o600)
	required, err := Open(Options{
		Path: missingPath, MaxSourceBytes: testMaxSource, MaxReceiptBytes: testMaxReceipt, RequireExisting: true,
	})
	if err != nil {
		t.Fatalf("open required existing source: %v", err)
	}
	if _, err := required.Read(context.Background()); err != nil {
		t.Fatalf("read required existing source: %v", err)
	}

	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	empty, _ := store.Read(context.Background())
	valid := testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:a", "request:a")
	tests := []struct {
		name   string
		mutate func(*config.CommitRequest)
	}{
		{name: "invalid revision", mutate: func(request *config.CommitRequest) { request.ExpectedRevision = "sha256:bad" }},
		{name: "blank actor", mutate: func(request *config.CommitRequest) { request.ActorRef = " " }},
		{name: "blank request", mutate: func(request *config.CommitRequest) { request.RequestRef = "" }},
		{name: "blank fingerprint", mutate: func(request *config.CommitRequest) { request.Fingerprint = "" }},
		{name: "noncanonical fingerprint", mutate: func(request *config.CommitRequest) { request.Fingerprint = "fingerprint:value" }},
		{name: "control actor", mutate: func(request *config.CommitRequest) { request.ActorRef = "actor:\ncontrol" }},
		{name: "unicode control actor", mutate: func(request *config.CommitRequest) { request.ActorRef = "actor:\u0085control" }},
		{name: "invalid UTF-8 actor", mutate: func(request *config.CommitRequest) { request.ActorRef = "actor:" + string([]byte{0xff}) }},
		{name: "zero time", mutate: func(request *config.CommitRequest) { request.ChangedAt = time.Time{} }},
		{name: "invalid key", mutate: func(request *config.CommitRequest) { request.ChangedKeys = []config.Key{"API.Locale"} }},
		{name: "duplicate key", mutate: func(request *config.CommitRequest) { request.ChangedKeys = []config.Key{"api.locale", "api.locale"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := cloneRequest(valid)
			test.mutate(&request)
			_, err := store.Commit(context.Background(), request)
			assertStoreCode(t, err, config.DocumentStoreRequestInvalid)
		})
	}

	tinyReceiptStore, err := Open(Options{Path: filepath.Join(privateTempDir(t), "orquesta.toml"), MaxSourceBytes: testMaxSource, MaxReceiptBytes: 64})
	if err != nil {
		t.Fatalf("open tiny receipt store: %v", err)
	}
	empty, _ = tinyReceiptStore.Read(context.Background())
	_, err = tinyReceiptStore.Commit(context.Background(), testCommitRequest(empty.Revision, []byte("api.locale = \"en\"\n"), "actor:a", "request:a"))
	assertStoreCode(t, err, config.DocumentStoreReceiptTooLarge)
	assertAbsent(t, tinyReceiptStore.nextPath)
	assertAbsent(t, tinyReceiptStore.pendingPath)
}

func TestReservedPathsAreExactDetachedAdapterNamespaces(t *testing.T) {
	first := ReservedPaths("/private/orquesta.toml")
	want := []string{
		"/private/orquesta.toml.lock", "/private/orquesta.toml.next", "/private/orquesta.toml.receipts",
	}
	if !reflect.DeepEqual(first, want) || len(ReservedPaths("")) != 0 {
		t.Fatalf("reserved paths = %#v", first)
	}
	first[0] = "tampered"
	if got := ReservedPaths("/private/orquesta.toml"); !reflect.DeepEqual(got, want) {
		t.Fatalf("reserved paths share caller storage: %#v", got)
	}
}

func TestStoreKeepsLargeSourceOutOfBoundedReceipt(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store, err := Open(Options{Path: path, MaxSourceBytes: 2 << 20, MaxReceiptBytes: 16 << 10})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	empty, err := store.Read(context.Background())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	marker := []byte("SOURCE_CONTENT_MUST_NEVER_ENTER_RECEIPT_5d1868f2")
	replacement := append(bytes.Repeat([]byte("x"), 1<<20-len(marker)), marker...)
	request := testCommitRequest(empty.Revision, replacement, "actor:large", "request:large")
	result, err := store.Commit(context.Background(), request)
	if err != nil {
		t.Fatalf("commit 1 MiB source: %v", err)
	}
	if result.Document.Revision != revisionFor(replacement) || !bytes.Equal(result.Document.Content, replacement) {
		t.Fatalf("large result mismatch")
	}
	receiptInfo, err := os.Stat(store.receiptPath(request.ActorRef, request.RequestRef))
	if err != nil || receiptInfo.Size() >= 16<<10 {
		t.Fatalf("receipt size = %v/%v", receiptInfo, err)
	}
	receiptPayload, err := os.ReadFile(store.receiptPath(request.ActorRef, request.RequestRef))
	if err != nil || bytes.Contains(receiptPayload, marker) {
		t.Fatalf("source content leaked into receipt: %v", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(receiptPayload, &envelope); err != nil {
		t.Fatalf("decode receipt envelope: %v", err)
	}
	assertExactJSONKeys(t, envelope, "schema_version", "document_type", "receipt")
	var receipt map[string]json.RawMessage
	if err := json.Unmarshal(envelope["receipt"], &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	assertExactJSONKeys(t, receipt,
		"receipt_ref", "actor_ref", "request_ref", "fingerprint", "before_revision", "after_revision",
		"changed_keys", "pending_restart_keys", "changed_at",
	)
	assertReceiptTree(t, store.receiptDirectory, 16<<10)
}

func TestStoreAllowsPendingRestartKeysFromEarlierMutations(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	empty, _ := store.Read(context.Background())
	first := testCommitRequest(empty.Revision, []byte("a = 1\n"), "actor:a", "request:first")
	first.ChangedKeys = []config.Key{"server.listen"}
	first.PendingRestartKeys = []config.Key{"server.listen"}
	committed, err := store.Commit(context.Background(), first)
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}
	second := testCommitRequest(committed.Document.Revision, []byte("a = 1\nb = 2\n"), "actor:a", "request:second")
	second.ChangedKeys = []config.Key{"api.locale"}
	second.PendingRestartKeys = []config.Key{"api.locale", "server.listen"}
	if _, err := store.Commit(context.Background(), second); err != nil {
		t.Fatalf("pending restart superset rejected: %v", err)
	}
}

func TestStoreReplayFingerprintFencesChangedMetadata(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "orquesta.toml")
	store := openTestStore(t, path, nil)
	empty, _ := store.Read(context.Background())
	request := testCommitRequest(empty.Revision, []byte("a = 1\n"), "actor:a", "request:delimiter")
	request.ChangedKeys = nil
	request.PendingRestartKeys = []config.Key{"pending_restart"}
	if _, err := store.Commit(context.Background(), request); err != nil {
		t.Fatalf("commit: %v", err)
	}
	conflict := cloneRequest(request)
	conflict.ChangedKeys = []config.Key{"pending_restart"}
	conflict.PendingRestartKeys = nil
	conflict.Fingerprint = string(revisionFor([]byte("metadata-moved")))
	_, err := store.Commit(context.Background(), conflict)
	assertStoreCode(t, err, config.DocumentStoreReplayConflict)
}

func openTestStore(t *testing.T, path string, failpoint func(string) error) *Store {
	t.Helper()
	store, err := Open(Options{
		Path: path, MaxSourceBytes: testMaxSource, MaxReceiptBytes: testMaxReceipt, Failpoint: failpoint,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func testCommitRequest(expected config.Revision, replacement []byte, actorRef, requestRef string) config.CommitRequest {
	return config.CommitRequest{
		ExpectedRevision:   expected,
		Replacement:        append([]byte(nil), replacement...),
		ActorRef:           actorRef,
		RequestRef:         requestRef,
		Fingerprint:        string(revisionFor([]byte("fingerprint:" + requestRef))),
		ChangedKeys:        []config.Key{"api.locale"},
		PendingRestartKeys: []config.Key{"api.locale"},
		ChangedAt:          time.Date(2035, 1, 2, 3, 4, 5, 0, time.FixedZone("test", 3600)),
	}
}

func cloneRequest(request config.CommitRequest) config.CommitRequest {
	request.Replacement = append([]byte(nil), request.Replacement...)
	request.ChangedKeys = append([]config.Key(nil), request.ChangedKeys...)
	request.PendingRestartKeys = append([]config.Key(nil), request.PendingRestartKeys...)
	return request
}

func assertStoreCode(t *testing.T, err error, code config.DocumentStoreErrorCode) {
	t.Helper()
	if !config.IsDocumentStoreError(err, code) {
		t.Fatalf("error = %v cause=%v, want code %s", err, errors.Unwrap(err), code)
	}
}

func assertMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != mode || !info.Mode().IsRegular() {
		t.Fatalf("mode %s = %v/%v, want %o regular", path, info, err, mode)
	}
}

func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("path %s still exists: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func TestPrivateFilesystemPredicatesRejectForeignOwner(t *testing.T) {
	foreignUID := uint32(os.Geteuid()) + 1
	regular := fakeFileInfo{mode: 0o600, stat: syscall.Stat_t{Nlink: 1, Uid: foreignUID}}
	if privateRegular(regular, 1024, privateWriteMode) {
		t.Fatal("foreign-owned regular file accepted")
	}
	directory := fakeFileInfo{mode: os.ModeDir | 0o700, stat: syscall.Stat_t{Nlink: 1, Uid: foreignUID}}
	if privateDirectory(directory) {
		t.Fatal("foreign-owned directory accepted")
	}
	currentUID := uint32(os.Geteuid())
	setuidRegular := fakeFileInfo{mode: os.ModeSetuid | 0o600, stat: syscall.Stat_t{Nlink: 1, Uid: currentUID}}
	if privateRegular(setuidRegular, 1024, privateWriteMode) {
		t.Fatal("setuid regular file accepted")
	}
	stickyDirectory := fakeFileInfo{mode: os.ModeDir | os.ModeSticky | 0o700, stat: syscall.Stat_t{Nlink: 1, Uid: currentUID}}
	if privateDirectory(stickyDirectory) {
		t.Fatal("sticky private directory accepted")
	}
}

type fakeFileInfo struct {
	mode os.FileMode
	stat syscall.Stat_t
}

func (info fakeFileInfo) Name() string       { return "fake" }
func (info fakeFileInfo) Size() int64        { return 0 }
func (info fakeFileInfo) Mode() os.FileMode  { return info.mode }
func (info fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (info fakeFileInfo) IsDir() bool        { return info.mode.IsDir() }
func (info fakeFileInfo) Sys() any           { return &info.stat }

func assertReceiptTree(t *testing.T, root string, maximum int64) {
	t.Helper()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != uint32(os.Geteuid()) {
			t.Fatalf("receipt owner invalid: %s", path)
		}
		if info.IsDir() {
			if info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("receipt directory mode invalid: %s %o", path, info.Mode().Perm())
			}
			return nil
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o400 || stat.Nlink != 1 || info.Size() > maximum {
			t.Fatalf("receipt file invalid: %s mode=%v nlink=%d size=%d", path, info.Mode(), stat.Nlink, info.Size())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk receipts: %v", err)
	}
}

func privateTempDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatalf("chmod temp dir: %v", err)
	}
	return directory
}

func assertExactJSONKeys(t *testing.T, values map[string]json.RawMessage, expected ...string) {
	t.Helper()
	if len(values) != len(expected) {
		t.Fatalf("JSON keys = %v, want exact %v", reflect.ValueOf(values).MapKeys(), expected)
	}
	for _, key := range expected {
		if _, found := values[key]; !found {
			t.Fatalf("JSON lacks exact key %q: %v", key, reflect.ValueOf(values).MapKeys())
		}
	}
}
