package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestManagerViewUsesDocumentStoreEnvironmentAndRedactsSensitiveValues(t *testing.T) {
	content := []byte("[runtime.codex]\ncredential_ref = \"credential:old\"\nmodel = \"old-model\"\n")
	active := managerTestSnapshot(t, content)
	store := newManagerFakeStore(content)
	environment := map[string]string{"ORQUESTA_RUNTIME_CODEX_MODEL": "new-model"}
	manager := newTestManager(t, store, active, environment)
	environment["ORQUESTA_RUNTIME_CODEX_MODEL"] = "mutated-model"
	for index := range active.entries {
		if active.entries[index].key == KeyRuntimeCodexModel {
			active.entries[index].value = "mutated-active-model"
		}
	}

	view, err := manager.View(context.Background())
	if err != nil {
		t.Fatalf("view: %v", err)
	}
	if view.SourceRevision != store.revision() || view.ActiveHash != active.Hash() || view.DesiredHash == active.Hash() ||
		!view.PendingRestart || !reflect.DeepEqual(view.PendingRestartKeys, []Key{KeyRuntimeCodexModel}) {
		t.Fatalf("view = %#v", view)
	}
	definitions := Definitions()
	if len(view.Keys) != len(definitions) {
		t.Fatalf("key views = %d, want %d", len(view.Keys), len(definitions))
	}
	for index, definition := range definitions {
		if view.Keys[index].Key != definition.Key {
			t.Fatalf("key order[%d] = %s, want %s", index, view.Keys[index].Key, definition.Key)
		}
	}
	credential := managerKeyView(t, view, KeyRuntimeCodexCredentialRef)
	if credential.ActiveValue != redactedValue || credential.DesiredValue != redactedValue || !credential.Sensitive {
		t.Fatalf("credential view leaked: %#v", credential)
	}
	model := managerKeyView(t, view, KeyRuntimeCodexModel)
	if model.Source != SourceEnv || model.ActiveValue != "old-model" || model.DesiredValue != "new-model" || !model.PendingRestart {
		t.Fatalf("model view = %#v", model)
	}
	payload, err := json.Marshal(view)
	if err != nil || bytes.Contains(payload, []byte("credential:old")) {
		t.Fatalf("view JSON leaked credential reference: %s err=%v", payload, err)
	}

	view.PendingRestartKeys[0] = KeyAPILocale
	view.Keys[0].DesiredValue = "tampered"
	again, err := manager.View(context.Background())
	if err != nil || !reflect.DeepEqual(again.PendingRestartKeys, []Key{KeyRuntimeCodexModel}) || again.Keys[0].DesiredValue == "tampered" {
		t.Fatalf("view mutation escaped: %#v err=%v", again, err)
	}
}

func TestManagerUpdateReplaysNoopAndTracksPrivatePendingAcrossMutations(t *testing.T) {
	content := []byte("[runtime.codex]\ncredential_ref = \"credential:old\"\n\n[scheduler]\npoll_interval = \"500ms\"\n")
	active := managerTestSnapshot(t, content)
	store := newManagerFakeStore(content)
	nowCall := 0
	manager, err := NewManager(ManagerOptions{
		Store: store, Active: active,
		Now: func() time.Time {
			nowCall++
			return time.Date(2035, 1, 2, 3, 4, nowCall, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	initial, _ := manager.View(context.Background())
	credentialRequest := UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:credential", ExpectedRevision: initial.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyRuntimeCodexCredentialRef, Value: CredentialRef("credential:new")}},
	}
	first, err := manager.Update(context.Background(), credentialRequest)
	if err != nil {
		t.Fatalf("credential update: %v", err)
	}
	if first.Replayed || !first.View.PendingRestart || first.View.ActiveHash != first.View.DesiredHash ||
		!reflect.DeepEqual(first.View.PendingRestartKeys, []Key{KeyRuntimeCodexCredentialRef}) {
		t.Fatalf("credential update = %#v", first)
	}
	credentialView := managerKeyView(t, first.View, KeyRuntimeCodexCredentialRef)
	if credentialView.ActiveValue != redactedValue || credentialView.DesiredValue != redactedValue || !credentialView.PendingRestart {
		t.Fatalf("sensitive pending view leaked: %#v", credentialView)
	}
	originalChangedAt := first.Receipt.ChangedAt
	beforeReplayCommits := store.commitCount()
	replayed, err := manager.Update(context.Background(), credentialRequest)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Receipt, first.Receipt) || replayed.Receipt.ChangedAt != originalChangedAt {
		t.Fatalf("replay = %#v err=%v, want original %#v", replayed, err, first)
	}
	if store.commitCount() != beforeReplayCommits+1 {
		t.Fatalf("no-op replay did not reach store: commits=%d/%d", store.commitCount(), beforeReplayCommits)
	}
	replayed.Receipt.ChangedKeys[0] = KeyAPILocale
	replayed.View.PendingRestartKeys[0] = KeyAPILocale
	detached, err := manager.Update(context.Background(), credentialRequest)
	if err != nil || detached.Receipt.ChangedKeys[0] != KeyRuntimeCodexCredentialRef ||
		detached.View.PendingRestartKeys[0] != KeyRuntimeCodexCredentialRef {
		t.Fatalf("result mutation escaped: %#v err=%v", detached, err)
	}

	pollRequest := UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:poll", ExpectedRevision: replayed.View.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeySchedulerPollInterval, Value: "750ms"}},
	}
	second, err := manager.Update(context.Background(), pollRequest)
	if err != nil {
		t.Fatalf("poll update: %v", err)
	}
	if !reflect.DeepEqual(second.View.PendingRestartKeys, []Key{KeyRuntimeCodexCredentialRef, KeySchedulerPollInterval}) {
		t.Fatalf("pending after second mutation = %v", second.View.PendingRestartKeys)
	}
	laterReplay, err := manager.Update(context.Background(), credentialRequest)
	if err != nil || !laterReplay.Replayed || !reflect.DeepEqual(laterReplay.Receipt, first.Receipt) ||
		!reflect.DeepEqual(laterReplay.View.PendingRestartKeys, second.View.PendingRestartKeys) ||
		!bytes.Contains(store.documentSnapshot().Content, []byte("poll_interval = \"750ms\"")) {
		t.Fatalf("replay after later state = %#v err=%v", laterReplay, err)
	}

	restartedActive := managerTestSnapshot(t, store.documentSnapshot().Content)
	restarted := newTestManager(t, store, restartedActive, nil)
	restartReplay, err := restarted.Update(context.Background(), credentialRequest)
	if err != nil || !restartReplay.Replayed || restartReplay.View.PendingRestart ||
		!reflect.DeepEqual(restartReplay.Receipt, first.Receipt) {
		t.Fatalf("replay after restart = %#v err=%v", restartReplay, err)
	}

	revertRequest := UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:revert", ExpectedRevision: second.View.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyRuntimeCodexCredentialRef, Value: CredentialRef("credential:old")}},
	}
	reverted, err := manager.Update(context.Background(), revertRequest)
	if err != nil || !reflect.DeepEqual(reverted.View.PendingRestartKeys, []Key{KeySchedulerPollInterval}) {
		t.Fatalf("revert = %#v err=%v", reverted, err)
	}

	beforeNoopCommits := store.commitCount()
	_, err = manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:noop", ExpectedRevision: reverted.View.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeySchedulerPollInterval, Value: "750ms"}},
	})
	if !HasErrorCode(err, ErrorUpdateNoChanges) || store.commitCount() != beforeNoopCommits {
		t.Fatalf("first noop = %v commits=%d/%d", err, store.commitCount(), beforeNoopCommits)
	}

	_, err = manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:secret", ExpectedRevision: reverted.View.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyRuntimeCodexCredentialRef, Value: "sk-secret"}},
	})
	if !HasErrorCode(err, ErrorValueInvalid) {
		t.Fatalf("raw secret error = %v", err)
	}
}

