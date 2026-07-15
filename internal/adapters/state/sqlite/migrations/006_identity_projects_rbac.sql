CREATE TABLE principals (
    ref TEXT PRIMARY KEY CHECK (
        length(trim(ref)) > 0
        AND instr(ref, char(0)) = 0
        AND instr(ref, char(10)) = 0
        AND instr(ref, char(13)) = 0
    ),
    actor_ref TEXT NOT NULL CHECK (
        length(trim(actor_ref)) > 0
        AND instr(actor_ref, char(0)) = 0
        AND instr(actor_ref, char(10)) = 0
        AND instr(actor_ref, char(13)) = 0
    ),
    kind TEXT NOT NULL CHECK (kind IN ('human', 'service')),
    authentication_method TEXT NOT NULL CHECK (
        length(trim(authentication_method)) > 0
        AND instr(authentication_method, char(0)) = 0
        AND instr(authentication_method, char(10)) = 0
        AND instr(authentication_method, char(13)) = 0
    )
) STRICT;

INSERT INTO principals(ref, actor_ref, kind, authentication_method)
SELECT 'migration:v09:' || historical.actor_ref,
       historical.actor_ref,
       'human',
       'migration.v09'
FROM (
    SELECT actor_ref FROM intents
    UNION
    SELECT actor_ref FROM goals
    UNION
    SELECT actor_ref FROM work_items
    UNION
    SELECT confirmed_by AS actor_ref FROM app_specs
) AS historical
ORDER BY historical.actor_ref;

CREATE TRIGGER principals_immutable_update
BEFORE UPDATE ON principals
BEGIN
    SELECT RAISE(ABORT, 'sqlite.principal_immutable');
END;

CREATE TRIGGER principals_immutable_delete
BEFORE DELETE ON principals
BEGIN
    SELECT RAISE(ABORT, 'sqlite.principal_immutable');
END;

CREATE TABLE workspaces (
    ref TEXT PRIMARY KEY CHECK (
        length(trim(ref)) > 0
        AND instr(ref, char(0)) = 0
        AND instr(ref, char(10)) = 0
        AND instr(ref, char(13)) = 0
    )
) STRICT;

CREATE TABLE groups (
    ref TEXT PRIMARY KEY CHECK (
        length(trim(ref)) > 0
        AND instr(ref, char(0)) = 0
        AND instr(ref, char(10)) = 0
        AND instr(ref, char(13)) = 0
    ),
    workspace_ref TEXT NOT NULL REFERENCES workspaces(ref) ON DELETE RESTRICT
) STRICT;

CREATE INDEX groups_workspace_idx ON groups(workspace_ref, ref);

CREATE TABLE projects (
    ref TEXT PRIMARY KEY CHECK (
        length(trim(ref)) > 0
        AND instr(ref, char(0)) = 0
        AND instr(ref, char(10)) = 0
        AND instr(ref, char(13)) = 0
    ),
    group_ref TEXT NOT NULL REFERENCES groups(ref) ON DELETE RESTRICT
) STRICT;

CREATE INDEX projects_group_idx ON projects(group_ref, ref);

CREATE TABLE repositories (
    ref TEXT PRIMARY KEY CHECK (
        length(trim(ref)) > 0
        AND instr(ref, char(0)) = 0
        AND instr(ref, char(10)) = 0
        AND instr(ref, char(13)) = 0
    ),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT
) STRICT;

CREATE INDEX repositories_project_idx ON repositories(project_ref, ref);

CREATE TRIGGER workspaces_immutable_update
BEFORE UPDATE ON workspaces
BEGIN
    SELECT RAISE(ABORT, 'sqlite.workspace_immutable');
END;

CREATE TRIGGER workspaces_immutable_delete
BEFORE DELETE ON workspaces
BEGIN
    SELECT RAISE(ABORT, 'sqlite.workspace_immutable');
END;

CREATE TRIGGER groups_immutable_update
BEFORE UPDATE ON groups
BEGIN
    SELECT RAISE(ABORT, 'sqlite.group_immutable');
END;

CREATE TRIGGER groups_immutable_delete
BEFORE DELETE ON groups
BEGIN
    SELECT RAISE(ABORT, 'sqlite.group_immutable');
END;

CREATE TRIGGER projects_immutable_update
BEFORE UPDATE ON projects
BEGIN
    SELECT RAISE(ABORT, 'sqlite.project_immutable');
END;

CREATE TRIGGER projects_immutable_delete
BEFORE DELETE ON projects
BEGIN
    SELECT RAISE(ABORT, 'sqlite.project_immutable');
END;

CREATE TRIGGER repositories_immutable_update
BEFORE UPDATE ON repositories
BEGIN
    SELECT RAISE(ABORT, 'sqlite.repository_immutable');
END;

CREATE TRIGGER repositories_immutable_delete
BEFORE DELETE ON repositories
BEGIN
    SELECT RAISE(ABORT, 'sqlite.repository_immutable');
END;

