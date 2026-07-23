package bootstrap

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestRealGitSQLiteWorkspaceLifecycleEndToEnd(t *testing.T) {
	fixture := v16LoadFixture(t)
	t.Run("clean_explicit_integration_closes_goal", func(t *testing.T) {
		v16TestCleanLifecycle(t, fixture)
	})
	t.Run("conflict_and_stale_stay_pending_with_target_intact", func(t *testing.T) {
		v16TestConflictAndStale(t, fixture)
	})
}

func v16TestCleanLifecycle(t *testing.T, fixture v16E2EFixture) {
	change := fixture.Changes[0]
	harness := newV16Harness(t, fixture, map[string]v16Write{
		"clean-change": v16FixtureWrites(change.Writes),
	})
	goalRef := harness.submit(t, harness.access, "request:v16-real-clean", "clean-change", change.WriteSet)
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	record := harness.get(t, harness.access, goalRef)
	if record.Goal.State() != goal.GoalStateRunning || len(record.WorkspaceBindings) != 1 ||
		len(record.ChangeSets) != 1 || len(record.IntegrationReceipts) != 0 ||
		len(record.Artifacts) != 3 || len(record.Attestations) != 2 ||
		record.Executions[0].State != application.ExecutionAwaitingIntegration {
		t.Fatalf("PASS closed or skipped pending state: goal=%s bindings=%d changes=%d artifacts=%d attestations=%d integrations=%d execution=%s",
			record.Goal.State(), len(record.WorkspaceBindings), len(record.ChangeSets),
			len(record.Artifacts), len(record.Attestations), len(record.IntegrationReceipts), record.Executions[0].State)
	}
	harness.driveReviews(t, goalRef)
	record = harness.get(t, harness.access, goalRef)
	before := v16Git(t, harness.git, harness.seed, "rev-parse", fixture.GitFixture.TargetRef)
	pending := harness.pending(t, harness.access)
	if len(pending) != 1 || pending[0].ChangeSet.Ref != record.ChangeSets[0].Ref {
		t.Fatalf("pending before explicit integration=%+v", pending)
	}
	admitted, err := harness.runtime.Orchestrator().IntegrateChange(context.Background(), harness.access,
		application.IntegrateChangeRequest{
			RequestRef: "request:v16-integrate-clean", GoalRef: goalRef,
			ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: before,
		})
	if err != nil || !admitted.Created {
		t.Fatalf("explicit integrate admission=%+v err=%v", admitted, err)
	}
	harness.process(t, application.ActionIntegrateChange)
	closed := harness.get(t, harness.access, goalRef)
	after := v16Git(t, harness.git, harness.seed, "rev-parse", fixture.GitFixture.TargetRef)
	if closed.Goal.State() != goal.GoalStateSucceeded || before == after ||
		len(closed.IntegrationReceipts) != 1 ||
		closed.IntegrationReceipts[0].Status != ports.IntegrationStatusIntegrated ||
		len(harness.pending(t, harness.access)) != 0 {
		t.Fatalf("explicit integration did not close: goal=%s before=%s after=%s receipts=%+v",
			closed.Goal.State(), before, after, closed.IntegrationReceipts)
	}
	v16AssertFactsPathFree(t, harness, closed)
}

func v16TestConflictAndStale(t *testing.T, fixture v16E2EFixture) {
	conflict := fixture.Changes[1]
	harness := newV16Harness(t, fixture, map[string]v16Write{
		"conflict-change": v16FixtureWrites(conflict.Writes),
		"stale-change":    {"src/stale.txt": "stale candidate\n"},
	})
	conflictGoal := harness.submit(t, harness.access, "request:v16-conflict", "conflict-change", conflict.WriteSet)
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	harness.driveReviews(t, conflictGoal)
	staleGoal := harness.submit(t, harness.access, "request:v16-stale", "stale-change", []string{"src/stale.txt"})
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	harness.driveReviews(t, staleGoal)
	conflictRecord := harness.get(t, harness.access, conflictGoal)
	staleRecord := harness.get(t, harness.access, staleGoal)
	base := conflictRecord.WorkspaceBindings[0].BaseOID
	v16ApplyFixtureTarget(t, harness.git, harness.seed, fixture)
	advanced := v16Git(t, harness.git, harness.seed, "rev-parse", fixture.GitFixture.TargetRef)
	if advanced == base {
		t.Fatal("target fixture did not advance")
	}
	requests := []application.IntegrateChangeRequest{
		{RequestRef: "request:v16-integrate-conflict", GoalRef: conflictGoal,
			ChangeRef: conflictRecord.ChangeSets[0].Ref, ExpectedTargetOID: advanced},
		{RequestRef: "request:v16-integrate-stale", GoalRef: staleGoal,
			ChangeRef: staleRecord.ChangeSets[0].Ref, ExpectedTargetOID: base},
	}
	for _, request := range requests {
		if result, err := harness.runtime.Orchestrator().IntegrateChange(context.Background(), harness.access, request); err != nil || !result.Created {
			t.Fatalf("admit pending outcome=%+v err=%v", result, err)
		}
		harness.process(t, application.ActionIntegrateChange)
		if target := v16Git(t, harness.git, harness.seed, "rev-parse", fixture.GitFixture.TargetRef); target != advanced {
			t.Fatalf("non-applied integration mutated target: got=%s want=%s", target, advanced)
		}
	}
	v16AssertConflictAndStale(t, harness, conflictGoal, staleGoal)
}

