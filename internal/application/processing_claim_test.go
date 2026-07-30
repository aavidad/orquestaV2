// Este fichero acredita que el reclamo global y el despacho interno conservan
// el contrato público de ProcessNext al separarse en dos pasos privados.
package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestProcessNextPreservesClaimRequestAndEmptyOutcomes(t *testing.T) {
	now := time.Date(2026, 7, 30, 22, 0, 0, 0, time.UTC)
	expectedError := errors.New("state.claim_failed")
	cases := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "not_found"},
		{name: "error", err: expectedError, wantErr: expectedError},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			state := &processingClaimState{claimErr: test.err}
			capabilities := ports.AgentCapabilities{
				ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
				RoleKeys: []string{"role:worker"}, SkillRefs: []string{"skill:test"},
			}
			policy := testBudgetPolicy(now)
			orchestrator := &Orchestrator{
				state: state, ids: &sequentialIDs{}, claimLease: 3 * time.Minute,
				attestTestClaimLease: 7 * time.Minute,
				agentCapabilities:    capabilities, budgetPolicy: policy,
			}

			result, err := orchestrator.ProcessNext(context.Background(), "worker:exact")

			if result != (ProcessResult{}) || !errors.Is(err, test.wantErr) {
				t.Fatalf("outcome result=%+v err=%v", result, err)
			}
			wantRequest := ClaimRequest{
				WorkerRef: "worker:exact", Token: "claim:001", LeaseDuration: 3 * time.Minute,
				AttestTestLeaseDuration: 7 * time.Minute,
				Capabilities:            capabilities, BudgetPolicy: policy,
			}
			if state.claimCalls != 1 || !reflect.DeepEqual(state.request, wantRequest) {
				t.Fatalf("claim calls=%d request=%+v want=%+v", state.claimCalls, state.request, wantRequest)
			}
		})
	}
}

func TestProcessClaimPreservesDispositionAndDispatch(t *testing.T) {
	goalRef, err := goal.NewGoalRef("goal:processing-claim")
	if err != nil {
		t.Fatal(err)
	}
	knownError := errors.New("state.known_action")
	cases := []struct {
		name              string
		kind              ActionKind
		disposition       ActionClaimDisposition
		getGoalErr        error
		wantErr           error
		wantStateConflict bool
		wantGetGoal       int
		wantQuarantine    int
	}{
		{
			name: "irreversible", kind: ActionLaunchAgent,
			disposition: ActionClaimDispositionRetryBudgetIrreversible,
			getGoalErr:  knownError, wantErr: knownError, wantGetGoal: 1,
		},
		{
			name: "invalid_disposition", kind: ActionLaunchAgent,
			disposition: "invalid", wantStateConflict: true,
		},
		{
			name: "known_action", kind: ActionRevokeSession,
			getGoalErr: knownError, wantErr: knownError, wantGetGoal: 1,
		},
		{
			name: "unknown_action", kind: ActionKind("unknown"),
			wantQuarantine: 1,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			state := &processingClaimState{getGoalErr: test.getGoalErr}
			orchestrator := &Orchestrator{
				state: state,
				clock: &mutableClock{now: time.Date(2026, 7, 30, 22, 0, 0, 0, time.UTC)},
			}
			claim := ActionClaim{
				Action:      ActionRecord{Ref: "action:test", GoalRef: goalRef, Kind: test.kind},
				Disposition: test.disposition, DeliveryAttempt: 1,
			}

			result, processErr := orchestrator.processClaim(context.Background(), claim)

			if !result.Processed || result.GoalRef != goalRef || result.Action != test.kind {
				t.Fatalf("result=%+v", result)
			}
			if test.wantStateConflict {
				if !IsStateError(processErr, StateConflict) {
					t.Fatalf("error=%v want state conflict", processErr)
				}
			} else if test.wantErr != nil {
				if !errors.Is(processErr, test.wantErr) {
					t.Fatalf("error=%v want=%v", processErr, test.wantErr)
				}
			} else if processErr == nil || processErr.Error() != "application.action_kind_invalid:unknown" {
				t.Fatalf("unknown action error=%v", processErr)
			}
			if state.getGoalCalls != test.wantGetGoal || state.quarantineCalls != test.wantQuarantine {
				t.Fatalf("calls get_goal=%d quarantine=%d", state.getGoalCalls, state.quarantineCalls)
			}
		})
	}
}

