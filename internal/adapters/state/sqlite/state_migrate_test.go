package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateStateOnlyAlreadyTargetWALDoesNotMutateOrCreateSidecars(t *testing.T) {
	path := seedStateMigrationDatabase(t, recoverySchemaV38ExpiredLaunchContinuation)
	setStateMigrationWALMode(t, path)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	receipt, err := MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
	if err != nil {
		t.Fatalf("MigrateStateOnly() error=%s", sqliteTestErrorChain(err))
	}
	if receipt.State != StateMigrationAlreadyAtTarget || !receipt.ReadOnlyVerified ||
		receipt.TargetVersion != recoverySchemaV38ExpiredLaunchContinuation ||
		receipt.PostCommitCauseCode != "" {
		t.Fatalf("receipt=%+v", receipt)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("already-at-target verification changed database bytes")
	}
	assertStateMigrationNoSidecars(t, path)
}

func TestMigrateStateOnlyMovesSidecarFreeWALV39ToV41(t *testing.T) {
	path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
	setStateMigrationWALMode(t, path)
	receipt, err := MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
	if err != nil {
		t.Fatalf("MigrateStateOnly() error=%s", sqliteTestErrorChain(err))
	}
	if receipt.State != StateMigrationMigrated || !receipt.ReadOnlyVerified {
		t.Fatalf("receipt=%+v", receipt)
	}
	assertStateMigrationNoSidecars(t, path)
}

func TestMigrateStateOnlyMovesExactV39ToV41AndReopensReadOnly(t *testing.T) {
	path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
	receipt, err := MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
	if err != nil {
		t.Fatalf("MigrateStateOnly() error=%s", sqliteTestErrorChain(err))
	}
	if receipt.SchemaVersion != 1 || receipt.Mode != StateMigrationModeNoRuntime ||
		receipt.State != StateMigrationMigrated ||
		receipt.RequestedFrom != 39 || receipt.TargetVersion != 41 ||
		!strings.HasPrefix(receipt.TargetSchemaRef, schemaRefPrefix) ||
		!strings.HasPrefix(receipt.TargetLogicalSHA256, "sha256:") ||
		!strings.HasPrefix(receipt.DatabaseIdentity, "local-state:sha256:") ||
		!receipt.ReadOnlyVerified || receipt.PostCommitCauseCode != "" {
		t.Fatalf("receipt=%+v", receipt)
	}
	replayed, err := MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
	if err != nil {
		t.Fatalf("idempotent MigrateStateOnly() error=%s", sqliteTestErrorChain(err))
	}
	if replayed.State != StateMigrationAlreadyAtTarget || !replayed.ReadOnlyVerified ||
		replayed.TargetSchemaRef != receipt.TargetSchemaRef ||
		replayed.TargetLogicalSHA256 != receipt.TargetLogicalSHA256 ||
		replayed.DatabaseIdentity != receipt.DatabaseIdentity {
		t.Fatalf("replayed receipt=%+v initial=%+v", replayed, receipt)
	}
	database := openStateMigrationRaw(t, path)
	defer database.Close()
	var version, receipts, tables int
	sqliteTestNoError(t, database.QueryRow(`SELECT
 (SELECT user_version FROM pragma_user_version),
 (SELECT COUNT(*) FROM schema_migrations WHERE version IN (40,41)),
 (SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name IN (
  'agent_launch_reconciliation_authorities','agent_launch_expired_continuation_authorities'))`).Scan(
		&version, &receipts, &tables,
	))
	if version != 41 || receipts != 2 || tables != 2 {
		t.Fatalf("version=%d receipts=%d tables=%d", version, receipts, tables)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), database); err != nil {
		t.Fatalf("reopened recovery validation=%s", sqliteTestErrorChain(err))
	}
	assertStateMigrationNoSidecars(t, path)
}

func TestMigrateStateOnlyRollsBackBothStepsOnFailureAfterV40(t *testing.T) {
	path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
	_, err := migrateStateOnly(context.Background(), stateMigrationTestRequest(path), stateMigrationHooks{
		afterStep: func(version int) error {
			if version == recoverySchemaV38TerminalLaunchReconciliation {
				return errors.New("test.stop_after_v40")
			}
			return nil
		},
	})
	if err == nil || !strings.Contains(sqliteTestErrorChain(err), "test.stop_after_v40") {
		t.Fatalf("rollback error=%s", sqliteTestErrorChain(err))
	}
	database := openStateMigrationRaw(t, path)
	defer database.Close()
	var version, receipts, tables int
	sqliteTestNoError(t, database.QueryRow(`SELECT
 (SELECT user_version FROM pragma_user_version),
 (SELECT COUNT(*) FROM schema_migrations WHERE version IN (40,41)),
 (SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name IN (
  'agent_launch_reconciliation_authorities','agent_launch_expired_continuation_authorities'))`).Scan(
		&version, &receipts, &tables,
	))
	if version != 39 || receipts != 0 || tables != 0 {
		t.Fatalf("partial migration survived version=%d receipts=%d tables=%d", version, receipts, tables)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), database); err != nil {
		t.Fatalf("rolled back V39 invalid=%s", sqliteTestErrorChain(err))
	}
	assertStateMigrationNoSidecars(t, path)
}

