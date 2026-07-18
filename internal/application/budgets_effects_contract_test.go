package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestEffectRequiresExactLiveApprovalBeforeAdapterInvocation(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		assertNoEffectClaim(t, fixture)
	})
	t.Run("denied", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:denied", EffectDenied)
		assertNoEffectClaim(t, fixture)
	})
	t.Run("new_denial_supersedes_approval", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:approved-before-denial", EffectApproved)
		decideFixtureEffect(t, fixture, fixture.access, "request:latest-denial", EffectDenied)
		assertNoEffectClaim(t, fixture)
	})
	t.Run("expired", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Second)
		decideFixtureEffect(t, fixture, fixture.access, "request:expired", EffectApproved)
		fixture.clock.Advance(2 * time.Second)
		assertNoEffectClaim(t, fixture)
	})
	t.Run("future_decision", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:future-decision", EffectApproved)
		fixture.repository.memoryRepository.mu.Lock()
		record := fixture.repository.memoryRepository.records[fixture.intent.Subject.GoalRef]
		approval := &record.EffectApprovals[len(record.EffectApprovals)-1]
		approval.DecidedAt = fixture.clock.Now().Add(time.Second)
		approval.ExpiresAt = approval.DecidedAt.Add(fixture.intent.ApprovalTTL)
		fixture.repository.memoryRepository.records[fixture.intent.Subject.GoalRef] = record
		fixture.repository.memoryRepository.mu.Unlock()
		result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:future-decision")
		if !result.Processed || err == nil || fixture.agent.launches != 0 {
			t.Fatalf("future approval crossed adapter: result=%+v launches=%d err=%v", result, fixture.agent.launches, err)
		}
	})
	t.Run("wrong_digest", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		_, err := fixture.orchestrator.DecideEffect(context.Background(), fixture.access, DecideEffectRequest{
			RequestRef: "request:wrong-digest", GoalRef: fixture.intent.Subject.GoalRef,
			IntentRef: fixture.intent.Ref, ExpectedIntentDigest: strings.Repeat("f", 64),
			Decision: EffectApproved, Reason: "wrong digest must fail",
		})
		if !IsStateError(err, StateConflict) {
			t.Fatalf("wrong digest error=%v", err)
		}
		assertNoEffectClaim(t, fixture)
	})
	t.Run("wrong_project", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		principal, _, err := fixture.access.values()
		if err != nil {
			t.Fatal(err)
		}
		otherProject, _ := goal.NewProjectRef("project:effect-other")
		wrongAccess, _ := NewAccess(principal, otherProject)
		_, err = fixture.orchestrator.DecideEffect(context.Background(), wrongAccess, DecideEffectRequest{
			RequestRef: "request:wrong-project", GoalRef: fixture.intent.Subject.GoalRef,
			IntentRef: fixture.intent.Ref, ExpectedIntentDigest: fixture.intent.Digest,
			Decision: EffectApproved, Reason: "wrong project must fail",
		})
		if err == nil {
			t.Fatal("wrong project approved")
		}
		assertNoEffectClaim(t, fixture)
	})
	t.Run("wrong_target", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		_, err := fixture.orchestrator.DecideEffect(context.Background(), fixture.access, DecideEffectRequest{
			RequestRef: "request:wrong-target", GoalRef: fixture.intent.Subject.GoalRef,
			IntentRef: "effect-intent:other", ExpectedIntentDigest: fixture.intent.Digest,
			Decision: EffectApproved, Reason: "wrong target must fail",
		})
		if err == nil {
			t.Fatal("wrong target approved")
		}
		assertNoEffectClaim(t, fixture)
	})
	t.Run("wrong_policy", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:wrong-policy", EffectApproved)
		fixture.repository.memoryRepository.mu.Lock()
		record := fixture.repository.memoryRepository.records[fixture.intent.Subject.GoalRef]
		record.EffectApprovals[len(record.EffectApprovals)-1].PolicyHash = strings.Repeat("e", 64)
		fixture.repository.memoryRepository.records[fixture.intent.Subject.GoalRef] = record
		fixture.repository.memoryRepository.mu.Unlock()
		assertNoEffectClaim(t, fixture)
	})
	t.Run("wrong_policy_revision", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:wrong-policy-revision", EffectApproved)
		fixture.repository.memoryRepository.mu.Lock()
		record := fixture.repository.memoryRepository.records[fixture.intent.Subject.GoalRef]
		record.EffectApprovals[len(record.EffectApprovals)-1].PolicyRevision++
		fixture.repository.memoryRepository.records[fixture.intent.Subject.GoalRef] = record
		fixture.repository.memoryRepository.mu.Unlock()
		assertNoEffectClaim(t, fixture)
	})
	t.Run("wrong_fence", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:wrong-fence", EffectApproved)
		fixture.orchestrator.state = corruptReservationFenceState{StateRepository: fixture.repository}
		result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:wrong-fence")
		if !result.Processed || err == nil || fixture.agent.launches != 0 {
			t.Fatalf("wrong fence crossed adapter: result=%+v launches=%d err=%v",
				result, fixture.agent.launches, err)
		}
		assertOneExactRelease(t, fixture.repository.memoryRepository, fixture.intent.Subject.GoalRef)
	})
	t.Run("pre_call_cas", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:pre-call-cas", EffectApproved)
		fixture.orchestrator.state = launchPrepareConflictState{StateRepository: fixture.repository}
		result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:pre-call-cas")
		if err != nil || !result.Processed || fixture.agent.launches != 0 {
			t.Fatalf("prepare CAS crossed adapter: result=%+v launches=%d err=%v",
				result, fixture.agent.launches, err)
		}
		assertOneExactRelease(t, fixture.repository.memoryRepository, fixture.intent.Subject.GoalRef)
	})
	t.Run("revoked", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		owner := testPrincipal(t, "principal:effect-owner", "actor:effect-owner", identity.PrincipalKindHuman)
		project := fixture.intent.Subject.ProjectRef
		fixture.accessRepository.setRole(owner.Ref, project, identity.RoleProjectOwner)
		ownerAccess := mustDirectorAccess(t, owner, project)
		decideFixtureEffect(t, fixture, ownerAccess, "request:revoked", EffectApproved)
		active, err := fixture.accessRepository.Membership(context.Background(), owner.Ref, project)
		if err != nil {
			t.Fatal(err)
		}
		revoked, err := identity.NewMembership(identity.MembershipInput{
			PrincipalRef: owner.Ref, ProjectRef: project, Role: active.Role(), Revision: active.Revision() + 1,
			Status: identity.MembershipRevoked, GrantedBy: active.GrantedBy(), GrantedAt: active.GrantedAt(),
			RevokedBy: owner.Ref, RevokedAt: fixture.clock.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
		fixture.accessRepository.seedMembership(revoked)
		result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:revoked")
		if err != nil || !result.Processed || fixture.agent.launches != 0 {
			t.Fatalf("revoked authority crossed adapter: result=%+v launches=%d err=%v",
				result, fixture.agent.launches, err)
		}
		assertOneExactRelease(t, fixture.repository.memoryRepository, fixture.intent.Subject.GoalRef)
	})
	t.Run("live", func(t *testing.T) {
		fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
		decideFixtureEffect(t, fixture, fixture.access, "request:live", EffectApproved)
		result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:live")
		record, loadErr := fixture.repository.GetGoal(context.Background(), fixture.intent.Subject.GoalRef)
		if err != nil || loadErr != nil || !result.Processed || fixture.agent.launches != 1 ||
			len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 1 {
			t.Fatalf("live approval did not invoke exactly once: result=%+v launches=%d attempts=%d receipts=%d err=%v/%v",
				result, fixture.agent.launches, len(record.EffectAttempts), len(record.EffectReceipts), err, loadErr)
		}
	})
}

