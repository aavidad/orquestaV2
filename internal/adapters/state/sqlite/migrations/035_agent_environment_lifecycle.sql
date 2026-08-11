-- B11/B12: la aplicacion conserva una sola frontera durable Q -> P -> C.
-- Los hechos de efecto siguen en los ledgers existentes; esta tabla solo
-- proyecta el token fisico actual y la autoridad CAS necesaria tras restart.
PRAGMA legacy_alter_table=ON;

ALTER TABLE outbox RENAME TO outbox_v35;
CREATE TABLE outbox (
 ref TEXT PRIMARY KEY,
 kind TEXT NOT NULL CHECK(kind IN (
  'launch_agent','observe_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
  'stop_agent','deliver_mailbox','prepare_workspace','commit_change','attest_test','integrate_change',
  'admit_mailbox','revoke_execution_session')),
 goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
 work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,control_ref TEXT,
 change_ref TEXT NOT NULL DEFAULT '',expected_target_oid TEXT NOT NULL DEFAULT '',
 admission_request_ref TEXT NOT NULL DEFAULT '',admission_request_fingerprint TEXT NOT NULL DEFAULT '',
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),mailbox_message_ref TEXT,
 available_at INTEGER NOT NULL,claim_token TEXT,claimed_by TEXT,claimed_until INTEGER,
 delivery_attempt INTEGER NOT NULL DEFAULT 0 CHECK(delivery_attempt>=0),
 fence INTEGER NOT NULL DEFAULT 0 CHECK(fence>=0),
 completed_at INTEGER,retired_at INTEGER,quarantined_at INTEGER,last_error_code TEXT NOT NULL DEFAULT '',
 governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)),effect_intent_ref TEXT,
 review_gate_digest TEXT NOT NULL DEFAULT '',council_subject_digest TEXT NOT NULL DEFAULT '',
 council_resolution_kind TEXT NOT NULL DEFAULT '' CHECK(council_resolution_kind IN ('','accepted_round','skip')),
 council_decision_ref TEXT REFERENCES council_decisions(ref) ON DELETE RESTRICT,council_decision_digest TEXT,
 council_skip_ref TEXT REFERENCES council_skips(ref) ON DELETE RESTRICT,
 council_skip_digest TEXT CHECK(
  (council_resolution_kind='' AND council_subject_digest='' AND council_decision_ref IS NULL
   AND council_decision_digest IS NULL AND council_skip_ref IS NULL AND council_skip_digest IS NULL
   AND (kind<>'integrate_change' OR completed_at IS NOT NULL)) OR
  (kind='integrate_change' AND length(council_subject_digest)=71
   AND substr(council_subject_digest,1,7)='sha256:' AND
   ((council_resolution_kind='accepted_round' AND council_decision_ref IS NOT NULL
     AND length(council_decision_digest)=71 AND substr(council_decision_digest,1,7)='sha256:'
     AND council_skip_ref IS NULL AND council_skip_digest IS NULL) OR
    (council_resolution_kind='skip' AND council_skip_ref IS NOT NULL
     AND length(council_skip_digest)=71 AND substr(council_skip_digest,1,7)='sha256:'
     AND council_decision_ref IS NULL AND council_decision_digest IS NULL)))),
 recovery_effect_attempt_ref TEXT REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref)
  REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE CASCADE,
 FOREIGN KEY(control_ref) REFERENCES controls(ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,mailbox_message_ref,plan_generation,work_item_ref,execution_ref,work_item_generation)
  REFERENCES mailbox_envelopes(goal_ref,ref,plan_generation,parent_work_item_ref,
   recipient_execution_ref,recipient_work_item_generation) ON DELETE RESTRICT,
 CHECK(
  (kind IN ('launch_agent','observe_agent','quiesce_agent','preserve_agent_environment',
            'close_agent_environment','prepare_workspace','admit_mailbox','revoke_execution_session')
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
  (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL
   AND delivery_attempt>0 AND fence>0)),
 CHECK(kind NOT IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
       OR (governance_version=1 AND effect_intent_ref IS NOT NULL)),
 CHECK(quarantined_at IS NULL OR completed_at IS NOT NULL),
 CHECK(retired_at IS NULL OR kind='deliver_mailbox')
) STRICT;
INSERT INTO outbox SELECT * FROM outbox_v35;
DROP TABLE outbox_v35;