func TestManagerUpdateCanonicalizesOrderUnsetAndFingerprint(t *testing.T) {
	content := []byte("[api]\nlocale = \"en\"\n\n[scheduler]\npoll_interval = \"500ms\"\n")
	active := managerTestSnapshot(t, content)
	store := newManagerFakeStore(content)
	manager := newTestManager(t, store, active, nil)
	view, _ := manager.View(context.Background())
	request := UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:ordered", ExpectedRevision: view.SourceRevision, Confirm: true,
		Changes: []Change{
			{Key: KeySchedulerPollInterval, Value: "700ms"},
			{Key: KeyAPILocale, Unset: true},
		},
	}
	result, err := manager.Update(context.Background(), request)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	committed := store.lastCommit()
	if !reflect.DeepEqual(committed.ChangedKeys, []Key{KeyAPILocale, KeySchedulerPollInterval}) ||
		committed.Fingerprint == "" || !validManagerRevision(Revision(committed.Fingerprint)) {
		t.Fatalf("commit request = %#v", committed)
	}
	if bytes.Contains(committed.Replacement, []byte("locale")) || !bytes.Contains(committed.Replacement, []byte("poll_interval = \"700ms\"")) {
		t.Fatalf("canonical replacement = %s", committed.Replacement)
	}

	reordered := request
	reordered.Changes = []Change{request.Changes[1], request.Changes[0]}
	replay, err := manager.Update(context.Background(), reordered)
	if err != nil || !replay.Replayed || !reflect.DeepEqual(replay.Receipt, result.Receipt) {
		t.Fatalf("reordered replay = %#v err=%v", replay, err)
	}
}

func TestManagerUpdateFingerprintHasStableVersionedVector(t *testing.T) {
	const revision = Revision("sha256:0000000000000000000000000000000000000000000000000000000000000000")
	got := updateFingerprint(UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:test", ExpectedRevision: revision,
	}, []normalizedManagerChange{{Key: KeyAPILocale, Value: "en"}})
	const want = "sha256:18363fb086f6aaa5311532ec7fd5647e7f4320e44a60dd1221fbe04ddce6d1ee"
	if got != want {
		t.Fatalf("versioned fingerprint = %s, want %s", got, want)
	}
}

func TestManagerReceiptIncludesOnlyKeysThatActuallyChanged(t *testing.T) {
	content := []byte("[api]\nlocale = \"es\"\n\n[scheduler]\npoll_interval = \"500ms\"\n")
	active := managerTestSnapshot(t, content)
	store := newManagerFakeStore(content)
	manager := newTestManager(t, store, active, nil)
	view, err := manager.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:mixed-diff", ExpectedRevision: view.SourceRevision, Confirm: true,
		Changes: []Change{
			{Key: KeyAPILocale, Value: "es"},
			{Key: KeySchedulerPollInterval, Value: "700ms"},
		},
	})
	if err != nil || !reflect.DeepEqual(result.Receipt.ChangedKeys, []Key{KeySchedulerPollInterval}) {
		t.Fatalf("mixed diff receipt = %#v err=%v", result.Receipt, err)
	}
	if committed := store.lastCommit(); !reflect.DeepEqual(committed.ChangedKeys, []Key{KeySchedulerPollInterval}) {
		t.Fatalf("mixed diff commit = %#v", committed)
	}
}

