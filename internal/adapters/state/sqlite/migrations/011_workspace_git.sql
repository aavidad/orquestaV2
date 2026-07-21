-- V16 local workspace/Git facts.  These tables are append-only evidence owned
-- by the existing StateRepository transaction; they are deliberately not a
-- second workspace store or an adapter-private queue.

-- SQLite cannot alter a CHECK. Rebuild the affected tables in this migration
-- while foreign keys are disabled by the migration runner. `legacy_alter_table`
-- deliberately keeps existing child foreign keys pointing at the canonical
-- names that are recreated below.
PRAGMA legacy_alter_table = ON;

DROP TRIGGER authorization_receipts_immutable_update;
DROP TRIGGER authorization_receipts_immutable_delete;
ALTER TABLE authorization_receipts RENAME TO authorization_receipts_v11;
CREATE TABLE authorization_receipts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0), request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT, project_ref TEXT NOT NULL CHECK (length(trim(project_ref)) > 0),
    permission TEXT NOT NULL CHECK (permission IN ('project.hierarchy.manage','project.membership.manage','goals.create','goals.amend','goals.get','goals.list','goals.direct','budgets.manage','effects.approve','changes.integrate','artifacts.read','project.status')),
    resource_ref TEXT NOT NULL CHECK (length(trim(resource_ref)) > 0), requested_at INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('allowed','denied')),
    role TEXT NOT NULL DEFAULT '' CHECK (role IN ('','platform_admin','project_owner','project_admin','contributor','reviewer','operator','viewer')),
    membership_revision INTEGER NOT NULL CHECK (membership_revision >= 0), reason_code TEXT NOT NULL CHECK (length(trim(reason_code)) > 0),
    decided_at INTEGER NOT NULL, recorded_at INTEGER NOT NULL, UNIQUE(principal_ref, request_ref),
    CHECK (decided_at >= requested_at), CHECK (recorded_at >= decided_at),
    CHECK ((outcome='allowed' AND role<>'' AND (role='platform_admin' OR membership_revision>0)) OR outcome='denied')
) STRICT;
INSERT INTO authorization_receipts SELECT * FROM authorization_receipts_v11;
DROP TABLE authorization_receipts_v11;
CREATE INDEX authorization_receipts_project_idx ON authorization_receipts(project_ref,recorded_at,ref);
CREATE UNIQUE INDEX authorization_receipts_mailbox_scope_idx ON authorization_receipts(ref,principal_ref,project_ref);
CREATE TRIGGER authorization_receipts_immutable_update BEFORE UPDATE ON authorization_receipts BEGIN SELECT RAISE(ABORT, 'sqlite.authorization_receipt_immutable'); END;
CREATE TRIGGER authorization_receipts_immutable_delete BEFORE DELETE ON authorization_receipts BEGIN SELECT RAISE(ABORT, 'sqlite.authorization_receipt_immutable'); END;

ALTER TABLE executions RENAME TO executions_v11;
CREATE TABLE executions (
    ref TEXT PRIMARY KEY, goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE, work_item_ref TEXT NOT NULL,
    attempt_no INTEGER NOT NULL CHECK (attempt_no > 0), max_execution_attempts INTEGER NOT NULL CHECK (max_execution_attempts > 0), replaces_execution_ref TEXT,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0), app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    repository_ref TEXT NOT NULL DEFAULT '', execution_workspace_ref TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL CHECK (state IN ('queued','dispatching','running','awaiting_commit','awaiting_integration','succeeded','failed','canceled','stopped')),
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
INSERT INTO executions(ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,plan_generation,app_spec_generation,spec_hash,state,artifact_media_type,idempotency_key,max_output_bytes,provider_ref,model_ref,agent_ref,external_ref,governance_version,budget_reservation_ref,effect_intent_ref,launch_receipt_ref,created_at,deadline_at,started_at,provider_accepted_at,last_observed_at,provider_observed_at,finished_at,failure_code,recipient_mailbox_retired)
SELECT ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,plan_generation,app_spec_generation,spec_hash,state,artifact_media_type,idempotency_key,max_output_bytes,provider_ref,model_ref,agent_ref,external_ref,governance_version,budget_reservation_ref,effect_intent_ref,launch_receipt_ref,created_at,deadline_at,started_at,provider_accepted_at,last_observed_at,provider_observed_at,finished_at,failure_code,recipient_mailbox_retired FROM executions_v11;
DROP TABLE executions_v11;

ALTER TABLE outbox RENAME TO outbox_v11;
CREATE TABLE outbox (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('launch_agent','observe_agent','stop_agent','deliver_mailbox','prepare_workspace','commit_change','integrate_change')),
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
       OR (kind='commit_change' AND mailbox_message_ref IS NULL AND control_ref IS NULL AND change_ref<>'' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='')
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
INSERT INTO outbox(ref,kind,goal_ref,work_item_ref,execution_ref,control_ref,plan_generation,work_item_generation,mailbox_message_ref,available_at,claim_token,claimed_by,claimed_until,delivery_attempt,fence,completed_at,retired_at,quarantined_at,last_error_code,governance_version,effect_intent_ref)
SELECT ref,kind,goal_ref,work_item_ref,execution_ref,control_ref,plan_generation,work_item_generation,mailbox_message_ref,available_at,claim_token,claimed_by,claimed_until,delivery_attempt,fence,completed_at,retired_at,quarantined_at,last_error_code,governance_version,effect_intent_ref FROM outbox_v11;
DROP TABLE outbox_v11;

