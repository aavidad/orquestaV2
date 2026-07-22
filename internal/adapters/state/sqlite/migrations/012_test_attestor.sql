-- V17 makes required tests, artifact occurrences and attestations durable.
-- CAS blobs remain in artifacts; occurrence identity and causal scope live in
-- artifact_occurrences so the same immutable blob may be cited more than once.
CREATE INDEX goals_state_ref_v17_idx ON goals(state, ref);
PRAGMA legacy_alter_table = ON;
ALTER TABLE executions RENAME TO executions_v12;
CREATE TABLE executions (
    ref TEXT PRIMARY KEY, goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE, work_item_ref TEXT NOT NULL,
    attempt_no INTEGER NOT NULL CHECK (attempt_no > 0), max_execution_attempts INTEGER NOT NULL CHECK (max_execution_attempts > 0), replaces_execution_ref TEXT,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0), app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    repository_ref TEXT NOT NULL DEFAULT '', execution_workspace_ref TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL CHECK (state IN ('queued','dispatching','running','awaiting_commit','awaiting_attestation','awaiting_integration','succeeded','failed','canceled','stopped')),
    artifact_media_type TEXT NOT NULL, idempotency_key TEXT NOT NULL UNIQUE, max_output_bytes INTEGER NOT NULL CHECK(max_output_bytes>0),
    provider_ref TEXT NOT NULL DEFAULT '', model_ref TEXT NOT NULL DEFAULT '', agent_ref TEXT NOT NULL DEFAULT '', external_ref TEXT NOT NULL DEFAULT '',
    governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)), budget_reservation_ref TEXT, effect_intent_ref TEXT, launch_receipt_ref TEXT,
    created_at INTEGER NOT NULL, deadline_at INTEGER, started_at INTEGER, provider_accepted_at INTEGER, last_observed_at INTEGER, provider_observed_at INTEGER, finished_at INTEGER,
    failure_code TEXT NOT NULL DEFAULT '', recipient_mailbox_retired INTEGER NOT NULL DEFAULT 0 CHECK(recipient_mailbox_retired IN (0,1)),
    UNIQUE(goal_ref,ref), UNIQUE(goal_ref,work_item_ref,ref), UNIQUE(goal_ref,work_item_ref,attempt_no),
    FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref,work_item_ref,replaces_execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
    CHECK ((attempt_no=1 AND replaces_execution_ref IS NULL) OR (attempt_no>1 AND replaces_execution_ref IS NOT NULL)),
    CHECK(attempt_no<=max_execution_attempts), CHECK((execution_workspace_ref='') = (repository_ref=''))
) STRICT;
INSERT INTO executions SELECT * FROM executions_v12;
DROP TABLE executions_v12;
ALTER TABLE outbox RENAME TO outbox_v12;
CREATE TABLE outbox (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('launch_agent','observe_agent','stop_agent','deliver_mailbox','prepare_workspace','commit_change','attest_test','integrate_change')),
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL, control_ref TEXT,
    change_ref TEXT NOT NULL DEFAULT '', expected_target_oid TEXT NOT NULL DEFAULT '',
    admission_request_ref TEXT NOT NULL DEFAULT '', admission_request_fingerprint TEXT NOT NULL DEFAULT '',
    plan_generation INTEGER NOT NULL CHECK(plan_generation>0), work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0), mailbox_message_ref TEXT,
    available_at INTEGER NOT NULL, claim_token TEXT, claimed_by TEXT, claimed_until INTEGER, delivery_attempt INTEGER NOT NULL DEFAULT 0 CHECK(delivery_attempt>=0), fence INTEGER NOT NULL DEFAULT 0 CHECK(fence>=0),
    completed_at INTEGER, retired_at INTEGER, quarantined_at INTEGER, last_error_code TEXT NOT NULL DEFAULT '', governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)), effect_intent_ref TEXT,
    FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE CASCADE,
    FOREIGN KEY(control_ref) REFERENCES controls(ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref,mailbox_message_ref,plan_generation,work_item_ref,execution_ref,work_item_generation) REFERENCES mailbox_envelopes(goal_ref,ref,plan_generation,parent_work_item_ref,recipient_execution_ref,recipient_work_item_generation) ON DELETE RESTRICT,
    CHECK((kind IN ('launch_agent','observe_agent','prepare_workspace') AND mailbox_message_ref IS NULL AND control_ref IS NULL AND change_ref='' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='')
       OR (kind IN ('commit_change','attest_test') AND mailbox_message_ref IS NULL AND control_ref IS NULL AND change_ref<>'' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='')
       OR (kind='integrate_change' AND mailbox_message_ref IS NULL AND control_ref IS NULL
           AND change_ref<>'' AND expected_target_oid<>'' AND admission_request_ref<>''
           AND length(admission_request_fingerprint)=64
           AND admission_request_fingerprint NOT GLOB '*[^0-9a-f]*'
           AND governance_version=1 AND effect_intent_ref IS NOT NULL)
       OR (kind='stop_agent' AND mailbox_message_ref IS NULL AND control_ref IS NOT NULL AND change_ref='' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='')
       OR (kind='deliver_mailbox' AND mailbox_message_ref IS NOT NULL AND control_ref IS NULL AND change_ref='' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='')),
    CHECK((claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL) OR (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL AND delivery_attempt>0 AND fence>0)),
    CHECK(quarantined_at IS NULL OR completed_at IS NOT NULL), CHECK(retired_at IS NULL OR kind='deliver_mailbox')
) STRICT;
INSERT INTO outbox SELECT * FROM outbox_v12;
DROP TABLE outbox_v12;
DROP TRIGGER effect_intents_immutable_update;
DROP TRIGGER effect_intents_immutable_delete;
ALTER TABLE effect_intents RENAME TO effect_intents_v12;
CREATE TABLE effect_intents (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0), request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),
 request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 action_ref TEXT NOT NULL UNIQUE CHECK(length(trim(action_ref))>0),
 action_kind TEXT NOT NULL CHECK(action_kind IN ('launch_agent','stop_agent','prepare_workspace','commit_change','attest_test','integrate_change')),
 kind TEXT NOT NULL CHECK(kind IN ('agent_launch','agent_stop','prepare_workspace','commit_change','attest_test','integrate_change')),
 project_ref TEXT NOT NULL, goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0), app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'), actor_ref TEXT NOT NULL CHECK(length(trim(actor_ref))>0),
 proposed_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
 permission TEXT NOT NULL CHECK(permission IN ('goals.create','goals.direct','changes.integrate')),
 authority_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
 demand_ref TEXT NOT NULL CHECK(length(trim(demand_ref))>0), demand_tokens INTEGER NOT NULL CHECK(demand_tokens>=0), demand_money_micros INTEGER NOT NULL CHECK(demand_money_micros>=0),
 demand_currency TEXT NOT NULL DEFAULT '' CHECK((demand_currency='' OR (length(demand_currency)=3 AND demand_currency NOT GLOB '*[^A-Z]*')) AND (demand_money_micros=0 OR demand_currency<>'')),
 demand_active_time_ns INTEGER NOT NULL CHECK(demand_active_time_ns>=0), demand_process_slots INTEGER NOT NULL CHECK(demand_process_slots>=0), demand_disk_bytes INTEGER NOT NULL CHECK(demand_disk_bytes>=0),
 security_criticality TEXT NOT NULL CHECK(security_criticality IN ('normal','sensitive','critical')), reasoning_effort TEXT NOT NULL CHECK(reasoning_effort IN ('low','medium','high','xhigh')),
 policy_hash TEXT NOT NULL CHECK(length(policy_hash)=64 AND policy_hash NOT GLOB '*[^0-9a-f]*'), policy_revision INTEGER NOT NULL CHECK(policy_revision>0), quota_retry_delay_ns INTEGER NOT NULL CHECK(quota_retry_delay_ns>0), approval_ttl_ns INTEGER NOT NULL CHECK(approval_ttl_ns>0),
 target_digest TEXT NOT NULL CHECK(length(target_digest)=64 AND target_digest NOT GLOB '*[^0-9a-f]*'), idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0), created_at INTEGER NOT NULL,
 digest TEXT NOT NULL UNIQUE CHECK(length(digest)=64 AND digest NOT GLOB '*[^0-9a-f]*'),
 FOREIGN KEY(goal_ref,project_ref) REFERENCES goals(ref,project_ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 CHECK((kind='agent_launch')=(action_kind='launch_agent')),
 CHECK((kind='agent_stop')=(action_kind='stop_agent')),
 CHECK((kind='prepare_workspace')=(action_kind='prepare_workspace')),
 CHECK((kind='commit_change')=(action_kind='commit_change')),
 CHECK((kind='attest_test')=(action_kind='attest_test')),
 CHECK((kind='integrate_change')=(action_kind='integrate_change'))
) STRICT;
INSERT INTO effect_intents SELECT * FROM effect_intents_v12;
DROP TABLE effect_intents_v12;
DROP TRIGGER effect_receipts_immutable_update;
DROP TRIGGER effect_receipts_immutable_delete;
ALTER TABLE effect_receipts RENAME TO effect_receipts_v12;
CREATE TABLE effect_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0), intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 intent_digest TEXT NOT NULL CHECK(length(intent_digest)=64 AND intent_digest NOT GLOB '*[^0-9a-f]*'), approval_ref TEXT NOT NULL REFERENCES effect_approvals(ref) ON DELETE RESTRICT,
 attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT, project_ref TEXT NOT NULL, goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0), app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0), spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'), actor_ref TEXT NOT NULL CHECK(length(trim(actor_ref))>0),
 action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT, action_fence INTEGER NOT NULL CHECK(action_fence>0), idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0), external_ref TEXT NOT NULL CHECK(length(trim(external_ref))>0),
 status TEXT NOT NULL CHECK(status IN ('accepted','stopped','already_stopped','already_completed','already_failed','prepared','committed','attested_passed','attested_failed','integrated','conflicted','stale')),
 usage_tokens INTEGER NOT NULL CHECK(usage_tokens>=0), usage_money_micros INTEGER NOT NULL CHECK(usage_money_micros>=0), usage_currency TEXT NOT NULL DEFAULT '', usage_active_time_ns INTEGER NOT NULL CHECK(usage_active_time_ns>=0), usage_process_slots INTEGER NOT NULL CHECK(usage_process_slots>=0), usage_disk_bytes INTEGER NOT NULL CHECK(usage_disk_bytes>=0), usage_known INTEGER NOT NULL CHECK(usage_known BETWEEN 0 AND 31), usage_quality TEXT NOT NULL CHECK(usage_quality IN ('unknown','estimated','measured','exact')), confirmed_at INTEGER NOT NULL,
 UNIQUE(action_ref,action_fence), CHECK((usage_known=0)=(usage_quality='unknown'))
) STRICT;
INSERT INTO effect_receipts SELECT * FROM effect_receipts_v12;
DROP TABLE effect_receipts_v12;
DROP TRIGGER action_consumption_receipt_guard;
DROP TRIGGER action_consumption_effect_receipt_guard;
DROP TRIGGER action_consumption_receipts_immutable_update;
DROP TRIGGER action_consumption_receipts_immutable_delete;
DROP INDEX action_consumption_scheduler_fence_idx;
DROP INDEX action_consumption_mailbox_fence_idx;
ALTER TABLE action_consumption_receipts RENAME TO action_consumption_receipts_v12;
CREATE TABLE action_consumption_receipts (
 action_ref TEXT PRIMARY KEY REFERENCES outbox(ref) ON DELETE RESTRICT, governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)),
 kind TEXT NOT NULL CHECK(kind IN ('launch_agent','observe_agent','stop_agent','deliver_mailbox','prepare_workspace','commit_change','attest_test','integrate_change')),
 goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL, change_ref TEXT NOT NULL DEFAULT '',
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0), work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0), mailbox_message_ref TEXT,
 fence INTEGER NOT NULL CHECK(fence>0), delivery_attempt INTEGER NOT NULL CHECK(delivery_attempt>0), claim_token TEXT NOT NULL UNIQUE, worker_ref TEXT NOT NULL,
 outcome TEXT NOT NULL CHECK(outcome IN ('completed','quarantined')), error_code TEXT NOT NULL DEFAULT '', consumed_at INTEGER NOT NULL, effect_receipt_ref TEXT, legacy_effect_status TEXT, legacy_effect_confirmed_at INTEGER,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,mailbox_message_ref,plan_generation,work_item_ref,execution_ref,work_item_generation) REFERENCES mailbox_envelopes(goal_ref,ref,plan_generation,parent_work_item_ref,recipient_execution_ref,recipient_work_item_generation) ON DELETE RESTRICT,
 CHECK((kind IN ('launch_agent','observe_agent','stop_agent','prepare_workspace','commit_change','attest_test','integrate_change') AND mailbox_message_ref IS NULL) OR (kind='deliver_mailbox' AND mailbox_message_ref IS NOT NULL)),
 CHECK(governance_version=0 OR (legacy_effect_status IS NULL AND legacy_effect_confirmed_at IS NULL))
) STRICT;
INSERT INTO action_consumption_receipts SELECT * FROM action_consumption_receipts_v12;
DROP TABLE action_consumption_receipts_v12;
-- Normalized immutable plan declarations. V16 rows remain empty: migration
-- does not invent required tests for historical work.
CREATE TABLE work_item_required_tests (
 goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL,
 position INTEGER NOT NULL CHECK(position>=0), ref TEXT NOT NULL CHECK(length(trim(ref))>0),
 tool_ref TEXT NOT NULL CHECK(length(trim(tool_ref))>0), working_directory TEXT NOT NULL CHECK(length(trim(working_directory))>0),
 PRIMARY KEY(goal_ref,work_item_ref,ref), UNIQUE(goal_ref,work_item_ref,position),
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE
) STRICT;
CREATE TABLE work_item_required_test_arguments (
 goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, required_test_ref TEXT NOT NULL,
 position INTEGER NOT NULL CHECK(position>=0), value TEXT NOT NULL CHECK(instr(value,char(0))=0),
 PRIMARY KEY(goal_ref,work_item_ref,required_test_ref,position),
 FOREIGN KEY(goal_ref,work_item_ref,required_test_ref)
  REFERENCES work_item_required_tests(goal_ref,work_item_ref,ref) ON DELETE CASCADE
) STRICT;

