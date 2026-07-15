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

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func TestV10MigrationPreservesSealedV09StateAndReopensIdempotently(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v09", "orquesta.sqlite")
	seedV09DatabaseForV10(t, path)

	raw := openRawV10TestDatabase(t, path)
	before := v10LegacyRows(t, raw)
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("migrate sealed V09 state: %v", err)
	}

	assertV10MigrationState(t, repository)
	after := v10LegacyRows(t, repository.db)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("V09 durable rows changed during V10 migration:\nbefore=%v\nafter=%v", before, after)
	}
	record, err := repository.GetGoal(context.Background(), mustRef(t, "goal:v10-v09", goal.NewGoalRef))
	if err != nil || record.Goal.Ref().String() != "goal:v10-v09" || record.Goal.Revision() != 3 {
		t.Fatalf("migrated aggregate differs: record=%+v err=%v", record, err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close first V10 open: %v", err)
	}

	reopened, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("reopen migrated V10 state: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	assertV10MigrationState(t, reopened)
	if second := v10LegacyRows(t, reopened.db); !reflect.DeepEqual(second, before) {
		t.Fatalf("second Open repeated or changed migration: before=%v after=%v", before, second)
	}
}

func TestV10MigrationBindsHistoricalRequesterToAppSpecConfirmer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reviewed-v09", "orquesta.sqlite")
	seedV09DatabaseForV10(t, path)
	raw := openRawV10TestDatabase(t, path)
	var appSpecTriggerSQL, executionTriggerSQL string
	if err := raw.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type = 'trigger' AND name = 'app_specs_immutable_update'`).Scan(&appSpecTriggerSQL); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type = 'trigger' AND name = 'executions_identity_immutable'`).Scan(&executionTriggerSQL); err != nil {
		t.Fatal(err)
	}
	record, err := readRecoveryV09GoalRecord(context.Background(), raw, "goal:v10-v09")
	if err != nil {
		t.Fatal(err)
	}
	original := record.Goal.AppSpec()
	reviewer := mustRef(t, "actor:historical-reviewer", goal.NewActorRef)
	reviewed, err := goal.NewInitialAppSpec(goal.AppSpecInput{
		Ref: original.Ref(), Intent: original.Intent(), Objective: original.Objective(),
		Reason: original.Reason(), ConfirmedBy: reviewer, ConfirmedAt: original.ConfirmedAt(),
	})
	if err != nil {
		t.Fatal(err)
	}
	mustV10Exec(t, raw, `DROP TRIGGER app_specs_immutable_update`)
	mustV10Exec(t, raw, `DROP TRIGGER executions_identity_immutable`)
	mustV10Exec(t, raw, `UPDATE app_specs SET confirmed_by = ?, hash = ?
WHERE ref = (SELECT app_spec_ref FROM goals WHERE ref = 'goal:v10-v09')`, reviewer.String(), reviewed.Hash())
	mustV10Exec(t, raw, `UPDATE executions SET spec_hash = ? WHERE goal_ref = 'goal:v10-v09'`, reviewed.Hash())
	mustV10Exec(t, raw, appSpecTriggerSQL)
	mustV10Exec(t, raw, executionTriggerSQL)
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	var requestedBy string
	if err := repository.db.QueryRow(`SELECT requested_by_ref FROM goals WHERE ref = 'goal:v10-v09'`).Scan(&requestedBy); err != nil {
		t.Fatal(err)
	}
	if requestedBy != "migration:v09:actor:historical-reviewer" {
		t.Fatalf("historical requested_by=%q", requestedBy)
	}
}

