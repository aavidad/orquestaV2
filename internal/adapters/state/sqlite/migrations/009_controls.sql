DROP TRIGGER mailbox_handoff_required_guard;
DROP TRIGGER mailbox_delivery_attempt_insert_guard;
DROP TRIGGER mailbox_delivery_attempt_progress_guard;
DROP TRIGGER mailbox_ack_guard;
DROP TRIGGER mailbox_retirement_guard;

CREATE TABLE goals_v9 (
    ref TEXT PRIMARY KEY,
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    requested_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    app_spec_ref TEXT NOT NULL UNIQUE REFERENCES app_specs(ref) ON DELETE RESTRICT,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    paused INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
    cancel_requested INTEGER NOT NULL DEFAULT 0 CHECK (cancel_requested IN (0, 1)),
    control_sequence INTEGER NOT NULL DEFAULT 0 CHECK (control_sequence >= 0),
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    closed_at INTEGER,
    plan_generation INTEGER NOT NULL CHECK (plan_generation >= 0),
    UNIQUE(requested_by_ref, project_ref, request_ref)
) STRICT;

INSERT INTO goals_v9(
    ref, request_ref, request_fingerprint, requested_by_ref, app_spec_ref,
    actor_ref, project_ref, state, revision, paused, cancel_requested,
    control_sequence, created_at, started_at, closed_at, plan_generation
)
SELECT ref, request_ref, request_fingerprint, requested_by_ref, app_spec_ref,
       actor_ref, project_ref, state, revision, 0, 0, 0,
       created_at, started_at, closed_at, plan_generation
FROM goals;

CREATE TABLE work_items_v9 (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals_v9(ref) ON DELETE CASCADE,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    objective TEXT NOT NULL,
    phase_key TEXT NOT NULL,
    role_key TEXT NOT NULL,
    parent_ref TEXT,
    output_contract TEXT NOT NULL CHECK (output_contract IN ('evidence_bundle', 'artifact', 'attestation')),
    skip_reason TEXT NOT NULL DEFAULT '' CHECK (skip_reason IN ('', 'dependency_failed', 'dependency_canceled')),
    interrupt_cause TEXT NOT NULL DEFAULT '' CHECK (interrupt_cause IN ('', 'execution_stopped', 'execution_failed')),
    rework_of TEXT,
    state TEXT NOT NULL CHECK (state IN (
        'pending', 'running', 'succeeded', 'failed', 'skipped',
        'interrupted', 'canceled', 'superseded'
    )),
    revision INTEGER NOT NULL CHECK (revision > 0),
    paused INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
    cancel_requested INTEGER NOT NULL DEFAULT 0 CHECK (cancel_requested IN (0, 1)),
    control_sequence INTEGER NOT NULL DEFAULT 0 CHECK (control_sequence >= 0),
    position INTEGER NOT NULL CHECK (position >= 0),
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    interrupted_at INTEGER,
    finished_at INTEGER,
    execution_ref TEXT,
    handoff_required INTEGER NOT NULL DEFAULT 0 CHECK (handoff_required IN (0, 1)),
    UNIQUE(goal_ref, ref),
    UNIQUE(goal_ref, position),
    FOREIGN KEY(goal_ref, phase_key) REFERENCES goal_phases(goal_ref, phase_key) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, parent_ref) REFERENCES work_items_v9(goal_ref, ref)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY(goal_ref, rework_of) REFERENCES work_items_v9(goal_ref, ref)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CHECK (parent_ref IS NULL OR parent_ref <> ref),
    CHECK (rework_of IS NULL OR rework_of <> ref),
    CHECK ((state = 'interrupted') = (interrupt_cause <> '' AND interrupted_at IS NOT NULL))
) STRICT;

INSERT INTO work_items_v9(
    ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
    parent_ref, output_contract, skip_reason, interrupt_cause, rework_of,
    state, revision, paused, cancel_requested, control_sequence, position,
    created_at, started_at, interrupted_at, finished_at, execution_ref,
    handoff_required
)
SELECT ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
       parent_ref, output_contract, skip_reason, '', NULL, state, revision,
       0, 0, 0, position, created_at, started_at, NULL, finished_at,
       execution_ref, handoff_required
FROM work_items;

