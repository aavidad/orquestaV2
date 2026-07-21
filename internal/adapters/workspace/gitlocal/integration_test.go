package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestConflictAndStaleIntegrationLeaveTargetUnchanged(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.workspacePath(prepare.WorkspaceRef)
	if err := os.WriteFile(filepath.Join(path, "allowed.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	change, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared))
	if err != nil {
		t.Fatal(err)
	}
	request := ports.IntegrationRequest{
		ChangeSetRef: change.ChangeSetRef, RepositoryRef: prepare.RepositoryRef,
		PrincipalRef: prepare.PrincipalRef, ProjectRef: prepare.ProjectRef,
		SourceOID: change.HeadOID, TargetRef: prepared.TargetRef, ExpectedTargetOID: prepared.BaseOID,
		ObjectFormat: prepared.ObjectFormat, IntentRef: "intent:integrate", AttemptRef: "attempt:integrate",
		ActionFence: 3, IdempotencyKey: "integrate:test", RequestedAt: time.Unix(1_700_000_020, 0).UTC(),
	}
	result, err := adapter.Integrate(context.Background(), request)
	if err != nil || result.Status != ports.IntegrationStatusIntegrated {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	stale := request
	stale.IdempotencyKey = "integrate:stale"
	result, err = adapter.Integrate(context.Background(), stale)
	if err != nil || result.Status != ports.IntegrationStatusStale || result.TargetAfterOID != prepared.BaseOID {
		t.Fatalf("stale result=%#v err=%v", result, err)
	}
}

func TestIntegrationReplayAllowsAdvancedRetryEnvelopeButRejectsStableSemanticPayloadConflict(t *testing.T) {
	testIntegrationRetryEnvelopeAndSemanticConflict(t)
}

func TestIntegrationReplayRejectsSameKeyWithDifferentPayload(t *testing.T) {
	testIntegrationRetryEnvelopeAndSemanticConflict(t)
}

func testIntegrationRetryEnvelopeAndSemanticConflict(t *testing.T) {
	t.Helper()
	adapter, prepare := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.workspacePath(prepare.WorkspaceRef)
	if err := os.WriteFile(filepath.Join(path, "allowed.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	change, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared))
	if err != nil {
		t.Fatal(err)
	}
	request := ports.IntegrationRequest{
		ChangeSetRef: change.ChangeSetRef, RepositoryRef: prepare.RepositoryRef,
		PrincipalRef: prepare.PrincipalRef, ProjectRef: prepare.ProjectRef,
		SourceOID: change.HeadOID, TargetRef: prepared.TargetRef, ExpectedTargetOID: prepared.BaseOID,
		ObjectFormat: prepared.ObjectFormat, IntentRef: "intent:integrate-payload",
		AttemptRef: "attempt:integrate-payload", ActionFence: 3,
		IdempotencyKey: "integrate:payload", RequestedAt: time.Unix(1_700_000_020, 0).UTC(),
	}
	if result, err := adapter.Integrate(context.Background(), request); err != nil || result.Status != ports.IntegrationStatusIntegrated {
		t.Fatalf("initial integration: result=%+v err=%v", result, err)
	}
	replay := request
	replay.AttemptRef, replay.ActionFence = "attempt:integrate-payload-replay", 4
	if result, err := adapter.Integrate(context.Background(), replay); err != nil || result.Status != ports.IntegrationStatusIntegrated {
		t.Fatalf("advanced retry envelope: result=%+v err=%v", result, err)
	}
	tampered := replay
	tampered.TargetRef = "refs/heads/not-the-original-target"
	if _, err := adapter.Integrate(context.Background(), tampered); ErrorCodeOf(err) != CodeChangeConflict {
		t.Fatalf("same-key stable semantic conflict err=%v, want %s", err, CodeChangeConflict)
	}
}

func TestConcurrentIntegrationCASPreservesLoserPending(t *testing.T) {
	adapter, requests, workspaceRefs := concurrentIntegrationFixture(t)
	reached := make(chan string, len(requests))
	reconciled := make(chan string, len(requests))
	release := make(chan struct{})
	adapter.integrationCASObserver = func(stage integrationCASStage, request ports.IntegrationRequest) {
		switch stage {
		case integrationCASReady:
			reached <- request.IdempotencyKey
			<-release
		case integrationCASReconcile:
			reconciled <- request.IdempotencyKey
		}
	}
	results, errors, done := runIntegrationPair(adapter, requests)
	readyKeys := make(map[string]bool, len(requests))
	for range requests {
		select {
		case key := <-reached:
			readyKeys[key] = true
		case <-time.After(10 * time.Second):
			close(release)
			t.Fatal("both integrations did not reach CAS boundary")
		}
	}
	if len(readyKeys) != len(requests) {
		close(release)
		t.Fatalf("CAS boundary keys=%v", readyKeys)
	}
	close(release)
	<-done
	assertConcurrentIntegrationCAS(t, adapter, requests, workspaceRefs, results, errors, reconciled)
}

func concurrentIntegrationFixture(t *testing.T) (*Adapter, []ports.IntegrationRequest, []ports.ExecutionWorkspaceRef) {
	t.Helper()
	adapter, firstPrepare := testAdapterAndPrepare(t)
	firstBinding, err := adapter.Prepare(context.Background(), firstPrepare)
	if err != nil {
		t.Fatal(err)
	}
	secondPrepare := firstPrepare
	secondPrepare.WorkspaceRef = mustWorkspace(t, "workspace:second")
	secondPrepare.ExecutionRef = mustExecution(t, "execution:second")
	secondPrepare.IdempotencyKey = "prepare:second"
	secondPrepare.IntentRef, secondPrepare.AttemptRef, secondPrepare.ActionFence = "intent:second", "attempt:second", 2
	secondBinding, err := adapter.Prepare(context.Background(), secondPrepare)
	if err != nil || firstBinding.BaseOID != secondBinding.BaseOID {
		t.Fatalf("shared target base: first=%s second=%s err=%v", firstBinding.BaseOID, secondBinding.BaseOID, err)
	}
	writeWorkspaceChange(t, adapter, firstPrepare.WorkspaceRef, "first")
	writeWorkspaceChange(t, adapter, secondPrepare.WorkspaceRef, "second")
	firstChange, err := adapter.Commit(context.Background(), testCommit(t, firstPrepare, firstBinding))
	if err != nil {
		t.Fatal(err)
	}
	secondCommit := testCommit(t, secondPrepare, secondBinding)
	secondCommit.ChangeSetRef, _ = ports.NewChangeSetRef("change:second")
	secondCommit.IdempotencyKey = "commit:second"
	secondCommit.IntentRef, secondCommit.AttemptRef, secondCommit.ActionFence = "intent:commit-second", "attempt:commit-second", 3
	secondChange, err := adapter.Commit(context.Background(), secondCommit)
	if err != nil {
		t.Fatal(err)
	}
	requests := []ports.IntegrationRequest{
		integrationRequest(firstPrepare, firstBinding, firstChange, "first", 4, 30),
		integrationRequest(secondPrepare, secondBinding, secondChange, "second", 5, 31),
	}
	workspaces := []ports.ExecutionWorkspaceRef{firstPrepare.WorkspaceRef, secondPrepare.WorkspaceRef}
	return adapter, requests, workspaces
}

func integrationRequest(prepare ports.WorkspacePrepareRequest, binding ports.WorkspacePrepared, change ports.CommitResult, suffix string, fence uint64, second int64) ports.IntegrationRequest {
	return ports.IntegrationRequest{
		ChangeSetRef: change.ChangeSetRef, RepositoryRef: prepare.RepositoryRef,
		PrincipalRef: prepare.PrincipalRef, ProjectRef: prepare.ProjectRef,
		SourceOID: change.HeadOID, TargetRef: binding.TargetRef, ExpectedTargetOID: binding.BaseOID,
		ObjectFormat: binding.ObjectFormat, IntentRef: "intent:integrate-" + suffix,
		AttemptRef: "attempt:integrate-" + suffix, ActionFence: fence,
		IdempotencyKey: "integrate:" + suffix, RequestedAt: time.Unix(1_700_000_000+second, 0).UTC(),
	}
}

func writeWorkspaceChange(t *testing.T, adapter *Adapter, workspace ports.ExecutionWorkspaceRef, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(adapter.workspacePath(workspace), "allowed.txt"), []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runIntegrationPair(adapter *Adapter, requests []ports.IntegrationRequest) ([]ports.IntegrationResult, []error, <-chan struct{}) {
	results := make([]ports.IntegrationResult, len(requests))
	errors := make([]error, len(requests))
	done := make(chan struct{})
	var group sync.WaitGroup
	for index := range requests {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			results[index], errors[index] = adapter.Integrate(context.Background(), requests[index])
		}(index)
	}
	go func() {
		group.Wait()
		close(done)
	}()
	return results, errors, done
}

func assertConcurrentIntegrationCAS(t *testing.T, adapter *Adapter, requests []ports.IntegrationRequest, workspaces []ports.ExecutionWorkspaceRef, results []ports.IntegrationResult, errors []error, reconciled <-chan string) {
	t.Helper()
	for _, err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	winner, loser := integrationWinnerAndLoser(results)
	if winner < 0 || loser < 0 || winner == loser {
		t.Fatalf("CAS outcomes=%#v", results)
	}
	select {
	case key := <-reconciled:
		if key != requests[loser].IdempotencyKey {
			t.Fatalf("reconciled key=%s loser=%s", key, requests[loser].IdempotencyKey)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("CAS loser did not traverse reconciliation")
	}
	if results[loser].MarkerRef != "" || results[loser].TargetAfterOID != requests[loser].ExpectedTargetOID {
		t.Fatalf("loser mutated evidence=%#v", results[loser])
	}
	repository := adapter.loc.(testLocator).binding.Path
	target, err := adapter.gitOID(context.Background(), repository, requests[winner].TargetRef)
	if err != nil || target != results[winner].TargetAfterOID {
		t.Fatalf("target=%s winner=%#v err=%v", target, results[winner], err)
	}
	loserHead, err := adapter.gitOID(context.Background(), adapter.workspacePath(workspaces[loser]), "HEAD")
	if err != nil || loserHead != requests[loser].SourceOID {
		t.Fatalf("loser changes not preserved: head=%s source=%s err=%v", loserHead, requests[loser].SourceOID, err)
	}
}

func integrationWinnerAndLoser(results []ports.IntegrationResult) (int, int) {
	winner, loser := -1, -1
	for index, result := range results {
		if result.Status == ports.IntegrationStatusIntegrated {
			winner = index
		} else if result.Status == ports.IntegrationStatusStale {
			loser = index
		}
	}
	return winner, loser
}
