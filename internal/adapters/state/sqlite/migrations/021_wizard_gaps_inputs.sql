-- V23: bind every new public Wizard gaps request to its exact canonical input.
-- The row is committed in the same transaction as either the Intake mutation
-- or the immutable no-op outcome named by outcome_receipt_ref.
CREATE TABLE wizard_gaps_input_receipts (
    ref TEXT PRIMARY KEY CHECK (
        length(ref) = 82
        AND substr(ref, 1, 18) = 'wizard-gaps-input:'
        AND substr(ref, 19) NOT GLOB '*[^0-9a-f]*'
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
    origin TEXT NOT NULL CHECK (origin IN ('chat', 'form')),
    source_intake_receipt_ref TEXT NOT NULL
        REFERENCES intake_receipts(ref) ON DELETE RESTRICT,
    outcome_kind TEXT NOT NULL CHECK (
        outcome_kind IN ('intake_mutation', 'wizard_gaps_noop')
    ),
    outcome_receipt_ref TEXT NOT NULL CHECK (
        length(trim(outcome_receipt_ref)) > 0
        AND instr(outcome_receipt_ref, char(0)) = 0
        AND instr(outcome_receipt_ref, char(10)) = 0
        AND instr(outcome_receipt_ref, char(13)) = 0
    ),
    fact_surface TEXT NOT NULL CHECK (
        fact_surface IN (
            '', 'human_ui', 'native_mobile', 'native_desktop',
            'server_service', 'kernel_module'
        )
    ),
    fact_sharing_intent TEXT NOT NULL CHECK (
        fact_sharing_intent IN ('', 'ambiguous', 'personal', 'shared')
    ),
    fact_corporate_identity TEXT NOT NULL CHECK (
        fact_corporate_identity IN ('', 'missing', 'declared')
    ),
    fact_target_users TEXT NOT NULL CHECK (
        fact_target_users IN ('', 'missing', 'declared')
    ),
    fact_integration_auth TEXT NOT NULL CHECK (
        fact_integration_auth IN ('', 'missing', 'declared')
    ),
    fact_integration_criticality TEXT NOT NULL CHECK (
        fact_integration_criticality IN ('', 'missing', 'declared')
    ),
    facts_digest TEXT NOT NULL CHECK (
        length(facts_digest) = 64
        AND facts_digest NOT GLOB '*[^0-9a-f]*'
    ),
    pack_refs_json TEXT NOT NULL CHECK (
        json_valid(pack_refs_json)
        AND json_type(pack_refs_json) = 'array'
    ),
    pack_refs_digest TEXT NOT NULL CHECK (
        length(pack_refs_digest) = 64
        AND pack_refs_digest NOT GLOB '*[^0-9a-f]*'
    ),
    selections_json TEXT NOT NULL CHECK (
        json_valid(selections_json)
        AND json_type(selections_json) = 'array'
    ),
    selections_digest TEXT NOT NULL CHECK (
        length(selections_digest) = 64
        AND selections_digest NOT GLOB '*[^0-9a-f]*'
    ),
    evaluator_schema TEXT NOT NULL CHECK (
        length(trim(evaluator_schema)) > 0
        AND evaluator_schema = trim(evaluator_schema)
    ),
    evaluator_version TEXT NOT NULL CHECK (
        length(trim(evaluator_version)) > 0
        AND evaluator_version = trim(evaluator_version)
    ),
    evaluator_semantic_digest TEXT NOT NULL CHECK (
        length(evaluator_semantic_digest) = 64
        AND evaluator_semantic_digest NOT GLOB '*[^0-9a-f]*'
    ),
    authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    UNIQUE(actor_ref, project_ref, request_ref)
) STRICT;

CREATE INDEX wizard_gaps_input_receipts_scope_idx
    ON wizard_gaps_input_receipts(
        actor_ref, project_ref, state_ref, expected_revision
    );

CREATE TRIGGER wizard_gaps_input_receipts_authorization_guard
BEFORE INSERT ON wizard_gaps_input_receipts
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
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_input_authorization_invalid');
END;

CREATE TRIGGER wizard_gaps_input_receipts_source_guard
BEFORE INSERT ON wizard_gaps_input_receipts
WHEN NOT EXISTS (
    SELECT 1
    FROM intake_receipts source
    WHERE source.ref = NEW.source_intake_receipt_ref
      AND source.actor_ref = NEW.actor_ref
      AND source.project_ref = NEW.project_ref
      AND source.state_ref = NEW.state_ref
      AND source.revision = NEW.expected_revision
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_input_source_invalid');
END;

CREATE TRIGGER wizard_gaps_input_receipts_outcome_guard
BEFORE INSERT ON wizard_gaps_input_receipts
WHEN (
    NEW.outcome_kind = 'intake_mutation'
    AND NOT EXISTS (
        SELECT 1
        FROM intake_receipts outcome
        WHERE outcome.ref = NEW.outcome_receipt_ref
          AND outcome.request_ref = NEW.request_ref
          AND outcome.actor_ref = NEW.actor_ref
          AND outcome.project_ref = NEW.project_ref
          AND outcome.state_ref = NEW.state_ref
          AND outcome.previous_revision = NEW.expected_revision
          AND outcome.revision = NEW.expected_revision + 1
          AND outcome.authorization_receipt_ref =
              NEW.authorization_receipt_ref
    )
) OR (
    NEW.outcome_kind = 'wizard_gaps_noop'
    AND NOT EXISTS (
        SELECT 1
        FROM wizard_gaps_outcomes outcome
        WHERE outcome.ref = NEW.outcome_receipt_ref
          AND outcome.request_ref = NEW.request_ref
          AND outcome.actor_ref = NEW.actor_ref
          AND outcome.project_ref = NEW.project_ref
          AND outcome.state_ref = NEW.state_ref
          AND outcome.expected_revision = NEW.expected_revision
          AND outcome.source_intake_receipt_ref =
              NEW.source_intake_receipt_ref
          AND outcome.authorization_receipt_ref =
              NEW.authorization_receipt_ref
    )
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_input_outcome_invalid');
END;

CREATE TRIGGER wizard_gaps_input_receipts_immutable_update
BEFORE UPDATE ON wizard_gaps_input_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_input_immutable');
END;

CREATE TRIGGER wizard_gaps_input_receipts_immutable_delete
BEFORE DELETE ON wizard_gaps_input_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.wizard_gaps_input_immutable');
END;
