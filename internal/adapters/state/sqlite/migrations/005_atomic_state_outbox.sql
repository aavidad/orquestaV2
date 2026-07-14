CREATE TABLE executions_v5 (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    attempt_no INTEGER NOT NULL CHECK (attempt_no > 0),
    max_execution_attempts INTEGER NOT NULL CHECK (max_execution_attempts > 0),
    replaces_execution_ref TEXT,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    app_spec_generation INTEGER NOT NULL CHECK (app_spec_generation > 0),
    spec_hash TEXT NOT NULL CHECK (length(spec_hash) = 64 AND spec_hash NOT GLOB '*[^0-9a-f]*'),
    state TEXT NOT NULL CHECK (state IN ('queued', 'dispatching', 'running', 'succeeded', 'failed')),
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
    UNIQUE(goal_ref, ref),
    UNIQUE(goal_ref, work_item_ref, ref),
    UNIQUE(goal_ref, work_item_ref, attempt_no),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, work_item_ref, replaces_execution_ref)
        REFERENCES executions_v5(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    CHECK (
        (attempt_no = 1 AND replaces_execution_ref IS NULL)
        OR (attempt_no > 1 AND replaces_execution_ref IS NOT NULL)
    ),
    CHECK (attempt_no <= max_execution_attempts)
) STRICT;

INSERT INTO executions_v5(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code
)
SELECT e.ref, e.goal_ref, e.work_item_ref, 1, e.max_attempts,
       NULL, g.plan_generation, spec.generation, spec.hash,
       e.state, e.artifact_media_type, e.idempotency_key,
       e.max_output_bytes, e.provider_ref,
       CASE WHEN e.provider_ref = '' THEN '' ELSE 'legacy:v4:model-unattributed' END,
       CASE WHEN e.provider_ref = '' THEN '' ELSE 'legacy:v4:agent-unattributed' END,
       e.external_ref, e.created_at, e.deadline_at, e.started_at,
       e.provider_accepted_at, e.last_observed_at, e.provider_observed_at,
       e.finished_at, e.failure_code
FROM executions e
JOIN goals g ON g.ref = e.goal_ref
JOIN app_specs spec ON spec.ref = g.app_spec_ref;

CREATE TABLE attestations_v5 (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    artifact_ref TEXT NOT NULL,
    policy TEXT NOT NULL,
    accepted_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions_v5(goal_ref, work_item_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, artifact_ref) REFERENCES artifacts(goal_ref, ref) ON DELETE RESTRICT
) STRICT;

INSERT INTO attestations_v5 SELECT * FROM attestations;

CREATE TABLE events_v5 (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT,
    execution_ref TEXT,
    occurred_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions_v5(goal_ref, work_item_ref, ref) ON DELETE CASCADE
) STRICT;

INSERT INTO events_v5 SELECT * FROM events;

CREATE TABLE outbox_v5 (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent')),
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    work_item_generation INTEGER NOT NULL CHECK (work_item_generation > 0),
    available_at INTEGER NOT NULL,
    claim_token TEXT,
    claimed_by TEXT,
    claimed_until INTEGER,
    delivery_attempt INTEGER NOT NULL DEFAULT 0 CHECK (delivery_attempt >= 0),
    fence INTEGER NOT NULL DEFAULT 0 CHECK (fence >= 0),
    completed_at INTEGER,
    quarantined_at INTEGER,
    last_error_code TEXT NOT NULL DEFAULT '',
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions_v5(goal_ref, work_item_ref, ref) ON DELETE CASCADE,
    CHECK (
        (claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL)
        OR
        (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL
            AND delivery_attempt > 0 AND fence > 0)
    ),
    CHECK (quarantined_at IS NULL OR completed_at IS NOT NULL)
) STRICT;

