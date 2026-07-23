package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

const testBusyTimeout = 3 * time.Second

const fastSQLiteTestDurability sqliteDurabilityProfile = "OFF"

func TestRepositoryOpenAppliesPrivateModesMigrationsAndPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-state", "orquesta.sqlite")
	repository := openFullTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	directoryInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat state directory: %v", err)
	}
	if got := directoryInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("state directory mode = %o, want 700", got)
	}
	databaseInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat database: %v", err)
	}
	if got := databaseInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("database mode = %o, want 600", got)
	}

	var journalMode string
	if err := repository.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
	var foreignKeys, busyTimeout, userVersion, synchronous int
	if err := repository.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if err := repository.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("busy_timeout: %v", err)
	}
	if err := repository.db.QueryRow("PRAGMA user_version").Scan(&userVersion); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if err := repository.db.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatalf("synchronous: %v", err)
	}
	if foreignKeys != 1 || busyTimeout != int(testBusyTimeout.Milliseconds()) || userVersion != recoverySchemaV21 {
		t.Fatalf("pragmas = fk:%d busy:%d version:%d", foreignKeys, busyTimeout, userVersion)
	}
	if synchronous != 2 {
		t.Fatalf("public Open synchronous = %d, want FULL", synchronous)
	}

	rows, err := repository.db.Query("SELECT name FROM sqlite_schema WHERE type = 'table' ORDER BY name")
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}
	wantTables := []string{
		"action_consumption_receipts", "app_specs", "artifact_occurrences", "artifacts", "attestation_test_outcomes", "attestations", "authorization_receipts",
		"budget_envelopes", "budget_reservations", "budget_settlements", "change_set_paths", "change_sets",
		"command_invocations", "command_outcomes", "controls",
		"council_decisions", "council_facts", "council_rounds", "council_skips",
		"director_decisions", "director_lease_receipts", "director_leases", "effect_approvals", "effect_attempts", "effect_intents", "effect_receipts", "events", "executions", "fairness_cursors",
		"goal_child_handoff_resolutions", "goal_phase_contract_refs", "goal_phases", "goals", "groups", "integration_receipts", "intents",
		"mailbox_admission_receipts", "mailbox_artifact_refs", "mailbox_delivery_acks", "mailbox_delivery_attempts", "mailbox_envelopes", "mailbox_retirements",
		"membership_audit_receipts", "merge_observations", "outbox", "principals", "project_memberships", "projects", "repositories", "review_records", "schema_migrations",
		"work_item_authorities", "work_item_dependencies", "work_item_fences", "work_item_required_test_arguments", "work_item_required_tests", "work_item_requirement_refs", "work_item_write_scopes", "work_items",
		"workspace_binding_write_scopes", "workspace_bindings", "workspaces",
	}
	if !reflect.DeepEqual(tables, wantTables) {
		t.Fatalf("tables = %#v, want %#v", tables, wantTables)
	}
	var migrationName string
	if err := repository.db.QueryRow("SELECT name FROM schema_migrations WHERE version = 1").Scan(&migrationName); err != nil {
		t.Fatalf("migration receipt: %v", err)
	}
	if migrationName != "001_initial.sql" {
		t.Fatalf("migration name = %q", migrationName)
	}
	if err := repository.db.QueryRow("SELECT name FROM schema_migrations WHERE version = 2").Scan(&migrationName); err != nil {
		t.Fatalf("DAG migration receipt: %v", err)
	}
	if migrationName != "002_dag.sql" {
		t.Fatalf("DAG migration name = %q", migrationName)
	}
	if err := repository.db.QueryRow("SELECT name FROM schema_migrations WHERE version = 3").Scan(&migrationName); err != nil {
		t.Fatalf("AppSpec migration receipt: %v", err)
	}
	if migrationName != "003_app_specs.sql" {
		t.Fatalf("AppSpec migration name = %q", migrationName)
	}
	if err := repository.db.QueryRow("SELECT name FROM schema_migrations WHERE version = 4").Scan(&migrationName); err != nil {
		t.Fatalf("phase contract migration receipt: %v", err)
	}
	if migrationName != "004_phase_contracts.sql" {
		t.Fatalf("phase contract migration name = %q", migrationName)
	}
	if err := repository.db.QueryRow("SELECT name FROM schema_migrations WHERE version = 5").Scan(&migrationName); err != nil {
		t.Fatalf("atomic state migration receipt: %v", err)
	}
	if migrationName != "005_atomic_state_outbox.sql" {
		t.Fatalf("atomic state migration name = %q", migrationName)
	}
	var appSpecColumns, intentColumns, foreignKeyViolations int
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('goals') WHERE name = 'app_spec_ref'`).Scan(&appSpecColumns); err != nil {
		t.Fatalf("goals app_spec_ref column: %v", err)
	}
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('goals') WHERE name = 'intent_ref'`).Scan(&intentColumns); err != nil {
		t.Fatalf("goals legacy intent_ref column: %v", err)
	}
	if err := repository.db.QueryRow("SELECT COUNT(*) FROM pragma_foreign_key_check").Scan(&foreignKeyViolations); err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	if appSpecColumns != 1 || intentColumns != 0 || foreignKeyViolations != 0 {
		t.Fatalf("Goal binding columns app_spec=%d intent=%d fk_violations=%d", appSpecColumns, intentColumns, foreignKeyViolations)
	}

	_, err = repository.db.Exec(`
INSERT INTO events(ref, kind, goal_ref, work_item_ref, execution_ref, occurred_at)
VALUES ('event:invalid-fk', 'invalid', 'goal:missing', 'work:missing', 'execution:missing', 1)`)
	if err == nil {
		t.Fatalf("foreign key violation accepted")
	}
	if _, err := repository.db.Exec(`INSERT INTO workspaces(ref) VALUES ('workspace:full-restart')`); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openFullTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	var count int
	if err := restarted.db.QueryRow(`SELECT COUNT(*) FROM workspaces WHERE ref='workspace:full-restart'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("FULL restart committed=%d err=%v", count, err)
	}
}

func TestFastSemanticRepositoryRestartPersistsCommittedData(t *testing.T) {
	repository, path := openTestRepository(t)
	if _, err := repository.db.Exec(`INSERT INTO workspaces(ref) VALUES ('workspace:fast-restart')`); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openFastTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	var synchronous, count int
	if err := restarted.db.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatal(err)
	}
	if err := restarted.db.QueryRow(`SELECT COUNT(*) FROM workspaces WHERE ref='workspace:fast-restart'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if synchronous != 0 || count != 1 {
		t.Fatalf("fast restart synchronous=%d committed=%d", synchronous, count)
	}
}

func TestRepositorySerializesWritersBeforeSQLiteBusyTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-state", "orquesta.sqlite")
	const busyTimeout = time.Millisecond
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: busyTimeout, MaxOpenConnections: 4,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = repository.Close() })
	if got := repository.writer.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("writer max connections = %d, want 1", got)
	}
	if got := repository.db.Stats().MaxOpenConnections; got != 4 {
		t.Fatalf("reader max connections = %d, want 4", got)
	}

	first, err := beginTransaction(context.Background(), repository)
	sqliteTestNoError(t, err)
	defer first.Rollback()
	if _, err := first.Exec(`INSERT INTO workspaces(ref) VALUES ('workspace:writer-first')`); err != nil {
		t.Fatal(err)
	}

	type writeResult struct{ err error }
	result := make(chan writeResult, 1)
	started := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	waitsBefore := repository.writer.Stats().WaitCount
	go func() {
		close(started)
		second, beginErr := beginTransaction(ctx, repository)
		if beginErr != nil {
			result <- writeResult{err: beginErr}
			return
		}
		defer second.Rollback()
		if _, execErr := second.ExecContext(ctx, `INSERT INTO workspaces(ref) VALUES ('workspace:writer-second')`); execErr != nil {
			result <- writeResult{err: execErr}
			return
		}
		result <- writeResult{err: commit(second)}
	}()
	<-started
	deadline := time.Now().Add(250 * time.Millisecond)
	for repository.writer.Stats().WaitCount == waitsBefore && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if repository.writer.Stats().WaitCount == waitsBefore {
		t.Fatal("second writer did not queue in dedicated pool")
	}
	time.Sleep(5 * busyTimeout)
	select {
	case got := <-result:
		t.Fatalf("second writer escaped before first commit: %v", got.err)
	default:
	}
	if err := commit(first); err != nil {
		t.Fatal(err)
	}
	if got := <-result; got.err != nil {
		t.Fatalf("queued writer failed after release: %v", got.err)
	}
	var count int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM workspaces WHERE ref LIKE 'workspace:writer-%'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("committed writers = %d, want 2", count)
	}
}

func TestRepositoryRejectsChangedAppliedMigration(t *testing.T) {
	repository, path := openTestRepository(t)
	if _, err := repository.db.Exec("UPDATE schema_migrations SET checksum = 'sha256:tampered' WHERE version = 1"); err != nil {
		t.Fatalf("tamper migration receipt: %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if reopened != nil {
		_ = reopened.Close()
		t.Fatalf("tampered migration history reopened")
	}
	if !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("tampered migration error = %v", err)
	}
}

