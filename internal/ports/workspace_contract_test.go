package ports

import (
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestWorkspaceContractsAreOpaqueAndCausal(t *testing.T) {
	workspace, err := NewExecutionWorkspaceRef("workspace:execution:one")
	if err != nil {
		t.Fatalf("workspace ref: %v", err)
	}
	principal, _ := identity.NewPrincipalRef("principal:one")
	repository, _ := identity.NewRepositoryRef("repository:one")
	actor, _ := goal.NewActorRef("actor:one")
	project, _ := goal.NewProjectRef("project:one")
	goalRef, _ := goal.NewGoalRef("goal:one")
	item, _ := goal.NewWorkItemRef("work-item:one")
	execution, _ := goal.NewExecutionRef("execution:one")
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	base := workspaceTestOID(GitObjectFormatSHA1, 'a')
	request := WorkspacePrepareRequest{
		WorkspaceRef: workspace, PrincipalRef: principal, ActorRef: actor, ProjectRef: project, RepositoryRef: repository,
		GoalRef: goalRef, WorkItemRef: item, ExecutionRef: execution, ExecutionAttempt: 1, PlanGeneration: 1,
		AppSpecGeneration: 1, AppSpecHash: workspaceTestHash(), WriteSet: []string{"internal/ports"},
		WriteSetDigest: WorkspaceWriteSetDigest([]string{"internal/ports"}), TargetRef: "ref:target", IntentRef: "intent:one", AttemptRef: "attempt:one",
		ActionFence: 1, IdempotencyKey: "idempotency:one", PreparedAt: now,
	}
	if err := ValidateWorkspacePrepareRequest(request); err != nil {
		t.Fatalf("valid prepare request: %v", err)
	}
	prepared := WorkspacePrepared{WorkspaceRef: workspace, RepositoryRef: repository, ExecutionRef: execution, TargetRef: request.TargetRef,
		BaseOID: base, ObjectFormat: GitObjectFormatSHA1, WriteSetDigest: request.WriteSetDigest, AdapterRef: "adapter:local",
		ReceiptRef: "receipt:workspace", PreparedAt: now}
	if err := ValidateWorkspacePrepared(request, prepared); err != nil {
		t.Fatalf("valid prepare receipt: %v", err)
	}
	request.WriteSet = []string{"internal/../secret"}
	if err := ValidateWorkspacePrepareRequest(request); WorkspaceContractErrorCode(err) != "workspace.write_set_invalid" {
		t.Fatalf("traversal write-set accepted: %v", err)
	}
}

func TestValidateGitOIDRejectsMalformedValues(t *testing.T) {
	if err := ValidateGitOID(workspaceTestOID(GitObjectFormatSHA1, 'a'), GitObjectFormatSHA1); err != nil {
		t.Fatalf("valid SHA-1 rejected: %v", err)
	}
	for _, value := range []string{"", "HEAD", workspaceTestOID(GitObjectFormatSHA256, 'a'), strings.Repeat("A", 40)} {
		if err := ValidateGitOID(value, GitObjectFormatSHA1); WorkspaceContractErrorCode(err) != "workspace.git_oid_invalid" {
			t.Fatalf("malformed OID %q accepted: %v", value, err)
		}
	}
}

func TestVersionControlContractsRejectChangedPathOutsideWriteSet(t *testing.T) {
	workspace, _ := NewExecutionWorkspaceRef("workspace:execution:one")
	change, _ := NewChangeSetRef("change:one")
	principal, _ := identity.NewPrincipalRef("principal:one")
	repository, _ := identity.NewRepositoryRef("repository:one")
	actor, _ := goal.NewActorRef("actor:one")
	project, _ := goal.NewProjectRef("project:one")
	goalRef, _ := goal.NewGoalRef("goal:one")
	item, _ := goal.NewWorkItemRef("work-item:one")
	execution, _ := goal.NewExecutionRef("execution:one")
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	base, parent, head, tree := workspaceTestOID(GitObjectFormatSHA1, 'a'), workspaceTestOID(GitObjectFormatSHA1, 'b'), workspaceTestOID(GitObjectFormatSHA1, 'c'), workspaceTestOID(GitObjectFormatSHA1, 'd')
	request := CommitRequest{ChangeSetRef: change, WorkspaceRef: workspace, PrincipalRef: principal, ActorRef: actor, ProjectRef: project,
		RepositoryRef: repository, GoalRef: goalRef, WorkItemRef: item, ExecutionRef: execution, ExecutionAttempt: 1, PlanGeneration: 1,
		AppSpecGeneration: 1, AppSpecHash: workspaceTestHash(), BaseOID: base, ObjectFormat: GitObjectFormatSHA1,
		WriteSet: []string{"internal/ports"}, WriteSetDigest: WorkspaceWriteSetDigest([]string{"internal/ports"}), IntentRef: "intent:commit",
		AttemptRef: "attempt:commit", ActionFence: 1, IdempotencyKey: "idempotency:commit", CommittedAt: now}
	result := CommitResult{ChangeSetRef: change, WorkspaceRef: workspace, RepositoryRef: repository, ExecutionRef: execution, BaseOID: base,
		ParentOID: parent, HeadOID: head, TreeOID: tree, ObjectFormat: GitObjectFormatSHA1, DiffDigest: workspaceTestHash(),
		ChangedPaths: []string{"outside.go"}, WriteSetDigest: request.WriteSetDigest, AdapterRef: "adapter:local", ReceiptRef: "receipt:commit", CommittedAt: now}
	if err := ValidateCommitResult(request, result); VersionControlContractErrorCode(err) != "version_control.commit_result_invalid" {
		t.Fatalf("out-of-scope commit accepted: %v", err)
	}
}

func TestIntegrationContractDoesNotClaimMarkerForUnappliedOutcome(t *testing.T) {
	change, _ := NewChangeSetRef("change:one")
	principal, _ := identity.NewPrincipalRef("principal:one")
	repository, _ := identity.NewRepositoryRef("repository:one")
	project, _ := goal.NewProjectRef("project:one")
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	base := workspaceTestOID(GitObjectFormatSHA1, 'a')
	request := IntegrationRequest{ChangeSetRef: change, RepositoryRef: repository, PrincipalRef: principal, ProjectRef: project,
		SourceOID: workspaceTestOID(GitObjectFormatSHA1, 'b'), TargetRef: "ref:target", ExpectedTargetOID: base,
		ObjectFormat: GitObjectFormatSHA1, IntentRef: "intent:integration", AttemptRef: "attempt:integration", ActionFence: 1,
		IdempotencyKey: "idempotency:integration", RequestedAt: now}
	result := IntegrationResult{ChangeSetRef: change, RepositoryRef: repository, SourceOID: request.SourceOID, TargetRef: request.TargetRef,
		TargetBeforeOID: base, TargetAfterOID: base, ObjectFormat: GitObjectFormatSHA1, Status: IntegrationStatusStale,
		MarkerRef: "refs/orquesta/effects/should-not-exist", ConflictDigest: workspaceTestHash(), AdapterRef: "adapter:local",
		ReceiptRef: "receipt:integration", RecordedAt: now}
	if err := ValidateIntegrationResult(request, result); VersionControlContractErrorCode(err) != "version_control.integration_nonclean_invalid" {
		t.Fatalf("unapplied stale integration claimed marker: %v", err)
	}
	result.MarkerRef = ""
	if err := ValidateIntegrationResult(request, result); err != nil {
		t.Fatalf("unapplied stale integration rejected: %v", err)
	}
}

func workspaceTestHash() string {
	return string(make([]byte, 0)) + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}

func workspaceTestOID(format GitObjectFormat, digit byte) string {
	length := 40
	if format == GitObjectFormatSHA256 {
		length = 64
	}
	return string(makeRepeatedByte(digit, length))
}

func makeRepeatedByte(value byte, length int) []byte {
	output := make([]byte, length)
	for index := range output {
		output[index] = value
	}
	return output
}
