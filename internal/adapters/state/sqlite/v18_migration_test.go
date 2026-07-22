package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/application"
)

func TestV18MigrationEmptyV17DatabaseSucceeds(t *testing.T) {
	path := emptySQLiteV17Database(t)
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("empty V17 migration: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	assertSQLiteV18MigrationHealthy(t, repository)
}

func TestV18MigrationBlocksEveryUndrainedLegacyIntegrationFrontierWithoutMutation(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, *sql.DB)
	}{
		{name: "pending"},
		{name: "claimed_before_vcs", mutate: claimLegacyIntegration},
		{name: "post_vcs_before_receipt", mutate: func(t *testing.T, database *sql.DB) {
			claimLegacyIntegration(t, database)
			mustV10Exec(t, database, `
INSERT INTO effect_attempts(ref,intent_ref,intent_digest,approval_ref,project_ref,goal_ref,
 work_item_ref,execution_ref,plan_generation,app_spec_generation,spec_hash,actor_ref,
 action_ref,action_fence,worker_ref,idempotency_key,started_at)
SELECT 'effect-attempt:legacy-post-vcs',intent.ref,intent.digest,approval.ref,intent.project_ref,
 intent.goal_ref,intent.work_item_ref,intent.execution_ref,intent.plan_generation,
 intent.app_spec_generation,intent.spec_hash,intent.actor_ref,action.ref,action.fence,
 action.claimed_by,'effect-attempt:legacy-post-vcs',action.available_at
FROM outbox action JOIN effect_intents intent ON intent.ref=action.effect_intent_ref
JOIN effect_approvals approval ON approval.intent_ref=intent.ref
WHERE action.kind='integrate_change' AND action.completed_at IS NULL`)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := seedSQLiteV18Integration(t, false)
			path := copyCurrentIntegrationFixtureToV17(t, source)
			if test.mutate != nil {
				database, err := sql.Open(driverName, path)
				sqliteTestNoError(t, err)
				test.mutate(t, database)
				sqliteTestNoError(t, database.Close())
			}
			_, err := Open(context.Background(), Options{
				Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
			})
			if err == nil || !strings.Contains(sqliteTestErrorChain(err), "sqlite.v18_upgrade_requires_v17_integration_drain") {
				t.Fatalf("undrained V17 frontier migrated: %s", sqliteTestErrorChain(err))
			}
			database, openErr := sql.Open(driverName, path)
			sqliteTestNoError(t, openErr)
			defer database.Close()
			var version, receipt int
			sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
			sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=13`).Scan(&receipt))
			if version != recoverySchemaV17 || receipt != 0 {
				t.Fatalf("blocked migration mutated schema version=%d receipt=%d", version, receipt)
			}
		})
	}
}

func TestV18MigrationPreservesCompletedV17IntegrationAndForeignKeys(t *testing.T) {
	source := seedSQLiteV18Integration(t, true)
	path := copyCurrentIntegrationFixtureToV17(t, source)
	before := snapshotV18MigrationHistoryPath(t, path)
	assertPopulatedV18MigrationHistory(t, before)
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("completed V17 migration: %s", sqliteTestErrorChain(err))
	}
	assertSQLiteV18MigrationHealthy(t, repository)
	assertV18MigrationHistoryEqual(t, before, snapshotV18MigrationHistory(t, repository.db, before))
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("completed V17 integration recovery: %v", err)
	}
	sqliteTestNoError(t, repository.Close())

	repository, err = Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("reopen completed V18 migration: %s", sqliteTestErrorChain(err))
	}
	t.Cleanup(func() { _ = repository.Close() })
	assertSQLiteV18MigrationHealthy(t, repository)
	assertV18MigrationHistoryEqual(t, before, snapshotV18MigrationHistory(t, repository.db, before))
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("reopened completed V18 integration recovery: %v", err)
	}
}

func seedSQLiteV18Integration(t *testing.T, complete bool) *sqliteV15System {
	t.Helper()
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest,
		application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent,
	)
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	result, err := system.orchestrator.IntegrateChange(context.Background(), system.access,
		application.IntegrateChangeRequest{
			RequestRef: "request:v18-migration-integration", GoalRef: goalRef,
			ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
		})
	if err != nil || !result.Created {
		t.Fatalf("integration admission result=%+v err=%v", result, err)
	}
	if complete {
		processSQLiteV16Actions(t, system, application.ActionIntegrateChange)
	}
	return system
}

func claimLegacyIntegration(t *testing.T, database *sql.DB) {
	t.Helper()
	mustV10Exec(t, database, `
UPDATE outbox SET claim_token='claim:legacy-integration',claimed_by='worker:legacy',
 claimed_until=available_at+1000000000,delivery_attempt=1,fence=1
WHERE kind='integrate_change' AND completed_at IS NULL`)
}

func emptySQLiteV17Database(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	sqliteTestNoError(t, os.Chmod(directory, 0o700))
	path := filepath.Join(directory, "v17-empty.db")
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(context.Background(), database,
		migrations[:recoverySchemaV17]))
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))
	return path
}

func copyCurrentIntegrationFixtureToV17(t *testing.T, system *sqliteV15System) string {
	t.Helper()
	sourcePath := system.path
	sqliteTestNoError(t, system.repository.Close())
	targetPath := emptySQLiteV17Database(t)
	database, err := sql.Open(driverName, targetPath)
	sqliteTestNoError(t, err)
	defer database.Close()
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
		copySharedV17Table(t, database, table)
	}
	for _, value := range triggers {
		_, err = database.Exec(value.statement)
		sqliteTestNoError(t, err)
	}
	_, err = database.Exec(`DETACH DATABASE source`)
	sqliteTestNoError(t, err)
	var violations int
	rows, err = database.Query(`PRAGMA foreign_key_check`)
	sqliteTestNoError(t, err)
	for rows.Next() {
		violations++
	}
	sqliteTestNoError(t, rows.Close())
	if violations != 0 {
		t.Fatalf("V17 fixture foreign key violations=%d", violations)
	}
	return targetPath
}

func copySharedV17Table(t *testing.T, database *sql.DB, table string) {
	t.Helper()
	targetColumns := sqliteFixtureColumns(t, database, "main", table)
	sourceColumns := sqliteFixtureColumns(t, database, "source", table)
	shared := make([]string, 0, len(targetColumns))
	for column := range targetColumns {
		if sourceColumns[column] {
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
	filter := v17FixtureFilter(table)
	statement := fmt.Sprintf(`INSERT INTO main."%s"(%s) SELECT %s FROM source."%s" %s`,
		strings.ReplaceAll(table, `"`, `""`), strings.Join(quoted, ","), strings.Join(quoted, ","),
		strings.ReplaceAll(table, `"`, `""`), filter)
	_, err := database.Exec(statement)
	if err != nil {
		t.Fatalf("copy V17 table %s: %v", table, err)
	}
}

func sqliteFixtureColumns(t *testing.T, database *sql.DB, schema, table string) map[string]bool {
	t.Helper()
	rows, err := database.Query(fmt.Sprintf(`PRAGMA %s.table_info("%s")`, schema,
		strings.ReplaceAll(table, `"`, `""`)))
	sqliteTestNoError(t, err)
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		sqliteTestNoError(t, rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey))
		result[name] = true
	}
	return result
}

