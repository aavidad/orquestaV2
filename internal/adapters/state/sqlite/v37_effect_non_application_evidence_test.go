package sqlite

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestV37RequeuePersistsExactStopNonApplicationWithoutSettlement(t *testing.T) {
	system, claim, execution, attempt := seedV37ClaimedStop(t)
	outcome := v37StopOutcome(attempt, system.clock.Now())
	if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: system.clock.Now().Add(time.Second),
		ErrorCode: "agent.stop_definitely_not_applied", OperationAt: system.clock.Now(),
		EffectAttemptOutcome: &outcome,
	}); err != nil {
		t.Fatalf("requeue definitely-not-applied: %v cause=%v", err, errors.Unwrap(err))
	}
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	if err != nil || len(record.EffectAttemptOutcomes) != 1 ||
		application.ValidateEffectAttemptOutcome(attempt, record.EffectAttemptOutcomes[0]) != nil ||
		len(record.BudgetSettlements) != 0 || len(record.EffectReceipts) != 1 {
		t.Fatalf("record=%+v err=%v", record, err)
	}
	var token any
	if err := system.repository.db.QueryRow(`SELECT claim_token FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(&token); err != nil || token != nil {
		t.Fatalf("claim not released token=%v err=%v", token, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid V37 recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestV37RejectsCrossedDuplicateAndTerminalStopOutcomeAtomically(t *testing.T) {
	for name, mutate := range map[string]func(*application.EffectAttemptOutcome){
		"crossed fence":  func(value *application.EffectAttemptOutcome) { value.ActionFence++ },
		"crossed intent": func(value *application.EffectAttemptOutcome) { value.IntentRef += ":crossed" },
		"future": func(value *application.EffectAttemptOutcome) {
			value.ObservedAt = value.ObservedAt.Add(2 * time.Minute)
		},
	} {
		t.Run(name, func(t *testing.T) {
			system, claim, execution, attempt := seedV37ClaimedStop(t)
			outcome := v37StopOutcome(attempt, system.clock.Now())
			mutate(&outcome)
			err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
				Claim: claim, Execution: execution, AvailableAt: system.clock.Now().Add(time.Second),
				ErrorCode: "agent.stop_definitely_not_applied", OperationAt: system.clock.Now(),
				EffectAttemptOutcome: &outcome,
			})
			if err == nil {
				t.Fatal("invalid outcome accepted")
			}
			var evidence int
			if scanErr := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_non_application_evidence`).Scan(&evidence); scanErr != nil || evidence != 0 {
				t.Fatalf("partial evidence=%d err=%v", evidence, scanErr)
			}
		})
	}
}