func TestMigrateStateOnlyRejectsPartialFutureSidecarsAndNonExactRequest(t *testing.T) {
	for _, version := range []int{recoverySchemaV38TerminalLaunchReconciliation} {
		t.Run("version-"+string(rune('0'+version%10)), func(t *testing.T) {
			path := seedStateMigrationDatabase(t, version)
			_, err := MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
			if err == nil {
				t.Fatalf("accepted source version=%d", version)
			}
			if got := stateMigrationUserVersion(t, path); got != version {
				t.Fatalf("source version changed=%d want=%d", got, version)
			}
		})
	}

	t.Run("future", func(t *testing.T) {
		path := seedStateMigrationDatabase(t, recoverySchemaV38ExpiredLaunchContinuation)
		database := openStateMigrationRaw(t, path)
		_, err := database.Exec(`PRAGMA user_version=42`)
		sqliteTestNoError(t, err)
		sqliteTestNoError(t, database.Close())
		_, err = MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
		if err == nil || stateMigrationUserVersion(t, path) != 42 {
			t.Fatalf("future migration err=%s", sqliteTestErrorChain(err))
		}
	})

	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		t.Run("sidecar"+suffix, func(t *testing.T) {
			path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path+suffix, []byte("foreign"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
			if err == nil || !strings.Contains(sqliteTestErrorChain(err), "sidecar_present") {
				t.Fatalf("sidecar %s err=%s", suffix, sqliteTestErrorChain(err))
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("sidecar %s rejection changed database bytes", suffix)
			}
			foreign, readErr := os.ReadFile(path + suffix)
			if readErr != nil || string(foreign) != "foreign" {
				t.Fatalf("sidecar %s was not preserved: bytes=%q err=%v", suffix, foreign, readErr)
			}
		})
	}

	t.Run("request", func(t *testing.T) {
		path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
		request := stateMigrationTestRequest(path)
		request.Mode = "runtime"
		if _, err := MigrateStateOnly(context.Background(), request); err == nil ||
			stateMigrationUserVersion(t, path) != 39 {
			t.Fatalf("accepted runtime mode err=%s", sqliteTestErrorChain(err))
		}
		request = stateMigrationTestRequest(path)
		request.Options.Path = filepath.Base(path)
		if _, err := MigrateStateOnly(context.Background(), request); err == nil {
			t.Fatal("accepted relative database path")
		}
		link := filepath.Join(filepath.Dir(path), "state-link.sqlite")
		if err := os.Symlink(path, link); err != nil {
			t.Fatal(err)
		}
		request = stateMigrationTestRequest(link)
		if _, err := MigrateStateOnly(context.Background(), request); err == nil {
			t.Fatal("accepted symlink database path")
		}
	})
}

func TestMigrateStateOnlyReturnsDurablePendingReceiptAfterCommitAndRetryObservesV41(t *testing.T) {
	path := seedStateMigrationDatabase(t, recoverySchemaV38LaunchRuntimeDigests)
	_, err := migrateStateOnly(context.Background(), stateMigrationTestRequest(path), stateMigrationHooks{
		afterCommit: func() error { return errors.New("test.crash_after_commit") },
	})
	var committed *CommittedStateMigrationError
	if !errors.As(err, &committed) {
		t.Fatalf("postcommit error=%s", sqliteTestErrorChain(err))
	}
	if committed.Receipt.State != StateMigrationCommittedPostcheckPending ||
		committed.Receipt.ReadOnlyVerified ||
		committed.Receipt.PostCommitCauseCode != "sqlite.state_migration_postcommit_check_failed" ||
		!strings.HasPrefix(committed.Receipt.TargetSchemaRef, schemaRefPrefix) ||
		!strings.HasPrefix(committed.Receipt.TargetLogicalSHA256, "sha256:") ||
		stateMigrationUserVersion(t, path) != recoverySchemaV38ExpiredLaunchContinuation {
		t.Fatalf("committed receipt=%+v error=%s", committed.Receipt, sqliteTestErrorChain(err))
	}

	replayed, err := MigrateStateOnly(context.Background(), stateMigrationTestRequest(path))
	if err != nil {
		t.Fatalf("retry error=%s", sqliteTestErrorChain(err))
	}
	if replayed.State != StateMigrationAlreadyAtTarget || !replayed.ReadOnlyVerified ||
		replayed.TargetSchemaRef != committed.Receipt.TargetSchemaRef ||
		replayed.TargetLogicalSHA256 != committed.Receipt.TargetLogicalSHA256 {
		t.Fatalf("retry receipt=%+v pending=%+v", replayed, committed.Receipt)
	}
}