func TestV10IdentitySchemaEnforcesHierarchyRolesCASAndAppendOnlyReceipts(t *testing.T) {
	repository, _ := openTestRepository(t)
	database := repository.db
	base := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC).UnixNano()

	mustV10Exec(t, database, `INSERT INTO principals(ref, actor_ref, kind, authentication_method) VALUES
('principal:v10-owner', 'actor:v10-owner', 'human', 'local_token'),
('principal:v10-service', 'actor:v10-service', 'service', 'local_token')`)
	mustV10Exec(t, database, `INSERT INTO workspaces(ref) VALUES ('workspace:v10')`)
	mustV10Exec(t, database, `INSERT INTO groups(ref, workspace_ref) VALUES ('group:v10', 'workspace:v10')`)
	mustV10Exec(t, database, `INSERT INTO projects(ref, group_ref) VALUES ('project:v10', 'group:v10')`)
	mustV10Exec(t, database, `INSERT INTO repositories(ref, project_ref) VALUES ('repository:v10', 'project:v10')`)

	if _, err := database.Exec(`INSERT INTO projects(ref, group_ref) VALUES ('project:orphan', 'group:missing')`); err == nil {
		t.Fatal("project without its exact group parent was accepted")
	}
	if _, err := database.Exec(`UPDATE groups SET workspace_ref = 'workspace:other' WHERE ref = 'group:v10'`); err == nil {
		t.Fatal("immutable hierarchy parent changed")
	}
	if _, err := database.Exec(`UPDATE principals SET actor_ref = 'actor:other' WHERE ref = 'principal:v10-owner'`); err == nil {
		t.Fatal("immutable principal changed")
	}

	roles := []string{
		"platform_admin", "project_owner", "project_admin", "contributor",
		"reviewer", "operator", "viewer",
	}
	for index, role := range roles {
		principal := fmt.Sprintf("principal:v10-role-%d", index)
		actor := fmt.Sprintf("actor:v10-role-%d", index)
		mustV10Exec(t, database,
			`INSERT INTO principals(ref, actor_ref, kind, authentication_method) VALUES (?, ?, 'human', 'test')`,
			principal, actor,
		)
		mustV10Exec(t, database, `INSERT INTO project_memberships(
principal_ref, project_ref, role, revision, status, granted_by_ref, granted_at
) VALUES (?, 'project:v10', ?, 1, 'active', 'principal:v10-owner', ?)`, principal, role, base)
	}
	if _, err := database.Exec(`INSERT INTO project_memberships(
principal_ref, project_ref, role, revision, status, granted_by_ref, granted_at
) VALUES ('principal:v10-service', 'project:v10', 'owner', 1, 'active', 'principal:v10-owner', ?)`, base); err == nil {
		t.Fatal("unknown role was accepted")
	}
	if _, err := database.Exec(`UPDATE project_memberships SET revision = 3
WHERE principal_ref = 'principal:v10-role-6' AND project_ref = 'project:v10'`); err == nil {
		t.Fatal("membership skipped expected revision")
	}
	mustV10Exec(t, database, `INSERT INTO membership_audit_receipts(
ref, request_ref, request_fingerprint, action, actor_ref, target_ref, project_ref,
role, previous_revision, revision, occurred_at
) VALUES ('audit:v10-grant', 'request:v10-grant', 'fingerprint:v10-grant', 'membership.granted',
'principal:v10-owner', 'principal:v10-role-6', 'project:v10', 'viewer', 0, 1, ?)`, base)
	mustV10Exec(t, database, `UPDATE project_memberships
SET revision = 2, status = 'revoked', revoked_by_ref = 'principal:v10-owner', revoked_at = ?
WHERE principal_ref = 'principal:v10-role-6' AND project_ref = 'project:v10'`, base+1)
	mustV10Exec(t, database, `INSERT INTO membership_audit_receipts(
ref, request_ref, request_fingerprint, action, actor_ref, target_ref, project_ref,
role, previous_revision, revision, occurred_at
) VALUES ('audit:v10-revoke', 'request:v10-revoke', 'fingerprint:v10-revoke', 'membership.revoked',
'principal:v10-owner', 'principal:v10-role-6', 'project:v10', 'viewer', 1, 2, ?)`, base+1)
	if _, err := database.Exec(`UPDATE membership_audit_receipts SET role = 'operator' WHERE ref = 'audit:v10-grant'`); err == nil {
		t.Fatal("membership audit receipt was mutable")
	}
	if _, err := database.Exec(`DELETE FROM membership_audit_receipts WHERE ref = 'audit:v10-grant'`); err == nil {
		t.Fatal("membership audit receipt was deletable")
	}
	if _, err := database.Exec(`INSERT INTO membership_audit_receipts(
ref, request_ref, request_fingerprint, action, actor_ref, target_ref, project_ref,
role, previous_revision, revision, occurred_at
) VALUES ('audit:v10-replay-conflict', 'request:v10-grant', 'different', 'membership.granted',
'principal:v10-owner', 'principal:v10-role-6', 'project:v10', 'viewer', 1, 2, ?)`, base+2); err == nil {
		t.Fatal("divergent membership request replay was accepted")
	}

	permissions := []string{
		"project.hierarchy.manage", "project.membership.manage", "goals.create", "goals.amend",
		"goals.get", "goals.list", "artifacts.read", "project.status",
	}
	for index, permission := range permissions {
		mustV10Exec(t, database, `INSERT INTO authorization_receipts(
ref, request_ref, request_fingerprint, principal_ref, project_ref, permission,
resource_ref, requested_at, outcome, role, membership_revision, reason_code,
decided_at, recorded_at
) VALUES (?, ?, ?, 'principal:v10-owner', 'project:unknown', ?, 'resource:unknown', ?,
'denied', '', 0, 'rbac.permission_denied', ?, ?)`,
			fmt.Sprintf("authorization:v10-%d", index), fmt.Sprintf("request:v10-auth-%d", index),
			fmt.Sprintf("fingerprint:v10-auth-%d", index), permission, base, base, base,
		)
	}
	if _, err := database.Exec(`INSERT INTO authorization_receipts(
ref, request_ref, request_fingerprint, principal_ref, project_ref, permission,
resource_ref, requested_at, outcome, role, membership_revision, reason_code,
decided_at, recorded_at
) VALUES ('authorization:v10-invalid', 'request:v10-invalid', 'fingerprint:v10-invalid',
'principal:v10-owner', 'project:unknown', 'unknown', 'resource:unknown', ?,
'denied', '', 0, 'rbac.permission_denied', ?, ?)`, base, base, base); err == nil {
		t.Fatal("unknown permission was accepted")
	}
	if _, err := database.Exec(`INSERT INTO authorization_receipts(
ref, request_ref, request_fingerprint, principal_ref, project_ref, permission,
resource_ref, requested_at, outcome, role, membership_revision, reason_code,
decided_at, recorded_at
) VALUES ('authorization:v10-replay-conflict', 'request:v10-auth-0', 'different',
'principal:v10-owner', 'project:unknown', 'goals.get', 'resource:unknown', ?,
'denied', '', 0, 'rbac.permission_denied', ?, ?)`, base, base, base); err == nil {
		t.Fatal("divergent authorization request replay was accepted")
	}
	if _, err := database.Exec(`UPDATE authorization_receipts SET reason_code = 'changed' WHERE ref = 'authorization:v10-0'`); err == nil {
		t.Fatal("authorization receipt was mutable")
	}
	if _, err := database.Exec(`DELETE FROM authorization_receipts WHERE ref = 'authorization:v10-0'`); err == nil {
		t.Fatal("authorization receipt was deletable")
	}
}

