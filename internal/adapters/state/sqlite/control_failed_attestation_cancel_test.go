package sqlite

import (
	"context"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteCancelFailedAttestationKeepsGoalReadableAndRevokesSessionAfterRestart(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteV15System(t, 4)
	attestor := &sqliteSplitVerdictAttestor{}
	broker := &sqliteExecutionSessionBroker{at: system.clock.Now()}
	system.orchestrator = newSQLiteV16OrchestratorWithStateAttestorAndSessions(
		t, system, system.repository, attestor, broker,
	)
	submitted, err := system.orchestrator.Submit(ctx, system.access, application.SubmitRequest{
		RequestRef: "request:sqlite-cancel-failed-attestation",
		Statement:  "cancel while preserving failed attestation evidence",
		Confirm:    true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:sqlite-cancel-failed-attestation",
				Key: "phase:sqlite-cancel-failed-attestation", TemplateRef: "phase-template:sqlite-cancel-failed-attestation",
			}},
			WorkItems: []application.WorkItemSpec{
				{
					Key: "writer-a", Objective: "produce an exact tested candidate",
					Phase: "phase:sqlite-cancel-failed-attestation", Role: "role:writer",
					WriteSet:       []string{"internal/cancel-failed-attestation-a"},
					CouncilPolicy:  council.PolicyRequired,
					RequiredTests:  sqliteRequiredTestSpecs("required-test:sqlite-cancel-failed-attestation-a"),
					OutputContract: goal.OutputContractEvidenceBundle,
				},
				{
					Key: "writer-b", Objective: "produce an exact tested candidate",
					Phase: "phase:sqlite-cancel-failed-attestation", Role: "role:writer",
					WriteSet:       []string{"internal/cancel-failed-attestation-b"},
					CouncilPolicy:  council.PolicyRequired,
					RequiredTests:  sqliteRequiredTestSpecs("required-test:sqlite-cancel-failed-attestation-b"),
					OutputContract: goal.OutputContractEvidenceBundle,
				},
			},
		},
	})
	sqliteTestNoError(t, err)
	for step := 0; step < 100; step++ {
		result, processErr := system.orchestrator.ProcessNext(ctx, "worker:sqlite-seed-split-attestation")
		if processErr != nil {
			t.Fatalf("seed split attestation step=%d result=%+v err=%v", step, result, processErr)
		}
		if !result.Processed {
			break
		}
		if step == 99 {
			t.Fatal("split attestation fixture did not quiesce")
		}
	}

	before, err := system.repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	var failedItem, liveItem goal.WorkItem
	var failedExecution, liveExecution application.ExecutionRecord
	for _, item := range before.Goal.WorkItems() {
		executionRef, bound := item.Execution()
		execution, found := sqliteExecutionByRef(before.Executions, executionRef)
		interruptCause, interrupted := item.InterruptCause()
		switch {
		case bound && found && item.State() == goal.WorkItemStateInterrupted && interrupted &&
			interruptCause == goal.WorkItemInterruptExecutionFailed &&
			execution.State == application.ExecutionFailed &&
			execution.FailureCode == "test_attestor.required_tests_failed":
			failedItem, failedExecution = item, execution
		case bound && found && item.State() == goal.WorkItemStateRunning &&
			execution.State == application.ExecutionAwaitingIntegration:
			liveItem, liveExecution = item, execution
		}
	}
	if failedItem.Ref().String() == "" || liveItem.Ref().String() == "" ||
		before.Goal.State() != goal.GoalStateRunning {
		t.Fatalf("split attestation goal=%s items=%+v executions=%+v",
			before.Goal.State(), before.Goal.WorkItems(), before.Executions)
	}
	baselineRevokes, baselineReplays := broker.revokes, broker.replays

	canceled, err := system.orchestrator.Control(ctx, system.access, application.ControlRequest{
		RequestRef: "request:sqlite-cancel-after-failed-attestation",
		Operation:  application.ControlCancel, Target: application.ControlTargetGoal,
		GoalRef: before.Goal.Ref(), ExpectedGoalRevision: before.Goal.Revision(),
		ExpectedPlanGeneration:    before.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: before.Goal.AppSpec().Generation(), ExpectedSpecHash: before.Goal.SpecHash(),
		Reason: "preserve exact failed candidate while closing goal",
	})
	if err != nil || !canceled.Created || canceled.Control.Status != application.ControlConfirmed {
		t.Fatalf("cancel failed attestation result=%+v err=%s", canceled, sqliteTestErrorChain(err))
	}
	after, err := system.repository.GetGoal(ctx, before.Goal.Ref())
	if err != nil {
		t.Fatalf("read canceled failed attestation: %s", sqliteTestErrorChain(err))
	}
	failedItem, _ = after.Goal.WorkItem(failedItem.Ref())
	liveItem, _ = after.Goal.WorkItem(liveItem.Ref())
	failedExecution, _ = sqliteExecutionByRef(after.Executions, failedExecution.Ref)
	liveExecution, _ = sqliteExecutionByRef(after.Executions, liveExecution.Ref)
	interruptCause, interrupted := failedItem.InterruptCause()
	if after.Goal.State() != goal.GoalStateCanceled ||
		failedItem.State() != goal.WorkItemStateCanceled || !interrupted ||
		interruptCause != goal.WorkItemInterruptExecutionFailed ||
		failedExecution.State != application.ExecutionFailed ||
		failedExecution.FailureCode != "test_attestor.required_tests_failed" ||
		liveItem.State() != goal.WorkItemStateCanceled ||
		liveExecution.State != application.ExecutionCanceled {
		t.Fatalf("canceled failed attestation goal=%s failed=%+v/%+v live=%+v/%+v",
			after.Goal.State(), failedItem, failedExecution, liveItem, liveExecution)
	}

	broker.revokeFailures = 1
	firstRevoke, err := system.orchestrator.ProcessNext(ctx, "worker:sqlite-cancel-failed-attestation-first")
	if err != nil || !firstRevoke.Processed || firstRevoke.Action != application.ActionRevokeSession ||
		broker.revokes != baselineRevokes+1 || !broker.revoked {
		t.Fatalf("first revoke result=%+v calls=%d revoked=%v err=%v",
			firstRevoke, broker.revokes, broker.revoked, err)
	}
	if _, err := system.repository.GetGoal(ctx, before.Goal.Ref()); err != nil {
		t.Fatalf("read after requeued revocation: %s", sqliteTestErrorChain(err))
	}
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("pending revocation recovery: %s", sqliteTestErrorChain(err))
	}

	system.clock.Advance(2 * time.Second)
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16OrchestratorWithStateAttestorAndSessions(
		t, system, system.repository, attestor, broker,
	)
	if _, err := system.repository.GetGoal(ctx, before.Goal.Ref()); err != nil {
		t.Fatalf("restart canceled failed attestation: %s", sqliteTestErrorChain(err))
	}
	secondRevoke, err := system.orchestrator.ProcessNext(ctx, "worker:sqlite-cancel-failed-attestation-restart")
	if err != nil || !secondRevoke.Processed || secondRevoke.Action != application.ActionRevokeSession ||
		broker.revokes != baselineRevokes+2 || broker.replays != baselineReplays+1 {
		t.Fatalf("idempotent revoke result=%+v calls=%d replays=%d err=%v",
			secondRevoke, broker.revokes, broker.replays, err)
	}
	var receipts int
	err = system.repository.db.QueryRow(`
SELECT COUNT(*) FROM action_consumption_receipts
WHERE kind='revoke_execution_session' AND execution_ref=? AND outcome='completed'`,
		liveExecution.Ref.String(),
	).Scan(&receipts)
	if err != nil || receipts != 1 {
		t.Fatalf("revocation receipts=%d err=%v", receipts, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("completed revocation recovery: %s", sqliteTestErrorChain(err))
	}
}

type sqliteSplitVerdictAttestor struct {
	mu     sync.Mutex
	calls  int
	passed sqliteTestAttestor
	failed sqliteTestAttestor
}

func (attestor *sqliteSplitVerdictAttestor) Attest(
	ctx context.Context,
	run ports.TestAttestationRun,
) (ports.TestAttestationResult, error) {
	attestor.mu.Lock()
	attestor.calls++
	call := attestor.calls
	attestor.mu.Unlock()
	if call == 1 {
		return attestor.passed.Attest(ctx, run)
	}
	attestor.failed.verdict = ports.TestAttestationFailed
	return attestor.failed.Attest(ctx, run)
}
