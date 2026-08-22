package sqlite

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"sync"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestV39RuntimeAPIsRejectInvalidOrCanceledContextBeforeState(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "v39-context-order")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	if _, err := system.repository.Prepare(nil, authority); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("nil Prepare err=%v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := system.repository.Prepare(canceled, authority); err != context.Canceled {
		t.Fatalf("canceled Prepare err=%v", err)
	}
	key := authority.Key
	key.ActionFence = math.MaxInt64 + 1
	if _, err := system.repository.ResolveRuntime(context.Background(), key); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("overflow ResolveRuntime err=%v", err)
	}
	if _, err := system.repository.ResolveRuntime(canceled, authority.Key); err != context.Canceled {
		t.Fatalf("canceled ResolveRuntime err=%v", err)
	}
}

func TestV39RuntimeDigestsRollbackWithParentInsertAndRetryExactly(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "v39-parent-rollback")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	runtime := sqliteMicroVMHostLaunchRuntimeDigests(authority.Key, "1")
	mustV10Exec(t, system.repository.db, `CREATE TRIGGER test_v39_abort_host_parent
AFTER INSERT ON microvm_host_launch_authorities
BEGIN SELECT RAISE(ABORT,'test.v39_abort_host_parent'); END`)

	if got, err := system.repository.PrepareWithRuntime(context.Background(), authority, runtime); err == nil || got.Key != (ports.MicroVMHostLaunchAuthorityKey{}) {
		t.Fatalf("aborted preparation=%+v err=%v", got, err)
	}
	assertV39RuntimeAndParentCounts(t, system.repository, authority.Key, 0, 0)
	mustV10Exec(t, system.repository.db, `DROP TRIGGER test_v39_abort_host_parent`)
	if got, err := system.repository.PrepareWithRuntime(context.Background(), authority, runtime); err != nil || got.Key != authority.Key {
		t.Fatalf("retry preparation=%+v err=%v", got, err)
	}
	assertV39RuntimeAndParentCounts(t, system.repository, authority.Key, 1, 1)
}

func TestV39MigrationClassifiesLegacyAndClosesEpoch(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "v39-legacy-epoch")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
		t.Fatal(err)
	}
	downgradeV39RuntimeDigestsToV38(t, system.repository.db)
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	defer reopened.Close()

	var legacy, runtime, epochCount int
	sqliteTestNoError(t, reopened.db.QueryRow(`SELECT
EXISTS(SELECT 1 FROM microvm_host_launch_runtime_digest_legacy_exemptions WHERE execution_ref=? AND action_fence=?),
EXISTS(SELECT 1 FROM microvm_host_launch_runtime_digests WHERE execution_ref=? AND action_fence=?),
(SELECT legacy_authority_count FROM microvm_host_launch_runtime_digest_epoch WHERE singleton=1)`,
		authority.Key.RunRef.String(), authority.Key.ActionFence,
		authority.Key.RunRef.String(), authority.Key.ActionFence).Scan(&legacy, &runtime, &epochCount))
	if legacy != 1 || runtime != 0 || epochCount != 1 {
		t.Fatalf("legacy/runtime/epoch=%d/%d/%d", legacy, runtime, epochCount)
	}
	if _, err := reopened.ResolveRuntime(context.Background(), authority.Key); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("legacy runtime lookup err=%v", err)
	}
	if _, err := reopened.db.Exec(`INSERT INTO microvm_host_launch_runtime_digest_legacy_exemptions(execution_ref,action_fence) VALUES(?,?)`, authority.Key.RunRef.String(), authority.Key.ActionFence); err == nil {
		t.Fatal("legacy epoch accepted a later exemption")
	}
	digests := sqliteMicroVMHostLaunchRuntimeDigests(authority.Key, "4")
	if _, err := reopened.db.Exec(`INSERT INTO microvm_host_launch_runtime_digests(execution_ref,action_fence,kernel_sha256,initramfs_sha256,profile_sha256,plan_sha256,concession_sha256) VALUES(?,?,?,?,?,?,?)`,
		authority.Key.RunRef.String(), authority.Key.ActionFence, digests.KernelSHA256, digests.InitramfsSHA256, digests.ProfileSHA256, digests.PlanSHA256, digests.ConcessionSHA256); err == nil {
		t.Fatal("legacy launch accepted a runtime supplement")
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), reopened.db); err != nil {
		t.Fatalf("classified legacy recovery: %v", err)
	}
}

func TestV39RuntimeDigestsConcurrentExactReplayAndDivergence(t *testing.T) {
	for _, test := range []struct {
		name       string
		secondSeed string
		wantErrors int
	}{{"exact", "1", 0}, {"divergent", "4", 1}} {
		t.Run(test.name, func(t *testing.T) {
			system, attempt := seedV27AmbiguousLaunch(t, "v39-concurrent-"+test.name)
			authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
			bindSQLiteMicroVMHostLaunchSession(t, system, authority)
			values := [2]ports.MicroVMHostLaunchRuntimeDigestsV1{
				sqliteMicroVMHostLaunchRuntimeDigests(authority.Key, "1"),
				sqliteMicroVMHostLaunchRuntimeDigests(authority.Key, test.secondSeed),
			}
			start := make(chan struct{})
			var wg sync.WaitGroup
			errs := make([]error, 2)
			for index := range values {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					<-start
					_, errs[index] = system.repository.PrepareWithRuntime(context.Background(), authority, values[index])
				}(index)
			}
			close(start)
			wg.Wait()
			errorCount := 0
			for _, err := range errs {
				if err != nil {
					errorCount++
				}
			}
			if errorCount != test.wantErrors {
				t.Fatalf("errors=%v want count=%d", errs, test.wantErrors)
			}
			assertV39RuntimeAndParentCounts(t, system.repository, authority.Key, 1, 1)
		})
	}
}

