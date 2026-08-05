package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestV28RecoveryPersistsHistoricalReceiptAndCurrentConsumptionAtomically(t *testing.T) {
	system, _, attempt := seedV27AmbiguousEffectAttempt(t, "v28-recovery-receipt-writer")
	system.clock.Advance(time.Minute + time.Nanosecond)
	claim := claimSQLiteV28Recovery(t, system, "claim:v28-recovery-receipt-writer", attempt.Ref)
	state := sqliteV28RecoveryLaunchAcceptedState(t, system, claim, attempt)

	if err := system.repository.RecordLaunchAccepted(context.Background(), state); err != nil {
		t.Fatalf("persist recovered receipt: %s", sqliteTestErrorChain(err))
	}
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	if len(record.EffectAttempts) != 1 || record.EffectAttempts[0] != attempt ||
		len(record.EffectReceipts) != 1 || record.EffectReceipts[0] != state.EffectReceipt ||
		len(record.ConsumptionReceipts) != 1 {
		t.Fatalf("attempts=%+v receipts=%+v consumptions=%+v",
			record.EffectAttempts, record.EffectReceipts, record.ConsumptionReceipts)
	}
	consumed := record.ConsumptionReceipts[0]
	if consumed.Fence != claim.Fence || consumed.EffectReceiptRef != state.EffectReceipt.Ref ||
		!consumed.ConsumedAt.Equal(state.OperationAt) || state.EffectReceipt.ActionFence != attempt.ActionFence ||
		!state.EffectReceipt.ConfirmedAt.Before(attempt.ClaimLeaseUntil) {
		t.Fatalf("receipt=%+v consumed=%+v claim fence=%d", state.EffectReceipt, consumed, claim.Fence)
	}
	var outboxFence int64
	var recoveryRef string
	var completedAt sql.NullInt64
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT fence,recovery_effect_attempt_ref,completed_at FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(
		&outboxFence, &recoveryRef, &completedAt,
	))
	if outboxFence != int64(claim.Fence) || recoveryRef != attempt.Ref ||
		!completedAt.Valid || completedAt.Int64 != requiredTime(state.OperationAt) {
		t.Fatalf("outbox fence=%d recovery=%q completed=%+v", outboxFence, recoveryRef, completedAt)
	}
	if err := system.repository.RecordLaunchAccepted(context.Background(), state); err == nil {
		t.Fatal("duplicate recovered receipt accepted")
	}
	replayed, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	if len(replayed.EffectAttempts) != 1 || len(replayed.EffectReceipts) != 1 ||
		len(replayed.ConsumptionReceipts) != 1 {
		t.Fatalf("duplicate changed attempts=%d receipts=%d consumptions=%d",
			len(replayed.EffectAttempts), len(replayed.EffectReceipts), len(replayed.ConsumptionReceipts))
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("durable recovered receipt invalid after reopen check: %s", sqliteTestErrorChain(err))
	}
}

