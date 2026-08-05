package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestV27EffectAttemptClaimLeaseMigrationBackfillsOnlyExactClaim(t *testing.T) {
	system, _, attempt := seedV27AmbiguousEffectAttempt(t, "derivable")
	path := cloneV27FixtureToV26(t, system, nil)
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("migrate exact claim: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })

	var lease sql.NullInt64
	sqliteTestNoError(t, repository.db.QueryRow(`
SELECT claim_lease_until FROM effect_attempts WHERE ref=?`, attempt.Ref).Scan(&lease))
	if !lease.Valid || lease.Int64 != attempt.ClaimLeaseUntil.UnixNano() {
		t.Fatalf("backfilled lease=%+v want=%d", lease, attempt.ClaimLeaseUntil.UnixNano())
	}
}

func TestV27EffectAttemptClaimLeaseMigrationKeepsResolvedLegacyNull(t *testing.T) {
	t.Run("causal receipt", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		system.submit(t, "request:v27-receipt")
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v27-receipt")
		if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
			t.Fatalf("receipt launch result=%+v err=%v", result, err)
		}
		attemptRef, actionRef := onlyV27AttemptRefs(t, system.repository.db)
		path := cloneV27FixtureToV26(t, system, func(database *sql.DB) {
			reclaimV27FixtureAction(t, database, actionRef)
		})
		repository, openErr := Open(context.Background(), Options{
			Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		})
		if openErr != nil {
			t.Fatalf("migrate receipt proof: %s", sqliteTestErrorChain(openErr))
		}
		t.Cleanup(func() { _ = repository.Close() })
		assertV27AttemptLeaseNull(t, repository.db, attemptRef)
	})

	t.Run("one exact zero release", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		system.submit(t, "request:v27-zero-release")
		system.external.launchErr = sqliteV15DefinitelyUnapplied{}
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v27-zero-release")
		if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
			t.Fatalf("zero-release launch result=%+v err=%v", result, err)
		}
		attemptRef, _ := onlyV27AttemptRefs(t, system.repository.db)
		path := cloneV27FixtureToV26(t, system, nil)
		repository, openErr := Open(context.Background(), Options{
			Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		})
		if openErr != nil {
			t.Fatalf("migrate zero-release proof: %s", sqliteTestErrorChain(openErr))
		}
		t.Cleanup(func() { _ = repository.Close() })
		assertV27AttemptLeaseNull(t, repository.db, attemptRef)
	})
}

func TestV27EffectAttemptClaimLeaseMigrationRejectsAmbiguityAtomically(t *testing.T) {
	system, _, attempt := seedV27AmbiguousEffectAttempt(t, "rollback")
	path := cloneV27FixtureToV26(t, system, func(database *sql.DB) {
		reclaimV27FixtureAction(t, database, attempt.ActionRef)
	})
	_, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err == nil || !strings.Contains(sqliteTestErrorChain(err), "sqlite.v27_effect_attempt_claim_lease_ambiguous") {
		t.Fatalf("ambiguous V26 attempt migrated: %s", sqliteTestErrorChain(err))
	}

	database := openFastV18MigrationFixture(t, path)
	defer database.Close()
	var version, receipt, leaseColumn, immutableTrigger int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM schema_migrations WHERE version=27`).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('effect_attempts') WHERE name='claim_lease_until'`).Scan(&leaseColumn))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema WHERE type='trigger' AND name='effect_attempts_immutable_update'`).Scan(&immutableTrigger))
	if version != recoverySchemaV38EnvironmentGate || receipt != 0 || leaseColumn != 0 || immutableTrigger != 1 {
		t.Fatalf("migration was not atomic: version=%d receipt=%d column=%d immutable=%d",
			version, receipt, leaseColumn, immutableTrigger)
	}
}

func TestV27EffectAttemptNewInsertAndReplayPreserveExactLease(t *testing.T) {
	system, claim, attempt := seedV27AmbiguousEffectAttempt(t, "writer")
	stored, found, err := readEffectAttemptByFence(
		context.Background(), system.repository.db, attempt.ActionRef, attempt.ActionFence,
	)
	if err != nil || !found || stored != attempt {
		t.Fatalf("read attempt=%+v found=%t err=%v want=%+v", stored, found, err, attempt)
	}

	replayed, created, err := system.repository.RecordEffectAttempt(
		context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: attempt.StartedAt,
		},
	)
	if err != nil || created || replayed != attempt {
		t.Fatalf("exact replay=%+v created=%t err=%v", replayed, created, err)
	}

	mutated := attempt
	mutated.ClaimLeaseUntil = mutated.ClaimLeaseUntil.Add(time.Nanosecond)
	if _, _, err := system.repository.RecordEffectAttempt(
		context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: mutated, OperationAt: mutated.StartedAt,
		},
	); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("mutated lease accepted: %v", err)
	}
}

func seedV27AmbiguousEffectAttempt(t *testing.T, suffix string) (*sqliteV15System, application.ActionClaim, application.EffectAttempt) {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:v27-"+suffix)
	claim := claimSQLiteV15(t, system, "claim:v27-"+suffix)
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	attempt.ClaimLeaseUntil = claim.LeaseUntil.UTC()
	persisted, created, err := system.repository.RecordEffectAttempt(
		context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: attempt.StartedAt,
		},
	)
	if err != nil || !created || persisted != attempt {
		t.Fatalf("seed attempt=%+v created=%t err=%v", persisted, created, err)
	}
	return system, claim, attempt
}

func onlyV27AttemptRefs(t *testing.T, database *sql.DB) (string, string) {
	t.Helper()
	var attemptRef, actionRef string
	sqliteTestNoError(t, database.QueryRow(`SELECT ref,action_ref FROM effect_attempts`).Scan(&attemptRef, &actionRef))
	return attemptRef, actionRef
}

func assertV27AttemptLeaseNull(t *testing.T, database *sql.DB, attemptRef string) {
	t.Helper()
	var lease sql.NullInt64
	sqliteTestNoError(t, database.QueryRow(`
