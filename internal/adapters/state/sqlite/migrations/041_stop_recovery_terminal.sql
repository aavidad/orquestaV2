-- STOP-RECOVERY-TERMINAL-07: un Stop recuperado puede confirmar el intento
-- fisico historico durante la lease de recovery. La accion actual solo cerca
-- la observacion y el commit; no existe un EffectAttempt en su nueva fence.
DROP TRIGGER effect_receipts_causal_guard;
CREATE TRIGGER effect_receipts_causal_guard BEFORE INSERT ON effect_receipts
WHEN NOT EXISTS (
 SELECT 1 FROM effect_attempts attempt JOIN effect_intents intent ON intent.ref=attempt.intent_ref
 WHERE attempt.ref=NEW.attempt_ref AND attempt.intent_ref=NEW.intent_ref
  AND attempt.intent_digest=NEW.intent_digest AND attempt.approval_ref=NEW.approval_ref
  AND attempt.project_ref=NEW.project_ref AND attempt.goal_ref=NEW.goal_ref
  AND attempt.work_item_ref=NEW.work_item_ref AND attempt.execution_ref=NEW.execution_ref
  AND attempt.plan_generation=NEW.plan_generation AND attempt.app_spec_generation=NEW.app_spec_generation
  AND attempt.spec_hash=NEW.spec_hash AND attempt.actor_ref=NEW.actor_ref
  AND attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.action_fence
  AND attempt.idempotency_key=NEW.idempotency_key AND attempt.claim_lease_until IS NOT NULL
  AND NEW.confirmed_at>=attempt.started_at
  AND (intent.kind IN ('agent_quiesce','agent_environment_preserve','agent_environment_close')
       OR NEW.confirmed_at<attempt.claim_lease_until
       OR (intent.kind='agent_stop' AND EXISTS (
        SELECT 1 FROM outbox action WHERE action.ref=NEW.action_ref
         AND action.kind='stop_agent' AND action.recovery_effect_attempt_ref=attempt.ref
         AND action.fence>attempt.action_fence AND action.completed_at=NEW.confirmed_at
         AND NEW.confirmed_at<action.claimed_until AND action.claim_token IS NOT NULL
         AND action.claimed_by IS NOT NULL AND NOT EXISTS (
          SELECT 1 FROM effect_attempts current
          WHERE current.action_ref=action.ref AND current.action_fence=action.fence))))
  AND ((intent.kind='agent_launch' AND NEW.status='accepted')
    OR (intent.kind='agent_quiesce' AND NEW.status='quiesced')
    OR (intent.kind='agent_environment_preserve' AND NEW.status='preserved')
    OR (intent.kind='agent_environment_close' AND NEW.status='closed')
    OR (intent.kind='agent_stop' AND NEW.status IN ('stopped','already_stopped','already_completed','already_failed'))
    OR (intent.kind='prepare_workspace' AND NEW.status='prepared')
    OR (intent.kind='commit_change' AND NEW.status='committed')
    OR (intent.kind='attest_test' AND NEW.status IN ('attested_passed','attested_failed'))
    OR (intent.kind='integrate_change' AND NEW.status IN ('integrated','conflicted','stale')))
  AND (intent.kind NOT IN ('agent_quiesce','agent_environment_preserve','agent_environment_close') OR EXISTS(
   SELECT 1 FROM agent_environment_lifecycles lifecycle
   WHERE lifecycle.execution_ref=attempt.execution_ref AND lifecycle.claim_action_ref=attempt.action_ref
    AND lifecycle.attempt_ref=attempt.ref AND lifecycle.claim_fence=attempt.action_fence)))
BEGIN SELECT RAISE(ABORT,'sqlite.effect_receipt_causal_invalid'); END;

DROP TRIGGER action_consumption_effect_receipt_guard;
CREATE TRIGGER action_consumption_effect_receipt_guard BEFORE INSERT ON action_consumption_receipts
WHEN NEW.governance_version=1 AND (
 (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt JOIN outbox action ON action.ref=NEW.action_ref
  WHERE receipt.ref=NEW.effect_receipt_ref AND receipt.action_ref=NEW.action_ref
   AND ((NEW.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
         AND receipt.action_fence=NEW.fence AND receipt.confirmed_at<=NEW.consumed_at
         AND EXISTS(SELECT 1 FROM agent_environment_lifecycles lifecycle
                    WHERE lifecycle.execution_ref=NEW.execution_ref
                     AND lifecycle.attempt_ref=receipt.attempt_ref
                     AND lifecycle.claim_token=NEW.claim_token))
    OR (action.recovery_effect_attempt_ref IS NULL AND receipt.action_fence=NEW.fence
        AND receipt.confirmed_at=NEW.consumed_at)
    OR (NEW.kind='launch_agent' AND action.recovery_effect_attempt_ref=receipt.attempt_ref
        AND receipt.action_fence<NEW.fence AND NOT EXISTS (
         SELECT 1 FROM effect_attempts current
         WHERE current.action_ref=NEW.action_ref AND current.action_fence=NEW.fence))
    OR (NEW.kind='stop_agent' AND action.recovery_effect_attempt_ref=receipt.attempt_ref
        AND receipt.action_fence<NEW.fence AND receipt.confirmed_at=NEW.consumed_at
        AND NOT EXISTS (SELECT 1 FROM effect_attempts current
         WHERE current.action_ref=NEW.action_ref AND current.action_fence=NEW.fence)))))
 OR (NEW.kind IN ('launch_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
                  'prepare_workspace','commit_change','attest_test','integrate_change')
     AND NEW.outcome='completed' AND NEW.error_code='' AND NEW.effect_receipt_ref IS NULL)
 OR (NEW.kind='stop_agent' AND NEW.outcome='completed' AND NEW.error_code=''
     AND NEW.effect_receipt_ref IS NULL AND (
      EXISTS (SELECT 1 FROM effect_attempts attempt
              WHERE attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.fence)
      OR EXISTS (SELECT 1 FROM outbox action WHERE action.ref=NEW.action_ref
                 AND action.recovery_effect_attempt_ref IS NOT NULL))))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_effect_receipt_invalid'); END;
