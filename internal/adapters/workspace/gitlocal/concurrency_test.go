package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestWorkspaceConcurrentPrepareCommitIntegrateRace(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	prepareResults, prepareErrors := runConcurrent(16, func() (ports.WorkspacePrepared, error) {
		return adapter.Prepare(context.Background(), request)
	})
	prepared := requireConcurrentEqual(t, prepareResults, prepareErrors)
	path := adapter.workspacePath(request.WorkspaceRef)
	if err := os.WriteFile(filepath.Join(path, "allowed.txt"), []byte("concurrent"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitRequest := testCommit(t, request, prepared)
	commitResults, commitErrors := runConcurrent(16, func() (ports.CommitResult, error) {
		return adapter.Commit(context.Background(), commitRequest)
	})
	committed := requireConcurrentEqual(t, commitResults, commitErrors)
	integration := testIntegrationRequest(request, prepared, committed)
	integrationResults, integrationErrors := runConcurrent(16, func() (ports.IntegrationResult, error) {
		return adapter.Integrate(context.Background(), integration)
	})
	result := requireConcurrentEqual(t, integrationResults, integrationErrors)
	if result.Status != ports.IntegrationStatusIntegrated {
		t.Fatalf("integration status=%s", result.Status)
	}
}

func runConcurrent[T any](count int, action func() (T, error)) ([]T, []error) {
	results := make([]T, count)
	errors := make([]error, count)
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := 0; index < count; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			results[index], errors[index] = action()
		}(index)
	}
	close(start)
	group.Wait()
	return results, errors
}

func requireConcurrentEqual[T any](t *testing.T, results []T, errors []error) T {
	t.Helper()
	if len(results) == 0 || len(results) != len(errors) {
		t.Fatal("invalid concurrent result set")
	}
	for index, err := range errors {
		if err != nil {
			t.Fatalf("concurrent call %d: %v", index, err)
		}
		if !reflect.DeepEqual(results[index], results[0]) {
			t.Fatalf("concurrent result %d differs: %#v != %#v", index, results[index], results[0])
		}
	}
	return results[0]
}

func testIntegrationRequest(
	prepare ports.WorkspacePrepareRequest,
	prepared ports.WorkspacePrepared,
	committed ports.CommitResult,
) ports.IntegrationRequest {
	return ports.IntegrationRequest{
		ChangeSetRef: committed.ChangeSetRef, RepositoryRef: prepare.RepositoryRef,
		PrincipalRef: prepare.PrincipalRef, ProjectRef: prepare.ProjectRef,
		SourceOID: committed.HeadOID, TargetRef: prepared.TargetRef, ExpectedTargetOID: prepared.BaseOID,
		ObjectFormat: prepared.ObjectFormat, IntentRef: "intent:concurrent-integration",
		AttemptRef: "attempt:concurrent-integration", ActionFence: 3,
		IdempotencyKey: "integrate:concurrent", RequestedAt: time.Unix(1_700_000_020, 0).UTC(),
	}
}
