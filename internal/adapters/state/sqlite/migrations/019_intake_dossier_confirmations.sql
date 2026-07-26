-- V23: one immutable confirmation freezes the exact intake/dossier and
-- publishes its Goal, DAG and outbox in the same StateRepository transaction.
CREATE TABLE intake_dossier_confirmations (
    ref TEXT PRIMARY KEY CHECK (
        length(ref) = 100
        AND substr(ref, 1, 36) = 'intake-dossier-confirmation-receipt:'
        AND substr(ref, 37) NOT GLOB '*[^0-9a-f]*'
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
    principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
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
    dossier_ref TEXT NOT NULL REFERENCES intake_dossiers(ref) ON DELETE RESTRICT,
    dossier_digest TEXT NOT NULL CHECK (
        length(dossier_digest) = 64
        AND dossier_digest NOT GLOB '*[^0-9a-f]*'
    ),
    plan_digest TEXT NOT NULL CHECK (
        length(plan_digest) = 64
        AND plan_digest NOT GLOB '*[^0-9a-f]*'
    ),
    goal_ref TEXT NOT NULL UNIQUE REFERENCES goals(ref) ON DELETE RESTRICT,
    app_spec_ref TEXT NOT NULL UNIQUE REFERENCES app_specs(ref) ON DELETE RESTRICT,
    spec_hash TEXT NOT NULL CHECK (
        length(spec_hash) = 64
        AND spec_hash NOT GLOB '*[^0-9a-f]*'
    ),
    authorization_receipt_ref TEXT NOT NULL
        REFERENCES authorization_receipts(ref) ON DELETE RESTRICT,
    confirmed_at INTEGER NOT NULL CHECK (confirmed_at > 0),
    UNIQUE(principal_ref, project_ref, request_ref),
    UNIQUE(actor_ref, project_ref, state_ref),
    UNIQUE(dossier_ref)
) STRICT;

CREATE INDEX intake_dossier_confirmations_scope_idx
    ON intake_dossier_confirmations(actor_ref, project_ref, dossier_ref, goal_ref);

CREATE TRIGGER intake_dossier_confirmations_authorization_guard
BEFORE INSERT ON intake_dossier_confirmations
WHEN NOT EXISTS (
    SELECT 1
    FROM authorization_receipts authorization
    JOIN principals principal ON principal.ref = authorization.principal_ref
    WHERE authorization.ref = NEW.authorization_receipt_ref
      AND authorization.request_ref =
          'authorization-request:intake-dossier-confirm:' || NEW.request_ref
      AND authorization.principal_ref = NEW.principal_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.create'
      AND authorization.resource_ref = NEW.project_ref
      AND authorization.outcome = 'allowed'
      AND principal.actor_ref = NEW.actor_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_confirmation_authorization_invalid');
END;

CREATE TRIGGER intake_dossier_confirmations_binding_guard
BEFORE INSERT ON intake_dossier_confirmations
WHEN NOT EXISTS (
    SELECT 1
    FROM intake_dossiers dossier
    JOIN intake_states state
      ON state.state_ref = dossier.state_ref
     AND state.actor_ref = dossier.actor_ref
     AND state.project_ref = dossier.project_ref
    JOIN goals goal ON goal.ref = NEW.goal_ref
    JOIN app_specs spec ON spec.ref = goal.app_spec_ref
    JOIN intents intent ON intent.ref = spec.intent_ref
    WHERE dossier.ref = NEW.dossier_ref
      AND dossier.actor_ref = NEW.actor_ref
      AND dossier.project_ref = NEW.project_ref
      AND dossier.state_ref = NEW.state_ref
      AND dossier.state_revision = NEW.state_revision
      AND dossier.state_digest = NEW.state_digest
      AND dossier.source_intake_receipt_ref = NEW.source_intake_receipt_ref
      AND dossier.dossier_digest = NEW.dossier_digest
      AND dossier.plan_digest = NEW.plan_digest
      AND state.revision = NEW.state_revision
      AND state.state_digest = NEW.state_digest
      AND state.receipt_ref = NEW.source_intake_receipt_ref
      AND goal.request_ref = NEW.request_ref
      AND goal.request_fingerprint = NEW.request_fingerprint
      AND goal.requested_by_ref = NEW.principal_ref
      AND goal.actor_ref = NEW.actor_ref
      AND goal.project_ref = NEW.project_ref
      AND goal.app_spec_ref = NEW.app_spec_ref
      AND goal.plan_generation = 1
      AND spec.generation = 1
      AND spec.parent_ref IS NULL
      AND spec.parent_hash IS NULL
      AND spec.hash = NEW.spec_hash
      AND spec.confirmed_by = NEW.actor_ref
      AND spec.confirmed_at = NEW.confirmed_at
      AND spec.reason =
          'operator.intake_dossier_confirmation:' || NEW.dossier_ref
      AND intent.actor_ref = NEW.actor_ref
      AND intent.project_ref = NEW.project_ref
      AND intent.statement = json_extract(dossier.snapshot_json, '$.statement')
      AND spec.objective = json_extract(dossier.snapshot_json, '$.objective')
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_confirmation_binding_invalid');
END;

CREATE TRIGGER intake_dossier_confirmations_immutable_update
BEFORE UPDATE ON intake_dossier_confirmations
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_confirmation_immutable');
END;

CREATE TRIGGER intake_dossier_confirmations_immutable_delete
BEFORE DELETE ON intake_dossier_confirmations
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_confirmation_immutable');
END;

CREATE TRIGGER intake_receipts_frozen_after_dossier_confirmation
BEFORE INSERT ON intake_receipts
WHEN NEW.operation = 'apply' AND EXISTS (
    SELECT 1
    FROM intake_dossier_confirmations confirmation
    WHERE confirmation.actor_ref = NEW.actor_ref
      AND confirmation.project_ref = NEW.project_ref
      AND confirmation.state_ref = NEW.state_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_confirmed');
END;

CREATE TRIGGER intake_dossier_generation_frozen_after_confirmation
BEFORE INSERT ON intake_dossier_generation_receipts
WHEN EXISTS (
    SELECT 1
    FROM intake_dossier_confirmations confirmation
    WHERE confirmation.actor_ref = NEW.actor_ref
      AND confirmation.project_ref = NEW.project_ref
      AND confirmation.state_ref = NEW.state_ref
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.intake_dossier_confirmed');
END;