func TestManagerSourcePathRejectsRuntimeOverlapBeforeCommit(t *testing.T) {
	const sourcePath = "/tmp/orquesta-manager-source.toml"
	content := []byte("[api]\nlocale = \"es\"\n")
	active, err := Resolve(ResolveOptions{TOML: content, SourcePath: sourcePath})
	if err != nil {
		t.Fatalf("resolve active: %v", err)
	}
	store := newManagerFakeStore(content)
	manager, err := NewManager(ManagerOptions{
		Store: store, Active: active, SourcePath: sourcePath,
		Now: func() time.Time { return time.Date(2035, 1, 2, 3, 4, 5, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	view, _ := manager.View(context.Background())
	_, err = manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:overlap", ExpectedRevision: view.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyConfigEffectivePath, Value: sourcePath}},
	})
	if !HasErrorCode(err, ErrorCrossValidation) || store.commitCount() != 0 {
		t.Fatalf("overlap error=%v commits=%d", err, store.commitCount())
	}
}

func TestManagerReservedPathsRejectEveryStoreSidecarBeforeCommit(t *testing.T) {
	const sourcePath = "/tmp/orquesta-manager-reserved.toml"
	reservedPaths := []string{sourcePath + ".lock", sourcePath + ".next", sourcePath + ".receipts"}
	content := []byte("[api]\nlocale = \"es\"\n")
	active, err := Resolve(ResolveOptions{TOML: content, SourcePath: sourcePath})
	if err != nil {
		t.Fatalf("resolve active: %v", err)
	}
	for _, reservedPath := range reservedPaths {
		t.Run(reservedPath, func(t *testing.T) {
			store := newManagerFakeStore(content)
			optionsPaths := append([]string(nil), reservedPaths...)
			manager, err := NewManager(ManagerOptions{
				Store: store, Active: active, SourcePath: sourcePath, ReservedPaths: optionsPaths,
				Now: func() time.Time { return time.Date(2035, 1, 2, 3, 4, 5, 0, time.UTC) },
			})
			if err != nil {
				t.Fatalf("new manager: %v", err)
			}
			optionsPaths[0] = "/tmp/mutated-after-construction"
			view, err := manager.View(context.Background())
			if err != nil {
				t.Fatalf("view: %v", err)
			}
			_, err = manager.Update(context.Background(), UpdateRequest{
				ActorRef: "actor:test", RequestRef: "request:sidecar:" + reservedPath,
				ExpectedRevision: view.SourceRevision, Confirm: true,
				Changes: []Change{{Key: KeyConfigEffectivePath, Value: reservedPath}},
			})
			if !HasErrorCode(err, ErrorCrossValidation) || store.commitCount() != 0 {
				t.Fatalf("reserved overlap error=%v commits=%d", err, store.commitCount())
			}
		})
	}
}

func TestManagerReplayDoesNotRevalidateCommandAgainstLaterState(t *testing.T) {
	active := managerTestSnapshot(t, nil)
	store := newManagerFakeStore(nil)
	manager := newTestManager(t, store, active, nil)
	initial, _ := manager.View(context.Background())
	originalRequest := UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:timeout-original", ExpectedRevision: initial.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyRuntimeCodexTimeout, Value: "40m"}},
	}
	original, err := manager.Update(context.Background(), originalRequest)
	if err != nil {
		t.Fatalf("original update: %v", err)
	}
	later, err := manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:timeout-later", ExpectedRevision: original.View.SourceRevision, Confirm: true,
		Changes: []Change{
			{Key: KeyRuntimeCodexTimeout, Value: "5m"},
			{Key: KeySchedulerExecutionTimeout, Value: "10m"},
		},
	})
	if err != nil {
		t.Fatalf("later update: %v", err)
	}
	replayed, err := manager.Update(context.Background(), originalRequest)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Receipt, original.Receipt) ||
		replayed.View.SourceRevision != later.View.SourceRevision {
		t.Fatalf("cross-state replay = %#v err=%v", replayed, err)
	}
	document := store.documentSnapshot()
	if !bytes.Contains(document.Content, []byte("timeout = \"5m0s\"")) ||
		!bytes.Contains(document.Content, []byte("execution_timeout = \"10m0s\"")) ||
		bytes.Contains(document.Content, []byte("timeout = \"40m0s\"")) {
		t.Fatalf("replay changed later state: %s", document.Content)
	}
}

