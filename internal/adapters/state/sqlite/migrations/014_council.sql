-- V19 adds Council facts to the existing Goal authority. No policy is
-- backfilled: empty survives only for terminal history and read-only items.
PRAGMA legacy_alter_table=ON;
DROP TRIGGER authorization_receipts_immutable_update;
DROP TRIGGER authorization_receipts_immutable_delete;
DROP INDEX authorization_receipts_project_idx;
DROP INDEX authorization_receipts_mailbox_scope_idx;
ALTER TABLE authorization_receipts RENAME TO authorization_receipts_v14;
CREATE TABLE authorization_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),
 request_fingerprint TEXT NOT NULL CHECK(length(trim(request_fingerprint))>0),principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
 project_ref TEXT NOT NULL CHECK(length(trim(project_ref))>0),
 permission TEXT NOT NULL CHECK(permission IN ('project.hierarchy.manage','project.membership.manage','goals.create','goals.amend','goals.get','goals.list','goals.direct','budgets.manage','effects.approve','changes.integrate','council.skip','artifacts.read','project.status')),
 resource_ref TEXT NOT NULL CHECK(length(trim(resource_ref))>0),requested_at INTEGER NOT NULL,outcome TEXT NOT NULL CHECK(outcome IN ('allowed','denied')),
 role TEXT NOT NULL DEFAULT '' CHECK(role IN ('','platform_admin','project_owner','project_admin','contributor','reviewer','operator','viewer')),
 membership_revision INTEGER NOT NULL CHECK(membership_revision>=0),reason_code TEXT NOT NULL CHECK(length(trim(reason_code))>0),
 decided_at INTEGER NOT NULL,recorded_at INTEGER NOT NULL,UNIQUE(principal_ref,request_ref),CHECK(decided_at>=requested_at),CHECK(recorded_at>=decided_at),
 CHECK((outcome='allowed' AND role<>'' AND (role='platform_admin' OR membership_revision>0)) OR outcome='denied')
) STRICT;
INSERT INTO authorization_receipts SELECT * FROM authorization_receipts_v14;
DROP TABLE authorization_receipts_v14;
PRAGMA legacy_alter_table=OFF;
CREATE INDEX authorization_receipts_project_idx ON authorization_receipts(project_ref,recorded_at,ref);
CREATE UNIQUE INDEX authorization_receipts_mailbox_scope_idx ON authorization_receipts(ref,principal_ref,project_ref);
CREATE TRIGGER authorization_receipts_immutable_update BEFORE UPDATE ON authorization_receipts BEGIN SELECT RAISE(ABORT,'sqlite.authorization_receipt_immutable'); END;
CREATE TRIGGER authorization_receipts_immutable_delete BEFORE DELETE ON authorization_receipts BEGIN SELECT RAISE(ABORT,'sqlite.authorization_receipt_immutable'); END;

ALTER TABLE work_items ADD COLUMN council_policy TEXT NOT NULL DEFAULT ''
 CHECK(council_policy IN ('','auto','required','skip_by_operator'));
CREATE TRIGGER work_items_council_policy_immutable
BEFORE UPDATE OF council_policy ON work_items
WHEN NEW.council_policy<>OLD.council_policy
BEGIN SELECT RAISE(ABORT,'sqlite.work_item_council_policy_immutable'); END;

