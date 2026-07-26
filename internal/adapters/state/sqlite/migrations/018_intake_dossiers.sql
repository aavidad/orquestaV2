-- V23: immutable content-addressed Wizard dossiers. Generation is a durable
-- request fact; confirmation, freeze and Goal creation remain later commands.
CREATE TABLE intake_dossiers (
    ref TEXT PRIMARY KEY CHECK (
        length(ref) = 79
        AND substr(ref, 1, 15) = 'intake-dossier:'
        AND substr(ref, 16) NOT GLOB '*[^0-9a-f]*'
    ),
    actor_ref TEXT NOT NULL CHECK (
        length(trim(actor_ref)) > 0
        AND instr(actor_ref, char(0)) = 0
        AND instr(actor_ref, char(10)) = 0
        AND instr(actor_ref, char(13)) = 0
    ),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    state_ref TEXT NOT NULL CHECK (
        state_ref GLOB 'intake:?*'
        AND length(state_ref) <= 512
        AND state_ref = trim(state_ref)
        AND instr(state_ref, char(0)) = 0
        AND instr(state_ref, char(10)) = 0
        AND instr(state_ref, char(13)) = 0
    ),
    state_revision INTEGER NOT NULL CHECK (state_revision > 0),
    state_digest TEXT NOT NULL CHECK (
        length(state_digest) = 64
        AND state_digest NOT GLOB '*[^0-9a-f]*'
    ),
    source_intake_receipt_ref TEXT NOT NULL
        REFERENCES intake_receipts(ref) ON DELETE RESTRICT,
    schema_ref TEXT NOT NULL CHECK (
        schema_ref = 'orquesta.intake.dossier.v1'
    ),
    plan_digest TEXT NOT NULL CHECK (
        length(plan_digest) = 64
        AND plan_digest NOT GLOB '*[^0-9a-f]*'
    ),
    dossier_digest TEXT NOT NULL CHECK (
        length(dossier_digest) = 64
        AND dossier_digest NOT GLOB '*[^0-9a-f]*'
    ),
    snapshot_json TEXT NOT NULL CHECK (
        json_valid(snapshot_json) AND json_type(snapshot_json) = 'object'
    ),
    generation_receipt_ref TEXT NOT NULL UNIQUE,
    CHECK (ref = 'intake-dossier:' || dossier_digest),
    FOREIGN KEY(state_ref, actor_ref, project_ref)
        REFERENCES intake_states(state_ref, actor_ref, project_ref)
        ON DELETE RESTRICT,
    FOREIGN KEY(generation_receipt_ref)
        REFERENCES intake_dossier_generation_receipts(ref)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
) STRICT;

CREATE INDEX intake_dossiers_scope_idx
    ON intake_dossiers(actor_ref, project_ref, state_ref, state_revision, ref);

CREATE TABLE intake_dossier_generation_receipts (
    ref TEXT PRIMARY KEY CHECK (
        length(ref) = 98
        AND substr(ref, 1, 34) = 'intake-dossier-generation-receipt:'
        AND substr(ref, 35) NOT GLOB '*[^0-9a-f]*'
    ),
    request_ref TEXT NOT NULL CHECK (
        length(trim(request_ref)) > 0
        AND length(request_ref) <= 512
        AND instr(request_ref, char(0)) = 0
        AND instr(request_ref, char(10)) = 0
        AND instr(request_ref, char(13)) = 0
    ),
    request_fingerprint TEXT NOT NULL CHECK (
        length(request_fingerprint) = 64
        AND request_fingerprint NOT GLOB '*[^0-9a-f]*'
    ),
    actor_ref TEXT NOT NULL CHECK (
        length(trim(actor_ref)) > 0
        AND instr(actor_ref, char(0)) = 0
        AND instr(actor_ref, char(10)) = 0
        AND instr(actor_ref, char(13)) = 0
    ),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    state_ref TEXT NOT NULL CHECK (
        state_ref GLOB 'intake:?*'
        AND length(state_ref) <= 512
        AND state_ref = trim(state_ref)
        AND instr(state_ref, char(0)) = 0
        AND instr(state_ref, char(10)) = 0
        AND instr(state_ref, char(13)) = 0
    ),
    state_revision INTEGER NOT NULL CHECK (state_revision > 0),
    state_digest TEXT NOT NULL CHECK (
        length(state_digest) = 64
        AND state_digest NOT GLOB '*[^0-9a-f]*'
    ),
    source_intake_receipt_ref TEXT NOT NULL
        REFERENCES intake_receipts(ref) ON DELETE RESTRICT,
    dossier_ref TEXT NOT NULL,
    dossier_digest TEXT NOT NULL CHECK (
        length(dossier_digest) = 64
        AND dossier_digest NOT GLOB '*[^0-9a-f]*'
    ),
    plan_digest TEXT NOT NULL CHECK (
        length(plan_digest) = 64
        AND plan_digest NOT GLOB '*[^0-9a-f]*'
    ),
    authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    CHECK (dossier_ref = 'intake-dossier:' || dossier_digest),
    UNIQUE(actor_ref, project_ref, request_ref),
    FOREIGN KEY(dossier_ref) REFERENCES intake_dossiers(ref)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
) STRICT;

