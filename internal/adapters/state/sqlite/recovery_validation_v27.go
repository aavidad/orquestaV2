package sqlite

import (
	"context"
	"database/sql"
)

func validateRecoveryV27EffectAttemptClaimLease(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryEffectAttemptClaimLease(ctx, tx, false)
}

func validateRecoveryV40EffectAttemptClaimLease(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryEffectAttemptClaimLease(ctx, tx, true)
}

func validateRecoveryEffectAttemptClaimLease(ctx context.Context, tx *sql.Tx, allowTerminalReconciliation bool) error {
	lateReceiptQuery := `SELECT COUNT(*) FROM effect_receipts receipt
JOIN effect_attempts attempt ON attempt.ref=receipt.attempt_ref
WHERE attempt.claim_lease_until IS NOT NULL
 AND receipt.confirmed_at>=attempt.claim_lease_until`
	var lifecycleTables int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_schema
WHERE type='table' AND name='agent_environment_lifecycles'`).Scan(&lifecycleTables); err != nil {
		return err
	}
	if lifecycleTables == 1 {
		lateReceiptQuery = `SELECT COUNT(*) FROM effect_receipts receipt
JOIN effect_attempts attempt ON attempt.ref=receipt.attempt_ref
JOIN effect_intents intent ON intent.ref=attempt.intent_ref
JOIN outbox action ON action.ref=attempt.action_ref
WHERE attempt.claim_lease_until IS NOT NULL
 AND receipt.confirmed_at>=attempt.claim_lease_until
 AND NOT (
  intent.kind IN ('agent_quiesce','agent_environment_preserve','agent_environment_close')
  AND EXISTS (
   SELECT 1 FROM action_consumption_receipts consumed
   JOIN agent_environment_lifecycles lifecycle ON lifecycle.execution_ref=consumed.execution_ref
   WHERE consumed.action_ref=attempt.action_ref
    AND consumed.effect_receipt_ref=receipt.ref
    AND consumed.goal_ref=attempt.goal_ref
    AND consumed.work_item_ref=attempt.work_item_ref
    AND consumed.execution_ref=attempt.execution_ref
    AND consumed.fence=attempt.action_fence
    AND consumed.claim_token=action.claim_token
    AND consumed.worker_ref=action.claimed_by
    AND consumed.delivery_attempt=action.delivery_attempt
    AND consumed.consumed_at=action.completed_at
    AND consumed.consumed_at>=receipt.confirmed_at
    AND consumed.governance_version=1
    AND consumed.outcome='completed'
    AND consumed.error_code=''
    AND consumed.kind=CASE intent.kind
     WHEN 'agent_quiesce' THEN 'quiesce_agent'
     WHEN 'agent_environment_preserve' THEN 'preserve_agent_environment'
     WHEN 'agent_environment_close' THEN 'close_agent_environment' END
    AND action.kind=consumed.kind
    AND action.effect_intent_ref=intent.ref
    AND action.goal_ref=attempt.goal_ref
    AND action.work_item_ref=attempt.work_item_ref
    AND action.execution_ref=attempt.execution_ref
    AND lifecycle.goal_ref=attempt.goal_ref
    AND lifecycle.work_item_ref=attempt.work_item_ref))`
	}
	if allowTerminalReconciliation {
		lateReceiptQuery += `
 AND NOT (` + terminalAgentLaunchReconciliationReceiptProof + `)`
	}
	return validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{
		{
			"sqlite.recovery_v27_effect_attempt_claim_lease_invalid",
			`SELECT COUNT(*) FROM effect_attempts attempt
WHERE attempt.claim_lease_until IS NOT NULL
 AND attempt.claim_lease_until<=attempt.started_at`,
		},
		{
			"sqlite.recovery_v27_effect_receipt_outside_claim_lease",
			lateReceiptQuery,
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