-- A CAS identity may have many causal occurrences.
CREATE TABLE artifact_occurrences (
 occurrence_ref TEXT PRIMARY KEY CHECK(length(trim(occurrence_ref))>0),
 kind TEXT NOT NULL CHECK(kind IN ('agent_output','test_subject_manifest','test_attestation_report')),
 goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL,
 artifact_ref TEXT NOT NULL, execution_attempt INTEGER NOT NULL CHECK(execution_attempt>0),
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0), work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),
 app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 created_at INTEGER NOT NULL,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT
) STRICT;
INSERT INTO artifact_occurrences(
 occurrence_ref,kind,goal_ref,work_item_ref,execution_ref,artifact_ref,
 execution_attempt,plan_generation,work_item_generation,app_spec_generation,spec_hash,created_at
)
SELECT 'artifact-occurrence:migrated-v16:' || a.ref, 'agent_output', a.goal_ref, a.work_item_ref,
       t.execution_ref, a.ref, e.attempt_no, e.plan_generation, wi.revision,
       e.app_spec_generation, e.spec_hash, a.created_at
FROM artifacts a
JOIN attestations t ON t.goal_ref=a.goal_ref AND t.artifact_ref=a.ref
JOIN executions e ON e.goal_ref=t.goal_ref AND e.work_item_ref=t.work_item_ref AND e.ref=t.execution_ref
JOIN work_items wi ON wi.goal_ref=a.goal_ref AND wi.ref=a.work_item_ref;