SELECT claim_lease_until FROM effect_attempts WHERE ref=?`, attemptRef).Scan(&lease))
	if lease.Valid {
		t.Fatalf("resolved historical attempt acquired lease=%d", lease.Int64)
	}
}

func reclaimV27FixtureAction(t *testing.T, database *sql.DB, actionRef string) {
	t.Helper()
	result, err := database.Exec(`
UPDATE outbox
SET claim_token=claim_token || ':later', claimed_by='worker:v27-later',
    claimed_until=claimed_until+60000000000, delivery_attempt=delivery_attempt+1, fence=fence+1
WHERE ref=?`, actionRef)
	sqliteTestNoError(t, err)
	changed, err := result.RowsAffected()
	sqliteTestNoError(t, err)
	if changed != 1 {
		t.Fatalf("reclaim fixture rows=%d", changed)
	}
}

func cloneV27FixtureToV26(
	t *testing.T,
	system *sqliteV15System,
	mutate func(*sql.DB),
) string {
	t.Helper()
	sourcePath := system.path
	sqliteTestNoError(t, system.repository.Close())
	directory := t.TempDir()
	sqliteTestNoError(t, os.Chmod(directory, 0o700))
	targetPath := filepath.Join(directory, "effect-attempt-v26.db")
	database := openFastV18MigrationFixture(t, targetPath)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(
		context.Background(), database, migrations[:recoverySchemaV38EnvironmentGate],
	))
	_, err = database.Exec(`ATTACH DATABASE ? AS source`, sourcePath)
	sqliteTestNoError(t, err)
	_, err = database.Exec(`PRAGMA foreign_keys=OFF`)
	sqliteTestNoError(t, err)

	type trigger struct{ name, statement string }
	rows, err := database.Query(`SELECT name,sql FROM main.sqlite_schema WHERE type='trigger' ORDER BY name`)
	sqliteTestNoError(t, err)
	var triggers []trigger
	for rows.Next() {
		var value trigger
		sqliteTestNoError(t, rows.Scan(&value.name, &value.statement))
		triggers = append(triggers, value)
	}
	sqliteTestNoError(t, rows.Close())
	for _, value := range triggers {
		_, err = database.Exec(`DROP TRIGGER "` + strings.ReplaceAll(value.name, `"`, `""`) + `"`)
		sqliteTestNoError(t, err)
	}

	rows, err = database.Query(`SELECT name FROM main.sqlite_schema
WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name<>'schema_migrations' ORDER BY name`)
	sqliteTestNoError(t, err)
	var tables []string
	for rows.Next() {
		var table string
		sqliteTestNoError(t, rows.Scan(&table))
		tables = append(tables, table)
	}
	sqliteTestNoError(t, rows.Close())
	for _, table := range tables {
		copySharedV27Table(t, database, table)
	}
	if mutate != nil {
		mutate(database)
	}
	for _, value := range triggers {
		_, err = database.Exec(value.statement)
		sqliteTestNoError(t, err)
	}
	_, err = database.Exec(`DETACH DATABASE source`)
	sqliteTestNoError(t, err)
	var violations int
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations))
	if violations != 0 {
		t.Fatalf("V26 fixture foreign-key violations=%d", violations)
	}
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(targetPath, 0o600))
	return targetPath
}

func copySharedV27Table(t *testing.T, database *sql.DB, table string) {
	t.Helper()
	target := v27FixtureColumns(t, database, "main", table)
	sourceSet := make(map[string]bool)
	for _, column := range v27FixtureColumns(t, database, "source", table) {
		sourceSet[column] = true
	}
	var shared []string
	for _, column := range target {
		if sourceSet[column] {
			shared = append(shared, column)
		}
	}
	if len(shared) == 0 {
		return
	}
	quoted := make([]string, len(shared))
	for index, column := range shared {
		quoted[index] = `"` + strings.ReplaceAll(column, `"`, `""`) + `"`
	}
	statement := fmt.Sprintf(`INSERT INTO main."%s"(%s) SELECT %s FROM source."%s"`,
		strings.ReplaceAll(table, `"`, `""`), strings.Join(quoted, ","), strings.Join(quoted, ","),
		strings.ReplaceAll(table, `"`, `""`))
	_, err := database.Exec(statement)
	if err != nil {
		t.Fatalf("copy V27 table %s: %v", table, err)
	}
}

func v27FixtureColumns(t *testing.T, database *sql.DB, schema, table string) []string {
	t.Helper()
	rows, err := database.Query(fmt.Sprintf(`PRAGMA %s.table_info("%s")`, schema,
		strings.ReplaceAll(table, `"`, `""`)))
	sqliteTestNoError(t, err)
	defer rows.Close()
	var result []string
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		sqliteTestNoError(t, rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey))
		result = append(result, name)
	}
	return result
}
