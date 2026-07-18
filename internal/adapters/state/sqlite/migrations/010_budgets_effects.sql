DROP TRIGGER authorization_receipts_immutable_update;
DROP TRIGGER authorization_receipts_immutable_delete;
DROP INDEX authorization_receipts_project_idx;
CREATE TABLE authorization_receipts_v10 (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL CHECK (length(trim(project_ref)) > 0),
    permission TEXT NOT NULL CHECK (permission IN (
        'project.hierarchy.manage', 'project.membership.manage',
        'goals.create', 'goals.amend', 'goals.get', 'goals.list', 'goals.direct',
        'budgets.manage', 'effects.approve', 'artifacts.read', 'project.status'
    )),
    resource_ref TEXT NOT NULL CHECK (length(trim(resource_ref)) > 0),
    requested_at INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('allowed', 'denied')),
    role TEXT NOT NULL DEFAULT '' CHECK (role IN (
        '', 'platform_admin', 'project_owner', 'project_admin', 'contributor',
        'reviewer', 'operator', 'viewer'
    )),
    membership_revision INTEGER NOT NULL CHECK (membership_revision >= 0),
    reason_code TEXT NOT NULL CHECK (length(trim(reason_code)) > 0),
    decided_at INTEGER NOT NULL,
    recorded_at INTEGER NOT NULL,
    UNIQUE(principal_ref, request_ref),
    CHECK (decided_at >= requested_at),
    CHECK (recorded_at >= decided_at),
    CHECK (
        (outcome = 'allowed' AND role <> ''
            AND (role = 'platform_admin' OR membership_revision > 0))
        OR outcome = 'denied'
    )
) STRICT;
INSERT INTO authorization_receipts_v10(
    ref, request_ref, request_fingerprint, principal_ref, project_ref,
    permission, resource_ref, requested_at, outcome, role,
    membership_revision, reason_code, decided_at, recorded_at
)
SELECT ref, request_ref, request_fingerprint, principal_ref, project_ref,
       permission, resource_ref, requested_at, outcome, role,
       membership_revision, reason_code, decided_at, recorded_at
FROM authorization_receipts;
PRAGMA legacy_alter_table = ON;
DROP TABLE authorization_receipts;
ALTER TABLE authorization_receipts_v10 RENAME TO authorization_receipts;
PRAGMA legacy_alter_table = OFF;
CREATE INDEX authorization_receipts_project_idx
    ON authorization_receipts(project_ref, recorded_at, ref);
CREATE UNIQUE INDEX authorization_receipts_mailbox_scope_idx
    ON authorization_receipts(ref, principal_ref, project_ref);

CREATE TRIGGER authorization_receipts_immutable_update
BEFORE UPDATE ON authorization_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.authorization_receipt_immutable'); END;

CREATE TRIGGER authorization_receipts_immutable_delete
BEFORE DELETE ON authorization_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.authorization_receipt_immutable'); END;

ALTER TABLE work_items ADD COLUMN governance_version INTEGER NOT NULL DEFAULT 0
    CHECK (governance_version IN (0, 1));
ALTER TABLE work_items ADD COLUMN budget_demand_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE work_items ADD COLUMN budget_tokens INTEGER NOT NULL DEFAULT 0 CHECK (budget_tokens >= 0);
ALTER TABLE work_items ADD COLUMN budget_money_micros INTEGER NOT NULL DEFAULT 0 CHECK (budget_money_micros >= 0);
ALTER TABLE work_items ADD COLUMN budget_currency TEXT NOT NULL DEFAULT '';
ALTER TABLE work_items ADD COLUMN budget_active_time_ns INTEGER NOT NULL DEFAULT 0 CHECK (budget_active_time_ns >= 0);
ALTER TABLE work_items ADD COLUMN budget_process_slots INTEGER NOT NULL DEFAULT 0 CHECK (budget_process_slots >= 0);
ALTER TABLE work_items ADD COLUMN budget_disk_bytes INTEGER NOT NULL DEFAULT 0 CHECK (budget_disk_bytes >= 0);
ALTER TABLE work_items ADD COLUMN security_criticality TEXT NOT NULL DEFAULT 'normal'
    CHECK (security_criticality IN ('normal', 'sensitive', 'critical'));
ALTER TABLE work_items ADD COLUMN reasoning_effort TEXT NOT NULL DEFAULT 'medium'
    CHECK (reasoning_effort IN ('low', 'medium', 'high', 'xhigh'));

CREATE TRIGGER work_items_governance_insert_guard
BEFORE INSERT ON work_items
WHEN NEW.governance_version = 1 AND (
    length(trim(NEW.budget_demand_ref)) = 0
    OR (NEW.budget_money_micros > 0 AND length(NEW.budget_currency) <> 3)
    OR (NEW.budget_currency <> '' AND (
        length(NEW.budget_currency) <> 3 OR NEW.budget_currency GLOB '*[^A-Z]*'
    ))
)
BEGIN SELECT RAISE(ABORT, 'sqlite.work_item_governance_invalid'); END;