DROP TRIGGER executions_identity_immutable;
DROP TRIGGER executions_workspace_write_once;
DROP TRIGGER executions_provider_identity_write_once;
DROP TRIGGER executions_replacement_guard;
DROP INDEX executions_goal_idx;
DROP INDEX executions_one_active_author_per_work_item_idx;
DROP INDEX executions_one_active_reviewer_round_idx;
DROP INDEX executions_distinct_review_process_idx;
PRAGMA legacy_alter_table=ON;
ALTER TABLE executions RENAME TO executions_v14;
CREATE TABLE executions (
 ref TEXT PRIMARY KEY,goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,work_item_ref TEXT NOT NULL,
 attempt_no INTEGER NOT NULL CHECK(attempt_no>0),max_execution_attempts INTEGER NOT NULL CHECK(max_execution_attempts>0),replaces_execution_ref TEXT,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 repository_ref TEXT NOT NULL DEFAULT '',execution_workspace_ref TEXT NOT NULL DEFAULT '',
 state TEXT NOT NULL CHECK(state IN ('queued','dispatching','running','awaiting_commit','awaiting_attestation','awaiting_integration','succeeded','failed','canceled','stopped')),
 purpose TEXT NOT NULL CHECK(purpose IN ('work','author','primary_review','adversarial_review','council_proposer','council_critic','council_arbiter')),
 review_subject_digest TEXT NOT NULL DEFAULT '',council_subject_digest TEXT NOT NULL DEFAULT '',
 artifact_media_type TEXT NOT NULL,idempotency_key TEXT NOT NULL UNIQUE,max_output_bytes INTEGER NOT NULL CHECK(max_output_bytes>0),
 provider_ref TEXT NOT NULL DEFAULT '',model_ref TEXT NOT NULL DEFAULT '',agent_ref TEXT NOT NULL DEFAULT '',external_ref TEXT NOT NULL DEFAULT '',
 governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)),budget_reservation_ref TEXT,effect_intent_ref TEXT,launch_receipt_ref TEXT,
 created_at INTEGER NOT NULL,deadline_at INTEGER,started_at INTEGER,provider_accepted_at INTEGER,last_observed_at INTEGER,provider_observed_at INTEGER,finished_at INTEGER,
 failure_code TEXT NOT NULL DEFAULT '',recipient_mailbox_retired INTEGER NOT NULL DEFAULT 0 CHECK(recipient_mailbox_retired IN (0,1)),
 UNIQUE(goal_ref,ref),UNIQUE(goal_ref,work_item_ref,ref),
 UNIQUE(goal_ref,work_item_ref,purpose,review_subject_digest,council_subject_digest,attempt_no),
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,replaces_execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 CHECK((attempt_no=1 AND replaces_execution_ref IS NULL) OR (attempt_no>1 AND replaces_execution_ref IS NOT NULL)),
 CHECK(attempt_no<=max_execution_attempts),CHECK((execution_workspace_ref='')=(repository_ref='')),
 CHECK((purpose IN ('work','author') AND review_subject_digest='' AND council_subject_digest='') OR
  (purpose IN ('primary_review','adversarial_review') AND length(review_subject_digest)=71 AND substr(review_subject_digest,1,7)='sha256:' AND council_subject_digest='') OR
  (purpose IN ('council_proposer','council_critic','council_arbiter') AND review_subject_digest='' AND length(council_subject_digest)=71 AND substr(council_subject_digest,1,7)='sha256:'))
) STRICT;
INSERT INTO executions(
 ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,plan_generation,app_spec_generation,spec_hash,
 repository_ref,execution_workspace_ref,state,purpose,review_subject_digest,council_subject_digest,artifact_media_type,idempotency_key,max_output_bytes,
 provider_ref,model_ref,agent_ref,external_ref,governance_version,budget_reservation_ref,effect_intent_ref,launch_receipt_ref,
 created_at,deadline_at,started_at,provider_accepted_at,last_observed_at,provider_observed_at,finished_at,failure_code,recipient_mailbox_retired)
SELECT ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,plan_generation,app_spec_generation,spec_hash,
 repository_ref,execution_workspace_ref,state,purpose,review_subject_digest,'',artifact_media_type,idempotency_key,max_output_bytes,
 provider_ref,model_ref,agent_ref,external_ref,governance_version,budget_reservation_ref,effect_intent_ref,launch_receipt_ref,
 created_at,deadline_at,started_at,provider_accepted_at,last_observed_at,provider_observed_at,finished_at,failure_code,recipient_mailbox_retired
FROM executions_v14;
DROP TABLE executions_v14;
PRAGMA legacy_alter_table=OFF;
CREATE INDEX executions_goal_idx ON executions(goal_ref,created_at,ref);
CREATE UNIQUE INDEX executions_one_active_author_per_work_item_idx ON executions(goal_ref,work_item_ref)
 WHERE purpose IN ('work','author') AND state IN ('queued','dispatching','running','awaiting_commit','awaiting_attestation','awaiting_integration');