ALTER TABLE effect_intents RENAME TO effect_intents_v35;
CREATE TABLE effect_intents (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),
 request_fingerprint TEXT NOT NULL CHECK(length(request_fingerprint)=64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'),
 action_ref TEXT NOT NULL UNIQUE CHECK(length(trim(action_ref))>0),
 action_kind TEXT NOT NULL CHECK(action_kind IN (
  'launch_agent','quiesce_agent','preserve_agent_environment','close_agent_environment','stop_agent',
  'prepare_workspace','commit_change','attest_test','integrate_change')),
 kind TEXT NOT NULL CHECK(kind IN (
  'agent_launch','agent_quiesce','agent_environment_preserve','agent_environment_close','agent_stop',
  'prepare_workspace','commit_change','attest_test','integrate_change')),
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 actor_ref TEXT NOT NULL CHECK(length(trim(actor_ref))>0),
 proposed_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
 permission TEXT NOT NULL CHECK(permission IN ('goals.create','goals.direct','changes.integrate')),
 authority_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
 demand_ref TEXT NOT NULL CHECK(length(trim(demand_ref))>0),demand_tokens INTEGER NOT NULL CHECK(demand_tokens>=0),
 demand_money_micros INTEGER NOT NULL CHECK(demand_money_micros>=0),
 demand_currency TEXT NOT NULL DEFAULT '' CHECK((demand_currency='' OR
  (length(demand_currency)=3 AND demand_currency NOT GLOB '*[^A-Z]*')) AND
  (demand_money_micros=0 OR demand_currency<>'')),
 demand_active_time_ns INTEGER NOT NULL CHECK(demand_active_time_ns>=0),
 demand_process_slots INTEGER NOT NULL CHECK(demand_process_slots>=0),
 demand_disk_bytes INTEGER NOT NULL CHECK(demand_disk_bytes>=0),
 security_criticality TEXT NOT NULL CHECK(security_criticality IN ('normal','sensitive','critical')),
 reasoning_effort TEXT NOT NULL CHECK(reasoning_effort IN ('low','medium','high','xhigh')),
 policy_hash TEXT NOT NULL CHECK(length(policy_hash)=64 AND policy_hash NOT GLOB '*[^0-9a-f]*'),
 policy_revision INTEGER NOT NULL CHECK(policy_revision>0),quota_retry_delay_ns INTEGER NOT NULL CHECK(quota_retry_delay_ns>0),
 approval_ttl_ns INTEGER NOT NULL CHECK(approval_ttl_ns>0),
 target_digest TEXT NOT NULL CHECK(length(target_digest)=64 AND target_digest NOT GLOB '*[^0-9a-f]*'),
 idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),created_at INTEGER NOT NULL,
 digest TEXT NOT NULL UNIQUE CHECK(length(digest)=64 AND digest NOT GLOB '*[^0-9a-f]*'),
 council_subject_digest TEXT NOT NULL DEFAULT '',
 council_decision_ref TEXT REFERENCES council_decisions(ref) ON DELETE RESTRICT,council_decision_digest TEXT,
 council_skip_ref TEXT REFERENCES council_skips(ref) ON DELETE RESTRICT,
 council_skip_digest TEXT CHECK(
  (council_subject_digest='' AND council_decision_ref IS NULL AND council_decision_digest IS NULL
   AND council_skip_ref IS NULL AND council_skip_digest IS NULL) OR
  (length(council_subject_digest)=71 AND substr(council_subject_digest,1,7)='sha256:' AND
   ((council_decision_ref IS NOT NULL AND length(council_decision_digest)=71
     AND substr(council_decision_digest,1,7)='sha256:' AND council_skip_ref IS NULL AND council_skip_digest IS NULL) OR
    (council_skip_ref IS NOT NULL AND length(council_skip_digest)=71
     AND substr(council_skip_digest,1,7)='sha256:' AND council_decision_ref IS NULL AND council_decision_digest IS NULL)))),
 FOREIGN KEY(goal_ref,project_ref) REFERENCES goals(ref,project_ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref)
  REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 CHECK((kind='agent_launch')=(action_kind='launch_agent')),
 CHECK((kind='agent_quiesce')=(action_kind='quiesce_agent')),
 CHECK((kind='agent_environment_preserve')=(action_kind='preserve_agent_environment')),
 CHECK((kind='agent_environment_close')=(action_kind='close_agent_environment')),
 CHECK((kind='agent_stop')=(action_kind='stop_agent')),
 CHECK((kind='prepare_workspace')=(action_kind='prepare_workspace')),
 CHECK((kind='commit_change')=(action_kind='commit_change')),
 CHECK((kind='attest_test')=(action_kind='attest_test')),
 CHECK((kind='integrate_change')=(action_kind='integrate_change'))
) STRICT;
INSERT INTO effect_intents SELECT * FROM effect_intents_v35;
DROP TABLE effect_intents_v35;

ALTER TABLE effect_receipts RENAME TO effect_receipts_v35;
CREATE TABLE effect_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 intent_digest TEXT NOT NULL CHECK(length(intent_digest)=64 AND intent_digest NOT GLOB '*[^0-9a-f]*'),
 approval_ref TEXT NOT NULL REFERENCES effect_approvals(ref) ON DELETE RESTRICT,
 attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),app_spec_generation INTEGER NOT NULL CHECK(app_spec_generation>0),
 spec_hash TEXT NOT NULL CHECK(length(spec_hash)=64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
 actor_ref TEXT NOT NULL CHECK(length(trim(actor_ref))>0),
 action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT,action_fence INTEGER NOT NULL CHECK(action_fence>0),
 idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),
 external_ref TEXT NOT NULL CHECK(length(trim(external_ref))>0),
 status TEXT NOT NULL CHECK(status IN (
  'accepted','quiesced','preserved','closed','stopped','already_stopped','already_completed','already_failed',
  'prepared','committed','attested_passed','attested_failed','integrated','conflicted','stale')),
 usage_tokens INTEGER NOT NULL CHECK(usage_tokens>=0),usage_money_micros INTEGER NOT NULL CHECK(usage_money_micros>=0),
 usage_currency TEXT NOT NULL DEFAULT '',usage_active_time_ns INTEGER NOT NULL CHECK(usage_active_time_ns>=0),
 usage_process_slots INTEGER NOT NULL CHECK(usage_process_slots>=0),usage_disk_bytes INTEGER NOT NULL CHECK(usage_disk_bytes>=0),
 usage_known INTEGER NOT NULL CHECK(usage_known BETWEEN 0 AND 31),
 usage_quality TEXT NOT NULL CHECK(usage_quality IN ('unknown','estimated','measured','exact')),confirmed_at INTEGER NOT NULL,
 UNIQUE(action_ref,action_fence),CHECK((usage_known=0)=(usage_quality='unknown'))
) STRICT;
INSERT INTO effect_receipts SELECT * FROM effect_receipts_v35;
DROP TABLE effect_receipts_v35;