CREATE TRIGGER work_items_governance_update_guard
BEFORE UPDATE OF governance_version, budget_demand_ref, budget_tokens,
    budget_money_micros, budget_currency, budget_active_time_ns,
    budget_process_slots, budget_disk_bytes, security_criticality,
    reasoning_effort ON work_items
WHEN NEW.governance_version <> OLD.governance_version
  OR NEW.budget_demand_ref <> OLD.budget_demand_ref
  OR NEW.budget_tokens <> OLD.budget_tokens
  OR NEW.budget_money_micros <> OLD.budget_money_micros
  OR NEW.budget_currency <> OLD.budget_currency
  OR NEW.budget_active_time_ns <> OLD.budget_active_time_ns
  OR NEW.budget_process_slots <> OLD.budget_process_slots
  OR NEW.budget_disk_bytes <> OLD.budget_disk_bytes
  OR NEW.security_criticality <> OLD.security_criticality
  OR NEW.reasoning_effort <> OLD.reasoning_effort
BEGIN SELECT RAISE(ABORT, 'sqlite.work_item_governance_immutable'); END;

ALTER TABLE executions ADD COLUMN governance_version INTEGER NOT NULL DEFAULT 0
    CHECK (governance_version IN (0, 1));
ALTER TABLE executions ADD COLUMN budget_reservation_ref TEXT;
ALTER TABLE executions ADD COLUMN effect_intent_ref TEXT;
ALTER TABLE executions ADD COLUMN launch_receipt_ref TEXT;

ALTER TABLE outbox ADD COLUMN governance_version INTEGER NOT NULL DEFAULT 0
    CHECK (governance_version IN (0, 1));
ALTER TABLE outbox ADD COLUMN effect_intent_ref TEXT;

-- Historical launch/stop actions have no causal V15 approval. Preserve them
-- pending and auditable, but make the required reauthorization explicit.
UPDATE outbox
SET last_error_code = 'governance.legacy_reauthorization_required'
WHERE governance_version = 0
  AND kind IN ('launch_agent', 'stop_agent')
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;

CREATE TABLE budget_envelopes (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    subject_ref TEXT NOT NULL CHECK (length(trim(subject_ref)) > 0),
    scope TEXT NOT NULL CHECK (scope IN ('deployment', 'project', 'goal')),
    tokens INTEGER NOT NULL CHECK (tokens >= 0),
    money_micros INTEGER NOT NULL CHECK (money_micros >= 0),
    currency TEXT NOT NULL DEFAULT '' CHECK (
        (currency = '' OR (length(currency) = 3 AND currency NOT GLOB '*[^A-Z]*'))
        AND (money_micros = 0 OR currency <> '')
    ),
    active_time_ns INTEGER NOT NULL CHECK (active_time_ns >= 0),
    process_slots INTEGER NOT NULL CHECK (process_slots >= 0),
    disk_bytes INTEGER NOT NULL CHECK (disk_bytes >= 0),
    revision INTEGER NOT NULL CHECK (revision > 0),
    policy_hash TEXT NOT NULL CHECK (
        length(policy_hash) = 64 AND policy_hash NOT GLOB '*[^0-9a-f]*'
    ),
    created_at INTEGER NOT NULL,
    UNIQUE(scope, subject_ref, policy_hash, revision)
) STRICT;

CREATE INDEX budget_envelopes_subject_idx
    ON budget_envelopes(scope, subject_ref, revision DESC);

CREATE TABLE work_item_authorities (
    work_item_ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL,
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    permission TEXT NOT NULL CHECK (permission IN ('goals.create', 'goals.direct')),
    source TEXT NOT NULL CHECK (source IN ('goal_confirmation', 'director_decision')),
    authorization_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    recorded_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT
) STRICT;

CREATE INDEX work_item_authorities_goal_idx ON work_item_authorities(goal_ref, work_item_ref);

