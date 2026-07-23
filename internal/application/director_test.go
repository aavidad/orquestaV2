package application

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type directorTestSystem struct {
	clock             *mutableClock
	repository        *memoryRepository
	accessStore       *memoryAccessRepository
	orchestrator      *Orchestrator
	project           goal.ProjectRef
	owner             identity.Principal
	service           identity.Principal
	contributor       identity.Principal
	ownerAccess       Access
	serviceAccess     Access
	contributorAccess Access
	goal              GoalRecord
}

type directorStateSnapshot struct {
	goal             goal.GoalSnapshot
	executions       []ExecutionRecord
	actionCount      int
	eventCount       int
	directorLease    DirectorLeaseRecord
	directorRequests int
}

func TestDirectorLeaseClaimRenewTakeoverAndAuthorization(t *testing.T) {
	ctx := context.Background()
	system := newDirectorTestSystem(t)

	beforeDenied := system.snapshot(t)
	if _, err := system.orchestrator.ClaimDirector(ctx, system.contributorAccess, ClaimDirectorRequest{
		RequestRef: "director-claim:contributor", GoalRef: system.goal.Goal.Ref(),
	}); !v12Forbidden(err) {
		t.Fatalf("contributor claimed Director: %v", err)
	}
	system.assertState(t, beforeDenied)

	claimed, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, ClaimDirectorRequest{
		RequestRef: "director-claim:owner", GoalRef: system.goal.Goal.Ref(),
	})
	if err != nil || !claimed.Changed || claimed.Lease.PrincipalRef != system.owner.Ref ||
		claimed.Lease.Fence != 1 || claimed.Lease.Token == "" ||
		!claimed.Lease.LeaseUntil.Equal(system.clock.Now().Add(time.Minute)) {
		t.Fatalf("owner claim=%+v err=%v", claimed, err)
	}
	replayed, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, ClaimDirectorRequest{
		RequestRef: "director-claim:owner", GoalRef: system.goal.Goal.Ref(),
	})
	if err != nil || replayed.Changed || replayed.Lease != claimed.Lease {
		t.Fatalf("claim replay=%+v err=%v", replayed, err)
	}
	beforeMismatchEffects := system.sideEffectCounters()
	otherGoal, err := goal.NewGoalRef("goal:director-other")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, ClaimDirectorRequest{
		RequestRef: "director-claim:owner", GoalRef: otherGoal,
	}); !IsStateError(err, StateConflict) {
		t.Fatalf("claim payload mismatch error=%v", err)
	}
	if got := system.sideEffectCounters(); got != beforeMismatchEffects {
		t.Fatalf("claim mismatch performed auth/ID effect: got=%v want=%v", got, beforeMismatchEffects)
	}

	beforeConflict := system.snapshot(t)
	if _, err := system.orchestrator.ClaimDirector(ctx, system.serviceAccess, ClaimDirectorRequest{
		RequestRef: "director-claim:service-live", GoalRef: system.goal.Goal.Ref(),
	}); !IsStateError(err, StateAlreadyClaimed) {
		t.Fatalf("live lease takeover error=%v", err)
	}
	system.assertState(t, beforeConflict)

	if _, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, RenewDirectorRequest{
		RequestRef: "director-renew:wrong-token", GoalRef: system.goal.Goal.Ref(),
		Token: "director-lease-token:wrong", Fence: claimed.Lease.Fence,
	}); !IsStateError(err, StateConflict) {
		t.Fatalf("wrong token renewal error=%v", err)
	}
	if _, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, RenewDirectorRequest{
		RequestRef: "director-renew:wrong-fence", GoalRef: system.goal.Goal.Ref(),
		Token: claimed.Lease.Token, Fence: claimed.Lease.Fence + 1,
	}); !IsStateError(err, StateConflict) {
		t.Fatalf("wrong fence renewal error=%v", err)
	}

	system.clock.Advance(10 * time.Second)
	renewed, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, RenewDirectorRequest{
		RequestRef: "director-renew:owner", GoalRef: system.goal.Goal.Ref(),
		Token: claimed.Lease.Token, Fence: claimed.Lease.Fence,
	})
	if err != nil || !renewed.Changed || renewed.Lease.Token != claimed.Lease.Token ||
		renewed.Lease.Fence != claimed.Lease.Fence ||
		!renewed.Lease.LeaseUntil.After(claimed.Lease.LeaseUntil) {
		t.Fatalf("renewed=%+v err=%v", renewed, err)
	}
	renewReplay, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, RenewDirectorRequest{
		RequestRef: "director-renew:owner", GoalRef: system.goal.Goal.Ref(),
		Token: claimed.Lease.Token, Fence: claimed.Lease.Fence,
	})
	if err != nil || renewReplay.Changed || renewReplay.Lease != renewed.Lease {
		t.Fatalf("renew replay=%+v err=%v", renewReplay, err)
	}

	system.clock.Advance(time.Minute + time.Second)
	beforeTakeover := system.snapshot(t)
	takeover, err := system.orchestrator.ClaimDirector(ctx, system.serviceAccess, ClaimDirectorRequest{
		RequestRef: "director-claim:service-takeover", GoalRef: system.goal.Goal.Ref(),
	})
	if err != nil || !takeover.Changed || takeover.Lease.PrincipalRef != system.service.Ref ||
		takeover.Lease.Fence != claimed.Lease.Fence+1 || takeover.Lease.Token == claimed.Lease.Token {
		t.Fatalf("takeover=%+v err=%v", takeover, err)
	}
	afterTakeover := system.snapshot(t)
	if !reflect.DeepEqual(afterTakeover.goal, beforeTakeover.goal) ||
		!reflect.DeepEqual(afterTakeover.executions, beforeTakeover.executions) ||
		afterTakeover.actionCount != beforeTakeover.actionCount ||
		afterTakeover.eventCount != beforeTakeover.eventCount {
		t.Fatal("Director takeover mutated Goal plan or outbox")
	}
	if _, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, RenewDirectorRequest{
		RequestRef: "director-renew:stale-owner", GoalRef: system.goal.Goal.Ref(),
		Token: claimed.Lease.Token, Fence: claimed.Lease.Fence,
	}); !IsStateError(err, StateConflict) {
		t.Fatalf("stale owner renewed after takeover: %v", err)
	}
	beforeExpiredReplay := system.snapshot(t)
	beforeExpiredEffects := system.sideEffectCounters()
	if _, err := system.orchestrator.ClaimDirector(ctx, system.ownerAccess, ClaimDirectorRequest{
		RequestRef: "director-claim:owner", GoalRef: system.goal.Goal.Ref(),
	}); !IsStateError(err, StateConflict) {
		t.Fatalf("expired/superseded claim replay error=%v", err)
	}
	if _, err := system.orchestrator.RenewDirector(ctx, system.ownerAccess, RenewDirectorRequest{
		RequestRef: "director-renew:owner", GoalRef: system.goal.Goal.Ref(),
		Token: claimed.Lease.Token, Fence: claimed.Lease.Fence,
	}); !IsStateError(err, StateConflict) {
		t.Fatalf("expired/superseded renew replay error=%v", err)
	}
	system.assertState(t, beforeExpiredReplay)
	if got := system.sideEffectCounters(); got != beforeExpiredEffects {
		t.Fatalf("expired replay performed auth/ID effect: got=%v want=%v", got, beforeExpiredEffects)
	}
}

