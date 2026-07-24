ALTER TABLE executions ADD COLUMN execution_session_ref TEXT NOT NULL DEFAULT '' CHECK(execution_session_ref='' OR (length(execution_session_ref)=89 AND substr(execution_session_ref,1,25)='execution-session:sha256:'));
CREATE TRIGGER executions_session_ref_once BEFORE UPDATE OF execution_session_ref ON executions
WHEN NEW.execution_session_ref<>OLD.execution_session_ref AND (OLD.execution_session_ref<>'' OR NEW.execution_session_ref='')
BEGIN SELECT RAISE(ABORT,'sqlite.execution_session_ref_immutable'); END;
DROP TRIGGER authorization_receipts_immutable_update;
DROP TRIGGER authorization_receipts_immutable_delete;
DROP INDEX authorization_receipts_project_idx;
DROP INDEX authorization_receipts_mailbox_scope_idx;
DROP TRIGGER action_consumption_receipt_guard;
DROP TRIGGER action_consumption_effect_receipt_guard;
DROP TRIGGER action_consumption_receipts_immutable_update;
DROP TRIGGER action_consumption_receipts_immutable_delete;
DROP INDEX action_consumption_scheduler_fence_idx;
DROP INDEX action_consumption_mailbox_fence_idx;
DROP TRIGGER outbox_identity_immutable;
DROP TRIGGER outbox_mailbox_recipient_guard;
DROP TRIGGER outbox_mailbox_recipient_insert_guard;
DROP TRIGGER outbox_mailbox_retirement_guard;
DROP TRIGGER outbox_governance_insert_guard;
DROP TRIGGER outbox_governance_immutable;
DROP TRIGGER outbox_integration_admission_immutable;
DROP INDEX outbox_claim_token_idx;
DROP INDEX outbox_one_active_stop_per_execution_idx;
DROP INDEX outbox_one_active_mailbox_idx;
DROP INDEX outbox_one_active_change_effect_idx;
DROP INDEX outbox_claimable_idx;
DROP INDEX outbox_integration_admission_request_idx;
DROP INDEX outbox_one_active_per_item_generation_idx;
PRAGMA legacy_alter_table=ON;
ALTER TABLE authorization_receipts RENAME TO authorization_receipts_v16;
ALTER TABLE action_consumption_receipts RENAME TO action_consumption_receipts_v16;
ALTER TABLE outbox RENAME TO outbox_v16;
CREATE TABLE authorization_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),
 request_fingerprint TEXT NOT NULL CHECK(length(trim(request_fingerprint))>0),
 principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
 project_ref TEXT NOT NULL CHECK(length(trim(project_ref))>0),
 permission TEXT NOT NULL CHECK(permission IN ('project.hierarchy.manage','project.membership.manage','goals.create','goals.amend','goals.get','goals.list','goals.direct','budgets.manage','effects.approve','changes.integrate','council.skip','artifacts.read','project.status')),
 resource_ref TEXT NOT NULL CHECK(length(trim(resource_ref))>0),requested_at INTEGER NOT NULL,
 outcome TEXT NOT NULL CHECK(outcome IN ('allowed','denied')),
 role TEXT NOT NULL DEFAULT '' CHECK(role IN ('','platform_admin','project_owner','project_admin','contributor','reviewer','operator','viewer','execution_service')),
 membership_revision INTEGER NOT NULL CHECK(membership_revision>=0),reason_code TEXT NOT NULL CHECK(length(trim(reason_code))>0),
 decided_at INTEGER NOT NULL,recorded_at INTEGER NOT NULL,UNIQUE(principal_ref,request_ref),
 CHECK(decided_at>=requested_at),CHECK(recorded_at>=decided_at),
 CHECK((outcome='allowed' AND role<>'' AND (role='platform_admin' OR membership_revision>0)) OR outcome='denied')
) STRICT;
INSERT INTO authorization_receipts SELECT * FROM authorization_receipts_v16;
CREATE TABLE outbox (
 ref TEXT PRIMARY KEY,
 kind TEXT NOT NULL CHECK(kind IN ('launch_agent','observe_agent','stop_agent','deliver_mailbox','prepare_workspace','commit_change','attest_test','integrate_change','admit_mailbox','revoke_execution_session')),
 goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,control_ref TEXT,
 change_ref TEXT NOT NULL DEFAULT '',expected_target_oid TEXT NOT NULL DEFAULT '',
 admission_request_ref TEXT NOT NULL DEFAULT '',admission_request_fingerprint TEXT NOT NULL DEFAULT '',
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),mailbox_message_ref TEXT,
 available_at INTEGER NOT NULL,claim_token TEXT,claimed_by TEXT,claimed_until INTEGER,
 delivery_attempt INTEGER NOT NULL DEFAULT 0 CHECK(delivery_attempt>=0),fence INTEGER NOT NULL DEFAULT 0 CHECK(fence>=0),
 completed_at INTEGER,retired_at INTEGER,quarantined_at INTEGER,last_error_code TEXT NOT NULL DEFAULT '',
 governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)),effect_intent_ref TEXT,
 review_gate_digest TEXT NOT NULL DEFAULT '',council_subject_digest TEXT NOT NULL DEFAULT '',
 council_resolution_kind TEXT NOT NULL DEFAULT '' CHECK(council_resolution_kind IN ('','accepted_round','skip')),
 council_decision_ref TEXT REFERENCES council_decisions(ref) ON DELETE RESTRICT,council_decision_digest TEXT,
 council_skip_ref TEXT REFERENCES council_skips(ref) ON DELETE RESTRICT,
 council_skip_digest TEXT CHECK(
  (council_resolution_kind='' AND council_subject_digest='' AND council_decision_ref IS NULL AND council_decision_digest IS NULL
   AND council_skip_ref IS NULL AND council_skip_digest IS NULL AND (kind<>'integrate_change' OR completed_at IS NOT NULL)) OR
  (kind='integrate_change' AND length(council_subject_digest)=71 AND substr(council_subject_digest,1,7)='sha256:' AND
   ((council_resolution_kind='accepted_round' AND council_decision_ref IS NOT NULL
     AND length(council_decision_digest)=71 AND substr(council_decision_digest,1,7)='sha256:'
     AND council_skip_ref IS NULL AND council_skip_digest IS NULL) OR
    (council_resolution_kind='skip' AND council_skip_ref IS NOT NULL
     AND length(council_skip_digest)=71 AND substr(council_skip_digest,1,7)='sha256:'
     AND council_decision_ref IS NULL AND council_decision_digest IS NULL)))),
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(control_ref) REFERENCES controls(ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,mailbox_message_ref,plan_generation,work_item_ref,execution_ref,work_item_generation)
  REFERENCES mailbox_envelopes(goal_ref,ref,plan_generation,parent_work_item_ref,recipient_execution_ref,recipient_work_item_generation) ON DELETE RESTRICT,
 CHECK(
  (kind IN ('launch_agent','observe_agent','prepare_workspace','admit_mailbox','revoke_execution_session')
   AND mailbox_message_ref IS NULL AND control_ref IS NULL AND change_ref='' AND expected_target_oid=''
   AND admission_request_ref='' AND admission_request_fingerprint='') OR
  (kind IN ('commit_change','attest_test') AND mailbox_message_ref IS NULL AND control_ref IS NULL
   AND change_ref<>'' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='') OR
  (kind='integrate_change' AND mailbox_message_ref IS NULL AND control_ref IS NULL
   AND change_ref<>'' AND expected_target_oid<>'' AND admission_request_ref<>''
   AND length(admission_request_fingerprint)=64 AND admission_request_fingerprint NOT GLOB '*[^0-9a-f]*'
   AND governance_version=1 AND effect_intent_ref IS NOT NULL) OR
  (kind='stop_agent' AND mailbox_message_ref IS NULL AND control_ref IS NOT NULL
   AND change_ref='' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='') OR
  (kind='deliver_mailbox' AND mailbox_message_ref IS NOT NULL AND control_ref IS NULL
   AND change_ref='' AND expected_target_oid='' AND admission_request_ref='' AND admission_request_fingerprint='')),
 CHECK((claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL) OR
  (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL AND delivery_attempt>0 AND fence>0)),
 CHECK(quarantined_at IS NULL OR completed_at IS NOT NULL),
 CHECK(retired_at IS NULL OR kind='deliver_mailbox')
) STRICT;
INSERT INTO outbox SELECT * FROM outbox_v16;
CREATE TABLE action_consumption_receipts (
 action_ref TEXT PRIMARY KEY REFERENCES outbox(ref) ON DELETE RESTRICT,
 governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)),
 kind TEXT NOT NULL CHECK(kind IN ('launch_agent','observe_agent','stop_agent','deliver_mailbox','prepare_workspace','commit_change','attest_test','integrate_change','admit_mailbox','revoke_execution_session')),
 goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,change_ref TEXT NOT NULL DEFAULT '',
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),mailbox_message_ref TEXT,
 fence INTEGER NOT NULL CHECK(fence>0),delivery_attempt INTEGER NOT NULL CHECK(delivery_attempt>0),
 claim_token TEXT NOT NULL UNIQUE,worker_ref TEXT NOT NULL,
 outcome TEXT NOT NULL CHECK(outcome IN ('completed','quarantined')),error_code TEXT NOT NULL DEFAULT '',
 consumed_at INTEGER NOT NULL,effect_receipt_ref TEXT,legacy_effect_status TEXT,legacy_effect_confirmed_at INTEGER,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,mailbox_message_ref,plan_generation,work_item_ref,execution_ref,work_item_generation)
  REFERENCES mailbox_envelopes(goal_ref,ref,plan_generation,parent_work_item_ref,recipient_execution_ref,recipient_work_item_generation) ON DELETE RESTRICT,
 CHECK((kind IN ('launch_agent','observe_agent','stop_agent','prepare_workspace','commit_change','attest_test','integrate_change','admit_mailbox','revoke_execution_session')
   AND mailbox_message_ref IS NULL) OR (kind='deliver_mailbox' AND mailbox_message_ref IS NOT NULL)),
 CHECK(governance_version=0 OR (legacy_effect_status IS NULL AND legacy_effect_confirmed_at IS NULL))
) STRICT;
INSERT INTO action_consumption_receipts SELECT * FROM action_consumption_receipts_v16;
DROP TABLE action_consumption_receipts_v16;
DROP TABLE outbox_v16;
DROP TABLE authorization_receipts_v16;
PRAGMA legacy_alter_table=OFF;
CREATE INDEX authorization_receipts_project_idx ON authorization_receipts(project_ref,recorded_at,ref);
CREATE UNIQUE INDEX authorization_receipts_mailbox_scope_idx ON authorization_receipts(ref,principal_ref,project_ref);
CREATE TRIGGER authorization_receipts_immutable_update
BEFORE UPDATE ON authorization_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.authorization_receipt_immutable'); END;
CREATE TRIGGER authorization_receipts_immutable_delete
BEFORE DELETE ON authorization_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.authorization_receipt_immutable'); END;
CREATE UNIQUE INDEX outbox_claim_token_idx ON outbox(claim_token) WHERE claim_token IS NOT NULL;
CREATE UNIQUE INDEX outbox_one_active_stop_per_execution_idx ON outbox(goal_ref,execution_ref) WHERE kind='stop_agent'
 AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_mailbox_idx ON outbox(mailbox_message_ref) WHERE kind='deliver_mailbox';
