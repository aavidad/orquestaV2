package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type attemptFaultMode string

const (
	attemptPersistThenError     attemptFaultMode = "persist_then_error"
	attemptPersistThenMalformed attemptFaultMode = "persist_then_malformed"
	attemptAlreadyExists        attemptFaultMode = "already_exists"
	attemptLeaseMissing         attemptFaultMode = "lease_missing"
	attemptLeaseEqualsStarted   attemptFaultMode = "lease_equals_started"
	attemptLeaseMutated         attemptFaultMode = "lease_mutated"
)

type attemptFaultState struct {
	StateRepository
	mode attemptFaultMode
}

func (state attemptFaultState) RecordEffectAttempt(
	ctx context.Context,
	request RecordEffectAttemptState,
) (EffectAttempt, bool, error) {
	attempt, _, err := state.StateRepository.RecordEffectAttempt(ctx, request)
	if err != nil {
		return EffectAttempt{}, false, err
	}
	switch state.mode {
	case attemptPersistThenError:
		return EffectAttempt{}, false, errors.New("test.effect_attempt_persist_ambiguous")
	case attemptPersistThenMalformed:
		attempt.Ref += ":malformed"
		return attempt, true, nil
	case attemptLeaseMissing:
		attempt.ClaimLeaseUntil = time.Time{}
		return attempt, true, nil
	case attemptLeaseEqualsStarted:
		attempt.ClaimLeaseUntil = attempt.StartedAt
		return attempt, true, nil
	case attemptLeaseMutated:
		attempt.ClaimLeaseUntil = attempt.ClaimLeaseUntil.Add(time.Nanosecond)
		return attempt, true, nil
	case attemptAlreadyExists:
		return attempt, false, nil
	default:
		return attempt, true, nil
	}
}

type terminalEffectFaultState struct {
	StateRepository
	stop, prepare, commit, integrate bool
}

func TestActionCallContextIsBoundedByClaimLease(t *testing.T) {
	now := time.Now().UTC()
	orchestrator := &Orchestrator{clock: &mutableClock{now: now}}
	ctx, cancel := orchestrator.actionCallContext(context.Background(), ActionClaim{LeaseUntil: now.Add(50 * time.Millisecond)})
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || deadline.Before(now.Add(40*time.Millisecond)) || deadline.After(now.Add(100*time.Millisecond)) {
		t.Fatalf("action deadline=%s present=%t", deadline, ok)
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("action context outlived claim lease")
	}
	expired, expiredCancel := orchestrator.actionCallContext(context.Background(), ActionClaim{LeaseUntil: now})
	defer expiredCancel()
	select {
	case <-expired.Done():
	default:
		t.Fatal("expired action lease produced live adapter context")
	}
}

type claimBoundStopController struct {
	calls         int
	deadline      time.Time
	enteredAt     time.Time
	hasDeadline   bool
	unboundedWait time.Duration
}

func (*claimBoundStopController) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (controller *claimBoundStopController) Stop(
	ctx context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	controller.calls++
	controller.enteredAt = time.Now()
	controller.deadline, controller.hasDeadline = ctx.Deadline()
	select {
	case <-ctx.Done():
	case <-time.After(controller.unboundedWait):
		return ports.AgentStopReceipt{}, errors.New("test.stop_context_unbounded")
	}
	return ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, StopEffectAttemptRef: request.StopEffectAttemptRef,
		StopActionFence: request.StopActionFence, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: ports.AgentStopPending,
	}, nil
}

type liveQuarantineContextState struct {
	StateRepository
	contextErr error
}

func (state *liveQuarantineContextState) QuarantineAction(
	ctx context.Context,
	request ActionQuarantinedState,
) error {
	state.contextErr = ctx.Err()
	if state.contextErr != nil {
		return state.contextErr
	}
	return state.StateRepository.QuarantineAction(ctx, request)
}

