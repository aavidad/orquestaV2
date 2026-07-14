ALTER TABLE goals
ADD COLUMN plan_generation INTEGER NOT NULL DEFAULT 1 CHECK (plan_generation > 0);

CREATE TABLE goal_phases (
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    phase_key TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY(goal_ref, phase_key),
    UNIQUE(goal_ref, position)
) STRICT;

INSERT INTO goal_phases(goal_ref, phase_key, position)
SELECT ref, 'phase:default', 0 FROM goals;

CREATE TABLE work_items_next (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    objective TEXT NOT NULL,
    phase_key TEXT NOT NULL,
    role_key TEXT NOT NULL,
    output_contract TEXT NOT NULL CHECK (output_contract IN ('evidence_bundle', 'artifact', 'attestation')),
    skip_reason TEXT NOT NULL DEFAULT '' CHECK (skip_reason IN ('', 'dependency_failed')),
    state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'succeeded', 'failed', 'skipped')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    position INTEGER NOT NULL CHECK (position >= 0),
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER,
    execution_ref TEXT,
    UNIQUE(goal_ref, ref),
    UNIQUE(goal_ref, position),
    FOREIGN KEY(goal_ref, phase_key) REFERENCES goal_phases(goal_ref, phase_key) ON DELETE RESTRICT
) STRICT;

INSERT INTO work_items_next(
    ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
    output_contract, skip_reason, state, revision, position, created_at,
    started_at, finished_at, execution_ref
)
SELECT ref, goal_ref, actor_ref, project_ref, objective, 'phase:default',
       'role:worker', 'evidence_bundle', '', state, revision, position,
       created_at, started_at, finished_at, execution_ref
FROM work_items;

CREATE TABLE executions_next (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL CHECK (state IN ('queued', 'dispatching', 'running', 'succeeded', 'failed')),
    artifact_media_type TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    max_output_bytes INTEGER NOT NULL CHECK (max_output_bytes > 0),
    max_attempts INTEGER NOT NULL CHECK (max_attempts > 0),
    provider_ref TEXT NOT NULL DEFAULT '',
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
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items_next(goal_ref, ref) ON DELETE CASCADE
) STRICT;

INSERT INTO executions_next(
    ref, goal_ref, work_item_ref, state, artifact_media_type, idempotency_key,
    max_output_bytes, max_attempts, provider_ref, external_ref, created_at,
    deadline_at, started_at, provider_accepted_at, last_observed_at,
    provider_observed_at, finished_at, failure_code
)
SELECT ref, goal_ref, work_item_ref, state, artifact_media_type, idempotency_key,
       max_output_bytes, max_attempts, provider_ref, external_ref, created_at,
       CASE WHEN state = 'queued' THEN NULL ELSE deadline_at END,
       CASE WHEN state = 'queued' THEN NULL ELSE started_at END,
       provider_accepted_at, last_observed_at, provider_observed_at, finished_at,
       failure_code
FROM executions;

CREATE TABLE artifacts_next (
    ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    digest TEXT NOT NULL,
    media_type TEXT NOT NULL,
    size INTEGER NOT NULL CHECK (size >= 0),
    created_at INTEGER NOT NULL,
    PRIMARY KEY(goal_ref, ref),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items_next(goal_ref, ref) ON DELETE CASCADE
) STRICT;

INSERT INTO artifacts_next SELECT * FROM artifacts;

CREATE TABLE attestations_next (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    artifact_ref TEXT NOT NULL,
    policy TEXT NOT NULL,
    accepted_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items_next(goal_ref, ref) ON DELETE CASCADE,
	FOREIGN KEY(goal_ref, work_item_ref, execution_ref) REFERENCES executions_next(goal_ref, work_item_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, artifact_ref) REFERENCES artifacts_next(goal_ref, ref) ON DELETE RESTRICT
) STRICT;

INSERT INTO attestations_next SELECT * FROM attestations;

CREATE TABLE events_next (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT,
    execution_ref TEXT,
    occurred_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items_next(goal_ref, ref) ON DELETE CASCADE,
	FOREIGN KEY(goal_ref, work_item_ref, execution_ref) REFERENCES executions_next(goal_ref, work_item_ref, ref) ON DELETE CASCADE
) STRICT;

INSERT INTO events_next SELECT * FROM events;

CREATE TABLE outbox_next (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent')),
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    available_at INTEGER NOT NULL,
    claim_token TEXT,
    claimed_by TEXT,
    claimed_until INTEGER,
    attempt INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    completed_at INTEGER,
    quarantined_at INTEGER,
    last_error_code TEXT NOT NULL DEFAULT '',
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items_next(goal_ref, ref) ON DELETE CASCADE,
	FOREIGN KEY(goal_ref, work_item_ref, execution_ref) REFERENCES executions_next(goal_ref, work_item_ref, ref) ON DELETE CASCADE,
    CHECK (
        (claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL)
        OR
        (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL)
    ),
    CHECK (quarantined_at IS NULL OR completed_at IS NOT NULL)
) STRICT;

INSERT INTO outbox_next SELECT * FROM outbox;

DROP TABLE attestations;
DROP TABLE events;
DROP TABLE outbox;
DROP TABLE artifacts;
DROP TABLE executions;
DROP TABLE work_items;

ALTER TABLE work_items_next RENAME TO work_items;
ALTER TABLE executions_next RENAME TO executions;
ALTER TABLE artifacts_next RENAME TO artifacts;
ALTER TABLE attestations_next RENAME TO attestations;
ALTER TABLE events_next RENAME TO events;
ALTER TABLE outbox_next RENAME TO outbox;

CREATE TABLE work_item_dependencies (
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    dependency_ref TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY(work_item_ref, dependency_ref),
    UNIQUE(work_item_ref, position),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, dependency_ref) REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    CHECK (work_item_ref <> dependency_ref)
) STRICT;

CREATE TABLE work_item_write_scopes (
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    scope TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY(work_item_ref, scope),
    UNIQUE(work_item_ref, position),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE
) STRICT;

CREATE INDEX executions_goal_idx ON executions(goal_ref);
CREATE INDEX artifacts_goal_idx ON artifacts(goal_ref, created_at, ref);
CREATE INDEX artifacts_work_item_idx ON artifacts(work_item_ref, created_at, ref);
CREATE INDEX attestations_goal_idx ON attestations(goal_ref, accepted_at, ref);
CREATE INDEX attestations_work_item_idx ON attestations(work_item_ref, accepted_at, ref);
CREATE INDEX events_goal_idx ON events(goal_ref, occurred_at, ref);
CREATE UNIQUE INDEX outbox_claim_token_idx ON outbox(claim_token) WHERE claim_token IS NOT NULL;
CREATE INDEX outbox_claimable_idx ON outbox(completed_at, available_at, claimed_until, ref);