CREATE TABLE executions_v9 (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals_v9(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    attempt_no INTEGER NOT NULL CHECK (attempt_no > 0),
    max_execution_attempts INTEGER NOT NULL CHECK (max_execution_attempts > 0),
    replaces_execution_ref TEXT,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    state TEXT NOT NULL CHECK (state IN (
        'queued', 'dispatching', 'running', 'succeeded', 'failed', 'canceled', 'stopped'
    )),
    artifact_media_type TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    max_output_bytes INTEGER NOT NULL CHECK (max_output_bytes > 0),
    provider_ref TEXT NOT NULL DEFAULT '',
    model_ref TEXT NOT NULL DEFAULT '',
    agent_ref TEXT NOT NULL DEFAULT '',
    external_ref TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    deadline_at INTEGER,
    started_at INTEGER,
    provider_accepted_at INTEGER,
    last_observed_at INTEGER,
    provider_observed_at INTEGER,
    finished_at INTEGER,
    failure_code TEXT NOT NULL DEFAULT '',
    recipient_mailbox_retired INTEGER NOT NULL DEFAULT 0
        CHECK (recipient_mailbox_retired IN (0, 1)),
    UNIQUE(goal_ref, ref),
    UNIQUE(goal_ref, work_item_ref, ref),
    UNIQUE(goal_ref, work_item_ref, attempt_no),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items_v9(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, work_item_ref, replaces_execution_ref)
        REFERENCES executions_v9(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    CHECK (
        (attempt_no = 1 AND replaces_execution_ref IS NULL)
        OR (attempt_no > 1 AND replaces_execution_ref IS NOT NULL)
    ),
    CHECK (attempt_no <= max_execution_attempts)
) STRICT;

INSERT INTO executions_v9(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code, recipient_mailbox_retired
)
SELECT ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
       replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
       state, artifact_media_type, idempotency_key, max_output_bytes,
       provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
       started_at, provider_accepted_at, last_observed_at, provider_observed_at,
       finished_at, failure_code,
       EXISTS (SELECT 1 FROM mailbox_retirements retirement
               WHERE retirement.recipient_execution_ref = executions.ref)
FROM executions;

CREATE TABLE outbox_v9 (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent', 'stop_agent', 'deliver_mailbox')),
    goal_ref TEXT NOT NULL REFERENCES goals_v9(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    control_ref TEXT,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    work_item_generation INTEGER NOT NULL CHECK (work_item_generation > 0),
    mailbox_message_ref TEXT,
    available_at INTEGER NOT NULL,
    claim_token TEXT,
    claimed_by TEXT,
    claimed_until INTEGER,
    delivery_attempt INTEGER NOT NULL DEFAULT 0 CHECK (delivery_attempt >= 0),
    fence INTEGER NOT NULL DEFAULT 0 CHECK (fence >= 0),
    completed_at INTEGER,
    retired_at INTEGER,
    quarantined_at INTEGER,
    last_error_code TEXT NOT NULL DEFAULT '',
    FOREIGN KEY(goal_ref, work_item_ref)
        REFERENCES work_items_v9(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions_v9(goal_ref, work_item_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(control_ref) REFERENCES controls(ref) ON DELETE RESTRICT,
    FOREIGN KEY(
        goal_ref, mailbox_message_ref, plan_generation, work_item_ref,
        execution_ref, work_item_generation
    ) REFERENCES mailbox_envelopes(
        goal_ref, ref, plan_generation, parent_work_item_ref,
        recipient_execution_ref, recipient_work_item_generation
    ) ON DELETE RESTRICT,
    CHECK (
        (kind IN ('launch_agent', 'observe_agent') AND mailbox_message_ref IS NULL AND control_ref IS NULL)
        OR (kind = 'stop_agent' AND mailbox_message_ref IS NULL AND control_ref IS NOT NULL)
        OR (kind = 'deliver_mailbox' AND mailbox_message_ref IS NOT NULL AND control_ref IS NULL)
    ),
    CHECK (
        (claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL)
        OR
        (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL
            AND delivery_attempt > 0 AND fence > 0)
    ),
    CHECK (quarantined_at IS NULL OR completed_at IS NOT NULL),
    CHECK (retired_at IS NULL OR kind = 'deliver_mailbox')
) STRICT;

INSERT INTO outbox_v9(
    ref, kind, goal_ref, work_item_ref, execution_ref, control_ref,
    plan_generation, work_item_generation, mailbox_message_ref, available_at,
    claim_token, claimed_by, claimed_until, delivery_attempt, fence,
    completed_at, retired_at, quarantined_at, last_error_code
)
SELECT ref, kind, goal_ref, work_item_ref, execution_ref, NULL,
       plan_generation, work_item_generation, mailbox_message_ref, available_at,
       claim_token, claimed_by, claimed_until, delivery_attempt, fence,
       completed_at, retired_at, quarantined_at, last_error_code
FROM outbox;

CREATE TABLE action_consumption_receipts_v9 (
    action_ref TEXT PRIMARY KEY REFERENCES outbox_v9(ref) ON DELETE RESTRICT,
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
    effect_status TEXT CHECK (effect_status IS NULL OR effect_status IN (
        'stopped', 'already_stopped', 'already_completed', 'already_failed'
    )),
    effect_confirmed_at INTEGER,
    FOREIGN KEY(goal_ref, work_item_ref)
        REFERENCES work_items_v9(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions_v9(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(
        goal_ref, mailbox_message_ref, plan_generation, work_item_ref,
        execution_ref, work_item_generation
    ) REFERENCES mailbox_envelopes(
        goal_ref, ref, plan_generation, parent_work_item_ref,
        recipient_execution_ref, recipient_work_item_generation
    ) ON DELETE RESTRICT,
    CHECK (
        (kind IN ('launch_agent', 'observe_agent', 'stop_agent') AND mailbox_message_ref IS NULL)
        OR (kind = 'deliver_mailbox' AND mailbox_message_ref IS NOT NULL)
    ),
    CHECK (
        (effect_receipt_ref IS NULL AND effect_status IS NULL AND effect_confirmed_at IS NULL)
        OR (kind = 'stop_agent' AND outcome = 'completed' AND error_code = ''
            AND length(trim(effect_receipt_ref)) > 0
            AND effect_status IS NOT NULL AND effect_confirmed_at = consumed_at)
    ),
    CHECK (kind <> 'stop_agent' OR outcome <> 'completed' OR error_code <> ''
           OR effect_receipt_ref IS NOT NULL)
) STRICT;

INSERT INTO action_consumption_receipts_v9(
    action_ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, mailbox_message_ref, fence,
    delivery_attempt, claim_token, worker_ref, outcome, error_code, consumed_at
)
SELECT action_ref, kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, mailbox_message_ref, fence,
       delivery_attempt, claim_token, worker_ref, outcome, error_code, consumed_at
FROM action_consumption_receipts;

CREATE TABLE director_decisions_v9 (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    goal_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    lease_fence INTEGER NOT NULL CHECK (lease_fence > 0),
    source_goal_revision INTEGER NOT NULL CHECK (source_goal_revision > 0),
    source_plan_generation INTEGER NOT NULL CHECK (source_plan_generation >= 0),
    cause TEXT NOT NULL DEFAULT '' CHECK (cause IN ('', 'split_pending', 'execution_stopped', 'execution_failed')),
    source_work_item_ref TEXT,
    source_work_item_revision INTEGER NOT NULL DEFAULT 0 CHECK (source_work_item_revision >= 0),
    source_execution_ref TEXT,
    source_execution_attempt INTEGER NOT NULL DEFAULT 0 CHECK (source_execution_attempt >= 0),
    applied_goal_revision INTEGER NOT NULL CHECK (applied_goal_revision = source_goal_revision + 1),
    applied_plan_generation INTEGER NOT NULL CHECK (applied_plan_generation = source_plan_generation + 1),
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    decided_at INTEGER NOT NULL,
    UNIQUE(principal_ref, project_ref, request_ref),
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals_v9(ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, source_work_item_ref) REFERENCES work_items_v9(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, source_work_item_ref, source_execution_ref)
        REFERENCES executions_v9(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    CHECK (
        (cause = '' AND source_work_item_ref IS NULL AND source_work_item_revision = 0
            AND source_execution_ref IS NULL AND source_execution_attempt = 0)
        OR
        (cause <> '' AND source_work_item_ref IS NOT NULL AND source_work_item_revision > 0
            AND source_execution_ref IS NOT NULL AND source_execution_attempt > 0)
    )
) STRICT;

INSERT INTO director_decisions_v9(
    ref, request_ref, request_fingerprint, authorization_receipt_ref,
    goal_ref, project_ref, principal_ref, lease_fence,
    source_goal_revision, source_plan_generation, cause,
    source_work_item_ref, source_work_item_revision,
    source_execution_ref, source_execution_attempt,
    applied_goal_revision, applied_plan_generation, reason, decided_at
)
SELECT ref, request_ref, request_fingerprint, authorization_receipt_ref,
       goal_ref, project_ref, principal_ref, lease_fence,
       source_goal_revision, source_plan_generation, '', NULL, 0, NULL, 0,
       applied_goal_revision, applied_plan_generation, reason, decided_at
FROM director_decisions;

DROP TABLE action_consumption_receipts;
DROP TABLE outbox;
DROP TABLE director_decisions;
DROP TABLE executions;
DROP TABLE work_items;
DROP TABLE goals;

ALTER TABLE goals_v9 RENAME TO goals;
ALTER TABLE work_items_v9 RENAME TO work_items;
ALTER TABLE executions_v9 RENAME TO executions;
ALTER TABLE outbox_v9 RENAME TO outbox;
ALTER TABLE action_consumption_receipts_v9 RENAME TO action_consumption_receipts;
ALTER TABLE director_decisions_v9 RENAME TO director_decisions;

CREATE INDEX goals_actor_project_created_idx
    ON goals(actor_ref, project_ref, created_at DESC, ref DESC);
CREATE INDEX goals_project_created_idx
    ON goals(project_ref, created_at DESC, ref DESC);
CREATE UNIQUE INDEX goals_ref_project_v7_idx ON goals(ref, project_ref);

CREATE INDEX work_items_parent_idx ON work_items(goal_ref, parent_ref, position)
WHERE parent_ref IS NOT NULL;
CREATE UNIQUE INDEX work_items_mailbox_lineage_idx
    ON work_items(goal_ref, ref, parent_ref);
CREATE INDEX work_items_rework_idx ON work_items(goal_ref, rework_of, position)
WHERE rework_of IS NOT NULL;

CREATE INDEX executions_goal_idx ON executions(goal_ref, created_at, ref);
CREATE UNIQUE INDEX executions_one_active_per_work_item_idx
    ON executions(goal_ref, work_item_ref)
    WHERE state IN ('queued', 'dispatching', 'running');

CREATE UNIQUE INDEX outbox_claim_token_idx
    ON outbox(claim_token) WHERE claim_token IS NOT NULL;
CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx
    ON outbox(goal_ref, work_item_ref, plan_generation, work_item_generation)
    WHERE kind IN ('launch_agent', 'observe_agent')
      AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_stop_per_execution_idx
    ON outbox(goal_ref, execution_ref)
    WHERE kind = 'stop_agent'
      AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;
CREATE UNIQUE INDEX outbox_one_active_mailbox_idx
    ON outbox(mailbox_message_ref)
    WHERE kind = 'deliver_mailbox';
CREATE INDEX outbox_claimable_idx
    ON outbox(kind, completed_at, retired_at, quarantined_at, available_at, claimed_until, ref);

CREATE UNIQUE INDEX action_consumption_scheduler_fence_idx
    ON action_consumption_receipts(goal_ref, work_item_ref, fence)
    WHERE kind IN ('launch_agent', 'observe_agent', 'stop_agent');
CREATE UNIQUE INDEX action_consumption_mailbox_fence_idx
    ON action_consumption_receipts(mailbox_message_ref, fence)
    WHERE kind = 'deliver_mailbox';

CREATE UNIQUE INDEX director_decisions_goal_idx
    ON director_decisions(goal_ref, applied_plan_generation);

CREATE TABLE controls (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    authorization_receipt_ref TEXT NOT NULL REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
	goal_revision INTEGER NOT NULL CHECK (goal_revision > 0),
    work_item_ref TEXT,
    work_item_revision INTEGER NOT NULL DEFAULT 0 CHECK (work_item_revision >= 0),
    execution_ref TEXT,
    execution_attempt INTEGER NOT NULL DEFAULT 0 CHECK (execution_attempt >= 0),
    operation TEXT NOT NULL CHECK (operation IN ('pause', 'resume', 'cancel', 'stop', 'retry')),
    target TEXT NOT NULL CHECK (target IN ('goal', 'work_item', 'execution')),
    mode TEXT NOT NULL DEFAULT '' CHECK (mode IN ('', 'cooperative', 'forced')),
    reason TEXT NOT NULL CHECK (reason = trim(reason) AND length(reason) > 0),
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    status TEXT NOT NULL CHECK (status IN ('requested', 'confirmed', 'superseded')),
    requested_at INTEGER NOT NULL,
    confirmed_at INTEGER,
    receipt_ref TEXT,
    supersedes_control_ref TEXT,
    superseded_at INTEGER,
    superseded_by_control_ref TEXT,
    UNIQUE(principal_ref, project_ref, request_ref),
    UNIQUE(goal_ref, ref),
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(supersedes_control_ref) REFERENCES controls(ref) ON DELETE RESTRICT,
    FOREIGN KEY(superseded_by_control_ref) REFERENCES controls(ref) ON DELETE RESTRICT,
    CHECK ((work_item_ref IS NULL) = (work_item_revision = 0)),
    CHECK ((execution_ref IS NULL) = (execution_attempt = 0)),
    CHECK (
        (status = 'requested' AND confirmed_at IS NULL AND receipt_ref IS NULL
            AND superseded_at IS NULL AND superseded_by_control_ref IS NULL)
        OR (status = 'confirmed' AND confirmed_at IS NOT NULL AND receipt_ref IS NOT NULL
            AND superseded_at IS NULL AND superseded_by_control_ref IS NULL)
        OR (status = 'superseded' AND confirmed_at IS NULL AND receipt_ref IS NULL
            AND superseded_at IS NOT NULL AND superseded_at >= requested_at
            AND superseded_by_control_ref IS NOT NULL)
    ),
    CHECK (supersedes_control_ref IS NULL OR
        (operation = 'stop' AND target = 'execution' AND mode = 'forced')),
    CHECK (status <> 'superseded' OR
        (operation = 'stop' AND target = 'execution' AND mode = 'cooperative'
            AND supersedes_control_ref IS NULL)),
    CHECK (
        (target = 'goal' AND work_item_ref IS NULL AND execution_ref IS NULL)
        OR (target = 'work_item' AND work_item_ref IS NOT NULL)
        OR (target = 'execution' AND work_item_ref IS NOT NULL AND execution_ref IS NOT NULL)
    )
) STRICT;

CREATE INDEX controls_goal_idx ON controls(goal_ref, requested_at, ref);
CREATE UNIQUE INDEX controls_supersedes_unique_idx
    ON controls(supersedes_control_ref) WHERE supersedes_control_ref IS NOT NULL;
CREATE UNIQUE INDEX controls_superseded_by_unique_idx
    ON controls(superseded_by_control_ref) WHERE superseded_by_control_ref IS NOT NULL;

CREATE TRIGGER controls_immutable_delete
BEFORE DELETE ON controls BEGIN
    SELECT RAISE(ABORT, 'sqlite.control_immutable');
END;

CREATE TRIGGER controls_update_guard
BEFORE UPDATE ON controls
WHEN NEW.ref <> OLD.ref
 OR NEW.request_ref <> OLD.request_ref
 OR NEW.request_fingerprint <> OLD.request_fingerprint
 OR NEW.authorization_receipt_ref <> OLD.authorization_receipt_ref
 OR NEW.principal_ref <> OLD.principal_ref
 OR NEW.project_ref <> OLD.project_ref
 OR NEW.goal_ref <> OLD.goal_ref
 OR NEW.goal_revision <> OLD.goal_revision
 OR NEW.work_item_ref IS NOT OLD.work_item_ref
 OR NEW.work_item_revision <> OLD.work_item_revision
 OR NEW.execution_ref IS NOT OLD.execution_ref
 OR NEW.execution_attempt <> OLD.execution_attempt
 OR NEW.operation <> OLD.operation
 OR NEW.target <> OLD.target
 OR NEW.mode <> OLD.mode
 OR NEW.reason <> OLD.reason
 OR NEW.plan_generation <> OLD.plan_generation
 OR NEW.app_spec_generation <> OLD.app_spec_generation
 OR NEW.spec_hash <> OLD.spec_hash
 OR NEW.requested_at <> OLD.requested_at
 OR OLD.status <> 'requested'
 OR NEW.supersedes_control_ref IS NOT OLD.supersedes_control_ref
 OR NEW.status NOT IN ('requested', 'confirmed', 'superseded')
 OR (NEW.status = 'requested' AND (
        NEW.confirmed_at IS NOT NULL OR NEW.receipt_ref IS NOT NULL
        OR NEW.superseded_at IS NOT NULL OR NEW.superseded_by_control_ref IS NOT NULL
    ))
 OR (NEW.status = 'confirmed' AND (
        NEW.superseded_at IS NOT NULL OR NEW.superseded_by_control_ref IS NOT NULL
    ))
 OR (NEW.status = 'superseded' AND (
        OLD.operation <> 'stop' OR OLD.target <> 'execution' OR OLD.mode <> 'cooperative'
        OR OLD.supersedes_control_ref IS NOT NULL
        OR NEW.confirmed_at IS NOT NULL OR NEW.receipt_ref IS NOT NULL
        OR NEW.superseded_at IS NULL OR NEW.superseded_at < OLD.requested_at
        OR NEW.superseded_by_control_ref IS NULL
    ))
BEGIN
    SELECT RAISE(ABORT, 'sqlite.control_update_invalid');
END;

CREATE TRIGGER goals_app_spec_scope_guard
BEFORE INSERT ON goals
WHEN NOT EXISTS (
    SELECT 1 FROM app_specs spec JOIN intents intent ON intent.ref = spec.intent_ref
    WHERE spec.ref = NEW.app_spec_ref
      AND intent.actor_ref = NEW.actor_ref AND intent.project_ref = NEW.project_ref
)
BEGIN SELECT RAISE(ABORT, 'sqlite.goal_app_spec_scope_invalid'); END;

CREATE TRIGGER goals_app_spec_immutable
BEFORE UPDATE OF request_ref, request_fingerprint, requested_by_ref,
    app_spec_ref, actor_ref, project_ref ON goals
BEGIN SELECT RAISE(ABORT, 'sqlite.goal_app_spec_immutable'); END;

CREATE TRIGGER work_items_handoff_required_immutable
BEFORE UPDATE OF handoff_required ON work_items
WHEN NEW.handoff_required <> OLD.handoff_required
BEGIN SELECT RAISE(ABORT, 'sqlite.work_item_handoff_required_immutable'); END;

CREATE TRIGGER executions_identity_immutable
BEFORE UPDATE OF goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    artifact_media_type, idempotency_key, max_output_bytes, created_at
ON executions
BEGIN SELECT RAISE(ABORT, 'sqlite.execution_identity_immutable'); END;

CREATE TRIGGER executions_provider_identity_write_once
BEFORE UPDATE OF provider_ref, model_ref, agent_ref, external_ref ON executions
WHEN NOT (
    (NEW.provider_ref = OLD.provider_ref AND NEW.model_ref = OLD.model_ref
        AND NEW.agent_ref = OLD.agent_ref AND NEW.external_ref = OLD.external_ref)
    OR
    (OLD.state = 'dispatching' AND NEW.state = 'running'
        AND OLD.provider_ref = '' AND OLD.model_ref = '' AND OLD.agent_ref = '' AND OLD.external_ref = ''
        AND length(trim(NEW.provider_ref)) > 0 AND length(trim(NEW.model_ref)) > 0
        AND length(trim(NEW.agent_ref)) > 0 AND length(trim(NEW.external_ref)) > 0)
)
BEGIN SELECT RAISE(ABORT, 'sqlite.execution_provider_identity_write_once'); END;

CREATE TRIGGER executions_replacement_guard
BEFORE INSERT ON executions
WHEN NEW.attempt_no > 1 AND NOT EXISTS (
    SELECT 1 FROM executions previous
    WHERE previous.goal_ref = NEW.goal_ref
      AND previous.work_item_ref = NEW.work_item_ref
      AND previous.ref = NEW.replaces_execution_ref
      AND previous.attempt_no + 1 = NEW.attempt_no
      AND previous.max_execution_attempts = NEW.max_execution_attempts
      AND previous.plan_generation = NEW.plan_generation
      AND previous.app_spec_generation = NEW.app_spec_generation
      AND previous.spec_hash = NEW.spec_hash
      AND previous.state IN ('failed', 'stopped')
      AND previous.finished_at IS NOT NULL
      AND NEW.created_at >= previous.finished_at
)
BEGIN SELECT RAISE(ABORT, 'sqlite.execution_replacement_invalid'); END;

CREATE TRIGGER outbox_identity_immutable
BEFORE UPDATE OF kind, goal_ref, work_item_ref, execution_ref, control_ref,
    plan_generation, work_item_generation, mailbox_message_ref
ON outbox BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_identity_immutable'); END;

CREATE TRIGGER outbox_mailbox_recipient_guard
BEFORE UPDATE ON outbox
WHEN NEW.kind = 'deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM mailbox_envelopes envelope
    WHERE envelope.ref = NEW.mailbox_message_ref AND envelope.goal_ref = NEW.goal_ref
      AND envelope.plan_generation = NEW.plan_generation
      AND envelope.parent_work_item_ref = NEW.work_item_ref
      AND envelope.recipient_execution_ref = NEW.execution_ref
      AND envelope.recipient_work_item_generation = NEW.work_item_generation
      AND envelope.recipient_principal_ref = NEW.claimed_by
)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_recipient_mismatch'); END;

CREATE TRIGGER outbox_mailbox_recipient_insert_guard
BEFORE INSERT ON outbox
WHEN NEW.kind = 'deliver_mailbox' AND NEW.claimed_by IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM mailbox_envelopes envelope
    WHERE envelope.ref = NEW.mailbox_message_ref AND envelope.goal_ref = NEW.goal_ref
      AND envelope.plan_generation = NEW.plan_generation
      AND envelope.parent_work_item_ref = NEW.work_item_ref
      AND envelope.recipient_execution_ref = NEW.execution_ref
      AND envelope.recipient_work_item_generation = NEW.work_item_generation
      AND envelope.recipient_principal_ref = NEW.claimed_by
)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_recipient_mismatch'); END;

CREATE TRIGGER outbox_mailbox_retirement_guard
BEFORE UPDATE OF retired_at ON outbox
WHEN NEW.retired_at IS NOT OLD.retired_at AND (
    OLD.retired_at IS NOT NULL OR NEW.retired_at IS NULL OR NOT EXISTS (
        SELECT 1 FROM mailbox_retirements retirement
        WHERE retirement.mailbox_message_ref = NEW.mailbox_message_ref
          AND retirement.action_ref = NEW.ref
          AND retirement.recipient_execution_ref = NEW.execution_ref
          AND retirement.retired_at = NEW.retired_at
    )
)
BEGIN SELECT RAISE(ABORT, 'sqlite.outbox_mailbox_retirement_invalid'); END;

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
      AND action.fence = NEW.fence AND action.delivery_attempt = NEW.delivery_attempt
      AND action.claim_token = NEW.claim_token AND action.claimed_by = NEW.worker_ref
      AND action.last_error_code = NEW.error_code AND action.completed_at = NEW.consumed_at
      AND ((NEW.outcome = 'completed' AND action.quarantined_at IS NULL)
        OR (NEW.outcome = 'quarantined' AND action.quarantined_at = NEW.consumed_at))
      AND (NEW.kind <> 'deliver_mailbox' OR EXISTS (
          SELECT 1 FROM mailbox_envelopes envelope
          WHERE envelope.ref = NEW.mailbox_message_ref
            AND envelope.recipient_principal_ref = NEW.worker_ref
      ))
)
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_invalid'); END;

CREATE TRIGGER action_consumption_receipts_immutable_update
BEFORE UPDATE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable'); END;
CREATE TRIGGER action_consumption_receipts_immutable_delete
BEFORE DELETE ON action_consumption_receipts
BEGIN SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable'); END;

CREATE TRIGGER director_decisions_immutable_update
BEFORE UPDATE ON director_decisions
BEGIN SELECT RAISE(ABORT, 'sqlite.director_decision_immutable'); END;
CREATE TRIGGER director_decisions_immutable_delete
BEFORE DELETE ON director_decisions
BEGIN SELECT RAISE(ABORT, 'sqlite.director_decision_immutable'); END;

CREATE TRIGGER mailbox_handoff_required_guard
BEFORE INSERT ON mailbox_envelopes
WHEN NOT EXISTS (
    SELECT 1 FROM work_items child
    WHERE child.goal_ref = NEW.goal_ref
      AND child.ref = NEW.child_work_item_ref
      AND child.parent_ref = NEW.parent_work_item_ref
      AND child.handoff_required = 1
)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_handoff_required'); END;

CREATE TRIGGER mailbox_delivery_attempt_insert_guard
BEFORE INSERT ON mailbox_delivery_attempts
WHEN NEW.delivery_request_ref IS NOT NULL
 OR NEW.delivery_request_fingerprint IS NOT NULL
 OR NEW.delivery_authorization_receipt_ref IS NOT NULL
 OR NEW.delivery_ref IS NOT NULL
 OR NEW.delivered_at IS NOT NULL
 OR NEW.consumption_request_ref IS NOT NULL
 OR NEW.consumption_request_fingerprint IS NOT NULL
 OR NEW.consumption_authorization_receipt_ref IS NOT NULL
 OR NEW.consumption_ref IS NOT NULL
 OR NEW.consumed_at IS NOT NULL
 OR NEW.fence <> COALESCE((
        SELECT MAX(attempt.fence) FROM mailbox_delivery_attempts attempt
        WHERE attempt.mailbox_message_ref = NEW.mailbox_message_ref
    ), 0) + 1
 OR NOT EXISTS (
    SELECT 1 FROM outbox action
    JOIN mailbox_envelopes envelope ON envelope.ref = NEW.mailbox_message_ref
    JOIN authorization_receipts authorization
      ON authorization.ref = NEW.claim_authorization_receipt_ref
    WHERE action.mailbox_message_ref = NEW.mailbox_message_ref
      AND action.goal_ref = envelope.goal_ref
      AND action.work_item_ref = envelope.parent_work_item_ref
      AND action.execution_ref = envelope.recipient_execution_ref
      AND action.plan_generation = envelope.plan_generation
      AND action.work_item_generation = envelope.recipient_work_item_generation
      AND envelope.project_ref = NEW.project_ref
      AND envelope.recipient_principal_ref = NEW.recipient_principal_ref
      AND action.claim_token = NEW.claim_token
      AND action.claimed_by = NEW.recipient_principal_ref
      AND action.claimed_until = NEW.lease_until
      AND action.delivery_attempt = NEW.fence
      AND action.fence = NEW.fence
      AND action.completed_at IS NULL AND action.retired_at IS NULL
      AND action.quarantined_at IS NULL
      AND authorization.principal_ref = NEW.recipient_principal_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.get'
      AND authorization.resource_ref = NEW.mailbox_message_ref
      AND authorization.outcome = 'allowed'
)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_delivery_attempt_invalid'); END;

CREATE TRIGGER mailbox_delivery_attempt_progress_guard
BEFORE UPDATE ON mailbox_delivery_attempts
WHEN NOT (
    NEW.mailbox_message_ref = OLD.mailbox_message_ref
    AND NEW.action_ref = OLD.action_ref
    AND NEW.project_ref = OLD.project_ref
    AND NEW.recipient_principal_ref = OLD.recipient_principal_ref
    AND NEW.fence = OLD.fence AND NEW.claim_token = OLD.claim_token
    AND NEW.claim_request_ref = OLD.claim_request_ref
    AND NEW.claim_request_fingerprint = OLD.claim_request_fingerprint
    AND NEW.claim_authorization_receipt_ref = OLD.claim_authorization_receipt_ref
    AND NEW.claimed_at = OLD.claimed_at AND NEW.lease_until = OLD.lease_until
    AND EXISTS (
        SELECT 1 FROM outbox action
        WHERE action.ref = OLD.action_ref
          AND action.mailbox_message_ref = OLD.mailbox_message_ref
          AND action.claim_token = OLD.claim_token
          AND action.claimed_by = OLD.recipient_principal_ref
          AND action.claimed_until = OLD.lease_until
          AND action.delivery_attempt = OLD.fence AND action.fence = OLD.fence
          AND action.completed_at IS NULL AND action.retired_at IS NULL
          AND action.quarantined_at IS NULL
    )
    AND (
        (
            OLD.delivery_request_ref IS NULL
            AND OLD.delivery_request_fingerprint IS NULL
            AND OLD.delivery_authorization_receipt_ref IS NULL
            AND OLD.delivery_ref IS NULL AND OLD.delivered_at IS NULL
            AND OLD.consumption_request_ref IS NULL
            AND OLD.consumption_request_fingerprint IS NULL
            AND OLD.consumption_authorization_receipt_ref IS NULL
            AND OLD.consumption_ref IS NULL AND OLD.consumed_at IS NULL
            AND NEW.delivery_request_ref IS NOT NULL
            AND NEW.delivery_request_fingerprint IS NOT NULL
            AND NEW.delivery_authorization_receipt_ref IS NOT NULL
            AND NEW.delivery_ref IS NOT NULL AND NEW.delivered_at IS NOT NULL
            AND NEW.delivered_at >= OLD.claimed_at AND NEW.delivered_at < OLD.lease_until
            AND NEW.consumption_request_ref IS NULL
            AND NEW.consumption_request_fingerprint IS NULL
            AND NEW.consumption_authorization_receipt_ref IS NULL
            AND NEW.consumption_ref IS NULL AND NEW.consumed_at IS NULL
            AND EXISTS (
                SELECT 1 FROM authorization_receipts authorization
                WHERE authorization.ref = NEW.delivery_authorization_receipt_ref
                  AND authorization.principal_ref = OLD.recipient_principal_ref
                  AND authorization.project_ref = OLD.project_ref
                  AND authorization.permission = 'goals.get'
                  AND authorization.resource_ref = OLD.mailbox_message_ref
                  AND authorization.outcome = 'allowed'
            )
        )
        OR
        (
            OLD.delivery_request_ref IS NOT NULL
            AND NEW.delivery_request_ref = OLD.delivery_request_ref
            AND NEW.delivery_request_fingerprint = OLD.delivery_request_fingerprint
            AND NEW.delivery_authorization_receipt_ref = OLD.delivery_authorization_receipt_ref
            AND NEW.delivery_ref = OLD.delivery_ref
            AND OLD.delivered_at IS NOT NULL AND NEW.delivered_at = OLD.delivered_at
            AND OLD.consumption_request_ref IS NULL
            AND OLD.consumption_request_fingerprint IS NULL
            AND OLD.consumption_authorization_receipt_ref IS NULL
            AND OLD.consumption_ref IS NULL AND OLD.consumed_at IS NULL
            AND NEW.consumption_request_ref IS NOT NULL
            AND NEW.consumption_request_fingerprint IS NOT NULL
            AND NEW.consumption_authorization_receipt_ref IS NOT NULL
            AND NEW.consumption_ref IS NOT NULL AND NEW.consumed_at IS NOT NULL
            AND NEW.consumed_at >= OLD.delivered_at AND NEW.consumed_at < OLD.lease_until
            AND EXISTS (
                SELECT 1 FROM authorization_receipts authorization
                WHERE authorization.ref = NEW.consumption_authorization_receipt_ref
                  AND authorization.principal_ref = OLD.recipient_principal_ref
                  AND authorization.project_ref = OLD.project_ref
                  AND authorization.permission = 'goals.get'
                  AND authorization.resource_ref = OLD.mailbox_message_ref
                  AND authorization.outcome = 'allowed'
            )
        )
    )
)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_delivery_attempt_progress_invalid'); END;

CREATE TRIGGER mailbox_ack_guard
BEFORE INSERT ON mailbox_delivery_acks
WHEN EXISTS (
    SELECT 1 FROM mailbox_retirements retirement
    WHERE retirement.mailbox_message_ref = NEW.mailbox_message_ref
)
OR NOT EXISTS (
    SELECT 1 FROM mailbox_envelopes envelope
    JOIN mailbox_delivery_attempts attempt ON attempt.mailbox_message_ref = envelope.ref
    JOIN action_consumption_receipts receipt ON receipt.action_ref = NEW.action_ref
    JOIN outbox action ON action.ref = NEW.action_ref
    JOIN authorization_receipts authorization ON authorization.ref = NEW.authorization_receipt_ref
    WHERE envelope.ref = NEW.mailbox_message_ref
      AND envelope.project_ref = NEW.project_ref
      AND envelope.recipient_principal_ref = NEW.recipient_principal_ref
      AND NEW.expected_plan_generation >= envelope.plan_generation
      AND attempt.action_ref = NEW.action_ref
      AND attempt.recipient_principal_ref = NEW.recipient_principal_ref
      AND attempt.consumed_at IS NOT NULL
      AND receipt.kind = 'deliver_mailbox'
      AND receipt.mailbox_message_ref = NEW.mailbox_message_ref
      AND receipt.fence = attempt.fence AND receipt.delivery_attempt = attempt.fence
      AND receipt.claim_token = attempt.claim_token
      AND receipt.worker_ref = NEW.recipient_principal_ref
      AND receipt.outcome = 'completed' AND receipt.consumed_at = attempt.consumed_at
      AND NEW.acked_at >= receipt.consumed_at
      AND action.completed_at = receipt.consumed_at
      AND action.retired_at IS NULL AND action.quarantined_at IS NULL
      AND authorization.principal_ref = NEW.recipient_principal_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.get'
      AND authorization.resource_ref = NEW.mailbox_message_ref
      AND authorization.outcome = 'allowed'
)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_ack_invalid'); END;

CREATE TRIGGER mailbox_retirement_guard
BEFORE INSERT ON mailbox_retirements
WHEN EXISTS (
    SELECT 1 FROM mailbox_delivery_acks ack
    WHERE ack.mailbox_message_ref = NEW.mailbox_message_ref
)
OR NOT EXISTS (
    SELECT 1 FROM mailbox_envelopes envelope
    JOIN outbox action ON action.ref = NEW.action_ref
    JOIN executions execution ON execution.ref = NEW.recipient_execution_ref
    WHERE envelope.ref = NEW.mailbox_message_ref
      AND envelope.recipient_execution_ref = NEW.recipient_execution_ref
      AND action.kind = 'deliver_mailbox'
      AND action.mailbox_message_ref = NEW.mailbox_message_ref
      AND action.execution_ref = NEW.recipient_execution_ref
      AND action.retired_at IS NULL
      AND execution.goal_ref = envelope.goal_ref
      AND execution.work_item_ref = envelope.parent_work_item_ref
      AND execution.state IN ('succeeded', 'failed', 'stopped', 'canceled')
      AND (
          execution.failure_code = NEW.failure_code
          OR (execution.state = 'succeeded' AND NEW.failure_code = 'application.execution_canceled')
      )
      AND execution.finished_at = NEW.retired_at
)
BEGIN SELECT RAISE(ABORT, 'sqlite.mailbox_retirement_invalid'); END;