func TestRepositoryMigratesPopulatedV1StateToDAGSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy", "orquesta.sqlite")
	seedPopulatedV1Database(t, path)
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("migrate populated V1: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })

	runningRef := mustRef(t, "goal:v1-running", goal.NewGoalRef)
	running, err := repository.GetGoal(context.Background(), runningRef)
	if err != nil || running.Goal.Revision() != 3 || running.Goal.PlanGeneration() != 1 ||
		len(running.Executions) != 1 || running.Executions[0].State != application.ExecutionQueued ||
		!running.Executions[0].StartedAt.IsZero() || !running.Executions[0].DeadlineAt.IsZero() {
		t.Fatalf("migrated running goal = %+v err=%v", running, err)
	}
	runningItem := onlyItem(t, running.Goal)
	if runningItem.Phase() != goal.DefaultPhaseKey() || runningItem.Role() != goal.DefaultRoleKey() ||
		runningItem.OutputContract().Kind() != goal.OutputContractEvidenceBundle {
		t.Fatalf("legacy plan defaults lost: %+v", runningItem)
	}

	closedRef := mustRef(t, "goal:v1-closed", goal.NewGoalRef)
	closed, err := repository.GetGoal(context.Background(), closedRef)
	if err != nil || closed.Goal.Revision() != 6 || closed.Goal.State() != goal.GoalStateSucceeded ||
		len(closed.Executions) != 1 || len(closed.Artifacts) != 1 || len(closed.Attestations) != 1 {
		t.Fatalf("migrated closed goal = %+v err=%v", closed, err)
	}
	status, err := repository.Status(context.Background(), running.Goal.Project())
	if err != nil || status.PendingActions != 1 || status.Goals != 2 {
		t.Fatalf("migrated status = %+v err=%v", status, err)
	}
}

func seedPopulatedV1Database(t *testing.T, path string) {
	t.Helper()
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatalf("prepare V1 path: %v", err)
	}
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if err != nil {
		t.Fatalf("open V1 seed: %v", err)
	}
	defer database.Close()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	transaction, err := database.Begin()
	if err != nil {
		t.Fatalf("begin V1 seed: %v", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.Exec(migrations[0].sql); err != nil {
		t.Fatalf("apply V1 schema: %v", err)
	}
	if _, err := transaction.Exec(
		"INSERT INTO schema_migrations(version, name, checksum) VALUES (1, ?, ?)",
		migrations[0].name, migrations[0].checksum,
	); err != nil {
		t.Fatalf("record V1 migration: %v", err)
	}
	if _, err := transaction.Exec("PRAGMA user_version = 1"); err != nil {
		t.Fatalf("set V1 user_version: %v", err)
	}
	base := time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC)
	seedV1Goal(t, transaction, "v1-running", "running", 3, "pending", 1, "queued", base, false)
	seedV1Goal(t, transaction, "v1-closed", "succeeded", 6, "succeeded", 3, "succeeded", base.Add(time.Minute), true)
	if err := transaction.Commit(); err != nil {
		t.Fatalf("commit V1 seed: %v", err)
	}
}

func seedV1Goal(
	t *testing.T,
	transaction *sql.Tx,
	suffix, goalState string,
	goalRevision int,
	itemState string,
	itemRevision int,
	executionState string,
	base time.Time,
	withEvidence bool,
) {
	t.Helper()
	actor := mustRef(t, "actor:v1", goal.NewActorRef)
	project := mustRef(t, "project:v1", goal.NewProjectRef)
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref: mustRef(t, "intent:"+suffix, goal.NewIntentRef), Actor: actor, Project: project,
		Statement: "legacy " + suffix, SubmittedAt: base,
	})
	if err != nil {
		t.Fatalf("legacy intent: %v", err)
	}
	intentSnapshot := intent.Snapshot()
	goalRef := "goal:" + suffix
	itemRef := "work-item:" + suffix
	executionRef := "execution:" + suffix
	var closedAt, itemStartedAt, itemFinishedAt, executionStartedAt, providerAcceptedAt, observedAt, finishedAt any
	providerRef, externalRef := "", ""
	var itemExecutionRef any
	if itemState != "pending" {
		itemStartedAt = requiredTime(base)
		executionStartedAt = requiredTime(base)
		itemExecutionRef = executionRef
	}
	if withEvidence {
		closedAt = requiredTime(base.Add(2 * time.Second))
		itemFinishedAt = requiredTime(base.Add(time.Second))
		providerAcceptedAt = requiredTime(base)
		observedAt = requiredTime(base.Add(time.Second))
		finishedAt = requiredTime(base.Add(time.Second))
		providerRef, externalRef = "provider:v1", "external:v1"
	}
	mustExec := func(query string, arguments ...any) {
		t.Helper()
		if _, err := transaction.Exec(query, arguments...); err != nil {
			t.Fatalf("seed V1 %s: %v", suffix, err)
		}
	}
	mustExec(`INSERT INTO intents(ref, actor_ref, project_ref, statement, submitted_at, hash) VALUES (?, ?, ?, ?, ?, ?)`,
		intentSnapshot.Ref, intentSnapshot.ActorRef, intentSnapshot.ProjectRef, intentSnapshot.Statement,
		requiredTime(intentSnapshot.SubmittedAt), intentSnapshot.Hash)
	mustExec(`INSERT INTO goals(ref, request_ref, request_fingerprint, intent_ref, actor_ref, project_ref, state, revision, created_at, started_at, closed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, goalRef, "request:"+suffix, "fingerprint:"+suffix,
		intentSnapshot.Ref, actor.String(), project.String(), goalState, goalRevision, requiredTime(base), requiredTime(base), closedAt)
	mustExec(`INSERT INTO work_items(ref, goal_ref, actor_ref, project_ref, objective, state, revision, position, created_at, started_at, finished_at, execution_ref)
VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)`, itemRef, goalRef, actor.String(), project.String(),
		"legacy "+suffix, itemState, itemRevision, requiredTime(base), itemStartedAt, itemFinishedAt, itemExecutionRef)
	mustExec(`INSERT INTO executions(ref, goal_ref, work_item_ref, state, artifact_media_type, idempotency_key, max_output_bytes, max_attempts,
provider_ref, external_ref, created_at, deadline_at, started_at, provider_accepted_at, last_observed_at, provider_observed_at, finished_at, failure_code)
VALUES (?, ?, ?, ?, 'text/plain', ?, 1024, 3, ?, ?, ?, ?, ?, ?, ?, ?, ?, '')`, executionRef, goalRef, itemRef,
		executionState, "idempotency:"+suffix, providerRef, externalRef, requiredTime(base), requiredTime(base.Add(time.Hour)),
		executionStartedAt, providerAcceptedAt, observedAt, observedAt, finishedAt)
	mustExec(`INSERT INTO events(ref, kind, goal_ref, work_item_ref, execution_ref, occurred_at) VALUES (?, 'goal.created', ?, ?, ?, ?)`,
		"event:"+suffix, goalRef, itemRef, executionRef, requiredTime(base))
	completedAt := any(nil)
	claimToken, claimedBy, claimedUntil := any(nil), any(nil), any(nil)
	attempt := 0
	if withEvidence {
		completedAt = requiredTime(base.Add(time.Second))
		claimToken = "claim:migrated:" + suffix
		claimedBy = "worker:migrated"
		claimedUntil = requiredTime(base.Add(time.Hour))
		attempt = 1
		mustExec(`INSERT INTO artifacts(ref, goal_ref, work_item_ref, digest, media_type, size, created_at) VALUES (?, ?, ?, 'sha256:v1', 'text/plain', 2, ?)`,
			"artifact:"+suffix, goalRef, itemRef, requiredTime(base.Add(time.Second)))
		mustExec(`INSERT INTO attestations(ref, goal_ref, work_item_ref, execution_ref, artifact_ref, policy, accepted_at) VALUES (?, ?, ?, ?, ?, 'legacy', ?)`,
			"attestation:"+suffix, goalRef, itemRef, executionRef, "artifact:"+suffix, requiredTime(base.Add(time.Second)))
	}
	mustExec(`INSERT INTO outbox(
ref, kind, goal_ref, work_item_ref, execution_ref, available_at,
claim_token, claimed_by, claimed_until, attempt, completed_at
) VALUES (?, 'launch_agent', ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"action:"+suffix, goalRef, itemRef, executionRef, requiredTime(base),
		claimToken, claimedBy, claimedUntil, attempt, completedAt)
}

func TestRepositoryCreateGetListScopedIdempotencyAndRestart(t *testing.T) {
	repository, path := openTestRepository(t)
	first := newCreateFixture(t, "first", "request:shared", "fingerprint:first", "actor:local-owner", "project:default")
	record, created, err := createLegacyGoal(t, repository, first)
	if err != nil || !created {
		t.Fatalf("create first = created:%v err:%v", created, err)
	}
	assertRecordMatchesCreate(t, record, first)

	replayed, created, err := createLegacyGoal(t, repository, first)
	if err != nil || created || replayed.Goal.Ref() != first.Goal.Ref() {
		t.Fatalf("idempotent replay = created:%v ref:%s err:%v", created, replayed.Goal.Ref().String(), err)
	}
	freshReplay := newCreateFixtureWithStatement(
		t, "first-retry", "request:shared", "fingerprint:first", "actor:local-owner", "project:default",
		first.Goal.AppSpec().Intent().Statement(),
	)
	replayed, created, err = createLegacyGoal(t, repository, freshReplay)
	if err != nil || created || replayed.Goal.Ref() != first.Goal.Ref() {
		t.Fatalf("semantic replay with fresh refs = created:%v ref:%s err:%v", created, replayed.Goal.Ref().String(), err)
	}
	semanticConflict := newCreateFixtureWithStatement(
		t, "first-conflict", "request:shared", "fingerprint:first", "actor:local-owner", "project:default",
		"different statement with forged fingerprint",
	)
	if _, _, err := createLegacyGoal(t, repository, semanticConflict); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("semantic replay conflict = %v", err)
	}
	conflicting := first
	conflicting.RequestFingerprint = "fingerprint:changed"
	if _, _, err := createLegacyGoal(t, repository, conflicting); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("fingerprint conflict = %v", err)
	}

	second := newCreateFixture(t, "second", "request:shared", "fingerprint:second", "actor:local-owner", "project:other")
	if _, created, err := createLegacyGoal(t, repository, second); err != nil || !created {
		t.Fatalf("same request in other scope = created:%v err:%v", created, err)
	}
	got, err := repository.GetGoal(context.Background(), first.Goal.Ref())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	assertRecordMatchesCreate(t, got, first)
	listed, err := repository.ListGoals(context.Background(), first.Goal.Project(), 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 || listed[0].Ref != first.Goal.Ref() || listed[0].ArtifactCount != 0 {
		t.Fatalf("listed = %#v", listed)
	}
	if listed[0].IntentRef != first.Goal.Intent() || listed[0].AppSpecRef != first.Goal.AppSpec().Ref() ||
		listed[0].AppSpecGeneration != first.Goal.AppSpec().Generation() || listed[0].SpecHash != first.Goal.SpecHash() {
		t.Fatalf("listed AppSpec binding = %#v", listed[0])
	}
	status, err := repository.Status(context.Background(), first.Goal.Project())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Goals != 1 || status.RunningGoals != 1 || status.PendingActions != 1 || status.QuarantinedActions != 0 {
		t.Fatalf("status before restart = %+v", status)
	}

	if err := repository.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	repository, err = Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	got, err = repository.GetGoal(context.Background(), first.Goal.Ref())
	if err != nil {
		t.Fatalf("get after restart: %v", err)
	}
	assertRecordMatchesCreate(t, got, first)
	if _, created, err := createLegacyGoal(t, repository, first); err != nil || created {
		t.Fatalf("replay after restart = created:%v err:%v", created, err)
	}
	for table, want := range map[string]int{"app_specs": 2, "goals": 2, "intents": 2, "executions": 2, "events": 2, "outbox": 2} {
		if got := tableCount(t, repository, table); got != want {
			t.Fatalf("%s rows = %d, want %d", table, got, want)
		}
	}
}

