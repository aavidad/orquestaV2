CREATE TABLE app_specs (
    ref TEXT PRIMARY KEY,
    intent_ref TEXT NOT NULL UNIQUE REFERENCES intents(ref) ON DELETE RESTRICT,
    generation INTEGER NOT NULL CHECK (generation > 0),
    parent_ref TEXT UNIQUE REFERENCES app_specs(ref) ON DELETE RESTRICT,
    parent_hash TEXT,
    objective TEXT NOT NULL CHECK (length(trim(objective)) > 0),
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    confirmed_by TEXT NOT NULL CHECK (length(trim(confirmed_by)) > 0),
    confirmed_at INTEGER NOT NULL,
    hash TEXT NOT NULL UNIQUE CHECK (length(hash) = 64 AND hash NOT GLOB '*[^0-9a-f]*'),
    CHECK (
        (generation = 1 AND parent_ref IS NULL AND parent_hash IS NULL)
        OR
        (generation > 1 AND parent_ref IS NOT NULL AND parent_hash IS NOT NULL
            AND length(parent_hash) = 64 AND parent_hash NOT GLOB '*[^0-9a-f]*')
    )
) STRICT;

CREATE TABLE goals_v3 (
    ref TEXT PRIMARY KEY,
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    app_spec_ref TEXT NOT NULL UNIQUE REFERENCES app_specs(ref) ON DELETE RESTRICT,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'succeeded', 'failed')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    closed_at INTEGER,
    plan_generation INTEGER NOT NULL CHECK (plan_generation >= 0),
    UNIQUE(actor_ref, project_ref, request_ref)
) STRICT;

-- orquesta:go-backfill app_specs

DROP TABLE goals;
ALTER TABLE goals_v3 RENAME TO goals;

CREATE INDEX goals_actor_project_created_idx
    ON goals(actor_ref, project_ref, created_at DESC, ref DESC);

CREATE TRIGGER intents_immutable_update
BEFORE UPDATE ON intents
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intent_immutable');
END;

CREATE TRIGGER intents_immutable_delete
BEFORE DELETE ON intents
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intent_immutable');
END;

CREATE TRIGGER app_specs_confirmation_guard
BEFORE INSERT ON app_specs
WHEN NOT EXISTS (
    SELECT 1
    FROM intents i
    WHERE i.ref = NEW.intent_ref
      AND NEW.confirmed_at >= i.submitted_at
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.app_spec_confirmation_invalid');
END;

CREATE TRIGGER app_specs_parent_guard
BEFORE INSERT ON app_specs
WHEN NEW.generation > 1 AND NOT EXISTS (
    SELECT 1
    FROM app_specs parent
    JOIN intents parent_intent ON parent_intent.ref = parent.intent_ref
    JOIN intents child_intent ON child_intent.ref = NEW.intent_ref
    WHERE parent.ref = NEW.parent_ref
      AND parent.hash = NEW.parent_hash
      AND parent.generation + 1 = NEW.generation
      AND parent_intent.actor_ref = child_intent.actor_ref
      AND parent_intent.project_ref = child_intent.project_ref
      AND child_intent.submitted_at >= parent.confirmed_at
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.app_spec_parent_invalid');
END;

CREATE TRIGGER app_specs_immutable_update
BEFORE UPDATE ON app_specs
BEGIN
    SELECT RAISE(ABORT, 'sqlite.app_spec_immutable');
END;

CREATE TRIGGER app_specs_immutable_delete
BEFORE DELETE ON app_specs
BEGIN
    SELECT RAISE(ABORT, 'sqlite.app_spec_immutable');
END;

CREATE TRIGGER goals_app_spec_scope_guard
BEFORE INSERT ON goals
WHEN NOT EXISTS (
    SELECT 1
    FROM app_specs spec
    JOIN intents intent ON intent.ref = spec.intent_ref
    WHERE spec.ref = NEW.app_spec_ref
      AND intent.actor_ref = NEW.actor_ref
      AND intent.project_ref = NEW.project_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.goal_app_spec_scope_invalid');
END;

CREATE TRIGGER goals_app_spec_immutable
BEFORE UPDATE OF app_spec_ref, actor_ref, project_ref ON goals
BEGIN
    SELECT RAISE(ABORT, 'sqlite.goal_app_spec_immutable');
END;
