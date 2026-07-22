-- V18 binds two independent reviewer executions to each exact author
-- ChangeSet + required-tests PASS round.
DROP TRIGGER executions_identity_immutable;
DROP TRIGGER executions_workspace_write_once;
DROP TRIGGER executions_provider_identity_write_once;
DROP TRIGGER executions_replacement_guard;
DROP INDEX executions_goal_idx;
DROP INDEX executions_one_active_per_work_item_idx;
PRAGMA legacy_alter_table = ON;
ALTER TABLE executions RENAME TO executions_v13;
CREATE TABLE executions (
 ref TEXT PRIMARY KEY, goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE, work_item_ref TEXT NOT NULL,
 attempt_no INTEGER NOT NULL CHECK(attempt_no>0), max_execution_attempts INTEGER NOT NULL CHECK(max_execution_attempts>0), replaces_execution_ref TEXT,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0), app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 repository_ref TEXT NOT NULL DEFAULT '', execution_workspace_ref TEXT NOT NULL DEFAULT '',
 state TEXT NOT NULL CHECK(state IN ('queued','dispatching','running','awaiting_commit','awaiting_attestation','awaiting_integration','succeeded','failed','canceled','stopped')),
 purpose TEXT NOT NULL CHECK(purpose IN ('work','author','primary_review','adversarial_review')),
 review_subject_digest TEXT NOT NULL DEFAULT '',
 artifact_media_type TEXT NOT NULL, idempotency_key TEXT NOT NULL UNIQUE, max_output_bytes INTEGER NOT NULL CHECK(max_output_bytes>0),
 provider_ref TEXT NOT NULL DEFAULT '', model_ref TEXT NOT NULL DEFAULT '', agent_ref TEXT NOT NULL DEFAULT '', external_ref TEXT NOT NULL DEFAULT '',
 governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)), budget_reservation_ref TEXT, effect_intent_ref TEXT, launch_receipt_ref TEXT,
 created_at INTEGER NOT NULL, deadline_at INTEGER, started_at INTEGER, provider_accepted_at INTEGER, last_observed_at INTEGER, provider_observed_at INTEGER, finished_at INTEGER,
 failure_code TEXT NOT NULL DEFAULT '', recipient_mailbox_retired INTEGER NOT NULL DEFAULT 0 CHECK(recipient_mailbox_retired IN (0,1)),
 UNIQUE(goal_ref,ref), UNIQUE(goal_ref,work_item_ref,ref), UNIQUE(goal_ref,work_item_ref,purpose,review_subject_digest,attempt_no),
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,replaces_execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 CHECK((attempt_no=1 AND replaces_execution_ref IS NULL) OR (attempt_no>1 AND replaces_execution_ref IS NOT NULL)),
 CHECK(attempt_no<=max_execution_attempts), CHECK((execution_workspace_ref='')=(repository_ref='')),
 CHECK((purpose IN ('work','author') AND review_subject_digest='') OR
       (purpose IN ('primary_review','adversarial_review') AND length(review_subject_digest)=71 AND substr(review_subject_digest,1,7)='sha256:'))
) STRICT;
INSERT INTO executions(
 ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,
 plan_generation,app_spec_generation,spec_hash,repository_ref,execution_workspace_ref,state,purpose,review_subject_digest,
 artifact_media_type,idempotency_key,max_output_bytes,provider_ref,model_ref,agent_ref,external_ref,
 governance_version,budget_reservation_ref,effect_intent_ref,launch_receipt_ref,created_at,deadline_at,started_at,
 provider_accepted_at,last_observed_at,provider_observed_at,finished_at,failure_code,recipient_mailbox_retired)
SELECT e.ref,e.goal_ref,e.work_item_ref,e.attempt_no,e.max_execution_attempts,e.replaces_execution_ref,
 e.plan_generation,e.app_spec_generation,e.spec_hash,e.repository_ref,e.execution_workspace_ref,e.state,
 CASE WHEN e.execution_workspace_ref<>'' AND EXISTS(
   SELECT 1 FROM work_item_required_tests required
   WHERE required.goal_ref=e.goal_ref AND required.work_item_ref=e.work_item_ref) THEN 'author' ELSE 'work' END,'',
 e.artifact_media_type,e.idempotency_key,e.max_output_bytes,e.provider_ref,e.model_ref,e.agent_ref,e.external_ref,
 e.governance_version,e.budget_reservation_ref,e.effect_intent_ref,e.launch_receipt_ref,e.created_at,e.deadline_at,e.started_at,
 e.provider_accepted_at,e.last_observed_at,e.provider_observed_at,e.finished_at,e.failure_code,e.recipient_mailbox_retired