func TestDirectorPlanUsesGoalApplyPlanExistingOutboxAndIdempotency(t *testing.T) {
	ctx := context.Background()
	system := newDirectorTestSystem(t)
	claimed := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:plan-owner")
	source := system.goal
	firstRequest := ProposeDirectorPlanRequest{
		RequestRef: "director-plan:first", GoalRef: source.Goal.Ref(),
		ExpectedGoalRevision:   source.Goal.Revision(),
		ExpectedPlanGeneration: source.Goal.PlanGeneration(),
		LeaseToken:             claimed.Token, LeaseFence: claimed.Fence,
		Reason: "web_application alias and docs/*.md are recoverable; preserve and replan",
		Plan: PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:v12-research", Key: "phase:research",
				TemplateRef: "phase-template:v12-research",
			}},
			WorkItems: []WorkItemSpec{{
				Key: "work:research", Objective: "inspect recoverable input",
				Phase: "phase:research", Role: "role:researcher",
				WriteSet: []string{"docs"}, CouncilPolicy: "skip_by_operator", RequiredTests: requiredTestSpecs("required-test:director-research"),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	}
	before := system.snapshot(t)
	result, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, firstRequest)
	if err != nil || !result.Created {
		t.Fatalf("first proposal=%+v err=%v", result, err)
	}
	firstRecord, err := system.repository.GetGoal(ctx, source.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if firstRecord.Goal.Revision() != source.Goal.Revision()+1 ||
		firstRecord.Goal.PlanGeneration() != source.Goal.PlanGeneration()+1 ||
		firstRecord.Goal.Actor() != system.owner.ActorRef || firstRecord.Goal.WorkItemCount() != 2 ||
		len(firstRecord.Executions) != len(source.Executions)+1 ||
		result.Decision.PrincipalRef != system.owner.Ref || result.Decision.LeaseFence != claimed.Fence ||
		result.Decision.Reason != firstRequest.Reason {
		t.Fatalf("first proposal lost authority/causality: %+v", result)
	}
	items := firstRecord.Goal.WorkItems()
	if !reflect.DeepEqual(items[:source.Goal.WorkItemCount()], source.Goal.WorkItems()) ||
		items[1].Actor() != source.Goal.Actor() || items[1].Project() != source.Goal.Project() {
		t.Fatal("proposal rewrote prefix or attributed WorkItem to Director")
	}
	after := system.snapshot(t)
	if after.actionCount != before.actionCount+1 || after.eventCount != before.eventCount+2 {
		t.Fatalf("proposal did not use existing event/outbox: before=%+v after=%+v", before, after)
	}

	replay, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, firstRequest)
	if err != nil || replay.Created || replay.Decision != result.Decision {
		t.Fatalf("proposal replay=%+v err=%v", replay, err)
	}
	system.assertState(t, after)

	wrongToken := directorAppendRequest(
		firstRecord.Goal, claimed, "director-plan:wrong-token", "work:must-not-exist",
	)
	wrongToken.LeaseToken = "director-lease-token:wrong"
	if _, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, wrongToken); !IsStateError(err, StateConflict) {
		t.Fatalf("wrong token proposal error=%v", err)
	}
	system.assertState(t, after)

	system.clock.Advance(time.Minute + time.Second)
	expired := wrongToken
	expired.RequestRef = "director-plan:expired"
	expired.LeaseToken = claimed.Token
	if _, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, expired); !IsStateError(err, StateConflict) {
		t.Fatalf("expired Director proposal error=%v", err)
	}
	system.assertState(t, after)

	takeover := claimDirectorForTest(t, system, system.serviceAccess, "director-claim:plan-service")
	stale := wrongToken
	stale.RequestRef = "director-plan:stale-fence"
	stale.LeaseToken = claimed.Token
	stale.LeaseFence = claimed.Fence
	if _, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, stale); !IsStateError(err, StateConflict) {
		t.Fatalf("stale Director proposal error=%v", err)
	}

	current, err := system.repository.GetGoal(ctx, source.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	secondRequest := ProposeDirectorPlanRequest{
		RequestRef: "director-plan:service", GoalRef: current.Goal.Ref(),
		ExpectedGoalRevision: current.Goal.Revision(), ExpectedPlanGeneration: current.Goal.PlanGeneration(),
		LeaseToken: takeover.Token, LeaseFence: takeover.Fence,
		Reason: "service Director appends dependency without replacing creator",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "work:review", Objective: "review research", Phase: "phase:research",
			Role: "role:reviewer", Dependencies: []string{items[1].Ref().String()},
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	second, err := system.orchestrator.ProposeDirectorPlan(ctx, system.serviceAccess, secondRequest)
	if err != nil || !second.Created {
		t.Fatalf("service proposal=%+v err=%v", second, err)
	}
	secondRecord, err := system.repository.GetGoal(ctx, source.Goal.Ref())
	if err != nil || secondRecord.Goal.PlanGeneration() != 3 ||
		secondRecord.Goal.Actor() != system.owner.ActorRef || second.Decision.PrincipalRef != system.service.Ref ||
		secondRecord.Goal.WorkItems()[2].Actor() != system.owner.ActorRef ||
		len(secondRecord.Executions) != len(current.Executions) {
		t.Fatalf("service proposal=%+v err=%v", second, err)
	}

	currentState := system.snapshot(t)
	beforeLateReplayEffects := system.sideEffectCounters()
	lateReplay, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, firstRequest)
	if err != nil || lateReplay.Created || lateReplay.Decision.Ref != result.Decision.Ref ||
		lateReplay.Decision.AppliedPlanGeneration != 2 {
		t.Fatalf("late historical proposal replay=%+v err=%v", lateReplay, err)
	}
	system.assertState(t, currentState)
	if got := system.sideEffectCounters(); got != beforeLateReplayEffects {
		t.Fatalf("late proposal replay performed auth/ID effect: got=%v want=%v", got, beforeLateReplayEffects)
	}
	conflictingReplay := firstRequest
	conflictingReplay.Reason = "same request_ref with different payload"
	if _, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, conflictingReplay); !IsStateError(err, StateConflict) {
		t.Fatalf("proposal replay payload mismatch error=%v", err)
	}
	system.assertState(t, currentState)
	if got := system.sideEffectCounters(); got != beforeLateReplayEffects {
		t.Fatalf("conflicting replay performed auth/ID effect: got=%v want=%v", got, beforeLateReplayEffects)
	}
	activeMembership, err := system.accessStore.Membership(ctx, system.owner.Ref, system.project)
	if err != nil {
		t.Fatal(err)
	}
	revokedMembership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: activeMembership.PrincipalRef(), ProjectRef: activeMembership.ProjectRef(),
		Role: activeMembership.Role(), Revision: activeMembership.Revision() + 1,
		Status:    identity.MembershipRevoked,
		GrantedBy: activeMembership.GrantedBy(), GrantedAt: activeMembership.GrantedAt(),
		RevokedBy: system.service.Ref, RevokedAt: system.clock.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	system.accessStore.seedMembership(revokedMembership)
	beforeRevokedReplayEffects := system.sideEffectCounters()
	if _, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, firstRequest); !v12Forbidden(err) {
		t.Fatalf("revoked Director recovered historical replay: %v", err)
	}
	system.assertState(t, currentState)
	if got := system.sideEffectCounters(); got != beforeRevokedReplayEffects {
		t.Fatalf("revoked replay created auth/ID effect: got=%v want=%v", got, beforeRevokedReplayEffects)
	}
}

