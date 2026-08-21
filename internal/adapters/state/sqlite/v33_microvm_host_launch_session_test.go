package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

const v32MicroVMHostLaunchMigrationSHA256 = "sha256:74e4266f541a9bda624ecc7cf993be6d3cf11794d6dbf08367395be70f33fe7a"

func TestV33MigrationPrefixesPreserveV32ChecksumAndAddSessionRatchet(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != recoverySchemaLatest {
		t.Fatalf("migration count=%d latest=%d", len(migrations), recoverySchemaLatest)
	}
	v32 := migrations[recoverySchemaV38MicroVMHostLaunch-1]
	v33 := migrations[recoverySchemaV38MicroVMHostSession-1]
	if v32.name != "032_microvm_host_launch_authority.sql" ||
		v32.checksum != v32MicroVMHostLaunchMigrationSHA256 {
		t.Fatalf("V32 migration changed name=%q checksum=%q", v32.name, v32.checksum)
	}
	if v33.name != "033_microvm_host_launch_session.sql" ||
		!strings.Contains(v33.sql, "execution.execution_session_ref<>''") ||
		!strings.Contains(v33.sql, "execution.execution_session_ref=NEW.session_ref") {
		t.Fatalf("V33 migration lacks session ratchet name=%q", v33.name)
	}
	prefixV32, err := recoveryMigrationPrefix(migrations, recoverySchemaV38MicroVMHostLaunch)
	if err != nil || len(prefixV32) != recoverySchemaV38MicroVMHostLaunch ||
		prefixV32[len(prefixV32)-1].checksum != v32MicroVMHostLaunchMigrationSHA256 {
		t.Fatalf("V32 prefix len=%d err=%v", len(prefixV32), err)
	}
	prefixV33, err := recoveryMigrationPrefix(migrations, recoverySchemaV38MicroVMHostSession)
	if err != nil || len(prefixV33) != recoverySchemaV38MicroVMHostSession ||
		prefixV33[len(prefixV33)-1].name != "033_microvm_host_launch_session.sql" {
		t.Fatalf("V33 prefix len=%d err=%v", len(prefixV33), err)
	}
}

func TestV33CausalTriggerRequiresExactNonEmptyExecutionSession(t *testing.T) {
	t.Run("exact", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		_, attempt := seedV32EffectAttempt(t, system, "v33-session-exact")
		authority := newV32MicroVMHostLaunchAuthority(t, attempt)
		bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
		sqliteTestNoError(t, insertV32MicroVMHostLaunchAuthority(system.repository.db, authority))
	})

	t.Run("empty", func(t *testing.T) {
		system, attempt := seedV27AmbiguousLaunch(t, "v33-session-empty")
		authority := newV32MicroVMHostLaunchAuthority(t, attempt)
		assertV32CausalInsertRejected(t, system.repository.db, authority)
	})

	t.Run("crossed", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		_, attempt := seedV32EffectAttempt(t, system, "v33-session-crossed")
		authority := newV32MicroVMHostLaunchAuthority(t, attempt)
		bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
		crossed := recoveryV32ExecutionSessionRef(t, "v33-crossed-authority")
		authority.sessionRef = crossed.String()
		requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
			ports.MicroVMHostLaunchAuthorityKey{
				RunRef: attempt.Subject.ExecutionRef, ActionFence: attempt.ActionFence,
			},
			attempt.Ref,
			crossed,
		)
		if err != nil {
			t.Fatal(err)
		}
		authority.requestRef = requestRef
		assertV32CausalInsertRejected(t, system.repository.db, authority)
	})
}

func TestV33MigrationPreservesValidV32Authority(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	_, attempt := seedV32EffectAttempt(t, system, "v33-upgrade-valid")
	authority := newV32MicroVMHostLaunchAuthority(t, attempt)
	bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
	sqliteTestNoError(t, insertV32MicroVMHostLaunchAuthority(system.repository.db, authority))
	downgradeV33MicroVMHostSessionToCanonicalV32(t, system.repository.db)
	requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, system.repository.db)
	sqliteTestNoError(t, system.repository.Close())

	reopened := openFullTestRepository(t, Options{
		Path: system.path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: system.clock.Now,
	})
	requireRecoveryV32MicroVMHostLaunchAuthorityValid(t, reopened.db)
	key := ports.MicroVMHostLaunchAuthorityKey{
		RunRef: attempt.Subject.ExecutionRef, ActionFence: attempt.ActionFence,
	}
	resolved, err := reopened.Resolve(context.Background(), key)
	if err != nil || resolved.SessionRef.String() != authority.sessionRef || resolved.ExternalRef != "" {
		t.Fatalf("migrated authority=%+v err=%v", resolved, err)
	}
	var version, receipt int
	sqliteTestNoError(t, reopened.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, reopened.db.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38MicroVMHostSession,
	).Scan(&receipt))
	if version != recoverySchemaLatest || receipt != 1 {
		t.Fatalf("migrated version=%d receipt=%d", version, receipt)
	}
}

