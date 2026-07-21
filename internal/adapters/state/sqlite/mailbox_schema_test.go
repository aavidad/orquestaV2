package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestMailboxSchemaV13MigratesV7PreservesOutboxAndEnforcesSingleQueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v7", "orquesta.sqlite")
	seedV09DatabaseForV10(t, path)
	database := openRawV10TestDatabase(t, path)
	mailboxUpgradeV7(t, database)

	mustV10Exec(t, database, `UPDATE work_item_fences SET fence = 1`)
	mustV10Exec(t, database, `
UPDATE outbox
SET claim_token = 'claim:v7-preserved', claimed_by = 'worker:v7-preserved',
    claimed_until = available_at + 100, delivery_attempt = 1, fence = 1,
    completed_at = available_at + 10
WHERE ref = 'action:v10-v09'`)
	mustV10Exec(t, database, `
INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
)
SELECT ref, kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, fence, delivery_attempt,
       claim_token, claimed_by, 'completed', last_error_code, completed_at
FROM outbox WHERE ref = 'action:v10-v09'`)

	legacyOutbox := mailboxRows(t, database, `
SELECT ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
       work_item_generation, available_at, claim_token, claimed_by,
       claimed_until, delivery_attempt, fence, completed_at,
       quarantined_at, last_error_code
FROM outbox ORDER BY ref`)
	legacyReceipts := mailboxRows(t, database, `
SELECT action_ref, kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, fence, delivery_attempt,
       claim_token, worker_ref, outcome, error_code, consumed_at
FROM action_consumption_receipts ORDER BY action_ref`)
	if len(legacyOutbox) == 0 || len(legacyReceipts) == 0 {
		t.Fatal("V7 preservation seed lacks outbox or consumption receipt rows")
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("migrate V7 mailbox schema: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	if after := mailboxRows(t, repository.db, `
SELECT ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
       work_item_generation, available_at, claim_token, claimed_by,
       claimed_until, delivery_attempt, fence, completed_at,
       quarantined_at, last_error_code
FROM outbox ORDER BY ref`); !reflect.DeepEqual(after, legacyOutbox) {
		t.Fatalf("V7 outbox rows changed: before=%v after=%v", legacyOutbox, after)
	}
	if after := mailboxRows(t, repository.db, `
SELECT action_ref, kind, goal_ref, work_item_ref, execution_ref,
       plan_generation, work_item_generation, fence, delivery_attempt,
       claim_token, worker_ref, outcome, error_code, consumed_at
FROM action_consumption_receipts ORDER BY action_ref`); !reflect.DeepEqual(after, legacyReceipts) {
		t.Fatalf("V7 consumption receipts changed: before=%v after=%v", legacyReceipts, after)
	}

	var version int
	var migrationName string
	if err := repository.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT name FROM schema_migrations WHERE version = 8`).Scan(&migrationName); err != nil {
		t.Fatal(err)
	}
	if version != recoverySchemaV16 || migrationName != "008_mailbox.sql" {
		t.Fatalf("mailbox migration identity: version=%d name=%q", version, migrationName)
	}

	for _, table := range []string{
		"mailbox_envelopes", "mailbox_admission_receipts", "mailbox_artifact_refs",
		"mailbox_delivery_attempts", "mailbox_delivery_acks",
		"mailbox_retirements",
		"goal_child_handoff_resolutions",
	} {
		mailboxRequireSchemaObject(t, repository.db, "table", table)
	}
	for _, index := range []string{
		"outbox_one_active_per_item_generation_idx", "outbox_one_active_mailbox_idx",
		"action_consumption_scheduler_fence_idx", "action_consumption_mailbox_fence_idx",
	} {
		mailboxRequireSchemaObject(t, repository.db, "index", index)
	}
	for _, trigger := range []string{
		"work_items_handoff_required_immutable",
		"mailbox_envelopes_immutable_update", "mailbox_admission_receipt_guard",
		"mailbox_handoff_required_guard", "mailbox_delivery_attempt_insert_guard",
		"mailbox_delivery_attempt_progress_guard", "mailbox_ack_guard",
		"mailbox_delivery_acks_immutable_update",
		"mailbox_retirement_guard", "outbox_mailbox_retirement_guard",
		"mailbox_retirements_immutable_update",
		"goal_child_handoff_resolutions_immutable_update",
	} {
		mailboxRequireSchemaObject(t, repository.db, "trigger", trigger)
	}
	var handoffColumns, requiredEdges int
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('work_items') WHERE name = 'handoff_required'`,
	).Scan(&handoffColumns); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM work_items WHERE handoff_required <> 0`).Scan(&requiredEdges); err != nil {
		t.Fatal(err)
	}
	if handoffColumns != 1 || requiredEdges != 0 {
		t.Fatalf("legacy handoff migration columns=%d required=%d", handoffColumns, requiredEdges)
	}
	var legacyGoalRef string
	if err := repository.db.QueryRow(`SELECT ref FROM goals ORDER BY ref LIMIT 1`).Scan(&legacyGoalRef); err != nil {
		t.Fatal(err)
	}
	legacyItems, err := readWorkItems(context.Background(), repository.db, legacyGoalRef, nil, nil)
	if err != nil || len(legacyItems) == 0 {
		t.Fatalf("read migrated legacy handoff policy items=%d err=%v", len(legacyItems), err)
	}
	for _, item := range legacyItems {
		if item.HandoffRequired == nil || *item.HandoffRequired {
			t.Fatalf("legacy edge did not migrate explicitly false: %+v", item)
		}
	}
	mailboxRequireSQLError(
		t, repository.db,
		`UPDATE work_items SET handoff_required = 1 WHERE goal_ref = ?`, legacyGoalRef,
	)

	for _, table := range []string{"outbox", "action_consumption_receipts"} {
		var columns int
		if err := repository.db.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'mailbox_message_ref'`, table,
		).Scan(&columns); err != nil {
			t.Fatal(err)
		}
		if columns != 1 {
			t.Fatalf("%s mailbox_message_ref columns=%d", table, columns)
		}
	}

	var queueTables []string
	rows, err := repository.db.Query(`
SELECT name FROM sqlite_schema
WHERE type = 'table' AND (lower(name) LIKE '%queue%' OR lower(name) LIKE '%outbox%')
ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		queueTables = append(queueTables, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(queueTables, []string{"outbox"}) {
		t.Fatalf("mailbox introduced another queue authority: %v", queueTables)
	}

	var schedulerIndexSQL, mailboxIndexSQL, envelopeSQL string
	if err := repository.db.QueryRow(`SELECT lower(sql) FROM sqlite_schema WHERE name = 'outbox_one_active_per_item_generation_idx'`).Scan(&schedulerIndexSQL); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT lower(sql) FROM sqlite_schema WHERE name = 'outbox_one_active_mailbox_idx'`).Scan(&mailboxIndexSQL); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT lower(sql) FROM sqlite_schema WHERE name = 'mailbox_envelopes'`).Scan(&envelopeSQL); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"launch_agent", "observe_agent"} {
		if !strings.Contains(schedulerIndexSQL, token) {
			t.Fatalf("scheduler uniqueness lacks %q: %s", token, schedulerIndexSQL)
		}
	}
	if strings.Contains(schedulerIndexSQL, "deliver_mailbox") ||
		!strings.Contains(mailboxIndexSQL, "mailbox_message_ref") {
		t.Fatalf("outbox uniqueness scopes invalid: scheduler=%s mailbox=%s", schedulerIndexSQL, mailboxIndexSQL)
	}
	for _, token := range []string{"'child_delivery'", "summary", "content_hash"} {
		if !strings.Contains(envelopeSQL, token) {
			t.Fatalf("mailbox envelope schema lacks %q: %s", token, envelopeSQL)
		}
	}
	for _, token := range []string{"'message'", "'handoff'"} {
		if strings.Contains(envelopeSQL, token) {
			t.Fatalf("mailbox envelope schema retains deferred kind %q: %s", token, envelopeSQL)
		}
	}

	var foreignKeyViolations int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&foreignKeyViolations); err != nil {
		t.Fatal(err)
	}
	if foreignKeyViolations != 0 {
		t.Fatalf("mailbox migration foreign-key violations=%d", foreignKeyViolations)
	}
}