ALTER TABLE attestations RENAME TO attestations_v12;
CREATE TABLE attestations (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 kind TEXT NOT NULL CHECK(kind IN ('artifact_provenance','required_tests')),
 verdict TEXT NOT NULL CHECK(verdict IN ('observed','passed','failed')),
 goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL,
 execution_attempt INTEGER NOT NULL CHECK(execution_attempt>0), plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0), app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 artifact_ref TEXT NOT NULL, subject_digest TEXT NOT NULL DEFAULT '', workspace_binding_digest TEXT NOT NULL DEFAULT '',
 change_set_ref TEXT, change_set_digest TEXT NOT NULL DEFAULT '', manifest_artifact_ref TEXT, report_artifact_ref TEXT,
 attestor_ref TEXT NOT NULL DEFAULT '', receipt_ref TEXT NOT NULL DEFAULT '', policy_ref TEXT NOT NULL,
 required_tests_digest TEXT NOT NULL DEFAULT '', policy_digest TEXT NOT NULL DEFAULT '',
 effect_intent_ref TEXT, effect_attempt_ref TEXT, effect_fence INTEGER NOT NULL DEFAULT 0 CHECK(effect_fence>=0), effect_receipt_ref TEXT,
 started_at INTEGER NOT NULL, finished_at INTEGER NOT NULL,
 policy TEXT NOT NULL, accepted_at INTEGER NOT NULL,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(change_set_ref) REFERENCES change_sets(ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,manifest_artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,report_artifact_ref) REFERENCES artifacts(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(effect_intent_ref) REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 FOREIGN KEY(effect_attempt_ref) REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 FOREIGN KEY(effect_receipt_ref) REFERENCES effect_receipts(ref) ON DELETE RESTRICT,
 CHECK(finished_at>=started_at AND accepted_at=finished_at AND policy=policy_ref),
 CHECK((kind='artifact_provenance' AND verdict='observed' AND change_set_ref IS NULL
        AND manifest_artifact_ref IS NULL AND report_artifact_ref IS NULL
        AND attestor_ref='' AND receipt_ref='' AND required_tests_digest=''
        AND effect_intent_ref IS NULL AND effect_attempt_ref IS NULL AND effect_fence=0 AND effect_receipt_ref IS NULL)
    OR (kind='required_tests' AND verdict IN ('passed','failed') AND length(subject_digest)=64
        AND length(workspace_binding_digest)=64 AND change_set_ref IS NOT NULL AND length(change_set_digest)=64
        AND manifest_artifact_ref IS NOT NULL AND report_artifact_ref IS NOT NULL
        AND length(trim(attestor_ref))>0 AND length(trim(receipt_ref))>0
        AND length(required_tests_digest)=64 AND length(policy_digest)=64
        AND effect_intent_ref IS NOT NULL AND effect_attempt_ref IS NOT NULL
        AND effect_fence>0 AND effect_receipt_ref IS NOT NULL))
) STRICT;
INSERT INTO attestations(
 ref,kind,verdict,goal_ref,work_item_ref,execution_ref,execution_attempt,
 plan_generation,work_item_generation,app_spec_generation,spec_hash,artifact_ref,
 policy_ref,started_at,finished_at,policy,accepted_at
)
SELECT t.ref,'artifact_provenance','observed',t.goal_ref,t.work_item_ref,t.execution_ref,
       e.attempt_no,e.plan_generation,wi.revision,e.app_spec_generation,e.spec_hash,
       t.artifact_ref,t.policy,t.accepted_at,t.accepted_at,t.policy,t.accepted_at
