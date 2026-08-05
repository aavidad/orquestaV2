package sqlite

import (
	"context"
	"database/sql"
)

func validateRecoveryV27EffectAttemptClaimLease(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{
		{
			"sqlite.recovery_v27_effect_attempt_claim_lease_invalid",
			`SELECT COUNT(*) FROM effect_attempts attempt
WHERE attempt.claim_lease_until IS NOT NULL
 AND attempt.claim_lease_until<=attempt.started_at`,
		},
		{
			"sqlite.recovery_v27_effect_receipt_outside_claim_lease",
			`SELECT COUNT(*) FROM effect_receipts receipt
JOIN effect_attempts attempt ON attempt.ref=receipt.attempt_ref
WHERE attempt.claim_lease_until IS NOT NULL
 AND receipt.confirmed_at>=attempt.claim_lease_until`,
		},
		{
			"sqlite.recovery_v27_effect_attempt_claim_lease_missing_proof",
			`SELECT COUNT(*) FROM effect_attempts attempt
WHERE attempt.claim_lease_until IS NULL
 AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt
  WHERE receipt.attempt_ref=attempt.ref
   AND receipt.intent_digest=attempt.intent_digest
   AND receipt.approval_ref=attempt.approval_ref
   AND receipt.project_ref=attempt.project_ref
   AND receipt.goal_ref=attempt.goal_ref
   AND receipt.work_item_ref=attempt.work_item_ref
   AND receipt.execution_ref=attempt.execution_ref
   AND receipt.plan_generation=attempt.plan_generation
   AND receipt.app_spec_generation=attempt.app_spec_generation
   AND receipt.spec_hash=attempt.spec_hash
   AND receipt.actor_ref=attempt.actor_ref
   AND receipt.action_ref=attempt.action_ref
   AND receipt.intent_ref=attempt.intent_ref
   AND receipt.action_fence=attempt.action_fence
   AND receipt.idempotency_key=attempt.idempotency_key
 )
 AND ((SELECT COUNT(*) FROM budget_settlements related
       WHERE related.causal_attempt_ref=attempt.ref)<>1
  OR (SELECT COUNT(*)
      FROM budget_settlements settlement
      JOIN budget_reservations reservation ON reservation.ref=settlement.reservation_ref
      WHERE settlement.causal_attempt_ref=attempt.ref
       AND reservation.action_ref=attempt.action_ref
       AND reservation.effect_intent_ref=attempt.intent_ref
       AND reservation.fence<=attempt.action_fence
       AND settlement.reserved_tokens=reservation.tokens
       AND settlement.reserved_money_micros=reservation.money_micros
       AND settlement.reserved_currency=reservation.currency
       AND settlement.reserved_active_time_ns=reservation.active_time_ns
       AND settlement.reserved_process_slots=reservation.process_slots
       AND settlement.reserved_disk_bytes=reservation.disk_bytes
       AND settlement.observed_known=31
       AND settlement.observed_quality='exact'
       AND settlement.observed_tokens=0
       AND settlement.observed_money_micros=0
       AND settlement.observed_currency=reservation.currency
       AND settlement.observed_active_time_ns=0
       AND settlement.observed_process_slots=0
       AND settlement.observed_disk_bytes=0
       AND settlement.charged_tokens=0
       AND settlement.charged_money_micros=0
       AND settlement.charged_currency=reservation.currency
       AND settlement.charged_active_time_ns=0
       AND settlement.charged_process_slots=0
       AND settlement.charged_disk_bytes=0
       AND settlement.released_tokens=reservation.tokens
       AND settlement.released_money_micros=reservation.money_micros
       AND settlement.released_currency=reservation.currency
       AND settlement.released_active_time_ns=reservation.active_time_ns
       AND settlement.released_process_slots=reservation.process_slots
       AND settlement.released_disk_bytes=reservation.disk_bytes
       AND settlement.overrun_tokens=0
       AND settlement.overrun_money_micros=0
       AND settlement.overrun_currency=reservation.currency
       AND settlement.overrun_active_time_ns=0
       AND settlement.overrun_process_slots=0
       AND settlement.overrun_disk_bytes=0)<>1)`,
		},
	})
}
