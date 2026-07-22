package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/ports"
)

type attemptFaultMode string

const (
	attemptPersistThenError     attemptFaultMode = "persist_then_error"
	attemptPersistThenMalformed attemptFaultMode = "persist_then_malformed"
	attemptAlreadyExists        attemptFaultMode = "already_exists"
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