ALTER TABLE action_consumption_receipts RENAME TO action_consumption_receipts_v35;
CREATE TABLE action_consumption_receipts (
 action_ref TEXT PRIMARY KEY REFERENCES outbox(ref) ON DELETE RESTRICT,
 governance_version INTEGER NOT NULL DEFAULT 0 CHECK(governance_version IN (0,1)),
 kind TEXT NOT NULL CHECK(kind IN (
  'launch_agent','observe_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
  'stop_agent','deliver_mailbox','prepare_workspace','commit_change','attest_test','integrate_change',
  'admit_mailbox','revoke_execution_session')),
 goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL,change_ref TEXT NOT NULL DEFAULT '',
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),
 mailbox_message_ref TEXT,fence INTEGER NOT NULL CHECK(fence>0),delivery_attempt INTEGER NOT NULL CHECK(delivery_attempt>0),
 claim_token TEXT NOT NULL UNIQUE,worker_ref TEXT NOT NULL,
 outcome TEXT NOT NULL CHECK(outcome IN ('completed','quarantined')),error_code TEXT NOT NULL DEFAULT '',
 consumed_at INTEGER NOT NULL,effect_receipt_ref TEXT,legacy_effect_status TEXT,legacy_effect_confirmed_at INTEGER,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref)
  REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,mailbox_message_ref,plan_generation,work_item_ref,execution_ref,work_item_generation)
  REFERENCES mailbox_envelopes(goal_ref,ref,plan_generation,parent_work_item_ref,
   recipient_execution_ref,recipient_work_item_generation) ON DELETE RESTRICT,
 CHECK((kind IN ('launch_agent','observe_agent','quiesce_agent','preserve_agent_environment',
                 'close_agent_environment','stop_agent','prepare_workspace','commit_change','attest_test',
                 'integrate_change','admit_mailbox','revoke_execution_session') AND mailbox_message_ref IS NULL)
  OR (kind='deliver_mailbox' AND mailbox_message_ref IS NOT NULL)),
 CHECK(governance_version=0 OR (legacy_effect_status IS NULL AND legacy_effect_confirmed_at IS NULL))
) STRICT;
INSERT INTO action_consumption_receipts SELECT * FROM action_consumption_receipts_v35;
DROP TABLE action_consumption_receipts_v35;