FROM attestations_v12 t
JOIN executions e ON e.goal_ref=t.goal_ref AND e.work_item_ref=t.work_item_ref AND e.ref=t.execution_ref
JOIN work_items wi ON wi.goal_ref=t.goal_ref AND wi.ref=t.work_item_ref;
DROP TABLE attestations_v12;

CREATE TABLE attestation_test_outcomes (
 attestation_ref TEXT NOT NULL REFERENCES attestations(ref) ON DELETE CASCADE,
 goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, position INTEGER NOT NULL CHECK(position>=0),
 required_test_ref TEXT NOT NULL, exit_code INTEGER NOT NULL CHECK(exit_code BETWEEN 0 AND 255),
 output_digest TEXT NOT NULL CHECK(length(output_digest)=64 AND output_digest NOT GLOB '*[^0-9a-f]*'),
 PRIMARY KEY(attestation_ref,position), UNIQUE(attestation_ref,required_test_ref),
 FOREIGN KEY(goal_ref,work_item_ref,required_test_ref)
  REFERENCES work_item_required_tests(goal_ref,work_item_ref,ref) ON DELETE RESTRICT
) STRICT;

PRAGMA legacy_alter_table = OFF;

CREATE INDEX executions_goal_idx ON executions(goal_ref,created_at,ref);
CREATE UNIQUE INDEX executions_one_active_per_work_item_idx
 ON executions(goal_ref,work_item_ref)
 WHERE state IN ('queued','dispatching','running','awaiting_commit','awaiting_attestation','awaiting_integration');

