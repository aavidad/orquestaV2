-- B11: amplía la cerca durable de recovery sin reescribir la migración 037.
-- Stop solo puede recuperar el único intento ambiguo exacto; una prueba
-- definitely_not_applied habilita el reintento ordinario sin esta referencia.
DROP TRIGGER outbox_recovery_effect_claim_guard;

CREATE TRIGGER outbox_recovery_effect_claim_guard
BEFORE UPDATE OF claim_token,claimed_by,claimed_until,delivery_attempt,fence,recovery_effect_attempt_ref ON outbox
WHEN (NEW.recovery_effect_attempt_ref IS NOT OLD.recovery_effect_attempt_ref AND NOT (
      NEW.recovery_effect_attempt_ref IS NOT NULL AND NEW.claim_token IS NOT NULL
      AND NEW.claim_token IS NOT OLD.claim_token AND NEW.claimed_by IS NOT NULL
      AND NEW.claimed_until IS NOT NULL AND NEW.delivery_attempt=OLD.delivery_attempt+1 AND NEW.fence>OLD.fence))
 OR (NEW.recovery_effect_attempt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_attempts attempt JOIN effect_intents intent ON intent.ref=attempt.intent_ref
  WHERE attempt.ref=NEW.recovery_effect_attempt_ref AND NEW.governance_version=1
   AND ((NEW.kind='launch_agent' AND intent.kind='agent_launch')
     OR (NEW.kind='stop_agent' AND intent.kind='agent_stop'))
   AND NEW.effect_intent_ref=attempt.intent_ref AND attempt.action_ref=NEW.ref AND attempt.action_fence<NEW.fence
   AND attempt.claim_lease_until IS NOT NULL AND intent.action_ref=NEW.ref
   AND NEW.completed_at IS NULL AND NEW.retired_at IS NULL AND NEW.quarantined_at IS NULL
   AND ((NEW.claim_token IS NOT NULL AND NEW.claimed_by IS NOT NULL AND NEW.claimed_until IS NOT NULL) OR
    (NEW.claim_token IS NULL AND NEW.claimed_by IS NULL AND NEW.claimed_until IS NULL
     AND OLD.claim_token IS NOT NULL AND OLD.claimed_by IS NOT NULL AND OLD.claimed_until IS NOT NULL
     AND NEW.recovery_effect_attempt_ref IS OLD.recovery_effect_attempt_ref
     AND NEW.delivery_attempt=OLD.delivery_attempt AND NEW.fence=OLD.fence
     AND NEW.available_at>=OLD.available_at AND length(trim(NEW.last_error_code))>0))
   AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt
                   WHERE receipt.action_ref=attempt.action_ref OR receipt.intent_ref=attempt.intent_ref)
   AND NOT EXISTS (SELECT 1 FROM budget_settlements settlement WHERE settlement.causal_attempt_ref=attempt.ref)
   AND NOT EXISTS (SELECT 1 FROM effect_attempts current
                   WHERE current.action_ref=attempt.action_ref AND current.action_fence=NEW.fence)
   AND ((NEW.kind='launch_agent' AND (SELECT COUNT(*) FROM effect_attempts peer
          WHERE peer.action_ref=attempt.action_ref AND peer.intent_ref=attempt.intent_ref
          AND NOT EXISTS (SELECT 1 FROM budget_settlements settled
                          WHERE settled.causal_attempt_ref=peer.ref))=1)
     OR (NEW.kind='stop_agent' AND (SELECT COUNT(*) FROM effect_attempts peer
          WHERE peer.action_ref=attempt.action_ref AND peer.intent_ref=attempt.intent_ref
          AND NOT EXISTS (SELECT 1 FROM effect_non_application_evidence evidence
                          WHERE evidence.attempt_ref=peer.ref AND evidence.outcome='definitely_not_applied'))=1)))
)
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_recovery_effect_claim_invalid'); END;
