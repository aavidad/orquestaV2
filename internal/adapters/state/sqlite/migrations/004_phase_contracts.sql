CREATE TABLE goal_phases_next (
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    ref TEXT NOT NULL,
    phase_key TEXT NOT NULL,
    template_ref TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY(goal_ref, ref),
    UNIQUE(goal_ref, phase_key),
    UNIQUE(goal_ref, position)
) STRICT;

INSERT INTO goal_phases_next(goal_ref, ref, phase_key, template_ref, position)
SELECT goal_ref, 'phase-instance:' || phase_key, phase_key,
       'phase-template:' || phase_key, position
FROM goal_phases;

DROP TABLE goal_phases;
ALTER TABLE goal_phases_next RENAME TO goal_phases;

CREATE TABLE work_items_next (
    ref TEXT PRIMARY KEY,
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    objective TEXT NOT NULL,
    phase_key TEXT NOT NULL,
    role_key TEXT NOT NULL,
    parent_ref TEXT,
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
    FOREIGN KEY(goal_ref, phase_key) REFERENCES goal_phases(goal_ref, phase_key) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, parent_ref) REFERENCES work_items_next(goal_ref, ref)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CHECK (parent_ref IS NULL OR parent_ref <> ref)
) STRICT;

INSERT INTO work_items_next(
    ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
    parent_ref, output_contract, skip_reason, state, revision, position,
    created_at, started_at, finished_at, execution_ref
)
SELECT ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
       NULL, output_contract, skip_reason, state, revision, position,
       created_at, started_at, finished_at, execution_ref
FROM work_items;

DROP TABLE work_items;
ALTER TABLE work_items_next RENAME TO work_items;

CREATE INDEX work_items_parent_idx ON work_items(goal_ref, parent_ref, position)
WHERE parent_ref IS NOT NULL;

CREATE TABLE goal_phase_contract_refs (
    goal_ref TEXT NOT NULL,
    phase_ref TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('input', 'criterion')),
    value TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY(goal_ref, phase_ref, kind, value),
    UNIQUE(goal_ref, phase_ref, kind, position),
    FOREIGN KEY(goal_ref, phase_ref) REFERENCES goal_phases(goal_ref, ref) ON DELETE CASCADE
) STRICT;

CREATE TABLE work_item_requirement_refs (
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('skill', 'tool', 'capability')),
    value TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY(goal_ref, work_item_ref, kind, value),
    UNIQUE(goal_ref, work_item_ref, kind, position),
    FOREIGN KEY(goal_ref, work_item_ref) REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE
) STRICT;