CREATE UNIQUE INDEX executions_one_active_reviewer_round_idx ON executions(goal_ref,work_item_ref,purpose,review_subject_digest)
 WHERE purpose IN ('primary_review','adversarial_review') AND state IN ('queued','dispatching','running');
CREATE UNIQUE INDEX executions_one_active_council_role_idx ON executions(goal_ref,work_item_ref,purpose,council_subject_digest)
 WHERE purpose IN ('council_proposer','council_critic','council_arbiter') AND state IN ('queued','dispatching','running');
CREATE UNIQUE INDEX executions_distinct_review_process_idx ON executions(goal_ref,work_item_ref,external_ref)
 WHERE external_ref<>'' AND purpose IN ('author','primary_review','adversarial_review','council_proposer','council_critic','council_arbiter');
CREATE TRIGGER executions_identity_immutable BEFORE UPDATE OF goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,
 plan_generation,app_spec_generation,spec_hash,purpose,review_subject_digest,council_subject_digest,artifact_media_type,idempotency_key,max_output_bytes,created_at ON executions
BEGIN SELECT RAISE(ABORT,'sqlite.execution_identity_immutable'); END;
CREATE TRIGGER executions_workspace_write_once BEFORE UPDATE OF repository_ref,execution_workspace_ref ON executions
WHEN NOT ((NEW.repository_ref=OLD.repository_ref AND NEW.execution_workspace_ref=OLD.execution_workspace_ref) OR
 (OLD.repository_ref='' AND OLD.execution_workspace_ref='' AND NEW.repository_ref<>'' AND NEW.execution_workspace_ref<>''))
BEGIN SELECT RAISE(ABORT,'sqlite.execution_workspace_write_once'); END;
CREATE TRIGGER executions_provider_identity_write_once BEFORE UPDATE OF provider_ref,model_ref,agent_ref,external_ref ON executions
WHEN NOT ((NEW.provider_ref=OLD.provider_ref AND NEW.model_ref=OLD.model_ref AND NEW.agent_ref=OLD.agent_ref AND NEW.external_ref=OLD.external_ref) OR
 (OLD.state='dispatching' AND NEW.state='running' AND OLD.provider_ref='' AND OLD.model_ref='' AND OLD.agent_ref='' AND OLD.external_ref='' AND
 length(trim(NEW.provider_ref))>0 AND length(trim(NEW.model_ref))>0 AND length(trim(NEW.agent_ref))>0 AND length(trim(NEW.external_ref))>0))
BEGIN SELECT RAISE(ABORT,'sqlite.execution_provider_identity_write_once'); END;
CREATE TRIGGER executions_replacement_guard BEFORE INSERT ON executions WHEN NEW.attempt_no>1 AND NOT EXISTS (
 SELECT 1 FROM executions previous WHERE previous.goal_ref=NEW.goal_ref AND previous.work_item_ref=NEW.work_item_ref
 AND previous.ref=NEW.replaces_execution_ref AND previous.attempt_no+1=NEW.attempt_no AND previous.max_execution_attempts=NEW.max_execution_attempts
 AND previous.plan_generation=NEW.plan_generation AND previous.app_spec_generation=NEW.app_spec_generation AND previous.spec_hash=NEW.spec_hash
 AND previous.purpose=NEW.purpose AND previous.review_subject_digest=NEW.review_subject_digest AND previous.council_subject_digest=NEW.council_subject_digest
 AND previous.state IN ('failed','stopped') AND previous.finished_at IS NOT NULL AND NEW.created_at>=previous.finished_at)
BEGIN SELECT RAISE(ABORT,'sqlite.execution_replacement_invalid'); END;