func TestManagerReplayDoesNotDependOnCurrentClock(t *testing.T) {
	content := []byte("[api]\nlocale = \"es\"\n")
	active := managerTestSnapshot(t, content)
	store := newManagerFakeStore(content)
	clockHealthy := true
	manager, err := NewManager(ManagerOptions{
		Store: store, Active: active,
		Now: func() time.Time {
			if !clockHealthy {
				return time.Time{}
			}
			return time.Date(2035, 1, 2, 3, 4, 5, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial, _ := manager.View(context.Background())
	request := UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:clock-replay", ExpectedRevision: initial.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyAPILocale, Value: "en"}},
	}
	committed, err := manager.Update(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	clockHealthy = false
	replayed, err := manager.Update(context.Background(), request)
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(replayed.Receipt, committed.Receipt) {
		t.Fatalf("clock-independent replay = %#v, %v", replayed, err)
	}
}

func TestManagerStaleReplayProbeCannotCommitAfterABA(t *testing.T) {
	content := []byte("[api]\nlocale = \"es\"\n")
	active := managerTestSnapshot(t, content)
	store := newManagerFakeStore(content)
	manager := newTestManager(t, store, active, nil)
	initial, err := manager.View(context.Background())
	if err != nil {
		t.Fatalf("initial view: %v", err)
	}
	original := store.documentSnapshot()
	advanced, err := manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:advance", ExpectedRevision: initial.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyAPILocale, Value: "en"}},
	})
	if err != nil || advanced.View.SourceRevision == original.Revision {
		t.Fatalf("advance = %#v err=%v", advanced, err)
	}
	store.setBeforeCommit(func(store *managerFakeStore, request CommitRequest) {
		store.document = cloneManagerDocument(original)
	})
	_, err = manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:never-committed", ExpectedRevision: original.Revision, Confirm: true,
		Changes: []Change{{Key: KeyAPILocale, Value: "es"}},
	})
	if !IsDocumentStoreError(err, DocumentStoreRevisionConflict) {
		t.Fatalf("ABA replay probe error = %v", err)
	}
	if document := store.documentSnapshot(); !reflect.DeepEqual(document, original) {
		t.Fatalf("replay probe wrote stale state: %#v want %#v", document, original)
	}
	if request := store.lastCommit(); !request.ReplayOnly {
		t.Fatalf("stale request was not replay-only: %#v", request)
	}
}

func TestManagerRejectsIncoherentDocumentStoreResults(t *testing.T) {
	content := []byte("[api]\nlocale = \"es\"\n")
	active := managerTestSnapshot(t, content)
	tests := []struct {
		name   string
		mutate func(CommitRequest, CommitResult) CommitResult
	}{
		{name: "document hash", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Document.Revision = Revision(managerSHA256([]byte("other")))
			return result
		}},
		{name: "different replacement", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Document.Content = []byte("[api]\nlocale = \"es\"\n\n[scheduler]\npoll_interval = \"700ms\"\n")
			result.Document.Revision = Revision(managerSHA256(result.Document.Content))
			result.Receipt.AfterRevision = result.Document.Revision
			return result
		}},
		{name: "actor", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Receipt.ActorRef = "actor:other"
			return result
		}},
		{name: "before revision", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Receipt.BeforeRevision = result.Receipt.AfterRevision
			return result
		}},
		{name: "changed keys", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Receipt.ChangedKeys = []Key{KeyServerListen}
			return result
		}},
		{name: "pending keys", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Receipt.PendingRestartKeys = []Key{Key("unknown.key")}
			return result
		}},
		{name: "replay no-op revisions", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Replayed = true
			result.Receipt.AfterRevision = result.Receipt.BeforeRevision
			return result
		}},
		{name: "non canonical UTC", mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Receipt.ChangedAt = result.Receipt.ChangedAt.In(time.FixedZone("offset", 3600))
			return result
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &managerAdversarialStore{document: StoredDocument{
				Revision: Revision(managerSHA256(content)), Content: append([]byte(nil), content...),
			}, mutate: test.mutate}
			manager := newTestManager(t, store, active, nil)
			_, err := manager.Update(context.Background(), UpdateRequest{
				ActorRef: "actor:test", RequestRef: "request:" + test.name,
				ExpectedRevision: store.document.Revision, Confirm: true,
				Changes: []Change{{Key: KeyAPILocale, Value: "en"}},
			})
			if !HasErrorCode(err, ErrorStoreResultInvalid) {
				t.Fatalf("incoherent result error = %v", err)
			}
		})
	}
}

func TestManagerViewRejectsIncoherentStoredDocument(t *testing.T) {
	content := []byte("[api]\nlocale = \"es\"\n")
	active := managerTestSnapshot(t, content)
	store := &managerAdversarialStore{document: StoredDocument{
		Revision: Revision(managerSHA256([]byte("different"))), Content: content,
	}}
	manager := newTestManager(t, store, active, nil)
	if _, err := manager.View(context.Background()); !HasErrorCode(err, ErrorStoreResultInvalid) {
		t.Fatalf("incoherent read error = %v", err)
	}
}

func TestManagerRejectsNonRestartPendingKeyOnReplay(t *testing.T) {
	content := []byte("[api]\nlocale = \"es\"\n")
	active := managerTestSnapshot(t, content)
	store := &managerAdversarialStore{
		document: StoredDocument{Revision: Revision(managerSHA256(content)), Content: content},
		mutate: func(_ CommitRequest, result CommitResult) CommitResult {
			result.Replayed = true
			result.Receipt.PendingRestartKeys = []Key{KeyAPILocale}
			return result
		},
	}
	manager := newTestManager(t, store, active, nil)
	definition := manager.registry.byKey[KeyAPILocale]
	definition.RestartRequired = false
	manager.registry.byKey[KeyAPILocale] = definition
	_, err := manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:hot-pending", ExpectedRevision: store.document.Revision, Confirm: true,
		Changes: []Change{{Key: KeyAPILocale, Value: "en"}},
	})
	if !HasErrorCode(err, ErrorStoreResultInvalid) {
		t.Fatalf("non-restart pending replay error = %v", err)
	}
}