FROM executions_v13 e;
DROP TABLE executions_v13;
PRAGMA legacy_alter_table = OFF;

CREATE INDEX executions_goal_idx ON executions(goal_ref,created_at,ref);
CREATE UNIQUE INDEX executions_one_active_author_per_work_item_idx ON executions(goal_ref,work_item_ref)
 WHERE purpose IN ('work','author') AND state IN ('queued','dispatching','running','awaiting_commit','awaiting_attestation','awaiting_integration');
CREATE UNIQUE INDEX executions_one_active_reviewer_round_idx ON executions(goal_ref,work_item_ref,purpose,review_subject_digest)
 WHERE purpose IN ('primary_review','adversarial_review') AND state IN ('queued','dispatching','running');
CREATE UNIQUE INDEX executions_distinct_review_process_idx ON executions(goal_ref,work_item_ref,external_ref)
 WHERE external_ref<>'' AND purpose IN ('author','primary_review','adversarial_review');
CREATE TRIGGER executions_identity_immutable
BEFORE UPDATE OF goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,plan_generation,
 app_spec_generation,spec_hash,purpose,review_subject_digest,artifact_media_type,idempotency_key,max_output_bytes,created_at ON executions
BEGIN SELECT RAISE(ABORT,'sqlite.execution_identity_immutable'); END;
CREATE TRIGGER executions_workspace_write_once
BEFORE UPDATE OF repository_ref,execution_workspace_ref ON executions
WHEN NOT ((NEW.repository_ref=OLD.repository_ref AND NEW.execution_workspace_ref=OLD.execution_workspace_ref)
 OR (OLD.repository_ref='' AND OLD.execution_workspace_ref='' AND NEW.repository_ref<>'' AND NEW.execution_workspace_ref<>''))
BEGIN SELECT RAISE(ABORT,'sqlite.execution_workspace_write_once'); END;
CREATE TRIGGER executions_provider_identity_write_once
BEFORE UPDATE OF provider_ref,model_ref,agent_ref,external_ref ON executions
WHEN NOT ((NEW.provider_ref=OLD.provider_ref AND NEW.model_ref=OLD.model_ref AND NEW.agent_ref=OLD.agent_ref AND NEW.external_ref=OLD.external_ref)
 OR (OLD.state='dispatching' AND NEW.state='running' AND OLD.provider_ref='' AND OLD.model_ref='' AND OLD.agent_ref='' AND OLD.external_ref=''
     AND length(trim(NEW.provider_ref))>0 AND length(trim(NEW.model_ref))>0 AND length(trim(NEW.agent_ref))>0 AND length(trim(NEW.external_ref))>0))
BEGIN SELECT RAISE(ABORT,'sqlite.execution_provider_identity_write_once'); END;
CREATE TRIGGER executions_replacement_guard
BEFORE INSERT ON executions WHEN NEW.attempt_no>1 AND NOT EXISTS (
 SELECT 1 FROM executions previous WHERE previous.goal_ref=NEW.goal_ref AND previous.work_item_ref=NEW.work_item_ref
  AND previous.ref=NEW.replaces_execution_ref AND previous.attempt_no+1=NEW.attempt_no
  AND previous.max_execution_attempts=NEW.max_execution_attempts AND previous.plan_generation=NEW.plan_generation
  AND previous.app_spec_generation=NEW.app_spec_generation AND previous.spec_hash=NEW.spec_hash
  AND previous.purpose=NEW.purpose AND previous.review_subject_digest=NEW.review_subject_digest
  AND previous.state IN ('failed','stopped') AND previous.finished_at IS NOT NULL AND NEW.created_at>=previous.finished_at)
BEGIN SELECT RAISE(ABORT,'sqlite.execution_replacement_invalid'); END;