func TestRepositoryListGoalsScopesProjectBeforeLimit(t *testing.T) {
	repository, _ := openTestRepository(t)
	owner := newCreateFixture(t, "a-owner", "request:owner", "fingerprint:owner", "actor:owner", "project:shared")
	other := newCreateFixture(t, "z-other", "request:other", "fingerprint:other", "actor:other", "project:shared")
	for _, state := range []application.CreateGoalState{owner, other} {
		if _, created, err := createLegacyGoal(t, repository, state); err != nil || !created {
			t.Fatalf("create %s = created:%v err:%v", state.Goal.Ref(), created, err)
		}
	}

	listed, err := repository.ListGoals(context.Background(), owner.Goal.Project(), 1)
	if err != nil {
		t.Fatalf("list owner: %v", err)
	}
	if len(listed) != 1 || listed[0].Ref != other.Goal.Ref() {
		t.Fatalf("project-scoped list = %#v", listed)
	}
}

func TestRepositoryConcurrentCreateIsIdempotent(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := authorizeLegacyCreateState(t, repository, newCreateFixture(t, "concurrent", "request:concurrent", "fingerprint:concurrent", "actor:local-owner", "project:default"))
	const callers = 12
	start := make(chan struct{})
	errorsByCall := make(chan error, callers)
	var createdCount atomic.Int64
	var wait sync.WaitGroup
	for index := 0; index < callers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			record, created, err := repository.CreateGoal(context.Background(), state)
			if err == nil && record.Goal.Ref() != state.Goal.Ref() {
				err = errors.New("wrong goal returned")
			}
			if created {
				createdCount.Add(1)
			}
			errorsByCall <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsByCall)
	for err := range errorsByCall {
		if err != nil {
			t.Fatalf("concurrent create: %v", err)
		}
	}
	if createdCount.Load() != 1 {
		t.Fatalf("created count = %d, want 1", createdCount.Load())
	}
	if got := tableCount(t, repository, "goals"); got != 1 {
		t.Fatalf("goal rows = %d, want 1", got)
	}
}

func TestRepositoryRejectsInvalidExecutionAndLifecycleContracts(t *testing.T) {
	repository, _ := openTestRepository(t)

	deadline := newCreateFixture(t, "deadline", "request:deadline", "fingerprint:deadline", "actor:local-owner", "project:default")
	deadline.Executions[0].DeadlineAt = deadline.Executions[0].CreatedAt
	if _, _, err := createLegacyGoal(t, repository, deadline); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("non-strict deadline = %v", err)
	}

	queuedProvider := newCreateFixture(t, "queued-provider", "request:queued-provider", "fingerprint:queued-provider", "actor:local-owner", "project:default")
	queuedProvider.Executions[0].ProviderRef = "provider:unexpected"
	queuedProvider.Executions[0].ExternalRef = "external:unexpected"
	if _, _, err := createLegacyGoal(t, repository, queuedProvider); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("queued provider fields = %v", err)
	}

	observeCreate := newCreateFixture(t, "observe-create", "request:observe-create", "fingerprint:observe-create", "actor:local-owner", "project:default")
	observeCreate.Actions[0].Kind = application.ActionObserveAgent
	if _, _, err := createLegacyGoal(t, repository, observeCreate); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("observe action on create = %v", err)
	}

	pendingCreate := newCreateFixture(t, "pending-create", "request:pending-create", "fingerprint:pending-create", "actor:local-owner", "project:default")
	pendingSnapshot := pendingCreate.Goal.Snapshot()
	pendingSnapshot.State = goal.GoalStatePending
	pendingSnapshot.Revision--
	pendingSnapshot.StartedAt = time.Time{}
	pendingGoal, err := goal.RestoreGoal(pendingSnapshot)
	if err != nil {
		t.Fatalf("restore pending goal: %v", err)
	}
	pendingCreate.Goal = pendingGoal
	if _, _, err := createLegacyGoal(t, repository, pendingCreate); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("pending goal on create = %v", err)
	}

	state := newCreateFixture(t, "lifecycle", "request:lifecycle", "fingerprint:lifecycle", "actor:local-owner", "project:default")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create valid lifecycle fixture: %v", err)
	}
	claim := mustClaim(t, repository, "worker:lifecycle", "claim:lifecycle", state.Executions[0].CreatedAt)
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get lifecycle fixture: %v", err)
	}
	item := onlyItem(t, record.Goal)
	launchAt := state.Executions[0].CreatedAt.Add(time.Second)
	launchedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, launchAt,
	)
	if err != nil {
		t.Fatalf("start lifecycle item: %v", err)
	}
	invalidLaunch := application.LaunchAcceptedState{
		Claim: claim, Execution: record.Executions[0], OperationAt: launchAt,
		NextAction: application.ActionRecord{
			Ref: "action:observe:lifecycle", Kind: application.ActionObserveAgent,
			GoalRef: launchedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: record.Executions[0].Ref,
			AvailableAt: launchAt,
		},
		Event: application.EventRecord{
			Ref: "event:execution-accepted:lifecycle", Kind: "execution.accepted",
			GoalRef: launchedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: record.Executions[0].Ref,
			OccurredAt: launchAt,
		},
	}
	repository.now = func() time.Time { return launchAt }
	if err := repository.RecordLaunchAccepted(context.Background(), invalidLaunch); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("launch with queued execution = %v", err)
	}
	runningExecution := record.Executions[0]
	runningExecution.State = application.ExecutionRunning
	runningExecution.ProviderRef = "provider:codex"
	runningExecution.ModelRef = "model:codex"
	runningExecution.AgentRef = "agent:codex"
	runningExecution.ExternalRef = "external:test"
	runningExecution.StartedAt = launchAt
	runningExecution.DeadlineAt = launchAt.Add(time.Hour)
	runningExecution.ProviderAcceptedAt = launchAt
	directRunning := invalidLaunch
	directRunning.Execution = runningExecution
	directRunning.NextAction.PlanGeneration = runningExecution.PlanGeneration
	directRunning.NextAction.WorkItemGeneration = onlyItem(t, launchedGoal).Revision()
	if err := repository.RecordLaunchAccepted(context.Background(), directRunning); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("queued execution bypassed dispatching CAS = %v", err)
	}
	if err := repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: runningExecution, AvailableAt: launchAt.Add(time.Second), OperationAt: launchAt,
	}); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("launch action requeued as running = %v", err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: launchedGoal,
		Execution: preparedExecution, OperationAt: launchAt,
		Event: application.EventRecord{
			Ref: "event:execution-dispatching:lifecycle", Kind: "execution.dispatching",
			GoalRef: launchedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref,
			OccurredAt: launchAt,
		},
	}); err != nil {
		t.Fatalf("prepare lifecycle launch: %v", err)
	}
	queuedAgain := preparedExecution
	queuedAgain.State = application.ExecutionQueued
	if err := repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: queuedAgain, AvailableAt: launchAt.Add(time.Second), OperationAt: launchAt,
	}); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("dispatching execution regressed to queued = %v", err)
	}
}

func TestValidateExecutionRejectsIncoherentStates(t *testing.T) {
	base := newCreateFixture(t, "execution-contract", "request:execution-contract", "fingerprint:execution-contract", "actor:local-owner", "project:default").Executions[0]
	tests := map[string]application.ExecutionRecord{}

	deadline := base
	deadline.DeadlineAt = deadline.CreatedAt
	tests["deadline not strict"] = deadline

	running := base
	running.State = application.ExecutionRunning
	running.StartedAt = running.CreatedAt
	tests["running without provider"] = running

	succeeded := running
	succeeded.ProviderRef = "provider:test"
	succeeded.ExternalRef = "external:test"
	succeeded.State = application.ExecutionSucceeded
	succeeded.FinishedAt = succeeded.StartedAt
	succeeded.FailureCode = "unexpected.failure"
	tests["succeeded with failure"] = succeeded

	failed := base
	failed.State = application.ExecutionFailed
	failed.FinishedAt = failed.CreatedAt.Add(-time.Nanosecond)
	failed.FailureCode = "agent.failed"
	tests["failed before create"] = failed

	partialProvider := failed
	partialProvider.StartedAt = partialProvider.CreatedAt
	partialProvider.ProviderRef = "provider:test"
	tests["failed with partial provider identity"] = partialProvider

	for name, execution := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateExecution(execution); err == nil {
				t.Fatalf("incoherent execution accepted: %+v", execution)
			}
		})
	}
}