func TestManagerRejectsNonReplayResultForReplayOnlyProbe(t *testing.T) {
	originalContent := []byte("[api]\nlocale = \"es\"\n")
	advancedContent := []byte("[api]\nlocale = \"en\"\n")
	active := managerTestSnapshot(t, originalContent)
	store := &managerAdversarialStore{document: StoredDocument{
		Revision: Revision(managerSHA256(advancedContent)), Content: advancedContent,
	}}
	manager := newTestManager(t, store, active, nil)
	_, err := manager.Update(context.Background(), UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:bad-replay", Confirm: true,
		ExpectedRevision: Revision(managerSHA256(originalContent)),
		Changes:          []Change{{Key: KeyAPILocale, Value: "es"}},
	})
	if !HasErrorCode(err, ErrorStoreResultInvalid) || !store.last.ReplayOnly {
		t.Fatalf("non-replay probe error=%v request=%#v", err, store.last)
	}
}

func TestManagerConcurrentStoresPreserveSingleCASWinner(t *testing.T) {
	content := []byte("[scheduler]\npoll_interval = \"500ms\"\n")
	active := managerTestSnapshot(t, content)
	store := newManagerFakeStore(content)
	reader := newTestManager(t, store, active, nil)
	view, _ := reader.View(context.Background())
	const writers = 12
	start := make(chan struct{})
	errorsByWriter := make([]error, writers)
	var wait sync.WaitGroup
	for index := 0; index < writers; index++ {
		manager := newTestManager(t, store, active, nil)
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, errorsByWriter[index] = manager.Update(context.Background(), UpdateRequest{
				ActorRef: "actor:race", RequestRef: "request:" + string(rune('a'+index)),
				ExpectedRevision: view.SourceRevision, Confirm: true,
				Changes: []Change{{Key: KeySchedulerPollInterval, Value: time.Duration(600+index) * time.Millisecond}},
			})
		}(index)
	}
	close(start)
	wait.Wait()
	successes, conflicts := 0, 0
	for _, err := range errorsByWriter {
		switch {
		case err == nil:
			successes++
		case IsDocumentStoreError(err, DocumentStoreRevisionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected writer error: %v", err)
		}
	}
	if successes != 1 || conflicts != writers-1 {
		t.Fatalf("success/conflict = %d/%d", successes, conflicts)
	}
}

func TestManagerDoctorClassifiesAndReportsConflictsWithoutStoreAccess(t *testing.T) {
	active := managerTestSnapshot(t, nil)
	store := newManagerFakeStore(nil)
	manager := newTestManager(t, store, active, nil)
	valid := DoctorRequest{Proposals: []DoctorProposal{
		{Key: "server.bind", TargetKey: KeyServerListen, SemanticRef: "orquesta.config.server.listen"},
		{Key: "api.language", TargetKey: KeyAPILocale, SemanticRef: "orquesta.config.api.language", Alias: "api.locale", RemoveAfterRevision: "2027-01-01.0"},
		{Key: "telemetry.sample_interval", SemanticRef: "orquesta.config.telemetry.sample_interval", GoName: "TelemetrySampleInterval", EnvAlias: "ORQUESTA_TELEMETRY_SAMPLE_INTERVAL", Type: "duration", Scope: "telemetry"},
	}}
	beforeReads, beforeCommits := store.readCount(), store.commitCount()
	report, err := manager.Doctor(context.Background(), valid)
	if err != nil || len(report.Accepted) != 3 || len(report.Conflicts) != 0 ||
		report.Accepted[0].Mode != DoctorReuse || report.Accepted[1].Mode != DoctorReplace || report.Accepted[2].Mode != DoctorNew {
		t.Fatalf("doctor report = %#v err=%v", report, err)
	}
	if store.readCount() != beforeReads || store.commitCount() != beforeCommits {
		t.Fatal("doctor accessed document store")
	}
	conflicts, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{
		{Key: "server.other", SemanticRef: "orquesta.config.server.other", GoName: "ServerListen", EnvAlias: "ORQUESTA_SERVER_LISTEN", Type: "string", Scope: "server"},
		{Key: "api.old", TargetKey: KeyAPILocale, SemanticRef: "orquesta.config.api.old", Alias: "api.locale"},
	}})
	if err != nil || len(conflicts.Conflicts) < 3 || len(conflicts.Accepted) != 0 {
		t.Fatalf("doctor conflicts = %#v err=%v", conflicts, err)
	}
	codes := make(map[DoctorConflictCode]bool)
	for _, conflict := range conflicts.Conflicts {
		codes[conflict.Code] = true
	}
	for _, code := range []DoctorConflictCode{DoctorConflictGoName, DoctorConflictEnvAlias, DoctorConflictRetirement} {
		if !codes[code] {
			t.Fatalf("doctor lacks conflict %s: %#v", code, conflicts)
		}
	}
	duplicateReuse, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{
		{Key: "server.bind", TargetKey: KeyServerListen, SemanticRef: "orquesta.config.server.listen"},
		{Key: "server.bind", TargetKey: KeyServerMCPPath, SemanticRef: "orquesta.config.server.mcp_path"},
	}})
	if err != nil || len(duplicateReuse.Accepted) != 1 || len(duplicateReuse.Conflicts) == 0 {
		t.Fatalf("duplicate reuse = %#v err=%v", duplicateReuse, err)
	}
	futureRevision, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{{
		Key: "api.language_future", TargetKey: KeyAPILocale, SemanticRef: "orquesta.config.api.language_future",
		Alias: "api.locale", RemoveAfterRevision: "2026-07-15.10",
	}}})
	if err != nil || len(futureRevision.Accepted) != 1 || len(futureRevision.Conflicts) != 0 {
		t.Fatalf("doctor future revision = %#v err=%v", futureRevision, err)
	}
	expiredRevision, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{{
		Key: "api.expired", TargetKey: KeyAPILocale, SemanticRef: "orquesta.config.api.expired",
		Alias: "api.locale", RemoveAfterRevision: "2026-07-15.7",
	}}})
	if err != nil || len(expiredRevision.Accepted) != 0 || len(expiredRevision.Conflicts) != 1 ||
		expiredRevision.Conflicts[0].Code != DoctorConflictRetirement {
		t.Fatalf("doctor expired revision = %#v err=%v", expiredRevision, err)
	}
	badGoName, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{{
		Key: "telemetry.bad", SemanticRef: "orquesta.config.telemetry.bad", GoName: "bad name",
		EnvAlias: "ORQUESTA_TELEMETRY_BAD", Type: "duration", Scope: "telemetry",
	}}})
	if err != nil || len(badGoName.Accepted) != 0 || len(badGoName.Conflicts) != 1 ||
		badGoName.Conflicts[0].Code != DoctorConflictGoName {
		t.Fatalf("invalid Go name = %#v err=%v", badGoName, err)
	}
	reuseMetadata, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{{
		Key: "server.bind", TargetKey: KeyServerListen, SemanticRef: "orquesta.config.server.listen",
		GoName: "ServerMCPPath", EnvAlias: "ORQUESTA_SERVER_MCP_PATH", Type: "duration", Scope: "runtime",
	}}})
	if err != nil || len(reuseMetadata.Accepted) != 0 || len(reuseMetadata.Conflicts) != 4 {
		t.Fatalf("contradictory reuse metadata = %#v err=%v", reuseMetadata, err)
	}
	invalidKeys, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{
		{Key: "telemetry", SemanticRef: "orquesta.config.telemetry", GoName: "TelemetryRoot", EnvAlias: "ORQUESTA_TELEMETRY_ROOT", Type: "string", Scope: "telemetry"},
		{Key: "server.listen.child", SemanticRef: "orquesta.config.server.listen.child", GoName: "ServerListenChild", EnvAlias: "ORQUESTA_SERVER_LISTEN_CHILD", Type: "string", Scope: "server"},
		{Key: "telemetry.sample", SemanticRef: "orquesta.config.telemetry.sample", GoName: "TelemetrySample", EnvAlias: "ORQUESTA_TELEMETRY_SAMPLE", Type: "string", Scope: "telemetry"},
		{Key: "telemetry.sample.child", SemanticRef: "orquesta.config.telemetry.sample.child", GoName: "TelemetrySampleChild", EnvAlias: "ORQUESTA_TELEMETRY_SAMPLE_CHILD", Type: "string", Scope: "telemetry"},
	}})
	if err != nil || len(invalidKeys.Accepted) != 1 || len(invalidKeys.Conflicts) != 3 {
		t.Fatalf("invalid/prefix keys = %#v err=%v", invalidKeys, err)
	}
	report.Accepted[0].Key = "tampered"
	again, err := manager.Doctor(context.Background(), valid)
	if err != nil || again.Accepted[0].Key == "tampered" {
		t.Fatalf("doctor result mutation escaped: %#v err=%v", again, err)
	}
}

