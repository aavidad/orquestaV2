package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestTerminalLaunchPostReceiptFailureReplaysOnlyExactLocalPersistence(t *testing.T) {
	for name, firstError := range map[string]error{
		"local":     errors.New("test.local_terminal_persistence_failure"),
		"temporary": temporaryAgentTestError{},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newTerminalPostReceiptTestFixture(t, name, firstError, 1)

			result, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
			if err != nil || !result.Processed || result.Action != ActionLaunchAgent {
				t.Fatalf("ProcessClaim result=%+v err=%v", result, err)
			}
			launchCalls := fixture.launchCalls()
			if launchCalls != 0 || fixture.launcher.reconcileCalls != 0 ||
				fixture.launcher.continueCalls != 1 || fixture.launcher.effectCalls != 1 {
				t.Fatalf("Launch=%d Reconcile=%d Continue=%d effects=%d want 0/0/1/1",
					launchCalls, fixture.launcher.reconcileCalls,
					fixture.launcher.continueCalls, fixture.launcher.effectCalls)
			}
			state := fixture.state
			if !state.persisted || len(state.attempts) != 1 || len(state.completions) != 2 ||
				len(state.requeues) != 0 || len(state.quarantines) != 0 ||
				!reflect.DeepEqual(state.completions[0], state.completions[1]) {
				t.Fatalf("persisted=%t attempts=%d completions=%d requeues=%d quarantines=%d exact=%t",
					state.persisted, len(state.attempts), len(state.completions), len(state.requeues), len(state.quarantines),
					len(state.completions) == 2 && reflect.DeepEqual(state.completions[0], state.completions[1]))
			}
		})
	}
}

func TestTerminalLaunchPostReceiptPersistentLocalFailureRequeuesIdempotentContinuation(t *testing.T) {
	fixture := newTerminalPostReceiptTestFixture(
		t, "persistent", errors.New("test.persistent_terminal_persistence_failure"), 2,
	)

	first, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
	if err != nil || !first.Processed || fixture.state.persisted || len(fixture.state.completions) != 2 ||
		len(fixture.state.requeues) != 1 || len(fixture.state.quarantines) != 0 ||
		fixture.launcher.continueCalls != 1 || fixture.launcher.effectCalls != 1 {
		t.Fatalf("first=%+v err=%v persisted=%t completions=%d requeues=%d quarantines=%d Continue=%d effects=%d",
			first, err, fixture.state.persisted, len(fixture.state.completions), len(fixture.state.requeues),
			len(fixture.state.quarantines), fixture.launcher.continueCalls, fixture.launcher.effectCalls)
	}

	fixture.claim.Token = "claim:terminal-post-receipt-second-pass"
	fixture.claim.WorkerRef = "worker:terminal-post-receipt-second-pass"
	fixture.claim.DeliveryAttempt++
	fixture.claim.Fence++
	fixture.claim.LeaseUntil = fixture.claim.LeaseUntil.Add(time.Minute)
	fixture.persistRecoveryClaim()
	second, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
	if err != nil || !second.Processed || !fixture.state.persisted || len(fixture.state.attempts) != 2 ||
		len(fixture.state.completions) != 3 || len(fixture.state.requeues) != 1 ||
		len(fixture.state.quarantines) != 0 {
		t.Fatalf("second=%+v err=%v persisted=%t attempts=%d completions=%d requeues=%d quarantines=%d",
			second, err, fixture.state.persisted, len(fixture.state.attempts), len(fixture.state.completions),
			len(fixture.state.requeues), len(fixture.state.quarantines))
	}
	if fixture.launchCalls() != 0 || fixture.launcher.reconcileCalls != 0 ||
		fixture.launcher.continueCalls != 2 || fixture.launcher.effectCalls != 1 {
		t.Fatalf("Launch=%d Reconcile=%d Continue=%d effects=%d want 0/0/2/1",
			fixture.launchCalls(), fixture.launcher.reconcileCalls,
			fixture.launcher.continueCalls, fixture.launcher.effectCalls)
	}
	firstCompletion, replayedCompletion := fixture.state.completions[0], fixture.state.completions[2]
	if firstCompletion.EffectReceipt != replayedCompletion.EffectReceipt ||
		firstCompletion.Execution != replayedCompletion.Execution ||
		!reflect.DeepEqual(firstCompletion.NextAction, replayedCompletion.NextAction) ||
		firstCompletion.Event != replayedCompletion.Event {
		t.Fatalf("terminal replay changed receipt or result: first=%+v replay=%+v",
			firstCompletion, replayedCompletion)
	}
}

type terminalPostReceiptTestFixture struct {
	agentLaunchRecoveryFixture
	state    *terminalPostReceiptState
	launcher *terminalPostReceiptLauncher
}

