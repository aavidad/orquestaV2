-- B11: prueba neutral e inmutable de que un intento fisico no se aplico.
-- No es receipt ni liquidacion presupuestaria; solo autoriza el reintento del
-- mismo efecto despues de que application libere atomicamente su claim.
CREATE TABLE effect_non_application_evidence (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 intent_digest TEXT NOT NULL CHECK(length(intent_digest)=64 AND intent_digest NOT GLOB '*[^0-9a-f]*'),
 approval_ref TEXT NOT NULL REFERENCES effect_approvals(ref) ON DELETE RESTRICT,
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 actor_ref TEXT NOT NULL CHECK(length(trim(actor_ref))>0),
 action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT,action_fence INTEGER NOT NULL CHECK(action_fence>0),
 idempotency_key TEXT NOT NULL CHECK(length(trim(idempotency_key))>0),
 outcome TEXT NOT NULL CHECK(outcome='definitely_not_applied'),observed_at INTEGER NOT NULL
) STRICT;

CREATE TRIGGER effect_non_application_evidence_insert_guard
BEFORE INSERT ON effect_non_application_evidence
WHEN NOT EXISTS (
 SELECT 1 FROM effect_attempts attempt
 JOIN effect_intents intent ON intent.ref=attempt.intent_ref
 WHERE attempt.ref=NEW.attempt_ref
   AND intent.kind='agent_stop'
   AND NEW.ref='effect-attempt-outcome:' || attempt.ref || ':definitely-not-applied'
   AND NEW.intent_ref=attempt.intent_ref AND NEW.intent_digest=attempt.intent_digest AND NEW.approval_ref=attempt.approval_ref
   AND NEW.project_ref=attempt.project_ref AND NEW.goal_ref=attempt.goal_ref AND NEW.work_item_ref=attempt.work_item_ref AND NEW.execution_ref=attempt.execution_ref
   AND NEW.plan_generation=attempt.plan_generation AND NEW.app_spec_generation=attempt.app_spec_generation
   AND NEW.spec_hash=attempt.spec_hash AND NEW.actor_ref=attempt.actor_ref
   AND NEW.action_ref=attempt.action_ref AND NEW.action_fence=attempt.action_fence AND NEW.idempotency_key=attempt.idempotency_key
   AND NEW.observed_at>=attempt.started_at AND NEW.observed_at<=attempt.claim_lease_until
   AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
   AND NOT EXISTS (SELECT 1 FROM budget_settlements settlement WHERE settlement.causal_attempt_ref=attempt.ref)
)
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_non_application_evidence_invalid'); END;

CREATE TRIGGER effect_non_application_evidence_immutable_update
BEFORE UPDATE ON effect_non_application_evidence
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_non_application_evidence_immutable'); END;

CREATE TRIGGER effect_non_application_evidence_immutable_delete
BEFORE DELETE ON effect_non_application_evidence
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_non_application_evidence_immutable'); END;