-- La aplicación ya distingue preservación ligada a workspace y preservación
-- física sin workspace. La tabla histórica solo admitía el primer caso.
ALTER TABLE agent_environment_receipts RENAME TO agent_environment_receipts_v35;
CREATE TABLE agent_environment_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,
 execution_ref TEXT NOT NULL UNIQUE,
 workspace_scope TEXT NOT NULL CHECK(workspace_scope IN ('','workspace_bound','workspace_absent')),
 workspace_ref TEXT,workspace_binding_digest TEXT NOT NULL,base_oid TEXT NOT NULL,
 object_format TEXT NOT NULL CHECK(object_format IN ('','sha1','sha256')),
 change_set_ref TEXT,change_digest TEXT,
 state TEXT NOT NULL CHECK(state='preserved_pending_review'),
 execution_attempt INTEGER NOT NULL CHECK(execution_attempt>0),
 external_ref TEXT NOT NULL CHECK(length(trim(external_ref))>0),fence INTEGER NOT NULL CHECK(fence>0),
 bundle_ref TEXT NOT NULL,bundle_digest TEXT NOT NULL,inventory_ref TEXT NOT NULL,inventory_digest TEXT NOT NULL,
 configuration_digest TEXT NOT NULL,rootfs_digest TEXT NOT NULL,seal_digest TEXT NOT NULL,
 provider_receipt_ref TEXT NOT NULL CHECK(length(trim(provider_receipt_ref))>0),
 sealed_at INTEGER NOT NULL,preserved_at INTEGER NOT NULL,recorded_at INTEGER NOT NULL,
 physical_manifest_ref TEXT CHECK(physical_manifest_ref IS NULL OR (
  typeof(physical_manifest_ref)='text' AND length(CAST(physical_manifest_ref AS BLOB)) BETWEEN 1 AND 512
  AND trim(physical_manifest_ref)=physical_manifest_ref AND instr(physical_manifest_ref,char(0))=0)),
 physical_manifest_digest TEXT CHECK(
  (physical_manifest_ref IS NULL)=(physical_manifest_digest IS NULL) AND
  (physical_manifest_digest IS NULL OR (length(physical_manifest_digest)=64
   AND physical_manifest_digest NOT GLOB '*[^0-9a-f]*'))),
 FOREIGN KEY(goal_ref,project_ref) REFERENCES goals(ref,project_ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref)
  REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(workspace_ref) REFERENCES workspace_bindings(execution_workspace_ref) ON DELETE RESTRICT,
 FOREIGN KEY(change_set_ref) REFERENCES change_sets(ref) ON DELETE RESTRICT,
 CHECK(
  (workspace_scope IN ('','workspace_bound') AND workspace_ref IS NOT NULL
   AND length(workspace_binding_digest)=64 AND workspace_binding_digest NOT GLOB '*[^0-9a-f]*'
   AND length(base_oid)=CASE object_format WHEN 'sha1' THEN 40 ELSE 64 END
   AND base_oid NOT GLOB '*[^0-9a-f]*')
  OR
  (workspace_scope='workspace_absent' AND workspace_ref IS NULL AND workspace_binding_digest=''
   AND base_oid='' AND object_format='' AND change_set_ref IS NULL AND change_digest IS NULL)),
 CHECK((change_set_ref IS NULL)=(change_digest IS NULL)),
 CHECK(bundle_ref='artifact:sha256:'||bundle_digest AND inventory_ref='artifact:sha256:'||inventory_digest),
 CHECK(length(bundle_digest)=64 AND bundle_digest NOT GLOB '*[^0-9a-f]*'
  AND length(inventory_digest)=64 AND inventory_digest NOT GLOB '*[^0-9a-f]*'),
 CHECK(length(configuration_digest)=64 AND configuration_digest NOT GLOB '*[^0-9a-f]*'
  AND length(rootfs_digest)=64 AND rootfs_digest NOT GLOB '*[^0-9a-f]*'
  AND length(seal_digest)=64 AND seal_digest NOT GLOB '*[^0-9a-f]*'),
 CHECK(change_digest IS NULL OR (length(change_digest)=64 AND change_digest NOT GLOB '*[^0-9a-f]*')),
 CHECK(sealed_at<=preserved_at AND preserved_at<=recorded_at)
) STRICT;
INSERT INTO agent_environment_receipts(
 ref,idempotency_key,project_ref,goal_ref,work_item_ref,execution_ref,workspace_scope,
 workspace_ref,workspace_binding_digest,base_oid,object_format,change_set_ref,change_digest,
 state,execution_attempt,external_ref,fence,bundle_ref,bundle_digest,inventory_ref,inventory_digest,
 configuration_digest,rootfs_digest,seal_digest,provider_receipt_ref,sealed_at,preserved_at,recorded_at,
 physical_manifest_ref,physical_manifest_digest
) SELECT
 ref,idempotency_key,project_ref,goal_ref,work_item_ref,execution_ref,'',
 workspace_ref,workspace_binding_digest,base_oid,object_format,change_set_ref,change_digest,
 state,execution_attempt,external_ref,fence,bundle_ref,bundle_digest,inventory_ref,inventory_digest,
 configuration_digest,rootfs_digest,seal_digest,provider_receipt_ref,sealed_at,preserved_at,recorded_at,
 physical_manifest_ref,physical_manifest_digest
FROM agent_environment_receipts_v35;
DROP TABLE agent_environment_receipts_v35;
CREATE UNIQUE INDEX agent_environment_receipts_workspace_idx
 ON agent_environment_receipts(workspace_ref) WHERE workspace_ref IS NOT NULL;
CREATE TRIGGER agent_environment_receipts_immutable_update BEFORE UPDATE ON agent_environment_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.agent_environment_receipt_immutable'); END;
CREATE TRIGGER agent_environment_receipts_immutable_delete BEFORE DELETE ON agent_environment_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.agent_environment_receipt_immutable'); END;

