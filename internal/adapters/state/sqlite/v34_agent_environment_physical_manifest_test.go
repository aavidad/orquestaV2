package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
)

const v33MicroVMHostSessionMigrationSHA256 = "sha256:f3f92c97ae29386e043677bf52143d5a2608e5b66772744c3bf484c700340d69"

func TestV34MigrationPreservesV33AndAddsAtomicPhysicalManifest(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != recoverySchemaLatest {
		t.Fatalf("migration count=%d latest=%d", len(migrations), recoverySchemaLatest)
	}
	v33 := migrations[recoverySchemaV38MicroVMHostSession-1]
	v34 := migrations[recoverySchemaV38PhysicalManifest-1]
	if v33.name != "033_microvm_host_launch_session.sql" ||
		v33.checksum != v33MicroVMHostSessionMigrationSHA256 {
		t.Fatalf("V33 migration changed name=%q checksum=%q", v33.name, v33.checksum)
	}
	for _, fragment := range []string{
		"physical_manifest_ref", "physical_manifest_digest",
		"(physical_manifest_ref IS NULL)=(physical_manifest_digest IS NULL)",
		"length(CAST(physical_manifest_ref AS BLOB)) BETWEEN 1 AND 512",
		"physical_manifest_digest NOT GLOB '*[^0-9a-f]*'",
	} {
		if v34.name != "034_agent_environment_physical_manifest.sql" ||
			!strings.Contains(v34.sql, fragment) {
			t.Fatalf("V34 migration lacks %q name=%q", fragment, v34.name)
		}
	}
	prefixV33, err := recoveryMigrationPrefix(migrations, recoverySchemaV38MicroVMHostSession)
	if err != nil || len(prefixV33) != recoverySchemaV38MicroVMHostSession ||
		prefixV33[len(prefixV33)-1].checksum != v33MicroVMHostSessionMigrationSHA256 {
		t.Fatalf("V33 prefix len=%d err=%v", len(prefixV33), err)
	}
	prefixV34, err := recoveryMigrationPrefix(migrations, recoverySchemaV38PhysicalManifest)
	if err != nil || len(prefixV34) != recoverySchemaV38PhysicalManifest ||
		prefixV34[len(prefixV34)-1].name != "034_agent_environment_physical_manifest.sql" {
		t.Fatalf("V34 prefix len=%d err=%v", len(prefixV34), err)
	}
}

func TestV34MigrationUpgradesCanonicalV33Atomically(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	downgradeV34PhysicalManifestToCanonicalV33(t, system.repository.db)
	sqliteTestNoError(t, system.repository.Close())

	reopened := openFullTestRepository(t, Options{
		Path: system.path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: system.clock.Now,
	})
	var version, receipt, columns, foreignKeyViolations int
	var name string
	sqliteTestNoError(t, reopened.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, reopened.db.QueryRow(`
SELECT name FROM schema_migrations WHERE version=?`, recoverySchemaV38PhysicalManifest).Scan(&name))
	sqliteTestNoError(t, reopened.db.QueryRow(`
SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV38PhysicalManifest).Scan(&receipt))
	sqliteTestNoError(t, reopened.db.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('agent_environment_receipts')
WHERE name IN ('physical_manifest_ref','physical_manifest_digest')`).Scan(&columns))
	sqliteTestNoError(t, reopened.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&foreignKeyViolations))
	if version != recoverySchemaLatest || name != "034_agent_environment_physical_manifest.sql" ||
		receipt != 1 || columns != 2 || foreignKeyViolations != 0 {
		t.Fatalf("upgrade version=%d name=%q receipt=%d columns=%d fk=%d",
			version, name, receipt, columns, foreignKeyViolations)
	}
}