func TestManagerDoctorAcceptedReplacementMatchesCanonicalRegistry(t *testing.T) {
	manager := newTestManager(t, newManagerFakeStore(nil), managerTestSnapshot(t, nil), nil)
	report, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{{
		Key: "api.language", TargetKey: KeyAPILocale, SemanticRef: "orquesta.config.api.language",
		Alias: "api.locale", RemoveAfterRevision: "2027-01-01.0",
	}}})
	if err != nil || len(report.Accepted) != 1 || len(report.Conflicts) != 0 {
		t.Fatalf("applicable replacement = %#v err=%v", report, err)
	}
	target, _ := manager.registry.definition(KeyAPILocale)
	replacement := report.Accepted[0]
	if replacement.GoName != target.GoName || replacement.EnvAlias != target.EnvAlias ||
		replacement.Type != string(target.Type) || replacement.Scope != target.Scope {
		t.Fatalf("replacement metadata not inherited from target: %+v target=%+v", replacement, target)
	}
	managerAssertDoctorReplacementApplies(t, replacement)

	invalidSemantic, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{
		{Key: "telemetry.semantic", SemanticRef: "telemetry.semantic", GoName: "TelemetrySemantic", EnvAlias: "ORQUESTA_TELEMETRY_SEMANTIC", Type: "string", Scope: "telemetry"},
		{Key: "api.semantic", TargetKey: KeyAPILocale, SemanticRef: "network.api.semantic", Alias: "api.locale", RemoveAfterRevision: "2027-01-01.0"},
	}})
	if err != nil || len(invalidSemantic.Accepted) != 0 || len(invalidSemantic.Conflicts) != 2 {
		t.Fatalf("non-canonical semantics = %#v err=%v", invalidSemantic, err)
	}
	for _, conflict := range invalidSemantic.Conflicts {
		if conflict.Code != DoctorConflictSemantic {
			t.Fatalf("non-canonical semantic produced %s: %#v", conflict.Code, invalidSemantic)
		}
	}

	incompatible, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{{
		Key: "api.incompatible", TargetKey: KeyAPILocale, SemanticRef: "orquesta.config.api.incompatible",
		Alias: "api.locale", RemoveAfterRevision: "2027-01-01.0", Type: "duration", Scope: "runtime",
	}}})
	if err != nil || len(incompatible.Accepted) != 0 || len(incompatible.Conflicts) != 2 {
		t.Fatalf("incompatible replacement = %#v err=%v", incompatible, err)
	}

	crossValidated, err := manager.Doctor(context.Background(), DoctorRequest{Proposals: []DoctorProposal{{
		Key: "server.address", TargetKey: KeyServerListen, SemanticRef: "orquesta.config.server.address",
		Alias: "server.listen", RemoveAfterRevision: "2027-01-01.0",
	}}})
	if err != nil || len(crossValidated.Accepted) != 0 || len(crossValidated.Conflicts) != 1 ||
		crossValidated.Conflicts[0].Code != DoctorConflictShape {
		t.Fatalf("cross-validated replacement = %#v err=%v", crossValidated, err)
	}
}