CREATE TABLE effect_intents (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (
        length(request_fingerprint) = 64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'
    ),
    action_ref TEXT NOT NULL UNIQUE CHECK (length(trim(action_ref)) > 0),
    action_kind TEXT NOT NULL CHECK (action_kind IN ('launch_agent', 'stop_agent')),
    kind TEXT NOT NULL CHECK (kind IN ('agent_launch', 'agent_stop')),
    project_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    actor_ref TEXT NOT NULL CHECK (length(trim(actor_ref)) > 0),
    proposed_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    permission TEXT NOT NULL CHECK (permission IN ('goals.create', 'goals.direct')),
    authority_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    demand_ref TEXT NOT NULL CHECK (length(trim(demand_ref)) > 0),
    demand_tokens INTEGER NOT NULL CHECK (demand_tokens >= 0),
    demand_money_micros INTEGER NOT NULL CHECK (demand_money_micros >= 0),
    demand_currency TEXT NOT NULL DEFAULT '' CHECK (
        (demand_currency = '' OR (length(demand_currency) = 3 AND demand_currency NOT GLOB '*[^A-Z]*'))
        AND (demand_money_micros = 0 OR demand_currency <> '')
    ),
    demand_active_time_ns INTEGER NOT NULL CHECK (demand_active_time_ns >= 0),
    demand_process_slots INTEGER NOT NULL CHECK (demand_process_slots >= 0),
    demand_disk_bytes INTEGER NOT NULL CHECK (demand_disk_bytes >= 0),
    security_criticality TEXT NOT NULL CHECK (security_criticality IN ('normal', 'sensitive', 'critical')),
    reasoning_effort TEXT NOT NULL CHECK (reasoning_effort IN ('low', 'medium', 'high', 'xhigh')),
    policy_hash TEXT NOT NULL CHECK (length(policy_hash) = 64 AND policy_hash NOT GLOB '*[^0-9a-f]*'),
	policy_revision INTEGER NOT NULL CHECK (policy_revision > 0),
	quota_retry_delay_ns INTEGER NOT NULL CHECK (quota_retry_delay_ns > 0),
	approval_ttl_ns INTEGER NOT NULL CHECK (approval_ttl_ns > 0),
	target_digest TEXT NOT NULL CHECK (length(target_digest) = 64 AND target_digest NOT GLOB '*[^0-9a-f]*'),
	idempotency_key TEXT NOT NULL UNIQUE CHECK (length(trim(idempotency_key)) > 0),
    created_at INTEGER NOT NULL,
    digest TEXT NOT NULL UNIQUE CHECK (length(digest) = 64 AND digest NOT GLOB '*[^0-9a-f]*'),
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    CHECK ((kind = 'agent_launch') = (action_kind = 'launch_agent'))
) STRICT;

CREATE INDEX effect_intents_goal_idx ON effect_intents(goal_ref, created_at, ref);

CREATE TABLE effect_approvals (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (
        length(request_fingerprint) = 64 AND request_fingerprint NOT GLOB '*[^0-9a-f]*'
    ),
    intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
    intent_digest TEXT NOT NULL CHECK (length(intent_digest) = 64 AND intent_digest NOT GLOB '*[^0-9a-f]*'),
    project_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    actor_ref TEXT NOT NULL CHECK (length(trim(actor_ref)) > 0),
    proposed_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    decided_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    decision TEXT NOT NULL CHECK (decision IN ('approved', 'denied')),
    source TEXT NOT NULL CHECK (source IN ('goal_confirmation', 'director_decision', 'explicit_decision')),
    security_criticality TEXT NOT NULL CHECK (security_criticality IN ('normal', 'sensitive', 'critical')),
	policy_hash TEXT NOT NULL CHECK (length(policy_hash) = 64 AND policy_hash NOT GLOB '*[^0-9a-f]*'),
	policy_revision INTEGER NOT NULL CHECK (policy_revision > 0),
	target_digest TEXT NOT NULL CHECK (length(target_digest) = 64 AND target_digest NOT GLOB '*[^0-9a-f]*'),
	reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    idempotency_key TEXT NOT NULL CHECK (length(trim(idempotency_key)) > 0),
    authorization_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    decided_at INTEGER NOT NULL,
    expires_at INTEGER,
    CHECK ((decision = 'approved' AND source = 'explicit_decision' AND expires_at > decided_at)
        OR (decision = 'approved' AND source <> 'explicit_decision' AND expires_at IS NULL)
        OR (decision = 'denied' AND source = 'explicit_decision' AND expires_at IS NULL))
) STRICT;

CREATE INDEX effect_approvals_live_idx
    ON effect_approvals(intent_ref, decision, expires_at DESC, decided_at DESC, ref);