func TestMailboxSchemaV13EnforcesExactRecipientProgressAndImmutableCausalReceipts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mailbox-contract", "orquesta.sqlite")
	seedV09DatabaseForV10(t, path)
	database := openRawV10TestDatabase(t, path)
	mailboxUpgradeV7(t, database)
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	database = repository.db
	base := mailboxUnix(time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC))

	var goalRef, projectRef, parentRef, parentExecutionRef, phaseKey string
	var planGeneration, parentGeneration int64
	if err := database.QueryRow(`
SELECT goal.ref, goal.project_ref, goal.plan_generation,
       item.ref, item.revision, item.phase_key, execution.ref
FROM goals goal
JOIN work_items item ON item.goal_ref = goal.ref
JOIN executions execution
  ON execution.goal_ref = item.goal_ref AND execution.work_item_ref = item.ref
WHERE goal.ref = 'goal:v10-v09'
ORDER BY item.position, execution.attempt_no
LIMIT 1`).Scan(
		&goalRef, &projectRef, &planGeneration,
		&parentRef, &parentGeneration, &phaseKey, &parentExecutionRef,
	); err != nil {
		t.Fatal(err)
	}

	mustV10Exec(t, database, `INSERT INTO workspaces(ref) VALUES ('workspace:mailbox-schema')`)
	mustV10Exec(t, database, `INSERT INTO groups(ref, workspace_ref)
VALUES ('group:mailbox-schema', 'workspace:mailbox-schema')`)
	mustV10Exec(t, database, `INSERT INTO projects(ref, group_ref) VALUES (?, 'group:mailbox-schema')`, projectRef)
	mustV10Exec(t, database, `INSERT INTO principals(ref, actor_ref, kind, authentication_method) VALUES
('principal:mailbox-source', 'actor:mailbox-source', 'service', 'test'),
('principal:mailbox-recipient', 'actor:mailbox-recipient', 'service', 'test'),
('principal:mailbox-wrong', 'actor:mailbox-wrong', 'service', 'test')`)

	childRef := "work-item:mailbox-child"
	childExecutionRef := "execution:mailbox-child"
	mustV10Exec(t, database, `
INSERT INTO work_items(
    ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
    parent_ref, output_contract, skip_reason, state, revision, position,
    created_at, started_at, finished_at, execution_ref, handoff_required
)
SELECT ?, goal_ref, 'actor:mailbox-source', project_ref, 'mailbox child', ?,
       'mailbox.child', ref, 'artifact', '', 'succeeded', 1, 1000,
       ?, ?, ?, ?, 1
FROM work_items WHERE goal_ref = ? AND ref = ?`,
		childRef, phaseKey, base, base, base, childExecutionRef, goalRef, parentRef)
	mustV10Exec(t, database, `
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code
)
SELECT ?, goal_ref, ?, 1, max_execution_attempts, NULL, plan_generation,
       app_spec_generation, spec_hash, 'succeeded', artifact_media_type,
       'idempotency:mailbox-child', max_output_bytes, '', '', '', '', ?, NULL,
       ?, NULL, NULL, NULL, ?, ''
FROM executions WHERE goal_ref = ? AND work_item_ref = ? AND ref = ?`,
		childExecutionRef, childRef, base, base, base, goalRef, parentRef, parentExecutionRef)
	mustV10Exec(t, database, `INSERT INTO artifacts(
    ref, goal_ref, work_item_ref, digest, media_type, size, created_at
) VALUES ('artifact:mailbox-child', ?, ?, 'sha256:mailbox-child', 'text/plain', 7, ?)`,
		goalRef, childRef, base)

	mailboxInsertAuthorization(t, database, "auth:mailbox-admit", "principal:mailbox-source",
		projectRef, "goals.direct", goalRef, base)
	for _, operation := range []string{"claim", "delivery", "consumption", "ack"} {
		mailboxInsertAuthorization(t, database, "auth:mailbox-"+operation,
			"principal:mailbox-recipient", projectRef, "goals.get", "message:mailbox-one", base)
	}

	mailboxInsertEnvelope(t, database, mailboxEnvelopeSeed{
		Ref: "message:mailbox-one", RequestRef: "request:mailbox-admit-one",
		Fingerprint: "fingerprint:mailbox-admit-one", ProjectRef: projectRef,
		GoalRef: goalRef, PlanGeneration: planGeneration, Kind: "child_delivery",
		Hash: strings.Repeat("a", 64), SourcePrincipalRef: "principal:mailbox-source",
		ChildWorkItemRef: childRef, SourceExecutionRef: childExecutionRef,
		RecipientPrincipalRef: "principal:mailbox-recipient", ParentWorkItemRef: parentRef,
		RecipientExecutionRef: parentExecutionRef, RecipientGeneration: parentGeneration,
		AdmittedAt: base,
	})
	mailboxRequireSQLError(t, database, `INSERT INTO mailbox_envelopes(
    ref, request_ref, request_fingerprint, project_ref, goal_ref, plan_generation,
    kind, summary, content_hash, source_principal_ref, child_work_item_ref,
    source_execution_ref, source_work_item_generation, recipient_principal_ref,
    parent_work_item_ref, recipient_execution_ref, recipient_work_item_generation,
    admitted_at
)
SELECT 'message:mailbox-invalid-hash', 'request:mailbox-invalid-hash',
       'fingerprint:mailbox-invalid-hash', project_ref, goal_ref, plan_generation,
       kind, summary, upper(content_hash), source_principal_ref, child_work_item_ref,
       source_execution_ref, source_work_item_generation, recipient_principal_ref,
       parent_work_item_ref, recipient_execution_ref, recipient_work_item_generation,
       admitted_at
FROM mailbox_envelopes WHERE ref = 'message:mailbox-one'`)
	mustV10Exec(t, database, `INSERT INTO mailbox_admission_receipts(
    ref, mailbox_message_ref, request_ref, request_fingerprint,
    authorization_receipt_ref, project_ref, goal_ref, source_principal_ref, admitted_at
) VALUES (
    'receipt:mailbox-admit-one', 'message:mailbox-one', 'request:mailbox-admit-one',
    'fingerprint:mailbox-admit-one', 'auth:mailbox-admit', ?, ?,
    'principal:mailbox-source', ?
)`, projectRef, goalRef, base)
	mustV10Exec(t, database, `INSERT INTO mailbox_artifact_refs(
    mailbox_message_ref, goal_ref, artifact_ref, position
) VALUES ('message:mailbox-one', ?, 'artifact:mailbox-child', 0)`, goalRef)
	mailboxInsertDeliveryAction(t, database, "action:mailbox-one", "message:mailbox-one",
		goalRef, parentRef, parentExecutionRef, planGeneration, parentGeneration, base)

	mailboxRequireSQLError(t, database, `INSERT INTO mailbox_envelopes(
    ref, request_ref, request_fingerprint, project_ref, goal_ref, plan_generation,
    kind, summary, content_hash, source_principal_ref, child_work_item_ref,
    source_execution_ref, source_work_item_generation, recipient_principal_ref,
    parent_work_item_ref, recipient_execution_ref, recipient_work_item_generation,
    admitted_at
)
SELECT 'message:mailbox-deferred-kind', 'request:mailbox-deferred-kind',
       'fingerprint:mailbox-deferred-kind', project_ref, goal_ref, plan_generation,
       'message', summary, ?, source_principal_ref, child_work_item_ref,
       source_execution_ref, source_work_item_generation, recipient_principal_ref,
       parent_work_item_ref, recipient_execution_ref, recipient_work_item_generation,
       admitted_at + 1
FROM mailbox_envelopes WHERE ref = 'message:mailbox-one'`, strings.Repeat("b", 64))
	mailboxRequireSQLError(t, database, `INSERT INTO mailbox_envelopes(
    ref, request_ref, request_fingerprint, project_ref, goal_ref, plan_generation,
    kind, summary, content_hash, source_principal_ref, child_work_item_ref,
    source_execution_ref, source_work_item_generation, recipient_principal_ref,
    parent_work_item_ref, recipient_execution_ref, recipient_work_item_generation,
    admitted_at
)
SELECT 'message:mailbox-duplicate-lineage', 'request:mailbox-duplicate-lineage',
       'fingerprint:mailbox-duplicate-lineage', project_ref, goal_ref, plan_generation,
       kind, summary, ?, source_principal_ref, child_work_item_ref,
       source_execution_ref, source_work_item_generation, recipient_principal_ref,
       parent_work_item_ref, recipient_execution_ref, recipient_work_item_generation,
       admitted_at + 1
FROM mailbox_envelopes WHERE ref = 'message:mailbox-one'`, strings.Repeat("b", 64))

	var activeMailboxActions int
	if err := database.QueryRow(`SELECT COUNT(*) FROM outbox
WHERE kind = 'deliver_mailbox' AND work_item_ref = ? AND completed_at IS NULL`, parentRef,
	).Scan(&activeMailboxActions); err != nil {
		t.Fatal(err)
	}
	if activeMailboxActions != 1 {
		t.Fatalf("active mailbox actions to one child lineage=%d, want 1", activeMailboxActions)
	}
	mailboxRequireSQLError(t, database, `INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, mailbox_message_ref, available_at
) VALUES ('action:mailbox-one-duplicate', 'deliver_mailbox', ?, ?, ?, ?, ?,
          'message:mailbox-one', ?)`,
		goalRef, parentRef, parentExecutionRef, planGeneration, parentGeneration, base)
	mailboxRequireSQLError(t, database, `INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, mailbox_message_ref, available_at
) VALUES ('action:mailbox-wrong-execution', 'deliver_mailbox', ?, ?, ?, ?, 1,
          'message:mailbox-one', ?)`,
		goalRef, childRef, childExecutionRef, planGeneration, base)
	mailboxRequireSQLError(t, database, `UPDATE mailbox_envelopes
SET summary = 'changed' WHERE ref = 'message:mailbox-one'`)
	mailboxRequireSQLError(t, database, `UPDATE mailbox_admission_receipts
SET admitted_at = admitted_at + 1 WHERE mailbox_message_ref = 'message:mailbox-one'`)
	leaseUntil := base + int64(time.Minute)
	mailboxRequireSQLError(t, database, `UPDATE outbox SET
    claim_token = 'claim:mailbox-one', claimed_by = 'principal:mailbox-wrong',
    claimed_until = ?, delivery_attempt = 1, fence = 1
WHERE ref = 'action:mailbox-one'`, leaseUntil)
	mustV10Exec(t, database, `UPDATE outbox SET
    claim_token = 'claim:mailbox-one', claimed_by = 'principal:mailbox-recipient',
    claimed_until = ?, delivery_attempt = 1, fence = 1
WHERE ref = 'action:mailbox-one'`, leaseUntil)
	mustV10Exec(t, database, `INSERT INTO mailbox_delivery_attempts(
	    mailbox_message_ref, action_ref, project_ref, recipient_principal_ref,
	    fence, claim_token,
	    claim_request_ref, claim_request_fingerprint, claim_authorization_receipt_ref,
	    claimed_at, lease_until
	) VALUES (
	    'message:mailbox-one', 'action:mailbox-one', ?,
	    'principal:mailbox-recipient', 1, 'claim:mailbox-one',
	    'request:mailbox-claim', 'fingerprint:mailbox-claim', 'auth:mailbox-claim', ?, ?
	)`, projectRef, base+2, leaseUntil)

	mailboxRequireSQLError(t, database, `UPDATE mailbox_delivery_attempts SET
    consumption_request_ref = 'request:mailbox-consume',
    consumption_request_fingerprint = 'fingerprint:mailbox-consume',
    consumption_authorization_receipt_ref = 'auth:mailbox-consumption',
    consumption_ref = 'receipt:mailbox-consumption', consumed_at = ?
WHERE mailbox_message_ref = 'message:mailbox-one' AND fence = 1`, base+4)
	mustV10Exec(t, database, `UPDATE mailbox_delivery_attempts SET
    delivery_request_ref = 'request:mailbox-delivery',
    delivery_request_fingerprint = 'fingerprint:mailbox-delivery',
    delivery_authorization_receipt_ref = 'auth:mailbox-delivery',
    delivery_ref = 'receipt:mailbox-delivery', delivered_at = ?
WHERE mailbox_message_ref = 'message:mailbox-one' AND fence = 1`, base+3)
	mailboxRequireSQLError(t, database, `UPDATE mailbox_delivery_attempts
SET claim_request_fingerprint = 'changed'
WHERE mailbox_message_ref = 'message:mailbox-one' AND fence = 1`)
	mustV10Exec(t, database, `UPDATE mailbox_delivery_attempts SET
    consumption_request_ref = 'request:mailbox-consume',
    consumption_request_fingerprint = 'fingerprint:mailbox-consume',
    consumption_authorization_receipt_ref = 'auth:mailbox-consumption',
    consumption_ref = 'receipt:mailbox-consumption', consumed_at = ?
WHERE mailbox_message_ref = 'message:mailbox-one' AND fence = 1`, base+4)
	mailboxRequireSQLError(t, database, `UPDATE mailbox_delivery_attempts
SET delivered_at = delivered_at + 1
WHERE mailbox_message_ref = 'message:mailbox-one' AND fence = 1`)

	mustV10Exec(t, database, `UPDATE outbox SET completed_at = ?
WHERE ref = 'action:mailbox-one'`, base+4)
	mustV10Exec(t, database, `INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, mailbox_message_ref, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at
) VALUES (
    'action:mailbox-one', 'deliver_mailbox', ?, ?, ?, ?, ?,
    'message:mailbox-one', 1, 1, 'claim:mailbox-one',
    'principal:mailbox-recipient', 'completed', '', ?
)`, goalRef, parentRef, parentExecutionRef, planGeneration, parentGeneration, base+4)
	mustV10Exec(t, database, `INSERT INTO mailbox_delivery_acks(
	    ref, request_ref, request_fingerprint, authorization_receipt_ref,
	    mailbox_message_ref, action_ref, project_ref,
	    expected_goal_revision, expected_plan_generation, recipient_principal_ref,
	    outcome, effect_or_rework_ref, acked_at
	) VALUES (
	    'receipt:mailbox-ack', 'request:mailbox-ack', 'fingerprint:mailbox-ack',
	    'auth:mailbox-ack', 'message:mailbox-one', 'action:mailbox-one', ?, 1, ?,
	    'principal:mailbox-recipient', 'acknowledged', 'effect:mailbox-child', ?
	)`, projectRef, planGeneration, base+5)
	mustV10Exec(t, database, `INSERT INTO goal_child_handoff_resolutions(
    goal_ref, parent_work_item_ref, child_work_item_ref, mailbox_message_ref,
    outcome, receipt_ref, resolved_at
) VALUES (?, ?, ?, 'message:mailbox-one', 'acknowledged',
          'receipt:mailbox-ack', ?)`, goalRef, parentRef, childRef, base+5)
	mailboxRequireSQLError(t, database, `INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, mailbox_message_ref, available_at, completed_at
) VALUES ('action:mailbox-terminal-duplicate', 'deliver_mailbox', ?, ?, ?, ?, ?,
          'message:mailbox-one', ?, ?)`, goalRef, parentRef, parentExecutionRef,
		planGeneration, parentGeneration, base+6, base+6)
	mailboxRequireSQLError(t, database, `UPDATE mailbox_delivery_acks
SET outcome = 'blocked' WHERE ref = 'receipt:mailbox-ack'`)
	mailboxRequireSQLError(t, database, `UPDATE goal_child_handoff_resolutions
SET outcome = 'blocked' WHERE receipt_ref = 'receipt:mailbox-ack'`)

	var violations int
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil {
		t.Fatal(err)
	}
	if violations != 0 {
		t.Fatalf("mailbox causal fixture foreign-key violations=%d", violations)
	}
}