ALTER TABLE outbox ADD COLUMN review_gate_digest TEXT NOT NULL DEFAULT '';
DROP INDEX outbox_one_active_per_item_generation_idx;
CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx
 ON outbox(goal_ref,work_item_ref,execution_ref,plan_generation,work_item_generation)
 WHERE kind IN ('launch_agent','observe_agent','prepare_workspace','commit_change','attest_test','integrate_change')
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
DROP TRIGGER outbox_integration_admission_immutable;
CREATE TRIGGER outbox_integration_admission_immutable
BEFORE UPDATE OF change_ref,expected_target_oid,admission_request_ref,admission_request_fingerprint,review_gate_digest ON outbox
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_integration_admission_immutable'); END;

DROP TRIGGER artifact_occurrences_causal_guard;
DROP TRIGGER artifact_occurrences_immutable_update;
DROP TRIGGER artifact_occurrences_immutable_delete;
DROP INDEX artifact_occurrences_goal_idx;
DROP INDEX artifact_occurrences_artifact_idx;
PRAGMA legacy_alter_table = ON;
ALTER TABLE artifact_occurrences RENAME TO artifact_occurrences_v13;
CREATE TABLE artifact_occurrences (
 occurrence_ref TEXT PRIMARY KEY CHECK(length(trim(occurrence_ref))>0),
 kind TEXT NOT NULL CHECK(kind IN ('agent_output','test_subject_manifest','test_attestation_report','review_assessment','review_diagnostic')),
 goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,artifact_ref TEXT NOT NULL,
 execution_attempt INTEGER NOT NULL CHECK(execution_attempt>0),plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),created_at INTEGER NOT NULL,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT
) STRICT;
INSERT INTO artifact_occurrences SELECT * FROM artifact_occurrences_v13;
DROP TABLE artifact_occurrences_v13;
PRAGMA legacy_alter_table = OFF;
CREATE INDEX artifact_occurrences_goal_idx ON artifact_occurrences(goal_ref,created_at,occurrence_ref);
CREATE INDEX artifact_occurrences_artifact_idx ON artifact_occurrences(goal_ref,artifact_ref,created_at,occurrence_ref);
CREATE TRIGGER artifact_occurrences_causal_guard BEFORE INSERT ON artifact_occurrences WHEN NOT EXISTS (
 SELECT 1 FROM executions execution JOIN artifacts artifact ON artifact.goal_ref=NEW.goal_ref AND artifact.ref=NEW.artifact_ref
 WHERE execution.goal_ref=NEW.goal_ref AND execution.work_item_ref=NEW.work_item_ref AND execution.ref=NEW.execution_ref
  AND execution.attempt_no=NEW.execution_attempt AND execution.plan_generation=NEW.plan_generation
  AND execution.app_spec_generation=NEW.app_spec_generation AND execution.spec_hash=NEW.spec_hash)
BEGIN SELECT RAISE(ABORT,'sqlite.artifact_occurrence_causal_invalid'); END;
CREATE TRIGGER artifact_occurrences_immutable_update BEFORE UPDATE ON artifact_occurrences BEGIN SELECT RAISE(ABORT,'sqlite.artifact_occurrence_immutable'); END;
CREATE TRIGGER artifact_occurrences_immutable_delete BEFORE DELETE ON artifact_occurrences BEGIN SELECT RAISE(ABORT,'sqlite.artifact_occurrence_immutable'); END;