CREATE TABLE agent_environment_lifecycles (
 execution_ref TEXT PRIMARY KEY,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,
 snapshot_schema TEXT NOT NULL CHECK(snapshot_schema='orquesta.agent-environment-lifecycle.snapshot.v1'),
 revision INTEGER NOT NULL CHECK(revision>0),launch_receipt_ref TEXT NOT NULL CHECK(length(trim(launch_receipt_ref))>0),
 token_state TEXT NOT NULL CHECK(token_state IN ('active','quiesced','preserved','closed')),
 snapshot_json TEXT NOT NULL CHECK(typeof(snapshot_json)='text' AND length(snapshot_json)>2),
 claim_action_ref TEXT REFERENCES outbox(ref) ON DELETE RESTRICT,claim_token TEXT,claim_worker_ref TEXT,
 claim_delivery_attempt INTEGER,claim_fence INTEGER,claim_lease_until INTEGER,
 action_approval_attached INTEGER NOT NULL DEFAULT 0 CHECK(action_approval_attached IN (0,1)),
 attempt_ref TEXT REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 preservation_ref TEXT REFERENCES agent_environment_receipts(ref) ON DELETE RESTRICT,
 next_action_ref TEXT REFERENCES outbox(ref) ON DELETE RESTRICT,
 next_action_approval_attached INTEGER NOT NULL DEFAULT 0 CHECK(next_action_approval_attached IN (0,1)),
 ready_to_finalize INTEGER NOT NULL DEFAULT 0 CHECK(ready_to_finalize IN (0,1)),recorded_at INTEGER NOT NULL,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref)
  REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 CHECK((claim_action_ref IS NULL AND claim_token IS NULL AND claim_worker_ref IS NULL
        AND claim_delivery_attempt IS NULL AND claim_fence IS NULL AND claim_lease_until IS NULL
        AND attempt_ref IS NULL AND action_approval_attached=0)
    OR (claim_action_ref IS NOT NULL AND claim_token IS NOT NULL AND claim_worker_ref IS NOT NULL
        AND claim_delivery_attempt>0 AND claim_fence>0 AND claim_lease_until IS NOT NULL
        AND attempt_ref IS NOT NULL)),
 CHECK((next_action_ref IS NULL)=(next_action_approval_attached=0)),
 CHECK(ready_to_finalize=0 OR (token_state='closed' AND next_action_ref IS NULL))
) STRICT;

PRAGMA legacy_alter_table=OFF;

CREATE UNIQUE INDEX outbox_claim_token_idx ON outbox(claim_token) WHERE claim_token IS NOT NULL;
CREATE UNIQUE INDEX outbox_one_active_stop_per_execution_idx ON outbox(goal_ref,execution_ref)
 WHERE kind='stop_agent' AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_mailbox_idx ON outbox(mailbox_message_ref) WHERE kind='deliver_mailbox';
CREATE UNIQUE INDEX outbox_one_active_change_effect_idx ON outbox(kind,change_ref)
 WHERE kind IN ('attest_test','integrate_change') AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE INDEX outbox_claimable_idx ON outbox(kind,completed_at,retired_at,quarantined_at,available_at,claimed_until,ref);
CREATE INDEX outbox_integration_admission_request_idx ON outbox(goal_ref,admission_request_ref) WHERE kind='integrate_change';
CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx
 ON outbox(goal_ref,work_item_ref,execution_ref,plan_generation,work_item_generation)
 WHERE kind IN ('launch_agent','observe_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
                'prepare_workspace','commit_change','attest_test','integrate_change','admit_mailbox')
 AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;

CREATE INDEX effect_intents_goal_idx ON effect_intents(goal_ref,created_at,ref);
CREATE UNIQUE INDEX effect_intents_integration_request_scope_idx
 ON effect_intents(proposed_by_ref,project_ref,request_ref) WHERE kind='integrate_change';
CREATE INDEX effect_receipts_goal_idx ON effect_receipts(goal_ref,confirmed_at,ref);
CREATE UNIQUE INDEX action_consumption_scheduler_fence_idx
 ON action_consumption_receipts(goal_ref,work_item_ref,fence)
 WHERE kind IN ('launch_agent','observe_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
                'stop_agent','prepare_workspace','commit_change','attest_test','integrate_change',
                'admit_mailbox','revoke_execution_session');
CREATE UNIQUE INDEX action_consumption_mailbox_fence_idx
 ON action_consumption_receipts(mailbox_message_ref,fence) WHERE kind='deliver_mailbox';

CREATE TRIGGER outbox_identity_immutable BEFORE UPDATE OF kind,goal_ref,work_item_ref,execution_ref,control_ref,
 plan_generation,work_item_generation,mailbox_message_ref ON outbox
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_identity_immutable'); END;
CREATE TRIGGER outbox_integration_admission_immutable
BEFORE UPDATE OF change_ref,expected_target_oid,admission_request_ref,admission_request_fingerprint,
 review_gate_digest,council_subject_digest,council_resolution_kind,council_decision_ref,
 council_decision_digest,council_skip_ref,council_skip_digest ON outbox
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_integration_admission_immutable'); END;
CREATE TRIGGER outbox_mailbox_recipient_guard BEFORE UPDATE ON outbox
WHEN NEW.kind='deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM mailbox_envelopes envelope WHERE envelope.ref=NEW.mailbox_message_ref
 AND envelope.goal_ref=NEW.goal_ref AND envelope.plan_generation=NEW.plan_generation
 AND envelope.parent_work_item_ref=NEW.work_item_ref AND envelope.recipient_execution_ref=NEW.execution_ref
 AND envelope.recipient_work_item_generation=NEW.work_item_generation
 AND envelope.recipient_principal_ref=NEW.claimed_by)
BEGIN SELECT RAISE(ABORT,'sqlite.mailbox_recipient_mismatch'); END;
CREATE TRIGGER outbox_mailbox_recipient_insert_guard BEFORE INSERT ON outbox
WHEN NEW.kind='deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
 SELECT 1 FROM mailbox_envelopes envelope WHERE envelope.ref=NEW.mailbox_message_ref
 AND envelope.goal_ref=NEW.goal_ref AND envelope.plan_generation=NEW.plan_generation
 AND envelope.parent_work_item_ref=NEW.work_item_ref AND envelope.recipient_execution_ref=NEW.execution_ref
 AND envelope.recipient_work_item_generation=NEW.work_item_generation
 AND envelope.recipient_principal_ref=NEW.claimed_by)
