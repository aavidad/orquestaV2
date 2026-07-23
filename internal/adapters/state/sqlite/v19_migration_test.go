package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestV19MigrationEmptyV18DatabaseCreatesCouncilFoundation(t *testing.T) {
	path := emptySQLiteV18Database(t)
	repository, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	if err != nil {
		t.Fatalf("empty V18 migration: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	assertSQLiteV19MigrationHealthy(t, repository.db)
	for _, table := range []string{"council_rounds", "council_facts", "council_decisions", "council_skips"} {
		var strict int
		sqliteTestNoError(t, repository.db.QueryRow(`SELECT strict FROM pragma_table_list WHERE name=?`, table).Scan(&strict))
		if strict != 1 {
			t.Fatalf("%s is not STRICT", table)
		}
	}
	for table, column := range map[string]string{
		"work_items": "council_policy", "executions": "council_subject_digest", "outbox": "council_resolution_kind",
	} {
		var count int
		sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&count))
		if count != 1 {
			t.Fatalf("%s.%s missing", table, column)
		}
	}
}

func TestV19MigrationRejectsEveryLiveWriteScopedV18FrontierAtomically(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, *sql.DB)
	}{
		{name: "pending_item", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE work_items SET state='pending',started_at=NULL,finished_at=NULL,execution_ref=NULL WHERE ref='work-item:v19-closed'`)
		}},
		{name: "running_item", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE work_items SET state='running',finished_at=NULL WHERE ref='work-item:v19-closed'`)
		}},
		{name: "pending_action", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE outbox SET completed_at=NULL,claim_token=NULL,claimed_by=NULL,claimed_until=NULL,delivery_attempt=0,fence=0 WHERE ref='action:v19-closed'`)
		}},
		{name: "claimed_action", mutate: func(t *testing.T, database *sql.DB) {
			mustV19Exec(t, database, `UPDATE outbox SET completed_at=NULL WHERE ref='action:v19-closed'`)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := populatedTerminalSQLiteV18Database(t)
			database, err := sql.Open(driverName, path)
			sqliteTestNoError(t, err)
			mustV19Exec(t, database, `INSERT INTO work_item_write_scopes(goal_ref,work_item_ref,scope,position) VALUES ('goal:v19-closed','work-item:v19-closed','internal/v19',0)`)
			test.mutate(t, database)
			sqliteTestNoError(t, database.Close())

			_, err = Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2})
			if err == nil || !strings.Contains(sqliteTestErrorChain(err), "sqlite.v19_upgrade_requires_v18_council_policy") {
				t.Fatalf("live V18 frontier migrated: %s", sqliteTestErrorChain(err))
			}
			assertV19MigrationNeverStarted(t, path)
		})
	}
}

func TestV19MigrationPreservesTerminalHistoryChecksumAndReopensIdempotently(t *testing.T) {
	path := populatedTerminalSQLiteV18Database(t)
	before := snapshotV18MigrationHistoryPath(t, path)
	repository, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	if err != nil {
		t.Fatalf("terminal V18 migration: %s", sqliteTestErrorChain(err))
	}
	assertSQLiteV19MigrationHealthy(t, repository.db)
	after := snapshotV18MigrationHistory(t, repository.db, before)
	assertV18MigrationHistoryEqual(t, before, after)
	var policy string
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT council_policy FROM work_items WHERE ref='work-item:v19-closed'`).Scan(&policy))
	if policy != "" {
		t.Fatalf("terminal history backfilled policy=%q", policy)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("V19 recovery validation: %v", err)
	}
	rewriteRecoveryTrigger(t, repository.db, "work_items_council_policy_immutable", func() {
		mustV19Exec(t, repository.db, `UPDATE work_items SET council_policy='required'
WHERE ref='work-item:v19-closed'`)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("V19 read-only policy recovery: %v", err)
	}
	sqliteTestNoError(t, repository.Close())

	reopened, err := Open(context.Background(), Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	if err != nil {
		t.Fatalf("reopen V19: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = reopened.Close() })
	assertSQLiteV19MigrationHealthy(t, reopened.db)
	var receipts int
	sqliteTestNoError(t, reopened.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV19).Scan(&receipts))
	if receipts != 1 {
		t.Fatalf("V19 migration receipts=%d", receipts)
	}
}

func TestV19MigrationRejectsPartialHistoricalInterruptPairAtomically(t *testing.T) {
	for _, test := range []struct {
		name   string
		update string
	}{
		{
			name: "cause_without_time",
			update: `UPDATE work_items SET state='superseded',revision=4,control_sequence=1,
interrupt_cause='execution_failed',interrupted_at=NULL WHERE ref='work-item:v19-closed'`,
		},
		{
			name: "time_without_cause",
			update: `UPDATE work_items SET state='superseded',revision=4,control_sequence=1,
interrupt_cause='',interrupted_at=finished_at WHERE ref='work-item:v19-closed'`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := populatedTerminalSQLiteV18Database(t)
			database, err := sql.Open(driverName, path)
			sqliteTestNoError(t, err)
			mustV19Exec(t, database, test.update)
			sqliteTestNoError(t, database.Close())

			_, err = Open(context.Background(), Options{
				Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
			})
			if err == nil || !strings.Contains(sqliteTestErrorChain(err), "constraint failed") {
				t.Fatalf("partial V18 interrupt pair migrated: %s", sqliteTestErrorChain(err))
			}
			assertV19MigrationNeverStarted(t, path)
		})
	}
}

func TestV19MigrationPreservesValidInterruptedPair(t *testing.T) {
	path := populatedTerminalSQLiteV18Database(t)
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	rewriteRecoveryTrigger(t, database, "artifact_occurrences_immutable_delete", func() {
		rewriteRecoveryTrigger(t, database, "attestations_immutable_delete", func() {
			rewriteRecoveryTrigger(t, database, "artifacts_immutable_delete", func() {
				mustV19Exec(t, database, `DELETE FROM attestation_test_outcomes`)
				mustV19Exec(t, database, `DELETE FROM artifact_occurrences`)
				mustV19Exec(t, database, `DELETE FROM attestations`)
				mustV19Exec(t, database, `DELETE FROM artifacts`)
			})
		})
	})
	mustV19Exec(t, database, `UPDATE goals SET state='running',revision=6,control_sequence=1,closed_at=NULL
WHERE ref='goal:v19-closed'`)
	mustV19Exec(t, database, `UPDATE work_items SET state='interrupted',revision=4,control_sequence=1,
interrupt_cause='execution_failed',interrupted_at=finished_at,finished_at=NULL
WHERE ref='work-item:v19-closed'`)
	mustV19Exec(t, database, `UPDATE executions SET state='failed',failure_code='application.execution_failed'
WHERE ref='execution:v19-closed'`)
	sqliteTestNoError(t, database.Close())

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
	})
	if err != nil {
		t.Fatalf("valid interrupted V18 migration: %s", sqliteTestErrorChain(err))
	}
	assertSQLiteV19MigrationHealthy(t, repository.db)
	var state, cause string
	var interruptedAt int64
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT state,interrupt_cause,interrupted_at
FROM work_items WHERE ref='work-item:v19-closed'`).Scan(&state, &cause, &interruptedAt))
	if state != "interrupted" || cause != "execution_failed" || interruptedAt == 0 {
		t.Fatalf("valid interrupted history=%q/%q/%d", state, cause, interruptedAt)
	}
	sqliteTestNoError(t, repository.Close())

	reopened, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
	})
	if err != nil {
		t.Fatalf("valid interrupted V19 reopen: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = reopened.Close() })
	assertSQLiteV19MigrationHealthy(t, reopened.db)
}

func TestV19MigrationPreservesSupersededTerminalWriterWithoutPolicy(t *testing.T) {
	path := populatedTerminalSQLiteV18Database(t)
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	rewriteRecoveryTrigger(t, database, "artifact_occurrences_immutable_delete", func() {
		rewriteRecoveryTrigger(t, database, "attestations_immutable_delete", func() {
			rewriteRecoveryTrigger(t, database, "artifacts_immutable_delete", func() {
				mustV19Exec(t, database, `DELETE FROM attestation_test_outcomes`)
				mustV19Exec(t, database, `DELETE FROM artifact_occurrences`)
				mustV19Exec(t, database, `DELETE FROM attestations`)
				mustV19Exec(t, database, `DELETE FROM artifacts`)
			})
		})
	})
	mustV19Exec(t, database, `UPDATE goals SET state='failed',revision=9,control_sequence=1,plan_generation=2
