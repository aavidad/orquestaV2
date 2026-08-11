package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"testing"
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

	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
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

func downgradeV34PhysicalManifestToCanonicalV33(t *testing.T, database *sql.DB) {
	t.Helper()
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