DROP TRIGGER artifact_occurrences_causal_guard;
DROP TRIGGER artifact_occurrences_immutable_update;
DROP TRIGGER artifact_occurrences_immutable_delete;
DROP INDEX artifact_occurrences_goal_idx;
DROP INDEX artifact_occurrences_artifact_idx;
PRAGMA legacy_alter_table=ON;
ALTER TABLE artifact_occurrences RENAME TO artifact_occurrences_v14;
CREATE TABLE artifact_occurrences (
 occurrence_ref TEXT PRIMARY KEY CHECK(length(trim(occurrence_ref))>0),
 kind TEXT NOT NULL CHECK(kind IN ('agent_output','test_subject_manifest','test_attestation_report','review_assessment','review_diagnostic','council_contribution')),
 goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,artifact_ref TEXT NOT NULL,
 execution_attempt INTEGER NOT NULL CHECK(execution_attempt>0),plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),created_at INTEGER NOT NULL,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT
) STRICT;
INSERT INTO artifact_occurrences SELECT * FROM artifact_occurrences_v14;
DROP TABLE artifact_occurrences_v14;
PRAGMA legacy_alter_table=OFF;
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

CREATE TABLE council_rounds (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,change_set_ref TEXT NOT NULL,
 project_ref TEXT NOT NULL,subject_digest TEXT NOT NULL CHECK(length(subject_digest)=71 AND substr(subject_digest,1,7)='sha256:'),
 review_subject_digest TEXT NOT NULL CHECK(length(review_subject_digest)=71 AND substr(review_subject_digest,1,7)='sha256:'),
 review_gate_digest TEXT NOT NULL CHECK(length(review_gate_digest)=71 AND substr(review_gate_digest,1,7)='sha256:'),
 policy TEXT NOT NULL CHECK(policy IN ('auto','required')),spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 opened_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,opener TEXT NOT NULL CHECK(opener IN ('auto','director')),
 director_fence INTEGER NOT NULL CHECK(director_fence>=0),request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),
 request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 authorization_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
 opened_at INTEGER NOT NULL,idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),
 UNIQUE(goal_ref,work_item_ref,subject_digest),UNIQUE(ref,goal_ref,work_item_ref,subject_digest),
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(change_set_ref) REFERENCES change_sets(ref) ON DELETE RESTRICT,
 FOREIGN KEY(project_ref) REFERENCES projects(ref) ON DELETE RESTRICT,
 CHECK((opener='auto' AND director_fence=0) OR (opener='director' AND director_fence>0))
) STRICT;
CREATE TABLE council_facts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),round_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,
 council_subject_digest TEXT NOT NULL CHECK(length(council_subject_digest)=71 AND substr(council_subject_digest,1,7)='sha256:'),
 role TEXT NOT NULL CHECK(role IN ('proposer','critic','arbiter')),ballot TEXT NOT NULL CHECK(ballot IN ('accept','reject','abstain','security_veto')),
 contribution_schema TEXT NOT NULL CHECK(contribution_schema='orquesta.council.contribution.v1'),
 body TEXT NOT NULL CHECK(length(trim(body))>0 AND length(body)<=8000 AND instr(body,char(0))=0),
 evidence_json TEXT NOT NULL CHECK(json_valid(evidence_json) AND json_type(evidence_json)='array' AND json_array_length(evidence_json) BETWEEN 1 AND 64),
 contribution_digest TEXT NOT NULL CHECK(length(contribution_digest)=71 AND substr(contribution_digest,1,7)='sha256:'),
 execution_ref TEXT NOT NULL,execution_attempt INTEGER NOT NULL CHECK(execution_attempt>0),launch_receipt_ref TEXT NOT NULL CHECK(length(trim(launch_receipt_ref))>0),
 external_ref TEXT NOT NULL CHECK(length(trim(external_ref))>0),artifact_occurrence_ref TEXT NOT NULL REFERENCES artifact_occurrences(occurrence_ref) ON DELETE RESTRICT,
 artifact_ref TEXT NOT NULL,artifact_digest TEXT NOT NULL CHECK(length(artifact_digest)=64 AND artifact_digest NOT GLOB '*[^0-9a-f]*'),
 idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),
 fact_digest TEXT NOT NULL UNIQUE CHECK(length(fact_digest)=71 AND substr(fact_digest,1,7)='sha256:'),recorded_at INTEGER NOT NULL,
 UNIQUE(round_ref,role),UNIQUE(execution_ref),UNIQUE(launch_receipt_ref),UNIQUE(artifact_occurrence_ref),
 FOREIGN KEY(round_ref,goal_ref,work_item_ref,council_subject_digest) REFERENCES council_rounds(ref,goal_ref,work_item_ref,subject_digest) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT
) STRICT;
CREATE TABLE council_decisions (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),round_ref TEXT NOT NULL UNIQUE,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,
 subject_digest TEXT NOT NULL CHECK(length(subject_digest)=71 AND substr(subject_digest,1,7)='sha256:'),
 outcome TEXT NOT NULL CHECK(outcome IN ('accepted','rejected','no_consensus','blocked_security')),
 decision_digest TEXT NOT NULL UNIQUE CHECK(length(decision_digest)=71 AND substr(decision_digest,1,7)='sha256:'),recorded_at INTEGER NOT NULL,
 FOREIGN KEY(round_ref,goal_ref,work_item_ref,subject_digest) REFERENCES council_rounds(ref,goal_ref,work_item_ref,subject_digest) ON DELETE RESTRICT
) STRICT;
CREATE TABLE council_skips (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,change_set_ref TEXT NOT NULL,project_ref TEXT NOT NULL,
 subject_digest TEXT NOT NULL UNIQUE CHECK(length(subject_digest)=71 AND substr(subject_digest,1,7)='sha256:'),
 review_subject_digest TEXT NOT NULL CHECK(length(review_subject_digest)=71 AND substr(review_subject_digest,1,7)='sha256:'),
 review_gate_digest TEXT NOT NULL CHECK(length(review_gate_digest)=71 AND substr(review_gate_digest,1,7)='sha256:'),
 policy TEXT NOT NULL CHECK(policy='skip_by_operator'),plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,reason TEXT NOT NULL CHECK(length(trim(reason))>0 AND length(reason)<=4000 AND instr(reason,char(0))=0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),
 skip_digest TEXT NOT NULL UNIQUE CHECK(length(skip_digest)=71 AND substr(skip_digest,1,7)='sha256:'),
 request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 authorization_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,recorded_at INTEGER NOT NULL,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(change_set_ref) REFERENCES change_sets(ref) ON DELETE RESTRICT,FOREIGN KEY(project_ref) REFERENCES projects(ref) ON DELETE RESTRICT
) STRICT;
CREATE INDEX council_rounds_goal_idx ON council_rounds(goal_ref,opened_at,ref);
CREATE INDEX council_facts_round_idx ON council_facts(round_ref,recorded_at,ref);
CREATE INDEX council_skips_goal_idx ON council_skips(goal_ref,recorded_at,ref);
CREATE TRIGGER council_rounds_immutable_update BEFORE UPDATE ON council_rounds BEGIN SELECT RAISE(ABORT,'sqlite.council_round_immutable'); END;
CREATE TRIGGER council_rounds_immutable_delete BEFORE DELETE ON council_rounds BEGIN SELECT RAISE(ABORT,'sqlite.council_round_immutable'); END;
CREATE TRIGGER council_facts_immutable_update BEFORE UPDATE ON council_facts BEGIN SELECT RAISE(ABORT,'sqlite.council_fact_immutable'); END;
CREATE TRIGGER council_facts_immutable_delete BEFORE DELETE ON council_facts BEGIN SELECT RAISE(ABORT,'sqlite.council_fact_immutable'); END;
CREATE TRIGGER council_decisions_immutable_update BEFORE UPDATE ON council_decisions BEGIN SELECT RAISE(ABORT,'sqlite.council_decision_immutable'); END;
CREATE TRIGGER council_decisions_immutable_delete BEFORE DELETE ON council_decisions BEGIN SELECT RAISE(ABORT,'sqlite.council_decision_immutable'); END;
CREATE TRIGGER council_skips_immutable_update BEFORE UPDATE ON council_skips BEGIN SELECT RAISE(ABORT,'sqlite.council_skip_immutable'); END;
CREATE TRIGGER council_skips_immutable_delete BEFORE DELETE ON council_skips BEGIN SELECT RAISE(ABORT,'sqlite.council_skip_immutable'); END;