CREATE TABLE project_memberships (
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    role TEXT NOT NULL CHECK (role IN (
        'platform_admin', 'project_owner', 'project_admin', 'contributor',
        'reviewer', 'operator', 'viewer'
    )),
    revision INTEGER NOT NULL CHECK (revision > 0),
    status TEXT NOT NULL CHECK (status IN ('active', 'revoked')),
    granted_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    granted_at INTEGER NOT NULL,
    revoked_by_ref TEXT REFERENCES principals(ref) ON DELETE RESTRICT,
    revoked_at INTEGER,
    PRIMARY KEY(principal_ref, project_ref),
    CHECK (
        (status = 'active' AND revoked_by_ref IS NULL AND revoked_at IS NULL)
        OR
        (status = 'revoked' AND revision >= 2
            AND revoked_by_ref IS NOT NULL AND revoked_at IS NOT NULL
            AND revoked_at >= granted_at)
    )
) STRICT;

CREATE INDEX project_memberships_project_status_idx
    ON project_memberships(project_ref, status, role, principal_ref);

CREATE TRIGGER project_memberships_revision_guard
BEFORE UPDATE ON project_memberships
WHEN NEW.principal_ref <> OLD.principal_ref
  OR NEW.project_ref <> OLD.project_ref
  OR NEW.revision <> OLD.revision + 1
  OR (NEW.status = 'revoked' AND (
        OLD.status <> 'active'
        OR NEW.role <> OLD.role
        OR NEW.granted_by_ref <> OLD.granted_by_ref
        OR NEW.granted_at <> OLD.granted_at
  ))
BEGIN
    SELECT RAISE(ABORT, 'sqlite.membership_revision_conflict');
END;

CREATE TRIGGER project_memberships_immutable_delete
BEFORE DELETE ON project_memberships
BEGIN
    SELECT RAISE(ABORT, 'sqlite.membership_immutable');
END;

CREATE TABLE membership_audit_receipts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    action TEXT NOT NULL CHECK (action IN ('membership.granted', 'membership.revoked')),
    actor_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    target_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN (
        'platform_admin', 'project_owner', 'project_admin', 'contributor',
        'reviewer', 'operator', 'viewer'
    )),
    previous_revision INTEGER NOT NULL CHECK (previous_revision >= 0),
    revision INTEGER NOT NULL CHECK (revision = previous_revision + 1),
    occurred_at INTEGER NOT NULL,
    UNIQUE(actor_ref, request_ref),
    FOREIGN KEY(target_ref, project_ref)
        REFERENCES project_memberships(principal_ref, project_ref) ON DELETE RESTRICT,
    CHECK (action <> 'membership.revoked' OR previous_revision > 0)
) STRICT;

CREATE INDEX membership_audit_project_idx
    ON membership_audit_receipts(project_ref, occurred_at, ref);

CREATE TRIGGER membership_audit_receipts_immutable_update
BEFORE UPDATE ON membership_audit_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.membership_audit_immutable');
END;

CREATE TRIGGER membership_audit_receipts_immutable_delete
BEFORE DELETE ON membership_audit_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.membership_audit_immutable');
END;

CREATE TABLE authorization_receipts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL CHECK (length(trim(project_ref)) > 0),
    permission TEXT NOT NULL CHECK (permission IN (
        'project.hierarchy.manage', 'project.membership.manage',
        'goals.create', 'goals.amend', 'goals.get', 'goals.list',
        'artifacts.read', 'project.status'
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

CREATE INDEX authorization_receipts_project_idx
    ON authorization_receipts(project_ref, recorded_at, ref);

CREATE TRIGGER authorization_receipts_immutable_update
BEFORE UPDATE ON authorization_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.authorization_receipt_immutable');
END;

CREATE TRIGGER authorization_receipts_immutable_delete
BEFORE DELETE ON authorization_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.authorization_receipt_immutable');
END;

CREATE TABLE goals_v6 (
    ref TEXT PRIMARY KEY,
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    requested_by_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    app_spec_ref TEXT NOT NULL UNIQUE REFERENCES app_specs(ref) ON DELETE RESTRICT,
    actor_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending', 'running', 'succeeded', 'failed')),
    revision INTEGER NOT NULL CHECK (revision > 0),
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    closed_at INTEGER,
    plan_generation INTEGER NOT NULL CHECK (plan_generation >= 0),
    UNIQUE(requested_by_ref, project_ref, request_ref)
) STRICT;

INSERT INTO goals_v6(
    ref, request_ref, request_fingerprint, requested_by_ref, app_spec_ref,
    actor_ref, project_ref, state, revision, created_at, started_at, closed_at,
    plan_generation
)
SELECT ref, request_ref, request_fingerprint, 'migration:v09:' || actor_ref, app_spec_ref,
       actor_ref, project_ref, state, revision, created_at, started_at, closed_at,
       plan_generation
FROM goals;

DROP TABLE goals;
ALTER TABLE goals_v6 RENAME TO goals;

CREATE INDEX goals_actor_project_created_idx
    ON goals(actor_ref, project_ref, created_at DESC, ref DESC);

CREATE INDEX goals_project_created_idx
    ON goals(project_ref, created_at DESC, ref DESC);

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
BEFORE UPDATE OF request_ref, request_fingerprint, requested_by_ref,
    app_spec_ref, actor_ref, project_ref ON goals
BEGIN
    SELECT RAISE(ABORT, 'sqlite.goal_app_spec_immutable');
END;
