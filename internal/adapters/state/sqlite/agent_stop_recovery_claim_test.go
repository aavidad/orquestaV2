package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestV38StopRecoveryClaimSurvivesRestartAndReclaimCAS(t *testing.T) {
	system, original, _, attempt := seedV37ClaimedStop(t)
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	mustV10Exec(t, system.repository.db, `UPDATE outbox
SET claim_token=NULL,claimed_by=NULL,claimed_until=NULL,available_at=?,last_error_code='agent.stop_unknown_applied'
WHERE ref=?`, requiredTime(system.clock.Now()), original.Action.Ref)

	request := application.ClaimRequest{
		WorkerRef: "worker:v38-stop-recovery-first", Token: "claim:v38-stop-recovery-first",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
		CapacityCandidates: system.capacidad,
	}
	first, found, err := system.repository.ClaimNextAction(context.Background(), request)
	if err != nil || !found || first.Action.Kind != application.ActionStopAgent ||
		first.Disposition != application.ActionClaimDispositionRecoverEffect ||
		first.RecoveryEffectAttemptRef != attempt.Ref || first.Fence <= attempt.ActionFence ||
		first.BudgetReservationRef != "" || first.CapacityReservation.Ref != "" {
		t.Fatalf("first=%+v found=%t err=%s", first, found, sqliteTestErrorChain(err))
	}
	sqliteTestNoError(t, system.repository.ValidateAgentStopRecoveryClaim(context.Background(), first))

	sqliteTestNoError(t, system.repository.Close())
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	sqliteTestNoError(t, restarted.ValidateAgentStopRecoveryClaim(context.Background(), first))

	system.clock.Advance(first.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	request.WorkerRef = "worker:v38-stop-recovery-second"
	request.Token = "claim:v38-stop-recovery-second"
	second, found, err := restarted.ClaimNextAction(context.Background(), request)
	if err != nil || !found || second.Disposition != application.ActionClaimDispositionRecoverEffect ||
		second.RecoveryEffectAttemptRef != attempt.Ref || second.Fence <= first.Fence {
		t.Fatalf("second=%+v found=%t err=%s", second, found, sqliteTestErrorChain(err))
	}
	if err := restarted.ValidateAgentStopRecoveryClaim(context.Background(), first); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale recovery claim accepted: %s", sqliteTestErrorChain(err))
	}
	sqliteTestNoError(t, restarted.ValidateAgentStopRecoveryClaim(context.Background(), second))
}

func TestV38StopRecoveryClaimRejectsResolvedReceipt(t *testing.T) {
	system, original, _, attempt := seedV37ClaimedStop(t)
	receipt := sqliteV15EffectReceipt(original, attempt, application.EffectStatusStopped, system.clock.Now())
	transaction, err := beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, insertEffectReceipt(context.Background(), transaction, original, receipt, receipt.ConfirmedAt))
	sqliteTestNoError(t, commit(transaction))
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	mustV10Exec(t, system.repository.db, `UPDATE outbox
SET claim_token=NULL,claimed_by=NULL,claimed_until=NULL,available_at=?,last_error_code='agent.stop_unknown_applied'
WHERE ref=?`, requiredTime(system.clock.Now()), original.Action.Ref)

	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v38-stop-resolved", Token: "claim:v38-stop-resolved",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
		CapacityCandidates: system.capacidad,
	})
	if !application.IsStateError(err, application.StateConflict) || found || claim != (application.ActionClaim{}) {
		t.Fatalf("resolved receipt claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
}