func TestV39ResolveRuntimeSurvivesRestartAndRejectsCorruption(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "v39-resolve-restart")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	want := sqliteMicroVMHostLaunchRuntimeDigests(authority.Key, "1")
	if _, err := system.repository.PrepareWithRuntime(context.Background(), authority, want); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	if got, err := reopened.ResolveRuntime(context.Background(), authority.Key); err != nil || got != want {
		t.Fatalf("restart runtime=%+v err=%v", got, err)
	}
	connection, err := reopened.db.Conn(context.Background())
	sqliteTestNoError(t, err)
	defer connection.Close()
	_, err = connection.ExecContext(context.Background(), `DROP TRIGGER microvm_host_launch_runtime_digests_immutable_update`)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=ON`)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(context.Background(), `UPDATE microvm_host_launch_runtime_digests SET kernel_sha256='INVALID' WHERE execution_ref=? AND action_fence=?`, authority.Key.RunRef.String(), authority.Key.ActionFence)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=OFF`)
	sqliteTestNoError(t, err)
	if _, err := reopened.ResolveRuntime(context.Background(), authority.Key); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("corrupt runtime lookup err=%v", err)
	}
}

func TestV39RecoveryRejectsRuntimePartitionXORViolations(t *testing.T) {
	for _, mode := range []string{"none", "both"} {
		t.Run(mode, func(t *testing.T) {
			system, attempt := seedV27AmbiguousLaunch(t, "v39-xor-"+mode)
			authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
			bindSQLiteMicroVMHostLaunchSession(t, system, authority)
			digests := sqliteMicroVMHostLaunchRuntimeDigests(authority.Key, "1")
			if _, err := system.repository.PrepareWithRuntime(context.Background(), authority, digests); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "none":
				mustV10Exec(t, system.repository.db, `DROP TRIGGER microvm_host_launch_runtime_digests_immutable_delete`)
				_, err := system.repository.db.Exec(`DELETE FROM microvm_host_launch_runtime_digests WHERE execution_ref=? AND action_fence=?`, authority.Key.RunRef.String(), authority.Key.ActionFence)
				sqliteTestNoError(t, err)
			case "both":
				mustV10Exec(t, system.repository.db, `DROP TRIGGER microvm_host_launch_runtime_digest_legacy_exemptions_closed_insert;
DROP TRIGGER microvm_host_launch_runtime_digest_epoch_immutable_update`)
				_, err := system.repository.db.Exec(`INSERT INTO microvm_host_launch_runtime_digest_legacy_exemptions(execution_ref,action_fence) VALUES(?,?)`, authority.Key.RunRef.String(), authority.Key.ActionFence)
				sqliteTestNoError(t, err)
				_, err = system.repository.db.Exec(`UPDATE microvm_host_launch_runtime_digest_epoch SET legacy_authority_count=1 WHERE singleton=1`)
				sqliteTestNoError(t, err)
			}
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil {
				t.Fatalf("recovery accepted XOR violation %s", mode)
			}
		})
	}
}

func sqliteMicroVMHostLaunchRuntimeDigests(key ports.MicroVMHostLaunchAuthorityKey, seed string) ports.MicroVMHostLaunchRuntimeDigestsV1 {
	return ports.MicroVMHostLaunchRuntimeDigestsV1{
		Key: key, KernelSHA256: strings.Repeat(seed, 64), InitramfsSHA256: strings.Repeat("2", 64),
		ProfileSHA256: strings.Repeat("3", 64), PlanSHA256: strings.Repeat("4", 64), ConcessionSHA256: strings.Repeat("5", 64),
	}
}

func assertV39RuntimeAndParentCounts(t *testing.T, repository *Repository, key ports.MicroVMHostLaunchAuthorityKey, wantRuntime, wantParent int) {
	t.Helper()
	var runtime, parent int
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT
(SELECT COUNT(*) FROM microvm_host_launch_runtime_digests WHERE execution_ref=? AND action_fence=?),
(SELECT COUNT(*) FROM microvm_host_launch_authorities WHERE execution_ref=? AND action_fence=?)`, key.RunRef.String(), key.ActionFence, key.RunRef.String(), key.ActionFence).Scan(&runtime, &parent))
	if runtime != wantRuntime || parent != wantParent {
		t.Fatalf("runtime/parent=%d/%d want=%d/%d", runtime, parent, wantRuntime, wantParent)
	}
}

func downgradeV39RuntimeDigestsToV38(t *testing.T, database *sql.DB) {
	t.Helper()
	mustV10Exec(t, database, `DROP TRIGGER microvm_host_launch_authorities_runtime_digest_required;
DROP TABLE microvm_host_launch_runtime_digests;
DROP TABLE microvm_host_launch_runtime_digest_legacy_exemptions;
DROP TABLE microvm_host_launch_runtime_digest_epoch;
DELETE FROM schema_migrations WHERE version=39;
PRAGMA user_version=38`)
}
