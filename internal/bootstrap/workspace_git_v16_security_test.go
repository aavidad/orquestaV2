package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/auth/localtoken"
	gitlocal "orquesta/internal/adapters/workspace/gitlocal"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func v16Principal(t *testing.T, principalValue, actorValue string) identity.Principal {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef(principalValue)
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef(actorValue)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, localtoken.AuthenticationMethod,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func v16Hierarchy(t *testing.T, projectValue string) identity.ProjectHierarchy {
	t.Helper()
	workspaceRef, err := identity.NewWorkspaceRef(projectValue)
	if err != nil {
		t.Fatal(err)
	}
	groupRef, err := identity.NewGroupRef(projectValue)
	if err != nil {
		t.Fatal(err)
	}
	projectRef, err := goal.NewProjectRef(projectValue)
	if err != nil {
		t.Fatal(err)
	}
	repositoryRef, err := identity.NewRepositoryRef(projectValue)
	if err != nil {
		t.Fatal(err)
	}
	hierarchy, err := identity.NewProjectHierarchy(identity.ProjectHierarchyInput{
		WorkspaceRef: workspaceRef, GroupRef: groupRef, GroupParentWorkspaceRef: workspaceRef,
		ProjectRef: projectRef, ProjectParentGroupRef: groupRef,
		RepositoryRef: repositoryRef, RepositoryParentProjectRef: projectRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	return hierarchy
}

func v16OpaqueDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func v16WorkspaceBranch(ref ports.ExecutionWorkspaceRef) string {
	return "orquesta/workspaces/" + v16OpaqueDigest(ref.String())
}

func v16WorkspaceBranchRef(ref ports.ExecutionWorkspaceRef) string {
	return "refs/heads/" + v16WorkspaceBranch(ref)
}

func v16WorkspaceBaseRef(ref ports.ExecutionWorkspaceRef) string {
	return "refs/orquesta/workspace-bases/" + v16OpaqueDigest(ref.String())
}

func v16CreateCommitObject(
	t *testing.T,
	harness *v16Harness,
	record application.GoalRecord,
	claim application.ActionClaim,
) string {
	t.Helper()
	if len(record.WorkspaceBindings) != 1 || claim.Action.ChangeRef.String() == "" {
		t.Fatalf("commit crash fixture incomplete: bindings=%d claim=%+v", len(record.WorkspaceBindings), claim)
	}
	binding := record.WorkspaceBindings[0]
	workspace := harness.workspacePath(binding.Ref)
	v16Git(t, harness.git, workspace, "add", "--all")
	tree := v16Git(t, harness.git, workspace, "write-tree")
	stamp := claim.Action.EffectIntent.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	return v16GitInput(t, harness.git, workspace,
		[]byte("orquesta changeset "+claim.Action.ChangeRef.String()+"\n"),
		[]string{
			"GIT_AUTHOR_NAME=Orquesta", "GIT_AUTHOR_EMAIL=orquesta@local", "GIT_AUTHOR_DATE=" + stamp,
			"GIT_COMMITTER_NAME=Orquesta", "GIT_COMMITTER_EMAIL=orquesta@local",
			"GIT_COMMITTER_DATE=" + stamp,
		}, "commit-tree", tree, "-p", binding.BaseOID)
}

func v16RealAdapter(
	t *testing.T,
	harness *v16Harness,
	repositoryRef identity.RepositoryRef,
) *gitlocal.Adapter {
	t.Helper()
	adapter, err := gitlocal.New(gitlocal.Config{
		Root: harness.workspaceRoot, GitCommand: harness.git, Now: time.Now,
		Locator: localRepositoryLocator{
			repositoryRef: repositoryRef, seedPath: harness.seed,
			targetRef: harness.fixture.GitFixture.TargetRef,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func v16ApplyIntegrationOutsideSQLite(
	t *testing.T,
	harness *v16Harness,
	record application.GoalRecord,
	claim application.ActionClaim,
) {
	t.Helper()
	if len(record.ChangeSets) != 1 || len(record.WorkspaceBindings) != 1 {
		t.Fatalf("integration crash fixture incomplete: %+v", record)
	}
	change, binding := record.ChangeSets[0], record.WorkspaceBindings[0]
	request := ports.IntegrationRequest{
		ChangeSetRef: change.Ref, RepositoryRef: change.RepositoryRef,
		PrincipalRef: claim.Action.EffectIntent.ProposedBy, ProjectRef: change.ProjectRef,
		SourceOID: change.HeadOID, TargetRef: binding.TargetRef,
		ExpectedTargetOID: claim.Action.ExpectedTargetOID, ObjectFormat: change.ObjectFormat,
		IntentRef: claim.Action.EffectIntent.Ref, AttemptRef: "effect-attempt:v16-crash-injected",
		ActionFence: claim.Fence, IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey,
		RequestedAt: claim.Action.EffectIntent.CreatedAt,
	}
	result, err := v16RealAdapter(t, harness, change.RepositoryRef).Integrate(context.Background(), request)
	if err != nil || result.Status != ports.IntegrationStatusIntegrated {
		t.Fatalf("apply Git integration outside SQLite: result=%+v err=%v", result, err)
	}
}

func v16ReleaseRecoveredWorkspace(t *testing.T, harness *v16Harness, record application.GoalRecord) {
	t.Helper()
	if len(record.WorkspaceBindings) != 1 {
		t.Fatalf("release fixture bindings=%d", len(record.WorkspaceBindings))
	}
	binding := record.WorkspaceBindings[0]
	adapter := v16RealAdapter(t, harness, binding.RepositoryRef)
	request := ports.WorkspacePrepareRequest{
		WorkspaceRef: binding.Ref, PrincipalRef: binding.PrincipalRef, ActorRef: binding.ActorRef,
		ProjectRef: binding.ProjectRef, RepositoryRef: binding.RepositoryRef, GoalRef: binding.GoalRef,
		WorkItemRef: binding.WorkItemRef, ExecutionRef: binding.ExecutionRef,
		ExecutionAttempt: binding.ExecutionAttempt, PlanGeneration: binding.PlanGeneration,
		AppSpecGeneration: binding.AppSpecGeneration, AppSpecHash: binding.SpecHash,
		WriteSet: binding.WriteSet, WriteSetDigest: binding.WriteSetDigest, TargetRef: binding.TargetRef,
		IntentRef: binding.EffectIntentRef, AttemptRef: "effect-attempt:v16-release-recovery",
		ActionFence: binding.EffectFence + 1, IdempotencyKey: binding.EffectIntentRef,
		PreparedAt: binding.PreparedAt,
	}
	if _, err := adapter.Prepare(context.Background(), request); err != nil {
		t.Fatalf("recover workspace before release: %v", err)
	}
	released, err := adapter.Release(context.Background(), ports.WorkspaceReleaseRequest{
		WorkspaceRef: binding.Ref, RepositoryRef: binding.RepositoryRef, ExecutionRef: binding.ExecutionRef,
		IdempotencyKey: "release:v16:" + binding.Ref.String(), RequestedAt: time.Now().UTC(),
	})
	if err != nil || !released.Released {
		t.Fatalf("release recovered workspace=%+v err=%v", released, err)
	}
	if _, err := os.Lstat(harness.workspacePath(binding.Ref)); !os.IsNotExist(err) {
		t.Fatalf("released workspace still exists: %v", err)
	}
}

func v16AssertFactsPathFree(t *testing.T, harness *v16Harness, record application.GoalRecord) {
	t.Helper()
	private := []string{harness.root, harness.seed, harness.workspaceRoot, harness.git}
	private = append(private, harness.fixture.PrivateLeakMarkers...)
	facts := []any{
		record.WorkspaceBindings, record.ChangeSets, record.MergeObservations,
		record.IntegrationReceipts, record.EffectReceipts,
	}
	for _, fact := range facts {
		encoded := fmt.Sprintf("%+v", fact)
		for _, marker := range private {
			if marker != "" && strings.Contains(encoded, marker) {
				t.Fatalf("durable fact leaks private adapter data %q: %s", marker, encoded)
			}
		}
	}
	v16AssertFactTypesPathFree(t)
}

func v16AssertFactTypesPathFree(t *testing.T) {
	t.Helper()
	values := []any{
		application.WorkspaceBinding{}, application.ChangeSet{},
		application.MergeObservation{}, application.IntegrationReceipt{},
	}
	for _, value := range values {
		typeOf := reflect.TypeOf(value)
		var forbidden []string
		for index := 0; index < typeOf.NumField(); index++ {
			name := typeOf.Field(index).Name
			if name == "Path" || strings.HasSuffix(name, "Path") || name == "URL" ||
				name == "Argv" || name == "Environment" || name == "Secret" {
				forbidden = append(forbidden, name)
			}
		}
		if len(forbidden) != 0 {
			sort.Strings(forbidden)
			t.Fatalf("%s exposes private fields %v", typeOf.Name(), forbidden)
		}
	}
}