CREATE UNIQUE INDEX outbox_claim_token_idx ON outbox(claim_token) WHERE claim_token IS NOT NULL;
CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx
 ON outbox(goal_ref,work_item_ref,plan_generation,work_item_generation)
 WHERE kind IN ('launch_agent','observe_agent','prepare_workspace','commit_change','attest_test','integrate_change')
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_stop_per_execution_idx
 ON outbox(goal_ref,execution_ref) WHERE kind='stop_agent'
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_mailbox_idx ON outbox(mailbox_message_ref) WHERE kind='deliver_mailbox';
CREATE UNIQUE INDEX outbox_one_active_change_effect_idx
 ON outbox(kind,change_ref) WHERE kind IN ('attest_test','integrate_change')
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE INDEX outbox_claimable_idx
 ON outbox(kind,completed_at,retired_at,quarantined_at,available_at,claimed_until,ref);
CREATE INDEX outbox_integration_admission_request_idx
 ON outbox(goal_ref,admission_request_ref) WHERE kind='integrate_change';

CREATE INDEX effect_intents_goal_idx ON effect_intents(goal_ref,created_at,ref);
CREATE UNIQUE INDEX effect_intents_integration_request_scope_idx
 ON effect_intents(proposed_by_ref,project_ref,request_ref) WHERE kind='integrate_change';
CREATE INDEX effect_receipts_goal_idx ON effect_receipts(goal_ref,confirmed_at,ref);
CREATE UNIQUE INDEX action_consumption_scheduler_fence_idx
 ON action_consumption_receipts(goal_ref,work_item_ref,fence)
 WHERE kind IN ('launch_agent','observe_agent','stop_agent','prepare_workspace','commit_change','attest_test','integrate_change');
CREATE UNIQUE INDEX action_consumption_mailbox_fence_idx
 ON action_consumption_receipts(mailbox_message_ref,fence) WHERE kind='deliver_mailbox';
CREATE INDEX artifact_occurrences_goal_idx
 ON artifact_occurrences(goal_ref,created_at,occurrence_ref);
CREATE INDEX artifact_occurrences_artifact_idx
 ON artifact_occurrences(goal_ref,artifact_ref,created_at,occurrence_ref);
CREATE INDEX attestations_goal_idx ON attestations(goal_ref,accepted_at,ref);
CREATE INDEX attestations_work_item_idx ON attestations(work_item_ref,accepted_at,ref);
CREATE UNIQUE INDEX attestations_required_tests_execution_idx
 ON attestations(execution_ref) WHERE kind='required_tests';

CREATE TRIGGER effect_intents_immutable_update BEFORE UPDATE ON effect_intents BEGIN SELECT RAISE(ABORT,'sqlite.effect_intent_immutable'); END;
CREATE TRIGGER effect_intents_immutable_delete BEFORE DELETE ON effect_intents BEGIN SELECT RAISE(ABORT,'sqlite.effect_intent_immutable'); END;
CREATE TRIGGER effect_receipts_immutable_update BEFORE UPDATE ON effect_receipts BEGIN SELECT RAISE(ABORT,'sqlite.effect_receipt_immutable'); END;
CREATE TRIGGER effect_receipts_immutable_delete BEFORE DELETE ON effect_receipts BEGIN SELECT RAISE(ABORT,'sqlite.effect_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_update BEFORE UPDATE ON action_consumption_receipts BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_delete BEFORE DELETE ON action_consumption_receipts BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER artifacts_immutable_update BEFORE UPDATE ON artifacts BEGIN SELECT RAISE(ABORT,'sqlite.artifact_immutable'); END;
CREATE TRIGGER artifacts_immutable_delete BEFORE DELETE ON artifacts BEGIN SELECT RAISE(ABORT,'sqlite.artifact_immutable'); END;
CREATE TRIGGER work_item_required_tests_immutable_update BEFORE UPDATE ON work_item_required_tests BEGIN SELECT RAISE(ABORT,'sqlite.required_test_immutable'); END;
CREATE TRIGGER work_item_required_tests_immutable_delete BEFORE DELETE ON work_item_required_tests BEGIN SELECT RAISE(ABORT,'sqlite.required_test_immutable'); END;
CREATE TRIGGER work_item_required_test_arguments_immutable_update BEFORE UPDATE ON work_item_required_test_arguments BEGIN SELECT RAISE(ABORT,'sqlite.required_test_argument_immutable'); END;
CREATE TRIGGER work_item_required_test_arguments_immutable_delete BEFORE DELETE ON work_item_required_test_arguments BEGIN SELECT RAISE(ABORT,'sqlite.required_test_argument_immutable'); END;
CREATE TRIGGER artifact_occurrences_immutable_update BEFORE UPDATE ON artifact_occurrences BEGIN SELECT RAISE(ABORT,'sqlite.artifact_occurrence_immutable'); END;
CREATE TRIGGER artifact_occurrences_immutable_delete BEFORE DELETE ON artifact_occurrences BEGIN SELECT RAISE(ABORT,'sqlite.artifact_occurrence_immutable'); END;
CREATE TRIGGER attestations_immutable_update BEFORE UPDATE ON attestations BEGIN SELECT RAISE(ABORT,'sqlite.attestation_immutable'); END;
CREATE TRIGGER attestations_immutable_delete BEFORE DELETE ON attestations BEGIN SELECT RAISE(ABORT,'sqlite.attestation_immutable'); END;
CREATE TRIGGER attestation_test_outcomes_immutable_update BEFORE UPDATE ON attestation_test_outcomes BEGIN SELECT RAISE(ABORT,'sqlite.attestation_outcome_immutable'); END;
CREATE TRIGGER attestation_test_outcomes_immutable_delete BEFORE DELETE ON attestation_test_outcomes BEGIN SELECT RAISE(ABORT,'sqlite.attestation_outcome_immutable'); END;

CREATE TRIGGER executions_identity_immutable
BEFORE UPDATE OF goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,plan_generation,
 app_spec_generation,spec_hash,artifact_media_type,idempotency_key,max_output_bytes,created_at ON executions
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
BEFORE INSERT ON executions
WHEN NEW.attempt_no>1 AND NOT EXISTS (
 SELECT 1 FROM executions previous
 WHERE previous.goal_ref=NEW.goal_ref AND previous.work_item_ref=NEW.work_item_ref AND previous.ref=NEW.replaces_execution_ref
  AND previous.attempt_no+1=NEW.attempt_no AND previous.max_execution_attempts=NEW.max_execution_attempts AND previous.plan_generation=NEW.plan_generation
  AND previous.app_spec_generation=NEW.app_spec_generation AND previous.spec_hash=NEW.spec_hash
  AND previous.state IN ('failed','stopped') AND previous.finished_at IS NOT NULL
  AND NEW.created_at>=previous.finished_at)
BEGIN SELECT RAISE(ABORT,'sqlite.execution_replacement_invalid'); END;

CREATE TRIGGER outbox_identity_immutable
BEFORE UPDATE OF kind,goal_ref,work_item_ref,execution_ref,control_ref,
 plan_generation,work_item_generation,mailbox_message_ref ON outbox
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_identity_immutable'); END;
CREATE TRIGGER outbox_integration_admission_immutable
BEFORE UPDATE OF change_ref,expected_target_oid,admission_request_ref,admission_request_fingerprint ON outbox
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_integration_admission_immutable'); END;
CREATE TRIGGER outbox_mailbox_recipient_guard
BEFORE UPDATE ON outbox
WHEN NEW.kind='deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM mailbox_envelopes envelope WHERE envelope.ref=NEW.mailbox_message_ref AND envelope.goal_ref=NEW.goal_ref
  AND envelope.plan_generation=NEW.plan_generation AND envelope.parent_work_item_ref=NEW.work_item_ref
  AND envelope.recipient_execution_ref=NEW.execution_ref AND envelope.recipient_work_item_generation=NEW.work_item_generation
  AND envelope.recipient_principal_ref=NEW.claimed_by)