ALTER TABLE effect_intents ADD COLUMN council_subject_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE effect_intents ADD COLUMN council_decision_ref TEXT REFERENCES council_decisions(ref) ON DELETE RESTRICT;
ALTER TABLE effect_intents ADD COLUMN council_decision_digest TEXT;
ALTER TABLE effect_intents ADD COLUMN council_skip_ref TEXT REFERENCES council_skips(ref) ON DELETE RESTRICT;
ALTER TABLE effect_intents ADD COLUMN council_skip_digest TEXT CHECK(
 (council_subject_digest='' AND council_decision_ref IS NULL AND council_decision_digest IS NULL AND council_skip_ref IS NULL AND council_skip_digest IS NULL) OR
 (length(council_subject_digest)=71 AND substr(council_subject_digest,1,7)='sha256:' AND
  ((council_decision_ref IS NOT NULL AND length(council_decision_digest)=71 AND substr(council_decision_digest,1,7)='sha256:' AND council_skip_ref IS NULL AND council_skip_digest IS NULL) OR
   (council_skip_ref IS NOT NULL AND length(council_skip_digest)=71 AND substr(council_skip_digest,1,7)='sha256:' AND council_decision_ref IS NULL AND council_decision_digest IS NULL)))) ;

ALTER TABLE outbox ADD COLUMN council_subject_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE outbox ADD COLUMN council_resolution_kind TEXT NOT NULL DEFAULT '' CHECK(council_resolution_kind IN ('','accepted_round','skip'));
ALTER TABLE outbox ADD COLUMN council_decision_ref TEXT REFERENCES council_decisions(ref) ON DELETE RESTRICT;
ALTER TABLE outbox ADD COLUMN council_decision_digest TEXT;
ALTER TABLE outbox ADD COLUMN council_skip_ref TEXT REFERENCES council_skips(ref) ON DELETE RESTRICT;
ALTER TABLE outbox ADD COLUMN council_skip_digest TEXT CHECK(
 (council_resolution_kind='' AND council_subject_digest='' AND council_decision_ref IS NULL AND council_decision_digest IS NULL AND council_skip_ref IS NULL AND council_skip_digest IS NULL AND (kind<>'integrate_change' OR completed_at IS NOT NULL)) OR
 (kind='integrate_change' AND length(council_subject_digest)=71 AND substr(council_subject_digest,1,7)='sha256:' AND
  ((council_resolution_kind='accepted_round' AND council_decision_ref IS NOT NULL AND length(council_decision_digest)=71 AND substr(council_decision_digest,1,7)='sha256:' AND council_skip_ref IS NULL AND council_skip_digest IS NULL) OR
   (council_resolution_kind='skip' AND council_skip_ref IS NOT NULL AND length(council_skip_digest)=71 AND substr(council_skip_digest,1,7)='sha256:' AND council_decision_ref IS NULL AND council_decision_digest IS NULL)))) ;
DROP TRIGGER outbox_integration_admission_immutable;
CREATE TRIGGER outbox_integration_admission_immutable BEFORE UPDATE OF change_ref,expected_target_oid,admission_request_ref,admission_request_fingerprint,
 review_gate_digest,council_subject_digest,council_resolution_kind,council_decision_ref,council_decision_digest,council_skip_ref,council_skip_digest ON outbox
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_integration_admission_immutable'); END;

ALTER TABLE director_decisions ADD COLUMN council_subject_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE director_decisions ADD COLUMN council_decision_ref TEXT REFERENCES council_decisions(ref) ON DELETE RESTRICT;
ALTER TABLE director_decisions ADD COLUMN council_decision_digest TEXT CHECK(
 (council_subject_digest='' AND council_decision_ref IS NULL AND council_decision_digest IS NULL) OR
 (length(council_subject_digest)=71 AND substr(council_subject_digest,1,7)='sha256:' AND
  council_decision_ref IS NOT NULL AND length(council_decision_digest)=71 AND substr(council_decision_digest,1,7)='sha256:')) ;