type mailboxEnvelopeSeed struct {
	Ref, RequestRef, Fingerprint, ProjectRef, GoalRef, Kind, Hash   string
	SourcePrincipalRef, ChildWorkItemRef, SourceExecutionRef        string
	RecipientPrincipalRef, ParentWorkItemRef, RecipientExecutionRef string
	PlanGeneration, RecipientGeneration                             int64
	AdmittedAt                                                      int64
}

func mailboxInsertEnvelope(t *testing.T, database *sql.DB, seed mailboxEnvelopeSeed) {
	t.Helper()
	mustV10Exec(t, database, `INSERT INTO mailbox_envelopes(
    ref, request_ref, request_fingerprint, project_ref, goal_ref, plan_generation,
    kind, summary, content_hash, source_principal_ref, child_work_item_ref,
    source_execution_ref, source_work_item_generation, recipient_principal_ref,
    parent_work_item_ref, recipient_execution_ref, recipient_work_item_generation,
    admitted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 'compact mailbox summary', ?, ?, ?, ?, 1, ?, ?, ?, ?, ?)`,
		seed.Ref, seed.RequestRef, seed.Fingerprint, seed.ProjectRef, seed.GoalRef,
		seed.PlanGeneration, seed.Kind, seed.Hash, seed.SourcePrincipalRef,
		seed.ChildWorkItemRef, seed.SourceExecutionRef, seed.RecipientPrincipalRef,
		seed.ParentWorkItemRef, seed.RecipientExecutionRef, seed.RecipientGeneration,
		seed.AdmittedAt)
}