CREATE TABLE budget_reservations (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    demand_ref TEXT NOT NULL CHECK (length(trim(demand_ref)) > 0),
    action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT,
    effect_intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    work_item_generation INTEGER NOT NULL CHECK (work_item_generation > 0),
    fence INTEGER NOT NULL CHECK (fence > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    policy_hash TEXT NOT NULL CHECK (length(policy_hash) = 64 AND policy_hash NOT GLOB '*[^0-9a-f]*'),
    tokens INTEGER NOT NULL CHECK (tokens >= 0),
    money_micros INTEGER NOT NULL CHECK (money_micros >= 0),
    currency TEXT NOT NULL DEFAULT '' CHECK (
        (currency = '' OR (length(currency) = 3 AND currency NOT GLOB '*[^A-Z]*'))
        AND (money_micros = 0 OR currency <> '')
    ),
    active_time_ns INTEGER NOT NULL CHECK (active_time_ns >= 0),
    process_slots INTEGER NOT NULL CHECK (process_slots >= 0),
    disk_bytes INTEGER NOT NULL CHECK (disk_bytes >= 0),
    reserved_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT
) STRICT;

CREATE INDEX budget_reservations_scope_idx
    ON budget_reservations(project_ref, goal_ref, reserved_at, ref);

CREATE TABLE budget_settlements (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    reservation_ref TEXT NOT NULL UNIQUE REFERENCES budget_reservations(ref) ON DELETE RESTRICT,
    reserved_tokens INTEGER NOT NULL CHECK (reserved_tokens >= 0),
    reserved_money_micros INTEGER NOT NULL CHECK (reserved_money_micros >= 0),
    reserved_currency TEXT NOT NULL DEFAULT '',
    reserved_active_time_ns INTEGER NOT NULL CHECK (reserved_active_time_ns >= 0),
    reserved_process_slots INTEGER NOT NULL CHECK (reserved_process_slots >= 0),
    reserved_disk_bytes INTEGER NOT NULL CHECK (reserved_disk_bytes >= 0),
    observed_tokens INTEGER NOT NULL CHECK (observed_tokens >= 0),
    observed_money_micros INTEGER NOT NULL CHECK (observed_money_micros >= 0),
    observed_currency TEXT NOT NULL DEFAULT '',
    observed_active_time_ns INTEGER NOT NULL CHECK (observed_active_time_ns >= 0),
    observed_process_slots INTEGER NOT NULL CHECK (observed_process_slots >= 0),
    observed_disk_bytes INTEGER NOT NULL CHECK (observed_disk_bytes >= 0),
    observed_known INTEGER NOT NULL CHECK (observed_known BETWEEN 0 AND 31),
    observed_quality TEXT NOT NULL CHECK (observed_quality IN ('unknown', 'estimated', 'measured', 'exact')),
    charged_tokens INTEGER NOT NULL CHECK (charged_tokens >= 0),
    charged_money_micros INTEGER NOT NULL CHECK (charged_money_micros >= 0),
    charged_currency TEXT NOT NULL DEFAULT '',
    charged_active_time_ns INTEGER NOT NULL CHECK (charged_active_time_ns >= 0),
    charged_process_slots INTEGER NOT NULL CHECK (charged_process_slots >= 0),
    charged_disk_bytes INTEGER NOT NULL CHECK (charged_disk_bytes >= 0),
    released_tokens INTEGER NOT NULL CHECK (released_tokens >= 0),
    released_money_micros INTEGER NOT NULL CHECK (released_money_micros >= 0),
    released_currency TEXT NOT NULL DEFAULT '',
    released_active_time_ns INTEGER NOT NULL CHECK (released_active_time_ns >= 0),
    released_process_slots INTEGER NOT NULL CHECK (released_process_slots >= 0),
    released_disk_bytes INTEGER NOT NULL CHECK (released_disk_bytes >= 0),
    overrun_tokens INTEGER NOT NULL CHECK (overrun_tokens >= 0),
    overrun_money_micros INTEGER NOT NULL CHECK (overrun_money_micros >= 0),
    overrun_currency TEXT NOT NULL DEFAULT '',
    overrun_active_time_ns INTEGER NOT NULL CHECK (overrun_active_time_ns >= 0),
    overrun_process_slots INTEGER NOT NULL CHECK (overrun_process_slots >= 0),
    overrun_disk_bytes INTEGER NOT NULL CHECK (overrun_disk_bytes >= 0),
    settled_at INTEGER NOT NULL,
    CHECK ((observed_known = 0) = (observed_quality = 'unknown'))
) STRICT;

CREATE TABLE fairness_cursors (
    scope TEXT NOT NULL CHECK (scope IN ('project', 'goal')),
    subject_ref TEXT NOT NULL CHECK (length(trim(subject_ref)) > 0),
    ordinal INTEGER NOT NULL CHECK (ordinal > 0),
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(scope, subject_ref)
) STRICT;

CREATE TABLE effect_attempts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
    intent_digest TEXT NOT NULL CHECK (length(intent_digest) = 64 AND intent_digest NOT GLOB '*[^0-9a-f]*'),
    approval_ref TEXT NOT NULL REFERENCES effect_approvals(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    actor_ref TEXT NOT NULL CHECK (length(trim(actor_ref)) > 0),
    action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT,
    action_fence INTEGER NOT NULL CHECK (action_fence > 0),
    worker_ref TEXT NOT NULL CHECK (length(trim(worker_ref)) > 0),
    idempotency_key TEXT NOT NULL CHECK (length(trim(idempotency_key)) > 0),
    started_at INTEGER NOT NULL,
    UNIQUE(action_ref, action_fence)
) STRICT;

CREATE INDEX effect_attempts_goal_idx ON effect_attempts(goal_ref, started_at, ref);

CREATE TABLE effect_receipts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
    intent_digest TEXT NOT NULL CHECK (length(intent_digest) = 64 AND intent_digest NOT GLOB '*[^0-9a-f]*'),
    approval_ref TEXT NOT NULL REFERENCES effect_approvals(ref) ON DELETE RESTRICT,
    attempt_ref TEXT NOT NULL UNIQUE REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    actor_ref TEXT NOT NULL CHECK (length(trim(actor_ref)) > 0),
    action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT,
    action_fence INTEGER NOT NULL CHECK (action_fence > 0),
    idempotency_key TEXT NOT NULL UNIQUE CHECK (length(trim(idempotency_key)) > 0),
    external_ref TEXT NOT NULL CHECK (length(trim(external_ref)) > 0),
    status TEXT NOT NULL CHECK (status IN ('accepted','stopped','already_stopped','already_completed','already_failed')),
    usage_tokens INTEGER NOT NULL CHECK (usage_tokens >= 0),
    usage_money_micros INTEGER NOT NULL CHECK (usage_money_micros >= 0),
    usage_currency TEXT NOT NULL DEFAULT '',
    usage_active_time_ns INTEGER NOT NULL CHECK (usage_active_time_ns >= 0),
    usage_process_slots INTEGER NOT NULL CHECK (usage_process_slots >= 0),
    usage_disk_bytes INTEGER NOT NULL CHECK (usage_disk_bytes >= 0),
    usage_known INTEGER NOT NULL CHECK (usage_known BETWEEN 0 AND 31),
    usage_quality TEXT NOT NULL CHECK (usage_quality IN ('unknown', 'estimated', 'measured', 'exact')),
    confirmed_at INTEGER NOT NULL,
    UNIQUE(action_ref, action_fence),
    CHECK ((usage_known = 0) = (usage_quality = 'unknown'))
) STRICT;