func TestRepositoryClaimIsAtomicRecoversExpiredLeaseAndQuarantines(t *testing.T) {
	repository, path := openTestRepository(t)
	state := newCreateFixture(t, "claim", "request:claim", "fingerprint:claim", "actor:local-owner", "project:default")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create: %v", err)
	}
	now := state.Executions[0].CreatedAt.Add(time.Second)
	repository.now = func() time.Time { return now }
	const workers = 32
	start := make(chan struct{})
	claims := make(chan application.ActionClaim, workers)
	errorsByWorker := make(chan error, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
				WorkerRef:     "worker:" + testIndex(index),
				Token:         "claim:" + testIndex(index),
				LeaseDuration: 10 * time.Second, Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
			})
			if err != nil {
				errorsByWorker <- err
				return
			}
			if found {
				claims <- claim
			}
		}()
	}
	close(start)
	wait.Wait()
	close(claims)
	close(errorsByWorker)
	for err := range errorsByWorker {
		if err != nil {
			t.Fatalf("concurrent claim: %v", err)
		}
	}
	var won []application.ActionClaim
	for claim := range claims {
		won = append(won, claim)
	}
	if len(won) != 1 || won[0].DeliveryAttempt != 1 || won[0].Fence != 1 {
		t.Fatalf("winning claims = %#v", won)
	}
	oldClaim := won[0]
	repository.now = func() time.Time { return now.Add(9 * time.Second) }
	if _, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:early", Token: "claim:early", LeaseDuration: time.Second, Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	}); err != nil || found {
		t.Fatalf("claim before expiry = found:%v err:%v", found, err)
	}
	boundary := application.ActionQuarantinedState{
		Claim: oldClaim, ErrorCode: "application.stale_boundary", OperationAt: oldClaim.LeaseUntil,
		Event: application.EventRecord{
			Ref: "event:stale-boundary", Kind: "action.quarantined", GoalRef: oldClaim.Action.GoalRef,
			WorkItemRef: oldClaim.Action.WorkItemRef, ExecutionRef: oldClaim.Action.ExecutionRef,
			OccurredAt: oldClaim.LeaseUntil,
		},
	}
	repository.now = func() time.Time { return oldClaim.LeaseUntil }
	if err := repository.QuarantineAction(context.Background(), boundary); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("claim valid at exclusive lease boundary: %v", err)
	}
	repository.now = func() time.Time { return now.Add(10 * time.Second) }
	recovered, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:recovery", Token: "claim:recovery", LeaseDuration: time.Second, Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found || recovered.DeliveryAttempt != 2 || recovered.Fence != oldClaim.Fence+1 {
		t.Fatalf("recovered claim = found:%v attempt:%d err:%v", found, recovered.DeliveryAttempt, err)
	}
	quarantineAt := recovered.LeaseUntil.Add(-time.Nanosecond)
	quarantine := application.ActionQuarantinedState{
		Claim:     recovered,
		ErrorCode: "application.action_invalid", OperationAt: quarantineAt,
		Event: application.EventRecord{
			Ref: "event:action-quarantined:claim", Kind: "action.quarantined",
			GoalRef: recovered.Action.GoalRef, WorkItemRef: recovered.Action.WorkItemRef,
			ExecutionRef: recovered.Action.ExecutionRef, OccurredAt: quarantineAt,
		},
	}
	stale := quarantine
	stale.Claim = oldClaim
	repository.now = func() time.Time { return quarantineAt }
	if err := repository.QuarantineAction(context.Background(), stale); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale claim mutation = %v", err)
	}
	if err := repository.QuarantineAction(context.Background(), quarantine); err != nil {
		t.Fatalf("quarantine: %v", err)
	}
	status, err := repository.Status(context.Background(), state.Goal.Project())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.PendingActions != 0 || status.QuarantinedActions != 1 {
		t.Fatalf("status after quarantine = %+v", status)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	repository, err = Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	status, err = repository.Status(context.Background(), state.Goal.Project())
	if err != nil || status.QuarantinedActions != 1 {
		t.Fatalf("quarantine after restart = %+v err:%v", status, err)
	}
}

func TestRepositoryMutationsAreAtomicCASAndRestoreEvidence(t *testing.T) {
	repository, path := openTestRepository(t)
	state := newCreateFixture(t, "success", "request:success", "fingerprint:success", "actor:local-owner", "project:default")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create: %v", err)
	}
	launchClaim := mustClaim(t, repository, "worker:launch", "claim:launch", state.Executions[0].CreatedAt)
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get before launch: %v", err)
	}
	item := onlyItem(t, record.Goal)
	launchAt := state.Executions[0].CreatedAt.Add(time.Second)
	launchedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, launchAt,
	)
	if err != nil {
		t.Fatalf("domain launch: %v", err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	repository.now = func() time.Time { return launchAt }
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: launchClaim, ExpectedGoalRevision: record.Goal.Revision(), Goal: launchedGoal,
		Execution: preparedExecution, OperationAt: launchAt,
		Event: application.EventRecord{
			Ref: "event:execution-dispatching:success", Kind: "execution.dispatching",
			GoalRef: launchedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref, OccurredAt: launchAt,
		},
	}); err != nil {
		t.Fatalf("prepare launch: %v", err)
	}
	launchedExecution := preparedExecution
	launchedExecution.State = application.ExecutionRunning
	launchedExecution.ProviderRef = "provider:codex"
	launchedExecution.ModelRef = "model:codex"
	launchedExecution.AgentRef = "agent:codex"
	launchedExecution.ExternalRef = "external:one"
	launchedExecution.StartedAt = launchAt
	launchedExecution.DeadlineAt = launchAt.Add(time.Hour)
	launchedExecution.ProviderAcceptedAt = launchAt.Add(-10 * time.Minute)
	observeAction := application.ActionRecord{
		Ref: "action:observe:success", Kind: application.ActionObserveAgent,
		GoalRef: launchedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: launchedExecution.Ref,
		PlanGeneration: launchedExecution.PlanGeneration, WorkItemGeneration: onlyItem(t, launchedGoal).Revision(),
		AvailableAt: launchAt,
	}
	launchState := application.LaunchAcceptedState{
		Claim: launchClaim, Execution: launchedExecution, NextAction: observeAction, OperationAt: launchAt,
		Event: application.EventRecord{
			Ref: "event:execution-accepted:success", Kind: "execution.accepted",
			GoalRef: launchedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: launchedExecution.Ref,
			OccurredAt: launchAt,
		},
	}
	if err := repository.RecordLaunchAccepted(context.Background(), launchState); err != nil {
		t.Fatalf("record launch: %v", err)
	}
	if err := repository.RecordLaunchAccepted(context.Background(), launchState); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("replayed launch = %v", err)
	}

	observeClaim := mustClaim(t, repository, "worker:observe", "claim:observe:1", launchAt)
	requeuedExecution := launchedExecution
	requeuedExecution.LastObservedAt = launchAt.Add(time.Second)
	requeuedExecution.ProviderObservedAt = launchAt.Add(-time.Hour)
	retryAt := launchAt.Add(5 * time.Second)
	repository.now = func() time.Time { return launchAt.Add(time.Second) }
	if err := repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: observeClaim, Execution: requeuedExecution, AvailableAt: retryAt, ErrorCode: "agent.pending",
		OperationAt: launchAt.Add(time.Second),
	}); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if _, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:too-early", Token: "claim:too-early", LeaseDuration: time.Second, Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	}); err != nil || found {
		t.Fatalf("early retry claim = found:%v err:%v", found, err)
	}
	successClaim := mustClaim(t, repository, "worker:observe", "claim:observe:2", retryAt)
	if successClaim.DeliveryAttempt != 2 {
		t.Fatalf("durable attempt = %d, want 2", successClaim.DeliveryAttempt)
	}
	record, err = repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get before success: %v", err)
	}
	item = onlyItem(t, record.Goal)
	succeededAt := retryAt.Add(time.Second)
	artifactDigest := strings.Repeat("a", 64)
	artifactRef := mustRef(t, "artifact:sha256:"+artifactDigest, goal.NewArtifactRef)
	attestationRef := mustRef(t, "attestation:success", goal.NewAttestationRef)
	succeededGoal, err := record.Goal.SucceedWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(),
		[]goal.ArtifactRef{artifactRef}, []goal.AttestationRef{attestationRef},
		succeededAt,
	)
	if err != nil {
		t.Fatalf("domain success item: %v", err)
	}
	succeededGoal, err = succeededGoal.Close(succeededGoal.Revision(), goal.GoalOutcomeSucceeded, succeededAt)
	if err != nil {
		t.Fatalf("domain close: %v", err)
	}
	succeededExecution := record.Executions[0]
	succeededExecution.State = application.ExecutionSucceeded
	succeededExecution.FinishedAt = succeededAt
	artifact := application.ArtifactRecord{
		OccurrenceRef: "artifact-occurrence:test-success:" + succeededExecution.Ref.String(),
		Kind:          application.ArtifactKindAgentOutput,
		Stored:        ports.StoredArtifact{Ref: artifactRef, Digest: artifactDigest, MediaType: "application/json", Size: 42},
		GoalRef:       succeededGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: succeededExecution.Ref,
		ExecutionAttempt: succeededExecution.AttemptNo, PlanGeneration: succeededExecution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AppSpecGeneration: succeededExecution.AppSpecGeneration,
		SpecHash: succeededExecution.SpecHash, CreatedAt: succeededAt,
	}
	attestation := application.AttestationRecord{
		Ref: attestationRef, Kind: application.AttestationKindArtifactProvenance,
		Verdict: application.AttestationVerdictObserved, GoalRef: succeededGoal.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: succeededExecution.Ref, ExecutionAttempt: succeededExecution.AttemptNo,
		PlanGeneration: succeededExecution.PlanGeneration, WorkItemGeneration: item.Revision(),
		AppSpecGeneration: succeededExecution.AppSpecGeneration, SpecHash: succeededExecution.SpecHash,
		ArtifactRef: artifact.Stored.Ref, SubjectDigest: strings.Repeat("b", 64),
		PolicyRef: "agent_output_present", PolicyDigest: strings.Repeat("c", 64),
		StartedAt: succeededAt, FinishedAt: succeededAt, Policy: "agent_output_present", AcceptedAt: succeededAt,
	}
	validEvents := []application.EventRecord{
		{Ref: "event:work-succeeded:success", Kind: "work_item.succeeded", GoalRef: succeededGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: succeededExecution.Ref, OccurredAt: succeededAt},
		{Ref: "event:goal-succeeded:success", Kind: "goal.succeeded", GoalRef: succeededGoal.Ref(), OccurredAt: succeededAt},
	}
	successState := application.GoalSucceededState{
		Claim: successClaim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: succeededGoal, Execution: succeededExecution, Artifact: artifact,
		Attestation: attestation, Events: validEvents, OperationAt: succeededAt,
	}
	repository.now = func() time.Time { return succeededAt }
	wrongCAS := successState
	wrongCAS.ExpectedGoalRevision++
	if err := repository.RecordGoalSucceeded(context.Background(), wrongCAS); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("wrong goal CAS = %v", err)
	}
	conflictingEvent := successState
	conflictingEvent.Events = append([]application.EventRecord(nil), validEvents...)
	conflictingEvent.Events[0].Ref = state.Events[0].Ref
	if err := repository.RecordGoalSucceeded(context.Background(), conflictingEvent); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("event conflict = %v", err)
	}
	afterRollback, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get after rollback: %v", err)
	}
	if afterRollback.Goal.State() != goal.GoalStateRunning || len(afterRollback.Artifacts) != 0 || afterRollback.Executions[0].State != application.ExecutionRunning {
		t.Fatalf("partial mutation escaped rollback: %+v", afterRollback)
	}
	if err := repository.RecordGoalSucceeded(context.Background(), successState); err != nil {
		t.Fatalf("record success: %v", err)
	}
	terminal, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get terminal: %v", err)
	}
	if terminal.Goal.State() != goal.GoalStateSucceeded || terminal.Executions[0].State != application.ExecutionSucceeded ||
		len(terminal.Artifacts) != 1 || len(terminal.Attestations) != 1 {
		t.Fatalf("terminal record = %+v", terminal)
	}
	terminalItem := onlyItem(t, terminal.Goal)
	if len(terminalItem.Artifacts()) != 1 || len(terminalItem.Attestations()) != 1 {
		t.Fatalf("evidence not restored into work item")
	}
	if !terminal.Executions[0].ProviderAcceptedAt.Equal(launchAt.Add(-10*time.Minute)) ||
		!terminal.Executions[0].ProviderObservedAt.Equal(launchAt.Add(-time.Hour)) {
		t.Fatalf("provider timestamps lost: %+v", terminal.Executions[0])
	}

	if err := repository.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	repository, err = Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	terminal, err = repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil || terminal.Goal.State() != goal.GoalStateSucceeded || len(terminal.Artifacts) != 1 {
		t.Fatalf("terminal after restart = %+v err:%v", terminal, err)
	}
	status, err := repository.Status(context.Background(), state.Goal.Project())
	if err != nil || status.RunningGoals != 0 || status.PendingActions != 0 {
		t.Fatalf("terminal status = %+v err:%v", status, err)
	}
}