BEGIN SELECT RAISE(ABORT,'sqlite.mailbox_recipient_mismatch'); END;
CREATE TRIGGER outbox_mailbox_retirement_guard BEFORE UPDATE OF retired_at ON outbox
WHEN NEW.retired_at IS NOT OLD.retired_at AND (
 OLD.retired_at IS NOT NULL OR NEW.retired_at IS NULL OR NOT EXISTS (
  SELECT 1 FROM mailbox_retirements retirement
  WHERE retirement.mailbox_message_ref=NEW.mailbox_message_ref AND retirement.action_ref=NEW.ref
  AND retirement.recipient_execution_ref=NEW.execution_ref AND retirement.retired_at=NEW.retired_at))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_mailbox_retirement_invalid'); END;
CREATE TRIGGER outbox_governance_insert_guard BEFORE INSERT ON outbox
WHEN (NEW.governance_version=0 AND NEW.effect_intent_ref IS NOT NULL)
 OR (NEW.governance_version=1 AND (
  NEW.kind NOT IN ('launch_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
                   'stop_agent','prepare_workspace','commit_change','attest_test','integrate_change')
  OR NEW.effect_intent_ref IS NULL OR NOT EXISTS (
   SELECT 1 FROM effect_intents intent WHERE intent.ref=NEW.effect_intent_ref
   AND intent.action_ref=NEW.ref AND intent.action_kind=NEW.kind AND intent.goal_ref=NEW.goal_ref
   AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=NEW.execution_ref
   AND intent.plan_generation=NEW.plan_generation)))
 OR (NEW.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
     AND NOT EXISTS (
      SELECT 1 FROM agent_environment_lifecycles lifecycle
      WHERE lifecycle.execution_ref=NEW.execution_ref AND lifecycle.goal_ref=NEW.goal_ref
       AND lifecycle.work_item_ref=NEW.work_item_ref AND lifecycle.next_action_ref IS NULL
       AND lifecycle.ready_to_finalize=0 AND (
        (NEW.kind='quiesce_agent' AND lifecycle.token_state='active'
         AND lifecycle.claim_action_ref IS NULL AND lifecycle.attempt_ref IS NULL
         AND lifecycle.revision=1)
        OR
        (NEW.kind IN ('preserve_agent_environment','close_agent_environment')
         AND lifecycle.token_state=CASE NEW.kind
          WHEN 'preserve_agent_environment' THEN 'active' ELSE 'quiesced' END
         AND EXISTS (
          SELECT 1 FROM outbox prior
          JOIN effect_receipts receipt ON receipt.attempt_ref=lifecycle.attempt_ref
          JOIN action_consumption_receipts consumed
           ON consumed.action_ref=prior.ref AND consumed.effect_receipt_ref=receipt.ref
          WHERE prior.ref=lifecycle.claim_action_ref
           AND prior.kind=CASE NEW.kind
            WHEN 'preserve_agent_environment' THEN 'quiesce_agent'
            ELSE 'preserve_agent_environment' END
           AND prior.goal_ref=NEW.goal_ref AND prior.work_item_ref=NEW.work_item_ref
           AND prior.execution_ref=NEW.execution_ref AND prior.completed_at=consumed.consumed_at
           AND prior.governance_version=1 AND receipt.action_ref=prior.ref
           AND receipt.action_fence=lifecycle.claim_fence
           AND consumed.fence=lifecycle.claim_fence
           AND consumed.claim_token=lifecycle.claim_token
           AND consumed.worker_ref=lifecycle.claim_worker_ref
           AND consumed.delivery_attempt=lifecycle.claim_delivery_attempt
           AND consumed.outcome='completed' AND consumed.error_code='')))))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_effect_intent_invalid'); END;
CREATE TRIGGER outbox_governance_immutable BEFORE UPDATE OF governance_version,effect_intent_ref ON outbox
WHEN NEW.governance_version<>OLD.governance_version OR NEW.effect_intent_ref IS NOT OLD.effect_intent_ref
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_governance_immutable'); END;
CREATE TRIGGER outbox_recovery_effect_claim_guard
BEFORE UPDATE OF claim_token,claimed_by,claimed_until,delivery_attempt,fence,recovery_effect_attempt_ref ON outbox
WHEN (NEW.recovery_effect_attempt_ref IS NOT OLD.recovery_effect_attempt_ref AND NOT (
      NEW.recovery_effect_attempt_ref IS NOT NULL AND NEW.claim_token IS NOT NULL
      AND NEW.claim_token IS NOT OLD.claim_token AND NEW.claimed_by IS NOT NULL
      AND NEW.claimed_until IS NOT NULL AND NEW.delivery_attempt=OLD.delivery_attempt+1 AND NEW.fence>OLD.fence))
 OR (NEW.recovery_effect_attempt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_attempts attempt JOIN effect_intents intent ON intent.ref=attempt.intent_ref
  WHERE attempt.ref=NEW.recovery_effect_attempt_ref AND NEW.kind='launch_agent' AND NEW.governance_version=1
   AND NEW.effect_intent_ref=attempt.intent_ref AND attempt.action_ref=NEW.ref AND attempt.action_fence<NEW.fence
   AND attempt.claim_lease_until IS NOT NULL AND intent.kind='agent_launch' AND intent.action_ref=NEW.ref
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
   AND (SELECT COUNT(*) FROM effect_attempts peer
        WHERE peer.action_ref=attempt.action_ref AND peer.intent_ref=attempt.intent_ref
        AND NOT EXISTS (SELECT 1 FROM budget_settlements settled
                        WHERE settled.causal_attempt_ref=peer.ref))=1))
