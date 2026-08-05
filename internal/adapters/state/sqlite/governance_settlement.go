package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
)

func insertEffectReceipt(
	ctx context.Context,
	transaction *sql.Tx,
	claim application.ActionClaim,
	receipt application.EffectReceipt,
	consumedAt time.Time,
) error {
	if !validText(receipt.Ref) || !validText(receipt.IntentRef) || !validText(receipt.IntentDigest) ||
		!validText(receipt.ApprovalRef) || !validText(receipt.AttemptRef) ||
		receipt.Subject.ProjectRef.String() == "" || receipt.Subject.GoalRef.String() == "" ||
		receipt.Subject.WorkItemRef.String() == "" || receipt.Subject.ExecutionRef.String() == "" ||
		receipt.Subject.PlanGeneration == 0 || receipt.Subject.AppSpecGeneration == 0 ||
		!validText(receipt.Subject.SpecHash) || receipt.Subject.ActorRef.String() == "" ||
		!validText(receipt.ActionRef) || receipt.ActionFence == 0 ||
		!validText(receipt.IdempotencyKey) || !validText(receipt.ExternalRef) ||
		!validSQLiteEffectStatus(receipt.Status) || receipt.ConfirmedAt.IsZero() ||
		governance.ValidateResourceUsage(receipt.Usage) != nil {
		return invalid(errors.New("sqlite.effect_receipt_invalid"))
	}
	confirmationClaimValid := claim.Disposition == application.ActionClaimDispositionNormal &&
		claim.Action.Ref == receipt.ActionRef && claim.Action.EffectIntentRef == receipt.IntentRef &&
		claim.Fence == receipt.ActionFence && receipt.ConfirmedAt.Equal(consumedAt)
	if claim.Disposition == application.ActionClaimDispositionRecoverEffect {
		confirmationClaimValid = claim.Action.Kind == application.ActionLaunchAgent &&
			claim.Action.Ref == receipt.ActionRef && claim.Action.EffectIntentRef == receipt.IntentRef &&
			claim.RecoveryEffectAttemptRef == receipt.AttemptRef && receipt.ActionFence < claim.Fence &&
			receipt.ConfirmedAt.Before(consumedAt)
	}
	if !confirmationClaimValid {
		return conflict(errors.New("sqlite.effect_receipt_claim_conflict"))
	}
	attempt, found, err := readEffectAttemptByFence(
		ctx, transaction, receipt.ActionRef, receipt.ActionFence,
	)
	if err != nil {
		return err
	}
	if !found || attempt.Ref != receipt.AttemptRef || attempt.IntentRef != receipt.IntentRef ||
		attempt.IntentDigest != receipt.IntentDigest || attempt.ApprovalRef != receipt.ApprovalRef ||
		attempt.Subject != receipt.Subject || attempt.IdempotencyKey != receipt.IdempotencyKey ||
		receipt.ConfirmedAt.Before(attempt.StartedAt) ||
		!receipt.ConfirmedAt.Before(attempt.ClaimLeaseUntil) {
		return conflict(errors.New("sqlite.effect_receipt_attempt_conflict"))
	}
	var outboxFence, claimedUntil int64
	var recoveryAttemptRef sql.NullString
	if err := transaction.QueryRowContext(ctx, `
SELECT fence,claimed_until,recovery_effect_attempt_ref FROM outbox WHERE ref=?`,
		claim.Action.Ref,
	).Scan(&outboxFence, &claimedUntil, &recoveryAttemptRef); err != nil {
		return mapDatabaseError(err)
	}
	if outboxFence != int64(claim.Fence) || claimedUntil != requiredTime(claim.LeaseUntil) ||
		(claim.Disposition == application.ActionClaimDispositionNormal && recoveryAttemptRef.Valid) ||
		(claim.Disposition == application.ActionClaimDispositionRecoverEffect &&
			(!recoveryAttemptRef.Valid || recoveryAttemptRef.String != receipt.AttemptRef)) {
		return conflict(errors.New("sqlite.effect_receipt_outbox_conflict"))
	}
	if claim.Disposition == application.ActionClaimDispositionNormal &&
		!receipt.ConfirmedAt.Before(time.Unix(0, claimedUntil).UTC()) {
		return invalid(errors.New("sqlite.effect_receipt_after_lease"))
	}
	intent, err := readEffectIntent(ctx, transaction, receipt.IntentRef)
	if err != nil {
		return err
	}
	if intent.Digest != receipt.IntentDigest || !validSQLiteEffectStatusForKind(intent.Kind, receipt.Status) {
		return invalid(errors.New("sqlite.effect_receipt_kind_status_invalid"))
	}
	usage := receipt.Usage.Resources
	_, err = transaction.ExecContext(ctx, `
INSERT INTO effect_receipts(
    ref, intent_ref, intent_digest, approval_ref, attempt_ref,
    project_ref, goal_ref, work_item_ref, execution_ref, plan_generation,
    app_spec_generation, spec_hash, actor_ref, action_ref, action_fence,
    idempotency_key, external_ref, status, usage_tokens, usage_money_micros,
    usage_currency, usage_active_time_ns, usage_process_slots, usage_disk_bytes,
    usage_known, usage_quality, confirmed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		receipt.Ref, receipt.IntentRef, receipt.IntentDigest, receipt.ApprovalRef, receipt.AttemptRef,
		receipt.Subject.ProjectRef.String(), receipt.Subject.GoalRef.String(),
		receipt.Subject.WorkItemRef.String(), receipt.Subject.ExecutionRef.String(),
		int64(receipt.Subject.PlanGeneration), int64(receipt.Subject.AppSpecGeneration),
		receipt.Subject.SpecHash, receipt.Subject.ActorRef.String(), receipt.ActionRef,
		int64(receipt.ActionFence), receipt.IdempotencyKey, receipt.ExternalRef, receipt.Status,
		usage.Tokens, usage.MoneyMicros, string(usage.Currency), usage.ActiveTimeNS,
		usage.ProcessSlots, usage.DiskBytes, int64(receipt.Usage.Known),
		string(receipt.Usage.Quality), requiredTime(receipt.ConfirmedAt),
	)
	return mapDatabaseError(err)
}

func validSQLiteEffectStatus(status application.EffectStatus) bool {
	switch status {
	case application.EffectStatusAccepted, application.EffectStatusStopped,
		application.EffectStatusAlreadyStopped, application.EffectStatusAlreadyCompleted,
		application.EffectStatusAlreadyFailed, application.EffectStatusPrepared,
		application.EffectStatusCommitted, application.EffectStatusAttestedPassed,
		application.EffectStatusAttestedFailed, application.EffectStatusIntegrated,
		application.EffectStatusConflicted, application.EffectStatusStale:
		return true
	default:
		return false
	}
}

func validSQLiteEffectStatusForKind(kind application.EffectKind, status application.EffectStatus) bool {
	switch kind {
	case application.EffectKindAgentLaunch:
		return status == application.EffectStatusAccepted
	case application.EffectKindAgentStop:
		return status == application.EffectStatusStopped ||
			status == application.EffectStatusAlreadyStopped ||
			status == application.EffectStatusAlreadyCompleted ||
			status == application.EffectStatusAlreadyFailed
	case application.EffectKindPrepareWorkspace:
		return status == application.EffectStatusPrepared
	case application.EffectKindCommitChange:
		return status == application.EffectStatusCommitted
	case application.EffectKindAttestTest:
		return status == application.EffectStatusAttestedPassed ||
			status == application.EffectStatusAttestedFailed
	case application.EffectKindIntegrateChange:
		return status == application.EffectStatusIntegrated ||
			status == application.EffectStatusConflicted ||
			status == application.EffectStatusStale
	default:
		return false
	}
}

func insertBudgetSettlement(
	ctx context.Context,
	transaction *sql.Tx,
	settlement governance.BudgetSettlement,
) error {
	if err := governance.ValidateBudgetSettlement(settlement); err != nil {
		return invalid(err)
	}
	reservation, err := readBudgetReservation(ctx, transaction, settlement.ReservationRef)
	if err != nil {
		return err
	}
	if settlement.SettledAt.Before(reservation.ReservedAt) {
		return invalid(errors.New("sqlite.budget_settlement_time_invalid"))
	}
	if reservation.Resources != settlement.Reserved {
		return conflict(errors.New("sqlite.budget_settlement_reservation_conflict"))
	}
	if settlement.CausalAttemptRef != "" {
		var causal int
		err = transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM effect_attempts attempt
WHERE attempt.ref=? AND attempt.action_ref=? AND attempt.intent_ref=? AND ?<=attempt.action_fence`,
			settlement.CausalAttemptRef, reservation.ActionRef, reservation.EffectIntentRef,
			reservation.Fence).Scan(&causal)
		if err != nil {
			return mapDatabaseError(err)
		}
		if causal != 1 || !governance.IsExactZeroRelease(settlement) {
			return conflict(errors.New("sqlite.budget_settlement_causal_attempt_invalid"))
		}
	}
	reserved, observed := settlement.Reserved, settlement.Observed.Resources
	charged, released, overrun := settlement.Charged, settlement.Released, settlement.Overrun
	_, err = transaction.ExecContext(ctx, `
INSERT INTO budget_settlements(
    ref, reservation_ref, causal_attempt_ref,
    reserved_tokens, reserved_money_micros, reserved_currency,
    reserved_active_time_ns, reserved_process_slots, reserved_disk_bytes,
    observed_tokens, observed_money_micros, observed_currency,
    observed_active_time_ns, observed_process_slots, observed_disk_bytes,
    observed_known, observed_quality,
    charged_tokens, charged_money_micros, charged_currency,
    charged_active_time_ns, charged_process_slots, charged_disk_bytes,
    released_tokens, released_money_micros, released_currency,
    released_active_time_ns, released_process_slots, released_disk_bytes,
    overrun_tokens, overrun_money_micros, overrun_currency,
    overrun_active_time_ns, overrun_process_slots, overrun_disk_bytes, settled_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
          ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, settlement.Ref, settlement.ReservationRef,
		nullableString(settlement.CausalAttemptRef),
		reserved.Tokens, reserved.MoneyMicros, string(reserved.Currency), reserved.ActiveTimeNS,
		reserved.ProcessSlots, reserved.DiskBytes, observed.Tokens, observed.MoneyMicros,
		string(observed.Currency), observed.ActiveTimeNS, observed.ProcessSlots, observed.DiskBytes,
		int64(settlement.Observed.Known), string(settlement.Observed.Quality),
		charged.Tokens, charged.MoneyMicros, string(charged.Currency), charged.ActiveTimeNS,
		charged.ProcessSlots, charged.DiskBytes, released.Tokens, released.MoneyMicros,
		string(released.Currency), released.ActiveTimeNS, released.ProcessSlots, released.DiskBytes,
		overrun.Tokens, overrun.MoneyMicros, string(overrun.Currency), overrun.ActiveTimeNS,
		overrun.ProcessSlots, overrun.DiskBytes, requiredTime(settlement.SettledAt),
	)
	return mapDatabaseError(err)
}