func directorAppendRequest(
	aggregate goal.Goal,
	lease DirectorLeaseRecord,
	requestRef string,
	itemKey string,
) ProposeDirectorPlanRequest {
	return ProposeDirectorPlanRequest{
		RequestRef: requestRef, GoalRef: aggregate.Ref(),
		ExpectedGoalRevision: aggregate.Revision(), ExpectedPlanGeneration: aggregate.PlanGeneration(),
		LeaseToken: lease.Token, LeaseFence: lease.Fence, Reason: "append recoverable work",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: itemKey, Objective: "must remain atomic", Phase: "phase:research",
			Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
}

func TestDirectorDependencyRejectsMissingExistingRefBeforeStateWrite(t *testing.T) {
	system := newDirectorTestSystem(t)
	lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:missing-ref")
	before := system.snapshot(t)
	_, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.ownerAccess, ProposeDirectorPlanRequest{
		RequestRef: "director-plan:missing-ref", GoalRef: system.goal.Goal.Ref(),
		ExpectedGoalRevision:   system.goal.Goal.Revision(),
		ExpectedPlanGeneration: system.goal.Goal.PlanGeneration(),
		LeaseToken:             lease.Token, LeaseFence: lease.Fence, Reason: "repairable plan form",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "work:child", Objective: "child", Phase: goal.DefaultPhaseKey().String(),
			Role: goal.DefaultRoleKey().String(), Dependencies: []string{"work-item:missing"},
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	if err == nil || err.Error() != "application.plan_dependency_unknown" {
		t.Fatalf("missing dependency error=%v", err)
	}
	system.assertState(t, before)
}

func newDirectorTestSystem(t *testing.T) *directorTestSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	accessStore := newMemoryAccessRepository()
	accessStore.defaultRole = ""
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestratorWithAccess(t, repository, accessStore, clock, agent)
	project, err := goal.NewProjectRef("project:director-test")
	if err != nil {
		t.Fatal(err)
	}
	owner := testPrincipal(t, "principal:director-owner", "actor:director-owner", identity.PrincipalKindHuman)
	service := testPrincipal(t, "principal:director-service", "actor:director-service", identity.PrincipalKindService)
	contributor := testPrincipal(t, "principal:director-contributor", "actor:director-contributor", identity.PrincipalKindHuman)
	seedDirectorMembership(t, accessStore, owner, project, identity.RoleProjectOwner, clock.Now())
	seedDirectorMembership(t, accessStore, service, project, identity.RoleOperator, clock.Now())
	seedDirectorMembership(t, accessStore, contributor, project, identity.RoleContributor, clock.Now())
	ownerAccess := mustDirectorAccess(t, owner, project)
	serviceAccess := mustDirectorAccess(t, service, project)
	contributorAccess := mustDirectorAccess(t, contributor, project)
	submitted, err := orchestrator.Submit(context.Background(), ownerAccess, SubmitRequest{
		RequestRef: "request:director-goal", Statement: "coordinate one transferable Goal", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit Director Goal: %v", err)
	}
	return &directorTestSystem{
		clock: clock, repository: repository, accessStore: accessStore, orchestrator: orchestrator,
		project: project, owner: owner, service: service, contributor: contributor,
		ownerAccess: ownerAccess, serviceAccess: serviceAccess, contributorAccess: contributorAccess,
		goal: submitted.Record,
	}
}

func seedDirectorMembership(
	t *testing.T,
	repository *memoryAccessRepository,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	role identity.Role,
	at time.Time,
) {
	t.Helper()
	membership, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: principal.Ref, ProjectRef: projectRef, Role: role,
		Revision: 1, Status: identity.MembershipActive,
		GrantedBy: principal.Ref, GrantedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	repository.seedMembership(membership)
}

func mustDirectorAccess(t *testing.T, principal identity.Principal, project goal.ProjectRef) Access {
	t.Helper()
	access, err := NewAccess(principal, project)
	if err != nil {
		t.Fatal(err)
	}
	return access
}

func claimDirectorForTest(
	t *testing.T,
	system *directorTestSystem,
	access Access,
	requestRef string,
) DirectorLeaseRecord {
	t.Helper()
	result, err := system.orchestrator.ClaimDirector(context.Background(), access, ClaimDirectorRequest{
		RequestRef: requestRef, GoalRef: system.goal.Goal.Ref(),
	})
	if err != nil || !result.Changed {
		t.Fatalf("claim %s: result=%+v err=%v", requestRef, result, err)
	}
	return result.Lease
}

func (system *directorTestSystem) snapshot(t *testing.T) directorStateSnapshot {
	t.Helper()
	system.repository.mu.Lock()
	defer system.repository.mu.Unlock()
	record, exists := system.repository.records[system.goal.Goal.Ref()]
	if !exists {
		t.Fatal("Director Goal missing")
	}
	return directorStateSnapshot{
		goal: record.Goal.Snapshot(), executions: append([]ExecutionRecord(nil), record.Executions...),
		actionCount: len(system.repository.actions), eventCount: len(system.repository.events),
		directorLease:    system.repository.directorLeases[system.goal.Goal.Ref()],
		directorRequests: len(system.repository.directorRequests),
	}
}

func (system *directorTestSystem) assertState(t *testing.T, want directorStateSnapshot) {
	t.Helper()
	if got := system.snapshot(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("partial Director effect:\ngot=%+v\nwant=%+v", got, want)
	}
}

func (system *directorTestSystem) sideEffectCounters() [2]int {
	system.accessStore.mu.Lock()
	authorizations := len(system.accessStore.authorizations)
	system.accessStore.mu.Unlock()
	ids := system.orchestrator.ids.(*sequentialIDs)
	ids.mu.Lock()
	allocated := ids.next
	ids.mu.Unlock()
	return [2]int{authorizations, allocated}
}

func v12Forbidden(err error) bool {
	return err != nil && strings.Contains(err.Error(), "application.forbidden")
}