WHERE ref='goal:v19-closed'`)
	mustV19Exec(t, database, `UPDATE work_items SET state='superseded',revision=4,control_sequence=1
WHERE ref='work-item:v19-closed'`)
	mustV19Exec(t, database, `UPDATE executions SET state='failed',failure_code='application.execution_superseded'
WHERE ref='execution:v19-closed'`)
	mustV19Exec(t, database, `INSERT INTO work_items(
 ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,
 skip_reason,interrupt_cause,rework_of,state,revision,paused,cancel_requested,control_sequence,
 position,created_at,started_at,interrupted_at,finished_at,execution_ref,handoff_required,
 governance_version,budget_demand_ref,budget_tokens,budget_money_micros,budget_currency,
 budget_active_time_ns,budget_process_slots,budget_disk_bytes,security_criticality,reasoning_effort)
SELECT 'work-item:v19-successor',goal_ref,actor_ref,project_ref,'terminal rework successor',
 phase_key,role_key,NULL,output_contract,'','',ref,'failed',3,0,0,0,1,created_at,started_at,
 NULL,finished_at,'execution:v19-successor',0,governance_version,budget_demand_ref,budget_tokens,
 budget_money_micros,budget_currency,budget_active_time_ns,budget_process_slots,budget_disk_bytes,
 security_criticality,reasoning_effort