func TestRepositoryRecordsFailedGoalAtomically(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newCreateFixture(t, "failed", "request:failed", "fingerprint:failed", "actor:local-owner", "project:default")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create: %v", err)
	}
	claim := mustClaim(t, repository, "worker:failed", "claim:failed", state.Executions[0].CreatedAt)
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	item := onlyItem(t, record.Goal)
	failedAt := state.Executions[0].CreatedAt.Add(time.Second)
	failedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, failedAt,
	)
	if err != nil {
		t.Fatalf("start item: %v", err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	repository.now = func() time.Time { return failedAt }
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: failedGoal,
		Execution: preparedExecution, OperationAt: failedAt,
		Event: application.EventRecord{
			Ref: "event:execution-dispatching:failed", Kind: "execution.dispatching",
			GoalRef: failedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref,
			OccurredAt: failedAt,
		},
	}); err != nil {
		t.Fatalf("prepare failed launch: %v", err)
	}
	runningItem := onlyItem(t, failedGoal)
	preparedGoalRevision := failedGoal.Revision()
	preparedItemRevision := runningItem.Revision()
	failedGoal, err = failedGoal.FailWorkItem(
		failedGoal.Revision(), runningItem.Revision(), runningItem.Ref(), failedAt,
	)
	if err != nil {
		t.Fatalf("fail item: %v", err)
	}
	failedGoal, err = failedGoal.Close(failedGoal.Revision(), goal.GoalOutcomeFailed, failedAt)
	if err != nil {
		t.Fatalf("close failed goal: %v", err)
	}
	failedExecution := preparedExecution
	failedExecution.State = application.ExecutionFailed
	failedExecution.FinishedAt = failedAt
	failedExecution.FailureCode = "agent.launch_failed"
	failure := application.GoalFailedState{
		Claim: claim, ExpectedGoalRevision: preparedGoalRevision, ExpectedItemRevision: preparedItemRevision,
		Goal: failedGoal, Execution: failedExecution,
		OperationAt: failedAt,
		Events: []application.EventRecord{
			{Ref: "event:work-failed:failed", Kind: "work_item.failed", GoalRef: failedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: failedExecution.Ref, OccurredAt: failedAt},
			{Ref: "event:goal-failed:failed", Kind: "goal.failed", GoalRef: failedGoal.Ref(), OccurredAt: failedAt},
		},
	}
	if err := repository.RecordGoalFailed(context.Background(), failure); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	terminal, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil || terminal.Goal.State() != goal.GoalStateFailed || terminal.Executions[0].FailureCode != "agent.launch_failed" {
		t.Fatalf("failed terminal = %+v err:%v", terminal, err)
	}
}

func TestRepositoryRoundTripsDAGPlanAndMultipleWorkItems(t *testing.T) {
	repository, path := openTestRepository(t)
	state := newDAGCreateFixture(t)
	created, fresh, err := createLegacyGoal(t, repository, state)
	if err != nil || !fresh {
		t.Fatalf("create DAG = fresh:%v err:%v", fresh, err)
	}
	if !reflect.DeepEqual(created.Goal.Snapshot(), state.Goal.Snapshot()) ||
		!reflect.DeepEqual(created.Executions, state.Executions) {
		t.Fatalf("DAG round trip mismatch:\n got=%+v\nwant=%+v", created, state)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	repository, err = Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8})
	if err != nil {
		t.Fatalf("reopen DAG: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	restarted, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil || !reflect.DeepEqual(restarted.Goal.Snapshot(), state.Goal.Snapshot()) {
		t.Fatalf("DAG restart mismatch: record=%+v err=%v", restarted, err)
	}
}