func TestV28NormalReceiptKeepsClaimFenceAndTime(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	created := system.submit(t, "request:v28-normal-receipt-writer")
	if result, err := system.orchestrator.ProcessNext(
		context.Background(), "worker:v28-normal-receipt-writer",
	); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("normal launch result=%+v err=%v", result, err)
	}
	record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	if len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 1 || len(record.ConsumptionReceipts) != 1 {
		t.Fatalf("attempts=%d receipts=%d consumptions=%d",
			len(record.EffectAttempts), len(record.EffectReceipts), len(record.ConsumptionReceipts))
	}
	attempt, receipt, consumed := record.EffectAttempts[0], record.EffectReceipts[0], record.ConsumptionReceipts[0]
	if receipt.ActionFence != attempt.ActionFence || consumed.Fence != receipt.ActionFence ||
		consumed.EffectReceiptRef != receipt.Ref || consumed.ConsumedAt != receipt.ConfirmedAt {
		t.Fatalf("attempt=%+v receipt=%+v consumed=%+v", attempt, receipt, consumed)
	}
	var recoveryRef sql.NullString
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT recovery_effect_attempt_ref FROM outbox WHERE ref=?`, receipt.ActionRef).Scan(&recoveryRef))
	if recoveryRef.Valid {
		t.Fatalf("normal outbox recovery ref=%q", recoveryRef.String)
	}
}

func TestV28RecoveryReceiptWriterRejectsCrossedClaimFenceRefAndTimesAtomically(t *testing.T) {
	tests := map[string]func(*application.LaunchAcceptedState, application.EffectAttempt){
		"normal divergent fence": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.Claim.Disposition = application.ActionClaimDispositionNormal
			state.Claim.RecoveryEffectAttemptRef = ""
		},
		"recovery ref missing": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.Claim.RecoveryEffectAttemptRef = ""
		},
		"recovery ref crossed": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.Claim.RecoveryEffectAttemptRef += ":crossed"
		},
		"recovery action crossed": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.Claim.Action.Kind = application.ActionStopAgent
		},
		"recovery fence not superior": func(state *application.LaunchAcceptedState, attempt application.EffectAttempt) {
			state.Claim.Fence = attempt.ActionFence
		},
		"receipt fence crossed to current": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.EffectReceipt.ActionFence = state.Claim.Fence
		},
		"receipt attempt crossed": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.EffectReceipt.AttemptRef += ":crossed"
		},
		"receipt before historical start": func(state *application.LaunchAcceptedState, attempt application.EffectAttempt) {
			state.EffectReceipt.ConfirmedAt = attempt.StartedAt.Add(-time.Nanosecond)
		},
		"receipt at historical lease boundary": func(state *application.LaunchAcceptedState, attempt application.EffectAttempt) {
			state.EffectReceipt.ConfirmedAt = attempt.ClaimLeaseUntil
		},
		"receipt after current operation": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.EffectReceipt.ConfirmedAt = state.OperationAt.Add(time.Nanosecond)
		},
		"current event time crossed": func(state *application.LaunchAcceptedState, _ application.EffectAttempt) {
			state.Event.OccurredAt = state.OperationAt.Add(time.Nanosecond)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			system, _, attempt := seedV27AmbiguousEffectAttempt(t, "v28-writer-negative-"+testRefSuffix(name))
			system.clock.Advance(time.Minute + time.Nanosecond)
			claim := claimSQLiteV28Recovery(t, system, "claim:v28-writer-negative:"+testRefSuffix(name), attempt.Ref)
			state := sqliteV28RecoveryLaunchAcceptedState(t, system, claim, attempt)
			mutate(&state, attempt)
			if err := system.repository.RecordLaunchAccepted(context.Background(), state); err == nil {
				t.Fatalf("crossed recovery state accepted: %+v", state)
			}
			assertV28RecoveryReceiptWriterUnchanged(t, system, claim, attempt)
		})
	}
}

func assertV28RecoveryReceiptWriterUnchanged(
	t *testing.T,
	system *sqliteV15System,
	claim application.ActionClaim,
	attempt application.EffectAttempt,
) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found || execution.State != application.ExecutionDispatching ||
		len(record.EffectAttempts) != 1 || record.EffectAttempts[0] != attempt ||
		len(record.EffectReceipts) != 0 || len(record.ConsumptionReceipts) != 0 {
		t.Fatalf("execution=%+v found=%t attempts=%+v receipts=%+v consumptions=%+v",
			execution, found, record.EffectAttempts, record.EffectReceipts, record.ConsumptionReceipts)
	}
	var fence int64
	var recoveryRef string
	var completedAt sql.NullInt64
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT fence,recovery_effect_attempt_ref,completed_at FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(
		&fence, &recoveryRef, &completedAt,
	))
	if fence != int64(claim.Fence) || recoveryRef != attempt.Ref || completedAt.Valid {
		t.Fatalf("outbox fence=%d recovery=%q completed=%+v", fence, recoveryRef, completedAt)
	}
}

func testRefSuffix(value string) string {
	result := make([]byte, len(value))
	for index := range value {
		if value[index] >= 'a' && value[index] <= 'z' {
			result[index] = value[index]
		} else {
			result[index] = '-'
		}
	}
	return string(result)
}

func sqliteV28RecoveryLaunchAcceptedState(
	t *testing.T,
	system *sqliteV15System,
	claim application.ActionClaim,
	attempt application.EffectAttempt,
) application.LaunchAcceptedState {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found {
		t.Fatal("recovery WorkItem missing")
	}
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found || execution.State != application.ExecutionDispatching {
		t.Fatalf("recovery execution=%+v found=%t", execution, found)
	}
	confirmedAt := attempt.ClaimLeaseUntil.Add(-time.Nanosecond)
	operationAt := system.clock.Now().UTC()
	receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusAccepted, confirmedAt)
	execution.State = application.ExecutionRunning
	execution.ProviderRef, execution.ModelRef, execution.AgentRef = "provider:codex", "model:codex", "agent:codex"
	execution.ExternalRef, execution.LaunchReceiptRef = "external:"+execution.Ref.String(), receipt.Ref
	execution.StartedAt, execution.ProviderAcceptedAt = attempt.StartedAt, confirmedAt
	execution.DeadlineAt = operationAt.Add(time.Hour)
	return application.LaunchAcceptedState{
		Claim: claim, Execution: execution, EffectReceipt: receipt, OperationAt: operationAt,
		NextAction: application.ActionRecord{
			Ref: "action:observe:" + execution.Ref.String(), Kind: application.ActionObserveAgent,
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: record.Goal.PlanGeneration(), WorkItemGeneration: item.Revision(), AvailableAt: operationAt,
		},
		Event: application.EventRecord{
			Ref: "event:recovered-accepted:" + execution.Ref.String(), Kind: "execution.accepted",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: operationAt,
		},
	}
}