CREATE UNIQUE INDEX outbox_one_active_change_effect_idx ON outbox(kind,change_ref) WHERE kind IN ('attest_test','integrate_change')
 AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE INDEX outbox_claimable_idx ON outbox(kind,completed_at,retired_at,quarantined_at,available_at,claimed_until,ref);
CREATE INDEX outbox_integration_admission_request_idx ON outbox(goal_ref,admission_request_ref) WHERE kind='integrate_change';
CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx ON outbox(goal_ref,work_item_ref,execution_ref,plan_generation,work_item_generation)
 WHERE kind IN ('launch_agent','observe_agent','prepare_workspace','commit_change','attest_test','integrate_change','admit_mailbox')
 AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX action_consumption_scheduler_fence_idx ON action_consumption_receipts(goal_ref,work_item_ref,fence)
 WHERE kind IN ('launch_agent','observe_agent','stop_agent','prepare_workspace','commit_change','attest_test','integrate_change','admit_mailbox','revoke_execution_session');
CREATE UNIQUE INDEX action_consumption_mailbox_fence_idx ON action_consumption_receipts(mailbox_message_ref,fence) WHERE kind='deliver_mailbox';
CREATE TRIGGER outbox_identity_immutable BEFORE UPDATE OF kind,goal_ref,work_item_ref,execution_ref,control_ref,
 plan_generation,work_item_generation,mailbox_message_ref ON outbox
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_identity_immutable'); END;
CREATE TRIGGER outbox_integration_admission_immutable
BEFORE UPDATE OF change_ref,expected_target_oid,admission_request_ref,admission_request_fingerprint,
 review_gate_digest,council_subject_digest,council_resolution_kind,council_decision_ref,council_decision_digest,council_skip_ref,council_skip_digest ON outbox
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
  WHERE retirement.mailbox_message_ref=NEW.mailbox_message_ref AND retirement.action_ref=NEW.ref
  AND retirement.recipient_execution_ref=NEW.execution_ref AND retirement.retired_at=NEW.retired_at))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_mailbox_retirement_invalid'); END;