FROM work_items WHERE ref='work-item:v19-closed'`)
	mustV19Exec(t, database, `INSERT INTO executions(
 ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,
 plan_generation,app_spec_generation,spec_hash,repository_ref,execution_workspace_ref,state,
 purpose,review_subject_digest,artifact_media_type,idempotency_key,max_output_bytes,provider_ref,
 model_ref,agent_ref,external_ref,governance_version,budget_reservation_ref,effect_intent_ref,
 launch_receipt_ref,created_at,deadline_at,started_at,provider_accepted_at,last_observed_at,
 provider_observed_at,finished_at,failure_code,recipient_mailbox_retired)
SELECT 'execution:v19-successor',goal_ref,'work-item:v19-successor',1,max_execution_attempts,NULL,
 2,app_spec_generation,spec_hash,'','','failed','work','',artifact_media_type,
 'idempotency:v19-successor',max_output_bytes,provider_ref,model_ref,agent_ref,
 'external:v19-successor',0,NULL,NULL,NULL,created_at,deadline_at,started_at,
 provider_accepted_at,last_observed_at,provider_observed_at,finished_at,
 'application.execution_failed',0
FROM executions WHERE ref='execution:v19-closed'`)
	mustV19Exec(t, database, `INSERT INTO work_item_write_scopes(goal_ref,work_item_ref,scope,position)