BEGIN SELECT RAISE(ABORT,'sqlite.mailbox_recipient_mismatch'); END;
CREATE TRIGGER outbox_mailbox_recipient_insert_guard
BEFORE INSERT ON outbox
WHEN NEW.kind='deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM mailbox_envelopes envelope WHERE envelope.ref=NEW.mailbox_message_ref AND envelope.goal_ref=NEW.goal_ref
  AND envelope.plan_generation=NEW.plan_generation AND envelope.parent_work_item_ref=NEW.work_item_ref
  AND envelope.recipient_execution_ref=NEW.execution_ref AND envelope.recipient_work_item_generation=NEW.work_item_generation
  AND envelope.recipient_principal_ref=NEW.claimed_by)
BEGIN SELECT RAISE(ABORT,'sqlite.mailbox_recipient_mismatch'); END;
CREATE TRIGGER outbox_mailbox_retirement_guard
BEFORE UPDATE OF retired_at ON outbox
WHEN NEW.retired_at IS NOT OLD.retired_at AND (
 OLD.retired_at IS NOT NULL OR NEW.retired_at IS NULL OR NOT EXISTS (
  SELECT 1 FROM mailbox_retirements retirement
  WHERE retirement.mailbox_message_ref=NEW.mailbox_message_ref
   AND retirement.action_ref=NEW.ref
   AND retirement.recipient_execution_ref=NEW.execution_ref
   AND retirement.retired_at=NEW.retired_at))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_mailbox_retirement_invalid'); END;
CREATE TRIGGER outbox_governance_insert_guard
BEFORE INSERT ON outbox
WHEN (NEW.governance_version=0 AND NEW.effect_intent_ref IS NOT NULL)
 OR (NEW.governance_version=1 AND (
  NEW.kind NOT IN ('launch_agent','stop_agent','prepare_workspace','commit_change','attest_test','integrate_change')
  OR NEW.effect_intent_ref IS NULL OR NOT EXISTS (
   SELECT 1 FROM effect_intents intent
   WHERE intent.ref=NEW.effect_intent_ref AND intent.action_ref=NEW.ref
    AND intent.action_kind=NEW.kind AND intent.goal_ref=NEW.goal_ref
    AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=NEW.execution_ref
    AND intent.plan_generation=NEW.plan_generation)))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_effect_intent_invalid'); END;
CREATE TRIGGER outbox_governance_immutable
BEFORE UPDATE OF governance_version,effect_intent_ref ON outbox
WHEN NEW.governance_version<>OLD.governance_version OR NEW.effect_intent_ref IS NOT OLD.effect_intent_ref
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_governance_immutable'); END;

