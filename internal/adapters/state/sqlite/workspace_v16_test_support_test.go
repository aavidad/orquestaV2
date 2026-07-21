package sqlite

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

type sqliteTestWorkspaceManager struct {
	mu       sync.Mutex
	prepared map[ports.ExecutionWorkspaceRef]ports.WorkspacePrepared
}

type sqliteTestVersionControl struct {
	mu           sync.Mutex
	commits      map[ports.ChangeSetRef]ports.CommitResult
	integrations map[string]ports.IntegrationResult
}

func processSQLiteWorkspaceLaunches(
	t *testing.T,
	orchestrator *application.Orchestrator,
	worker string,
	count int,
) {
	t.Helper()
	launched := 0
	for steps := 0; launched < count && steps < count*2; steps++ {
		result, err := orchestrator.ProcessNext(context.Background(), worker)
		if err != nil || !result.Processed ||
			(result.Action != application.ActionPrepareWorkspace && result.Action != application.ActionLaunchAgent) {
			t.Fatalf("workspace launch %d/%d: result=%+v err=%v", launched, count, result, err)
		}
		if result.Action == application.ActionLaunchAgent {
			launched++
		}
	}
	if launched != count {
		t.Fatalf("workspace launches=%d want=%d", launched, count)
	}
}

func (manager *sqliteTestWorkspaceManager) Prepare(
	_ context.Context,
	request ports.WorkspacePrepareRequest,
) (ports.WorkspacePrepared, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if previous, found := manager.prepared[request.WorkspaceRef]; found {
		return previous, nil
	}
	prepared := ports.WorkspacePrepared{
		WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, TargetRef: "refs/heads/main",
		BaseOID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ObjectFormat: ports.GitObjectFormatSHA1,
		WriteSetDigest: request.WriteSetDigest, AdapterRef: "workspace-adapter:sqlite-test",
		ReceiptRef: "workspace-receipt:" + request.WorkspaceRef.String(), PreparedAt: request.PreparedAt,
	}
	if manager.prepared == nil {
		manager.prepared = make(map[ports.ExecutionWorkspaceRef]ports.WorkspacePrepared)
	}
	manager.prepared[request.WorkspaceRef] = prepared
	return prepared, nil
}

func (manager *sqliteTestWorkspaceManager) Inspect(
	_ context.Context,
	request ports.WorkspaceInspectRequest,
) (ports.WorkspaceInspection, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	prepared, found := manager.prepared[request.WorkspaceRef]
	if !found {
		return ports.WorkspaceInspection{}, errors.New("sqlite_test.workspace_missing")
	}
	return ports.WorkspaceInspection{
		WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, BaseOID: prepared.BaseOID, HeadOID: prepared.BaseOID,
		TreeOID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", WriteSetDigest: prepared.WriteSetDigest,
		AdapterRef: prepared.AdapterRef, InspectedAt: prepared.PreparedAt,
	}, nil
}

func (manager *sqliteTestWorkspaceManager) Release(
	_ context.Context,
	request ports.WorkspaceReleaseRequest,
) (ports.WorkspaceReleaseReceipt, error) {
	return ports.WorkspaceReleaseReceipt{
		WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, Released: false,
		ReceiptRef: "workspace-release:" + request.WorkspaceRef.String(), ReleasedAt: request.RequestedAt,
	}, nil
}

func (control *sqliteTestVersionControl) Commit(
	_ context.Context,
	request ports.CommitRequest,
) (ports.CommitResult, error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if previous, found := control.commits[request.ChangeSetRef]; found {
		return previous, nil
	}
	result := ports.CommitResult{
		ChangeSetRef: request.ChangeSetRef, WorkspaceRef: request.WorkspaceRef,
		RepositoryRef: request.RepositoryRef, ExecutionRef: request.ExecutionRef,
		BaseOID: request.BaseOID, ParentOID: request.BaseOID,
		HeadOID: sqliteTestGitOID('c'), TreeOID: sqliteTestGitOID('d'),
		ObjectFormat: request.ObjectFormat, DiffDigest: strings.Repeat("e", 64),
		ChangedPaths: append([]string(nil), request.WriteSet...), WriteSetDigest: request.WriteSetDigest,
		ParentChangeRef: request.ParentChangeRef, AdapterRef: "version-control:sqlite-test",
		ReceiptRef: "commit-receipt:" + request.ChangeSetRef.String(), CommittedAt: request.CommittedAt,
	}
	if control.commits == nil {
		control.commits = make(map[ports.ChangeSetRef]ports.CommitResult)
	}
	control.commits[request.ChangeSetRef] = result
	return result, nil
}

func (*sqliteTestVersionControl) PreviewIntegration(
	_ context.Context,
	request ports.IntegrationPreviewRequest,
) (ports.IntegrationPreview, error) {
	return ports.IntegrationPreview{
		ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetOID: request.TargetOID,
		ObjectFormat: request.ObjectFormat, Status: ports.MergeStatusClean,
		CandidateTreeOID: sqliteTestGitOID('f'), AdapterRef: "version-control:sqlite-test",
		ObservedAt: request.RequestedAt,
	}, nil
}

func (control *sqliteTestVersionControl) Integrate(
	_ context.Context,
	request ports.IntegrationRequest,
) (ports.IntegrationResult, error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if previous, found := control.integrations[request.IdempotencyKey]; found {
		return previous, nil
	}
	result := ports.IntegrationResult{
		ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef,
		TargetBeforeOID: request.ExpectedTargetOID, TargetAfterOID: sqliteTestGitOID('1'),
		TreeOID: sqliteTestGitOID('f'), ObjectFormat: request.ObjectFormat,
		Status: ports.IntegrationStatusIntegrated, MarkerRef: "marker:sqlite-test",
		AdapterRef: "version-control:sqlite-test",
		ReceiptRef: "integration-receipt:" + request.ChangeSetRef.String(),
		RecordedAt: request.RequestedAt,
	}
	if control.integrations == nil {
		control.integrations = make(map[string]ports.IntegrationResult)
	}
	control.integrations[request.IdempotencyKey] = result
	return result, nil
}

func newSQLiteV16Orchestrator(
	t *testing.T,
	system *sqliteV15System,
) *application.Orchestrator {
	t.Helper()
	orchestrator, err := application.New(application.Dependencies{
		State: system.repository, Access: system.repository,
		Launcher: system.external, Observer: system.external, Controller: system.external,
		Artifacts: system.external, WorkspaceManager: &sqliteTestWorkspaceManager{},
		VersionControl: &sqliteTestVersionControl{}, Clock: system.clock, IDs: system.ids,
		MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6,
		ClaimLease: time.Minute, DirectorLeaseDuration: 30 * time.Second,
		EffectApprovalTTL: system.policy.EffectApprovalTTL, BudgetPolicy: system.policy,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return orchestrator
}

func sqliteTestGitOID(value byte) string {
	return strings.Repeat(string([]byte{value}), 40)
}