CREATE TABLE review_records (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,change_set_ref TEXT NOT NULL,
 subject_digest TEXT NOT NULL CHECK(length(subject_digest)=71 AND substr(subject_digest,1,7)='sha256:'),
 role TEXT NOT NULL CHECK(role IN ('primary','adversarial')),verdict TEXT NOT NULL CHECK(verdict IN ('approve','changes_requested')),
 reviewer_execution_ref TEXT NOT NULL,reviewer_execution_attempt INTEGER NOT NULL CHECK(reviewer_execution_attempt>0),
 launch_receipt_ref TEXT NOT NULL CHECK(length(trim(launch_receipt_ref))>0),principal_ref TEXT NOT NULL,agent_ref TEXT NOT NULL,
 external_ref TEXT NOT NULL CHECK(length(trim(external_ref))>0),
 assessment_artifact_ref TEXT NOT NULL,assessment_digest TEXT NOT NULL CHECK(length(trim(assessment_digest))>0),recorded_at INTEGER NOT NULL,
 UNIQUE(goal_ref,work_item_ref,subject_digest,role),UNIQUE(reviewer_execution_ref),UNIQUE(launch_receipt_ref),
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(change_set_ref) REFERENCES change_sets(ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,reviewer_execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,assessment_artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(principal_ref) REFERENCES principals(ref) ON DELETE RESTRICT
) STRICT;
CREATE INDEX review_records_goal_idx ON review_records(goal_ref,recorded_at,ref);
CREATE TRIGGER review_records_causal_guard BEFORE INSERT ON review_records WHEN NOT EXISTS (
 SELECT 1 FROM executions execution JOIN change_sets change_set ON change_set.ref=NEW.change_set_ref
 JOIN artifact_occurrences occurrence ON occurrence.goal_ref=NEW.goal_ref AND occurrence.artifact_ref=NEW.assessment_artifact_ref
 JOIN artifacts artifact ON artifact.goal_ref=NEW.goal_ref AND artifact.ref=NEW.assessment_artifact_ref
 JOIN effect_intents intent ON intent.ref=execution.effect_intent_ref
 JOIN work_item_authorities authority ON authority.goal_ref=NEW.goal_ref AND authority.work_item_ref=NEW.work_item_ref
 JOIN attestations test_pass ON test_pass.goal_ref=NEW.goal_ref AND test_pass.work_item_ref=NEW.work_item_ref
  AND test_pass.execution_ref=change_set.execution_ref AND test_pass.change_set_ref=change_set.ref
  AND test_pass.kind='required_tests' AND test_pass.verdict='passed'
 WHERE execution.goal_ref=NEW.goal_ref AND execution.work_item_ref=NEW.work_item_ref AND execution.ref=NEW.reviewer_execution_ref
  AND NEW.ref='review:' || execution.ref
  AND execution.attempt_no=NEW.reviewer_execution_attempt AND execution.review_subject_digest=NEW.subject_digest
  AND ((NEW.role='primary' AND execution.purpose='primary_review') OR (NEW.role='adversarial' AND execution.purpose='adversarial_review'))
  AND execution.launch_receipt_ref=NEW.launch_receipt_ref AND execution.agent_ref=NEW.agent_ref
  AND execution.external_ref=NEW.external_ref
  AND execution.state='succeeded' AND NEW.recorded_at=execution.finished_at
  AND intent.action_kind='launch_agent' AND intent.kind='agent_launch'
  AND intent.project_ref=(SELECT project_ref FROM goals WHERE ref=NEW.goal_ref)
  AND intent.goal_ref=NEW.goal_ref AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=execution.ref
  AND intent.plan_generation=execution.plan_generation AND intent.app_spec_generation=execution.app_spec_generation
  AND intent.spec_hash=execution.spec_hash AND intent.proposed_by_ref=NEW.principal_ref
  AND authority.principal_ref=NEW.principal_ref AND authority.permission=intent.permission
  AND authority.authorization_receipt_ref=intent.authority_receipt_ref
  AND occurrence.kind='review_assessment' AND occurrence.execution_ref=execution.ref
  AND occurrence.work_item_ref=NEW.work_item_ref AND occurrence.execution_attempt=execution.attempt_no
  AND occurrence.plan_generation=execution.plan_generation AND occurrence.work_item_generation=test_pass.work_item_generation
  AND occurrence.app_spec_generation=execution.app_spec_generation AND occurrence.spec_hash=execution.spec_hash
  AND occurrence.created_at=NEW.recorded_at AND artifact.digest=NEW.assessment_digest
  AND change_set.goal_ref=NEW.goal_ref AND change_set.work_item_ref=NEW.work_item_ref)
BEGIN SELECT RAISE(ABORT,'sqlite.review_record_causal_invalid'); END;
CREATE TRIGGER review_records_immutable_update BEFORE UPDATE ON review_records BEGIN SELECT RAISE(ABORT,'sqlite.review_record_immutable'); END;
CREATE TRIGGER review_records_immutable_delete BEFORE DELETE ON review_records BEGIN SELECT RAISE(ABORT,'sqlite.review_record_immutable'); END;
