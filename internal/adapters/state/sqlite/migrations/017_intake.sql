-- V23: one durable Wizard intake authority. The current snapshot is mutable
-- only by revision CAS; every accepted command also retains its immutable
-- snapshot and receipt so exact replay survives later revisions and restarts.
CREATE TABLE intake_states (
    state_ref TEXT NOT NULL CHECK (
        state_ref GLOB 'intake:?*'
        AND length(state_ref) <= 512
        AND state_ref = trim(state_ref)
        AND instr(state_ref, char(0)) = 0
        AND instr(state_ref, char(10)) = 0
        AND instr(state_ref, char(13)) = 0
    ),
    actor_ref TEXT NOT NULL CHECK (
        length(trim(actor_ref)) > 0
        AND instr(actor_ref, char(0)) = 0
        AND instr(actor_ref, char(10)) = 0
        AND instr(actor_ref, char(13)) = 0
    ),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    schema_ref TEXT NOT NULL CHECK (schema_ref = 'orquesta.intake.state.v1'),
    revision INTEGER NOT NULL CHECK (revision > 0),
    max_question_rounds INTEGER NOT NULL CHECK (max_question_rounds > 0),
    question_rounds INTEGER NOT NULL CHECK (
        question_rounds >= 0 AND question_rounds <= max_question_rounds
    ),
    state_digest TEXT NOT NULL CHECK (
        length(state_digest) = 64
        AND state_digest NOT GLOB '*[^0-9a-f]*'
    ),
    snapshot_json TEXT NOT NULL CHECK (
        json_valid(snapshot_json) AND json_type(snapshot_json) = 'object'
    ),
    receipt_ref TEXT NOT NULL UNIQUE CHECK (
        length(trim(receipt_ref)) > 0
        AND instr(receipt_ref, char(0)) = 0
        AND instr(receipt_ref, char(10)) = 0
        AND instr(receipt_ref, char(13)) = 0
    ),
    PRIMARY KEY(state_ref, actor_ref, project_ref),
    FOREIGN KEY(receipt_ref) REFERENCES intake_receipts(ref)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
) STRICT;

CREATE INDEX intake_states_scope_idx
    ON intake_states(actor_ref, project_ref, state_ref);

CREATE TABLE intake_receipts (
    ref TEXT PRIMARY KEY CHECK (
        length(ref) = 79
        AND substr(ref, 1, 15) = 'intake-receipt:'
        AND substr(ref, 16) NOT GLOB '*[^0-9a-f]*'
    ),
    request_ref TEXT NOT NULL CHECK (
        length(trim(request_ref)) > 0
        AND instr(request_ref, char(0)) = 0
        AND instr(request_ref, char(10)) = 0
        AND instr(request_ref, char(13)) = 0
    ),
    request_fingerprint TEXT NOT NULL CHECK (
        length(request_fingerprint) = 64
        AND request_fingerprint NOT GLOB '*[^0-9a-f]*'
    ),
    operation TEXT NOT NULL CHECK (operation IN ('create', 'apply')),
    actor_ref TEXT NOT NULL CHECK (
        length(trim(actor_ref)) > 0
        AND instr(actor_ref, char(0)) = 0
        AND instr(actor_ref, char(10)) = 0
        AND instr(actor_ref, char(13)) = 0
    ),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    state_ref TEXT NOT NULL,
    previous_revision INTEGER NOT NULL CHECK (previous_revision >= 0),
    revision INTEGER NOT NULL CHECK (
        revision > 0 AND revision = previous_revision + 1
    ),
    state_digest TEXT NOT NULL CHECK (
        length(state_digest) = 64
        AND state_digest NOT GLOB '*[^0-9a-f]*'
    ),
    authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    snapshot_json TEXT NOT NULL CHECK (
        json_valid(snapshot_json) AND json_type(snapshot_json) = 'object'
    ),
    UNIQUE(actor_ref, project_ref, request_ref),
    UNIQUE(state_ref, actor_ref, project_ref, revision),
    FOREIGN KEY(state_ref, actor_ref, project_ref)
        REFERENCES intake_states(state_ref, actor_ref, project_ref)
        ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
) STRICT;

CREATE INDEX intake_receipts_state_revision_idx
    ON intake_receipts(state_ref, revision, ref);

CREATE TRIGGER intake_receipts_authorization_guard
BEFORE INSERT ON intake_receipts
WHEN NOT EXISTS (
    SELECT 1
    FROM authorization_receipts authorization
    JOIN principals principal ON principal.ref = authorization.principal_ref
    WHERE authorization.ref = NEW.authorization_receipt_ref
      AND authorization.request_ref = CASE NEW.operation
          WHEN 'create' THEN 'authorization-request:intake-create:' || NEW.request_ref
          WHEN 'apply' THEN 'authorization-request:intake-apply:' || NEW.request_ref
      END
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.create'
      AND authorization.resource_ref = NEW.project_ref
      AND authorization.outcome = 'allowed'
      AND principal.actor_ref = NEW.actor_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_authorization_invalid');
END;

CREATE TRIGGER intake_states_insert_receipt_guard
BEFORE INSERT ON intake_states
WHEN NOT EXISTS (
    SELECT 1
    FROM intake_receipts receipt
    WHERE receipt.ref = NEW.receipt_ref
      AND receipt.operation = 'create'
      AND receipt.actor_ref = NEW.actor_ref
      AND receipt.project_ref = NEW.project_ref
      AND receipt.state_ref = NEW.state_ref
      AND receipt.previous_revision = 0
      AND receipt.revision = NEW.revision
      AND receipt.state_digest = NEW.state_digest
      AND receipt.snapshot_json = NEW.snapshot_json
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_creation_receipt_invalid');
END;

CREATE TRIGGER intake_states_revision_guard
BEFORE UPDATE ON intake_states
WHEN NEW.state_ref <> OLD.state_ref
  OR NEW.actor_ref <> OLD.actor_ref
  OR NEW.project_ref <> OLD.project_ref
  OR NEW.schema_ref <> OLD.schema_ref
  OR NEW.max_question_rounds <> OLD.max_question_rounds
  OR NEW.revision <> OLD.revision + 1
  OR NEW.question_rounds < OLD.question_rounds
  OR NEW.receipt_ref = OLD.receipt_ref
  OR NOT EXISTS (
      SELECT 1
      FROM intake_receipts receipt
      WHERE receipt.ref = NEW.receipt_ref
        AND receipt.operation = 'apply'
        AND receipt.actor_ref = NEW.actor_ref
        AND receipt.project_ref = NEW.project_ref
        AND receipt.state_ref = NEW.state_ref
        AND receipt.previous_revision = OLD.revision
        AND receipt.revision = NEW.revision
        AND receipt.state_digest = NEW.state_digest
        AND receipt.snapshot_json = NEW.snapshot_json
  )
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_revision_conflict');
END;

CREATE TRIGGER intake_states_immutable_delete
BEFORE DELETE ON intake_states
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_state_immutable');
END;

CREATE TRIGGER intake_receipts_immutable_update
BEFORE UPDATE ON intake_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_receipt_immutable');
END;

CREATE TRIGGER intake_receipts_immutable_delete
BEFORE DELETE ON intake_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_receipt_immutable');
END;