CREATE INDEX effect_receipts_goal_idx ON effect_receipts(goal_ref, confirmed_at, ref);

DROP TRIGGER action_consumption_receipt_guard;
DROP TRIGGER action_consumption_receipts_immutable_update;
DROP TRIGGER action_consumption_receipts_immutable_delete;
DROP INDEX action_consumption_scheduler_fence_idx;
DROP INDEX action_consumption_mailbox_fence_idx;
PRAGMA legacy_alter_table = ON;
ALTER TABLE action_consumption_receipts RENAME TO action_consumption_receipts_v9;

CREATE TABLE action_consumption_receipts (
    action_ref TEXT PRIMARY KEY REFERENCES outbox(ref) ON DELETE RESTRICT,
    governance_version INTEGER NOT NULL DEFAULT 0 CHECK (governance_version IN (0, 1)),
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent', 'stop_agent', 'deliver_mailbox')),
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    work_item_generation INTEGER NOT NULL CHECK (work_item_generation > 0),
    mailbox_message_ref TEXT,
    fence INTEGER NOT NULL CHECK (fence > 0),
    delivery_attempt INTEGER NOT NULL CHECK (delivery_attempt > 0),
    claim_token TEXT NOT NULL UNIQUE,
    worker_ref TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('completed', 'quarantined')),
    error_code TEXT NOT NULL DEFAULT '',
    consumed_at INTEGER NOT NULL,
    effect_receipt_ref TEXT,
    legacy_effect_status TEXT,
    legacy_effect_confirmed_at INTEGER,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, mailbox_message_ref, plan_generation, work_item_ref,
        execution_ref, work_item_generation) REFERENCES mailbox_envelopes(
        goal_ref, ref, plan_generation, parent_work_item_ref,
        recipient_execution_ref, recipient_work_item_generation
    ) ON DELETE RESTRICT,
    CHECK ((kind IN ('launch_agent', 'observe_agent', 'stop_agent') AND mailbox_message_ref IS NULL)
        OR (kind = 'deliver_mailbox' AND mailbox_message_ref IS NOT NULL)),
    CHECK (governance_version = 0 OR kind <> 'launch_agent'
        OR outcome <> 'completed' OR error_code <> '' OR effect_receipt_ref IS NOT NULL),
    CHECK (governance_version = 0 OR (legacy_effect_status IS NULL AND legacy_effect_confirmed_at IS NULL))
) STRICT;

