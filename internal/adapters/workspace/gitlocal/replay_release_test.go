package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestCommitReplayRejectsArbitraryPermittedChild(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.workspacePath(prepare.WorkspaceRef)
	if err := os.WriteFile(filepath.Join(path, "allowed.txt"), []byte("arbitrary"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, path, "add", "allowed.txt")
	gitTest(t, path, "commit", "-m", "orquesta changeset change:test")
	if _, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared)); ErrorCodeOf(err) != CodeChangeConflict {
		t.Fatalf("arbitrary child replay err=%v", err)
	}
}

func TestPrepareCrashAfterEffectClaimKeepsPinnedBaseWhenTargetAdvances(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	repository, binding, err := adapter.repository(context.Background(), request.RepositoryRef)
	if err != nil {
		t.Fatal(err)
	}
	_, originalBase, format, err := adapter.prepareTarget(context.Background(), repository, binding.TargetRef)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ensureWorkspaceBaseRef(context.Background(), repository, request.WorkspaceRef,
		originalBase, format); err != nil {
		t.Fatal(err)
	}
	message := effectMarkerMessage("prepare", prepareRequestDigest(request))
	if err := adapter.ensureEffectClaim(context.Background(), repository, format, request.IdempotencyKey,
		message, request.PreparedAt, CodeWorkspaceConflict); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "allowed.txt"), []byte("advanced target"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repository, "add", "allowed.txt")
	gitTest(t, repository, "commit", "-m", "advance target after prepare claim")
	advanced := testGitOID(t, adapter, repository, binding.TargetRef)
	prepared, err := freshAdapter(t, adapter).Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BaseOID != originalBase || prepared.BaseOID == advanced {
		t.Fatalf("base changed across crash: got=%s pinned=%s target=%s", prepared.BaseOID, originalBase, advanced)
	}
}

func TestIntegrationLostCASRechecksExactMarkerPayload(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adapter.workspacePath(prepare.WorkspaceRef), "allowed.txt"),
		[]byte("lost CAS"), 0o600); err != nil {
		t.Fatal(err)
	}
	committed, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared))
	if err != nil {
		t.Fatal(err)
	}
	request := testIntegrationRequest(prepare, prepared, committed)
	first, err := adapter.Integrate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	repository, _, err := adapter.repository(context.Background(), request.RepositoryRef)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := adapter.reconcileIntegrationCAS(context.Background(), repository,
		markerRef(request.IdempotencyKey), request)
	if err != nil || !reflect.DeepEqual(replayed, first) {
		t.Fatalf("lost-CAS replay=%+v first=%+v err=%v", replayed, first, err)
	}
	tampered := request
	tampered.TargetRef = "refs/heads/different"
	if _, err := adapter.reconcileIntegrationCAS(context.Background(), repository,
		markerRef(request.IdempotencyKey), tampered); ErrorCodeOf(err) != CodeChangeConflict {
		t.Fatalf("lost-CAS changed payload err=%v", err)
	}
}

func TestWorkspaceReleaseReplaySurvivesAdapterRestart(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	if _, err := adapter.Prepare(context.Background(), prepare); err != nil {
		t.Fatal(err)
	}
	request := testReleaseRequest(prepare)
	first, err := adapter.Release(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Release(context.Background(), request)
	if err != nil || !reflect.DeepEqual(second, first) {
		t.Fatalf("same-process replay=%+v first=%+v err=%v", second, first, err)
	}
	fresh := freshAdapter(t, adapter)
	replayed, err := fresh.Release(context.Background(), request)
	if err != nil || !reflect.DeepEqual(replayed, first) {
		t.Fatalf("restart replay=%+v first=%+v err=%v", replayed, first, err)
	}
	changed := request
	changed.ExecutionRef = mustExecution(t, "execution:release-other")
	if _, err := fresh.Release(context.Background(), changed); ErrorCodeOf(err) != CodeWorkspaceConflict {
		t.Fatalf("same-key changed release err=%v", err)
	}
}

func TestWorkspaceReleaseReconcilesCrashAfterRemoval(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	if _, err := adapter.Prepare(context.Background(), prepare); err != nil {
		t.Fatal(err)
	}
	request := testReleaseRequest(prepare)
	repository, _, err := adapter.repository(context.Background(), request.RepositoryRef)
	if err != nil {
		t.Fatal(err)
	}
	format, err := adapter.repositoryObjectFormat(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	message := effectMarkerMessage("release", releaseRequestDigest(request))
	if err := adapter.ensureEffectClaim(context.Background(), repository, format, request.IdempotencyKey,
		message, request.RequestedAt, CodeWorkspaceConflict); err != nil {
		t.Fatal(err)
	}
	if err := adapter.releaseWorkspace(context.Background(), repository, request); err != nil {
		t.Fatal(err)
	}
	replayed, err := freshAdapter(t, adapter).Release(context.Background(), request)
	if err != nil || !replayed.Released {
		t.Fatalf("crash replay=%+v err=%v", replayed, err)
	}
}

func testReleaseRequest(prepare ports.WorkspacePrepareRequest) ports.WorkspaceReleaseRequest {
	return ports.WorkspaceReleaseRequest{
		WorkspaceRef: prepare.WorkspaceRef, RepositoryRef: prepare.RepositoryRef,
		ExecutionRef: prepare.ExecutionRef, IdempotencyKey: "release:test",
		RequestedAt: time.Unix(1_700_000_040, 0).UTC(),
	}
}

func freshAdapter(t *testing.T, adapter *Adapter) *Adapter {
	t.Helper()
	fresh, err := New(Config{Root: adapter.root, GitCommand: adapter.git, Locator: adapter.loc, Now: adapter.now})
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func testGitOID(t *testing.T, adapter *Adapter, repository, ref string) string {
	t.Helper()
	oid, err := adapter.gitOID(context.Background(), repository, ref)
	if err != nil {
		t.Fatal(err)
	}
	return oid
}