DROP TRIGGER effect_attempts_causal_guard;
CREATE TRIGGER effect_attempts_causal_guard
BEFORE INSERT ON effect_attempts
WHEN NOT EXISTS (
 SELECT 1 FROM effect_intents intent JOIN effect_approvals approval ON approval.ref=NEW.approval_ref AND approval.intent_ref=intent.ref
 JOIN outbox action ON action.ref=NEW.action_ref JOIN executions execution ON execution.ref=NEW.execution_ref
 WHERE intent.ref=NEW.intent_ref AND intent.digest=NEW.intent_digest AND approval.intent_digest=intent.digest AND approval.decision='approved'
  AND action.effect_intent_ref=intent.ref AND action.fence=NEW.action_fence AND intent.project_ref=NEW.project_ref AND intent.goal_ref=NEW.goal_ref
  AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=NEW.execution_ref AND intent.plan_generation=NEW.plan_generation
  AND intent.app_spec_generation=NEW.app_spec_generation AND intent.spec_hash=NEW.spec_hash AND intent.actor_ref=NEW.actor_ref
  AND NEW.started_at>=intent.created_at AND NEW.started_at>=approval.decided_at
  AND (approval.source<>'explicit_decision' OR NEW.started_at<approval.expires_at)
  AND NEW.started_at<action.claimed_until
  AND ((intent.kind='agent_launch' AND execution.state='dispatching'
        AND execution.governance_version=1 AND execution.effect_intent_ref=intent.ref AND execution.budget_reservation_ref IS NOT NULL
        AND EXISTS(SELECT 1 FROM budget_reservations reservation WHERE reservation.ref=execution.budget_reservation_ref AND reservation.reserved_at<=NEW.started_at))
    OR (intent.kind='agent_stop' AND execution.state='running')
    OR (intent.kind='prepare_workspace' AND execution.state='queued')
    OR (intent.kind='commit_change' AND execution.state='awaiting_commit')
    OR (intent.kind='attest_test' AND execution.state='awaiting_attestation')
    OR (intent.kind='integrate_change' AND execution.state='awaiting_integration'))
  AND EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=NEW.goal_ref AND event.work_item_ref=NEW.work_item_ref AND event.execution_ref=NEW.execution_ref
              AND event.kind=CASE intent.kind
               WHEN 'agent_launch' THEN 'execution.dispatching'
               WHEN 'agent_stop' THEN 'execution.accepted'
               WHEN 'prepare_workspace' THEN 'execution.queued'
               WHEN 'commit_change' THEN 'execution.output_ready'
               WHEN 'attest_test' THEN 'change.committed' WHEN 'integrate_change' THEN 'test_attestation.passed' END
              AND event.occurred_at<=NEW.started_at))
BEGIN SELECT RAISE(ABORT,'sqlite.effect_attempt_causal_invalid'); END;

CREATE TRIGGER effect_receipts_causal_guard
BEFORE INSERT ON effect_receipts
WHEN NOT EXISTS (
 SELECT 1 FROM effect_attempts attempt
 JOIN effect_intents intent ON intent.ref=attempt.intent_ref
 JOIN outbox action ON action.ref=attempt.action_ref
 WHERE attempt.ref=NEW.attempt_ref AND attempt.intent_ref=NEW.intent_ref
  AND attempt.intent_digest=NEW.intent_digest AND attempt.approval_ref=NEW.approval_ref
  AND attempt.project_ref=NEW.project_ref AND attempt.goal_ref=NEW.goal_ref
  AND attempt.work_item_ref=NEW.work_item_ref AND attempt.execution_ref=NEW.execution_ref
  AND attempt.plan_generation=NEW.plan_generation
  AND attempt.app_spec_generation=NEW.app_spec_generation
  AND attempt.spec_hash=NEW.spec_hash AND attempt.actor_ref=NEW.actor_ref
  AND attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.action_fence
  AND attempt.idempotency_key=NEW.idempotency_key AND NEW.confirmed_at<action.claimed_until
  AND ((intent.kind='agent_launch' AND NEW.status='accepted')
    OR (intent.kind='agent_stop' AND NEW.status IN ('stopped','already_stopped','already_completed','already_failed'))
    OR (intent.kind='prepare_workspace' AND NEW.status='prepared')
    OR (intent.kind='commit_change' AND NEW.status='committed')
    OR (intent.kind='attest_test' AND NEW.status IN ('attested_passed','attested_failed'))
    OR (intent.kind='integrate_change' AND NEW.status IN ('integrated','conflicted','stale'))))
BEGIN SELECT RAISE(ABORT,'sqlite.effect_receipt_causal_invalid'); END;

CREATE TRIGGER action_consumption_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NOT EXISTS (
 SELECT 1 FROM outbox action
 WHERE action.ref=NEW.action_ref AND action.kind=NEW.kind
  AND action.goal_ref=NEW.goal_ref AND action.work_item_ref=NEW.work_item_ref
  AND action.execution_ref=NEW.execution_ref AND action.change_ref=NEW.change_ref
  AND action.plan_generation=NEW.plan_generation
  AND action.work_item_generation=NEW.work_item_generation
  AND action.mailbox_message_ref IS NEW.mailbox_message_ref
  AND action.fence=NEW.fence AND action.delivery_attempt=NEW.delivery_attempt
  AND NEW.consumed_at<action.claimed_until AND action.claim_token=NEW.claim_token
  AND action.claimed_by=NEW.worker_ref AND action.last_error_code=NEW.error_code
  AND action.completed_at=NEW.consumed_at
  AND ((NEW.outcome='completed' AND action.quarantined_at IS NULL)
    OR (NEW.outcome='quarantined' AND action.quarantined_at=NEW.consumed_at)))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_invalid'); END;