CREATE TRIGGER outbox_integration_admission_immutable
BEFORE UPDATE OF change_ref, expected_target_oid, admission_request_ref, admission_request_fingerprint ON outbox
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_integration_admission_immutable'); END;
CREATE INDEX outbox_integration_admission_request_idx
    ON outbox(goal_ref,admission_request_ref) WHERE kind='integrate_change';

DROP TRIGGER effect_intents_immutable_update;
DROP TRIGGER effect_intents_immutable_delete;
ALTER TABLE effect_intents RENAME TO effect_intents_v11;
CREATE TABLE effect_intents (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0), request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),
 request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 action_ref TEXT NOT NULL UNIQUE CHECK(length(trim(action_ref))>0),
 action_kind TEXT NOT NULL CHECK(action_kind IN ('launch_agent','stop_agent','prepare_workspace','commit_change','integrate_change')),
 kind TEXT NOT NULL CHECK(kind IN ('agent_launch','agent_stop','prepare_workspace','commit_change','integrate_change')),
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
 CHECK((kind='integrate_change')=(action_kind='integrate_change'))
) STRICT;
INSERT INTO effect_intents SELECT * FROM effect_intents_v11;
DROP TABLE effect_intents_v11;
CREATE INDEX effect_intents_goal_idx ON effect_intents(goal_ref,created_at,ref);
CREATE UNIQUE INDEX effect_intents_integration_request_scope_idx
    ON effect_intents(proposed_by_ref,project_ref,request_ref)
    WHERE kind='integrate_change';
CREATE TRIGGER effect_intents_immutable_update BEFORE UPDATE ON effect_intents BEGIN SELECT RAISE(ABORT, 'sqlite.effect_intent_immutable'); END;
CREATE TRIGGER effect_intents_immutable_delete BEFORE DELETE ON effect_intents BEGIN SELECT RAISE(ABORT, 'sqlite.effect_intent_immutable'); END;

DROP TRIGGER effect_approvals_immutable_update;
DROP TRIGGER effect_approvals_immutable_delete;
ALTER TABLE effect_approvals RENAME TO effect_approvals_v11;
CREATE TABLE effect_approvals (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0), request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0), request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT, intent_digest TEXT NOT NULL CHECK(length(intent_digest)=64 AND intent_digest NOT GLOB '*[^0-9a-f]*'),
 project_ref TEXT NOT NULL, goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL, plan_generation INTEGER NOT NULL CHECK(plan_generation>0), app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0), spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'), actor_ref TEXT NOT NULL CHECK(length(trim(actor_ref))>0),
 proposed_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT, decided_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT, decision TEXT NOT NULL CHECK(decision IN ('approved','denied')),
 source TEXT NOT NULL CHECK(source IN ('goal_confirmation','director_decision','explicit_decision','integration_decision')),
 security_criticality TEXT NOT NULL CHECK(security_criticality IN ('normal','sensitive','critical')), policy_hash TEXT NOT NULL CHECK(length(policy_hash)=64 AND policy_hash NOT GLOB '*[^0-9a-f]*'), policy_revision INTEGER NOT NULL CHECK(policy_revision>0), target_digest TEXT NOT NULL CHECK(length(target_digest)=64 AND target_digest NOT GLOB '*[^0-9a-f]*'), reason TEXT NOT NULL CHECK(length(trim(reason))>0), idempotency_key TEXT NOT NULL CHECK(length(trim(idempotency_key))>0), authorization_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT, decided_at INTEGER NOT NULL, expires_at INTEGER,
 CHECK((decision='approved' AND source='explicit_decision' AND expires_at>decided_at) OR (decision='approved' AND source<>'explicit_decision' AND expires_at IS NULL) OR (decision='denied' AND source='explicit_decision' AND expires_at IS NULL))
) STRICT;
INSERT INTO effect_approvals SELECT * FROM effect_approvals_v11;
DROP TABLE effect_approvals_v11;
CREATE INDEX effect_approvals_live_idx ON effect_approvals(intent_ref,decision,expires_at DESC,decided_at DESC,ref);
CREATE TRIGGER effect_approvals_immutable_update BEFORE UPDATE ON effect_approvals BEGIN SELECT RAISE(ABORT, 'sqlite.effect_approval_immutable'); END;
CREATE TRIGGER effect_approvals_immutable_delete BEFORE DELETE ON effect_approvals BEGIN SELECT RAISE(ABORT, 'sqlite.effect_approval_immutable'); END;