func managerAssertDoctorReplacementApplies(t *testing.T, decision DoctorDecision) {
	t.Helper()
	var source registryFile
	if err := json.Unmarshal([]byte(generatedRegistryJSON), &source); err != nil {
		t.Fatalf("decode generated registry: %v", err)
	}
	found := false
	for index := range source.Keys {
		if source.Keys[index].Key != decision.TargetKey {
			continue
		}
		found = true
		source.Keys[index].Key = decision.Key
		source.Keys[index].SemanticRef = decision.SemanticRef
		source.Keys[index].GoName = decision.GoName
		source.Keys[index].EnvAlias = decision.EnvAlias
		source.Keys[index].Type = valueType(decision.Type)
		source.Keys[index].Scope = decision.Scope
	}
	if !found {
		t.Fatalf("replacement target %s not found", decision.TargetKey)
	}
	for validatorIndex := range source.CrossValidators {
		for keyIndex := range source.CrossValidators[validatorIndex].Keys {
			if source.CrossValidators[validatorIndex].Keys[keyIndex] == decision.TargetKey {
				source.CrossValidators[validatorIndex].Keys[keyIndex] = decision.Key
			}
		}
	}
	source.Aliases = append(source.Aliases, registryAliasDefinition{
		Kind: AliasKindTOMLKey, Name: decision.Alias, Target: decision.Key,
		IntroducedRevision: source.Revision, RemoveAfterRevision: decision.RemoveAfterRevision,
	})
	payload, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("encode replacement registry: %v", err)
	}
	if err := ValidateRegistrySource(payload); err != nil {
		t.Fatalf("Doctor accepted replacement that registry rejects: %v: %v", err, errors.Unwrap(err))
	}
}

func TestManagerRejectsInvalidDependenciesRequestsAndContext(t *testing.T) {
	active := managerTestSnapshot(t, nil)
	store := newManagerFakeStore(nil)
	if _, err := NewManager(ManagerOptions{Store: store, Active: active}); !HasErrorCode(err, ErrorManagerInvalid) {
		t.Fatalf("missing clock = %v", err)
	}
	manager := newTestManager(t, store, active, nil)
	if _, err := manager.View(nil); !HasErrorCode(err, ErrorManagerInvalid) {
		t.Fatalf("nil view context = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.View(cancelled); !HasErrorCode(err, ErrorManagerInvalid) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled view = %v", err)
	}
	view, _ := manager.View(context.Background())
	base := UpdateRequest{
		ActorRef: "actor:test", RequestRef: "request:test", ExpectedRevision: view.SourceRevision, Confirm: true,
		Changes: []Change{{Key: KeyAPILocale, Value: "en"}},
	}
	tests := []struct {
		name   string
		mutate func(*UpdateRequest)
		code   ErrorCode
	}{
		{name: "confirmation", mutate: func(request *UpdateRequest) { request.Confirm = false }, code: ErrorUpdateConfirmation},
		{name: "actor", mutate: func(request *UpdateRequest) { request.ActorRef = "actor:\ninvalid" }, code: ErrorUpdateInvalid},
		{name: "revision", mutate: func(request *UpdateRequest) { request.ExpectedRevision = "bad" }, code: ErrorUpdateInvalid},
		{name: "empty changes", mutate: func(request *UpdateRequest) { request.Changes = nil }, code: ErrorUpdateInvalid},
		{name: "duplicate", mutate: func(request *UpdateRequest) { request.Changes = append(request.Changes, request.Changes[0]) }, code: ErrorUpdateInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := base
			request.Changes = append([]Change(nil), base.Changes...)
			test.mutate(&request)
			_, err := manager.Update(context.Background(), request)
			if !HasErrorCode(err, test.code) {
				t.Fatalf("error = %v, want %s", err, test.code)
			}
		})
	}
}

type managerFakeStore struct {
	mu           sync.Mutex
	document     StoredDocument
	receipts     map[string]managerFakeReceipt
	reads        int
	commits      int
	last         CommitRequest
	beforeCommit func(*managerFakeStore, CommitRequest)
}

type managerFakeReceipt struct {
	request CommitRequest
	result  CommitResult
}

func newManagerFakeStore(content []byte) *managerFakeStore {
	content = append([]byte{}, content...)
	return &managerFakeStore{
		document: StoredDocument{Revision: Revision(managerSHA256(content)), Content: content},
		receipts: make(map[string]managerFakeReceipt),
	}
}