CREATE TRIGGER outbox_governance_insert_guard
BEFORE INSERT ON outbox
WHEN (NEW.governance_version=0 AND NEW.effect_intent_ref IS NOT NULL)
 OR (NEW.governance_version=1 AND (
  NEW.kind NOT IN ('launch_agent','stop_agent','prepare_workspace','commit_change','attest_test','integrate_change')
  OR NEW.effect_intent_ref IS NULL OR NOT EXISTS (
   SELECT 1 FROM effect_intents intent WHERE intent.ref=NEW.effect_intent_ref AND intent.action_ref=NEW.ref
   AND intent.action_kind=NEW.kind AND intent.goal_ref=NEW.goal_ref AND intent.work_item_ref=NEW.work_item_ref
   AND intent.execution_ref=NEW.execution_ref AND intent.plan_generation=NEW.plan_generation)))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_effect_intent_invalid'); END;
CREATE TRIGGER outbox_governance_immutable
BEFORE UPDATE OF governance_version,effect_intent_ref ON outbox
WHEN NEW.governance_version<>OLD.governance_version OR NEW.effect_intent_ref IS NOT OLD.effect_intent_ref
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_governance_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_update
BEFORE UPDATE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_delete
BEFORE DELETE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NOT EXISTS (
 SELECT 1 FROM outbox action
 WHERE action.ref=NEW.action_ref AND action.kind=NEW.kind
 AND action.goal_ref=NEW.goal_ref AND action.work_item_ref=NEW.work_item_ref
 AND action.execution_ref=NEW.execution_ref AND action.change_ref=NEW.change_ref
 AND action.plan_generation=NEW.plan_generation AND action.work_item_generation=NEW.work_item_generation
 AND action.mailbox_message_ref IS NEW.mailbox_message_ref AND action.fence=NEW.fence
 AND action.delivery_attempt=NEW.delivery_attempt AND NEW.consumed_at<action.claimed_until
 AND action.claim_token=NEW.claim_token AND action.claimed_by=NEW.worker_ref
 AND action.last_error_code=NEW.error_code AND action.completed_at=NEW.consumed_at
 AND ((NEW.outcome='completed' AND action.quarantined_at IS NULL)
 OR (NEW.outcome='quarantined' AND action.quarantined_at=NEW.consumed_at)))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_invalid'); END;
CREATE TRIGGER action_consumption_effect_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NEW.governance_version=1 AND (
 (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt
  WHERE receipt.ref=NEW.effect_receipt_ref AND receipt.action_ref=NEW.action_ref AND receipt.action_fence=NEW.fence))
 OR (NEW.kind IN ('launch_agent','prepare_workspace','commit_change','attest_test','integrate_change')
  AND NEW.outcome='completed' AND NEW.error_code='' AND NEW.effect_receipt_ref IS NULL)
 OR (NEW.kind='stop_agent' AND NEW.outcome='completed' AND NEW.error_code=''
  AND NEW.effect_receipt_ref IS NULL AND EXISTS (
   SELECT 1 FROM effect_attempts attempt
   WHERE attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.fence)))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_effect_receipt_invalid'); END;
