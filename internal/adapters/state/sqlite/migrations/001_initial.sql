CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    checksum TEXT NOT NULL,
    applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
) STRICT;

CREATE TABLE intents (
    ref TEXT PRIMARY KEY,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    statement TEXT NOT NULL,
    submitted_at INTEGER NOT NULL,
    hash TEXT NOT NULL
) STRICT;

CREATE TABLE goals (
    ref TEXT PRIMARY KEY,
    request_ref TEXT NOT NULL,
    request_fingerprint TEXT NOT NULL,
    intent_ref TEXT NOT NULL UNIQUE REFERENCES intents(ref) ON DELETE RESTRICT,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'succeeded', 'failed')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    closed_at INTEGER,
    UNIQUE(actor_ref, project_ref, request_ref)
) STRICT;

CREATE INDEX goals_actor_project_created_idx
    ON goals(actor_ref, project_ref, created_at DESC, ref DESC);

CREATE TABLE work_items (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    objective TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'succeeded', 'failed')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    position INTEGER NOT NULL CHECK (position >= 0),
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER,
    execution_ref TEXT,
    UNIQUE(goal_ref, position)
) STRICT;

CREATE TABLE executions (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL UNIQUE REFERENCES work_items(ref) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK (state IN ('queued', 'running', 'succeeded', 'failed')),
    artifact_media_type TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    max_output_bytes INTEGER NOT NULL CHECK (max_output_bytes > 0),
    max_attempts INTEGER NOT NULL CHECK (max_attempts > 0),
    provider_ref TEXT NOT NULL DEFAULT '',
    external_ref TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    deadline_at INTEGER NOT NULL,
    started_at INTEGER,
    provider_accepted_at INTEGER,
    last_observed_at INTEGER,
    provider_observed_at INTEGER,
    finished_at INTEGER,
    failure_code TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX executions_goal_idx ON executions(goal_ref);

CREATE TABLE artifacts (
    ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL REFERENCES work_items(ref) ON DELETE CASCADE,
    digest TEXT NOT NULL,
    media_type TEXT NOT NULL,
    size INTEGER NOT NULL CHECK (size >= 0),
    created_at INTEGER NOT NULL,
    PRIMARY KEY(goal_ref, ref)
) STRICT;

CREATE INDEX artifacts_goal_idx ON artifacts(goal_ref, created_at, ref);
CREATE INDEX artifacts_work_item_idx ON artifacts(work_item_ref, created_at, ref);

CREATE TABLE attestations (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL REFERENCES work_items(ref) ON DELETE CASCADE,
    execution_ref TEXT NOT NULL REFERENCES executions(ref) ON DELETE CASCADE,
    artifact_ref TEXT NOT NULL,
    policy TEXT NOT NULL,
    accepted_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, artifact_ref) REFERENCES artifacts(goal_ref, ref) ON DELETE RESTRICT
) STRICT;

CREATE INDEX attestations_goal_idx ON attestations(goal_ref, accepted_at, ref);
CREATE INDEX attestations_work_item_idx ON attestations(work_item_ref, accepted_at, ref);

CREATE TABLE events (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL REFERENCES work_items(ref) ON DELETE CASCADE,
    execution_ref TEXT NOT NULL REFERENCES executions(ref) ON DELETE CASCADE,
    occurred_at INTEGER NOT NULL
) STRICT;

CREATE INDEX events_goal_idx ON events(goal_ref, occurred_at, ref);

CREATE TABLE outbox (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent')),
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL REFERENCES work_items(ref) ON DELETE CASCADE,
    execution_ref TEXT NOT NULL REFERENCES executions(ref) ON DELETE CASCADE,
    available_at INTEGER NOT NULL,
    claim_token TEXT,
    claimed_by TEXT,
    claimed_until INTEGER,
    attempt INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    completed_at INTEGER,
    quarantined_at INTEGER,
    last_error_code TEXT NOT NULL DEFAULT '',
    CHECK (
        (claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL)
        OR
        (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL)
    ),
    CHECK (quarantined_at IS NULL OR completed_at IS NOT NULL)
) STRICT;

CREATE UNIQUE INDEX outbox_claim_token_idx
    ON outbox(claim_token)
    WHERE claim_token IS NOT NULL;

CREATE INDEX outbox_claimable_idx
    ON outbox(completed_at, available_at, claimed_until, ref);
