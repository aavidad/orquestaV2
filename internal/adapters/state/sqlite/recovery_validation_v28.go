package sqlite

import (
	"context"
	"database/sql"
)

func validateRecoveryV28EffectRecoveryClaim(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryEffectRecoveryClaim(ctx, tx, false, false, false)
}

func validateRecoveryV30EffectRecoveryClaim(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryEffectRecoveryClaim(ctx, tx, true, false, false)
}

func validateRecoveryV40EffectRecoveryClaim(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryEffectRecoveryClaim(ctx, tx, true, true, false)
}

func validateRecoveryV41EffectRecoveryClaim(ctx context.Context, tx *sql.Tx) error {
	return validateRecoveryEffectRecoveryClaim(ctx, tx, true, true, true)
}

func validateRecoveryEffectRecoveryClaim(
	ctx context.Context,
	tx *sql.Tx,
	allowUnclaimedRetry, allowStopPending, allowStopTerminal bool,
) error {
	pendingClaim := `(action.claim_token IS NOT NULL AND action.claimed_by IS NOT NULL
           AND action.claimed_until IS NOT NULL)`
	if allowUnclaimedRetry {
		pendingClaim = `(` + pendingClaim + `
       OR (action.claim_token IS NULL AND action.claimed_by IS NULL
		   AND action.claimed_until IS NULL AND length(trim(action.last_error_code))>0))`
	}
	validKind := `(action.kind='launch_agent' AND intent.kind='agent_launch')`
	if allowStopPending {
		validKind = `(` + validKind + ` OR (action.kind='stop_agent' AND intent.kind='agent_stop'))`
	}
	stopTerminal := ""
	if allowStopTerminal {
		stopTerminal = ` OR
	(action.kind='stop_agent' AND action.completed_at IS NOT NULL AND action.quarantined_at IS NULL
     AND EXISTS (SELECT 1 FROM action_consumption_receipts consumed
      JOIN effect_receipts receipt ON receipt.ref=consumed.effect_receipt_ref
      WHERE consumed.action_ref=action.ref AND consumed.fence=action.fence
       AND consumed.claim_token=action.claim_token AND consumed.worker_ref=action.claimed_by
       AND consumed.delivery_attempt=action.delivery_attempt
       AND consumed.consumed_at=action.completed_at AND consumed.outcome='completed'
       AND consumed.error_code='' AND receipt.attempt_ref=action.recovery_effect_attempt_ref
       AND receipt.action_fence=attempt.action_fence AND receipt.confirmed_at=consumed.consumed_at
       AND receipt.status IN ('stopped','already_stopped','already_completed','already_failed')))`
	}
	return validateRecoveryV17Checks(ctx, tx, []recoveryV17Check{
		{
			"sqlite.recovery_v28_effect_recovery_claim_invalid",
			`SELECT COUNT(*) FROM outbox action
LEFT JOIN effect_attempts attempt ON attempt.ref=action.recovery_effect_attempt_ref
LEFT JOIN effect_intents intent ON intent.ref=attempt.intent_ref
WHERE action.recovery_effect_attempt_ref IS NOT NULL AND
	(NOT ` + validKind + ` OR action.governance_version<>1
	  OR attempt.ref IS NULL OR attempt.action_ref<>action.ref
	  OR attempt.intent_ref<>action.effect_intent_ref
	  OR attempt.claim_lease_until IS NULL
	  OR attempt.action_fence>=action.fence
  OR EXISTS (SELECT 1 FROM effect_attempts current
             WHERE current.action_ref=action.ref
               AND current.action_fence=action.fence)
  OR EXISTS (SELECT 1 FROM budget_settlements settlement
             WHERE settlement.causal_attempt_ref=attempt.ref)
  OR NOT (
    (action.completed_at IS NULL AND action.quarantined_at IS NULL
     AND ` + pendingClaim + `
     AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt
                     WHERE receipt.action_ref=action.ref
                        OR receipt.intent_ref=action.effect_intent_ref)
     AND (SELECT COUNT(*) FROM effect_attempts peer
          WHERE peer.action_ref=action.ref AND peer.intent_ref=action.effect_intent_ref
           AND NOT EXISTS (SELECT 1 FROM budget_settlements settled
                           WHERE settled.causal_attempt_ref=peer.ref))=1)
    OR
	(action.kind='launch_agent'
	 AND (action.completed_at IS NOT NULL OR action.quarantined_at IS NOT NULL)
     AND EXISTS (SELECT 1 FROM action_consumption_receipts consumed
       LEFT JOIN effect_receipts receipt ON receipt.ref=consumed.effect_receipt_ref
       WHERE consumed.action_ref=action.ref AND consumed.fence=action.fence
        AND consumed.claim_token=action.claim_token AND consumed.worker_ref=action.claimed_by
        AND consumed.consumed_at=action.completed_at
        AND (consumed.effect_receipt_ref IS NULL
             OR (receipt.attempt_ref=action.recovery_effect_attempt_ref
		             AND receipt.action_fence=attempt.action_fence))))` + stopTerminal + `))`,
		},
		{
			"sqlite.recovery_v28_effect_receipt_attempt_invalid",
			`SELECT COUNT(*) FROM effect_receipts receipt
LEFT JOIN effect_attempts attempt ON attempt.ref=receipt.attempt_ref
LEFT JOIN effect_intents intent ON intent.ref=attempt.intent_ref
WHERE attempt.ref IS NULL OR receipt.action_ref<>attempt.action_ref
 OR receipt.action_fence<>attempt.action_fence
 OR receipt.confirmed_at<attempt.started_at
	 OR (attempt.claim_lease_until IS NOT NULL
	     AND receipt.confirmed_at>=attempt.claim_lease_until
	     AND intent.kind NOT IN ('agent_quiesce','agent_environment_preserve','agent_environment_close')
	     AND NOT (intent.kind='agent_stop' AND EXISTS (
	      SELECT 1 FROM outbox recovered WHERE recovered.ref=receipt.action_ref
	       AND recovered.recovery_effect_attempt_ref=attempt.ref
	       AND recovered.completed_at=receipt.confirmed_at
	       AND receipt.confirmed_at<recovered.claimed_until)))`,
		},
		{
			"sqlite.recovery_v28_effect_consumption_claim_invalid",
			`SELECT COUNT(*) FROM action_consumption_receipts consumed
JOIN outbox action ON action.ref=consumed.action_ref
JOIN effect_receipts receipt ON receipt.ref=consumed.effect_receipt_ref
WHERE NOT (
 (action.recovery_effect_attempt_ref IS NULL
  AND receipt.action_fence=consumed.fence
  AND receipt.confirmed_at=consumed.consumed_at)
 OR
 (consumed.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
  AND action.recovery_effect_attempt_ref IS NULL
  AND receipt.action_fence=consumed.fence
  AND receipt.confirmed_at<=consumed.consumed_at
  AND consumed.claim_token=action.claim_token
  AND consumed.worker_ref=action.claimed_by
  AND consumed.delivery_attempt=action.delivery_attempt
  AND consumed.consumed_at=action.completed_at
  AND consumed.outcome='completed'
  AND consumed.error_code=''
  AND EXISTS (
   SELECT 1 FROM effect_attempts attempt
   JOIN effect_intents intent ON intent.ref=attempt.intent_ref
   WHERE attempt.ref=receipt.attempt_ref
    AND attempt.action_ref=consumed.action_ref
    AND attempt.action_fence=consumed.fence
    AND attempt.goal_ref=consumed.goal_ref
    AND attempt.work_item_ref=consumed.work_item_ref
    AND attempt.execution_ref=consumed.execution_ref
    AND intent.kind=CASE consumed.kind
     WHEN 'quiesce_agent' THEN 'agent_quiesce'
     WHEN 'preserve_agent_environment' THEN 'agent_environment_preserve'
     WHEN 'close_agent_environment' THEN 'agent_environment_close' END))
 OR
 (consumed.kind='launch_agent'
  AND action.recovery_effect_attempt_ref=receipt.attempt_ref
  AND receipt.action_fence<consumed.fence
  AND NOT EXISTS (SELECT 1 FROM effect_attempts current
                  WHERE current.action_ref=consumed.action_ref
	                    AND current.action_fence=consumed.fence))
 OR
 (consumed.kind='stop_agent'
  AND action.recovery_effect_attempt_ref=receipt.attempt_ref
  AND receipt.action_fence<consumed.fence
  AND receipt.confirmed_at=consumed.consumed_at
  AND NOT EXISTS (SELECT 1 FROM effect_attempts current
                  WHERE current.action_ref=consumed.action_ref
                    AND current.action_fence=consumed.fence)))`,
		},
	})
}

