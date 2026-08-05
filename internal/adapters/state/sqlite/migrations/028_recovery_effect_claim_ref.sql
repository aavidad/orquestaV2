-- B10.5d2: el outbox conserva la decisión durable de reconciliar un intento
-- físico histórico. El receipt sigue poseyendo action_fence; el comprobante
-- de consumo posee la cerca del claim actual.
ALTER TABLE outbox
 ADD COLUMN recovery_effect_attempt_ref TEXT
 REFERENCES effect_attempts(ref) ON DELETE RESTRICT;

-- La referencia solo puede fijarse durante un claim de launch y queda ligada
-- al único intento histórico pendiente. Nunca autoriza crear un segundo
-- intento en la cerca nueva.
CREATE TRIGGER outbox_recovery_effect_claim_guard
BEFORE UPDATE OF claim_token,claimed_by,claimed_until,delivery_attempt,fence,recovery_effect_attempt_ref ON outbox
WHEN (NEW.recovery_effect_attempt_ref IS NOT OLD.recovery_effect_attempt_ref AND NOT (
      NEW.claim_token IS NOT NULL AND NEW.claim_token IS NOT OLD.claim_token
      AND NEW.claimed_by IS NOT NULL AND NEW.claimed_until IS NOT NULL
      AND NEW.delivery_attempt=OLD.delivery_attempt+1 AND NEW.fence>OLD.fence))
 OR (NEW.recovery_effect_attempt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_attempts attempt
  JOIN effect_intents intent ON intent.ref=attempt.intent_ref
  WHERE attempt.ref=NEW.recovery_effect_attempt_ref
   AND NEW.kind='launch_agent' AND NEW.governance_version=1
   AND NEW.effect_intent_ref=attempt.intent_ref
   AND attempt.action_ref=NEW.ref AND attempt.action_fence<NEW.fence
   AND attempt.claim_lease_until IS NOT NULL
   AND intent.kind='agent_launch' AND intent.action_ref=NEW.ref
   AND NEW.claim_token IS NOT NULL AND NEW.claimed_by IS NOT NULL
   AND NEW.claimed_until IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt
                   WHERE receipt.action_ref=attempt.action_ref
                      OR receipt.intent_ref=attempt.intent_ref)
   AND NOT EXISTS (SELECT 1 FROM budget_settlements settlement
                   WHERE settlement.causal_attempt_ref=attempt.ref)
   AND NOT EXISTS (SELECT 1 FROM effect_attempts current
                   WHERE current.action_ref=attempt.action_ref
                     AND current.action_fence=NEW.fence)
   AND (SELECT COUNT(*) FROM effect_attempts peer
        WHERE peer.action_ref=attempt.action_ref
          AND peer.intent_ref=attempt.intent_ref
          AND NOT EXISTS (SELECT 1 FROM budget_settlements settled
                          WHERE settled.causal_attempt_ref=peer.ref))=1
 ))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_recovery_effect_claim_invalid'); END;

DROP TRIGGER effect_receipts_causal_guard;
CREATE TRIGGER effect_receipts_causal_guard
BEFORE INSERT ON effect_receipts
WHEN NOT EXISTS (
 SELECT 1 FROM effect_attempts attempt
 JOIN effect_intents intent ON intent.ref=attempt.intent_ref
 WHERE attempt.ref=NEW.attempt_ref AND attempt.intent_ref=NEW.intent_ref
  AND attempt.intent_digest=NEW.intent_digest AND attempt.approval_ref=NEW.approval_ref
  AND attempt.project_ref=NEW.project_ref AND attempt.goal_ref=NEW.goal_ref
  AND attempt.work_item_ref=NEW.work_item_ref AND attempt.execution_ref=NEW.execution_ref
  AND attempt.plan_generation=NEW.plan_generation
  AND attempt.app_spec_generation=NEW.app_spec_generation
  AND attempt.spec_hash=NEW.spec_hash AND attempt.actor_ref=NEW.actor_ref
  AND attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.action_fence
  AND attempt.idempotency_key=NEW.idempotency_key
  AND attempt.claim_lease_until IS NOT NULL
  AND NEW.confirmed_at>=attempt.started_at
  AND NEW.confirmed_at<attempt.claim_lease_until
  AND ((intent.kind='agent_launch' AND NEW.status='accepted')
    OR (intent.kind='agent_stop' AND NEW.status IN ('stopped','already_stopped','already_completed','already_failed'))
    OR (intent.kind='prepare_workspace' AND NEW.status='prepared')
    OR (intent.kind='commit_change' AND NEW.status='committed')
    OR (intent.kind='attest_test' AND NEW.status IN ('attested_passed','attested_failed'))
    OR (intent.kind='integrate_change' AND NEW.status IN ('integrated','conflicted','stale'))))
BEGIN SELECT RAISE(ABORT,'sqlite.effect_receipt_causal_invalid'); END;

DROP TRIGGER action_consumption_effect_receipt_guard;
CREATE TRIGGER action_consumption_effect_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NEW.governance_version=1 AND (
 (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt
  JOIN outbox action ON action.ref=NEW.action_ref
  WHERE receipt.ref=NEW.effect_receipt_ref AND receipt.action_ref=NEW.action_ref
   AND ((action.recovery_effect_attempt_ref IS NULL
         AND receipt.action_fence=NEW.fence
         AND receipt.confirmed_at=NEW.consumed_at)
     OR (NEW.kind='launch_agent'
         AND action.recovery_effect_attempt_ref=receipt.attempt_ref
         AND receipt.action_fence<NEW.fence
         AND NOT EXISTS (SELECT 1 FROM effect_attempts current
                         WHERE current.action_ref=NEW.action_ref
                           AND current.action_fence=NEW.fence))))
 )
 OR (NEW.kind IN ('launch_agent','prepare_workspace','commit_change','attest_test','integrate_change')
     AND NEW.outcome='completed' AND NEW.error_code='' AND NEW.effect_receipt_ref IS NULL)
 OR (NEW.kind='stop_agent' AND NEW.outcome='completed' AND NEW.error_code=''
     AND NEW.effect_receipt_ref IS NULL AND EXISTS (
      SELECT 1 FROM effect_attempts attempt
      WHERE attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.fence)))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_effect_receipt_invalid'); END;
