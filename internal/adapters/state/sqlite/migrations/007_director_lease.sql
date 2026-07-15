CREATE TABLE authorization_receipts_v7 (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    project_ref TEXT NOT NULL CHECK (length(trim(project_ref)) > 0),
    permission TEXT NOT NULL CHECK (permission IN (
        'project.hierarchy.manage', 'project.membership.manage',
        'goals.create', 'goals.amend', 'goals.get', 'goals.list', 'goals.direct',
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

INSERT INTO authorization_receipts_v7(
    ref, request_ref, request_fingerprint, principal_ref, project_ref,
    permission, resource_ref, requested_at, outcome, role,
    membership_revision, reason_code, decided_at, recorded_at
)
SELECT ref, request_ref, request_fingerprint, principal_ref, project_ref,
       permission, resource_ref, requested_at, outcome, role,
       membership_revision, reason_code, decided_at, recorded_at
FROM authorization_receipts;

DROP TABLE authorization_receipts;
ALTER TABLE authorization_receipts_v7 RENAME TO authorization_receipts;

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

CREATE UNIQUE INDEX goals_ref_project_v7_idx ON goals(ref, project_ref);

CREATE TABLE director_leases (
    goal_ref TEXT PRIMARY KEY,
    project_ref TEXT NOT NULL,
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    token TEXT NOT NULL UNIQUE CHECK (length(trim(token)) > 0),
    fence INTEGER NOT NULL CHECK (fence > 0),
    lease_until INTEGER NOT NULL,
    claim_request_ref TEXT NOT NULL CHECK (length(trim(claim_request_ref)) > 0),
    claim_request_fingerprint TEXT NOT NULL CHECK (length(trim(claim_request_fingerprint)) > 0),
    claim_authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    renew_request_ref TEXT CHECK (renew_request_ref IS NULL OR length(trim(renew_request_ref)) > 0),
    renew_request_fingerprint TEXT CHECK (
        renew_request_fingerprint IS NULL OR length(trim(renew_request_fingerprint)) > 0
    ),
    renew_authorization_receipt_ref TEXT
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT,
    CHECK (
        (renew_request_ref IS NULL
            AND renew_request_fingerprint IS NULL
            AND renew_authorization_receipt_ref IS NULL)
        OR
        (renew_request_ref IS NOT NULL
            AND renew_request_fingerprint IS NOT NULL
            AND renew_authorization_receipt_ref IS NOT NULL)
    )
) STRICT;

CREATE INDEX director_leases_principal_idx
    ON director_leases(principal_ref, lease_until, goal_ref);

CREATE UNIQUE INDEX director_leases_claim_request_idx
    ON director_leases(principal_ref, project_ref, claim_request_ref);

CREATE UNIQUE INDEX director_leases_renew_request_idx
    ON director_leases(principal_ref, project_ref, renew_request_ref)
    WHERE renew_request_ref IS NOT NULL;

CREATE TRIGGER director_leases_immutable_delete
BEFORE DELETE ON director_leases
BEGIN
    SELECT RAISE(ABORT, 'sqlite.director_lease_immutable');
END;

CREATE TABLE director_lease_receipts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    action TEXT NOT NULL CHECK (action IN ('claim', 'renew')),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    goal_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    fence INTEGER NOT NULL CHECK (fence > 0),
    lease_until INTEGER NOT NULL,
    occurred_at INTEGER NOT NULL,
    UNIQUE(principal_ref, project_ref, request_ref),
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT
) STRICT;

CREATE INDEX director_lease_receipts_goal_idx
    ON director_lease_receipts(goal_ref, fence, occurred_at, ref);

CREATE UNIQUE INDEX director_lease_receipts_claim_fence_idx
    ON director_lease_receipts(goal_ref, fence)
    WHERE action = 'claim';

CREATE TRIGGER director_leases_update_guard
BEFORE UPDATE ON director_leases
WHEN NOT (
    (
        NEW.goal_ref = OLD.goal_ref
        AND NEW.project_ref = OLD.project_ref
        AND NEW.fence = OLD.fence
        AND NEW.principal_ref = OLD.principal_ref
        AND NEW.token = OLD.token
        AND NEW.claim_request_ref = OLD.claim_request_ref
        AND NEW.claim_request_fingerprint = OLD.claim_request_fingerprint
        AND NEW.claim_authorization_receipt_ref = OLD.claim_authorization_receipt_ref
		AND NEW.renew_request_ref IS NOT NULL
		AND (OLD.renew_request_ref IS NULL OR NEW.renew_request_ref <> OLD.renew_request_ref)
		AND NEW.renew_request_fingerprint IS NOT NULL
		AND NEW.renew_authorization_receipt_ref IS NOT NULL
        AND (OLD.renew_authorization_receipt_ref IS NULL
            OR NEW.renew_authorization_receipt_ref <> OLD.renew_authorization_receipt_ref)
        AND NEW.updated_at > OLD.updated_at
        AND NEW.updated_at < OLD.lease_until
        AND NEW.lease_until > OLD.lease_until
        AND NEW.lease_until > NEW.updated_at
        AND EXISTS (
            SELECT 1 FROM director_lease_receipts receipt
            WHERE receipt.action = 'renew'
              AND receipt.request_ref = NEW.renew_request_ref
              AND receipt.request_fingerprint = NEW.renew_request_fingerprint
              AND receipt.authorization_receipt_ref = NEW.renew_authorization_receipt_ref
              AND receipt.goal_ref = NEW.goal_ref
              AND receipt.project_ref = NEW.project_ref
              AND receipt.principal_ref = NEW.principal_ref
              AND receipt.fence = NEW.fence
              AND receipt.lease_until = NEW.lease_until
              AND receipt.occurred_at = NEW.updated_at
        )
    )
    OR
    (
        NEW.goal_ref = OLD.goal_ref
        AND NEW.project_ref = OLD.project_ref
        AND NEW.fence = OLD.fence + 1
		AND NEW.token <> OLD.token
		AND NEW.claim_request_ref <> OLD.claim_request_ref
		AND NEW.claim_authorization_receipt_ref <> OLD.claim_authorization_receipt_ref
        AND NEW.renew_request_ref IS NULL
        AND NEW.renew_request_fingerprint IS NULL
        AND NEW.renew_authorization_receipt_ref IS NULL
        AND NEW.updated_at >= OLD.lease_until
        AND NEW.lease_until > NEW.updated_at
        AND EXISTS (
            SELECT 1 FROM director_lease_receipts receipt
            WHERE receipt.action = 'claim'
              AND receipt.request_ref = NEW.claim_request_ref
              AND receipt.request_fingerprint = NEW.claim_request_fingerprint
              AND receipt.authorization_receipt_ref = NEW.claim_authorization_receipt_ref
              AND receipt.goal_ref = NEW.goal_ref
              AND receipt.project_ref = NEW.project_ref
              AND receipt.principal_ref = NEW.principal_ref
              AND receipt.fence = NEW.fence
              AND receipt.lease_until = NEW.lease_until
              AND receipt.occurred_at = NEW.updated_at
        )
    )
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.director_lease_update_invalid');
END;

CREATE TRIGGER director_lease_receipts_immutable_update
BEFORE UPDATE ON director_lease_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.director_lease_receipt_immutable');
END;

CREATE TRIGGER director_lease_receipts_immutable_delete
BEFORE DELETE ON director_lease_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.director_lease_receipt_immutable');
END;

CREATE TABLE director_decisions (
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
    applied_goal_revision INTEGER NOT NULL CHECK (
        applied_goal_revision = source_goal_revision + 1
    ),
    applied_plan_generation INTEGER NOT NULL CHECK (
        applied_plan_generation = source_plan_generation + 1
    ),
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    decided_at INTEGER NOT NULL,
    UNIQUE(principal_ref, project_ref, request_ref),
    FOREIGN KEY(goal_ref, project_ref) REFERENCES goals(ref, project_ref) ON DELETE RESTRICT
) STRICT;

CREATE UNIQUE INDEX director_decisions_goal_idx
    ON director_decisions(goal_ref, applied_plan_generation);

CREATE TRIGGER director_decisions_immutable_update
BEFORE UPDATE ON director_decisions
BEGIN
    SELECT RAISE(ABORT, 'sqlite.director_decision_immutable');
END;

CREATE TRIGGER director_decisions_immutable_delete
BEFORE DELETE ON director_decisions
BEGIN
    SELECT RAISE(ABORT, 'sqlite.director_decision_immutable');
END;