INSERT INTO action_consumption_receipts(
    action_ref, governance_version, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, mailbox_message_ref, fence,
    delivery_attempt, claim_token, worker_ref, outcome, error_code, consumed_at,
    effect_receipt_ref, legacy_effect_status, legacy_effect_confirmed_at
)
SELECT action_ref, 0, kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, mailbox_message_ref, fence,
       delivery_attempt, claim_token, worker_ref, outcome, error_code, consumed_at,
       effect_receipt_ref, effect_status, effect_confirmed_at
FROM action_consumption_receipts_v9;

DROP TABLE action_consumption_receipts_v9;
PRAGMA legacy_alter_table = OFF;

CREATE UNIQUE INDEX action_consumption_scheduler_fence_idx
    ON action_consumption_receipts(goal_ref, work_item_ref, fence)
    WHERE kind IN ('launch_agent', 'observe_agent', 'stop_agent');
CREATE UNIQUE INDEX action_consumption_mailbox_fence_idx
    ON action_consumption_receipts(mailbox_message_ref, fence)
    WHERE kind = 'deliver_mailbox';

CREATE TRIGGER outbox_governance_insert_guard
BEFORE INSERT ON outbox
WHEN NEW.governance_version = 1 AND (
    (NEW.kind IN ('launch_agent', 'stop_agent') AND (
        NEW.effect_intent_ref IS NULL OR NOT EXISTS (
            SELECT 1 FROM effect_intents intent
            WHERE intent.ref = NEW.effect_intent_ref AND intent.action_ref = NEW.ref
              AND intent.action_kind = NEW.kind AND intent.goal_ref = NEW.goal_ref
              AND intent.work_item_ref = NEW.work_item_ref
              AND intent.execution_ref = NEW.execution_ref
              AND intent.plan_generation = NEW.plan_generation
        )
    ))
    OR (NEW.kind NOT IN ('launch_agent', 'stop_agent') AND NEW.effect_intent_ref IS NOT NULL)
)
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_effect_intent_invalid'); END;

CREATE TRIGGER outbox_governance_immutable
BEFORE UPDATE OF governance_version, effect_intent_ref ON outbox
WHEN NEW.governance_version <> OLD.governance_version
  OR NEW.effect_intent_ref IS NOT OLD.effect_intent_ref
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_governance_immutable'); END;

CREATE TRIGGER effect_approvals_causal_guard
BEFORE INSERT ON effect_approvals
WHEN NOT EXISTS (
    SELECT 1 FROM effect_intents intent
    WHERE intent.ref = NEW.intent_ref AND intent.digest = NEW.intent_digest
      AND intent.project_ref = NEW.project_ref AND intent.goal_ref = NEW.goal_ref
      AND intent.work_item_ref = NEW.work_item_ref
      AND intent.execution_ref = NEW.execution_ref
      AND intent.plan_generation = NEW.plan_generation
      AND intent.app_spec_generation = NEW.app_spec_generation
      AND intent.spec_hash = NEW.spec_hash AND intent.actor_ref = NEW.actor_ref
      AND NEW.decided_at >= intent.created_at
      AND intent.proposed_by_ref = NEW.proposed_by_ref
      AND intent.security_criticality = NEW.security_criticality
	      AND intent.policy_hash = NEW.policy_hash AND intent.policy_revision = NEW.policy_revision
	      AND intent.target_digest = NEW.target_digest
      AND intent.idempotency_key = NEW.idempotency_key
)
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_approval_causal_invalid'); END;