func TestProcessStopUsesClaimBoundContextWithoutReplayingTimedOutEffect(t *testing.T) {
	const lease = 40 * time.Millisecond
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	if _, err := system.orchestrator.Control(
		context.Background(),
		system.access,
		system.request(
			t, "control:claim-bound-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
		),
	); err != nil {
		t.Fatal(err)
	}
	claim, found, err := system.repository.ClaimNextAction(context.Background(), ClaimRequest{
		WorkerRef: "worker:claim-bound-stop", Token: "claim:claim-bound-stop",
		LeaseDuration: lease, Capabilities: testAgentCapabilities(),
	})
	if err != nil || !found || claim.Action.Kind != ActionStopAgent {
		t.Fatalf("claim stop: found=%v claim=%+v err=%v", found, claim, err)
	}
	controller := &claimBoundStopController{unboundedWait: time.Second}
	state := &liveQuarantineContextState{StateRepository: system.repository}
	system.orchestrator.controller = controller
	system.orchestrator.state = state

	started := time.Now()
	err = system.orchestrator.processStop(context.Background(), claim)
	elapsed := time.Since(started)
	if err == nil || err.Error() != effectUnknownAppliedCode {
		t.Fatalf("timed-out stop error=%v want=%s", err, effectUnknownAppliedCode)
	}
	callBudget := controller.deadline.Sub(controller.enteredAt)
	if controller.calls != 1 || !controller.hasDeadline ||
		callBudget <= 0 || callBudget > lease || elapsed >= controller.unboundedWait {
		t.Fatalf(
			"stop context calls=%d deadline=%s entered=%s budget=%s elapsed=%s lease=%s",
			controller.calls, controller.deadline, controller.enteredAt, callBudget, elapsed, lease,
		)
	}
	if state.contextErr != nil {
		t.Fatalf("quarantine inherited canceled effect context: %v", state.contextErr)
	}

	system.clock.Advance(time.Minute)
	result, recoveryErr := system.orchestrator.ProcessNext(context.Background(), "worker:claim-bound-stop-recovery")
	if recoveryErr != nil || result.Processed && result.Action == ActionStopAgent || controller.calls != 1 {
		t.Fatalf(
			"timed-out stop replayed: result=%+v calls=%d err=%v",
			result, controller.calls, recoveryErr,
		)
	}
	assertUnknownAppliedConsumption(t, system.record(t), ActionStopAgent)
}

func (state terminalEffectFaultState) ApplyControl(
	context.Context,
	ApplyControlState,
) (ControlRecord, bool, error) {
	if state.stop {
		return ControlRecord{}, false, errors.New("test.stop_terminal_persist_ambiguous")
	}
	return ControlRecord{}, false, errors.New("test.unexpected_control_write")
}

func (state terminalEffectFaultState) RecordWorkspacePrepared(context.Context, WorkspacePreparedState) error {
	if state.prepare {
		return errors.New("test.prepare_terminal_persist_ambiguous")
	}
	return errors.New("test.unexpected_prepare_write")
}

func (state terminalEffectFaultState) RecordChangeCommitted(context.Context, ChangeCommittedState) error {
	if state.commit {
		return errors.New("test.commit_terminal_persist_ambiguous")
	}
	return errors.New("test.unexpected_commit_write")
}

func (state terminalEffectFaultState) RecordIntegrationResult(context.Context, IntegrationResultState) error {
	if state.integrate {
		return errors.New("test.integrate_terminal_persist_ambiguous")
	}
	return errors.New("test.unexpected_integrate_write")
}

type physicalEffectFaultFixture struct {
	orchestrator *Orchestrator
	repository   *memoryRepository
	kind         ActionKind
	physical     func() int
}

