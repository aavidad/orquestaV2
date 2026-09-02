-- B12: sujeto y autoridad explícitos para continuar exactamente un Launch ya
-- reservado cuyo contrato temporal original caducó. No crea otra acción,
-- intención, intento físico, ejecución lógica ni ruta Launch ordinaria.
CREATE TABLE agent_launch_expired_continuation_subjects (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 reconciliation_authority_ref TEXT NOT NULL UNIQUE
   REFERENCES agent_launch_reconciliation_authorities(ref) ON DELETE RESTRICT,
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,
 action_ref TEXT NOT NULL UNIQUE REFERENCES outbox(ref) ON DELETE RESTRICT,
 effect_intent_ref TEXT NOT NULL UNIQUE REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 effect_intent_digest TEXT NOT NULL CHECK(length(effect_intent_digest)=64 AND effect_intent_digest NOT GLOB '*[^0-9a-f]*'),
 effect_attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),
 action_fence INTEGER NOT NULL CHECK(action_fence>0),
 request_key_sha256 TEXT NOT NULL CHECK(length(request_key_sha256)=64 AND request_key_sha256 NOT GLOB '*[^0-9a-f]*'),
 original_request_sha256 TEXT NOT NULL CHECK(length(original_request_sha256)=64 AND original_request_sha256 NOT GLOB '*[^0-9a-f]*'),
 amv_launch_ref TEXT NOT NULL CHECK(amv_launch_ref GLOB 'launch:*' AND length(trim(amv_launch_ref))>7),
 amv_execution_ref TEXT NOT NULL CHECK(amv_execution_ref GLOB 'ejecucion:*' AND length(trim(amv_execution_ref))>10),
 amv_run_ref TEXT NOT NULL CHECK(amv_run_ref GLOB 'run:*' AND length(trim(amv_run_ref))>4),
 amv_fence INTEGER NOT NULL CHECK(amv_fence>0),
 amv_generation INTEGER NOT NULL CHECK(amv_generation>0),
 amv_cid INTEGER NOT NULL CHECK(amv_cid>=3),
 amv_identity_sha256 TEXT NOT NULL CHECK(length(amv_identity_sha256)=64 AND amv_identity_sha256 NOT GLOB '*[^0-9a-f]*'),
 source_digest TEXT NOT NULL CHECK(length(source_digest)=64 AND source_digest NOT GLOB '*[^0-9a-f]*'),
 profile_descriptor_bytes BLOB NOT NULL CHECK(length(profile_descriptor_bytes) BETWEEN 1 AND 65536),
 profile_descriptor_bytes_sha256 TEXT NOT NULL CHECK(length(profile_descriptor_bytes_sha256)=64 AND profile_descriptor_bytes_sha256 NOT GLOB '*[^0-9a-f]*'),
 plan_bytes BLOB NOT NULL CHECK(length(plan_bytes) BETWEEN 1 AND 65536),
 plan_bytes_sha256 TEXT NOT NULL CHECK(length(plan_bytes_sha256)=64 AND plan_bytes_sha256 NOT GLOB '*[^0-9a-f]*'),
 concession_bytes BLOB NOT NULL CHECK(length(concession_bytes) BETWEEN 1 AND 65536),
 concession_bytes_sha256 TEXT NOT NULL CHECK(length(concession_bytes_sha256)=64 AND concession_bytes_sha256 NOT GLOB '*[^0-9a-f]*'),
 manifest_bytes BLOB NOT NULL CHECK(length(manifest_bytes) BETWEEN 1 AND 131072),
 manifest_bytes_sha256 TEXT NOT NULL CHECK(length(manifest_bytes_sha256)=64 AND manifest_bytes_sha256 NOT GLOB '*[^0-9a-f]*'),
 manifest_sha256 TEXT NOT NULL UNIQUE CHECK(length(manifest_sha256)=64 AND manifest_sha256 NOT GLOB '*[^0-9a-f]*'),
 prepared_at INTEGER NOT NULL,
 UNIQUE(amv_launch_ref,amv_execution_ref,amv_run_ref,amv_fence,amv_generation,amv_cid),
 CHECK(action_fence=amv_fence),
 FOREIGN KEY(goal_ref,project_ref) REFERENCES goals(ref,project_ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT
) STRICT;

CREATE TABLE agent_launch_expired_continuation_authorities (
 authority_ref TEXT PRIMARY KEY CHECK(authority_ref GLOB 'continuation:*' AND length(trim(authority_ref))>13),
 subject_ref TEXT NOT NULL UNIQUE REFERENCES agent_launch_expired_continuation_subjects(ref) ON DELETE RESTRICT,
 reconciliation_attempt_ref TEXT NOT NULL UNIQUE
   REFERENCES agent_launch_reconciliation_attempts(ref) ON DELETE RESTRICT,
 manifest_sha256 TEXT NOT NULL REFERENCES agent_launch_expired_continuation_subjects(manifest_sha256) ON DELETE RESTRICT,
 authority_sha256 TEXT NOT NULL UNIQUE CHECK(length(authority_sha256)=64 AND authority_sha256 NOT GLOB '*[^0-9a-f]*'),
 authority_bytes BLOB NOT NULL CHECK(length(authority_bytes) BETWEEN 1 AND 131072),
 authority_bytes_sha256 TEXT NOT NULL CHECK(length(authority_bytes_sha256)=64 AND authority_bytes_sha256 NOT GLOB '*[^0-9a-f]*'),
 key_id TEXT NOT NULL CHECK(length(trim(key_id)) BETWEEN 1 AND 128),
 key_epoch INTEGER NOT NULL CHECK(key_epoch>0),
 trust_revision INTEGER NOT NULL CHECK(trust_revision>0),
 public_key BLOB NOT NULL CHECK(length(public_key)=32),
 signature BLOB NOT NULL CHECK(length(signature)=64),
 issued_unix_ms INTEGER NOT NULL CHECK(issued_unix_ms>0),
 expires_unix_ms INTEGER NOT NULL CHECK(expires_unix_ms>issued_unix_ms AND expires_unix_ms-issued_unix_ms<=300000),
 admitted_unix_ms INTEGER NOT NULL,
 CHECK(admitted_unix_ms>=issued_unix_ms AND admitted_unix_ms<expires_unix_ms)
) STRICT;