func TestRepositoryRoundTripsPhaseContractsLineageAndRequirementsAfterRestart(t *testing.T) {
	repository, path := openTestRepository(t)
	state := newV05CreateFixture(t)
	created, fresh, err := createLegacyGoal(t, repository, state)
	if err != nil || !fresh {
		t.Fatalf("create V05 state = fresh:%v err:%v", fresh, err)
	}
	if !reflect.DeepEqual(created.Goal.Snapshot(), state.Goal.Snapshot()) ||
		!reflect.DeepEqual(created.Executions, state.Executions) {
		t.Fatalf("V05 round trip mismatch:\n got=%+v\nwant=%+v", created, state)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close V05 state: %v", err)
	}
	repository, err = Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8})
	if err != nil {
		t.Fatalf("reopen V05 state: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	restarted, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil || !reflect.DeepEqual(restarted.Goal.Snapshot(), state.Goal.Snapshot()) ||
		!reflect.DeepEqual(restarted.Executions, state.Executions) {
		t.Fatalf("V05 restart mismatch: record=%+v err=%v", restarted, err)
	}
	parent, found := workItemByObjective(restarted.Goal, "parent")
	if !found {
		t.Fatal("restored parent not found")
	}
	children := restarted.Goal.ChildWorkItems(parent.Ref())
	if len(children) != 1 || children[0].Objective() != "overlapping child" {
		t.Fatalf("restored children = %+v", children)
	}
}

func newV05CreateFixture(t *testing.T) application.CreateGoalState {
	t.Helper()
	base := newCreateFixture(t, "v05", "request:v05", "fingerprint:v05", "actor:local-owner", "project:default")
	pending, err := goal.NewGoal(mustRef(t, "goal:v05-plan", goal.NewGoalRef), base.Goal.AppSpec(), base.Goal.CreatedAt())
	if err != nil {
		t.Fatalf("new V05 goal: %v", err)
	}
	phaseKey, _ := goal.NewPhaseKey("phase:build")
	phase, err := goal.NewPhaseInstanceWithMetadata(goal.PhaseInstanceInput{
		Ref: mustRef(t, "phase-instance:build", goal.NewPhaseRef), Key: phaseKey,
		TemplateRef:   mustRef(t, "phase-template:program", goal.NewPhaseTemplateRef),
		InputRefs:     []goal.InputRef{mustRef(t, "input:app-spec", goal.NewInputRef)},
		CriterionRefs: []goal.CriterionRef{mustRef(t, "criterion:tests-green", goal.NewCriterionRef)},
	})
	if err != nil {
		t.Fatalf("new V05 phase: %v", err)
	}
	role, _ := goal.NewRoleKey("role:worker")
	parentRef := mustRef(t, "work-item:v05:parent", goal.NewWorkItemRef)
	childRef := mustRef(t, "work-item:v05:child", goal.NewWorkItemRef)
	freeRef := mustRef(t, "work-item:v05:free", goal.NewWorkItemRef)
	shared, _ := goal.NewWriteScope("internal/shared")
	overlap, _ := goal.NewWriteScope("internal/shared/file.go")
	free, _ := goal.NewWriteScope("docs/free.md")
	parent := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: parentRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "parent", CreatedAt: pending.CreatedAt(), Phase: phaseKey, Role: role,
		WriteSet: []goal.WriteScope{shared}, CouncilPolicy: council.PolicyRequired,
		RequiredTests: sqliteRequiredTests(t, "required-test:sqlite-v05-parent"), SkillRefs: []goal.SkillRef{mustRef(t, "skill:go", goal.NewSkillRef)},
		ToolRefs:       []goal.ToolRef{mustRef(t, "tool:test", goal.NewToolRef)},
		CapabilityRefs: []goal.CapabilityRef{mustRef(t, "capability:patch", goal.NewCapabilityRef)},
		OutputContract: goal.EvidenceBundleOutputContract(),
	})
	child := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: childRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "overlapping child", CreatedAt: pending.CreatedAt(), Phase: phaseKey, Role: role,
		Parent: parentRef, WriteSet: []goal.WriteScope{overlap}, CouncilPolicy: council.PolicyRequired,
		RequiredTests: sqliteRequiredTests(t, "required-test:sqlite-v05-child"), OutputContract: goal.EvidenceBundleOutputContract(),
	})
	independent := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: freeRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "independent", CreatedAt: pending.CreatedAt(), Phase: phaseKey, Role: role,
		WriteSet: []goal.WriteScope{free}, CouncilPolicy: council.PolicyRequired,
		RequiredTests: sqliteRequiredTests(t, "required-test:sqlite-v05-free"), OutputContract: goal.EvidenceBundleOutputContract(),
	})
	// Child-before-parent is valid plan ordering: lineage is not a dependency.
	// The durable self-FK must therefore be deferred until the transaction has
	// inserted the complete immutable plan. Generation two also proves the
	// adapter round-trips the append-only result rather than assuming gen one.
	plan, err := goal.NewPlan(goal.PlanInput{Generation: 1, Phases: []goal.PhaseInstance{phase}, WorkItems: []goal.WorkItem{child, parent}})
	if err != nil {
		t.Fatalf("new V05 plan: %v", err)
	}
	aggregate, err := pending.ApplyPlan(pending.Revision(), plan)
	if err == nil {
		plan, err = goal.NewPlan(goal.PlanInput{Generation: 2, Phases: []goal.PhaseInstance{phase}, WorkItems: []goal.WorkItem{child, parent, independent}})
	}
	if err == nil {
		aggregate, err = aggregate.ApplyPlan(aggregate.Revision(), plan)
	}
	if err == nil {
		aggregate, err = aggregate.Start(aggregate.Revision(), aggregate.CreatedAt())
	}
	if err != nil {
		t.Fatalf("start V05 plan: %v", err)
	}
	var executions []application.ExecutionRecord
	var actions []application.ActionRecord
	var events []application.EventRecord
	for index, item := range aggregate.ReadyWorkItems() {
		executionRef := mustRef(t, fmt.Sprintf("execution:v05:%d", index), goal.NewExecutionRef)
		executions = append(executions, application.ExecutionRecord{
			Ref: executionRef, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), State: application.ExecutionQueued,
			AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: aggregate.PlanGeneration(),
			AppSpecGeneration: aggregate.AppSpec().Generation(), SpecHash: aggregate.SpecHash(),
			ArtifactMediaType: "text/plain", IdempotencyKey: "execution:" + executionRef.String(),
			MaxOutputBytes: 1 << 20, CreatedAt: aggregate.CreatedAt(), Purpose: application.ExecutionPurposeWork,
		})
		actions = append(actions, application.ActionRecord{
			Ref: "action:launch:" + executionRef.String(), Kind: application.ActionLaunchAgent,
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef, AvailableAt: aggregate.CreatedAt(),
			PlanGeneration: aggregate.PlanGeneration(), WorkItemGeneration: item.Revision(),
		})
		events = append(events, application.EventRecord{
			Ref: "event:execution-queued:" + executionRef.String(), Kind: "execution.queued",
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef, OccurredAt: aggregate.CreatedAt(),
		})
	}
	events = append([]application.EventRecord{{Ref: "event:goal-created:v05", Kind: "goal.created", GoalRef: aggregate.Ref(), OccurredAt: aggregate.CreatedAt()}}, events...)
	return application.CreateGoalState{
		RequestRef: "request:v05", RequestFingerprint: "fingerprint:v05", Goal: aggregate,
		Executions: executions, Actions: actions, Events: events,
	}
}

func TestRepositoryParksLegacyReadySuccessorsWithoutSyntheticGovernance(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newDAGCreateFixture(t)
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create DAG: %v", err)
	}
	launchClaim := mustClaim(t, repository, "worker:missing-successor:launch", "claim:missing-successor:launch", state.Executions[0].CreatedAt)
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get queued DAG: %v", err)
	}
	root, found := workItemByObjective(record.Goal, "root")
	if !found {
		t.Fatal("root WorkItem not found")
	}
	launchAt := state.Executions[0].CreatedAt.Add(time.Second)
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), root.Revision(), root.Ref(), record.Executions[0].Ref, launchAt,
	)
	if err != nil {
		t.Fatalf("start root: %v", err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	repository.now = func() time.Time { return launchAt }
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: launchClaim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: preparedExecution, OperationAt: launchAt,
		Event: application.EventRecord{
			Ref: "event:execution-dispatching:missing-successor", Kind: "execution.dispatching",
			GoalRef: preparedGoal.Ref(), WorkItemRef: root.Ref(), ExecutionRef: preparedExecution.Ref,
			OccurredAt: launchAt,
		},
	}); err != nil {
		t.Fatalf("prepare root: %v", err)
	}
	runningExecution := preparedExecution
	runningExecution.State = application.ExecutionRunning
	runningExecution.ProviderRef = "provider:codex"
	runningExecution.ModelRef = "model:codex"
	runningExecution.AgentRef = "agent:codex"
	runningExecution.ExternalRef = "external:missing-successor"
	runningExecution.StartedAt = launchAt
	runningExecution.DeadlineAt = launchAt.Add(time.Hour)
	runningExecution.ProviderAcceptedAt = launchAt
	observeAction := application.ActionRecord{
		Ref: "action:observe:" + runningExecution.Ref.String(), Kind: application.ActionObserveAgent,
		GoalRef: preparedGoal.Ref(), WorkItemRef: root.Ref(), ExecutionRef: runningExecution.Ref,
		PlanGeneration:     runningExecution.PlanGeneration,
		WorkItemGeneration: func() goal.Revision { updated, _ := preparedGoal.WorkItem(root.Ref()); return updated.Revision() }(),
		AvailableAt:        launchAt,
	}
	if err := repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
		Claim: launchClaim, Execution: runningExecution, NextAction: observeAction, OperationAt: launchAt,
		Event: application.EventRecord{
			Ref: "event:execution-accepted:missing-successor", Kind: "execution.accepted",
			GoalRef: preparedGoal.Ref(), WorkItemRef: root.Ref(), ExecutionRef: runningExecution.Ref,
			OccurredAt: launchAt,
		},
	}); err != nil {
		t.Fatalf("accept root: %v", err)
	}

	observeClaim := mustClaim(t, repository, "worker:missing-successor:observe", "claim:missing-successor:observe", launchAt)
	record, err = repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("get running DAG: %v", err)
	}
	runningRoot, _ := workItemByObjective(record.Goal, "root")
	finishedAt := launchAt.Add(time.Second)
	artifactDigest := strings.Repeat("d", 64)
	artifactRef := mustRef(t, "artifact:sha256:"+artifactDigest, goal.NewArtifactRef)
	attestationRef := mustRef(t, "attestation:missing-successor", goal.NewAttestationRef)
	succeededGoal, err := record.Goal.SucceedWorkItem(
		record.Goal.Revision(), runningRoot.Revision(), runningRoot.Ref(),
		[]goal.ArtifactRef{artifactRef}, []goal.AttestationRef{attestationRef}, finishedAt,
	)
	if err != nil {
		t.Fatalf("succeed root: %v", err)
	}
	succeededExecution := record.Executions[0]
	succeededExecution.State = application.ExecutionSucceeded
	succeededExecution.LastObservedAt = finishedAt
	succeededExecution.ProviderObservedAt = finishedAt
	succeededExecution.FinishedAt = finishedAt
	var successorExecutions []application.ExecutionRecord
	var successorActions []application.ActionRecord
	successorEvents := []application.EventRecord{{
		Ref: "event:work-succeeded:missing-successor", Kind: "work_item.succeeded",
		GoalRef: succeededGoal.Ref(), WorkItemRef: runningRoot.Ref(), ExecutionRef: succeededExecution.Ref,
		OccurredAt: finishedAt,
	}}
	for _, successor := range succeededGoal.ReadyWorkItems() {
		executionRef := mustRef(t, "execution:missing-successor:"+successor.Ref().String(), goal.NewExecutionRef)
		execution := application.ExecutionRecord{
			Ref: executionRef, GoalRef: succeededGoal.Ref(), WorkItemRef: successor.Ref(),
			AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: succeededGoal.PlanGeneration(),
			AppSpecGeneration: succeededGoal.AppSpec().Generation(), SpecHash: succeededGoal.SpecHash(),
			State: application.ExecutionQueued, ArtifactMediaType: "text/plain",
			IdempotencyKey: "execution:" + executionRef.String(), MaxOutputBytes: 1 << 20, CreatedAt: finishedAt,
			Purpose: application.ExecutionPurposeWork,
		}
		actionKind, actionRef := application.ActionLaunchAgent, "action:launch:"+executionRef.String()
		if len(successor.WriteSet()) != 0 {
			execution.RepositoryRef, _ = identity.NewRepositoryRef("repository:" + succeededGoal.Project().String())
			execution.ExecutionWorkspaceRef, _ = ports.NewExecutionWorkspaceRef("execution-workspace:" + executionRef.String())
			actionKind, actionRef = application.ActionPrepareWorkspace, "action:prepare-workspace:"+executionRef.String()
		}
		if len(successor.WriteSet()) != 0 && len(successor.RequiredTests()) != 0 {
			execution.Purpose = application.ExecutionPurposeAuthor
		}
		authority := application.WorkItemAuthority{}
		for _, candidate := range record.WorkItemAuthorities {
			if candidate.WorkItemRef == successor.Ref() {
				authority = candidate
				break
			}
		}
		action := legacyGovernedExecutionAction(t, succeededGoal, successor, execution, application.ActionRecord{
			Ref: actionRef, Kind: actionKind, GoalRef: succeededGoal.Ref(), WorkItemRef: successor.Ref(),
			ExecutionRef: executionRef, PlanGeneration: execution.PlanGeneration,
			WorkItemGeneration: successor.Revision(), AvailableAt: finishedAt,
		}, authority, sqliteTestBudgetPolicy(succeededGoal.CreatedAt()))
		successorExecutions = append(successorExecutions, execution)
		successorActions = append(successorActions, action)
		successorEvents = append(successorEvents, application.EventRecord{
			Ref: "event:execution-queued:" + executionRef.String(), Kind: "execution.queued",
			GoalRef: succeededGoal.Ref(), WorkItemRef: successor.Ref(), ExecutionRef: executionRef,
			OccurredAt: finishedAt,
		})
	}
	repository.now = func() time.Time { return finishedAt }
	err = repository.RecordGoalSucceeded(context.Background(), application.GoalSucceededState{
		Claim: observeClaim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: runningRoot.Revision(),
		Goal: succeededGoal, Execution: succeededExecution, OperationAt: finishedAt,
		Artifact: application.ArtifactRecord{
			OccurrenceRef: "artifact-occurrence:missing-successor:" + succeededExecution.Ref.String(),
			Kind:          application.ArtifactKindAgentOutput,
			Stored:        ports.StoredArtifact{Ref: artifactRef, Digest: artifactDigest, MediaType: "text/plain", Size: 1},
			GoalRef:       succeededGoal.Ref(), WorkItemRef: runningRoot.Ref(), ExecutionRef: succeededExecution.Ref,
			ExecutionAttempt: succeededExecution.AttemptNo, PlanGeneration: succeededExecution.PlanGeneration,
			WorkItemGeneration: runningRoot.Revision(), AppSpecGeneration: succeededExecution.AppSpecGeneration,
			SpecHash: succeededExecution.SpecHash, CreatedAt: finishedAt,
		},
		Attestation: application.AttestationRecord{
			Ref: attestationRef, Kind: application.AttestationKindArtifactProvenance,
			Verdict: application.AttestationVerdictObserved, GoalRef: succeededGoal.Ref(), WorkItemRef: runningRoot.Ref(),
			ExecutionRef: succeededExecution.Ref, ExecutionAttempt: succeededExecution.AttemptNo,
			PlanGeneration: succeededExecution.PlanGeneration, WorkItemGeneration: runningRoot.Revision(),
			AppSpecGeneration: succeededExecution.AppSpecGeneration, SpecHash: succeededExecution.SpecHash,
			ArtifactRef: artifactRef, SubjectDigest: strings.Repeat("e", 64),
			PolicyRef: "test", PolicyDigest: strings.Repeat("f", 64),
			StartedAt: finishedAt, FinishedAt: finishedAt, Policy: "test", AcceptedAt: finishedAt,
		},
		NewExecutions: successorExecutions, NewActions: successorActions, Events: successorEvents,
	})
	if err != nil {
		t.Fatalf("park legacy successor: %v cause=%v", err, errors.Unwrap(err))
	}
	after, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	afterRoot, _ := workItemByObjective(after.Goal, "root")
	if err != nil || afterRoot.State() != goal.WorkItemStateSucceeded || after.Executions[0].State != application.ExecutionSucceeded ||
		len(after.Artifacts) != 1 || tableCount(t, repository, "executions") != 3 {
		t.Fatalf("successor scheduling or parent completion failed: record=%+v err=%v", after, err)
	}
}