func TestEffectCrashAfterApplyBeforeReceiptReconcilesOnce(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 18, 16, 0, 0, 0, time.UTC)}
	physical, crashed := 0, false
	var durable ports.AgentLaunchReceipt
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("reconciled"),
		Usage: governance.ResourceUsage{
			Resources: governance.ResourceVector{Tokens: 10, MoneyMicros: 20, Currency: "EUR"},
			Known:     governance.ResourceTokens | governance.ResourceMoney, Quality: governance.UsageQualityExact,
		},
	}}}
	agent.launchOverride = func(request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
		if durable.ReceiptRef == "" {
			physical++
			durable = launchReceiptForRequest(request, clock.Now())
		}
		if !crashed {
			crashed = true
			return ports.AgentLaunchReceipt{}, errors.New("provider disconnected after applying launch")
		}
		return durable, nil
	}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:crash-after-apply", Statement: "reconcile idempotently", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:crash"); err != nil {
		t.Fatal(err)
	}
	crashedRecord, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || physical != 1 || len(crashedRecord.BudgetReservations) != 1 ||
		len(crashedRecord.BudgetSettlements) != 0 || len(crashedRecord.EffectAttempts) != 1 {
		t.Fatalf("ambiguous crash accounting: physical=%d reservations=%d settlements=%d attempts=%d err=%v",
			physical, len(crashedRecord.BudgetReservations), len(crashedRecord.BudgetSettlements),
			len(crashedRecord.EffectAttempts), err)
	}
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:reconcile"); err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:observe"); err != nil {
		t.Fatal(err)
	}
	closed, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || physical != 1 || agent.launches != 2 || len(closed.BudgetReservations) != 1 ||
		len(closed.BudgetSettlements) != 1 || len(closed.EffectAttempts) != 2 || len(closed.EffectReceipts) != 1 {
		t.Fatalf("reconciliation duplicated effect/facts: physical=%d calls=%d reservations=%d settlements=%d attempts=%d receipts=%d err=%v",
			physical, agent.launches, len(closed.BudgetReservations), len(closed.BudgetSettlements),
			len(closed.EffectAttempts), len(closed.EffectReceipts), err)
	}
	if closed.BudgetSettlements[0].ReservationRef != closed.BudgetReservations[0].Ref {
		t.Fatalf("settled wrong reservation: %+v", closed.BudgetSettlements[0])
	}
	if closed.BudgetSettlements[0].Observed.Quality != governance.UsageQualityMeasured {
		t.Fatalf("mixed exact provider/measured application usage kept false quality: %+v",
			closed.BudgetSettlements[0].Observed)
	}
}