DROP TRIGGER effect_receipts_immutable_update;
DROP TRIGGER effect_receipts_immutable_delete;
ALTER TABLE effect_receipts RENAME TO effect_receipts_v11;
CREATE TABLE effect_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0), intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 intent_digest TEXT NOT NULL CHECK(length(intent_digest)=64 AND intent_digest NOT GLOB '*[^0-9a-f]*'), approval_ref TEXT NOT NULL REFERENCES effect_approvals(ref) ON DELETE RESTRICT,
 attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT, project_ref TEXT NOT NULL, goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0), app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0), spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'), actor_ref TEXT NOT NULL CHECK(length(trim(actor_ref))>0),
 action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT, action_fence INTEGER NOT NULL CHECK(action_fence>0), idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0), external_ref TEXT NOT NULL CHECK(length(trim(external_ref))>0),
 status TEXT NOT NULL CHECK(status IN ('accepted','stopped','already_stopped','already_completed','already_failed','prepared','committed','integrated','conflicted','stale')),
 usage_tokens INTEGER NOT NULL CHECK(usage_tokens>=0), usage_money_micros INTEGER NOT NULL CHECK(usage_money_micros>=0), usage_currency TEXT NOT NULL DEFAULT '', usage_active_time_ns INTEGER NOT NULL CHECK(usage_active_time_ns>=0), usage_process_slots INTEGER NOT NULL CHECK(usage_process_slots>=0), usage_disk_bytes INTEGER NOT NULL CHECK(usage_disk_bytes>=0), usage_known INTEGER NOT NULL CHECK(usage_known BETWEEN 0 AND 31), usage_quality TEXT NOT NULL CHECK(usage_quality IN ('unknown','estimated','measured','exact')), confirmed_at INTEGER NOT NULL,
 UNIQUE(action_ref,action_fence), CHECK((usage_known=0)=(usage_quality='unknown'))
) STRICT;
INSERT INTO effect_receipts SELECT * FROM effect_receipts_v11;
DROP TABLE effect_receipts_v11;
CREATE INDEX effect_receipts_goal_idx ON effect_receipts(goal_ref,confirmed_at,ref);
CREATE TRIGGER effect_receipts_immutable_update BEFORE UPDATE ON effect_receipts BEGIN SELECT RAISE(ABORT, 'sqlite.effect_receipt_immutable'); END;
CREATE TRIGGER effect_receipts_immutable_delete BEFORE DELETE ON effect_receipts BEGIN SELECT RAISE(ABORT, 'sqlite.effect_receipt_immutable'); END;

DROP TRIGGER action_consumption_receipt_guard;
DROP TRIGGER action_consumption_effect_receipt_guard;
DROP TRIGGER action_consumption_receipts_immutable_update;
DROP TRIGGER action_consumption_receipts_immutable_delete;
DROP INDEX action_consumption_scheduler_fence_idx;
DROP INDEX action_consumption_mailbox_fence_idx;
ALTER TABLE action_consumption_receipts RENAME TO action_consumption_receipts_v11;
CREATE TABLE action_consumption_receipts (
 action_ref TEXT PRIMARY KEY REFERENCES outbox(ref) ON DELETE RESTRICT, governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)),
 kind TEXT NOT NULL CHECK(kind IN ('launch_agent','observe_agent','stop_agent','deliver_mailbox','prepare_workspace','commit_change','integrate_change')),
 goal_ref TEXT NOT NULL, work_item_ref TEXT NOT NULL, execution_ref TEXT NOT NULL, change_ref TEXT NOT NULL DEFAULT '',
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0), work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0), mailbox_message_ref TEXT,
 fence INTEGER NOT NULL CHECK(fence>0), delivery_attempt INTEGER NOT NULL CHECK(delivery_attempt>0), claim_token TEXT NOT NULL UNIQUE, worker_ref TEXT NOT NULL,
 outcome TEXT NOT NULL CHECK(outcome IN ('completed','quarantined')), error_code TEXT NOT NULL DEFAULT '', consumed_at INTEGER NOT NULL, effect_receipt_ref TEXT, legacy_effect_status TEXT, legacy_effect_confirmed_at INTEGER,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,mailbox_message_ref,plan_generation,work_item_ref,execution_ref,work_item_generation) REFERENCES mailbox_envelopes(goal_ref,ref,plan_generation,parent_work_item_ref,recipient_execution_ref,recipient_work_item_generation) ON DELETE RESTRICT,
 CHECK((kind IN ('launch_agent','observe_agent','stop_agent','prepare_workspace','commit_change','integrate_change') AND mailbox_message_ref IS NULL) OR (kind='deliver_mailbox' AND mailbox_message_ref IS NOT NULL)),
 CHECK(governance_version=0 OR (legacy_effect_status IS NULL AND legacy_effect_confirmed_at IS NULL))
) STRICT;
INSERT INTO action_consumption_receipts(action_ref,governance_version,kind,goal_ref,work_item_ref,execution_ref,plan_generation,work_item_generation,mailbox_message_ref,fence,delivery_attempt,claim_token,worker_ref,outcome,error_code,consumed_at,effect_receipt_ref,legacy_effect_status,legacy_effect_confirmed_at)
SELECT action_ref,governance_version,kind,goal_ref,work_item_ref,execution_ref,plan_generation,work_item_generation,mailbox_message_ref,fence,delivery_attempt,claim_token,worker_ref,outcome,error_code,consumed_at,effect_receipt_ref,legacy_effect_status,legacy_effect_confirmed_at FROM action_consumption_receipts_v11;
DROP TABLE action_consumption_receipts_v11;
CREATE UNIQUE INDEX action_consumption_scheduler_fence_idx ON action_consumption_receipts(goal_ref,work_item_ref,fence) WHERE kind IN ('launch_agent','observe_agent','stop_agent','prepare_workspace','commit_change','integrate_change');
CREATE UNIQUE INDEX action_consumption_mailbox_fence_idx ON action_consumption_receipts(mailbox_message_ref,fence) WHERE kind='deliver_mailbox';
CREATE TRIGGER action_consumption_receipts_immutable_update BEFORE UPDATE ON action_consumption_receipts BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_delete BEFORE DELETE ON action_consumption_receipts BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable'); END;