CREATE INDEX intake_dossier_generation_receipts_dossier_idx
    ON intake_dossier_generation_receipts(
        actor_ref, project_ref, dossier_ref, ref
    );

CREATE TRIGGER intake_dossier_generation_receipts_authorization_guard
BEFORE INSERT ON intake_dossier_generation_receipts
WHEN NOT EXISTS (
    SELECT 1
    FROM authorization_receipts authorization
    JOIN principals principal ON principal.ref = authorization.principal_ref
    WHERE authorization.ref = NEW.authorization_receipt_ref
      AND authorization.request_ref =
          'authorization-request:intake-dossier-generate:' || NEW.request_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.create'
      AND authorization.resource_ref = NEW.project_ref
      AND authorization.outcome = 'allowed'
      AND principal.actor_ref = NEW.actor_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_authorization_invalid');
END;

CREATE TRIGGER intake_dossier_generation_receipts_current_source_guard
BEFORE INSERT ON intake_dossier_generation_receipts
WHEN NOT EXISTS (
    SELECT 1
    FROM intake_states state
    WHERE state.state_ref = NEW.state_ref
      AND state.actor_ref = NEW.actor_ref
      AND state.project_ref = NEW.project_ref
      AND state.revision = NEW.state_revision
      AND state.state_digest = NEW.state_digest
      AND state.receipt_ref = NEW.source_intake_receipt_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_source_conflict');
END;

CREATE TRIGGER intake_dossier_generation_receipts_existing_dossier_guard
BEFORE INSERT ON intake_dossier_generation_receipts
WHEN EXISTS (
    SELECT 1 FROM intake_dossiers dossier
    WHERE dossier.ref = NEW.dossier_ref
)
AND NOT EXISTS (
    SELECT 1
    FROM intake_dossiers dossier
    WHERE dossier.ref = NEW.dossier_ref
      AND dossier.actor_ref = NEW.actor_ref
      AND dossier.project_ref = NEW.project_ref
      AND dossier.state_ref = NEW.state_ref
      AND dossier.state_revision = NEW.state_revision
      AND dossier.state_digest = NEW.state_digest
      AND dossier.source_intake_receipt_ref = NEW.source_intake_receipt_ref
      AND dossier.plan_digest = NEW.plan_digest
      AND dossier.dossier_digest = NEW.dossier_digest
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_binding_invalid');
END;

CREATE TRIGGER intake_dossiers_generation_receipt_guard
BEFORE INSERT ON intake_dossiers
WHEN NOT EXISTS (
    SELECT 1
    FROM intake_dossier_generation_receipts receipt
    WHERE receipt.ref = NEW.generation_receipt_ref
      AND receipt.actor_ref = NEW.actor_ref
      AND receipt.project_ref = NEW.project_ref
      AND receipt.state_ref = NEW.state_ref
      AND receipt.state_revision = NEW.state_revision
      AND receipt.state_digest = NEW.state_digest
      AND receipt.source_intake_receipt_ref = NEW.source_intake_receipt_ref
      AND receipt.dossier_ref = NEW.ref
      AND receipt.dossier_digest = NEW.dossier_digest
      AND receipt.plan_digest = NEW.plan_digest
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_generation_receipt_invalid');
END;

CREATE TRIGGER intake_dossiers_immutable_update
BEFORE UPDATE ON intake_dossiers
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_immutable');
END;

CREATE TRIGGER intake_dossiers_immutable_delete
BEFORE DELETE ON intake_dossiers
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_immutable');
END;

CREATE TRIGGER intake_dossier_generation_receipts_immutable_update
BEFORE UPDATE ON intake_dossier_generation_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_generation_receipt_immutable');
END;

CREATE TRIGGER intake_dossier_generation_receipts_immutable_delete
BEFORE DELETE ON intake_dossier_generation_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_generation_receipt_immutable');
END;