CREATE TRIGGER budget_reservations_causal_guard
BEFORE INSERT ON budget_reservations
WHEN NOT EXISTS (
    SELECT 1 FROM outbox action JOIN effect_intents intent
      ON intent.ref = NEW.effect_intent_ref AND intent.action_ref = action.ref
    WHERE action.ref = NEW.action_ref AND action.kind = 'launch_agent'
      AND action.governance_version = 1 AND action.effect_intent_ref = intent.ref
      AND action.goal_ref = NEW.goal_ref AND action.work_item_ref = NEW.work_item_ref
      AND action.execution_ref = NEW.execution_ref
      AND action.plan_generation = NEW.plan_generation
      AND action.work_item_generation = NEW.work_item_generation
      AND intent.spec_hash = NEW.spec_hash AND intent.policy_hash = NEW.policy_hash
      AND (SELECT COUNT(*) FROM budget_envelopes envelope WHERE envelope.policy_hash=intent.policy_hash
        AND envelope.revision=intent.policy_revision AND (envelope.scope='deployment'
          OR (envelope.scope='project' AND envelope.subject_ref=intent.project_ref)
          OR (envelope.scope='goal' AND envelope.subject_ref=intent.goal_ref)))=3
)
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_reservation_causal_invalid'); END;
CREATE TRIGGER effect_attempts_causal_guard
BEFORE INSERT ON effect_attempts
WHEN NOT EXISTS (
    SELECT 1 FROM effect_intents intent JOIN effect_approvals approval
      ON approval.ref = NEW.approval_ref AND approval.intent_ref = intent.ref
    JOIN outbox action ON action.ref = NEW.action_ref
    JOIN executions execution ON execution.ref = NEW.execution_ref
    WHERE intent.ref = NEW.intent_ref AND intent.digest = NEW.intent_digest
      AND approval.intent_digest = intent.digest AND approval.decision = 'approved'
      AND action.effect_intent_ref = intent.ref AND action.fence = NEW.action_fence
      AND intent.project_ref = NEW.project_ref AND intent.goal_ref = NEW.goal_ref
      AND intent.work_item_ref = NEW.work_item_ref AND intent.execution_ref = NEW.execution_ref
      AND intent.plan_generation = NEW.plan_generation
      AND intent.app_spec_generation = NEW.app_spec_generation
      AND intent.spec_hash = NEW.spec_hash AND intent.actor_ref = NEW.actor_ref
      AND NEW.started_at >= intent.created_at AND NEW.started_at >= approval.decided_at
      AND (approval.source <> 'explicit_decision' OR NEW.started_at < approval.expires_at)
      AND NEW.started_at < action.claimed_until
      AND ((intent.kind='agent_launch' AND execution.state='dispatching' AND execution.governance_version=1
        AND execution.effect_intent_ref=intent.ref AND execution.budget_reservation_ref IS NOT NULL
        AND EXISTS(SELECT 1 FROM budget_reservations reservation WHERE reservation.ref=execution.budget_reservation_ref AND reservation.reserved_at<=NEW.started_at)) OR (intent.kind='agent_stop' AND execution.state='running'))
      AND EXISTS(SELECT 1 FROM events event WHERE event.goal_ref=NEW.goal_ref AND event.work_item_ref=NEW.work_item_ref AND event.execution_ref=NEW.execution_ref AND event.kind=CASE intent.kind WHEN 'agent_launch' THEN 'execution.dispatching' ELSE 'execution.accepted' END AND event.occurred_at<=NEW.started_at)
)
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_attempt_causal_invalid'); END;
CREATE TRIGGER effect_receipts_causal_guard
BEFORE INSERT ON effect_receipts
WHEN NOT EXISTS (
    SELECT 1 FROM effect_attempts attempt
    JOIN effect_intents intent ON intent.ref = attempt.intent_ref
    JOIN outbox action ON action.ref = attempt.action_ref
    WHERE attempt.ref = NEW.attempt_ref AND attempt.intent_ref = NEW.intent_ref
      AND attempt.intent_digest = NEW.intent_digest AND attempt.approval_ref = NEW.approval_ref
      AND attempt.project_ref = NEW.project_ref AND attempt.goal_ref = NEW.goal_ref
      AND attempt.work_item_ref = NEW.work_item_ref AND attempt.execution_ref = NEW.execution_ref
      AND attempt.plan_generation = NEW.plan_generation
      AND attempt.app_spec_generation = NEW.app_spec_generation
      AND attempt.spec_hash = NEW.spec_hash AND attempt.actor_ref = NEW.actor_ref
      AND attempt.action_ref = NEW.action_ref AND attempt.action_fence = NEW.action_fence
      AND attempt.idempotency_key = NEW.idempotency_key
      AND NEW.confirmed_at < action.claimed_until
      AND ((intent.kind = 'agent_launch' AND NEW.status = 'accepted')
        OR (intent.kind = 'agent_stop' AND NEW.status IN
          ('stopped','already_stopped','already_completed','already_failed')))
)
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_receipt_causal_invalid'); END;
CREATE TRIGGER budget_settlements_causal_guard
BEFORE INSERT ON budget_settlements
WHEN NEW.ref <> 'budget-settlement:' || NEW.reservation_ref
  OR NOT EXISTS (
      SELECT 1 FROM budget_reservations reservation
      WHERE reservation.ref = NEW.reservation_ref
        AND reservation.tokens = NEW.reserved_tokens
        AND reservation.money_micros = NEW.reserved_money_micros
        AND reservation.currency = NEW.reserved_currency
        AND reservation.active_time_ns = NEW.reserved_active_time_ns
        AND reservation.process_slots = NEW.reserved_process_slots
        AND reservation.disk_bytes = NEW.reserved_disk_bytes
        AND NEW.settled_at >= reservation.reserved_at
  )
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_settlement_causal_invalid'); END;
CREATE TRIGGER work_item_authorities_causal_guard
BEFORE INSERT ON work_item_authorities
WHEN NOT EXISTS (
    SELECT 1 FROM authorization_receipts authorization
    WHERE authorization.ref = NEW.authorization_receipt_ref
      AND authorization.principal_ref = NEW.principal_ref
      AND authorization.permission = NEW.permission
      AND authorization.outcome = 'allowed'
      AND (NEW.source = 'goal_confirmation') = (NEW.permission = 'goals.create')
)
BEGIN SELECT RAISE(ABORT, 'sqlite.work_item_authority_causal_invalid'); END;
CREATE TRIGGER action_consumption_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NOT EXISTS (
    SELECT 1 FROM outbox action
    WHERE action.ref = NEW.action_ref AND action.kind = NEW.kind
      AND action.goal_ref = NEW.goal_ref AND action.work_item_ref = NEW.work_item_ref
      AND action.execution_ref = NEW.execution_ref
      AND action.plan_generation = NEW.plan_generation
      AND action.work_item_generation = NEW.work_item_generation
      AND action.mailbox_message_ref IS NEW.mailbox_message_ref
      AND action.fence = NEW.fence AND action.delivery_attempt = NEW.delivery_attempt AND NEW.consumed_at < action.claimed_until
      AND action.claim_token = NEW.claim_token AND action.claimed_by = NEW.worker_ref
      AND action.last_error_code = NEW.error_code AND action.completed_at = NEW.consumed_at
      AND ((NEW.outcome = 'completed' AND action.quarantined_at IS NULL)
        OR (NEW.outcome = 'quarantined' AND action.quarantined_at = NEW.consumed_at))
)
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_invalid'); END;