func newTerminalPostReceiptTestFixture(
	t *testing.T,
	name string,
	firstError error,
	completionFailures int,
) terminalPostReceiptTestFixture {
	t.Helper()
	fixture := newAgentLaunchRecoveryFixture(t)
	fixture.claim.Disposition = ActionClaimDispositionReconcileTerminalLaunch
	fixture.claim.TerminalReconciliationRef = "agent-launch-reconciliation-authority:post-receipt-" + name
	fixture.claim.TerminalReconciliationFingerprint = strings.Repeat("a", 64)
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	fixture.persistRecoveryClaim()
	reconciliationAttemptRef := "agent-launch-reconciliation-attempt:" +
		fixture.claim.TerminalReconciliationRef + ":" + fixture.claim.Token
	continuation := ExpiredAgentLaunchContinuationRecordV41{
		SubjectRef:                 "expired-launch-continuation-subject:post-receipt-" + name,
		ReconciliationAuthorityRef: fixture.claim.TerminalReconciliationRef,
		ReconciliationAttemptRef:   reconciliationAttemptRef,
		ProjectRef:                 fixture.record.Goal.Project().String(),
		GoalRef:                    fixture.claim.Action.GoalRef.String(),
		WorkItemRef:                fixture.claim.Action.WorkItemRef.String(),
		ExecutionRef:               fixture.claim.Action.ExecutionRef.String(),
		ActionRef:                  fixture.claim.Action.Ref,
		EffectIntentRef:            fixture.claim.Action.EffectIntentRef,
		EffectIntentDigest:         fixture.claim.Action.EffectIntent.Digest,
		EffectAttemptRef:           fixture.attempt.Ref,
		PlanGeneration:             uint64(fixture.claim.Action.PlanGeneration),
		WorkItemGeneration:         uint64(fixture.claim.Action.WorkItemGeneration),
		ActionFence:                fixture.attempt.ActionFence,
	}
	state := &terminalPostReceiptState{
		StateRepository: fixture.repository,
		continuation:    continuation,
		firstError:      firstError,
		failures:        completionFailures,
	}
	launcher := &terminalPostReceiptLauncher{
		scriptedAgent: &scriptedAgent{now: fixture.clock.Now},
		acceptedAt:    fixture.attempt.StartedAt.Add(time.Second),
	}
	fixture.orchestrator.state = state
	fixture.orchestrator.launcher = launcher
	return terminalPostReceiptTestFixture{
		agentLaunchRecoveryFixture: fixture, state: state, launcher: launcher,
	}
}

func (fixture terminalPostReceiptTestFixture) launchCalls() int {
	fixture.launcher.scriptedAgent.mu.Lock()
	defer fixture.launcher.scriptedAgent.mu.Unlock()
	return len(fixture.launcher.scriptedAgent.launchRequests)
}

type terminalPostReceiptState struct {
	StateRepository
	continuation ExpiredAgentLaunchContinuationRecordV41
	firstError   error
	failures     int
	attempts     []RecordTerminalAgentLaunchReconciliationAttemptState
	completions  []TerminalAgentLaunchReconciliationCompletedState
	requeues     []TerminalAgentLaunchReconciliationRequeuedState
	quarantines  []TerminalAgentLaunchReconciliationQuarantinedState
	persisted    bool
}

func (state *terminalPostReceiptState) ValidateTerminalAgentLaunchReconciliationClaim(
	context.Context,
	ActionClaim,
) error {
	return nil
}

func (state *terminalPostReceiptState) RecordTerminalAgentLaunchReconciliationAttempt(
	_ context.Context,
	attempt RecordTerminalAgentLaunchReconciliationAttemptState,
) error {
	state.attempts = append(state.attempts, attempt)
	return nil
}

func (state *terminalPostReceiptState) RecordTerminalAgentLaunchReconciled(
	_ context.Context,
	completion TerminalAgentLaunchReconciliationCompletedState,
) error {
	state.completions = append(state.completions, completion)
	if len(state.completions) <= state.failures {
		return state.firstError
	}
	state.persisted = true
	return nil
}

func (state *terminalPostReceiptState) RequeueTerminalAgentLaunchReconciliation(
	_ context.Context,
	requeue TerminalAgentLaunchReconciliationRequeuedState,
) error {
	state.requeues = append(state.requeues, requeue)
	return nil
}

func (state *terminalPostReceiptState) QuarantineTerminalAgentLaunchReconciliation(
	_ context.Context,
	quarantine TerminalAgentLaunchReconciliationQuarantinedState,
) error {
	state.quarantines = append(state.quarantines, quarantine)
	return nil
}

func (state *terminalPostReceiptState) RecordExpiredAgentLaunchContinuationV41(
	_ context.Context,
	record ExpiredAgentLaunchContinuationRecordV41,
) (ExpiredAgentLaunchContinuationRecordV41, bool, error) {
	return record, false, nil
}

func (state *terminalPostReceiptState) ExpiredAgentLaunchContinuationV41(
	_ context.Context,
	authorityRef string,
) (ExpiredAgentLaunchContinuationRecordV41, bool, error) {
	if authorityRef != state.continuation.ReconciliationAuthorityRef {
		return ExpiredAgentLaunchContinuationRecordV41{}, false, nil
	}
	return state.continuation, true, nil
}

type terminalPostReceiptLauncher struct {
	*scriptedAgent
	acceptedAt                                 time.Time
	continueCalls, reconcileCalls, effectCalls int
	receipts                                   map[string]ports.AgentLaunchReceipt
}

func (launcher *terminalPostReceiptLauncher) ContinueExpiredAgentLaunchV41(
	_ context.Context,
	request ports.AgentLaunchRequest,
	_ ExpiredAgentLaunchContinuationRecordV41,
) (ports.AgentLaunchReceipt, error) {
	launcher.continueCalls++
	if previous, found := launcher.receipts[request.IdempotencyKey]; found {
		return previous, nil
	}
	receipt := launchReceiptForRequest(request, launcher.acceptedAt)
	receipt.RequierePreservacionEntorno = request.RequierePreservacionEntorno
	if launcher.receipts == nil {
		launcher.receipts = make(map[string]ports.AgentLaunchReceipt)
	}
	launcher.receipts[request.IdempotencyKey] = receipt
	launcher.effectCalls++
	return receipt, nil
}

func (launcher *terminalPostReceiptLauncher) ReconcileLaunch(
	_ context.Context,
	_ ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	launcher.reconcileCalls++
	return ports.AgentLaunchReceipt{}, errors.New("test.reconcile_must_not_run")
}