WITH migrated AS (
    SELECT o.*,
           SUM(CASE WHEN o.attempt > 0 THEN o.attempt ELSE 0 END) OVER (
               PARTITION BY o.goal_ref, o.work_item_ref
               ORDER BY COALESCE(o.completed_at, o.available_at), o.ref
               ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
           ) AS migrated_fence,
           CASE
               WHEN o.kind = 'launch_agent' THEN 1
               WHEN wi.state IN ('succeeded', 'failed') THEN wi.revision - 1
               ELSE wi.revision
           END AS migrated_work_item_generation
    FROM outbox o
    JOIN work_items wi ON wi.goal_ref = o.goal_ref AND wi.ref = o.work_item_ref
)
INSERT INTO outbox_v5(
    ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, available_at, claim_token, claimed_by, claimed_until,
    delivery_attempt, fence, completed_at, quarantined_at, last_error_code
)
SELECT migrated.ref, migrated.kind, migrated.goal_ref, migrated.work_item_ref,
       migrated.execution_ref, g.plan_generation, migrated.migrated_work_item_generation,
       migrated.available_at, migrated.claim_token, migrated.claimed_by,
       migrated.claimed_until, migrated.attempt,
       CASE WHEN migrated.attempt = 0 THEN 0 ELSE migrated.migrated_fence END,
       migrated.completed_at, migrated.quarantined_at, migrated.last_error_code
FROM migrated
JOIN goals g ON g.ref = migrated.goal_ref;

DROP TABLE attestations;
DROP TABLE events;
DROP TABLE outbox;
DROP TABLE executions;

ALTER TABLE executions_v5 RENAME TO executions;
ALTER TABLE attestations_v5 RENAME TO attestations;
ALTER TABLE events_v5 RENAME TO events;
ALTER TABLE outbox_v5 RENAME TO outbox;

CREATE INDEX executions_goal_idx ON executions(goal_ref, created_at, ref);
CREATE UNIQUE INDEX executions_one_active_per_work_item_idx
    ON executions(goal_ref, work_item_ref)
    WHERE state IN ('queued', 'dispatching', 'running');
CREATE INDEX attestations_goal_idx ON attestations(goal_ref, accepted_at, ref);
CREATE INDEX attestations_work_item_idx ON attestations(work_item_ref, accepted_at, ref);
CREATE INDEX events_goal_idx ON events(goal_ref, occurred_at, ref);
CREATE UNIQUE INDEX outbox_claim_token_idx ON outbox(claim_token) WHERE claim_token IS NOT NULL;
CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx
    ON outbox(goal_ref, work_item_ref, plan_generation, work_item_generation)
    WHERE completed_at IS NULL AND quarantined_at IS NULL;
CREATE INDEX outbox_claimable_idx
    ON outbox(completed_at, quarantined_at, available_at, claimed_until, ref);

CREATE TABLE work_item_fences (
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    fence INTEGER NOT NULL CHECK (fence >= 0),
    PRIMARY KEY(goal_ref, work_item_ref),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE
) STRICT;

INSERT INTO work_item_fences(goal_ref, work_item_ref, fence)
SELECT wi.goal_ref, wi.ref, COALESCE(MAX(o.fence), 0)
FROM work_items wi
LEFT JOIN outbox o ON o.goal_ref = wi.goal_ref AND o.work_item_ref = wi.ref
GROUP BY wi.goal_ref, wi.ref;

CREATE TABLE action_consumption_receipts (
    action_ref TEXT PRIMARY KEY REFERENCES outbox(ref) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent')),
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    work_item_generation INTEGER NOT NULL CHECK (work_item_generation > 0),
    fence INTEGER NOT NULL CHECK (fence > 0),
    delivery_attempt INTEGER NOT NULL CHECK (delivery_attempt > 0),
    claim_token TEXT NOT NULL UNIQUE,
    worker_ref TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('completed', 'quarantined')),
    error_code TEXT NOT NULL DEFAULT '',
    consumed_at INTEGER NOT NULL,
    UNIQUE(goal_ref, work_item_ref, fence),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT
) STRICT;

INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, fence, delivery_attempt, claim_token, worker_ref,
    outcome, error_code, consumed_at
)
SELECT ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
       work_item_generation, fence, delivery_attempt, claim_token, claimed_by,
       CASE WHEN quarantined_at IS NULL THEN 'completed' ELSE 'quarantined' END,
       last_error_code, completed_at
FROM outbox
WHERE completed_at IS NOT NULL AND claim_token IS NOT NULL;

CREATE TRIGGER executions_replacement_guard
BEFORE INSERT ON executions
WHEN NEW.attempt_no > 1 AND NOT EXISTS (
    SELECT 1
    FROM executions previous
    WHERE previous.goal_ref = NEW.goal_ref
      AND previous.work_item_ref = NEW.work_item_ref
      AND previous.ref = NEW.replaces_execution_ref
      AND previous.attempt_no + 1 = NEW.attempt_no
      AND previous.max_execution_attempts = NEW.max_execution_attempts
      AND previous.plan_generation = NEW.plan_generation
      AND previous.app_spec_generation = NEW.app_spec_generation
      AND previous.spec_hash = NEW.spec_hash
      AND previous.state = 'failed'
      AND previous.finished_at IS NOT NULL
      AND NEW.created_at >= previous.finished_at
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.execution_replacement_invalid');
END;

CREATE TRIGGER executions_provider_identity_write_once
BEFORE UPDATE OF provider_ref, model_ref, agent_ref, external_ref ON executions
WHEN NOT (
    (
        NEW.provider_ref = OLD.provider_ref
        AND NEW.model_ref = OLD.model_ref
        AND NEW.agent_ref = OLD.agent_ref
        AND NEW.external_ref = OLD.external_ref
    )
    OR
    (
        OLD.state = 'dispatching'
        AND NEW.state = 'running'
        AND OLD.provider_ref = ''
        AND OLD.model_ref = ''
        AND OLD.agent_ref = ''
        AND OLD.external_ref = ''
        AND length(trim(NEW.provider_ref)) > 0
        AND length(trim(NEW.model_ref)) > 0
        AND length(trim(NEW.agent_ref)) > 0
        AND length(trim(NEW.external_ref)) > 0
    )
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.execution_provider_identity_write_once');
END;

CREATE TRIGGER executions_identity_immutable
BEFORE UPDATE OF goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    artifact_media_type, idempotency_key, max_output_bytes, created_at
ON executions
BEGIN
    SELECT RAISE(ABORT, 'sqlite.execution_identity_immutable');
END;

CREATE TRIGGER outbox_identity_immutable
BEFORE UPDATE OF kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation
ON outbox
BEGIN
    SELECT RAISE(ABORT, 'sqlite.outbox_identity_immutable');
END;

CREATE TRIGGER action_consumption_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NOT EXISTS (
    SELECT 1
    FROM outbox action
    WHERE action.ref = NEW.action_ref
      AND action.kind = NEW.kind
      AND action.goal_ref = NEW.goal_ref
      AND action.work_item_ref = NEW.work_item_ref
      AND action.execution_ref = NEW.execution_ref
      AND action.plan_generation = NEW.plan_generation
      AND action.work_item_generation = NEW.work_item_generation
      AND action.fence = NEW.fence
      AND action.delivery_attempt = NEW.delivery_attempt
      AND action.claim_token = NEW.claim_token
      AND action.claimed_by = NEW.worker_ref
      AND action.last_error_code = NEW.error_code
      AND action.completed_at = NEW.consumed_at
      AND (
          (NEW.outcome = 'completed' AND action.quarantined_at IS NULL)
          OR
          (NEW.outcome = 'quarantined' AND action.quarantined_at = NEW.consumed_at)
      )
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_invalid');
END;

CREATE TRIGGER action_consumption_receipts_immutable_update
BEFORE UPDATE ON action_consumption_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable');
END;

CREATE TRIGGER action_consumption_receipts_immutable_delete
BEFORE DELETE ON action_consumption_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable');
END;