func TestV33MigrationAtomicallyRejectsHistoricalV32CrossedSession(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "v33-upgrade-corrupt")
	downgradeV33MicroVMHostSessionToCanonicalV32(t, system.repository.db)
	authority := newV32MicroVMHostLaunchAuthority(t, attempt)
	crossed := recoveryV32ExecutionSessionRef(t, "v33-historical-execution")
	mustV10Exec(t, system.repository.db, `
UPDATE executions SET execution_session_ref=? WHERE ref=? AND execution_session_ref=''`,
		crossed.String(), attempt.Subject.ExecutionRef.String())
	sqliteTestNoError(t, insertV32MicroVMHostLaunchAuthority(system.repository.db, authority))
	var triggerV32 string
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='microvm_host_launch_authorities_causal_insert'`).Scan(&triggerV32))
	sqliteTestNoError(t, system.repository.Close())

	reopened, err := Open(context.Background(), Options{
		Path: system.path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		Now: system.clock.Now,
	})
	if reopened != nil {
		_ = reopened.Close()
		t.Fatal("historical crossed V32 authority unexpectedly migrated")
	}
	if !application.IsStateError(err, application.StateConflict) ||
		!strings.Contains(sqliteTestErrorChain(err),
			"sqlite.microvm_host_launch_authority_session_migration_invalid") {
		t.Fatalf("crossed V32 migration error=%s", sqliteTestErrorChain(err))
	}

	raw := openRawV10TestDatabase(t, system.path)
	defer raw.Close()
	var version, receiptV33, receiptV32, guardObjects, rows int
	var triggerAfter, authoritySession, executionSession string
	sqliteTestNoError(t, raw.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, raw.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38MicroVMHostSession,
	).Scan(&receiptV33))
	sqliteTestNoError(t, raw.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND checksum=?`,
		recoverySchemaV38MicroVMHostLaunch, v32MicroVMHostLaunchMigrationSHA256,
	).Scan(&receiptV32))
	sqliteTestNoError(t, raw.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema
WHERE name IN ('microvm_host_launch_session_v33_preflight',
 'microvm_host_launch_session_v33_preflight_guard')`).Scan(&guardObjects))
	sqliteTestNoError(t, raw.QueryRow(`
SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='microvm_host_launch_authorities_causal_insert'`).Scan(&triggerAfter))
	sqliteTestNoError(t, raw.QueryRow(`
SELECT authority.session_ref,execution.execution_session_ref
FROM microvm_host_launch_authorities authority
JOIN executions execution ON execution.ref=authority.execution_ref
WHERE authority.execution_ref=? AND authority.action_fence=?`,
		authority.executionRef, authority.actionFence,
	).Scan(&authoritySession, &executionSession))
	sqliteTestNoError(t, raw.QueryRow(
		`SELECT COUNT(*) FROM microvm_host_launch_authorities`,
	).Scan(&rows))
	if version != recoverySchemaV38MicroVMHostLaunch || receiptV33 != 0 || receiptV32 != 1 ||
		guardObjects != 0 || rows != 1 || triggerAfter != triggerV32 ||
		authoritySession != authority.sessionRef || executionSession != crossed.String() {
		t.Fatalf("rollback version=%d receipts=%d/%d guards=%d rows=%d trigger_same=%t sessions=%q/%q",
			version, receiptV32, receiptV33, guardObjects, rows, triggerAfter == triggerV32,
			authoritySession, executionSession)
	}
	expectedInventory, err := canonicalSchemaInventoryDigest(recoverySchemaV38MicroVMHostLaunch)
	if err != nil {
		t.Fatal(err)
	}
	actualInventory, err := schemaInventoryDigest(context.Background(), raw)
	if err != nil || actualInventory != expectedInventory {
		t.Fatalf("rollback inventory actual=%s expected=%s err=%v",
			actualInventory, expectedInventory, err)
	}
}

func downgradeV33MicroVMHostSessionToCanonicalV32(t *testing.T, database *sql.DB) {
	t.Helper()
	downgradeV35AgentEnvironmentLifecycleToCanonicalV34(t, database)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	v32SQL := migrations[recoverySchemaV38MicroVMHostLaunch-1].sql
	start := strings.Index(v32SQL, "CREATE TRIGGER microvm_host_launch_authorities_causal_insert")
	if start < 0 {
		t.Fatal("V32 causal trigger definition not found")
	}
	endMarker := "\n\nCREATE TRIGGER microvm_host_launch_authorities_bind_once"
	end := strings.Index(v32SQL[start:], endMarker)
	if end < 0 {
		t.Fatal("V32 causal trigger definition not found")
	}
	causalTrigger := v32SQL[start : start+end]
	transaction, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback()
	if _, err := transaction.Exec(`ALTER TABLE agent_environment_receipts DROP COLUMN physical_manifest_digest`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`ALTER TABLE agent_environment_receipts DROP COLUMN physical_manifest_ref`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`DROP TRIGGER microvm_host_launch_authorities_causal_insert`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(causalTrigger); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(
		`DELETE FROM schema_migrations WHERE version IN (?,?)`,
		recoverySchemaV38MicroVMHostSession, recoverySchemaV38PhysicalManifest,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(`PRAGMA user_version=32`); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	var version, receipt int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38MicroVMHostSession,
	).Scan(&receipt))
	if version != recoverySchemaV38MicroVMHostLaunch || receipt != 0 {
		t.Fatalf("downgrade version=%d receipt=%d", version, receipt)
	}
	expected, err := canonicalSchemaInventoryDigest(recoverySchemaV38MicroVMHostLaunch)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := schemaInventoryDigest(context.Background(), database)
	if err != nil || actual != expected {
		t.Fatalf("V32 inventory actual=%s expected=%s err=%v", actual, expected, err)
	}
}