type definitelyUnappliedTemporaryError struct{}

func (definitelyUnappliedTemporaryError) Error() string              { return "test.capacity" }
func (definitelyUnappliedTemporaryError) Temporary() bool            { return true }
func (definitelyUnappliedTemporaryError) DefinitelyNotApplied() bool { return true }

func TestTemporaryLaunchReleasesOnlyWithDefinitelyNotAppliedEvidence(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 18, 17, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, launchErr: definitelyUnappliedTemporaryError{}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:definitely-unapplied", Statement: "release exact capacity slot", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:no-capacity"); err != nil {
		t.Fatal(err)
	}
	assertOneExactRelease(t, repository, submitted.Record.Goal.Ref())
	agent.mu.Lock()
	agent.launchErr = nil
	agent.mu.Unlock()
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:capacity-restored"); err != nil {
		t.Fatal(err)
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || len(record.BudgetReservations) != 2 || agent.launches != 2 {
		t.Fatalf("released slot not reusable: reservations=%d launches=%d err=%v",
			len(record.BudgetReservations), agent.launches, err)
	}
}

func assertNoEffectClaim(t *testing.T, fixture effectDecisionFixture) {
	t.Helper()
	result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:no-approval")
	record, loadErr := fixture.repository.GetGoal(context.Background(), fixture.intent.Subject.GoalRef)
	if err != nil || loadErr != nil || result.Processed || fixture.agent.launches != 0 ||
		len(record.BudgetReservations) != 0 || len(record.EffectAttempts) != 0 {
		t.Fatalf("effect admitted without live approval: result=%+v launches=%d reservations=%d attempts=%d err=%v/%v",
			result, fixture.agent.launches, len(record.BudgetReservations), len(record.EffectAttempts), err, loadErr)
	}
}