func newPhysicalEffectFaultFixture(t *testing.T, kind ActionKind) physicalEffectFaultFixture {
	t.Helper()
	if kind == ActionLaunchAgent {
		clock := &mutableClock{now: time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC)}
		repository := newMemoryRepository()
		agent := &scriptedAgent{now: clock.Now}
		orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
		actor, project := testScope(t)
		if _, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
			RequestRef: "request:physical-effect-fault", Statement: "launch once", Confirm: true,
		}); err != nil {
			t.Fatal(err)
		}
		return physicalEffectFaultFixture{orchestrator, repository, kind, func() int {
			agent.mu.Lock()
			defer agent.mu.Unlock()
			return agent.launches
		}}
	}
	if kind == ActionStopAgent {
		system := newControlTestSystem(t, nil)
		system.launch(t)
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		if _, err := system.orchestrator.Control(context.Background(), system.access, system.request(
			t, "control:local-effect-fault", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
		)); err != nil {
			t.Fatal(err)
		}
		return physicalEffectFaultFixture{system.orchestrator, system.repository, kind, func() int {
			system.agent.mu.Lock()
			defer system.agent.mu.Unlock()
			return system.agent.stopCalls
		}}
	}
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	switch kind {
	case ActionPrepareWorkspace:
		manager := system.orchestrator.workspaceManager.(*scriptedWorkspaceManager)
		return physicalEffectFaultFixture{system.orchestrator, system.repository, kind, func() int {
			manager.mu.Lock()
			defer manager.mu.Unlock()
			return len(manager.requests)
		}}
	case ActionCommitChange:
		system.process(t, ActionPrepareWorkspace, ActionLaunchAgent, ActionObserveAgent)
		return physicalEffectFaultFixture{system.orchestrator, system.repository, kind, func() int {
			system.control.mu.Lock()
			defer system.control.mu.Unlock()
			return len(system.control.commitRequests)
		}}
	case ActionAttestTest:
		system.processCommit(t)
		return physicalEffectFaultFixture{system.orchestrator, system.repository, kind, func() int {
			system.attestor.mu.Lock()
			defer system.attestor.mu.Unlock()
			return len(system.attestor.runs)
		}}
	case ActionIntegrateChange:
		system.processCommit(t)
		system.process(t, ActionAttestTest)
		system.approveReviews(t)
		record := system.record(t)
		if _, err := system.orchestrator.IntegrateChange(context.Background(), system.access, IntegrateChangeRequest{
			RequestRef: "request:local-effect-fault", GoalRef: record.Goal.Ref(),
			ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
		}); err != nil {
			t.Fatal(err)
		}
		return physicalEffectFaultFixture{system.orchestrator, system.repository, kind, func() int {
			system.control.mu.Lock()
			defer system.control.mu.Unlock()
			return len(system.control.integrationRequests)
		}}
	default:
		t.Fatalf("unsupported physical effect fixture %s", kind)
		return physicalEffectFaultFixture{}
	}
}

func TestEveryPhysicalEffectRequiresNewDurableAttempt(t *testing.T) {
	for _, kind := range []ActionKind{
		ActionLaunchAgent, ActionStopAgent, ActionPrepareWorkspace, ActionCommitChange,
		ActionAttestTest, ActionIntegrateChange,
	} {
		for _, mode := range []attemptFaultMode{
			attemptPersistThenError, attemptPersistThenMalformed, attemptAlreadyExists,
			attemptLeaseMissing, attemptLeaseEqualsStarted, attemptLeaseMutated,
		} {
			t.Run(string(kind)+"/"+string(mode), func(t *testing.T) {
				fixture := newPhysicalEffectFaultFixture(t, kind)
				before, settlementsBefore := fixture.physical(), budgetSettlementCount(fixture.repository)
				fixture.orchestrator.state = attemptFaultState{StateRepository: fixture.repository, mode: mode}
				result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:local-attempt-fault")
				if err == nil || err.Error() != effectUnknownAppliedCode || result.Action != kind ||
					fixture.physical() != before || budgetSettlementCount(fixture.repository) != settlementsBefore {
					t.Fatalf("result=%+v calls=%d/%d settlements=%d/%d err=%v", result,
						before, fixture.physical(), settlementsBefore, budgetSettlementCount(fixture.repository), err)
				}
				assertLocalEffectUnknown(t, fixture.repository, kind)
			})
		}
	}
}

func TestEffectAttemptPreservesExactClaimLeaseAcrossReplay(t *testing.T) {
	fixture := newEffectDecisionFixture(t, governance.SecurityCriticalitySensitive, time.Minute)
	decideFixtureEffect(t, fixture, fixture.access, "request:attempt-lease", EffectApproved)
	claim, found, err := fixture.orchestrator.ClaimNextAction(
		context.Background(), "worker:attempt-lease", ActionClaimSelection{},
	)
	if err != nil || !found {
		t.Fatalf("claim found=%t err=%v", found, err)
	}

	first, created, err := fixture.orchestrator.beginEffectAttempt(context.Background(), claim, fixture.clock.Now())
	if err != nil || !created || first.ClaimLeaseUntil != claim.LeaseUntil.UTC() {
		t.Fatalf("first created=%t lease=%s want=%s err=%v", created, first.ClaimLeaseUntil, claim.LeaseUntil.UTC(), err)
	}
	second, created, err := fixture.orchestrator.beginEffectAttempt(context.Background(), claim, fixture.clock.Now())
	if err != nil || created || second != first {
		t.Fatalf("replay created=%t attempt=%+v want=%+v err=%v", created, second, first, err)
	}
}