func TestV10MigrationCorruptionRollsBackSchemaDataReceiptAndVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt-v09", "orquesta.sqlite")
	seedV09DatabaseForV10(t, path)
	database := openRawV10TestDatabase(t, path)
	mustV10Exec(t, database, `DROP TRIGGER goals_app_spec_immutable`)
	mustV10Exec(t, database, `UPDATE goals SET actor_ref = '' WHERE ref = 'goal:v10-v09'`)
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if repository != nil {
		_ = repository.Close()
		t.Fatal("corrupt V09 state unexpectedly migrated")
	}
	if !application.IsStateError(err, application.StateConflict) &&
		!application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("corrupt migration error=%v", err)
	}

	database = openRawV10TestDatabase(t, path)
	defer database.Close()
	var version, receipt, v10Tables, requestedByColumns int
	if err := database.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 6`).Scan(&receipt); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema
WHERE type = 'table' AND name IN (
    'principals', 'workspaces', 'groups', 'projects', 'repositories',
    'project_memberships', 'membership_audit_receipts', 'authorization_receipts', 'goals_v6'
)`).Scan(&v10Tables); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('goals') WHERE name = 'requested_by_ref'`).Scan(&requestedByColumns); err != nil {
		t.Fatal(err)
	}
	if version != 5 || receipt != 0 || v10Tables != 0 || requestedByColumns != 0 {
		t.Fatalf("partial V10 migration escaped: version=%d receipt=%d tables=%d requested_by=%d",
			version, receipt, v10Tables, requestedByColumns)
	}
}

func seedV09DatabaseForV10(t *testing.T, path string) {
	t.Helper()
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatal(err)
	}
	database := openRawV10TestDatabase(t, path)
	defer database.Close()
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
	if err != nil || len(migrations) < 6 {
		t.Fatalf("load migration chain: count=%d err=%v", len(migrations), err)
	}
	base := time.Date(2026, 7, 15, 7, 0, 0, 0, time.UTC)
	for index := 0; index < 5; index++ {
		item := migrations[index]
		if item.version == 5 {
			if err := validateAtomicStateMigrationSource(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := transaction.Exec(item.preSQL); err != nil {
			t.Fatalf("apply V09 migration %d: %v", item.version, err)
		}
		if index == 0 {
			seedV1Goal(t, transaction, "v10-v09", "running", 3, "pending", 1, "queued", base, false)
		}
		if item.backfill {
			if err := backfillAppSpecs(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
		}
		if strings.TrimSpace(item.postSQL) != "" {
			if _, err := transaction.Exec(item.postSQL); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := transaction.Exec(
			`INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)`,
			item.version, item.name, item.checksum,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := transaction.Exec(`PRAGMA user_version = 5`); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
}

func assertV10MigrationState(t *testing.T, repository *Repository) {
	t.Helper()
	var version, receipts, principals, hierarchyRows, memberships, requestedBy, violations int
	if err := repository.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM principals
WHERE ref = 'migration:v09:actor:v1' AND actor_ref = 'actor:v1'
  AND kind = 'human' AND authentication_method = 'migration.v09'`).Scan(&principals); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT
    (SELECT COUNT(*) FROM workspaces) + (SELECT COUNT(*) FROM groups) +
    (SELECT COUNT(*) FROM projects) + (SELECT COUNT(*) FROM repositories)`).Scan(&hierarchyRows); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM project_memberships`).Scan(&memberships); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM goals g