func decideFixtureEffect(
	t *testing.T,
	fixture effectDecisionFixture,
	access Access,
	requestRef string,
	decision EffectDecision,
) EffectApproval {
	t.Helper()
	result, err := fixture.orchestrator.DecideEffect(context.Background(), access, DecideEffectRequest{
		RequestRef: requestRef, GoalRef: fixture.intent.Subject.GoalRef, IntentRef: fixture.intent.Ref,
		ExpectedIntentDigest: fixture.intent.Digest, Decision: decision, Reason: "exact approval decision",
	})
	if err != nil || !result.Created {
		t.Fatalf("decide effect: result=%+v err=%v", result, err)
	}
	return result.Approval
}

func assertOneExactRelease(t *testing.T, repository *memoryRepository, goalRef goal.GoalRef) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), goalRef)
	if err != nil || len(record.BudgetReservations) != 1 || len(record.BudgetSettlements) != 1 {
		t.Fatalf("exact release facts missing: reservations=%d settlements=%d err=%v",
			len(record.BudgetReservations), len(record.BudgetSettlements), err)
	}
	settlement := record.BudgetSettlements[0]
	zero := governance.ResourceVector{Currency: settlement.Reserved.Currency}
	if settlement.Charged != zero || settlement.Released != settlement.Reserved ||
		settlement.Observed.Known != governance.AllResourceDimensions ||
		settlement.Observed.Quality != governance.UsageQualityExact {
		t.Fatalf("release is not exact zero: %+v", settlement)
	}
}

func assertActionReservationReleased(t *testing.T, record GoalRecord, actionRef string) {
	t.Helper()
	for _, reservation := range record.BudgetReservations {
		if reservation.ActionRef != actionRef {
			continue
		}
		for _, settlement := range record.BudgetSettlements {
			zero := governance.ResourceVector{Currency: settlement.Reserved.Currency}
			if settlement.ReservationRef == reservation.Ref && settlement.Charged == zero &&
				settlement.Released == settlement.Reserved && settlement.Observed.Quality == governance.UsageQualityExact {
				return
			}
		}
	}
	t.Fatalf("action %s has no exact-zero release", actionRef)
}

func launchReceiptForRequest(request ports.AgentLaunchRequest, at time.Time) ports.AgentLaunchReceipt {
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
		ExternalRef: "external:" + request.ExecutionRef.String(), ReceiptRef: "receipt:crash:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey, AcceptedAt: at.UTC(),
	}
}

func TestDefinitelyNotAppliedSignalIsStructuralEvidence(t *testing.T) {
	ambiguous := temporaryAgentTestError{}
	definite := definitelyUnappliedTemporaryError{}
	permanent := definitelyUnappliedPermanentError{}
	if isDefinitelyNotAppliedAgentError(ambiguous) || !isDefinitelyNotAppliedAgentError(definite) ||
		!isDefinitelyNotAppliedAgentError(permanent) {
		t.Fatal("structural unapplied signal classification failed")
	}
}

type corruptReservationFenceState struct{ StateRepository }

func (state corruptReservationFenceState) ClaimNextAction(
	ctx context.Context,
	request ClaimRequest,
) (ActionClaim, bool, error) {
	claim, found, err := state.StateRepository.ClaimNextAction(ctx, request)
	if found && err == nil {
		claim.BudgetReservation.Fence = claim.Fence + 1
	}
	return claim, found, err
}

type launchPrepareConflictState struct{ StateRepository }

func (state launchPrepareConflictState) RecordLaunchPrepared(context.Context, LaunchPreparedState) error {
	return &StateError{Code: StateConflict}
}
