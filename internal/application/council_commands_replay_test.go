package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/identity"
)

func TestCouncilRequiredOpenUsesRecordedAuthorizationTimeAndReplays(t *testing.T) {
	system, record := councilCommandGateSystem(t, council.PolicyRequired)
	principal, project, err := system.access.values()
	appTestNoError(t, err)
	accessRepository := system.orchestrator.access.(*memoryAccessRepository)
	accessRepository.setRole(principal.Ref, project, identity.RoleProjectOwner)
	lease, err := system.orchestrator.ClaimDirector(context.Background(), system.access,
		ClaimDirectorRequest{RequestRef: "claim:council-recorded-time", GoalRef: system.goalRef})
	appTestNoError(t, err)

	clock := system.orchestrator.clock.(*mutableClock)
	recordedAt := clock.Now().Add(time.Second).UTC()
	councilCommandDelayedAuthorizer(t, accessRepository, time.Second)
	item := record.Goal.WorkItems()[0]
	request := OpenCouncilRoundRequest{RequestRef: "open:council-recorded-time", GoalRef: system.goalRef,
		ChangeRef: record.ChangeSets[0].Ref, ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(), LeaseToken: lease.Lease.Token, LeaseFence: lease.Lease.Fence}

	first, err := system.orchestrator.OpenCouncilRound(context.Background(), system.access, request)
	appTestNoError(t, err)
	if !first.Created || !first.Round.OpenedAt.Equal(recordedAt) {
		t.Fatalf("open did not use authorization recorded time: %+v want=%s", first, recordedAt)
	}
	clock.Advance(5 * time.Second)
	replay, err := system.orchestrator.OpenCouncilRound(context.Background(), system.access, request)
	appTestNoError(t, err)
	if replay.Created || replay.Round != first.Round {
		t.Fatalf("open replay=%+v first=%+v", replay, first)
	}
}

func TestCouncilSkipUsesRecordedAuthorizationTimeAndReplays(t *testing.T) {
	system, record := councilCommandGateSystem(t, council.PolicySkipByOperator)
	principal, project, err := system.access.values()
	appTestNoError(t, err)
	accessRepository := system.orchestrator.access.(*memoryAccessRepository)
	accessRepository.setRole(principal.Ref, project, identity.RoleProjectOwner)

	clock := system.orchestrator.clock.(*mutableClock)
	recordedAt := clock.Now().Add(time.Second).UTC()
	councilCommandDelayedAuthorizer(t, accessRepository, time.Second)
	item := record.Goal.WorkItems()[0]
	request := SkipCouncilRequest{RequestRef: "skip:council-recorded-time", GoalRef: system.goalRef,
		ChangeRef: record.ChangeSets[0].Ref, ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(), Reason: "authorized future durable receipt"}

	first, err := system.orchestrator.SkipCouncil(context.Background(), system.access, request)
	appTestNoError(t, err)
	if !first.Created || !first.Skip.RecordedAt.Equal(recordedAt) || !first.Skip.Skip.RecordedAtUTC.Equal(recordedAt) {
		t.Fatalf("skip did not use authorization recorded time: %+v want=%s", first, recordedAt)
	}
	clock.Advance(5 * time.Second)
	replay, err := system.orchestrator.SkipCouncil(context.Background(), system.access, request)
	appTestNoError(t, err)
	if replay.Created || replay.Skip != first.Skip {
		t.Fatalf("skip replay=%+v first=%+v", replay, first)
	}
}

func councilCommandGateSystem(t *testing.T, policy council.Policy) (*testAttestationSystem, GoalRecord) {
	t.Helper()
	system := newCouncilSystem(t, policy)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	return system, system.record(t)
}

func councilCommandDelayedAuthorizer(t *testing.T, repository *memoryAccessRepository, delay time.Duration) {
	t.Helper()
	receipts := make(map[string]identity.AuthorizationReceipt)
	repository.authorizeHook = func(request identity.AuthorizationRequest) (identity.AuthorizationReceipt, error) {
		if receipt, found := receipts[request.RequestRef()]; found {
			return receipt, nil
		}
		decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
			Request: request, Outcome: identity.AuthorizationAllowed, Role: identity.RoleProjectOwner,
			MembershipRevision: 1, ReasonCode: "access.allowed", DecidedAt: request.RequestedAt(),
		})
		if err != nil {
			return identity.AuthorizationReceipt{}, err
		}
		receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
			Ref: "authorization-receipt:recorded-after:" + request.RequestRef(), Decision: decision,
			RecordedAt: request.RequestedAt().Add(delay),
		})
		if err != nil {
			return identity.AuthorizationReceipt{}, err
		}
		receipts[request.RequestRef()] = receipt
		return receipt, nil
	}
}