func v17FixtureFilter(table string) string {
	reviewerRefs := `(SELECT ref FROM source.executions WHERE purpose IN ('primary_review','adversarial_review'))`
	reviewIntents := `(SELECT ref FROM source.effect_intents WHERE execution_ref IN ` + reviewerRefs + `)`
	switch table {
	case "executions":
		return `WHERE purpose NOT IN ('primary_review','adversarial_review')`
	case "outbox", "action_consumption_receipts", "events":
		return `WHERE execution_ref IS NULL OR execution_ref NOT IN ` + reviewerRefs
	case "effect_intents":
		return `WHERE ref NOT IN ` + reviewIntents
	case "effect_approvals":
		return `WHERE intent_ref NOT IN ` + reviewIntents
	case "effect_attempts", "effect_receipts":
		return `WHERE intent_ref NOT IN ` + reviewIntents
	case "budget_reservations":
		return `WHERE effect_intent_ref NOT IN ` + reviewIntents
	case "budget_settlements":
		return `WHERE reservation_ref NOT IN (SELECT ref FROM source.budget_reservations WHERE effect_intent_ref IN ` + reviewIntents + `)`
	case "artifacts":
		return `WHERE ref NOT IN (SELECT assessment_artifact_ref FROM source.review_records)`
	case "artifact_occurrences":
		return `WHERE kind NOT IN ('review_assessment','review_diagnostic')`
	default:
		return ""
	}
}

func assertSQLiteV18MigrationHealthy(t *testing.T, repository *Repository) {
	t.Helper()
	var version, receipt, violations int
	sqliteTestNoError(t, repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=13`).Scan(&receipt))
	rows, err := repository.db.Query(`PRAGMA foreign_key_check`)
	sqliteTestNoError(t, err)
	for rows.Next() {
		violations++
	}
	sqliteTestNoError(t, rows.Close())
	if version != recoverySchemaV18 || receipt != 1 || violations != 0 {
		t.Fatalf("V18 migration version=%d receipt=%d foreign-keys=%d", version, receipt, violations)
	}
}