func TestEveryPhysicalEffectReceiptRejectsExclusiveAttemptLeaseBoundary(t *testing.T) {
	boundary := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		kind   EffectKind
		status EffectStatus
	}{
		{EffectKindAgentLaunch, EffectStatusAccepted},
		{EffectKindAgentStop, EffectStatusStopped},
		{EffectKindPrepareWorkspace, EffectStatusPrepared},
		{EffectKindCommitChange, EffectStatusCommitted},
		{EffectKindAttestTest, EffectStatusAttestedPassed},
		{EffectKindIntegrateChange, EffectStatusIntegrated},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			intent := EffectIntent{
				Ref: "effect-intent:lease-boundary:" + string(test.kind), Digest: "digest:lease-boundary",
				ActionRef: "action:lease-boundary:" + string(test.kind), Kind: test.kind,
				IdempotencyKey: "idempotency:lease-boundary:" + string(test.kind),
			}
			approval := EffectApproval{Ref: "effect-approval:lease-boundary:" + string(test.kind)}
			claim := ActionClaim{
				Action:         ActionRecord{Ref: intent.ActionRef, EffectIntentRef: intent.Ref, EffectIntent: intent},
				EffectApproval: approval, Fence: 1, LeaseUntil: boundary,
			}
			attempt := EffectAttempt{
				Ref: "effect-attempt:lease-boundary:" + string(test.kind), IntentRef: intent.Ref,
				IntentDigest: intent.Digest, ApprovalRef: approval.Ref, ActionRef: intent.ActionRef,
				ActionFence: 1, IdempotencyKey: intent.IdempotencyKey,
				StartedAt: boundary.Add(-time.Second), ClaimLeaseUntil: boundary,
			}
			if _, err := effectReceipt(
				claim, attempt, "provider-receipt:lease-boundary", test.status, unknownUsage(), boundary,
			); err == nil || err.Error() != "application.effect_receipt_invalid" {
				t.Fatalf("receipt accepted exclusive lease boundary: %v", err)
			}
			finalLiveReceipt, err := effectReceipt(
				claim, attempt, "provider-receipt:lease-boundary", test.status, unknownUsage(), boundary.Add(-time.Nanosecond),
			)
			if err != nil {
				t.Fatalf("receipt rejected final live instant: %v", err)
			}
			if finalLiveReceipt.ActionFence != claim.Fence {
				t.Fatalf("normal path receipt fence=%d want claim fence=%d",
					finalLiveReceipt.ActionFence, claim.Fence)
			}
		})
	}
}