func newDAGCreateFixture(t *testing.T) application.CreateGoalState {
	t.Helper()
	base := newCreateFixture(t, "dag", "request:dag", "fingerprint:dag", "actor:local-owner", "project:default")
	pending, err := goal.NewGoal(mustRef(t, "goal:dag-plan", goal.NewGoalRef), base.Goal.AppSpec(), base.Goal.CreatedAt())
	if err != nil {
		t.Fatalf("new DAG goal: %v", err)
	}
	phaseBuild, _ := goal.NewPhaseKey("phase:build")
	phaseReview, _ := goal.NewPhaseKey("phase:review")
	build, _ := goal.NewPhaseInstance(phaseBuild)
	review, _ := goal.NewPhaseInstance(phaseReview)
	worker, _ := goal.NewRoleKey("role:worker")
	scopeA, _ := goal.NewWriteScope("internal/a")
	scopeB, _ := goal.NewWriteScope("internal/b")
	aRef := mustRef(t, "work-item:dag:a", goal.NewWorkItemRef)
	bRef := mustRef(t, "work-item:dag:b", goal.NewWorkItemRef)
	cRef := mustRef(t, "work-item:dag:c", goal.NewWorkItemRef)
	a := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: aRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "root", CreatedAt: pending.CreatedAt(), Phase: phaseBuild, Role: worker,
		WriteSet: []goal.WriteScope{scopeA}, CouncilPolicy: council.PolicyRequired,
		RequiredTests: sqliteRequiredTests(t, "required-test:sqlite-dag-a"), OutputContract: goal.EvidenceBundleOutputContract(),
	})
	b := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: bRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "left", CreatedAt: pending.CreatedAt(), Phase: phaseReview, Role: worker,
		Dependencies: []goal.WorkItemRef{aRef}, WriteSet: []goal.WriteScope{scopeB}, CouncilPolicy: council.PolicyRequired,
		RequiredTests:  sqliteRequiredTests(t, "required-test:sqlite-dag-b"),
		OutputContract: goal.EvidenceBundleOutputContract(),
	})
	c := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: cRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "right", CreatedAt: pending.CreatedAt(), Phase: phaseReview, Role: worker,
		Dependencies: []goal.WorkItemRef{aRef}, OutputContract: goal.EvidenceBundleOutputContract(),
	})
	plan, err := goal.NewPlan(goal.PlanInput{
		Generation: 1, Phases: []goal.PhaseInstance{build, review}, WorkItems: []goal.WorkItem{a, b, c},
	})
	if err != nil {
		t.Fatalf("new DAG plan: %v", err)
	}
	aggregate, err := pending.ApplyPlan(pending.Revision(), plan)
	if err == nil {
		aggregate, err = aggregate.Start(aggregate.Revision(), aggregate.CreatedAt())
	}
	if err != nil {
		t.Fatalf("start DAG: %v", err)
	}
	executionRef := mustRef(t, "execution:dag:a", goal.NewExecutionRef)
	execution := application.ExecutionRecord{
		Ref: executionRef, GoalRef: aggregate.Ref(), WorkItemRef: aRef, State: application.ExecutionQueued,
		AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: aggregate.PlanGeneration(),
		AppSpecGeneration: aggregate.AppSpec().Generation(), SpecHash: aggregate.SpecHash(),
		ArtifactMediaType: "text/plain", IdempotencyKey: "execution:dag:a", MaxOutputBytes: 1 << 20,
		CreatedAt: aggregate.CreatedAt(), Purpose: application.ExecutionPurposeWork,
	}
	return application.CreateGoalState{
		RequestRef: "request:dag", RequestFingerprint: "fingerprint:dag", Goal: aggregate,
		Executions: []application.ExecutionRecord{execution},
		Actions: []application.ActionRecord{{
			Ref: "action:launch:dag:a", Kind: application.ActionLaunchAgent, GoalRef: aggregate.Ref(),
			WorkItemRef: aRef, ExecutionRef: executionRef, AvailableAt: aggregate.CreatedAt(),
			PlanGeneration: aggregate.PlanGeneration(), WorkItemGeneration: a.Revision(),
		}},
		Events: []application.EventRecord{
			{Ref: "event:goal-created:dag", Kind: "goal.created", GoalRef: aggregate.Ref(), OccurredAt: aggregate.CreatedAt()},
			{Ref: "event:execution-queued:dag:a", Kind: "execution.queued", GoalRef: aggregate.Ref(), WorkItemRef: aRef, ExecutionRef: executionRef, OccurredAt: aggregate.CreatedAt()},
		},
	}
}

func openTestRepository(t *testing.T) (*Repository, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "private-state", "orquesta.sqlite")
	now := time.Date(2026, 7, 14, 12, 0, 0, 123456789, time.UTC)
	repository := openFastTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: func() time.Time { return now },
	})
	return repository, path
}

func openFastTestRepository(t *testing.T, options Options) *Repository {
	t.Helper()
	repository, err := openWithDurability(context.Background(), options, fastSQLiteTestDurability)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	return repository
}