func v16AssertConflictAndStale(t *testing.T, harness *v16Harness, conflictGoal, staleGoal goal.GoalRef) {
	conflicted := harness.get(t, harness.access, conflictGoal)
	staled := harness.get(t, harness.access, staleGoal)
	if len(conflicted.IntegrationReceipts) != 1 ||
		conflicted.IntegrationReceipts[0].Status != ports.IntegrationStatusConflicted ||
		len(staled.IntegrationReceipts) != 1 || staled.IntegrationReceipts[0].Status != ports.IntegrationStatusStale {
		t.Fatalf("structural outcomes conflict=%+v stale=%+v", conflicted.IntegrationReceipts, staled.IntegrationReceipts)
	}
	pending := harness.pending(t, harness.access)
	if len(pending) != 2 {
		t.Fatalf("conflict/stale not preserved pending: %+v", pending)
	}
	v16AssertFactsPathFree(t, harness, conflicted)
	v16AssertFactsPathFree(t, harness, staled)
}

func TestPendingChangesAreRBACScopedAndSurviveRestart(t *testing.T) {
	fixture := v16LoadFixture(t)
	harness := newV16Harness(t, fixture, map[string]v16Write{
		"owner-pending": {"src/owner.txt": "owner pending\n"},
		"bob-pending":   {"src/bob.txt": "bob pending\n"},
	})
	ctx := context.Background()
	owner, alphaHierarchy, err := localIdentityComposition(harness.runtime.config)
	if err != nil {
		t.Fatal(err)
	}
	bob := v16Principal(t, fixture.Actors[1].PrincipalRef, fixture.Actors[1].ActorRef)
	grant, err := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{
		RequestRef: "membership-request:v16-bob-alpha", Actor: owner, TargetRef: bob.Ref,
		ProjectRef: alphaHierarchy.ProjectRef(), Role: identity.RoleContributor,
		ExpectedRevision: 0, RequestedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if membership, _, created, err := harness.runtime.Orchestrator().GrantMembership(ctx, harness.access, grant, bob); err != nil || !created || membership.Role() != identity.RoleContributor {
		t.Fatalf("grant bob contributor=%+v created=%v err=%v", membership, created, err)
	}
	bobAlpha, err := application.NewAccess(bob, alphaHierarchy.ProjectRef())
	if err != nil {
		t.Fatal(err)
	}
	ownerGoal := harness.submit(t, harness.access, "request:v16-owner-pending", "owner-pending", []string{"src/owner.txt"})
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	harness.driveReviews(t, ownerGoal)
	bobGoal := harness.submit(t, bobAlpha, "request:v16-bob-pending", "bob-pending", []string{"src/bob.txt"})
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	harness.driveReviews(t, bobGoal)
	ownerRecord := harness.get(t, harness.access, ownerGoal)
	bobRecord := harness.get(t, bobAlpha, bobGoal)
	if ownerRecord.ChangeSets[0].ActorRef == bobRecord.ChangeSets[0].ActorRef {
		t.Fatal("actor fixture collapsed")
	}
	v16AssertPendingRBACAfterRestart(t, harness, bob, bobAlpha, alphaHierarchy, ownerRecord, bobRecord)
}

func v16AssertPendingRBACAfterRestart(
	t *testing.T,
	harness *v16Harness,
	bob identity.Principal,
	bobAlpha application.Access,
	alphaHierarchy identity.ProjectHierarchy,
	ownerRecord, bobRecord application.GoalRecord,
) {
	ctx := context.Background()
	beta := v16Hierarchy(t, "project:v16-beta")
	if err := harness.runtime.repository.ProvisionLocalAccess(ctx, bob, beta,
		identity.RoleProjectOwner, time.Now().UTC().Add(time.Second)); err != nil {
		t.Fatalf("provision beta: %v", err)
	}
	bobBeta, err := application.NewAccess(bob, beta.ProjectRef())
	if err != nil {
		t.Fatal(err)
	}
	harness.restart(t)
	ownerPending := harness.pending(t, harness.access)
	bobPending := harness.pending(t, bobAlpha)
	betaPending := harness.pending(t, bobBeta)
	if len(ownerPending) != 2 || len(bobPending) != 1 ||
		bobPending[0].ChangeSet.ActorRef != bob.ActorRef || len(betaPending) != 0 {
		t.Fatalf("RBAC pending owner=%d bob=%+v beta=%+v", len(ownerPending), bobPending, betaPending)
	}
	unknown := v16Principal(t, "principal:v16-unknown", "actor:v16-unknown")
	unknownAccess, err := application.NewAccess(unknown, alphaHierarchy.ProjectRef())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.runtime.Orchestrator().ListPendingChanges(ctx, unknownAccess,
		application.ListPendingChangesRequest{Limit: 100}); err == nil {
		t.Fatal("principal without membership listed pending changes")
	}
	for _, pending := range ownerPending {
		if pending.ChangeSet.ProjectRef != alphaHierarchy.ProjectRef() ||
			pending.ChangeSet.RepositoryRef != alphaHierarchy.RepositoryRef() {
			t.Fatalf("cross-project pending leak: %+v", pending.ChangeSet)
		}
	}
	v16AssertFactsPathFree(t, harness, ownerRecord)
	v16AssertFactsPathFree(t, harness, bobRecord)
}