func mailboxInsertAuthorization(
	t *testing.T,
	database *sql.DB,
	ref, principalRef, projectRef, permission, resourceRef string,
	at int64,
) {
	t.Helper()
	mustV10Exec(t, database, `INSERT INTO authorization_receipts(
    ref, request_ref, request_fingerprint, principal_ref, project_ref,
    permission, resource_ref, requested_at, outcome, role,
    membership_revision, reason_code, decided_at, recorded_at
) VALUES (?, 'request:' || ?, 'fingerprint:' || ?, ?, ?, ?, ?, ?,
          'allowed', 'project_owner', 1, 'allowed', ?, ?)`,
		ref, ref, ref, principalRef, projectRef, permission, resourceRef, at, at, at)
}

func mailboxInsertDeliveryAction(
	t *testing.T,
	database *sql.DB,
	ref, messageRef, goalRef, workItemRef, executionRef string,
	planGeneration, workItemGeneration, availableAt int64,
) {
	t.Helper()
	mustV10Exec(t, database, `INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, plan_generation,
    work_item_generation, mailbox_message_ref, available_at
) VALUES (?, 'deliver_mailbox', ?, ?, ?, ?, ?, ?, ?)`,
		ref, goalRef, workItemRef, executionRef, planGeneration,
		workItemGeneration, messageRef, availableAt)
}