JOIN app_specs spec ON spec.ref = g.app_spec_ref
WHERE g.ref = 'goal:v10-v09'
  AND g.requested_by_ref = 'migration:v09:' || spec.confirmed_by`).Scan(&requestedBy); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil {
		t.Fatal(err)
	}
	if version != recoverySchemaV12 || receipts != recoverySchemaV12 || principals != 1 || hierarchyRows != 0 ||
		memberships != 0 || requestedBy != 1 || violations != 0 {
		t.Fatalf("V10 migration state invalid: version=%d receipts=%d principals=%d hierarchy=%d memberships=%d requested_by=%d fk=%d",
			version, receipts, principals, hierarchyRows, memberships, requestedBy, violations)
	}
}

func v10LegacyRows(t *testing.T, database *sql.DB) map[string][]string {
	t.Helper()
	queries := map[string]string{
		"intents":                     `SELECT * FROM intents`,
		"app_specs":                   `SELECT * FROM app_specs`,
		"goals":                       `SELECT ref, request_ref, request_fingerprint, app_spec_ref, actor_ref, project_ref, state, revision, created_at, started_at, closed_at, plan_generation FROM goals`,
		"goal_phases":                 `SELECT * FROM goal_phases`,
		"goal_phase_contract_refs":    `SELECT * FROM goal_phase_contract_refs`,
		"work_items":                  `SELECT * FROM work_items`,
		"work_item_dependencies":      `SELECT * FROM work_item_dependencies`,
		"work_item_write_scopes":      `SELECT * FROM work_item_write_scopes`,
		"work_item_requirement_refs":  `SELECT * FROM work_item_requirement_refs`,
		"executions":                  `SELECT * FROM executions`,
		"artifacts":                   `SELECT * FROM artifacts`,
		"attestations":                `SELECT * FROM attestations`,
		"events":                      `SELECT * FROM events`,
		"outbox":                      `SELECT * FROM outbox`,
		"work_item_fences":            `SELECT * FROM work_item_fences`,
		"action_consumption_receipts": `SELECT * FROM action_consumption_receipts`,
	}
	result := make(map[string][]string, len(queries))
	for table, query := range queries {
		rows, err := database.Query(query)
		if err != nil {
			t.Fatalf("read legacy table %s: %v", table, err)
		}
		columns, err := rows.Columns()
		if err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for index := range values {
				destinations[index] = &values[index]
			}
			if err := rows.Scan(destinations...); err != nil {
				_ = rows.Close()
				t.Fatal(err)
			}
			result[table] = append(result[table], fmt.Sprint(values))
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		sort.Strings(result[table])
	}
	return result
}

func openRawV10TestDatabase(t *testing.T, path string) *sql.DB {
	t.Helper()
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func mustV10Exec(t *testing.T, database *sql.DB, query string, arguments ...any) {
	t.Helper()
	if _, err := database.Exec(query, arguments...); err != nil {
		t.Fatalf("V10 SQL failed: %v\n%s", err, query)
	}
}