CREATE TRIGGER action_consumption_effect_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NEW.governance_version = 1 AND (
    (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM effect_receipts receipt
        WHERE receipt.ref = NEW.effect_receipt_ref AND receipt.action_ref = NEW.action_ref
          AND receipt.action_fence = NEW.fence
    )) OR (NEW.kind = 'stop_agent' AND NEW.outcome = 'completed'
        AND NEW.error_code = '' AND NEW.effect_receipt_ref IS NULL AND EXISTS (
            SELECT 1 FROM effect_attempts attempt
            WHERE attempt.action_ref = NEW.action_ref AND attempt.action_fence = NEW.fence
        ))
)
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_effect_receipt_invalid'); END;

CREATE TRIGGER action_consumption_receipts_immutable_update
BEFORE UPDATE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_delete
BEFORE DELETE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable'); END;

CREATE TRIGGER budget_envelopes_immutable_update BEFORE UPDATE ON budget_envelopes
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_envelope_immutable'); END;
CREATE TRIGGER budget_envelopes_immutable_delete BEFORE DELETE ON budget_envelopes
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_envelope_immutable'); END;
CREATE TRIGGER work_item_authorities_immutable_update BEFORE UPDATE ON work_item_authorities
BEGIN SELECT RAISE(ABORT, 'sqlite.work_item_authority_immutable'); END;
CREATE TRIGGER work_item_authorities_immutable_delete BEFORE DELETE ON work_item_authorities
BEGIN SELECT RAISE(ABORT, 'sqlite.work_item_authority_immutable'); END;
CREATE TRIGGER effect_intents_immutable_update BEFORE UPDATE ON effect_intents
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_intent_immutable'); END;
CREATE TRIGGER effect_intents_immutable_delete BEFORE DELETE ON effect_intents
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_intent_immutable'); END;
CREATE TRIGGER effect_approvals_immutable_update BEFORE UPDATE ON effect_approvals
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_approval_immutable'); END;
CREATE TRIGGER effect_approvals_immutable_delete BEFORE DELETE ON effect_approvals
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_approval_immutable'); END;
CREATE TRIGGER budget_reservations_immutable_update BEFORE UPDATE ON budget_reservations
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_reservation_immutable'); END;
CREATE TRIGGER budget_reservations_immutable_delete BEFORE DELETE ON budget_reservations
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_reservation_immutable'); END;
CREATE TRIGGER budget_settlements_immutable_update BEFORE UPDATE ON budget_settlements
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_settlement_immutable'); END;
CREATE TRIGGER budget_settlements_immutable_delete BEFORE DELETE ON budget_settlements
BEGIN SELECT RAISE(ABORT, 'sqlite.budget_settlement_immutable'); END;
CREATE TRIGGER effect_attempts_immutable_update BEFORE UPDATE ON effect_attempts
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_attempt_immutable'); END;
CREATE TRIGGER effect_attempts_immutable_delete BEFORE DELETE ON effect_attempts
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_attempt_immutable'); END;
CREATE TRIGGER effect_receipts_immutable_update BEFORE UPDATE ON effect_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_receipt_immutable'); END;
CREATE TRIGGER effect_receipts_immutable_delete BEFORE DELETE ON effect_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_receipt_immutable'); END;