BEGIN SELECT RAISE(ABORT,'sqlite.outbox_recovery_effect_claim_invalid'); END;

CREATE TRIGGER effect_intents_immutable_update BEFORE UPDATE ON effect_intents
BEGIN SELECT RAISE(ABORT,'sqlite.effect_intent_immutable'); END;
CREATE TRIGGER effect_intents_immutable_delete BEFORE DELETE ON effect_intents
BEGIN SELECT RAISE(ABORT,'sqlite.effect_intent_immutable'); END;
CREATE TRIGGER effect_receipts_immutable_update BEFORE UPDATE ON effect_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.effect_receipt_immutable'); END;
CREATE TRIGGER effect_receipts_immutable_delete BEFORE DELETE ON effect_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.effect_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_update BEFORE UPDATE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_delete BEFORE DELETE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_immutable'); END;

DROP TRIGGER effect_attempts_causal_guard;
CREATE TRIGGER effect_attempts_causal_guard BEFORE INSERT ON effect_attempts
WHEN NOT EXISTS (
 SELECT 1 FROM effect_intents intent
 JOIN effect_approvals approval ON approval.ref=NEW.approval_ref AND approval.intent_ref=intent.ref
 JOIN outbox action ON action.ref=NEW.action_ref
 JOIN executions execution ON execution.ref=NEW.execution_ref
 WHERE intent.ref=NEW.intent_ref AND intent.digest=NEW.intent_digest AND approval.intent_digest=intent.digest
  AND approval.decision='approved' AND action.effect_intent_ref=intent.ref AND action.fence=NEW.action_fence
  AND intent.project_ref=NEW.project_ref AND intent.goal_ref=NEW.goal_ref
  AND intent.work_item_ref=NEW.work_item_ref AND intent.execution_ref=NEW.execution_ref
  AND intent.plan_generation=NEW.plan_generation AND intent.app_spec_generation=NEW.app_spec_generation
  AND intent.spec_hash=NEW.spec_hash AND intent.actor_ref=NEW.actor_ref
  AND NEW.started_at>=intent.created_at AND NEW.started_at>=approval.decided_at
  AND (approval.source<>'explicit_decision' OR NEW.started_at<approval.expires_at)
  AND NEW.started_at<action.claimed_until
  AND ((intent.kind='agent_launch' AND execution.state='dispatching'
        AND execution.governance_version=1 AND execution.effect_intent_ref=intent.ref
        AND execution.budget_reservation_ref IS NOT NULL
        AND EXISTS(SELECT 1 FROM budget_reservations reservation
                   WHERE reservation.ref=execution.budget_reservation_ref AND reservation.reserved_at<=NEW.started_at))
    OR (intent.kind='agent_stop' AND execution.state='running')
    OR (intent.kind='prepare_workspace' AND execution.state='queued')
    OR (intent.kind='commit_change' AND execution.state='awaiting_commit')
    OR (intent.kind='attest_test' AND execution.state='awaiting_attestation')
    OR (intent.kind='integrate_change' AND execution.state='awaiting_integration')
    OR (intent.kind IN ('agent_quiesce','agent_environment_preserve','agent_environment_close') AND execution.state='running'
        AND EXISTS(SELECT 1 FROM agent_environment_lifecycles lifecycle
                   WHERE lifecycle.execution_ref=NEW.execution_ref AND lifecycle.goal_ref=NEW.goal_ref
                   AND lifecycle.work_item_ref=NEW.work_item_ref
                   AND ((intent.kind='agent_quiesce' AND lifecycle.token_state='active')
                     OR (intent.kind='agent_environment_preserve' AND lifecycle.token_state='quiesced')
                     OR (intent.kind='agent_environment_close' AND lifecycle.token_state='preserved')))))
  AND (intent.kind IN ('agent_quiesce','agent_environment_preserve','agent_environment_close') OR EXISTS(
   SELECT 1 FROM events event WHERE event.goal_ref=NEW.goal_ref AND event.work_item_ref=NEW.work_item_ref
    AND event.execution_ref=NEW.execution_ref
    AND event.kind=CASE intent.kind WHEN 'agent_launch' THEN 'execution.dispatching'
     WHEN 'agent_stop' THEN 'execution.accepted' WHEN 'prepare_workspace' THEN 'execution.queued'
     WHEN 'commit_change' THEN 'execution.output_ready' WHEN 'attest_test' THEN 'change.committed'
     WHEN 'integrate_change' THEN 'test_attestation.passed' END
    AND event.occurred_at<=NEW.started_at)))
