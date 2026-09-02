-- B12: autoridad explícita y diario separado para resolver un lanzamiento que
-- ya terminó en cuarentena. La acción y su receipt original siguen inmutables.
CREATE TABLE agent_launch_reconciliation_authorities (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 job_ref TEXT NOT NULL UNIQUE CHECK(length(trim(job_ref))>0),
 request_ref TEXT NOT NULL UNIQUE CHECK(length(trim(request_ref))>0),
 request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 authorization_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
 principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,
 action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT,
 effect_intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 effect_intent_digest TEXT NOT NULL CHECK(length(effect_intent_digest)=64 AND effect_intent_digest NOT GLOB '*[^0-9a-f]*'),
 effect_attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),
 action_fence INTEGER NOT NULL CHECK(action_fence>0),
 original_receipt_fingerprint TEXT NOT NULL CHECK(length(original_receipt_fingerprint)=64 AND original_receipt_fingerprint NOT GLOB '*[^0-9a-f]*'),
 authorized_at INTEGER NOT NULL,
 UNIQUE(action_ref,effect_attempt_ref,action_fence),
 FOREIGN KEY(goal_ref,project_ref) REFERENCES goals(ref,project_ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT
) STRICT;

CREATE TABLE agent_launch_reconciliation_jobs (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 authority_ref TEXT NOT NULL UNIQUE REFERENCES agent_launch_reconciliation_authorities(ref) ON DELETE RESTRICT,
 available_at INTEGER NOT NULL,claim_token TEXT UNIQUE,claimed_by TEXT,claimed_until INTEGER,
 delivery_attempt INTEGER NOT NULL DEFAULT 0 CHECK(delivery_attempt>=0),
 fence INTEGER NOT NULL DEFAULT 0 CHECK(fence>=0),
 state TEXT NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','completed','quarantined')),
 last_error_code TEXT NOT NULL DEFAULT '',updated_at INTEGER NOT NULL,
 CHECK((claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL) OR
       (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL
        AND delivery_attempt>0 AND fence>0)),
 CHECK((state='pending') OR (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL))
) STRICT;

CREATE INDEX agent_launch_reconciliation_jobs_claimable_idx
 ON agent_launch_reconciliation_jobs(state,available_at,claimed_until,ref);

CREATE TABLE agent_launch_reconciliation_attempts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 authority_ref TEXT NOT NULL REFERENCES agent_launch_reconciliation_authorities(ref) ON DELETE RESTRICT,
 original_effect_attempt_ref TEXT NOT NULL REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 job_fence INTEGER NOT NULL CHECK(job_fence>0),delivery_attempt INTEGER NOT NULL CHECK(delivery_attempt>0),
 claim_token TEXT NOT NULL UNIQUE,worker_ref TEXT NOT NULL CHECK(length(trim(worker_ref))>0),
 started_at INTEGER NOT NULL,claim_lease_until INTEGER NOT NULL CHECK(claim_lease_until>started_at),
 UNIQUE(authority_ref,job_fence)
) STRICT;

CREATE TABLE agent_launch_reconciliation_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 authority_ref TEXT NOT NULL UNIQUE REFERENCES agent_launch_reconciliation_authorities(ref) ON DELETE RESTRICT,
 reconciliation_attempt_ref TEXT REFERENCES agent_launch_reconciliation_attempts(ref) ON DELETE RESTRICT,
 outcome TEXT NOT NULL CHECK(outcome IN ('completed','quarantined')),
 error_code TEXT NOT NULL DEFAULT '',effect_receipt_ref TEXT REFERENCES effect_receipts(ref) ON DELETE RESTRICT,
 completed_at INTEGER NOT NULL,
 CHECK((outcome='completed' AND error_code='' AND effect_receipt_ref IS NOT NULL AND reconciliation_attempt_ref IS NOT NULL)
    OR (outcome='quarantined' AND length(trim(error_code))>0 AND effect_receipt_ref IS NULL))
) STRICT;

CREATE TRIGGER agent_launch_reconciliation_authorities_immutable_update
 BEFORE UPDATE ON agent_launch_reconciliation_authorities
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_authority_immutable'); END;
CREATE TRIGGER agent_launch_reconciliation_authorities_immutable_delete
 BEFORE DELETE ON agent_launch_reconciliation_authorities
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_authority_immutable'); END;
CREATE TRIGGER agent_launch_reconciliation_jobs_identity_immutable
 BEFORE UPDATE OF ref,authority_ref ON agent_launch_reconciliation_jobs
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_job_identity_immutable'); END;
CREATE TRIGGER agent_launch_reconciliation_jobs_immutable_delete
 BEFORE DELETE ON agent_launch_reconciliation_jobs
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_job_immutable'); END;
CREATE TRIGGER agent_launch_reconciliation_attempts_immutable_update
 BEFORE UPDATE ON agent_launch_reconciliation_attempts
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_attempt_immutable'); END;
CREATE TRIGGER agent_launch_reconciliation_attempts_immutable_delete
 BEFORE DELETE ON agent_launch_reconciliation_attempts
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_attempt_immutable'); END;
CREATE TRIGGER agent_launch_reconciliation_receipts_immutable_update
 BEFORE UPDATE ON agent_launch_reconciliation_receipts
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_receipt_immutable'); END;
CREATE TRIGGER agent_launch_reconciliation_receipts_immutable_delete
 BEFORE DELETE ON agent_launch_reconciliation_receipts
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_reconciliation_receipt_immutable'); END;

-- Un launch terminal puede confirmar el intento histórico fuera de su lease
-- únicamente mientras un job autorizado está reclamado y su llamada de
-- reconciliación ya quedó registrada. El resto del ledger conserva el gate.
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
       OR (intent.kind='agent_launch' AND EXISTS (
        SELECT 1 FROM agent_launch_reconciliation_authorities authority
        JOIN agent_launch_reconciliation_jobs job ON job.authority_ref=authority.ref
        JOIN agent_launch_reconciliation_attempts reconciliation_attempt
          ON reconciliation_attempt.authority_ref=authority.ref
         AND reconciliation_attempt.job_fence=job.fence
         AND reconciliation_attempt.claim_token=job.claim_token
         AND reconciliation_attempt.worker_ref=job.claimed_by
         AND reconciliation_attempt.claim_lease_until=job.claimed_until
        WHERE authority.action_ref=attempt.action_ref
         AND authority.effect_intent_ref=attempt.intent_ref
         AND authority.effect_intent_digest=attempt.intent_digest
         AND authority.effect_attempt_ref=attempt.ref
         AND authority.action_fence=attempt.action_fence
         AND job.state='pending' AND job.claim_token IS NOT NULL
         AND NOT EXISTS (SELECT 1 FROM agent_launch_reconciliation_receipts settled
                         WHERE settled.authority_ref=authority.ref))))
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