func validateRecoveryV28Governance(ctx context.Context, tx *sql.Tx) error {
	checks := make([]recoveryV15Check, 0, len(recoveryV15Checks))
	for _, check := range recoveryV15Checks {
		switch check.code {
		case "sqlite.recovery_v15_effect_receipt_invalid", "sqlite.recovery_v15_effect_binding_invalid":
			continue
		default:
			checks = append(checks, check)
		}
	}
	checks = append(checks,
		recoveryV15Check{"sqlite.recovery_v15_effect_receipt_invalid", `
SELECT COUNT(*) FROM effect_receipts receipt LEFT JOIN effect_attempts attempt ON attempt.ref=receipt.attempt_ref
LEFT JOIN effect_intents intent ON intent.ref=receipt.intent_ref
LEFT JOIN outbox action ON action.ref=receipt.action_ref
WHERE attempt.ref IS NULL OR intent.ref IS NULL OR action.ref IS NULL
 OR receipt.intent_ref<>attempt.intent_ref OR receipt.intent_digest<>attempt.intent_digest
 OR receipt.approval_ref<>attempt.approval_ref OR receipt.project_ref<>attempt.project_ref
 OR receipt.goal_ref<>attempt.goal_ref OR receipt.work_item_ref<>attempt.work_item_ref
 OR receipt.execution_ref<>attempt.execution_ref OR receipt.plan_generation<>attempt.plan_generation
 OR receipt.app_spec_generation<>attempt.app_spec_generation OR receipt.spec_hash<>attempt.spec_hash
 OR receipt.actor_ref<>attempt.actor_ref OR receipt.action_ref<>attempt.action_ref
 OR receipt.action_fence<>attempt.action_fence OR receipt.idempotency_key<>attempt.idempotency_key
	 OR receipt.confirmed_at<attempt.started_at
	 OR (attempt.claim_lease_until IS NOT NULL AND receipt.confirmed_at>=attempt.claim_lease_until
	     AND intent.kind NOT IN ('agent_quiesce','agent_environment_preserve','agent_environment_close')
	     AND NOT (intent.kind='agent_stop' AND action.recovery_effect_attempt_ref=attempt.ref
	      AND action.completed_at=receipt.confirmed_at AND receipt.confirmed_at<action.claimed_until))
 OR (intent.kind='agent_launch' AND receipt.status<>'accepted')
 OR (intent.kind='agent_quiesce' AND receipt.status<>'quiesced')
 OR (intent.kind='agent_environment_preserve' AND receipt.status<>'preserved')
 OR (intent.kind='agent_environment_close' AND receipt.status<>'closed')
 OR (intent.kind='agent_stop' AND receipt.status NOT IN ('stopped','already_stopped','already_completed','already_failed'))
 OR (intent.kind='prepare_workspace' AND receipt.status<>'prepared')
 OR (intent.kind='commit_change' AND receipt.status<>'committed')
 OR (intent.kind='attest_test' AND receipt.status NOT IN ('attested_passed','attested_failed'))
 OR (intent.kind='integrate_change' AND receipt.status NOT IN ('integrated','conflicted','stale'))`},
		recoveryV15Check{"sqlite.recovery_v15_effect_binding_invalid", `
SELECT (SELECT COUNT(*) FROM executions execution LEFT JOIN effect_intents intent ON intent.ref=execution.effect_intent_ref
 LEFT JOIN budget_reservations reservation ON reservation.ref=execution.budget_reservation_ref
 LEFT JOIN effect_receipts receipt ON receipt.ref=execution.launch_receipt_ref
 WHERE (execution.governance_version=0 AND (reservation.ref IS NOT NULL OR intent.ref IS NOT NULL OR receipt.ref IS NOT NULL))
 OR (execution.governance_version=1 AND (intent.ref IS NULL OR reservation.ref IS NULL
  OR intent.execution_ref<>execution.ref OR reservation.execution_ref<>execution.ref
  OR reservation.effect_intent_ref<>intent.ref OR (receipt.ref IS NOT NULL AND receipt.intent_ref<>intent.ref)
  OR (execution.state IN ('running','succeeded','stopped','canceled') AND receipt.ref IS NULL))))
+(SELECT COUNT(*) FROM action_consumption_receipts consumed JOIN outbox action ON action.ref=consumed.action_ref
 LEFT JOIN effect_receipts receipt ON receipt.ref=consumed.effect_receipt_ref
 WHERE consumed.governance_version<>action.governance_version
 OR (consumed.consumed_at>=action.claimed_until
     AND consumed.kind NOT IN ('quiesce_agent','preserve_agent_environment','close_agent_environment'))
 OR (receipt.ref IS NOT NULL AND (receipt.action_ref<>consumed.action_ref OR NOT (
      (action.recovery_effect_attempt_ref IS NULL AND receipt.action_fence=consumed.fence
       AND ((consumed.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
             AND receipt.confirmed_at<=consumed.consumed_at)
            OR (consumed.kind NOT IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
                AND receipt.confirmed_at=consumed.consumed_at)))
      OR (consumed.kind='launch_agent' AND action.recovery_effect_attempt_ref=receipt.attempt_ref
       AND receipt.action_fence<consumed.fence
       AND NOT EXISTS(SELECT 1 FROM effect_attempts current
	          WHERE current.action_ref=consumed.action_ref AND current.action_fence=consumed.fence))
	  OR (consumed.kind='stop_agent' AND action.recovery_effect_attempt_ref=receipt.attempt_ref
	   AND receipt.action_fence<consumed.fence AND receipt.confirmed_at=consumed.consumed_at
	   AND NOT EXISTS(SELECT 1 FROM effect_attempts current
	      WHERE current.action_ref=consumed.action_ref AND current.action_fence=consumed.fence)))))
 OR (consumed.governance_version=1 AND consumed.kind='launch_agent' AND consumed.outcome='completed'
  AND consumed.error_code='' AND receipt.ref IS NULL)
 OR (consumed.governance_version=1 AND consumed.kind IN (
      'quiesce_agent','preserve_agent_environment','close_agent_environment',
      'prepare_workspace','commit_change','attest_test','integrate_change')
  AND consumed.outcome='completed' AND consumed.error_code='' AND receipt.ref IS NULL)
 OR (consumed.governance_version=1 AND consumed.kind='stop_agent' AND consumed.outcome='completed'
  AND consumed.error_code='' AND receipt.ref IS NULL AND EXISTS(SELECT 1 FROM effect_attempts attempt
      WHERE attempt.action_ref=consumed.action_ref AND attempt.action_fence=consumed.fence)))
+(SELECT COUNT(*) FROM effect_receipts receipt LEFT JOIN action_consumption_receipts consumed
 ON consumed.effect_receipt_ref=receipt.ref AND consumed.action_ref=receipt.action_ref
 WHERE consumed.action_ref IS NULL)`},
	)
	return validateRecoveryV17GovernanceWithChecks(ctx, tx, checks)
}