func TestEffectReceiptKeepsHistoricalFenceUnderAuthorizedRecoveryClaim(t *testing.T) {
	startedAt := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	intent := EffectIntent{
		Ref: "effect-intent:recovered-confirmation", Digest: "digest:recovered-confirmation",
		ActionRef: "action:recovered-confirmation", Kind: EffectKindAgentLaunch,
		IdempotencyKey: "idempotency:recovered-confirmation",
	}
	approval := EffectApproval{Ref: "effect-approval:recovered-confirmation"}
	claim := ActionClaim{
		Action: ActionRecord{
			Ref: intent.ActionRef, Kind: ActionLaunchAgent, EffectIntentRef: intent.Ref, EffectIntent: intent,
		},
		EffectApproval: approval, Fence: 7, LeaseUntil: startedAt.Add(2 * time.Minute),
	}
	attempt := EffectAttempt{
		Ref: "effect-attempt:recovered-confirmation", IntentRef: intent.Ref,
		IntentDigest: intent.Digest, ApprovalRef: approval.Ref, ActionRef: intent.ActionRef,
		ActionFence: 7, IdempotencyKey: intent.IdempotencyKey,
		StartedAt: startedAt, ClaimLeaseUntil: startedAt.Add(time.Minute),
	}

	normalReceipt, err := effectReceipt(
		claim, attempt, "provider-receipt:recovered-confirmation", EffectStatusAccepted,
		unknownUsage(), attempt.ClaimLeaseUntil.Add(-time.Nanosecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	if normalReceipt.ActionFence != claim.Fence {
		t.Fatalf("normal receipt fence=%d want claim fence=%d", normalReceipt.ActionFence, claim.Fence)
	}

	recoveryClaim := claim
	recoveryClaim.Fence = attempt.ActionFence + 1
	recoveryClaim.Disposition = ActionClaimDispositionRecoverEffect
	recoveryClaim.RecoveryEffectAttemptRef = attempt.Ref
	receipt, err := effectReceipt(
		recoveryClaim, attempt, "provider-receipt:recovered-confirmation", EffectStatusAccepted,
		unknownUsage(), attempt.ClaimLeaseUntil.Add(-time.Nanosecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ActionFence != attempt.ActionFence || recoveryClaim.Fence <= receipt.ActionFence {
		t.Fatalf("receipt historical fence=%d want=%d recovery claim fence=%d",
			receipt.ActionFence, attempt.ActionFence, recoveryClaim.Fence)
	}

	for name, mutate := range map[string]func(*EffectReceipt){
		"physical fence crossed":    func(candidate *EffectReceipt) { candidate.ActionFence++ },
		"historical lease boundary": func(candidate *EffectReceipt) { candidate.ConfirmedAt = attempt.ClaimLeaseUntil },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := receipt
			mutate(&candidate)
			if err := validateEffectReceipt(recoveryClaim, attempt, candidate); err == nil ||
				err.Error() != "application.effect_receipt_invalid" {
				t.Fatalf("invalid receipt accepted: %+v err=%v", candidate, err)
			}
		})
	}

	for name, mutate := range map[string]func(*ActionClaim){
		"normal divergent": func(candidate *ActionClaim) {
			candidate.Disposition = ""
			candidate.RecoveryEffectAttemptRef = ""
		},
		"recovery without ref":        func(candidate *ActionClaim) { candidate.RecoveryEffectAttemptRef = "" },
		"recovery other action":       func(candidate *ActionClaim) { candidate.Action.Kind = ActionStopAgent },
		"recovery fence not superior": func(candidate *ActionClaim) { candidate.Fence = attempt.ActionFence },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := recoveryClaim
			mutate(&candidate)
			if _, err := effectReceipt(
				candidate, attempt, "provider-receipt:invalid-confirmation", EffectStatusAccepted,
				unknownUsage(), attempt.StartedAt,
			); err == nil || err.Error() != "application.effect_receipt_invalid" {
				t.Fatalf("invalid confirmation claim accepted: %+v err=%v", candidate, err)
			}
		})
	}
}

func budgetSettlementCount(repository *memoryRepository) int {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	total := 0
	for _, record := range repository.records {
		total += len(record.BudgetSettlements)
	}
	return total
}

func TestLocalEffectTerminalPersistenceAmbiguityQuarantines(t *testing.T) {
	for _, kind := range []ActionKind{
		ActionStopAgent, ActionPrepareWorkspace, ActionCommitChange, ActionIntegrateChange,
	} {
		t.Run(string(kind), func(t *testing.T) {
			fixture := newPhysicalEffectFaultFixture(t, kind)
			before := fixture.physical()
			fault := terminalEffectFaultState{StateRepository: fixture.repository}
			switch kind {
			case ActionStopAgent:
				fault.stop = true
			case ActionPrepareWorkspace:
				fault.prepare = true
			case ActionCommitChange:
				fault.commit = true
			case ActionIntegrateChange:
				fault.integrate = true
			}
			fixture.orchestrator.state = fault
			result, err := fixture.orchestrator.ProcessNext(context.Background(), "worker:local-terminal-fault")
			if err == nil || err.Error() != effectUnknownAppliedCode || result.Action != kind ||
				fixture.physical() != before+1 {
				t.Fatalf("result=%+v calls=%d/%d err=%v", result, before, fixture.physical(), err)
			}
			assertLocalEffectUnknown(t, fixture.repository, kind)
		})
	}
}

func assertLocalEffectUnknown(t *testing.T, repository *memoryRepository, kind ActionKind) {
	t.Helper()
	repository.mu.Lock()
	defer repository.mu.Unlock()
	for _, record := range repository.records {
		assertUnknownAppliedConsumption(t, record, kind)
		return
	}
	t.Fatal("local effect Goal missing")
}

func assertUnknownAppliedConsumption(t *testing.T, record GoalRecord, kind ActionKind) {
	t.Helper()
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.Kind == kind && receipt.Outcome == ActionConsumedQuarantined &&
			receipt.ErrorCode == effectUnknownAppliedCode {
			return
		}
	}
	t.Fatalf("unknown-applied consumption missing kind=%s receipts=%+v", kind, record.ConsumptionReceipts)
}
