package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	var version, receipt, violations int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND name='014_council.sql'`, recoverySchemaV19).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations))
	if version != recoverySchemaV19 || receipt != 1 || violations != 0 {
		t.Fatalf("V19 migration version=%d receipt=%d foreign_keys=%d", version, receipt, violations)
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