func TestProcessNextDispatchesOneFoundClaimWithSameContext(t *testing.T) {
	goalRef, err := goal.NewGoalRef("goal:processing-found")
	if err != nil {
		t.Fatal(err)
	}
	witness := errors.New("state.get_goal_witness")
	claim := ActionClaim{
		Action: ActionRecord{
			Ref:  "action:revoke-execution-session:execution:test",
			Kind: ActionRevokeSession, GoalRef: goalRef,
		},
		Token: "claim:durable", WorkerRef: "worker:durable", DeliveryAttempt: 3, Fence: 2,
	}
	state := &processingClaimState{claim: claim, found: true, getGoalErr: witness}
	ids := &processingClaimIDs{}
	orchestrator := &Orchestrator{state: state, ids: ids}
	ctx := context.WithValue(context.Background(), processingContextKey{}, "same")

	result, processErr := orchestrator.ProcessNext(ctx, "worker:dispatch")

	wantResult := ProcessResult{Processed: true, GoalRef: goalRef, Action: ActionRevokeSession}
	if result != wantResult || !errors.Is(processErr, witness) || !reflect.DeepEqual(state.claim, claim) {
		t.Fatalf("result=%+v error=%v claim=%+v", result, processErr, state.claim)
	}
	if ids.calls != 1 || state.claimCalls != 1 || state.getGoalCalls != 1 ||
		state.claimContext != ctx || state.getGoalContext != ctx || state.getGoalRef != goalRef {
		t.Fatalf("calls ids=%d claim=%d dispatch=%d same_context=%v/%v goal=%s",
			ids.calls, state.claimCalls, state.getGoalCalls,
			state.claimContext == ctx, state.getGoalContext == ctx, state.getGoalRef)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if replay, replayErr := orchestrator.ProcessNext(canceled, "worker:dispatch"); replay != (ProcessResult{}) ||
		!errors.Is(replayErr, context.Canceled) || ids.calls != 2 ||
		state.claimCalls != 1 || state.getGoalCalls != 1 {
		t.Fatalf("canceled replay=%+v err=%v ids=%d claim=%d dispatch=%d",
			replay, replayErr, ids.calls, state.claimCalls, state.getGoalCalls)
	}
}

type processingContextKey struct{}

type processingClaimIDs struct {
	sequentialIDs
	calls int
}

func (ids *processingClaimIDs) NewID(ctx context.Context, prefix string) (string, error) {
	ids.calls++
	return ids.sequentialIDs.NewID(ctx, prefix)
}

type processingClaimState struct {
	StateRepository
	request         ClaimRequest
	claim           ActionClaim
	found           bool
	claimErr        error
	getGoalErr      error
	claimCalls      int
	getGoalCalls    int
	quarantineCalls int
	claimContext    context.Context
	getGoalContext  context.Context
	getGoalRef      goal.GoalRef
}

func (state *processingClaimState) ClaimNextAction(
	ctx context.Context,
	request ClaimRequest,
) (ActionClaim, bool, error) {
	state.claimCalls++
	state.claimContext = ctx
	state.request = request
	return state.claim, state.found, state.claimErr
}

func (state *processingClaimState) GetGoal(
	ctx context.Context,
	goalRef goal.GoalRef,
) (GoalRecord, error) {
	state.getGoalCalls++
	state.getGoalContext = ctx
	state.getGoalRef = goalRef
	return GoalRecord{}, state.getGoalErr
}

func (state *processingClaimState) QuarantineAction(
	_ context.Context,
	_ ActionQuarantinedState,
) error {
	state.quarantineCalls++
	return nil
}