func (store *managerFakeStore) Read(ctx context.Context) (StoredDocument, error) {
	if ctx == nil {
		return StoredDocument{}, errors.New("nil context")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.reads++
	return cloneManagerDocument(store.document), nil
}

func (store *managerFakeStore) Commit(ctx context.Context, request CommitRequest) (CommitResult, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.commits++
	store.last = cloneManagerCommitRequest(request)
	if beforeCommit := store.beforeCommit; beforeCommit != nil {
		store.beforeCommit = nil
		beforeCommit(store, cloneManagerCommitRequest(request))
	}
	identity := request.ActorRef + "\x00" + request.RequestRef
	if persisted, found := store.receipts[identity]; found {
		if persisted.request.Fingerprint != request.Fingerprint {
			return CommitResult{}, &DocumentStoreError{Code: DocumentStoreReplayConflict}
		}
		result := cloneManagerCommitResult(persisted.result)
		result.Document = cloneManagerDocument(store.document)
		result.Replayed = true
		return result, nil
	}
	if request.ReplayOnly {
		return CommitResult{}, &DocumentStoreError{Code: DocumentStoreRevisionConflict}
	}
	if store.document.Revision != request.ExpectedRevision {
		return CommitResult{}, &DocumentStoreError{Code: DocumentStoreRevisionConflict}
	}
	revision := Revision(managerSHA256(request.Replacement))
	result := CommitResult{
		Document: StoredDocument{Revision: revision, Content: append([]byte{}, request.Replacement...)},
		Receipt: ChangeReceipt{
			ReceiptRef: "receipt:" + request.RequestRef, ActorRef: request.ActorRef, RequestRef: request.RequestRef,
			Fingerprint: request.Fingerprint, BeforeRevision: request.ExpectedRevision, AfterRevision: revision,
			ChangedKeys: append([]Key{}, request.ChangedKeys...), PendingRestartKeys: append([]Key{}, request.PendingRestartKeys...),
			ChangedAt: request.ChangedAt,
		},
	}
	store.document = cloneManagerDocument(result.Document)
	store.receipts[identity] = managerFakeReceipt{request: cloneManagerCommitRequest(request), result: cloneManagerCommitResult(result)}
	return cloneManagerCommitResult(result), nil
}

func (store *managerFakeStore) setBeforeCommit(beforeCommit func(*managerFakeStore, CommitRequest)) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.beforeCommit = beforeCommit
}

func (store *managerFakeStore) revision() Revision {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.document.Revision
}

func (store *managerFakeStore) readCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.reads
}

func (store *managerFakeStore) commitCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.commits
}

func (store *managerFakeStore) lastCommit() CommitRequest {
	store.mu.Lock()
	defer store.mu.Unlock()
	return cloneManagerCommitRequest(store.last)
}

func (store *managerFakeStore) documentSnapshot() StoredDocument {
	store.mu.Lock()
	defer store.mu.Unlock()
	return cloneManagerDocument(store.document)
}

func newTestManager(t *testing.T, store DocumentStore, active Snapshot, environment map[string]string) *Manager {
	t.Helper()
	manager, err := NewManager(ManagerOptions{
		Store: store, Active: active, Environment: environment,
		Now: func() time.Time { return time.Date(2035, 1, 2, 3, 4, 5, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return manager
}

func managerTestSnapshot(t *testing.T, content []byte) Snapshot {
	t.Helper()
	snapshot, err := Resolve(ResolveOptions{TOML: append([]byte(nil), content...)})
	if err != nil {
		t.Fatalf("resolve active: %v", err)
	}
	return snapshot
}

func managerKeyView(t *testing.T, view ConfigView, key Key) ConfigKeyView {
	t.Helper()
	for _, candidate := range view.Keys {
		if candidate.Key == key {
			return candidate
		}
	}
	t.Fatalf("key view %s missing", key)
	return ConfigKeyView{}
}

func cloneManagerDocument(document StoredDocument) StoredDocument {
	document.Content = append([]byte{}, document.Content...)
	return document
}

func cloneManagerCommitRequest(request CommitRequest) CommitRequest {
	request.Replacement = append([]byte{}, request.Replacement...)
	request.ChangedKeys = append([]Key{}, request.ChangedKeys...)
	request.PendingRestartKeys = append([]Key{}, request.PendingRestartKeys...)
	return request
}

func cloneManagerCommitResult(result CommitResult) CommitResult {
	result.Document = cloneManagerDocument(result.Document)
	result.Receipt = cloneChangeReceipt(result.Receipt)
	return result
}

var _ DocumentStore = (*managerFakeStore)(nil)

type managerAdversarialStore struct {
	document StoredDocument
	last     CommitRequest
	mutate   func(CommitRequest, CommitResult) CommitResult
}

func (store *managerAdversarialStore) Read(context.Context) (StoredDocument, error) {
	return cloneManagerDocument(store.document), nil
}

func (store *managerAdversarialStore) Commit(_ context.Context, request CommitRequest) (CommitResult, error) {
	store.last = cloneManagerCommitRequest(request)
	revision := Revision(managerSHA256(request.Replacement))
	result := CommitResult{
		Document: StoredDocument{Revision: revision, Content: append([]byte(nil), request.Replacement...)},
		Receipt: ChangeReceipt{
			ReceiptRef: "receipt:adversarial", ActorRef: request.ActorRef, RequestRef: request.RequestRef,
			Fingerprint: request.Fingerprint, BeforeRevision: request.ExpectedRevision, AfterRevision: revision,
			ChangedKeys: append([]Key{}, request.ChangedKeys...), PendingRestartKeys: append([]Key{}, request.PendingRestartKeys...),
			ChangedAt: request.ChangedAt,
		},
	}
	if store.mutate != nil {
		result = store.mutate(cloneManagerCommitRequest(request), cloneManagerCommitResult(result))
	}
	return cloneManagerCommitResult(result), nil
}

var _ DocumentStore = (*managerAdversarialStore)(nil)
