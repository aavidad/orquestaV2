-- V23: reserve exact Wizard gaps invocations that produce no Intake mutation.
-- The historical Intake receipt remains the state snapshot authority; this
-- outcome is a separate immutable request receipt and never advances revision.
CREATE TABLE wizard_gaps_outcomes (
    ref TEXT PRIMARY KEY CHECK (
        length(ref) = 84
        AND substr(ref, 1, 20) = 'wizard-gaps-outcome:'
        AND substr(ref, 21) NOT GLOB '*[^0-9a-f]*'
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
    expected_revision INTEGER NOT NULL CHECK (expected_revision > 0),
    source_intake_receipt_ref TEXT NOT NULL
        REFERENCES intake_receipts(ref) ON DELETE RESTRICT,
    evaluator_schema TEXT NOT NULL CHECK (
        length(trim(evaluator_schema)) > 0
        AND evaluator_schema = trim(evaluator_schema)
        AND instr(evaluator_schema, char(0)) = 0
        AND instr(evaluator_schema, char(10)) = 0
        AND instr(evaluator_schema, char(13)) = 0
    ),
    evaluator_version TEXT NOT NULL CHECK (
        length(trim(evaluator_version)) > 0
        AND evaluator_version = trim(evaluator_version)
        AND instr(evaluator_version, char(0)) = 0
        AND instr(evaluator_version, char(10)) = 0
        AND instr(evaluator_version, char(13)) = 0
    ),
    evaluator_semantic_digest TEXT NOT NULL CHECK (
        length(evaluator_semantic_digest) = 64
        AND evaluator_semantic_digest NOT GLOB '*[^0-9a-f]*'
    ),
    evaluation_digest TEXT NOT NULL CHECK (
        length(evaluation_digest) = 64
        AND evaluation_digest NOT GLOB '*[^0-9a-f]*'
    ),
    authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    UNIQUE(actor_ref, project_ref, request_ref)
) STRICT;

CREATE INDEX wizard_gaps_outcomes_scope_idx
    ON wizard_gaps_outcomes(actor_ref, project_ref, state_ref, expected_revision);

CREATE TRIGGER wizard_gaps_outcomes_authorization_guard
BEFORE INSERT ON wizard_gaps_outcomes
WHEN NOT EXISTS (
    SELECT 1
    FROM authorization_receipts authorization
    JOIN principals principal ON principal.ref = authorization.principal_ref
    WHERE authorization.ref = NEW.authorization_receipt_ref
      AND authorization.request_ref =
          'authorization-request:intake-apply:' || NEW.request_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.create'
      AND authorization.resource_ref = NEW.project_ref
      AND authorization.outcome = 'allowed'
      AND principal.actor_ref = NEW.actor_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_outcome_authorization_invalid');
END;

CREATE TRIGGER wizard_gaps_outcomes_binding_guard
BEFORE INSERT ON wizard_gaps_outcomes
WHEN NOT EXISTS (
    SELECT 1
    FROM intake_receipts receipt
    JOIN intake_states state
      ON state.state_ref = receipt.state_ref
     AND state.actor_ref = receipt.actor_ref
     AND state.project_ref = receipt.project_ref
    WHERE receipt.ref = NEW.source_intake_receipt_ref
      AND receipt.actor_ref = NEW.actor_ref
      AND receipt.project_ref = NEW.project_ref
      AND receipt.state_ref = NEW.state_ref
      AND receipt.revision = NEW.expected_revision
      AND state.revision = NEW.expected_revision
      AND state.receipt_ref = NEW.source_intake_receipt_ref
      AND state.state_digest = receipt.state_digest
      AND state.snapshot_json = receipt.snapshot_json
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_outcome_binding_invalid');
END;

CREATE TRIGGER wizard_gaps_outcomes_request_guard
BEFORE INSERT ON wizard_gaps_outcomes
WHEN EXISTS (
    SELECT 1
    FROM intake_receipts receipt
    WHERE receipt.actor_ref = NEW.actor_ref
      AND receipt.project_ref = NEW.project_ref
      AND receipt.request_ref = NEW.request_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_outcome_request_conflict');
END;

CREATE TRIGGER intake_receipts_wizard_gaps_request_guard
BEFORE INSERT ON intake_receipts
WHEN EXISTS (
    SELECT 1
    FROM wizard_gaps_outcomes outcome
    WHERE outcome.actor_ref = NEW.actor_ref
      AND outcome.project_ref = NEW.project_ref
      AND outcome.request_ref = NEW.request_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_request_reserved_by_wizard_gaps_outcome');
END;

CREATE TRIGGER wizard_gaps_outcomes_immutable_update
BEFORE UPDATE ON wizard_gaps_outcomes
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_outcome_immutable');
END;

CREATE TRIGGER wizard_gaps_outcomes_immutable_delete
BEFORE DELETE ON wizard_gaps_outcomes
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_outcome_immutable');
END;