func TestV37RejectsDuplicateExactAndTerminalLedgerAtomically(t *testing.T) {
	t.Run("duplicate exact", func(t *testing.T) {
		system, claim, execution, attempt := seedV37ClaimedStop(t)
		outcome := v37StopOutcome(attempt, system.clock.Now())
		state := v37StopRequeueState(system, claim, execution, outcome)
		transaction, err := beginTransaction(context.Background(), system.repository)
		sqliteTestNoError(t, err)
		sqliteTestNoError(t, insertEffectAttemptOutcome(context.Background(), transaction, state))
		sqliteTestNoError(t, commit(transaction))
		assertV37RequeueRejectedWithoutMutation(t, system, state)
	})

	t.Run("terminal receipt", func(t *testing.T) {
		system, claim, execution, attempt := seedV37ClaimedStop(t)
		receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusStopped, system.clock.Now())
		transaction, err := beginTransaction(context.Background(), system.repository)
		sqliteTestNoError(t, err)
		sqliteTestNoError(t, insertEffectReceipt(context.Background(), transaction, claim, receipt, receipt.ConfirmedAt))
		sqliteTestNoError(t, commit(transaction))
		assertV37RequeueRejectedWithoutMutation(t, system,
			v37StopRequeueState(system, claim, execution, v37StopOutcome(attempt, system.clock.Now())))
	})

	t.Run("terminal settlement", func(t *testing.T) {
		system, claim, execution, attempt := seedV37ClaimedStop(t)
		record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
		sqliteTestNoError(t, err)
		if len(record.BudgetReservations) != 1 {
			t.Fatalf("launch reservation count=%d", len(record.BudgetReservations))
		}
		reservation := record.BudgetReservations[0]
		zero := governance.ResourceVector{Currency: reservation.Resources.Currency}
		settlement, err := governance.Reconcile(reservation, governance.ResourceUsage{
			Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
		})
		sqliteTestNoError(t, err)
		settlement.SettledAt = system.clock.Now()
		transaction, err := beginTransaction(context.Background(), system.repository)
		sqliteTestNoError(t, err)
		sqliteTestNoError(t, insertBudgetSettlement(context.Background(), transaction, settlement))
		sqliteTestNoError(t, commit(transaction))
		rewriteRecoveryTrigger(t, system.repository.db, "budget_settlements_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `UPDATE budget_settlements
SET causal_attempt_ref=? WHERE ref=?`, attempt.Ref, settlement.Ref)
		})
		state := v37StopRequeueState(system, claim, execution, v37StopOutcome(attempt, system.clock.Now()))
		assertV37RequeueRejectedWithoutMutation(t, system, state)
	})
}

func TestV37ConcurrentDefinitelyNotAppliedRequeueHasOneWriter(t *testing.T) {
	system, claim, execution, attempt := seedV37ClaimedStop(t)
	outcome := v37StopOutcome(attempt, system.clock.Now())
	state := v37StopRequeueState(system, claim, execution, outcome)
	start := make(chan struct{})
	errorsByWriter := make([]error, 2)
	var writers sync.WaitGroup
	for index := range errorsByWriter {
		writers.Add(1)
		go func() {
			defer writers.Done()
			<-start
			errorsByWriter[index] = system.repository.RequeueAction(context.Background(), state)
		}()
	}
	close(start)
	writers.Wait()
	succeeded, conflicted := 0, 0
	for _, err := range errorsByWriter {
		switch {
		case err == nil:
			succeeded++
		case application.IsStateError(err, application.StateConflict):
			conflicted++
		default:
			t.Fatalf("concurrent requeue error=%v cause=%v", err, errors.Unwrap(err))
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("concurrent requeue succeeded=%d conflicted=%d errors=%v", succeeded, conflicted, errorsByWriter)
	}
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	if err != nil || len(record.EffectAttemptOutcomes) != 1 ||
		application.ValidateEffectAttemptOutcome(attempt, record.EffectAttemptOutcomes[0]) != nil ||
		len(record.BudgetSettlements) != 0 {
		t.Fatalf("record=%+v err=%v", record, err)
	}
	var evidence int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_non_application_evidence`).Scan(&evidence); err != nil || evidence != 1 {
		t.Fatalf("evidence=%d err=%v", evidence, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery after concurrent requeue: %v cause=%v", err, errors.Unwrap(err))
	}
}

func v37StopRequeueState(
	system *sqliteV15System, claim application.ActionClaim, execution application.ExecutionRecord,
	outcome application.EffectAttemptOutcome,
) application.ActionRequeuedState {
	return application.ActionRequeuedState{
		Claim: claim, Execution: execution, AvailableAt: system.clock.Now().Add(time.Second),
		ErrorCode: "agent.stop_definitely_not_applied", OperationAt: system.clock.Now(),
		EffectAttemptOutcome: &outcome,
	}
}

func assertV37RequeueRejectedWithoutMutation(
	t *testing.T, system *sqliteV15System, state application.ActionRequeuedState,
) {
	t.Helper()
	before, err := system.repository.GetGoal(context.Background(), state.Claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	beforeOutbox := sqliteV28RecoveryRequeueOutbox(t, system.repository.db, state.Claim.Action.Ref)
	if err := system.repository.RequeueAction(context.Background(), state); err == nil {
		t.Fatal("terminal or duplicate stop outcome accepted")
	}
	after, err := system.repository.GetGoal(context.Background(), state.Claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	afterOutbox := sqliteV28RecoveryRequeueOutbox(t, system.repository.db, state.Claim.Action.Ref)
	if !reflect.DeepEqual(after, before) || afterOutbox != beforeOutbox {
		t.Fatalf("atomic rollback failed record_changed=%t outbox before=%+v after=%+v",
			!reflect.DeepEqual(after, before), beforeOutbox, afterOutbox)
	}
}

func v37StopOutcome(attempt application.EffectAttempt, at time.Time) application.EffectAttemptOutcome {
	return application.EffectAttemptOutcome{
		Ref:        "effect-attempt-outcome:" + attempt.Ref + ":definitely-not-applied",
		AttemptRef: attempt.Ref, IntentRef: attempt.IntentRef, IntentDigest: attempt.IntentDigest,
		ApprovalRef: attempt.ApprovalRef, Subject: attempt.Subject, ActionRef: attempt.ActionRef,
		ActionFence: attempt.ActionFence, IdempotencyKey: attempt.IdempotencyKey,
		Outcome: application.EffectAttemptDefinitelyNotApplied, ObservedAt: at.UTC(),
	}
}

func seedV37ClaimedStop(t *testing.T) (*sqliteV15System, application.ActionClaim, application.ExecutionRecord, application.EffectAttempt) {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	created := system.submit(t, "request:v37-stop")
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v37-launch")
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch=%+v err=%v", result, err)
	}
	record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item := record.Goal.WorkItems()[0]
	execution := record.Executions[0]
	_, err = system.orchestrator.Control(context.Background(), system.access, application.ControlRequest{
		RequestRef: "control:v37-stop", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(), ExecutionRef: execution.Ref,
		ExpectedExecutionAttempt: execution.AttemptNo, Mode: ports.AgentStopCooperative,
		Reason: "prove durable non-application",
	})
	sqliteTestNoError(t, err)
	claim := claimSQLiteV15(t, system, "claim:v37-stop")
	if claim.Action.Kind != application.ActionStopAgent {
		t.Fatalf("claimed %s", claim.Action.Kind)
	}
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	if _, createdAttempt, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	}); err != nil || !createdAttempt {
		t.Fatalf("attempt created=%v err=%v", createdAttempt, err)
	}
	return system, claim, execution, attempt
}