func openFullTestRepository(t *testing.T, options Options) *Repository {
	t.Helper()
	repository, err := Open(context.Background(), options)
	if err != nil {
		t.Fatalf("open FULL repository: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	return repository
}

func newCreateFixture(
	t *testing.T,
	suffix string,
	requestRef string,
	fingerprint string,
	actorValue string,
	projectValue string,
) application.CreateGoalState {
	t.Helper()
	return newCreateFixtureWithStatement(
		t, suffix, requestRef, fingerprint, actorValue, projectValue, "build "+suffix,
	)
}

func newCreateFixtureWithStatement(
	t *testing.T,
	suffix string,
	requestRef string,
	fingerprint string,
	actorValue string,
	projectValue string,
	statement string,
) application.CreateGoalState {
	t.Helper()
	base := time.Date(2026, 7, 14, 12, 0, 0, 123456789, time.UTC)
	actor := mustRef(t, actorValue, goal.NewActorRef)
	project := mustRef(t, projectValue, goal.NewProjectRef)
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref: mustRef(t, "intent:"+suffix, goal.NewIntentRef), Actor: actor, Project: project,
		Statement: statement, SubmittedAt: base,
	})
	if err != nil {
		t.Fatalf("new intent: %v", err)
	}
	spec, err := goal.NewInitialAppSpec(goal.AppSpecInput{
		Ref: mustRef(t, "app-spec:"+suffix, goal.NewAppSpecRef), Intent: intent,
		Objective: statement, Reason: "initial_confirmation", ConfirmedBy: actor, ConfirmedAt: base,
	})
	if err != nil {
		t.Fatalf("new app spec: %v", err)
	}
	aggregate, err := goal.NewGoal(mustRef(t, "goal:"+suffix, goal.NewGoalRef), spec, base)
	if err != nil {
		t.Fatalf("new goal: %v", err)
	}
	item := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: mustRef(t, "work-item:"+suffix, goal.NewWorkItemRef), Goal: aggregate.Ref(),
		Actor: actor, Project: project, Objective: statement, CreatedAt: base,
	})
	phase, err := goal.NewPhaseInstance(goal.DefaultPhaseKey())
	if err != nil {
		t.Fatalf("new phase: %v", err)
	}
	plan, err := goal.NewPlan(goal.PlanInput{Generation: 1, Phases: []goal.PhaseInstance{phase}, WorkItems: []goal.WorkItem{item}})
	if err != nil {
		t.Fatalf("new plan: %v", err)
	}
	aggregate, err = aggregate.ApplyPlan(aggregate.Revision(), plan)
	if err != nil {
		t.Fatalf("apply plan: %v", err)
	}
	aggregate, err = aggregate.Start(aggregate.Revision(), base)
	if err != nil {
		t.Fatalf("start goal: %v", err)
	}
	executionRef := mustRef(t, "execution:"+suffix, goal.NewExecutionRef)
	execution := application.ExecutionRecord{
		Ref: executionRef, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), State: application.ExecutionQueued,
		AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: aggregate.PlanGeneration(),
		AppSpecGeneration: aggregate.AppSpec().Generation(), SpecHash: aggregate.SpecHash(),
		ArtifactMediaType: "text/plain", IdempotencyKey: "execution:" + suffix,
		MaxOutputBytes: 1 << 20, CreatedAt: base, Purpose: application.ExecutionPurposeWork,
	}
	return application.CreateGoalState{
		RequestRef: requestRef, RequestFingerprint: fingerprint, Goal: aggregate,
		Executions: []application.ExecutionRecord{execution},
		Actions: []application.ActionRecord{{
			Ref: "action:launch:" + suffix, Kind: application.ActionLaunchAgent,
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef, AvailableAt: base,
			PlanGeneration: aggregate.PlanGeneration(), WorkItemGeneration: item.Revision(),
		}},
		Events: []application.EventRecord{{
			Ref: "event:goal-created:" + suffix, Kind: "goal.created",
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef, OccurredAt: base,
		}},
	}
}

func assertRecordMatchesCreate(t *testing.T, record application.GoalRecord, state application.CreateGoalState) {
	t.Helper()
	if record.RequestRef != state.RequestRef || record.RequestFingerprint != state.RequestFingerprint {
		t.Fatalf("request identity = %q/%q", record.RequestRef, record.RequestFingerprint)
	}
	if !reflect.DeepEqual(record.Goal.Snapshot(), state.Goal.Snapshot()) ||
		!reflect.DeepEqual(record.Executions, state.Executions) {
		t.Fatalf("record round trip mismatch:\n got=%+v\nwant=%+v", record, state)
	}
}

func mustClaim(t *testing.T, repository *Repository, worker, token string, now time.Time) application.ActionClaim {
	t.Helper()
	repository.now = func() time.Time { return now }
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: worker, Token: token, LeaseDuration: 30 * time.Second, Capabilities: sqliteTestCapabilities(),
		BudgetPolicy: sqliteTestBudgetPolicy(now),
	})
	if err != nil || !found {
		t.Fatalf("claim = found:%v err:%v", found, err)
	}
	if claim.Action.Kind == application.ActionPrepareWorkspace {
		prepareLegacyWorkspaceClaim(t, repository, claim, now)
		return mustClaim(t, repository, worker, token+":launch", now)
	}
	return claim
}

func prepareLegacyWorkspaceClaim(
	t *testing.T,
	repository *Repository,
	claim application.ActionClaim,
	at time.Time,
) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := legacyExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found || !executionFound {
		t.Fatal("legacy workspace claim scope missing")
	}
	attempt := application.EffectAttempt{
		Ref:       "effect-attempt:" + claim.Action.Ref + ":" + claim.Token,
		IntentRef: claim.Action.EffectIntent.Ref, IntentDigest: claim.Action.EffectIntent.Digest,
		ApprovalRef: claim.EffectApproval.Ref, Subject: claim.Action.EffectIntent.Subject,
		ActionRef: claim.Action.Ref, ActionFence: claim.Fence, WorkerRef: claim.WorkerRef,
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey, StartedAt: at,
	}
	persistedAttempt, _, err := repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: at,
	})
	if err != nil {
		t.Fatalf("record legacy workspace attempt: %v", err)
	}
	receipt := application.EffectReceipt{
		Ref:       "effect-receipt:" + claim.Action.EffectIntent.Ref,
		IntentRef: claim.Action.EffectIntent.Ref, IntentDigest: claim.Action.EffectIntent.Digest,
		ApprovalRef: claim.EffectApproval.Ref, AttemptRef: persistedAttempt.Ref,
		Subject: claim.Action.EffectIntent.Subject, ActionRef: claim.Action.Ref, ActionFence: claim.Fence,
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey,
		ExternalRef:    "workspace-external-receipt:" + execution.ExecutionWorkspaceRef.String(),
		Status:         application.EffectStatusPrepared,
		Usage:          governance.ResourceUsage{Quality: governance.UsageQualityUnknown}, ConfirmedAt: at,
	}
	authority := application.WorkItemAuthority{}
	for _, candidate := range record.WorkItemAuthorities {
		if candidate.WorkItemRef == item.Ref() {
			authority = candidate
			break
		}
	}
	next := application.ActionRecord{
		Ref: "action:launch:" + execution.Ref.String(), Kind: application.ActionLaunchAgent,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: at,
	}
	binding := application.WorkspaceBinding{
		Ref: execution.ExecutionWorkspaceRef, PrincipalRef: authority.PrincipalRef,
		ActorRef: record.Goal.Actor(), ProjectRef: record.Goal.Project(), RepositoryRef: execution.RepositoryRef,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		WriteSet: workItemWriteSetForSQLiteTest(item), WriteSetDigest: ports.WorkspaceWriteSetDigest(workItemWriteSetForSQLiteTest(item)),
		TargetRef: "refs/heads/main", BaseOID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ObjectFormat: ports.GitObjectFormatSHA1, AdapterRef: "workspace-adapter:legacy-test",
		EffectIntentRef: claim.Action.EffectIntent.Ref, EffectAttemptRef: persistedAttempt.Ref,
		EffectFence: claim.Fence, ReceiptRef: receipt.Ref, PreparedAt: at,
	}
	if err := repository.RecordWorkspacePrepared(context.Background(), application.WorkspacePreparedState{
		Claim: claim, Execution: execution, Binding: binding, NextAction: next,
		EffectReceipt: receipt, Event: application.EventRecord{
			Ref: "event:workspace-prepared:" + execution.Ref.String(), Kind: "workspace.prepared",
			GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		}, OperationAt: at,
	}); err != nil {
		t.Fatalf("record legacy workspace preparation: %v", err)
	}
}

func workItemWriteSetForSQLiteTest(item goal.WorkItem) []string {
	values := make([]string, 0, len(item.WriteSet()))
	for _, scope := range item.WriteSet() {
		values = append(values, scope.String())
	}
	return values
}

func sqliteTestCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:codex", ModelRef: "model:codex", AgentRef: "agent:codex", Unrestricted: true,
	}
}

func onlyItem(t *testing.T, aggregate goal.Goal) goal.WorkItem {
	t.Helper()
	items := aggregate.WorkItems()
	if len(items) != 1 {
		t.Fatalf("work item count = %d, want 1", len(items))
	}
	return items[0]
}

func workItemByObjective(aggregate goal.Goal, objective string) (goal.WorkItem, bool) {
	for _, item := range aggregate.WorkItems() {
		if item.Objective() == objective {
			return item, true
		}
	}
	return goal.WorkItem{}, false
}

func mustWorkItem(t *testing.T, input goal.NewWorkItemInput) goal.WorkItem {
	t.Helper()
	if len(input.WriteSet) != 0 && input.CouncilPolicy == "" {
		t.Fatal("write-scoped SQLite test WorkItem must declare Council policy")
	}
	item, err := goal.NewWorkItem(input)
	if err != nil {
		t.Fatalf("new work item: %v", err)
	}
	return item
}

func mustRef[T any](t *testing.T, value string, constructor func(string) (T, error)) T {
	t.Helper()
	ref, err := constructor(value)
	if err != nil {
		t.Fatalf("new ref %q: %v", value, err)
	}
	return ref
}

func tableCount(t *testing.T, repository *Repository, table string) int {
	t.Helper()
	allowed := []string{"app_specs", "events", "executions", "goals", "intents", "membership_audit_receipts", "outbox"}
	index := sort.SearchStrings(allowed, table)
	if index >= len(allowed) || allowed[index] != table {
		t.Fatalf("table %q not allowed in test helper", table)
	}
	var count int
	if err := repository.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func testIndex(index int) string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	if index < len(digits) {
		return string(digits[index])
	}
	return "overflow"
}

func sqliteTestErrorChain(err error) string {
	var chain []string
	for err != nil {
		chain = append(chain, err.Error())
		err = errors.Unwrap(err)
	}
	return strings.Join(chain, " -> ")
}