CREATE INDEX agent_launch_expired_continuation_authorities_subject_idx
 ON agent_launch_expired_continuation_authorities(subject_ref,issued_unix_ms,authority_ref);

CREATE TRIGGER agent_launch_expired_continuation_subjects_exact_replay
 BEFORE INSERT ON agent_launch_expired_continuation_subjects
 WHEN EXISTS (
  SELECT 1 FROM agent_launch_expired_continuation_subjects persisted
  WHERE persisted.ref IS NEW.ref
   AND persisted.reconciliation_authority_ref IS NEW.reconciliation_authority_ref
   AND persisted.project_ref IS NEW.project_ref AND persisted.goal_ref IS NEW.goal_ref
   AND persisted.work_item_ref IS NEW.work_item_ref AND persisted.execution_ref IS NEW.execution_ref
   AND persisted.action_ref IS NEW.action_ref AND persisted.effect_intent_ref IS NEW.effect_intent_ref
   AND persisted.effect_intent_digest IS NEW.effect_intent_digest
   AND persisted.effect_attempt_ref IS NEW.effect_attempt_ref
   AND persisted.plan_generation IS NEW.plan_generation
   AND persisted.work_item_generation IS NEW.work_item_generation
   AND persisted.action_fence IS NEW.action_fence
   AND persisted.request_key_sha256 IS NEW.request_key_sha256
   AND persisted.original_request_sha256 IS NEW.original_request_sha256
   AND persisted.amv_launch_ref IS NEW.amv_launch_ref
   AND persisted.amv_execution_ref IS NEW.amv_execution_ref
   AND persisted.amv_run_ref IS NEW.amv_run_ref AND persisted.amv_fence IS NEW.amv_fence
   AND persisted.amv_generation IS NEW.amv_generation AND persisted.amv_cid IS NEW.amv_cid
   AND persisted.amv_identity_sha256 IS NEW.amv_identity_sha256
   AND persisted.source_digest IS NEW.source_digest
   AND persisted.profile_descriptor_bytes IS NEW.profile_descriptor_bytes
   AND persisted.profile_descriptor_bytes_sha256 IS NEW.profile_descriptor_bytes_sha256
   AND persisted.plan_bytes IS NEW.plan_bytes AND persisted.plan_bytes_sha256 IS NEW.plan_bytes_sha256
   AND persisted.concession_bytes IS NEW.concession_bytes
   AND persisted.concession_bytes_sha256 IS NEW.concession_bytes_sha256
   AND persisted.manifest_bytes IS NEW.manifest_bytes
   AND persisted.manifest_bytes_sha256 IS NEW.manifest_bytes_sha256
   AND persisted.manifest_sha256 IS NEW.manifest_sha256
   AND persisted.prepared_at IS NEW.prepared_at
 )
 BEGIN SELECT RAISE(IGNORE); END;
CREATE TRIGGER agent_launch_expired_continuation_authorities_exact_replay
 BEFORE INSERT ON agent_launch_expired_continuation_authorities
 WHEN EXISTS (
  SELECT 1 FROM agent_launch_expired_continuation_authorities persisted
  WHERE persisted.authority_ref IS NEW.authority_ref AND persisted.subject_ref IS NEW.subject_ref
   AND persisted.reconciliation_attempt_ref IS NEW.reconciliation_attempt_ref
   AND persisted.manifest_sha256 IS NEW.manifest_sha256
   AND persisted.authority_sha256 IS NEW.authority_sha256
   AND persisted.authority_bytes IS NEW.authority_bytes
   AND persisted.authority_bytes_sha256 IS NEW.authority_bytes_sha256
   AND persisted.key_id IS NEW.key_id AND persisted.key_epoch IS NEW.key_epoch
   AND persisted.trust_revision IS NEW.trust_revision AND persisted.public_key IS NEW.public_key
   AND persisted.signature IS NEW.signature AND persisted.issued_unix_ms IS NEW.issued_unix_ms
   AND persisted.expires_unix_ms IS NEW.expires_unix_ms
   AND persisted.admitted_unix_ms IS NEW.admitted_unix_ms
 )
 BEGIN SELECT RAISE(IGNORE); END;

CREATE TRIGGER agent_launch_expired_continuation_subjects_immutable_update
 BEFORE UPDATE ON agent_launch_expired_continuation_subjects
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_expired_continuation_subject_immutable'); END;
CREATE TRIGGER agent_launch_expired_continuation_subjects_immutable_delete
 BEFORE DELETE ON agent_launch_expired_continuation_subjects
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_expired_continuation_subject_immutable'); END;
CREATE TRIGGER agent_launch_expired_continuation_authorities_immutable_update
 BEFORE UPDATE ON agent_launch_expired_continuation_authorities
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_expired_continuation_authority_immutable'); END;
CREATE TRIGGER agent_launch_expired_continuation_authorities_immutable_delete
 BEFORE DELETE ON agent_launch_expired_continuation_authorities
 BEGIN SELECT RAISE(ABORT,'sqlite.agent_launch_expired_continuation_authority_immutable'); END;