func TestV35MigrationUpgradesPopulatedV34EnvironmentReceipt(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteV15System(t, 2)
	system.external.requierePreservacion = true
	system.orchestrator = newSQLiteV16Orchestrator(t, system)
	created, err := system.orchestrator.Submit(ctx, system.access, application.SubmitRequest{
		RequestRef: "request:v35-populated-preservation", Statement: "preserve populated V34 receipt", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v35-populated", Key: "phase:v35-populated",
				TemplateRef: "phase-template:v35-populated",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "preserve V34 evidence", Phase: "phase:v35-populated",
				Role: "role:writer", WriteSet: []string{"internal/v35-populated"},
				RequiredTests: sqliteRequiredTestSpecs("required-test:v35-populated"),
				CouncilPolicy: council.PolicyAuto, OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteV16Actions(t, system, application.ActionPrepareWorkspace, application.ActionLaunchAgent)
	record, err := system.repository.GetGoal(ctx, created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	if len(record.Executions) != 1 || len(record.WorkspaceBindings) != 1 {
		t.Fatalf("populated V34 fixture incomplete: %+v", record)
	}
	execution, binding := record.Executions[0], record.WorkspaceBindings[0]
	var launchFence uint64
	for _, consumption := range record.ConsumptionReceipts {
		if consumption.Kind == application.ActionLaunchAgent && consumption.ExecutionRef == execution.Ref {
			launchFence = consumption.Fence
		}
	}
	receipt := comprobantePreservacionSQLite(
		t, record, execution, binding, launchFence, system.clock.Now(),
	)
	receipt.ManifiestoFisicoRef = "physical-manifest:v35-populated"
	receipt.ManifiestoFisicoDigest = strings.Repeat("e", 64)
	stored, written, err := system.repository.RegistrarPreservacionEntornoAgente(ctx, receipt)
	if err != nil || !written || !reflect.DeepEqual(stored, receipt) {
		t.Fatalf("seed populated V34 receipt written=%v got=%+v err=%v", written, stored, err)
	}
	downgradeV35AgentEnvironmentLifecycleToCanonicalV34(t, system.repository.db)
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openFullTestRepository(t, Options{
		Path: system.path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: system.clock.Now,
	})
	recovered, written, err := reopened.RegistrarPreservacionEntornoAgente(ctx, receipt)
	if err != nil || written || !reflect.DeepEqual(recovered, receipt) {
		t.Fatalf("upgraded populated V34 receipt written=%v got=%+v err=%v", written, recovered, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, reopened.db); err != nil {
		t.Fatalf("upgraded populated V34 recovery: %s", sqliteTestErrorChain(err))
	}
}

func downgradeV34PhysicalManifestToCanonicalV33(t *testing.T, database *sql.DB) {
	t.Helper()
	downgradeV35AgentEnvironmentLifecycleToCanonicalV34(t, database)
	transaction, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	for _, statement := range []string{
		`ALTER TABLE agent_environment_receipts DROP COLUMN physical_manifest_digest`,
		`ALTER TABLE agent_environment_receipts DROP COLUMN physical_manifest_ref`,
	} {
		if _, err := transaction.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := transaction.Exec(
		`DELETE FROM schema_migrations WHERE version=?`, recoverySchemaV38PhysicalManifest,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`PRAGMA user_version=33`); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	var version, receipt, columns int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38PhysicalManifest,
	).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('agent_environment_receipts')
WHERE name IN ('physical_manifest_ref','physical_manifest_digest')`).Scan(&columns))
	if version != recoverySchemaV38MicroVMHostSession || receipt != 0 || columns != 0 {
		t.Fatalf("downgrade version=%d receipt=%d columns=%d", version, receipt, columns)
	}
	expected, err := canonicalSchemaInventoryDigest(recoverySchemaV38MicroVMHostSession)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := schemaInventoryDigest(context.Background(), database)
	if err != nil || actual != expected {
		t.Fatalf("V33 inventory actual=%s expected=%s err=%v", actual, expected, err)
	}
}

func downgradeV35AgentEnvironmentLifecycleToCanonicalV34(t *testing.T, database *sql.DB) {
	t.Helper()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	prefix, err := recoveryMigrationPrefix(migrations, recoverySchemaV38PhysicalManifest)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := sql.Open(driverName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	canonical.SetMaxOpenConns(1)
	defer canonical.Close()
	if err := applyRecoveryMigrationPrefix(context.Background(), canonical, prefix); err != nil {
		t.Fatal(err)
	}

	tables := []string{
		"outbox",
		"effect_intents",
		"effect_receipts",
		"action_consumption_receipts",
		"agent_environment_receipts",
	}
	tableSQL := make(map[string]string, len(tables))
	tableColumns := make(map[string][]string, len(tables))
	for _, table := range tables {
		var statement string
		if err := canonical.QueryRow(`SELECT sql FROM sqlite_schema WHERE type='table' AND name=?`, table).
			Scan(&statement); err != nil {
			t.Fatal(err)
		}
		tableSQL[table] = statement
		rows, err := canonical.Query(
			"SELECT name FROM pragma_table_xinfo(" + quoteSQLiteIdentifier(table) + ") WHERE hidden=0 ORDER BY cid",
		)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var column string
			if err := rows.Scan(&column); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			tableColumns[table] = append(tableColumns[table], column)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	type schemaObject struct {
		kind, name, statement string
	}
	var objects []schemaObject
	rows, err := canonical.Query(`
SELECT type,name,sql FROM sqlite_schema
WHERE type IN ('index','trigger') AND sql IS NOT NULL
 AND (tbl_name IN ('outbox','effect_intents','effect_receipts',
                   'action_consumption_receipts','agent_environment_receipts')
      OR name='effect_attempts_causal_guard')
ORDER BY CASE type WHEN 'index' THEN 0 ELSE 1 END,name`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var object schemaObject
		if err := rows.Scan(&object.kind, &object.name, &object.statement); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		objects = append(objects, object)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	connection, err := database.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `PRAGMA legacy_alter_table=ON`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = connection.ExecContext(context.Background(), `PRAGMA legacy_alter_table=OFF`)
		_, _ = connection.ExecContext(context.Background(), `PRAGMA foreign_keys=ON`)
	}()
	transaction, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	if _, err := transaction.Exec(`DROP TABLE agent_launch_expired_continuation_authorities;
DROP TABLE agent_launch_expired_continuation_subjects;
DROP TABLE agent_launch_reconciliation_receipts;
DROP TABLE agent_launch_reconciliation_attempts;
DROP TABLE agent_launch_reconciliation_jobs;
DROP TABLE agent_launch_reconciliation_authorities`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`DROP TABLE effect_non_application_evidence`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`DROP TABLE wizard_gaps_result_snapshots`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`DROP TRIGGER effect_attempts_causal_guard`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`DROP TABLE agent_environment_lifecycles`); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		if _, err := transaction.Exec("ALTER TABLE " + quoteSQLiteIdentifier(table) +
			" RENAME TO " + quoteSQLiteIdentifier(table+"_v35_fixture")); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range tables {
		if _, err := transaction.Exec(tableSQL[table]); err != nil {
			t.Fatalf("create canonical %s: %v", table, err)
		}
		columns := make([]string, len(tableColumns[table]))
		for index, column := range tableColumns[table] {
			columns[index] = quoteSQLiteIdentifier(column)
		}
		columnList := strings.Join(columns, ",")
		if _, err := transaction.Exec("INSERT INTO " + quoteSQLiteIdentifier(table) +
			"(" + columnList + ") SELECT " + columnList + " FROM " +
			quoteSQLiteIdentifier(table+"_v35_fixture")); err != nil {
			t.Fatalf("copy canonical %s: %v", table, err)
		}
	}
	for _, table := range tables {
		if _, err := transaction.Exec("DROP TABLE " + quoteSQLiteIdentifier(table+"_v35_fixture")); err != nil {
			t.Fatal(err)
		}
	}
	for _, object := range objects {
		if _, err := transaction.Exec(object.statement); err != nil {
			t.Fatalf("create canonical %s %s: %v", object.kind, object.name, err)
		}
	}
	if _, err := transaction.Exec(
		`DROP TRIGGER microvm_host_launch_authorities_runtime_digest_required;
DROP TABLE microvm_host_launch_runtime_digests;
DROP TABLE microvm_host_launch_runtime_digest_legacy_exemptions;
DROP TABLE microvm_host_launch_runtime_digest_epoch;
DELETE FROM schema_migrations WHERE version IN (?,?,?,?,?,?,?)`,
		recoverySchemaV38EnvironmentLifecycle, recoverySchemaV23WizardGapsSnapshot,
		recoverySchemaV38StopNonApplication, recoverySchemaV38StopRecoveryClaim,
		recoverySchemaV38LaunchRuntimeDigests, recoverySchemaV38TerminalLaunchReconciliation,
		recoverySchemaV38ExpiredLaunchContinuation,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`PRAGMA user_version=34`); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	var foreignKeyViolations int
	if err := connection.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM pragma_foreign_key_check`).
		Scan(&foreignKeyViolations); err != nil || foreignKeyViolations != 0 {
		t.Fatalf("V34 foreign keys violations=%d err=%v", foreignKeyViolations, err)
	}
	expected, err := canonicalSchemaInventoryDigest(recoverySchemaV38PhysicalManifest)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := schemaInventoryDigest(context.Background(), connection)
	if err != nil || actual != expected {
		t.Fatalf("V34 inventory actual=%s expected=%s err=%v", actual, expected, err)
	}
}