PRAGMA legacy_alter_table = OFF;

CREATE UNIQUE INDEX repositories_project_ref_unique ON repositories(project_ref, ref);

CREATE TABLE workspace_bindings (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    execution_workspace_ref TEXT NOT NULL UNIQUE CHECK (length(trim(execution_workspace_ref)) > 0),
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    actor_ref TEXT NOT NULL CHECK (length(trim(actor_ref)) > 0),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    repository_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL UNIQUE,
    execution_attempt INTEGER NOT NULL CHECK (execution_attempt > 0),
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    app_spec_hash TEXT NOT NULL CHECK (length(app_spec_hash) = 64 AND app_spec_hash NOT GLOB '*[^0-9a-f]*'),
    write_set_digest TEXT NOT NULL CHECK (length(write_set_digest) = 64 AND write_set_digest NOT GLOB '*[^0-9a-f]*'),
    target_ref TEXT NOT NULL CHECK (length(trim(target_ref)) > 0),
    base_oid TEXT NOT NULL CHECK (length(trim(base_oid)) > 0),
    object_format TEXT NOT NULL CHECK (object_format IN ('sha1', 'sha256')),
    adapter_ref TEXT NOT NULL CHECK (length(trim(adapter_ref)) > 0),
    intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
    attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
    action_fence INTEGER NOT NULL CHECK (action_fence > 0),
    effect_receipt_ref TEXT NOT NULL UNIQUE REFERENCES effect_receipts(ref) ON DELETE RESTRICT,
    prepared_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(project_ref, repository_ref) REFERENCES repositories(project_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    CHECK (ref = execution_workspace_ref),
    CHECK (length(base_oid) = CASE object_format WHEN 'sha1' THEN 40 ELSE 64 END),
    CHECK (base_oid NOT GLOB '*[^0-9a-f]*')
) STRICT;

CREATE INDEX workspace_bindings_project_principal_idx
    ON workspace_bindings(project_ref, principal_ref, prepared_at DESC, ref);
CREATE INDEX workspace_bindings_execution_idx
    ON workspace_bindings(goal_ref, work_item_ref, execution_ref);

CREATE TABLE workspace_binding_write_scopes (
    workspace_binding_ref TEXT NOT NULL REFERENCES workspace_bindings(ref) ON DELETE RESTRICT,
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    write_scope TEXT NOT NULL CHECK (
        length(trim(write_scope)) > 0
        AND write_scope NOT LIKE '/%'
        AND write_scope NOT LIKE '%/../%'
        AND write_scope NOT LIKE '../%'
        AND write_scope NOT LIKE '%/..'
        AND instr(write_scope, char(0)) = 0
    ),
    PRIMARY KEY(workspace_binding_ref, ordinal),
    UNIQUE(workspace_binding_ref, write_scope)
) STRICT;

CREATE TABLE change_sets (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    workspace_binding_ref TEXT NOT NULL UNIQUE REFERENCES workspace_bindings(ref) ON DELETE RESTRICT,
    parent_change_ref TEXT REFERENCES change_sets(ref) ON DELETE RESTRICT,
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    actor_ref TEXT NOT NULL CHECK (length(trim(actor_ref)) > 0),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    repository_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL UNIQUE,
    execution_attempt INTEGER NOT NULL CHECK (execution_attempt > 0),
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    app_spec_hash TEXT NOT NULL CHECK (length(app_spec_hash) = 64 AND app_spec_hash NOT GLOB '*[^0-9a-f]*'),
    write_set_digest TEXT NOT NULL CHECK (length(write_set_digest) = 64 AND write_set_digest NOT GLOB '*[^0-9a-f]*'),
    base_oid TEXT NOT NULL CHECK (length(trim(base_oid)) > 0),
    parent_oid TEXT NOT NULL CHECK (length(trim(parent_oid)) > 0),
    head_oid TEXT NOT NULL CHECK (length(trim(head_oid)) > 0),
    tree_oid TEXT NOT NULL CHECK (length(trim(tree_oid)) > 0),
    object_format TEXT NOT NULL CHECK (object_format IN ('sha1', 'sha256')),
    diff_digest TEXT NOT NULL CHECK (length(diff_digest) = 64 AND diff_digest NOT GLOB '*[^0-9a-f]*'),
    intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
    attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
    action_fence INTEGER NOT NULL CHECK (action_fence > 0),
    idempotency_key TEXT NOT NULL UNIQUE CHECK (length(trim(idempotency_key)) > 0),
    adapter_ref TEXT NOT NULL CHECK (length(trim(adapter_ref)) > 0),
    effect_receipt_ref TEXT NOT NULL UNIQUE REFERENCES effect_receipts(ref) ON DELETE RESTRICT,
    committed_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(project_ref, repository_ref) REFERENCES repositories(project_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    CHECK (parent_change_ref IS NULL OR parent_change_ref <> ref),
    CHECK (length(base_oid) = CASE object_format WHEN 'sha1' THEN 40 ELSE 64 END),
    CHECK (length(parent_oid) = length(base_oid) AND length(head_oid) = length(base_oid)
        AND length(tree_oid) = length(base_oid)),
    CHECK (base_oid NOT GLOB '*[^0-9a-f]*' AND parent_oid NOT GLOB '*[^0-9a-f]*'
        AND head_oid NOT GLOB '*[^0-9a-f]*' AND tree_oid NOT GLOB '*[^0-9a-f]*')
) STRICT;

CREATE TABLE change_set_paths (
    change_set_ref TEXT NOT NULL REFERENCES change_sets(ref) ON DELETE RESTRICT,
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    repository_path TEXT NOT NULL CHECK (
        length(trim(repository_path)) > 0
        AND repository_path NOT LIKE '/%'
        AND repository_path NOT LIKE '%/../%'
        AND repository_path NOT LIKE '../%'
        AND repository_path NOT LIKE '%/..'
        AND instr(repository_path, char(0)) = 0
    ),
    PRIMARY KEY(change_set_ref, ordinal),
    UNIQUE(change_set_ref, repository_path)
) STRICT;

CREATE TABLE merge_observations (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    change_set_ref TEXT NOT NULL REFERENCES change_sets(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    repository_ref TEXT NOT NULL REFERENCES repositories(ref) ON DELETE RESTRICT,
    target_ref TEXT NOT NULL CHECK (length(trim(target_ref)) > 0),
    source_oid TEXT NOT NULL CHECK (length(trim(source_oid)) > 0),
    target_oid TEXT NOT NULL CHECK (length(trim(target_oid)) > 0),
    object_format TEXT NOT NULL CHECK (object_format IN ('sha1', 'sha256')),
    outcome TEXT NOT NULL CHECK (outcome IN ('clean', 'conflicted', 'stale')),
    candidate_tree_oid TEXT,
    conflict_digest TEXT,
    adapter_ref TEXT NOT NULL CHECK (length(trim(adapter_ref)) > 0),
    observed_at INTEGER NOT NULL,
    FOREIGN KEY(project_ref, repository_ref) REFERENCES repositories(project_ref, ref) ON DELETE RESTRICT,
    CHECK (length(source_oid) = CASE object_format WHEN 'sha1' THEN 40 ELSE 64 END),
    CHECK (length(target_oid) = length(source_oid)),
    CHECK ((outcome = 'clean' AND candidate_tree_oid IS NOT NULL AND conflict_digest IS NULL)
        OR (outcome IN ('conflicted', 'stale')
            AND candidate_tree_oid IS NULL AND conflict_digest IS NOT NULL)),
    CHECK (candidate_tree_oid IS NULL OR (length(candidate_tree_oid) = length(source_oid)
        AND candidate_tree_oid NOT GLOB '*[^0-9a-f]*')),
    CHECK (conflict_digest IS NULL OR (length(conflict_digest) = 64 AND conflict_digest NOT GLOB '*[^0-9a-f]*')),
    CHECK (source_oid NOT GLOB '*[^0-9a-f]*' AND target_oid NOT GLOB '*[^0-9a-f]*')
) STRICT;

CREATE INDEX merge_observations_change_set_idx
    ON merge_observations(change_set_ref, observed_at DESC, ref);

CREATE TABLE integration_receipts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    change_set_ref TEXT NOT NULL REFERENCES change_sets(ref) ON DELETE RESTRICT,
    merge_observation_ref TEXT NOT NULL UNIQUE REFERENCES merge_observations(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    repository_ref TEXT NOT NULL REFERENCES repositories(ref) ON DELETE RESTRICT,
    target_ref TEXT NOT NULL CHECK (length(trim(target_ref)) > 0),
    source_oid TEXT NOT NULL CHECK (length(trim(source_oid)) > 0),
    target_before_oid TEXT NOT NULL CHECK (length(trim(target_before_oid)) > 0),
    target_after_oid TEXT NOT NULL CHECK (length(trim(target_after_oid)) > 0),
    tree_oid TEXT,
    marker_ref TEXT,
    conflict_digest TEXT,
    object_format TEXT NOT NULL CHECK (object_format IN ('sha1', 'sha256')),
    status TEXT NOT NULL CHECK (status IN ('integrated', 'conflicted', 'stale')),
    intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
    attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
    action_fence INTEGER NOT NULL CHECK (action_fence > 0),
    effect_receipt_ref TEXT NOT NULL UNIQUE REFERENCES effect_receipts(ref) ON DELETE RESTRICT,
    adapter_ref TEXT NOT NULL CHECK (length(trim(adapter_ref)) > 0),
    confirmed_at INTEGER NOT NULL,
    FOREIGN KEY(project_ref, repository_ref) REFERENCES repositories(project_ref, ref) ON DELETE RESTRICT,
    CHECK (length(source_oid) = CASE object_format WHEN 'sha1' THEN 40 ELSE 64 END),
    CHECK (length(target_before_oid) = length(source_oid) AND length(target_after_oid) = length(source_oid)),
    CHECK ((status = 'integrated' AND target_after_oid <> target_before_oid
            AND tree_oid IS NOT NULL AND marker_ref IS NOT NULL AND conflict_digest IS NULL)
        OR (status IN ('conflicted', 'stale') AND target_after_oid = target_before_oid
            AND tree_oid IS NULL AND marker_ref IS NULL AND conflict_digest IS NOT NULL)),
    CHECK (tree_oid IS NULL OR (length(tree_oid) = length(source_oid) AND tree_oid NOT GLOB '*[^0-9a-f]*')),
    CHECK (conflict_digest IS NULL OR (length(conflict_digest) = 64 AND conflict_digest NOT GLOB '*[^0-9a-f]*')),
    CHECK (source_oid NOT GLOB '*[^0-9a-f]*' AND target_before_oid NOT GLOB '*[^0-9a-f]*'
        AND target_after_oid NOT GLOB '*[^0-9a-f]*')
) STRICT;

CREATE INDEX integration_receipts_change_set_idx
    ON integration_receipts(change_set_ref, confirmed_at DESC, ref);
CREATE INDEX integration_receipts_project_idx
    ON integration_receipts(project_ref, confirmed_at DESC, ref);

CREATE TRIGGER workspace_bindings_immutable_update BEFORE UPDATE ON workspace_bindings
BEGIN SELECT RAISE(ABORT, 'sqlite.workspace_binding_immutable'); END;
CREATE TRIGGER workspace_bindings_immutable_delete BEFORE DELETE ON workspace_bindings
BEGIN SELECT RAISE(ABORT, 'sqlite.workspace_binding_immutable'); END;
CREATE TRIGGER change_sets_immutable_update BEFORE UPDATE ON change_sets
BEGIN SELECT RAISE(ABORT, 'sqlite.change_set_immutable'); END;
CREATE TRIGGER change_sets_immutable_delete BEFORE DELETE ON change_sets
BEGIN SELECT RAISE(ABORT, 'sqlite.change_set_immutable'); END;
CREATE TRIGGER change_set_paths_immutable_update BEFORE UPDATE ON change_set_paths
BEGIN SELECT RAISE(ABORT, 'sqlite.change_set_path_immutable'); END;
CREATE TRIGGER change_set_paths_immutable_delete BEFORE DELETE ON change_set_paths
BEGIN SELECT RAISE(ABORT, 'sqlite.change_set_path_immutable'); END;
CREATE TRIGGER workspace_binding_write_scopes_immutable_update BEFORE UPDATE ON workspace_binding_write_scopes
BEGIN SELECT RAISE(ABORT, 'sqlite.workspace_binding_write_scope_immutable'); END;
CREATE TRIGGER workspace_binding_write_scopes_immutable_delete BEFORE DELETE ON workspace_binding_write_scopes
BEGIN SELECT RAISE(ABORT, 'sqlite.workspace_binding_write_scope_immutable'); END;
CREATE TRIGGER merge_observations_immutable_update BEFORE UPDATE ON merge_observations
BEGIN SELECT RAISE(ABORT, 'sqlite.merge_observation_immutable'); END;
CREATE TRIGGER merge_observations_immutable_delete BEFORE DELETE ON merge_observations
BEGIN SELECT RAISE(ABORT, 'sqlite.merge_observation_immutable'); END;
CREATE TRIGGER integration_receipts_immutable_update BEFORE UPDATE ON integration_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.integration_receipt_immutable'); END;
CREATE TRIGGER integration_receipts_immutable_delete BEFORE DELETE ON integration_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.integration_receipt_immutable'); END;

-- Rebuilds above intentionally replace canonical tables, so every index and
-- causal guard owned by those tables must be recreated here.  V16 extends the
-- same scheduler/effect ledger; it does not weaken the V9/V15 invariants.
CREATE INDEX executions_goal_idx ON executions(goal_ref, created_at, ref);
CREATE UNIQUE INDEX executions_one_active_per_work_item_idx
    ON executions(goal_ref, work_item_ref)
    WHERE state IN ('queued','dispatching','running','awaiting_commit','awaiting_integration');

CREATE TRIGGER executions_identity_immutable
BEFORE UPDATE OF goal_ref,work_item_ref,attempt_no,max_execution_attempts,
 replaces_execution_ref,plan_generation,app_spec_generation,spec_hash,
 artifact_media_type,idempotency_key,max_output_bytes,created_at ON executions
BEGIN SELECT RAISE(ABORT, 'sqlite.execution_identity_immutable'); END;
CREATE TRIGGER executions_workspace_write_once
BEFORE UPDATE OF repository_ref,execution_workspace_ref ON executions
WHEN NOT ((NEW.repository_ref=OLD.repository_ref AND NEW.execution_workspace_ref=OLD.execution_workspace_ref)
 OR (OLD.repository_ref='' AND OLD.execution_workspace_ref=''
     AND NEW.repository_ref<>'' AND NEW.execution_workspace_ref<>''))
BEGIN SELECT RAISE(ABORT, 'sqlite.execution_workspace_write_once'); END;
CREATE TRIGGER executions_provider_identity_write_once
BEFORE UPDATE OF provider_ref,model_ref,agent_ref,external_ref ON executions
WHEN NOT ((NEW.provider_ref=OLD.provider_ref AND NEW.model_ref=OLD.model_ref
            AND NEW.agent_ref=OLD.agent_ref AND NEW.external_ref=OLD.external_ref)
 OR (OLD.state='dispatching' AND NEW.state='running'
     AND OLD.provider_ref='' AND OLD.model_ref='' AND OLD.agent_ref='' AND OLD.external_ref=''
     AND length(trim(NEW.provider_ref))>0 AND length(trim(NEW.model_ref))>0
     AND length(trim(NEW.agent_ref))>0 AND length(trim(NEW.external_ref))>0))
BEGIN SELECT RAISE(ABORT, 'sqlite.execution_provider_identity_write_once'); END;
CREATE TRIGGER executions_replacement_guard
BEFORE INSERT ON executions
WHEN NEW.attempt_no>1 AND NOT EXISTS (
 SELECT 1 FROM executions previous
 WHERE previous.goal_ref=NEW.goal_ref AND previous.work_item_ref=NEW.work_item_ref
  AND previous.ref=NEW.replaces_execution_ref AND previous.attempt_no+1=NEW.attempt_no
  AND previous.max_execution_attempts=NEW.max_execution_attempts
  AND previous.plan_generation=NEW.plan_generation
  AND previous.app_spec_generation=NEW.app_spec_generation AND previous.spec_hash=NEW.spec_hash
  AND previous.state IN ('failed','stopped') AND previous.finished_at IS NOT NULL
  AND NEW.created_at>=previous.finished_at)
BEGIN SELECT RAISE(ABORT, 'sqlite.execution_replacement_invalid'); END;

CREATE UNIQUE INDEX outbox_claim_token_idx ON outbox(claim_token) WHERE claim_token IS NOT NULL;
CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx
    ON outbox(goal_ref,work_item_ref,plan_generation,work_item_generation)
    WHERE kind IN ('launch_agent','observe_agent','prepare_workspace','commit_change','integrate_change')
      AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_stop_per_execution_idx
    ON outbox(goal_ref,execution_ref) WHERE kind='stop_agent'
      AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_mailbox_idx
    ON outbox(mailbox_message_ref) WHERE kind='deliver_mailbox';
CREATE UNIQUE INDEX outbox_one_active_integration_per_change_idx
    ON outbox(change_ref) WHERE kind='integrate_change'
      AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE INDEX outbox_claimable_idx
    ON outbox(kind,completed_at,retired_at,quarantined_at,available_at,claimed_until,ref);

CREATE TRIGGER outbox_identity_immutable
BEFORE UPDATE OF kind,goal_ref,work_item_ref,execution_ref,control_ref,
 plan_generation,work_item_generation,mailbox_message_ref ON outbox
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_identity_immutable'); END;
CREATE TRIGGER outbox_mailbox_recipient_guard
BEFORE UPDATE ON outbox
WHEN NEW.kind='deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM mailbox_envelopes envelope
 WHERE envelope.ref=NEW.mailbox_message_ref AND envelope.goal_ref=NEW.goal_ref
  AND envelope.plan_generation=NEW.plan_generation
  AND envelope.parent_work_item_ref=NEW.work_item_ref
  AND envelope.recipient_execution_ref=NEW.execution_ref
  AND envelope.recipient_work_item_generation=NEW.work_item_generation
  AND envelope.recipient_principal_ref=NEW.claimed_by)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_recipient_mismatch'); END;
CREATE TRIGGER outbox_mailbox_recipient_insert_guard
BEFORE INSERT ON outbox
WHEN NEW.kind='deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM mailbox_envelopes envelope
 WHERE envelope.ref=NEW.mailbox_message_ref AND envelope.goal_ref=NEW.goal_ref
  AND envelope.plan_generation=NEW.plan_generation
  AND envelope.parent_work_item_ref=NEW.work_item_ref
  AND envelope.recipient_execution_ref=NEW.execution_ref
  AND envelope.recipient_work_item_generation=NEW.work_item_generation
  AND envelope.recipient_principal_ref=NEW.claimed_by)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_recipient_mismatch'); END;
CREATE TRIGGER outbox_mailbox_retirement_guard
BEFORE UPDATE OF retired_at ON outbox
WHEN NEW.retired_at IS NOT OLD.retired_at AND (
 OLD.retired_at IS NOT NULL OR NEW.retired_at IS NULL OR NOT EXISTS (
  SELECT 1 FROM mailbox_retirements retirement
  WHERE retirement.mailbox_message_ref=NEW.mailbox_message_ref
   AND retirement.action_ref=NEW.ref
   AND retirement.recipient_execution_ref=NEW.execution_ref
   AND retirement.retired_at=NEW.retired_at))
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_mailbox_retirement_invalid'); END;

CREATE TRIGGER outbox_governance_insert_guard
BEFORE INSERT ON outbox
WHEN (NEW.governance_version=0 AND NEW.effect_intent_ref IS NOT NULL)
 OR (NEW.governance_version=1 AND (
  NEW.kind NOT IN ('launch_agent','stop_agent','prepare_workspace','commit_change','integrate_change')
  OR NEW.effect_intent_ref IS NULL OR NOT EXISTS (
   SELECT 1 FROM effect_intents intent
   WHERE intent.ref=NEW.effect_intent_ref AND intent.action_ref=NEW.ref
    AND intent.action_kind=NEW.kind AND intent.goal_ref=NEW.goal_ref
    AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=NEW.execution_ref
    AND intent.plan_generation=NEW.plan_generation)))
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_effect_intent_invalid'); END;
CREATE TRIGGER outbox_governance_immutable
BEFORE UPDATE OF governance_version,effect_intent_ref ON outbox
WHEN NEW.governance_version<>OLD.governance_version
 OR NEW.effect_intent_ref IS NOT OLD.effect_intent_ref
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_governance_immutable'); END;

CREATE TRIGGER effect_approvals_causal_guard
BEFORE INSERT ON effect_approvals
WHEN NOT EXISTS (
 SELECT 1 FROM effect_intents intent
 WHERE intent.ref=NEW.intent_ref AND intent.digest=NEW.intent_digest
  AND intent.project_ref=NEW.project_ref AND intent.goal_ref=NEW.goal_ref
  AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=NEW.execution_ref
  AND intent.plan_generation=NEW.plan_generation
  AND intent.app_spec_generation=NEW.app_spec_generation
  AND intent.spec_hash=NEW.spec_hash AND intent.actor_ref=NEW.actor_ref
  AND NEW.decided_at>=intent.created_at AND intent.proposed_by_ref=NEW.proposed_by_ref
  AND intent.security_criticality=NEW.security_criticality
  AND intent.policy_hash=NEW.policy_hash AND intent.policy_revision=NEW.policy_revision
  AND intent.target_digest=NEW.target_digest AND intent.idempotency_key=NEW.idempotency_key)
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_approval_causal_invalid'); END;

DROP TRIGGER effect_attempts_causal_guard;
CREATE TRIGGER effect_attempts_causal_guard
BEFORE INSERT ON effect_attempts
WHEN NOT EXISTS (
 SELECT 1 FROM effect_intents intent JOIN effect_approvals approval
  ON approval.ref=NEW.approval_ref AND approval.intent_ref=intent.ref
 JOIN outbox action ON action.ref=NEW.action_ref
 JOIN executions execution ON execution.ref=NEW.execution_ref
 WHERE intent.ref=NEW.intent_ref AND intent.digest=NEW.intent_digest
  AND approval.intent_digest=intent.digest AND approval.decision='approved'
  AND action.effect_intent_ref=intent.ref AND action.fence=NEW.action_fence
  AND intent.project_ref=NEW.project_ref AND intent.goal_ref=NEW.goal_ref
  AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=NEW.execution_ref
  AND intent.plan_generation=NEW.plan_generation
  AND intent.app_spec_generation=NEW.app_spec_generation
  AND intent.spec_hash=NEW.spec_hash AND intent.actor_ref=NEW.actor_ref
  AND NEW.started_at>=intent.created_at AND NEW.started_at>=approval.decided_at
  AND (approval.source<>'explicit_decision' OR NEW.started_at<approval.expires_at)
  AND NEW.started_at<action.claimed_until
  AND ((intent.kind='agent_launch' AND execution.state='dispatching'
        AND execution.governance_version=1 AND execution.effect_intent_ref=intent.ref
        AND execution.budget_reservation_ref IS NOT NULL
        AND EXISTS(SELECT 1 FROM budget_reservations reservation
                   WHERE reservation.ref=execution.budget_reservation_ref
                     AND reservation.reserved_at<=NEW.started_at))
    OR (intent.kind='agent_stop' AND execution.state='running')
    OR (intent.kind='prepare_workspace' AND execution.state='queued')
    OR (intent.kind='commit_change' AND execution.state='awaiting_commit')
    OR (intent.kind='integrate_change' AND execution.state='awaiting_integration'))
  AND EXISTS(SELECT 1 FROM events event
             WHERE event.goal_ref=NEW.goal_ref AND event.work_item_ref=NEW.work_item_ref
              AND event.execution_ref=NEW.execution_ref
              AND event.kind=CASE intent.kind
               WHEN 'agent_launch' THEN 'execution.dispatching'
               WHEN 'agent_stop' THEN 'execution.accepted'
               WHEN 'prepare_workspace' THEN 'execution.queued'
               WHEN 'commit_change' THEN 'execution.output_ready'
               WHEN 'integrate_change' THEN 'change.committed' END
              AND event.occurred_at<=NEW.started_at))
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_attempt_causal_invalid'); END;

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
    OR (intent.kind='integrate_change' AND NEW.status IN ('integrated','conflicted','stale'))))
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_receipt_causal_invalid'); END;

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
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_invalid'); END;
CREATE TRIGGER action_consumption_effect_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NEW.governance_version=1 AND (
 (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt
  WHERE receipt.ref=NEW.effect_receipt_ref AND receipt.action_ref=NEW.action_ref
   AND receipt.action_fence=NEW.fence))
 OR (NEW.kind IN ('launch_agent','prepare_workspace','commit_change','integrate_change')
     AND NEW.outcome='completed' AND NEW.error_code='' AND NEW.effect_receipt_ref IS NULL)
 OR (NEW.kind='stop_agent' AND NEW.outcome='completed' AND NEW.error_code=''
     AND NEW.effect_receipt_ref IS NULL AND EXISTS (
      SELECT 1 FROM effect_attempts attempt
      WHERE attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.fence)))
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_effect_receipt_invalid'); END;