BEGIN SELECT RAISE(ABORT,'sqlite.effect_attempt_causal_invalid'); END;

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
       OR NEW.confirmed_at<attempt.claim_lease_until)
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

CREATE TRIGGER action_consumption_receipt_guard BEFORE INSERT ON action_consumption_receipts
WHEN NOT EXISTS (
 SELECT 1 FROM outbox action WHERE action.ref=NEW.action_ref AND action.kind=NEW.kind
  AND action.goal_ref=NEW.goal_ref AND action.work_item_ref=NEW.work_item_ref
  AND action.execution_ref=NEW.execution_ref AND action.change_ref=NEW.change_ref
  AND action.plan_generation=NEW.plan_generation AND action.work_item_generation=NEW.work_item_generation
  AND action.mailbox_message_ref IS NEW.mailbox_message_ref AND action.last_error_code=NEW.error_code
  AND action.completed_at=NEW.consumed_at AND ((NEW.outcome='completed' AND action.quarantined_at IS NULL)
   OR (NEW.outcome='quarantined' AND action.quarantined_at=NEW.consumed_at))
  AND ((NEW.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
        AND EXISTS(SELECT 1 FROM agent_environment_lifecycles lifecycle
                   WHERE lifecycle.execution_ref=NEW.execution_ref AND lifecycle.claim_action_ref=NEW.action_ref
                    AND lifecycle.claim_token=NEW.claim_token AND lifecycle.claim_worker_ref=NEW.worker_ref
                    AND lifecycle.claim_delivery_attempt=NEW.delivery_attempt AND lifecycle.claim_fence=NEW.fence
                    AND lifecycle.attempt_ref IN (SELECT attempt.ref FROM effect_attempts attempt
                                                  WHERE attempt.action_ref=NEW.action_ref
                                                   AND attempt.action_fence=NEW.fence)))
    OR (NEW.kind NOT IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
        AND action.fence=NEW.fence AND action.delivery_attempt=NEW.delivery_attempt
        AND NEW.consumed_at<action.claimed_until AND action.claim_token=NEW.claim_token
        AND action.claimed_by=NEW.worker_ref)))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_receipt_invalid'); END;

CREATE TRIGGER action_consumption_effect_receipt_guard BEFORE INSERT ON action_consumption_receipts
WHEN NEW.governance_version=1 AND (
 (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt JOIN outbox action ON action.ref=NEW.action_ref
  WHERE receipt.ref=NEW.effect_receipt_ref AND receipt.action_ref=NEW.action_ref
   AND ((NEW.kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')
         AND receipt.action_fence=NEW.fence AND receipt.confirmed_at<=NEW.consumed_at
         AND EXISTS(SELECT 1 FROM agent_environment_lifecycles lifecycle
                    WHERE lifecycle.execution_ref=NEW.execution_ref
                     AND lifecycle.attempt_ref=receipt.attempt_ref
                     AND lifecycle.claim_token=NEW.claim_token))
    OR (action.recovery_effect_attempt_ref IS NULL AND receipt.action_fence=NEW.fence
        AND receipt.confirmed_at=NEW.consumed_at)
    OR (NEW.kind='launch_agent' AND action.recovery_effect_attempt_ref=receipt.attempt_ref
        AND receipt.action_fence<NEW.fence AND NOT EXISTS (
         SELECT 1 FROM effect_attempts current
         WHERE current.action_ref=NEW.action_ref AND current.action_fence=NEW.fence)))))
 OR (NEW.kind IN ('launch_agent','quiesce_agent','preserve_agent_environment','close_agent_environment',
                  'prepare_workspace','commit_change','attest_test','integrate_change')
     AND NEW.outcome='completed' AND NEW.error_code='' AND NEW.effect_receipt_ref IS NULL)
 OR (NEW.kind='stop_agent' AND NEW.outcome='completed' AND NEW.error_code=''
     AND NEW.effect_receipt_ref IS NULL AND EXISTS (
      SELECT 1 FROM effect_attempts attempt
      WHERE attempt.action_ref=NEW.action_ref AND attempt.action_fence=NEW.fence)))
BEGIN SELECT RAISE(ABORT,'sqlite.action_consumption_effect_receipt_invalid'); END;

CREATE TRIGGER agent_environment_lifecycles_identity_immutable
BEFORE UPDATE OF execution_ref,goal_ref,work_item_ref,snapshot_schema,launch_receipt_ref
ON agent_environment_lifecycles
BEGIN SELECT RAISE(ABORT,'sqlite.agent_environment_lifecycle_identity_immutable'); END;
CREATE TRIGGER agent_environment_lifecycles_revision_guard
BEFORE UPDATE ON agent_environment_lifecycles
WHEN NEW.revision<>OLD.revision+1 OR NEW.recorded_at<OLD.recorded_at
BEGIN SELECT RAISE(ABORT,'sqlite.agent_environment_lifecycle_revision_invalid'); END;
CREATE TRIGGER agent_environment_lifecycles_immutable_delete
BEFORE DELETE ON agent_environment_lifecycles
BEGIN SELECT RAISE(ABORT,'sqlite.agent_environment_lifecycle_immutable'); END;
