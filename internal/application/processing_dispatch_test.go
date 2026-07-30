package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"orquesta/internal/goal"
)

func TestProcessNextComposesClaimAndProcessingEquivalently(t *testing.T) {
	witness := errors.New("state.processing_witness")
	claim := processingDispatchClaim(t)
	combinedState := &processingClaimState{claim: claim, found: true, getGoalErr: witness}
	combined := &Orchestrator{state: combinedState, ids: &sequentialIDs{}}

	combinedResult, combinedErr := combined.ProcessNext(context.Background(), "worker:dispatch")

	separateState := &processingClaimState{claim: claim, found: true, getGoalErr: witness}
	separate := &Orchestrator{state: separateState, ids: &sequentialIDs{}}
	separateClaim, found, claimErr := separate.ClaimNextAction(
		context.Background(), "worker:dispatch", ActionClaimSelection{},
	)
	if claimErr != nil || !found {
		t.Fatalf("ClaimNextAction() found=%v error=%v", found, claimErr)
	}
	separateResult, separateErr := separate.ProcessClaim(context.Background(), separateClaim)

	if combinedResult != separateResult ||
		!errors.Is(combinedErr, witness) || !errors.Is(separateErr, witness) {
		t.Fatalf("combined=%+v/%v separate=%+v/%v",
			combinedResult, combinedErr, separateResult, separateErr)
	}
	if combinedState.claimCalls != 1 || combinedState.getGoalCalls != 1 ||
		separateState.claimCalls != 1 || separateState.getGoalCalls != 1 ||
		!reflect.DeepEqual(combinedState.request, separateState.request) {
		t.Fatalf("combined calls=%d/%d separate=%d/%d requests=%+v/%+v",
			combinedState.claimCalls, combinedState.getGoalCalls,
			separateState.claimCalls, separateState.getGoalCalls,
			combinedState.request, separateState.request)
	}
}

func TestClaimNextActionOnlyClaimsAndProjectsSelection(t *testing.T) {
	claim := processingDispatchClaim(t)
	state := &processingClaimState{claim: claim, found: true}
	agent := &scriptedAgent{}
	orchestrator := &Orchestrator{state: state, ids: &sequentialIDs{}, launcher: agent}

	got, found, err := orchestrator.ClaimNextAction(
		context.Background(), "worker:no-launch", ActionClaimSelection{ExcludeLaunch: true},
	)

	if err != nil || !found || !reflect.DeepEqual(got, claim) {
		t.Fatalf("claim=%+v found=%v error=%v", got, found, err)
	}
	if state.claimCalls != 1 || state.getGoalCalls != 0 ||
		!state.request.ExcludeLaunch || agent.launches != 0 {
		t.Fatalf("claim calls=%d processing=%d exclude_launch=%v provider_launches=%d",
			state.claimCalls, state.getGoalCalls, state.request.ExcludeLaunch, agent.launches)
	}
}

func TestProcessClaimConsumesTheExactFencedClaimOnce(t *testing.T) {
	claim := processingDispatchClaim(t)
	claim.Action.Kind = ActionKind("dispatch_witness")
	state := &processingDispatchState{}
	orchestrator := &Orchestrator{
		state: state,
		clock: &mutableClock{},
	}

	result, err := orchestrator.ProcessClaim(context.Background(), claim)

	want := ProcessResult{
		Processed: true,
		GoalRef:   claim.Action.GoalRef,
		Action:    claim.Action.Kind,
	}
	if result != want || err == nil ||
		err.Error() != "application.action_kind_invalid:dispatch_witness" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	if state.claimCalls != 0 || len(state.quarantines) != 1 ||
		!reflect.DeepEqual(state.quarantines[0].Claim, claim) {
		t.Fatalf("claim calls=%d quarantines=%+v", state.claimCalls, state.quarantines)
	}
}

func TestClaimAndProcessValidateNilWorkerAndErrors(t *testing.T) {
	var unavailable *Orchestrator
	if _, _, err := unavailable.ClaimNextAction(
		context.Background(), "worker:test", ActionClaimSelection{},
	); err == nil || err.Error() != "application.unavailable" {
		t.Fatalf("nil ClaimNextAction() error=%v", err)
	}
	if _, err := unavailable.ProcessClaim(context.Background(), ActionClaim{}); err == nil ||
		err.Error() != "application.unavailable" {
		t.Fatalf("nil ProcessClaim() error=%v", err)
	}

	ids := &processingClaimIDs{}
	state := &processingClaimState{}
	orchestrator := &Orchestrator{state: state, ids: ids}
	if _, _, err := orchestrator.ClaimNextAction(
		context.Background(), " \t", ActionClaimSelection{},
	); err == nil || err.Error() != "application.worker_ref_required" {
		t.Fatalf("blank worker error=%v", err)
	}
	if ids.calls != 0 || state.claimCalls != 0 {
		t.Fatalf("blank worker reached ids/state: %d/%d", ids.calls, state.claimCalls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := orchestrator.ClaimNextAction(
		ctx, "worker:canceled", ActionClaimSelection{},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled claim error=%v", err)
	}
	if ids.calls != 1 || state.claimCalls != 0 {
		t.Fatalf("ID error reached state: ids=%d state=%d", ids.calls, state.claimCalls)
	}
}

func processingDispatchClaim(t *testing.T) ActionClaim {
	t.Helper()
	goalRef, err := goal.NewGoalRef("goal:processing-dispatch")
	if err != nil {
		t.Fatal(err)
	}
	return ActionClaim{
		Action: ActionRecord{
			Ref:     "action:revoke-execution-session:execution:dispatch",
			Kind:    ActionRevokeSession,
			GoalRef: goalRef,
		},
		Token: "claim:fenced", WorkerRef: "worker:dispatch",
		DeliveryAttempt: 3, Fence: 7,
	}
}

type processingDispatchState struct {
	StateRepository
	claimCalls  int
	quarantines []ActionQuarantinedState
}

func (state *processingDispatchState) ClaimNextAction(
	_ context.Context,
	_ ClaimRequest,
) (ActionClaim, bool, error) {
	state.claimCalls++
	return ActionClaim{}, false, nil
}

func (state *processingDispatchState) QuarantineAction(
	_ context.Context,
	mutation ActionQuarantinedState,
) error {
	state.quarantines = append(state.quarantines, mutation)
	return nil
}