VALUES ('goal:v19-closed','work-item:v19-closed','internal/v19',0)`)
	sqliteTestNoError(t, database.Close())

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("superseded terminal V18 migration: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	record, err := repository.GetGoal(context.Background(), mustRef(t, "goal:v19-closed", goal.NewGoalRef))
	sqliteTestNoError(t, err)
	source, found := record.Goal.WorkItem(mustRef(t, "work-item:v19-closed", goal.NewWorkItemRef))
	successor, successorFound := record.Goal.WorkItem(mustRef(t, "work-item:v19-successor", goal.NewWorkItemRef))
	reworkOf, linked := successor.ReworkOf()
	_, policyFound := source.CouncilPolicy()
	if !found || !successorFound || source.State() != goal.WorkItemStateSuperseded ||
		successor.State() != goal.WorkItemStateFailed || !linked || reworkOf != source.Ref() || policyFound {
		t.Fatalf("superseded migration source=%+v successor=%+v linked=%t policy=%t",
			source, successor, linked, policyFound)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("superseded terminal recovery: %s", sqliteTestErrorChain(err))
	}
}

func emptySQLiteV18Database(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	sqliteTestNoError(t, os.Chmod(directory, 0o700))
	path := filepath.Join(directory, "v18-empty.db")
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(context.Background(), database, migrations[:recoverySchemaV18]))
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	return path
}

func populatedTerminalSQLiteV18Database(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	sqliteTestNoError(t, os.Chmod(directory, 0o700))
	path := filepath.Join(directory, "v18-terminal.db")
	sqliteTestNoError(t, preparePrivateDatabase(path))
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	sqliteTestNoError(t, err)
	database.SetMaxOpenConns(1)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	tx, err := database.Begin()
	sqliteTestNoError(t, err)
	if _, err = tx.Exec(migrations[0].sql); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO schema_migrations(version,name,checksum) VALUES (1,?,?)`, migrations[0].name, migrations[0].checksum); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`PRAGMA user_version=1`); err != nil {
		t.Fatal(err)
	}
	seedV1Goal(t, tx, "v19-closed", "succeeded", 6, "succeeded", 3, "succeeded", time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC), true)
	sqliteTestNoError(t, tx.Commit())
	sqliteTestNoError(t, migrateSQLiteV19Prefix(t, database, migrations, 1, recoverySchemaV18))
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	return path
}

func migrateSQLiteV19Prefix(t *testing.T, database *sql.DB, migrations []migration, current, target int) error {
	t.Helper()
	connection, err := database.Conn(context.Background())
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err = connection.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	tx, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, _, err = applyMigrationSteps(context.Background(), tx, migrations[:target], current); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	_, err = connection.ExecContext(context.Background(), `PRAGMA foreign_keys=ON`)
	return err
}

func assertSQLiteV19MigrationHealthy(t *testing.T, database *sql.DB) {
	t.Helper()
	var version, receipt, violations, directorDecisionUniqueIndex int
	var workItemIndexes, workItemTriggers, workItemForeignKeys, staleWorkItemReferences int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND name='014_council.sql'`, recoverySchemaV19).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_index_list('director_decisions')
WHERE name='director_decisions_goal_idx' AND "unique"=1`).Scan(&directorDecisionUniqueIndex))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema WHERE type='index'
AND name IN ('work_items_parent_idx','work_items_mailbox_lineage_idx','work_items_rework_idx')`).Scan(&workItemIndexes))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema WHERE type='trigger'
AND name IN ('work_items_handoff_required_immutable','work_items_governance_insert_guard',
'work_items_governance_update_guard','work_items_council_policy_immutable')`).Scan(&workItemTriggers))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_list('work_items')
WHERE "table" IN ('goals','goal_phases','work_items')`).Scan(&workItemForeignKeys))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema
WHERE sql IS NOT NULL AND sql LIKE '%work_items_v19%'`).Scan(&staleWorkItemReferences))
	if version != recoverySchemaV19 || receipt != 1 || violations != 0 || directorDecisionUniqueIndex != 1 ||
		workItemIndexes != 3 || workItemTriggers != 4 || workItemForeignKeys != 7 || staleWorkItemReferences != 0 {
		t.Fatalf("V19 migration version=%d receipt=%d foreign_keys=%d director_decisions_unique_index=%d work_item_indexes=%d triggers=%d work_item_fks=%d stale_refs=%d",
			version, receipt, violations, directorDecisionUniqueIndex, workItemIndexes, workItemTriggers,
			workItemForeignKeys, staleWorkItemReferences)
	}
}

func assertV19MigrationNeverStarted(t *testing.T, path string) {
	t.Helper()
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	defer database.Close()
	var version, receipt, column int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV19).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('work_items') WHERE name='council_policy'`).Scan(&column))
	if version != recoverySchemaV18 || receipt != 0 || column != 0 {
		t.Fatalf("rejected migration mutated version=%d receipt=%d column=%d", version, receipt, column)
	}
}

func mustV19Exec(t *testing.T, database *sql.DB, statement string) {
	t.Helper()
	if _, err := database.Exec(statement); err != nil {
		t.Fatal(err)
	}
}