func TestRuntimeOpenAndStateMigrationShareExclusiveWriterLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "orquesta.sqlite")
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	second, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
	})
	if second != nil || err == nil || !strings.Contains(sqliteTestErrorChain(err), "state_writer_active") {
		t.Fatalf("second=%v err=%s", second, sqliteTestErrorChain(err))
	}
	request := stateMigrationTestRequest(path)
	if migrated, err := MigrateStateOnly(context.Background(), request); err == nil ||
		migrated != (StateMigrationReceipt{}) {
		t.Fatalf("live database migrated=%+v err=%s", migrated, sqliteTestErrorChain(err))
	}
}

func TestMigrateStateOnlyRejectsDivergentRunReferencesBeforeCreatingV40(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "state-migration-divergent-run")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	if _, err := system.repository.PrepareWithRuntime(
		context.Background(), authority, sqliteMicroVMHostLaunchRuntimeDigests(authority.Key, "1"),
	); err != nil {
		t.Fatal(err)
	}
	downgradeStateMigrationSourceToV39(t, system.repository.db)
	mustV10Exec(t, system.repository.db, `DROP TRIGGER microvm_host_launch_runtime_digests_immutable_delete`)
	_, err := system.repository.db.Exec(`DELETE FROM microvm_host_launch_runtime_digests
WHERE execution_ref=? AND action_fence=?`, authority.Key.RunRef.String(), authority.Key.ActionFence)
	sqliteTestNoError(t, err)
	path := system.path
	sqliteTestNoError(t, system.repository.Close())

	if _, err := MigrateStateOnly(context.Background(), stateMigrationTestRequest(path)); err == nil {
		t.Fatal("migration accepted crossed V39 run/runtime references")
	}
	database := openStateMigrationRaw(t, path)
	defer database.Close()
	var version, v40Tables int
	sqliteTestNoError(t, database.QueryRow(`SELECT
 (SELECT user_version FROM pragma_user_version),
 (SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name LIKE 'agent_launch_reconciliation_%')`).Scan(
		&version, &v40Tables,
	))
	if version != 39 || v40Tables != 0 {
		t.Fatalf("divergent source mutated version=%d V40 tables=%d", version, v40Tables)
	}
}

func seedStateMigrationDatabase(t *testing.T, version int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "orquesta.sqlite")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	if version <= 0 || version > len(migrations) {
		t.Fatalf("invalid seed version=%d", version)
	}
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(context.Background(), database, migrations[:version]))
	sqliteTestNoError(t, database.Close())
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	assertStateMigrationNoSidecars(t, path)
	return path
}

func stateMigrationTestRequest(path string) StateMigrationRequest {
	return StateMigrationRequest{
		Options: Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2},
		Mode:    StateMigrationModeNoRuntime, ExpectFrom: 39, ExpectTo: 41,
	}
}

func openStateMigrationRaw(t *testing.T, path string) *sql.DB {
	t.Helper()
	database, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	return database
}

func setStateMigrationWALMode(t *testing.T, path string) {
	t.Helper()
	database := openStateMigrationRaw(t, path)
	var mode string
	sqliteTestNoError(t, database.QueryRow(`PRAGMA journal_mode=WAL`).Scan(&mode))
	if mode != "wal" {
		t.Fatalf("journal mode=%q want=wal", mode)
	}
	var busy, log, checkpointed int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(
		&busy, &log, &checkpointed,
	))
	if busy != 0 || log != checkpointed {
		t.Fatalf("WAL checkpoint busy=%d log=%d checkpointed=%d", busy, log, checkpointed)
	}
	sqliteTestNoError(t, database.Close())
	assertStateMigrationNoSidecars(t, path)
}

func stateMigrationUserVersion(t *testing.T, path string) int {
	t.Helper()
	database := openStateMigrationRaw(t, path)
	defer database.Close()
	var version int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	return version
}

func assertStateMigrationNoSidecars(t *testing.T, path string) {
	t.Helper()
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Lstat(path + suffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("sidecar %s err=%v", suffix, err)
		}
	}
}

func downgradeStateMigrationSourceToV39(t *testing.T, database *sql.DB) {
	t.Helper()
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	canonical, err := sql.Open(driverName, ":memory:")
	sqliteTestNoError(t, err)
	canonical.SetMaxOpenConns(1)
	defer canonical.Close()
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(
		context.Background(), canonical, migrations[:recoverySchemaV38LaunchRuntimeDigests],
	))
	var causalGuard string
	sqliteTestNoError(t, canonical.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='effect_receipts_causal_guard'`).Scan(&causalGuard))
	mustV10Exec(t, database, `DROP TRIGGER effect_receipts_causal_guard;
DROP TABLE agent_launch_expired_continuation_authorities;
DROP TABLE agent_launch_expired_continuation_subjects;
DROP TABLE agent_launch_reconciliation_receipts;
DROP TABLE agent_launch_reconciliation_attempts;
DROP TABLE agent_launch_reconciliation_jobs;
DROP TABLE agent_launch_reconciliation_authorities;
DELETE FROM schema_migrations WHERE version IN (40,41);
PRAGMA user_version=39`)
	mustV10Exec(t, database, causalGuard)
}