CREATE TRIGGER action_consumption_effect_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NEW.governance_version=1 AND (
 (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt
  WHERE receipt.ref=NEW.effect_receipt_ref AND receipt.action_ref=NEW.action_ref
   AND receipt.action_fence=NEW.fence))
 OR (NEW.kind IN ('launch_agent','prepare_workspace','commit_change','attest_test','integrate_change')
     AND NEW.outcome='completed' AND NEW.error_code='' AND NEW.effect_receipt_ref IS NULL)
 OR (NEW.kind='stop_agent' AND NEW.outcome='completed' AND NEW.error_code=''
     AND NEW.effect_receipt_ref IS NULL AND EXISTS (
      SELECT 1 FROM effect_attempts attempt
      WHERE attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.fence)))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_effect_receipt_invalid'); END;

CREATE TRIGGER artifact_occurrences_causal_guard
BEFORE INSERT ON artifact_occurrences
WHEN NOT EXISTS (
 SELECT 1 FROM executions execution JOIN artifacts artifact
  ON artifact.goal_ref=NEW.goal_ref AND artifact.ref=NEW.artifact_ref
 WHERE execution.goal_ref=NEW.goal_ref AND execution.work_item_ref=NEW.work_item_ref
  AND execution.ref=NEW.execution_ref AND execution.attempt_no=NEW.execution_attempt
  AND execution.plan_generation=NEW.plan_generation
  AND execution.app_spec_generation=NEW.app_spec_generation
  AND execution.spec_hash=NEW.spec_hash)
BEGIN SELECT RAISE(ABORT,'sqlite.artifact_occurrence_causal_invalid'); END;
CREATE TRIGGER attestation_test_outcomes_causal_guard
BEFORE INSERT ON attestation_test_outcomes
WHEN NOT EXISTS (
 SELECT 1 FROM attestations attestation JOIN work_item_required_tests required
  ON required.goal_ref=NEW.goal_ref AND required.work_item_ref=NEW.work_item_ref
  AND required.ref=NEW.required_test_ref AND required.position=NEW.position
 WHERE attestation.ref=NEW.attestation_ref AND attestation.kind='required_tests'
  AND attestation.goal_ref=NEW.goal_ref AND attestation.work_item_ref=NEW.work_item_ref)
BEGIN SELECT RAISE(ABORT,'sqlite.attestation_outcome_causal_invalid'); END;

-- Retry requires an immutable exact-zero settlement naming its causal attempt.
ALTER TABLE budget_settlements ADD COLUMN causal_attempt_ref TEXT
 REFERENCES effect_attempts(ref) ON DELETE RESTRICT;
CREATE UNIQUE INDEX budget_settlements_causal_attempt_idx
 ON budget_settlements(causal_attempt_ref) WHERE causal_attempt_ref IS NOT NULL;
CREATE TRIGGER budget_settlements_causal_attempt_guard
BEFORE INSERT ON budget_settlements
WHEN NEW.causal_attempt_ref IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM effect_attempts attempt
 JOIN budget_reservations reservation ON reservation.ref=NEW.reservation_ref
 WHERE attempt.ref=NEW.causal_attempt_ref AND attempt.action_ref=reservation.action_ref
  AND attempt.intent_ref=reservation.effect_intent_ref AND reservation.fence<=attempt.action_fence
  AND NEW.settled_at>=reservation.reserved_at AND NEW.reserved_tokens=reservation.tokens
  AND NEW.reserved_money_micros=reservation.money_micros AND NEW.reserved_currency=reservation.currency
  AND NEW.reserved_active_time_ns=reservation.active_time_ns AND NEW.reserved_process_slots=reservation.process_slots
  AND NEW.reserved_disk_bytes=reservation.disk_bytes
  AND NEW.observed_known=31 AND NEW.observed_quality='exact'
  AND NEW.observed_tokens=0 AND NEW.observed_money_micros=0 AND NEW.observed_currency=reservation.currency
  AND NEW.observed_active_time_ns=0 AND NEW.observed_process_slots=0 AND NEW.observed_disk_bytes=0
  AND NEW.charged_tokens=0 AND NEW.charged_money_micros=0 AND NEW.charged_currency=reservation.currency
  AND NEW.charged_active_time_ns=0 AND NEW.charged_process_slots=0 AND NEW.charged_disk_bytes=0
  AND NEW.released_tokens=reservation.tokens AND NEW.released_money_micros=reservation.money_micros
  AND NEW.released_currency=reservation.currency AND NEW.released_active_time_ns=reservation.active_time_ns
  AND NEW.released_process_slots=reservation.process_slots AND NEW.released_disk_bytes=reservation.disk_bytes
  AND NEW.overrun_tokens=0 AND NEW.overrun_money_micros=0 AND NEW.overrun_currency=reservation.currency
  AND NEW.overrun_active_time_ns=0 AND NEW.overrun_process_slots=0 AND NEW.overrun_disk_bytes=0)
BEGIN SELECT RAISE(ABORT,'sqlite.budget_settlement_causal_attempt_invalid'); END;