func mailboxRequireSQLError(t *testing.T, database *sql.DB, query string, arguments ...any) {
	t.Helper()
	if _, err := database.Exec(query, arguments...); err == nil {
		t.Fatalf("mailbox schema accepted invalid SQL:\n%s", query)
	}
}

func mailboxUpgradeV7(t *testing.T, database *sql.DB) {
	t.Helper()
	database.SetMaxOpenConns(1)
	connection, err := database.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	transaction, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	migrations, err := loadMigrations()
	if err != nil || len(migrations) < 8 {
		t.Fatalf("load migration chain: count=%d err=%v", len(migrations), err)
	}
	current, migrated, err := applyMigrationSteps(context.Background(), transaction, migrations[:7], 5)
	if err != nil {
		t.Fatal(err)
	}
	if current != 7 || !migrated {
		t.Fatalf("V7 seed migration: current=%d migrated=%v", current, migrated)
	}
	if err := verifyForeignKeys(context.Background(), transaction); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
}

func mailboxRows(t *testing.T, database *sql.DB, query string, arguments ...any) []string {
	t.Helper()
	rows, err := database.Query(query, arguments...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var result []string
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			t.Fatal(err)
		}
		result = append(result, fmt.Sprint(values))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(result)
	return result
}

func mailboxRequireSchemaObject(t *testing.T, database *sql.DB, objectType, name string) {
	t.Helper()
	var count int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM sqlite_schema WHERE type = ? AND name = ?`, objectType, name,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("schema object %s %s count=%d", objectType, name, count)
	}
}

func mailboxUnix(at time.Time) int64 { return at.UTC().UnixNano() }
