ALTER TABLE work_items
    ADD COLUMN handoff_required INTEGER NOT NULL DEFAULT 0
    CHECK (handoff_required IN (0, 1));

CREATE TRIGGER work_items_handoff_required_immutable
BEFORE UPDATE OF handoff_required ON work_items
WHEN NEW.handoff_required <> OLD.handoff_required
BEGIN
    SELECT RAISE(ABORT, 'sqlite.work_item_handoff_required_immutable');
END;

CREATE UNIQUE INDEX work_items_mailbox_lineage_idx
    ON work_items(goal_ref, ref, parent_ref);

CREATE UNIQUE INDEX authorization_receipts_mailbox_scope_idx
    ON authorization_receipts(ref, principal_ref, project_ref);

CREATE TABLE mailbox_envelopes (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    project_ref TEXT NOT NULL REFERENCES projects(ref) ON DELETE RESTRICT,
    goal_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    kind TEXT NOT NULL CHECK (kind = 'child_delivery'),
    summary TEXT NOT NULL CHECK (length(summary) > 0),
    content_hash TEXT NOT NULL CHECK (
        length(content_hash) = 64
        AND content_hash NOT GLOB '*[^0-9a-f]*'
    ),
    source_principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    child_work_item_ref TEXT NOT NULL,
    source_execution_ref TEXT NOT NULL,
    source_work_item_generation INTEGER NOT NULL CHECK (source_work_item_generation > 0),
    recipient_principal_ref TEXT NOT NULL REFERENCES principals(ref) ON DELETE RESTRICT,
    parent_work_item_ref TEXT NOT NULL,
    recipient_execution_ref TEXT NOT NULL,
    recipient_work_item_generation INTEGER NOT NULL CHECK (recipient_work_item_generation > 0),
    admitted_at INTEGER NOT NULL,
    UNIQUE(source_principal_ref, project_ref, request_ref),
    UNIQUE(goal_ref, ref),
    UNIQUE(goal_ref, ref, parent_work_item_ref, child_work_item_ref),
    UNIQUE(
        goal_ref, ref, plan_generation, parent_work_item_ref,
        recipient_execution_ref, recipient_work_item_generation
    ),
    FOREIGN KEY(goal_ref, project_ref)
        REFERENCES goals(ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, child_work_item_ref, parent_work_item_ref)
        REFERENCES work_items(goal_ref, ref, parent_ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, child_work_item_ref, source_execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, parent_work_item_ref, recipient_execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    CHECK (child_work_item_ref <> parent_work_item_ref),
    CHECK (source_execution_ref <> recipient_execution_ref)
) STRICT;

CREATE UNIQUE INDEX mailbox_one_child_delivery_per_lineage_idx
    ON mailbox_envelopes(goal_ref, parent_work_item_ref, child_work_item_ref);

CREATE TABLE mailbox_admission_receipts (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    mailbox_message_ref TEXT NOT NULL UNIQUE,
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    authorization_receipt_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    source_principal_ref TEXT NOT NULL,
    admitted_at INTEGER NOT NULL,
    UNIQUE(source_principal_ref, project_ref, request_ref),
    FOREIGN KEY(goal_ref, mailbox_message_ref)
        REFERENCES mailbox_envelopes(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(authorization_receipt_ref, source_principal_ref, project_ref)
        REFERENCES authorization_receipts(ref, principal_ref, project_ref) ON DELETE RESTRICT
) STRICT;

CREATE TABLE mailbox_artifact_refs (
    mailbox_message_ref TEXT NOT NULL,
    goal_ref TEXT NOT NULL,
    artifact_ref TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY(mailbox_message_ref, artifact_ref),
    UNIQUE(mailbox_message_ref, position),
    FOREIGN KEY(goal_ref, mailbox_message_ref)
        REFERENCES mailbox_envelopes(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, artifact_ref)
        REFERENCES artifacts(goal_ref, ref) ON DELETE RESTRICT
) STRICT;

CREATE TRIGGER mailbox_envelopes_immutable_update
BEFORE UPDATE ON mailbox_envelopes
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_envelope_immutable');
END;

CREATE TRIGGER mailbox_envelopes_immutable_delete
BEFORE DELETE ON mailbox_envelopes
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_envelope_immutable');
END;

CREATE TRIGGER mailbox_handoff_required_guard
BEFORE INSERT ON mailbox_envelopes
WHEN NOT EXISTS (
    SELECT 1 FROM work_items child
    WHERE child.goal_ref = NEW.goal_ref
      AND child.ref = NEW.child_work_item_ref
      AND child.parent_ref = NEW.parent_work_item_ref
      AND child.handoff_required = 1
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_handoff_required');
END;

CREATE TRIGGER mailbox_admission_receipt_guard
BEFORE INSERT ON mailbox_admission_receipts
WHEN NOT EXISTS (
    SELECT 1
    FROM mailbox_envelopes envelope
    JOIN authorization_receipts authorization
      ON authorization.ref = NEW.authorization_receipt_ref
    WHERE envelope.ref = NEW.mailbox_message_ref
      AND envelope.goal_ref = NEW.goal_ref
      AND envelope.project_ref = NEW.project_ref
      AND envelope.source_principal_ref = NEW.source_principal_ref
      AND envelope.request_ref = NEW.request_ref
      AND envelope.request_fingerprint = NEW.request_fingerprint
      AND envelope.admitted_at = NEW.admitted_at
      AND authorization.principal_ref = NEW.source_principal_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.direct'
      AND authorization.resource_ref = NEW.goal_ref
      AND authorization.outcome = 'allowed'
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_admission_receipt_invalid');
END;

CREATE TRIGGER mailbox_admission_receipts_immutable_update
BEFORE UPDATE ON mailbox_admission_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_admission_receipt_immutable');
END;

CREATE TRIGGER mailbox_admission_receipts_immutable_delete
BEFORE DELETE ON mailbox_admission_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_admission_receipt_immutable');
END;

CREATE TRIGGER mailbox_artifact_refs_immutable_update
BEFORE UPDATE ON mailbox_artifact_refs
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_artifact_ref_immutable');
END;

CREATE TRIGGER mailbox_artifact_refs_immutable_delete
BEFORE DELETE ON mailbox_artifact_refs
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_artifact_ref_immutable');
END;

CREATE TABLE outbox_v8 (
    ref TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent', 'deliver_mailbox')),
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE CASCADE,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    work_item_generation INTEGER NOT NULL CHECK (work_item_generation > 0),
    mailbox_message_ref TEXT,
    available_at INTEGER NOT NULL,
    claim_token TEXT,
    claimed_by TEXT,
    claimed_until INTEGER,
    delivery_attempt INTEGER NOT NULL DEFAULT 0 CHECK (delivery_attempt >= 0),
    fence INTEGER NOT NULL DEFAULT 0 CHECK (fence >= 0),
    completed_at INTEGER,
    retired_at INTEGER,
    quarantined_at INTEGER,
    last_error_code TEXT NOT NULL DEFAULT '',
    FOREIGN KEY(goal_ref, work_item_ref)
        REFERENCES work_items(goal_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE CASCADE,
    FOREIGN KEY(
        goal_ref, mailbox_message_ref, plan_generation, work_item_ref,
        execution_ref, work_item_generation
    ) REFERENCES mailbox_envelopes(
        goal_ref, ref, plan_generation, parent_work_item_ref,
        recipient_execution_ref, recipient_work_item_generation
    ) ON DELETE RESTRICT,
    CHECK (
        (kind IN ('launch_agent', 'observe_agent') AND mailbox_message_ref IS NULL)
        OR (kind = 'deliver_mailbox' AND mailbox_message_ref IS NOT NULL)
    ),
    CHECK (
        (claim_token IS NULL AND claimed_by IS NULL AND claimed_until IS NULL)
        OR
        (claim_token IS NOT NULL AND claimed_by IS NOT NULL AND claimed_until IS NOT NULL
            AND delivery_attempt > 0 AND fence > 0)
    ),
    CHECK (quarantined_at IS NULL OR completed_at IS NOT NULL)
	, CHECK (retired_at IS NULL OR kind = 'deliver_mailbox')
) STRICT;

INSERT INTO outbox_v8(
    ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, mailbox_message_ref, available_at,
    claim_token, claimed_by, claimed_until, delivery_attempt, fence,
    completed_at, retired_at, quarantined_at, last_error_code
)
SELECT ref, kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, NULL, available_at,
       claim_token, claimed_by, claimed_until, delivery_attempt, fence,
       completed_at, NULL, quarantined_at, last_error_code
FROM outbox;

CREATE TABLE action_consumption_receipts_v8 (
    action_ref TEXT PRIMARY KEY REFERENCES outbox_v8(ref) ON DELETE RESTRICT,
    kind TEXT NOT NULL CHECK (kind IN ('launch_agent', 'observe_agent', 'deliver_mailbox')),
    goal_ref TEXT NOT NULL,
    work_item_ref TEXT NOT NULL,
    execution_ref TEXT NOT NULL,
    plan_generation INTEGER NOT NULL CHECK (plan_generation > 0),
    work_item_generation INTEGER NOT NULL CHECK (work_item_generation > 0),
    mailbox_message_ref TEXT,
    fence INTEGER NOT NULL CHECK (fence > 0),
    delivery_attempt INTEGER NOT NULL CHECK (delivery_attempt > 0),
    claim_token TEXT NOT NULL UNIQUE,
    worker_ref TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('completed', 'quarantined')),
    error_code TEXT NOT NULL DEFAULT '',
    consumed_at INTEGER NOT NULL,
    FOREIGN KEY(goal_ref, work_item_ref)
        REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, work_item_ref, execution_ref)
        REFERENCES executions(goal_ref, work_item_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(
        goal_ref, mailbox_message_ref, plan_generation, work_item_ref,
        execution_ref, work_item_generation
    ) REFERENCES mailbox_envelopes(
        goal_ref, ref, plan_generation, parent_work_item_ref,
        recipient_execution_ref, recipient_work_item_generation
    ) ON DELETE RESTRICT,
    CHECK (
        (kind IN ('launch_agent', 'observe_agent') AND mailbox_message_ref IS NULL)
        OR (kind = 'deliver_mailbox' AND mailbox_message_ref IS NOT NULL)
    )
) STRICT;

INSERT INTO action_consumption_receipts_v8(
    action_ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, mailbox_message_ref, fence,
    delivery_attempt, claim_token, worker_ref, outcome, error_code, consumed_at
)
SELECT action_ref, kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, NULL, fence,
       delivery_attempt, claim_token, worker_ref, outcome, error_code, consumed_at
FROM action_consumption_receipts;

DROP TABLE action_consumption_receipts;
DROP TABLE outbox;

ALTER TABLE outbox_v8 RENAME TO outbox;
ALTER TABLE action_consumption_receipts_v8 RENAME TO action_consumption_receipts;

CREATE UNIQUE INDEX outbox_claim_token_idx
    ON outbox(claim_token) WHERE claim_token IS NOT NULL;

CREATE UNIQUE INDEX outbox_one_active_per_item_generation_idx
    ON outbox(goal_ref, work_item_ref, plan_generation, work_item_generation)
    WHERE kind IN ('launch_agent', 'observe_agent')
      AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL;

CREATE UNIQUE INDEX outbox_one_active_mailbox_idx
    ON outbox(mailbox_message_ref)
    WHERE kind = 'deliver_mailbox';

CREATE INDEX outbox_claimable_idx
    ON outbox(kind, completed_at, retired_at, quarantined_at, available_at, claimed_until, ref);

CREATE UNIQUE INDEX action_consumption_scheduler_fence_idx
    ON action_consumption_receipts(goal_ref, work_item_ref, fence)
    WHERE kind IN ('launch_agent', 'observe_agent');

CREATE UNIQUE INDEX action_consumption_mailbox_fence_idx
    ON action_consumption_receipts(mailbox_message_ref, fence)
    WHERE kind = 'deliver_mailbox';

CREATE TRIGGER outbox_identity_immutable
BEFORE UPDATE OF kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, mailbox_message_ref
ON outbox
BEGIN
    SELECT RAISE(ABORT, 'sqlite.outbox_identity_immutable');
END;

CREATE TRIGGER outbox_mailbox_recipient_guard
BEFORE UPDATE ON outbox
WHEN NEW.kind = 'deliver_mailbox'
 AND NEW.claimed_by IS NOT NULL
 AND NOT EXISTS (
    SELECT 1 FROM mailbox_envelopes envelope
    WHERE envelope.ref = NEW.mailbox_message_ref
      AND envelope.goal_ref = NEW.goal_ref
      AND envelope.plan_generation = NEW.plan_generation
      AND envelope.parent_work_item_ref = NEW.work_item_ref
      AND envelope.recipient_execution_ref = NEW.execution_ref
      AND envelope.recipient_work_item_generation = NEW.work_item_generation
      AND envelope.recipient_principal_ref = NEW.claimed_by
 )
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_recipient_mismatch');
END;

CREATE TRIGGER outbox_mailbox_recipient_insert_guard
BEFORE INSERT ON outbox
WHEN NEW.kind = 'deliver_mailbox'
 AND NEW.claimed_by IS NOT NULL
 AND NOT EXISTS (
    SELECT 1 FROM mailbox_envelopes envelope
    WHERE envelope.ref = NEW.mailbox_message_ref
      AND envelope.goal_ref = NEW.goal_ref
      AND envelope.plan_generation = NEW.plan_generation
      AND envelope.parent_work_item_ref = NEW.work_item_ref
      AND envelope.recipient_execution_ref = NEW.execution_ref
      AND envelope.recipient_work_item_generation = NEW.work_item_generation
      AND envelope.recipient_principal_ref = NEW.claimed_by
 )
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_recipient_mismatch');
END;

CREATE TRIGGER action_consumption_receipt_guard
BEFORE INSERT ON action_consumption_receipts
WHEN NOT EXISTS (
    SELECT 1
    FROM outbox action
    WHERE action.ref = NEW.action_ref
      AND action.kind = NEW.kind
      AND action.goal_ref = NEW.goal_ref
      AND action.work_item_ref = NEW.work_item_ref
      AND action.execution_ref = NEW.execution_ref
      AND action.plan_generation = NEW.plan_generation
      AND action.work_item_generation = NEW.work_item_generation
      AND action.mailbox_message_ref IS NEW.mailbox_message_ref
      AND action.fence = NEW.fence
      AND action.delivery_attempt = NEW.delivery_attempt
      AND action.claim_token = NEW.claim_token
      AND action.claimed_by = NEW.worker_ref
      AND action.last_error_code = NEW.error_code
      AND action.completed_at = NEW.consumed_at
      AND (
          (NEW.outcome = 'completed' AND action.quarantined_at IS NULL)
          OR
          (NEW.outcome = 'quarantined' AND action.quarantined_at = NEW.consumed_at)
      )
      AND (
          NEW.kind <> 'deliver_mailbox'
          OR EXISTS (
              SELECT 1 FROM mailbox_envelopes envelope
              WHERE envelope.ref = NEW.mailbox_message_ref
                AND envelope.recipient_principal_ref = NEW.worker_ref
          )
      )
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_invalid');
END;

CREATE TRIGGER action_consumption_receipts_immutable_update
BEFORE UPDATE ON action_consumption_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable');
END;

CREATE TRIGGER action_consumption_receipts_immutable_delete
BEFORE DELETE ON action_consumption_receipts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.action_consumption_receipt_immutable');
END;

CREATE TABLE mailbox_delivery_attempts (
    mailbox_message_ref TEXT NOT NULL,
    action_ref TEXT NOT NULL,
    project_ref TEXT NOT NULL,
    recipient_principal_ref TEXT NOT NULL,
    fence INTEGER NOT NULL CHECK (fence > 0),
    claim_token TEXT NOT NULL UNIQUE CHECK (length(trim(claim_token)) > 0),
    claim_request_ref TEXT NOT NULL CHECK (length(trim(claim_request_ref)) > 0),
    claim_request_fingerprint TEXT NOT NULL CHECK (length(trim(claim_request_fingerprint)) > 0),
    claim_authorization_receipt_ref TEXT NOT NULL,
    claimed_at INTEGER NOT NULL,
    lease_until INTEGER NOT NULL,
    delivery_request_ref TEXT,
    delivery_request_fingerprint TEXT,
    delivery_authorization_receipt_ref TEXT,
    delivery_ref TEXT,
    delivered_at INTEGER,
    consumption_request_ref TEXT,
    consumption_request_fingerprint TEXT,
    consumption_authorization_receipt_ref TEXT,
    consumption_ref TEXT,
    consumed_at INTEGER,
    PRIMARY KEY(mailbox_message_ref, fence),
    UNIQUE(action_ref, fence),
    UNIQUE(recipient_principal_ref, project_ref, claim_request_ref),
    FOREIGN KEY(mailbox_message_ref)
        REFERENCES mailbox_envelopes(ref) ON DELETE RESTRICT,
    FOREIGN KEY(action_ref)
        REFERENCES outbox(ref) ON DELETE RESTRICT,
    FOREIGN KEY(claim_authorization_receipt_ref, recipient_principal_ref, project_ref)
        REFERENCES authorization_receipts(ref, principal_ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(delivery_authorization_receipt_ref, recipient_principal_ref, project_ref)
        REFERENCES authorization_receipts(ref, principal_ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(consumption_authorization_receipt_ref, recipient_principal_ref, project_ref)
        REFERENCES authorization_receipts(ref, principal_ref, project_ref) ON DELETE RESTRICT,
    CHECK (lease_until > claimed_at),
    CHECK (
        (
            delivery_request_ref IS NULL
            AND delivery_request_fingerprint IS NULL
            AND delivery_authorization_receipt_ref IS NULL
            AND delivery_ref IS NULL
            AND delivered_at IS NULL
            AND consumption_request_ref IS NULL
            AND consumption_request_fingerprint IS NULL
            AND consumption_authorization_receipt_ref IS NULL
            AND consumption_ref IS NULL
            AND consumed_at IS NULL
        )
        OR
        (
            delivery_request_ref IS NOT NULL AND length(trim(delivery_request_ref)) > 0
            AND delivery_request_fingerprint IS NOT NULL
            AND length(trim(delivery_request_fingerprint)) > 0
            AND delivery_authorization_receipt_ref IS NOT NULL
            AND delivery_ref IS NOT NULL AND length(trim(delivery_ref)) > 0
            AND delivered_at IS NOT NULL AND delivered_at >= claimed_at
            AND delivered_at < lease_until
            AND consumption_request_ref IS NULL
            AND consumption_request_fingerprint IS NULL
            AND consumption_authorization_receipt_ref IS NULL
            AND consumption_ref IS NULL
            AND consumed_at IS NULL
        )
        OR
        (
            delivery_request_ref IS NOT NULL AND length(trim(delivery_request_ref)) > 0
            AND delivery_request_fingerprint IS NOT NULL
            AND length(trim(delivery_request_fingerprint)) > 0
            AND delivery_authorization_receipt_ref IS NOT NULL
            AND delivery_ref IS NOT NULL AND length(trim(delivery_ref)) > 0
            AND delivered_at IS NOT NULL AND delivered_at >= claimed_at
            AND delivered_at < lease_until
            AND consumption_request_ref IS NOT NULL
            AND length(trim(consumption_request_ref)) > 0
            AND consumption_request_fingerprint IS NOT NULL
            AND length(trim(consumption_request_fingerprint)) > 0
            AND consumption_authorization_receipt_ref IS NOT NULL
            AND consumption_ref IS NOT NULL AND length(trim(consumption_ref)) > 0
            AND consumed_at IS NOT NULL AND consumed_at >= delivered_at
            AND consumed_at < lease_until
        )
    )
) STRICT;

CREATE UNIQUE INDEX mailbox_delivery_attempt_delivery_request_idx
    ON mailbox_delivery_attempts(recipient_principal_ref, project_ref, delivery_request_ref)
    WHERE delivery_request_ref IS NOT NULL;

CREATE UNIQUE INDEX mailbox_delivery_attempt_consumption_request_idx
    ON mailbox_delivery_attempts(recipient_principal_ref, project_ref, consumption_request_ref)
    WHERE consumption_request_ref IS NOT NULL;

CREATE TRIGGER mailbox_delivery_attempt_insert_guard
BEFORE INSERT ON mailbox_delivery_attempts
WHEN NEW.delivery_request_ref IS NOT NULL
 OR NEW.delivery_request_fingerprint IS NOT NULL
 OR NEW.delivery_authorization_receipt_ref IS NOT NULL
 OR NEW.delivery_ref IS NOT NULL
 OR NEW.delivered_at IS NOT NULL
 OR NEW.consumption_request_ref IS NOT NULL
 OR NEW.consumption_request_fingerprint IS NOT NULL
 OR NEW.consumption_authorization_receipt_ref IS NOT NULL
 OR NEW.consumption_ref IS NOT NULL
 OR NEW.consumed_at IS NOT NULL
 OR NEW.fence <> COALESCE((
        SELECT MAX(attempt.fence)
        FROM mailbox_delivery_attempts attempt
        WHERE attempt.mailbox_message_ref = NEW.mailbox_message_ref
    ), 0) + 1
 OR NOT EXISTS (
    SELECT 1
    FROM outbox action
    JOIN mailbox_envelopes envelope ON envelope.ref = NEW.mailbox_message_ref
    JOIN authorization_receipts authorization
      ON authorization.ref = NEW.claim_authorization_receipt_ref
    WHERE action.mailbox_message_ref = NEW.mailbox_message_ref
      AND action.goal_ref = envelope.goal_ref
      AND action.work_item_ref = envelope.parent_work_item_ref
      AND action.execution_ref = envelope.recipient_execution_ref
      AND action.plan_generation = envelope.plan_generation
      AND action.work_item_generation = envelope.recipient_work_item_generation
      AND envelope.project_ref = NEW.project_ref
      AND envelope.recipient_principal_ref = NEW.recipient_principal_ref
      AND action.claim_token = NEW.claim_token
      AND action.claimed_by = NEW.recipient_principal_ref
      AND action.claimed_until = NEW.lease_until
      AND action.delivery_attempt = NEW.fence
      AND action.fence = NEW.fence
      AND action.completed_at IS NULL
      AND action.retired_at IS NULL
      AND action.quarantined_at IS NULL
      AND authorization.principal_ref = NEW.recipient_principal_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.get'
      AND authorization.resource_ref = NEW.mailbox_message_ref
      AND authorization.outcome = 'allowed'
 )
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_delivery_attempt_invalid');
END;

CREATE TRIGGER mailbox_delivery_attempt_progress_guard
BEFORE UPDATE ON mailbox_delivery_attempts
WHEN NOT (
    NEW.mailbox_message_ref = OLD.mailbox_message_ref
    AND NEW.action_ref = OLD.action_ref
    AND NEW.project_ref = OLD.project_ref
    AND NEW.recipient_principal_ref = OLD.recipient_principal_ref
    AND NEW.fence = OLD.fence
    AND NEW.claim_token = OLD.claim_token
    AND NEW.claim_request_ref = OLD.claim_request_ref
    AND NEW.claim_request_fingerprint = OLD.claim_request_fingerprint
    AND NEW.claim_authorization_receipt_ref = OLD.claim_authorization_receipt_ref
    AND NEW.claimed_at = OLD.claimed_at
    AND NEW.lease_until = OLD.lease_until
    AND EXISTS (
        SELECT 1 FROM outbox action
        WHERE action.ref = OLD.action_ref
          AND action.mailbox_message_ref = OLD.mailbox_message_ref
          AND action.claim_token = OLD.claim_token
          AND action.claimed_by = OLD.recipient_principal_ref
          AND action.claimed_until = OLD.lease_until
          AND action.delivery_attempt = OLD.fence
          AND action.fence = OLD.fence
          AND action.completed_at IS NULL
          AND action.retired_at IS NULL
          AND action.quarantined_at IS NULL
    )
    AND (
        (
            OLD.delivery_request_ref IS NULL
            AND OLD.delivery_request_fingerprint IS NULL
            AND OLD.delivery_authorization_receipt_ref IS NULL
            AND OLD.delivery_ref IS NULL
            AND OLD.delivered_at IS NULL
            AND OLD.consumption_request_ref IS NULL
            AND OLD.consumption_request_fingerprint IS NULL
            AND OLD.consumption_authorization_receipt_ref IS NULL
            AND OLD.consumption_ref IS NULL
            AND OLD.consumed_at IS NULL
            AND NEW.delivery_request_ref IS NOT NULL
            AND NEW.delivery_request_fingerprint IS NOT NULL
            AND NEW.delivery_authorization_receipt_ref IS NOT NULL
            AND NEW.delivery_ref IS NOT NULL
            AND NEW.delivered_at IS NOT NULL
            AND NEW.delivered_at >= OLD.claimed_at
            AND NEW.delivered_at < OLD.lease_until
            AND NEW.consumption_request_ref IS NULL
            AND NEW.consumption_request_fingerprint IS NULL
            AND NEW.consumption_authorization_receipt_ref IS NULL
            AND NEW.consumption_ref IS NULL
            AND NEW.consumed_at IS NULL
            AND EXISTS (
                SELECT 1 FROM authorization_receipts authorization
                WHERE authorization.ref = NEW.delivery_authorization_receipt_ref
                  AND authorization.principal_ref = OLD.recipient_principal_ref
                  AND authorization.project_ref = OLD.project_ref
                  AND authorization.permission = 'goals.get'
                  AND authorization.resource_ref = OLD.mailbox_message_ref
                  AND authorization.outcome = 'allowed'
            )
        )
        OR
        (
            OLD.delivery_request_ref IS NOT NULL
            AND NEW.delivery_request_ref = OLD.delivery_request_ref
            AND NEW.delivery_request_fingerprint = OLD.delivery_request_fingerprint
            AND NEW.delivery_authorization_receipt_ref = OLD.delivery_authorization_receipt_ref
            AND NEW.delivery_ref = OLD.delivery_ref
            AND OLD.delivered_at IS NOT NULL
            AND NEW.delivered_at = OLD.delivered_at
            AND OLD.consumption_request_ref IS NULL
            AND OLD.consumption_request_fingerprint IS NULL
            AND OLD.consumption_authorization_receipt_ref IS NULL
            AND OLD.consumption_ref IS NULL
            AND OLD.consumed_at IS NULL
            AND NEW.consumption_request_ref IS NOT NULL
            AND NEW.consumption_request_fingerprint IS NOT NULL
            AND NEW.consumption_authorization_receipt_ref IS NOT NULL
            AND NEW.consumption_ref IS NOT NULL
            AND NEW.consumed_at IS NOT NULL
            AND NEW.consumed_at >= OLD.delivered_at
            AND NEW.consumed_at < OLD.lease_until
            AND EXISTS (
                SELECT 1 FROM authorization_receipts authorization
                WHERE authorization.ref = NEW.consumption_authorization_receipt_ref
                  AND authorization.principal_ref = OLD.recipient_principal_ref
                  AND authorization.project_ref = OLD.project_ref
                  AND authorization.permission = 'goals.get'
                  AND authorization.resource_ref = OLD.mailbox_message_ref
                  AND authorization.outcome = 'allowed'
            )
        )
    )
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_delivery_attempt_progress_invalid');
END;

CREATE TRIGGER mailbox_delivery_attempts_immutable_delete
BEFORE DELETE ON mailbox_delivery_attempts
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_delivery_attempt_immutable');
END;

CREATE TABLE mailbox_delivery_acks (
    ref TEXT PRIMARY KEY CHECK (length(trim(ref)) > 0),
    request_ref TEXT NOT NULL CHECK (length(trim(request_ref)) > 0),
    request_fingerprint TEXT NOT NULL CHECK (length(trim(request_fingerprint)) > 0),
    authorization_receipt_ref TEXT NOT NULL,
    mailbox_message_ref TEXT NOT NULL UNIQUE,
    action_ref TEXT NOT NULL UNIQUE,
    project_ref TEXT NOT NULL,
    expected_goal_revision INTEGER NOT NULL CHECK (expected_goal_revision > 0),
    expected_plan_generation INTEGER NOT NULL CHECK (expected_plan_generation > 0),
    recipient_principal_ref TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('acknowledged', 'blocked')),
    effect_or_rework_ref TEXT NOT NULL CHECK (length(trim(effect_or_rework_ref)) > 0),
    acked_at INTEGER NOT NULL,
    UNIQUE(recipient_principal_ref, project_ref, request_ref),
    UNIQUE(mailbox_message_ref, outcome, ref, acked_at),
    FOREIGN KEY(authorization_receipt_ref, recipient_principal_ref, project_ref)
        REFERENCES authorization_receipts(ref, principal_ref, project_ref) ON DELETE RESTRICT,
    FOREIGN KEY(mailbox_message_ref)
        REFERENCES mailbox_envelopes(ref) ON DELETE RESTRICT,
    FOREIGN KEY(action_ref)
        REFERENCES outbox(ref) ON DELETE RESTRICT
) STRICT;

CREATE TRIGGER mailbox_ack_guard
BEFORE INSERT ON mailbox_delivery_acks
WHEN EXISTS (
    SELECT 1 FROM mailbox_retirements retirement
    WHERE retirement.mailbox_message_ref = NEW.mailbox_message_ref
)
OR NOT EXISTS (
    SELECT 1
    FROM mailbox_envelopes envelope
    JOIN mailbox_delivery_attempts attempt
      ON attempt.mailbox_message_ref = envelope.ref
    JOIN action_consumption_receipts receipt
      ON receipt.action_ref = NEW.action_ref
    JOIN outbox action
      ON action.ref = NEW.action_ref
    JOIN authorization_receipts authorization
      ON authorization.ref = NEW.authorization_receipt_ref
    WHERE envelope.ref = NEW.mailbox_message_ref
      AND envelope.project_ref = NEW.project_ref
      AND envelope.recipient_principal_ref = NEW.recipient_principal_ref
      AND NEW.expected_plan_generation >= envelope.plan_generation
      AND attempt.action_ref = NEW.action_ref
      AND attempt.recipient_principal_ref = NEW.recipient_principal_ref
      AND attempt.consumed_at IS NOT NULL
      AND receipt.kind = 'deliver_mailbox'
      AND receipt.mailbox_message_ref = NEW.mailbox_message_ref
      AND receipt.fence = attempt.fence
      AND receipt.delivery_attempt = attempt.fence
      AND receipt.claim_token = attempt.claim_token
      AND receipt.worker_ref = NEW.recipient_principal_ref
      AND receipt.outcome = 'completed'
      AND receipt.consumed_at = attempt.consumed_at
      AND NEW.acked_at >= receipt.consumed_at
      AND action.completed_at = receipt.consumed_at
      AND action.retired_at IS NULL
      AND action.quarantined_at IS NULL
      AND authorization.principal_ref = NEW.recipient_principal_ref
      AND authorization.project_ref = NEW.project_ref
      AND authorization.permission = 'goals.get'
      AND authorization.resource_ref = NEW.mailbox_message_ref
      AND authorization.outcome = 'allowed'
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_ack_invalid');
END;

CREATE TRIGGER mailbox_delivery_acks_immutable_update
BEFORE UPDATE ON mailbox_delivery_acks
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_ack_immutable');
END;

CREATE TRIGGER mailbox_delivery_acks_immutable_delete
BEFORE DELETE ON mailbox_delivery_acks
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_ack_immutable');
END;

CREATE TABLE mailbox_retirements (
    mailbox_message_ref TEXT PRIMARY KEY
        REFERENCES mailbox_envelopes(ref) ON DELETE RESTRICT,
    action_ref TEXT NOT NULL UNIQUE REFERENCES outbox(ref) ON DELETE RESTRICT,
    recipient_execution_ref TEXT NOT NULL REFERENCES executions(ref) ON DELETE RESTRICT,
    failure_code TEXT NOT NULL CHECK (length(trim(failure_code)) > 0),
    retired_at INTEGER NOT NULL
) STRICT;

CREATE TRIGGER mailbox_retirement_guard
BEFORE INSERT ON mailbox_retirements
WHEN EXISTS (
    SELECT 1 FROM mailbox_delivery_acks ack
    WHERE ack.mailbox_message_ref = NEW.mailbox_message_ref
)
OR NOT EXISTS (
    SELECT 1
    FROM mailbox_envelopes envelope
    JOIN outbox action ON action.ref = NEW.action_ref
    JOIN executions execution ON execution.ref = NEW.recipient_execution_ref
    WHERE envelope.ref = NEW.mailbox_message_ref
      AND envelope.recipient_execution_ref = NEW.recipient_execution_ref
      AND action.kind = 'deliver_mailbox'
      AND action.mailbox_message_ref = NEW.mailbox_message_ref
      AND action.execution_ref = NEW.recipient_execution_ref
      AND action.retired_at IS NULL
      AND execution.goal_ref = envelope.goal_ref
      AND execution.work_item_ref = envelope.parent_work_item_ref
      AND execution.state = 'failed'
      AND execution.failure_code = NEW.failure_code
      AND execution.finished_at = NEW.retired_at
)
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_retirement_invalid');
END;

CREATE TRIGGER outbox_mailbox_retirement_guard
BEFORE UPDATE OF retired_at ON outbox
WHEN NEW.retired_at IS NOT OLD.retired_at
 AND (
    OLD.retired_at IS NOT NULL
    OR NEW.retired_at IS NULL
    OR NOT EXISTS (
        SELECT 1 FROM mailbox_retirements retirement
        WHERE retirement.mailbox_message_ref = NEW.mailbox_message_ref
          AND retirement.action_ref = NEW.ref
          AND retirement.recipient_execution_ref = NEW.execution_ref
          AND retirement.retired_at = NEW.retired_at
    )
 )
BEGIN
    SELECT RAISE(ABORT, 'sqlite.outbox_mailbox_retirement_invalid');
END;

CREATE TRIGGER mailbox_retirements_immutable_update
BEFORE UPDATE ON mailbox_retirements
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_retirement_immutable');
END;

CREATE TRIGGER mailbox_retirements_immutable_delete
BEFORE DELETE ON mailbox_retirements
BEGIN
    SELECT RAISE(ABORT, 'sqlite.mailbox_retirement_immutable');
END;

CREATE TABLE goal_child_handoff_resolutions (
    goal_ref TEXT NOT NULL REFERENCES goals(ref) ON DELETE RESTRICT,
    parent_work_item_ref TEXT NOT NULL,
    child_work_item_ref TEXT NOT NULL,
    mailbox_message_ref TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('acknowledged', 'blocked')),
    receipt_ref TEXT NOT NULL UNIQUE CHECK (length(trim(receipt_ref)) > 0),
    resolved_at INTEGER NOT NULL,
    PRIMARY KEY(goal_ref, parent_work_item_ref, child_work_item_ref),
    UNIQUE(goal_ref, mailbox_message_ref),
    FOREIGN KEY(goal_ref, parent_work_item_ref)
        REFERENCES work_items(goal_ref, ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, child_work_item_ref, parent_work_item_ref)
        REFERENCES work_items(goal_ref, ref, parent_ref) ON DELETE RESTRICT,
    FOREIGN KEY(goal_ref, mailbox_message_ref, parent_work_item_ref, child_work_item_ref)
        REFERENCES mailbox_envelopes(
            goal_ref, ref, parent_work_item_ref, child_work_item_ref
        ) ON DELETE RESTRICT,
    FOREIGN KEY(mailbox_message_ref, outcome, receipt_ref, resolved_at)
        REFERENCES mailbox_delivery_acks(
            mailbox_message_ref, outcome, ref, acked_at
        ) ON DELETE RESTRICT
) STRICT;

CREATE TRIGGER goal_child_handoff_resolutions_immutable_update
BEFORE UPDATE ON goal_child_handoff_resolutions
BEGIN
    SELECT RAISE(ABORT, 'sqlite.goal_child_handoff_resolution_immutable');
END;

CREATE TRIGGER goal_child_handoff_resolutions_immutable_delete
BEFORE DELETE ON goal_child_handoff_resolutions
BEGIN
    SELECT RAISE(ABORT, 'sqlite.goal_child_handoff_resolution_immutable');
END;
