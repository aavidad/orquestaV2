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
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

func TestV09RecoveryRoundTripPreservesCausalTablesAndClaims(t *testing.T) {
	repository, _ := openTestRepository(t)
	running, at := createRunningV06Fixture(t, repository, "v09-roundtrip")
	claim := mustClaim(t, repository, "worker:v09-active", "claim:v09-active", at)
	if claim.Action.Kind != application.ActionObserveAgent {
		t.Fatalf("active claim kind = %s", claim.Action.Kind)
	}
	beforeGoal, err := repository.GetGoal(context.Background(), running.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	beforeTables := recoveryCausalTables(t, repository.db)

	recovery, backupRoot, _ := newV09TestRecovery(t, repository, at.Add(time.Minute), nil)
	receipt, err := recovery.CreateBackup(context.Background())
	if err != nil {
		t.Fatalf("create online backup: %v cause=%v", err, errors.Unwrap(err))
	}
	migrations, _ := loadMigrations()
	if receipt.SchemaRef != migrationSchemaRef(migrations) || !strings.HasPrefix(receipt.Ref.String(), backupRefPrefix) {
		t.Fatalf("backup identity = %+v", receipt)
	}
	manifest := readV09Manifest(t, backupRoot)
	if manifest.LogicalSHA256 == "" || manifest.LogicalSHA256 == manifest.PayloadSHA256 ||
		manifest.SchemaRef != receipt.SchemaRef {
		t.Fatalf("manifest semantic binding = %+v", manifest)
	}
	verified, err := recovery.VerifyBackup(context.Background(), receipt.Ref)
	if err != nil || verified.ManifestSHA256 != receipt.ManifestSHA256 {
		t.Fatalf("verify = %+v err=%v", verified, err)
	}
	target, _ := application.NewRecoveryTargetRef("recovery-target:v09-unit-roundtrip")
	if _, err := recovery.RestoreBackup(context.Background(), receipt.Ref, target); err != nil {
		t.Fatalf("restore: %v", err)
	}
	targetPath, _ := recovery.TargetPath(target)
	restored, err := Open(context.Background(), Options{
		Path: targetPath, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("open restored state: %v", err)
	}
	defer restored.Close()
	afterGoal, err := restored.GetGoal(context.Background(), running.Goal.Ref())
	if err != nil || !reflect.DeepEqual(beforeGoal, afterGoal) {
		t.Fatalf("restored Goal differs: err=%v\nbefore=%+v\nafter=%+v", err, beforeGoal, afterGoal)
	}
	afterTables := recoveryCausalTables(t, restored.db)
	if !reflect.DeepEqual(beforeTables, afterTables) {
		t.Fatalf("causal rows changed:\nbefore=%v\nafter=%v", beforeTables, afterTables)
	}
}

func TestV09RecoveryOnlineSnapshotContainsWholeConcurrentCommit(t *testing.T) {
	repository, _ := openTestRepository(t)
	initial := authorizeRecoveryCreate(t, repository, newCreateFixture(t, "v09-before", "request:v09-before", "fingerprint:v09-before", "actor:v09", "project:v09"))
	if _, _, err := repository.CreateGoal(context.Background(), initial); err != nil {
		t.Fatal(err)
	}
	concurrent := authorizeRecoveryCreate(t, repository, newCreateFixture(t, "v09-concurrent", "request:v09-concurrent", "fingerprint:v09-concurrent", "actor:v09", "project:v09"))
	var once sync.Once
	var concurrentErr error
	recovery, _, _ := newV09TestRecovery(t, repository, initial.Goal.CreatedAt(), func(stage string) error {
		if stage == "after_first_backup_step" {
			once.Do(func() {
				_, _, concurrentErr = repository.CreateGoal(context.Background(), concurrent)
			})
		}
		return concurrentErr
	})
	receipt, err := recovery.CreateBackup(context.Background())
	if err != nil || concurrentErr != nil {
		t.Fatalf("concurrent online backup: backup=%#v cause=%v writer=%#v", err, errors.Unwrap(err), concurrentErr)
	}
	target, _ := application.NewRecoveryTargetRef("recovery-target:v09-concurrent-unit")
	if _, err := recovery.RestoreBackup(context.Background(), receipt.Ref, target); err != nil {
		t.Fatal(err)
	}
	targetPath, _ := recovery.TargetPath(target)
	restored, err := Open(context.Background(), Options{
		Path: targetPath, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	status, err := restored.Status(context.Background(), concurrent.Goal.Project())
	if err != nil || status.Goals < 1 || status.Goals > 2 {
		t.Fatalf("snapshot status = %+v err=%v", status, err)
	}
	_, err = restored.GetGoal(context.Background(), concurrent.Goal.Ref())
	if status.Goals == 1 && !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("pre-commit snapshot contains partial Goal: %v", err)
	}
	if status.Goals == 2 && err != nil {
		t.Fatalf("post-commit snapshot lacks complete Goal: %v", err)
	}
}

func TestV09RecoveryFailpointsLeaveNoPartialPublication(t *testing.T) {
	for _, stage := range []string{
		"after_first_backup_step", "after_backup_sync", "after_backup_publish",
		"after_restore_sync", "after_restore_publish",
	} {
		t.Run(stage, func(t *testing.T) {
			repository, _ := openTestRepository(t)
			injected := errors.New("v09.injected")
			var stableReceipt application.BackupReceipt
			var backupRoot, restoreRoot string
			if strings.HasPrefix(stage, "after_restore_") {
				stable, backup, restore := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
				backupRoot, restoreRoot = backup, restore
				var err error
				stableReceipt, err = stable.CreateBackup(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				_ = stable.Close()
			}
			if backupRoot == "" {
				base := t.TempDir()
				backupRoot, restoreRoot = filepath.Join(base, "backups"), filepath.Join(base, "restores")
			}
			recovery, err := NewRecovery(RecoveryOptions{
				Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
				Failpoint: func(current string) error {
					if current == stage {
						return injected
					}
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(stage, "after_restore_") {
				target, _ := application.NewRecoveryTargetRef("recovery-target:v09-failpoint-unit")
				_, err = recovery.RestoreBackup(context.Background(), stableReceipt.Ref, target)
				if !errors.Is(err, injected) {
					t.Fatalf("restore failpoint err=%v", err)
				}
				targetPath, _ := recovery.TargetPath(target)
				if _, err := os.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("failed restore published target: %v", err)
				}
				assertNoRecoveryStages(t, restoreRoot)
			} else {
				_, err = recovery.CreateBackup(context.Background())
				if !errors.Is(err, injected) {
					t.Fatalf("backup failpoint err=%v", err)
				}
				entries, readErr := os.ReadDir(backupRoot)
				if readErr != nil || len(entries) != 0 {
					t.Fatalf("failed backup residues=%v err=%v", entries, readErr)
				}
			}
		})
	}
}

func TestV09RestoreNeverOverwritesTargetCreatedAtPublicationBoundary(t *testing.T) {
	repository, _ := openTestRepository(t)
	stable, backupRoot, restoreRoot := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
	receipt, err := stable.CreateBackup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = stable.Close()
	target, _ := application.NewRecoveryTargetRef("recovery-target:v09-race")
	targetName := recoveryTargetName(target)
	targetPath := filepath.Join(restoreRoot, targetName)
	sentinel := []byte("preexisting-race-winner")
	racing, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
		Failpoint: func(stage string) error {
			if stage == "after_restore_sync" {
				return os.WriteFile(targetPath, sentinel, 0o600)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := racing.RestoreBackup(context.Background(), receipt.Ref, target); err == nil {
		t.Fatal("restore replaced a target created at publication boundary")
	}
	content, err := os.ReadFile(targetPath)
	if err != nil || !reflect.DeepEqual(content, sentinel) {
		t.Fatalf("race winner changed: %q err=%v", content, err)
	}
	assertNoRecoveryStages(t, restoreRoot)
}

func TestV09RecoveryRejectsHardlinksUnsafeRootsAndOperationsAfterClose(t *testing.T) {
	repository, path := openTestRepository(t)
	recovery, backupRoot, _ := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
	receipt, err := recovery.CreateBackup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	payload := findRecoveryFile(t, backupRoot, recoveryPayloadName)
	linked := payload + ".linked"
	if err := os.Link(payload, linked); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.VerifyBackup(context.Background(), receipt.Ref); err == nil {
		t.Fatal("hardlinked payload verified")
	}
	if err := os.Remove(linked); err != nil {
		t.Fatal(err)
	}
	if err := recovery.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.CreateBackup(context.Background()); err == nil {
		t.Fatal("closed recovery accepted operation")
	}
	if err := recovery.Close(); err != nil {
		t.Fatal(err)
	}

	unsafe := filepath.Join(t.TempDir(), "unsafe")
	if err := os.Mkdir(unsafe, 0o755); err != nil {
		t.Fatal(err)
	}
	if created, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: unsafe, RestoreRoot: filepath.Join(t.TempDir(), "restore"),
	}); err == nil {
		_ = created.Close()
		t.Fatal("unsafe root mode accepted")
	}
	if created, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: filepath.Join(t.TempDir(), "backup"), RestoreRoot: filepath.Dir(path),
	}); err == nil {
		_ = created.Close()
		t.Fatal("active state namespace accepted")
	}
}

func TestV09SchemaRefChangesWithMigrationChecksumSet(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	baseline := migrationSchemaRef(migrations)
	changed := append([]migration(nil), migrations...)
	changed[len(changed)-1].checksum = "sha256:" + strings.Repeat("0", 64)
	if baseline == migrationSchemaRef(changed) || !strings.HasPrefix(baseline, schemaRefPrefix) {
		t.Fatalf("schema ref is not bound to migration checksums: %s", baseline)
	}
}

func TestV09RecoveryValidationRejectsReceiptAndActiveFenceDivergence(t *testing.T) {
	t.Run("virgin_action_without_fence", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		state := authorizeRecoveryCreate(t, repository, newCreateFixture(t, "v09-virgin", "request:v09-virgin", "fingerprint:v09-virgin", "actor:v09", "project:v09"))
		if _, _, err := repository.CreateGoal(context.Background(), state); err != nil {
			t.Fatal(err)
		}
		var fences int
		if err := repository.db.QueryRow("SELECT COUNT(*) FROM work_item_fences").Scan(&fences); err != nil || fences != 0 {
			t.Fatalf("virgin fences=%d err=%v", fences, err)
		}
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
			t.Fatalf("virgin unclaimed action rejected: %v", err)
		}
	})
	t.Run("receipt_outbox_binding", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		_, _ = createRunningV06Fixture(t, repository, "v09-receipt-tamper")
		var guardSQL, immutableSQL string
		if err := repository.db.QueryRow(`SELECT sql FROM sqlite_schema WHERE name = 'action_consumption_receipt_guard'`).Scan(&guardSQL); err != nil {
			t.Fatal(err)
		}
		if err := repository.db.QueryRow(`SELECT sql FROM sqlite_schema WHERE name = 'action_consumption_receipts_immutable_update'`).Scan(&immutableSQL); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.db.Exec(`
DROP TRIGGER action_consumption_receipts_immutable_update;
DROP TRIGGER action_consumption_receipt_guard;
UPDATE action_consumption_receipts SET worker_ref = 'worker:tampered';`); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.db.Exec(guardSQL); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.db.Exec(immutableSQL); err != nil {
			t.Fatal(err)
		}
		actualSchema, err := schemaInventoryDigest(context.Background(), repository.db)
		if err != nil {
			t.Fatal(err)
		}
		expectedSchema, err := canonicalSchemaInventoryDigest(recoverySchemaV15)
		if err != nil || actualSchema != expectedSchema {
			t.Fatalf("test failed to restore canonical schema: actual=%s expected=%s err=%v", actualSchema, expectedSchema, err)
		}
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil {
			t.Fatal("receipt diverging from terminal outbox action validated")
		}
	})
	t.Run("active_claim_fence", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		state := authorizeRecoveryCreate(t, repository, newCreateFixture(t, "v09-fence-tamper", "request:v09-fence-tamper", "fingerprint:v09-fence-tamper", "actor:v09", "project:v09"))
		if _, _, err := repository.CreateGoal(context.Background(), state); err != nil {
			t.Fatal(err)
		}
		claim := mustClaim(t, repository, "worker:v09-fence", "claim:v09-fence", state.Goal.CreatedAt())
		if _, err := repository.db.Exec(`
UPDATE work_item_fences SET fence = fence + 1
WHERE goal_ref = ? AND work_item_ref = ?`, claim.Action.GoalRef.String(), claim.Action.WorkItemRef.String()); err != nil {
			t.Fatal(err)
		}
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil {
			t.Fatal("actively claimed action with superseded fence validated")
		}
	})
	t.Run("active_claim_missing_fence", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		state := authorizeRecoveryCreate(t, repository, newCreateFixture(t, "v09-fence-missing", "request:v09-fence-missing", "fingerprint:v09-fence-missing", "actor:v09", "project:v09"))
		if _, _, err := repository.CreateGoal(context.Background(), state); err != nil {
			t.Fatal(err)
		}
		claim := mustClaim(t, repository, "worker:v09-fence-missing", "claim:v09-fence-missing", state.Goal.CreatedAt())
		if _, err := repository.db.Exec(`
DELETE FROM work_item_fences
WHERE goal_ref = ? AND work_item_ref = ?`, claim.Action.GoalRef.String(), claim.Action.WorkItemRef.String()); err != nil {
			t.Fatal(err)
		}
		if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil {
			t.Fatal("actively claimed action without fence row validated")
		}
	})
}

func TestV09RecoveryRejectsOutboxGenerationDivergenceFromExecution(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newCreateFixture(
		t,
		"v09-outbox-generation-tamper",
		"request:v09-outbox-generation-tamper",
		"fingerprint:v09-outbox-generation-tamper",
		"actor:v09",
		"project:v09",
	)
	state = authorizeRecoveryCreate(t, repository, state)
	if _, _, err := repository.CreateGoal(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	var immutableSQL string
	if err := repository.db.QueryRow(`
SELECT sql FROM sqlite_schema WHERE name = 'outbox_identity_immutable'`).Scan(&immutableSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec(`DROP TRIGGER outbox_identity_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec(`
UPDATE outbox SET plan_generation = plan_generation + 1 WHERE goal_ref = ?`,
		state.Goal.Ref().String(),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec(immutableSQL); err != nil {
		t.Fatal(err)
	}
	actualSchema, err := schemaInventoryDigest(context.Background(), repository.db)
	if err != nil {
		t.Fatal(err)
	}
	expectedSchema, err := canonicalSchemaInventoryDigest(recoverySchemaV15)
	if err != nil || actualSchema != expectedSchema {
		t.Fatalf("test failed to restore canonical schema: actual=%s expected=%s err=%v", actualSchema, expectedSchema, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil {
		t.Fatal("outbox plan generation diverging from its execution validated")
	}
}

func TestV09RecoveryRejectsStaleActiveObserveGeneration(t *testing.T) {
	repository, _ := openTestRepository(t)
	_, _ = createRunningV06Fixture(t, repository, "v09-stale-observe-generation")
	var immutableSQL string
	if err := repository.db.QueryRow(`
SELECT sql FROM sqlite_schema WHERE name = 'outbox_identity_immutable'`).Scan(&immutableSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec(`DROP TRIGGER outbox_identity_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec(`
UPDATE outbox
SET work_item_generation = work_item_generation - 1
WHERE kind = 'observe_agent' AND completed_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec(immutableSQL); err != nil {
		t.Fatal(err)
	}
	actualSchema, err := schemaInventoryDigest(context.Background(), repository.db)
	if err != nil {
		t.Fatal(err)
	}
	expectedSchema, err := canonicalSchemaInventoryDigest(recoverySchemaV15)
	if err != nil || actualSchema != expectedSchema {
		t.Fatalf("test failed to restore canonical schema: actual=%s expected=%s err=%v", actualSchema, expectedSchema, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil {
		t.Fatal("active observation with stale work item generation validated")
	}
}

func TestV09RecoveryBacksUpDispatchingLaunchBeforeAndAfterRequeue(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newCreateFixture(
		t,
		"v09-dispatching-requeue",
		"request:v09-dispatching-requeue",
		"fingerprint:v09-dispatching-requeue",
		"actor:v09",
		"project:v09",
	)
	state = authorizeRecoveryCreate(t, repository, state)
	if _, _, err := repository.CreateGoal(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	claim := mustClaim(t, repository, "worker:v09-dispatching", "claim:v09-dispatching", state.Goal.CreatedAt())
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item := onlyItem(t, record.Goal)
	preparedAt := state.Goal.CreatedAt().Add(time.Second)
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, preparedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	repository.now = func() time.Time { return preparedAt }
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: preparedExecution, OperationAt: preparedAt,
		Event: application.EventRecord{
			Ref: "event:v09-dispatching-requeue", Kind: "execution.dispatching",
			GoalRef: preparedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref,
			OccurredAt: preparedAt,
		},
	}); err != nil {
		t.Fatal(err)
	}
	recovery, _, _ := newV09TestRecovery(t, repository, preparedAt, nil)
	if _, err := recovery.CreateBackup(context.Background()); err != nil {
		t.Fatalf("backup during durable launch preparation: %v", err)
	}
	if err := repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: claim, Execution: preparedExecution,
		AvailableAt: preparedAt.Add(time.Second), OperationAt: preparedAt,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.CreateBackup(context.Background()); err != nil {
		t.Fatalf("backup after dispatching launch requeue: %v", err)
	}
}

func TestV09RecoveryBacksUpActiveReplacementLaunch(t *testing.T) {
	repository, _ := openTestRepository(t)
	replacement, _ := buildReplacementState(t, repository, "v09-active-replacement")
	if err := repository.RecordExecutionReplaced(context.Background(), replacement); err != nil {
		t.Fatal(err)
	}
	recovery, _, _ := newV09TestRecovery(t, repository, replacement.OperationAt, nil)
	if _, err := recovery.CreateBackup(context.Background()); err != nil {
		t.Fatalf("backup with active replacement launch: %v", err)
	}
}

func TestV09RecoveryRejectsSchemaDriftDespiteValidMigrationLedger(t *testing.T) {
	repository, _ := openTestRepository(t)
	recovery, backupRoot, _ := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
	if _, err := repository.db.Exec(`
DROP TRIGGER action_consumption_receipt_guard;
DROP INDEX outbox_claimable_idx;`); err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.CreateBackup(context.Background()); err == nil {
		t.Fatal("backup asserted canonical SchemaRef after trigger/index drift")
	}
	entries, err := os.ReadDir(backupRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("schema-drift backup published residues=%v err=%v", entries, err)
	}
}

func TestV09RecoveryRejectsZeroClockSpecialModesAndSymlinkComponents(t *testing.T) {
	repository, _ := openTestRepository(t)
	zero, _, _ := newV09TestRecovery(t, repository, time.Time{}, nil)
	if _, err := zero.CreateBackup(context.Background()); err == nil {
		t.Fatal("zero recovery clock accepted")
	}

	base := t.TempDir()
	backupRoot := filepath.Join(base, "special-backup")
	restoreRoot := filepath.Join(base, "restore")
	if err := os.Mkdir(backupRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(backupRoot, os.ModeSetuid|0o700); err != nil {
		t.Fatal(err)
	}
	if created, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
	}); err == nil {
		_ = created.Close()
		t.Fatal("special-mode recovery root accepted")
	}

	realRoot := filepath.Join(base, "real")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	linkedRoot := filepath.Join(base, "linked")
	if err := os.Symlink(realRoot, linkedRoot); err != nil {
		t.Fatal(err)
	}
	if created, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: linkedRoot, RestoreRoot: filepath.Join(base, "other-restore"),
	}); err == nil {
		_ = created.Close()
		t.Fatal("symlink recovery component accepted")
	}
}

func TestV09RecoveryRootLocksAndOwnedStageCleanup(t *testing.T) {
	repository, _ := openTestRepository(t)
	first, backupRoot, restoreRoot := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
	if second, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
	}); err == nil {
		_ = second.Close()
		t.Fatal("second recovery owner acquired locked roots")
	}
	backupStage := filepath.Join(backupRoot, ".backup-1-1.next")
	if err := os.Mkdir(backupStage, 0o700); err != nil {
		t.Fatal(err)
	}
	restoreStage := filepath.Join(restoreRoot, "."+strings.Repeat("a", 64)+".partial")
	if err := os.WriteFile(restoreStage, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(backupRoot, ".backup-operator-note.next")
	if err := os.WriteFile(unknown, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for _, stale := range []string{backupStage, restoreStage} {
		if _, err := os.Lstat(stale); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("owned recovery stage survived reopen: %s err=%v", stale, err)
		}
	}
	if content, err := os.ReadFile(unknown); err != nil || string(content) != "preserve" {
		t.Fatalf("non-owned file removed: %q err=%v", content, err)
	}
}

func TestV09RecoveryRejectsRootPathReplacementAfterLock(t *testing.T) {
	t.Run("renamed_backup_root_replaced_by_directory", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		recovery, backupRoot, _ := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
		lockedRoot := backupRoot + ".locked"
		if err := os.Rename(backupRoot, lockedRoot); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(backupRoot, 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := recovery.CreateBackup(context.Background()); err == nil {
			t.Fatal("replacement backup namespace accepted after root lock")
		}
		entries, err := os.ReadDir(backupRoot)
		if err != nil || len(entries) != 0 {
			t.Fatalf("replacement backup namespace touched: entries=%v err=%v", entries, err)
		}
	})

	t.Run("renamed_restore_root_replaced_by_symlink", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		recovery, _, restoreRoot := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
		receipt, err := recovery.CreateBackup(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		lockedRoot := restoreRoot + ".locked"
		if err := os.Rename(restoreRoot, lockedRoot); err != nil {
			t.Fatal(err)
		}
		substitute := filepath.Join(filepath.Dir(restoreRoot), "substitute")
		if err := os.Mkdir(substitute, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(substitute, restoreRoot); err != nil {
			t.Fatal(err)
		}
		target, _ := application.NewRecoveryTargetRef("recovery-target:v09-replaced-root")
		if _, err := recovery.RestoreBackup(context.Background(), receipt.Ref, target); err == nil {
			t.Fatal("symlink restore namespace accepted after root lock")
		}
		entries, err := os.ReadDir(substitute)
		if err != nil || len(entries) != 0 {
			t.Fatalf("symlink substitute namespace touched: entries=%v err=%v", entries, err)
		}
		if _, err := recovery.TargetPath(target); err == nil {
			t.Fatal("TargetPath exposed substituted restore namespace")
		}
	})
}

func TestV09RecoveryUnsupportedPlatformsRemainExplicitAndBuildable(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	common := read("recovery_files.go")
	if strings.Contains(common, "syscall.Stat_t") {
		t.Fatal("common recovery files retain Linux-only file metadata")
	}
	linux := read("recovery_fileinfo_linux.go")
	for _, required := range []string{
		"//go:build linux", "func validateOwner", "func linkCount", "*syscall.Stat_t",
	} {
		if !strings.Contains(linux, required) {
			t.Fatalf("Linux file metadata contract missing %q", required)
		}
	}
	unsupported := read("recovery_fileinfo_unsupported.go")
	for _, required := range []string{
		"//go:build !linux", "sqlite.recovery_owner_metadata_unsupported", "return 0, false",
	} {
		if !strings.Contains(unsupported, required) {
			t.Fatalf("unsupported file metadata contract missing %q", required)
		}
	}
	constructor := read("recovery.go")
	support := strings.Index(constructor, "recoveryRootLockSupportError()")
	preflight := strings.Index(constructor, "preflightRecoveryRoot(options.BackupRoot)")
	create := strings.Index(constructor, "createPrivateSQLiteDirectoryChain(backupRoot)")
	if support < 0 || preflight < 0 || create < 0 || support > preflight || support > create {
		t.Fatalf("unsupported guard order invalid: support=%d preflight=%d create=%d", support, preflight, create)
	}
	lockUnsupported := read("recovery_lock_unsupported.go")
	if !strings.Contains(lockUnsupported, "//go:build !linux") ||
		!strings.Contains(lockUnsupported, "sqlite.recovery_root_lock_unsupported") {
		t.Fatal("unsupported root-lock contract is not explicit")
	}
}

func TestV09RecoveryRejectsRootReplacementBetweenVerificationAndIO(t *testing.T) {
	t.Run("backup_stage_collision_in_substitute", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		base := t.TempDir()
		backupRoot := filepath.Join(base, "backups")
		restoreRoot := filepath.Join(base, "restores")
		lockedRoot := backupRoot + ".locked"
		now := time.Unix(1_720_000_000, 123).UTC()
		stageName := ".backup-" + fmt.Sprint(now.UnixNano()) + "-1.next"
		sentinel := []byte("substitute-stage-must-remain-untouched")
		var substituteBefore map[string]v09RecoveryTreeEntry
		swapped := false
		reachedBackupStep := false
		recovery, err := NewRecovery(RecoveryOptions{
			Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
			Now: func() time.Time { return now },
			Failpoint: func(stage string) error {
				switch stage {
				case "after_recovery_root_verify":
					if swapped {
						return nil
					}
					swapped = true
					if err := os.Rename(backupRoot, lockedRoot); err != nil {
						return err
					}
					if err := os.MkdirAll(filepath.Join(backupRoot, stageName), 0o700); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(backupRoot, stageName, "sentinel"), sentinel, 0o600); err != nil {
						return err
					}
					var err error
					substituteBefore, err = snapshotV09RecoveryTree(backupRoot)
					return err
				case "after_first_backup_step":
					reachedBackupStep = true
				}
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer recovery.Close()
		if _, err := recovery.CreateBackup(context.Background()); err == nil {
			t.Fatal("backup accepted root replaced after initial verification")
		}
		if !swapped {
			t.Fatal("root-replacement failpoint did not run")
		}
		if !reachedBackupStep {
			t.Fatal("backup did not continue through locked root after pathname replacement")
		}
		after, err := snapshotV09RecoveryTree(backupRoot)
		if err != nil || !reflect.DeepEqual(after, substituteBefore) {
			t.Fatalf("substitute backup root changed: err=%v\nbefore=%v\nafter=%v", err, substituteBefore, after)
		}
		assertV09DirectoryEmpty(t, lockedRoot)
	})

	t.Run("verify_rejects_byte_identical_invalid_mode_substitute", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		stable, backupRoot, restoreRoot := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
		receipt, err := stable.CreateBackup(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if err := stable.Close(); err != nil {
			t.Fatal(err)
		}
		before, err := snapshotV09RecoveryTree(backupRoot)
		if err != nil {
			t.Fatal(err)
		}
		lockedRoot := backupRoot + ".locked"
		swapped := false
		reachedBackupRead := false
		var substituteBefore map[string]v09RecoveryTreeEntry
		digest := strings.TrimPrefix(receipt.Ref.String(), backupRefPrefix)
		recovery, err := NewRecovery(RecoveryOptions{
			Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
			Failpoint: func(stage string) error {
				switch stage {
				case "after_recovery_root_verify":
					if swapped {
						return nil
					}
					swapped = true
					if err := os.Rename(backupRoot, lockedRoot); err != nil {
						return err
					}
					if err := cloneV09RecoveryTree(lockedRoot, backupRoot); err != nil {
						return err
					}
					if err := os.Chmod(filepath.Join(backupRoot, digest, recoveryPayloadName), 0o400); err != nil {
						return err
					}
					var err error
					substituteBefore, err = snapshotV09RecoveryTree(backupRoot)
					return err
				case "after_backup_read":
					reachedBackupRead = true
				}
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer recovery.Close()
		if _, err := recovery.VerifyBackup(context.Background(), receipt.Ref); err == nil {
			t.Fatal("verification accepted byte-identical replacement root")
		}
		if !reachedBackupRead {
			t.Fatal("verification reopened invalid substitute instead of reading locked root")
		}
		after, err := snapshotV09RecoveryTree(backupRoot)
		if err != nil || !reflect.DeepEqual(after, substituteBefore) {
			t.Fatalf("substitute backup tree changed: err=%v\nbefore=%v\nafter=%v", err, substituteBefore, after)
		}
		after, err = snapshotV09RecoveryTree(lockedRoot)
		if err != nil || !reflect.DeepEqual(after, before) {
			t.Fatalf("locked backup tree changed: err=%v\nbefore=%v\nafter=%v", err, before, after)
		}
	})

	t.Run("restore_stage_collision_in_substitute", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		stable, backupRoot, restoreRoot := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
		receipt, err := stable.CreateBackup(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if err := stable.Close(); err != nil {
			t.Fatal(err)
		}
		lockedRoot := restoreRoot + ".locked"
		target, _ := application.NewRecoveryTargetRef("recovery-target:v09-root-swap-sentinel")
		stageName := "." + strings.TrimSuffix(recoveryTargetName(target), ".sqlite") + ".partial"
		sentinel := []byte("root-substitute-must-remain-untouched")
		var substituteBefore map[string]v09RecoveryTreeEntry
		swapped := false
		reachedRestoreSync := false
		recovery, err := NewRecovery(RecoveryOptions{
			Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
			Failpoint: func(stage string) error {
				switch stage {
				case "after_recovery_root_verify":
					if swapped {
						return nil
					}
					swapped = true
					if err := os.Rename(restoreRoot, lockedRoot); err != nil {
						return err
					}
					if err := os.Mkdir(restoreRoot, 0o700); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(restoreRoot, stageName), sentinel, 0o600); err != nil {
						return err
					}
					var err error
					substituteBefore, err = snapshotV09RecoveryTree(restoreRoot)
					return err
				case "after_restore_sync":
					reachedRestoreSync = true
				}
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer recovery.Close()
		_, restoreErr := recovery.RestoreBackup(context.Background(), receipt.Ref, target)
		if restoreErr == nil {
			t.Fatal("restore accepted root replaced after initial verification")
		}
		if !reachedRestoreSync {
			t.Fatalf("restore reopened colliding substitute instead of using locked root: %v cause=%v", restoreErr, errors.Unwrap(restoreErr))
		}
		after, err := snapshotV09RecoveryTree(restoreRoot)
		if err != nil || !reflect.DeepEqual(after, substituteBefore) {
			t.Fatalf("replacement restore root changed: err=%v\nbefore=%v\nafter=%v", err, substituteBefore, after)
		}
		assertV09DirectoryEmpty(t, lockedRoot)
	})
}

func TestV09RestoreRetainsInspectedPayloadHandleAcrossRootSwap(t *testing.T) {
	repository, _ := openTestRepository(t)
	stable, backupRoot, restoreRoot := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
	receipt, err := stable.CreateBackup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := stable.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotV09RecoveryTree(backupRoot)
	if err != nil {
		t.Fatal(err)
	}
	lockedRoot := backupRoot + ".locked"
	swapped := false
	reachedRestoreSync := false
	recovery, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
		Failpoint: func(stage string) error {
			switch stage {
			case "after_backup_inspection":
				if swapped {
					return nil
				}
				swapped = true
				if err := os.Rename(backupRoot, lockedRoot); err != nil {
					return err
				}
				return os.Mkdir(backupRoot, 0o700)
			case "after_restore_sync":
				reachedRestoreSync = true
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer recovery.Close()
	target, _ := application.NewRecoveryTargetRef("recovery-target:v09-inspected-handle")
	_, restoreErr := recovery.RestoreBackup(context.Background(), receipt.Ref, target)
	if restoreErr == nil {
		t.Fatal("restore accepted backup root replaced after inspection")
	}
	if !reachedRestoreSync {
		t.Fatalf("restore reopened substitute instead of copying retained inspected payload handle: %v cause=%v", restoreErr, errors.Unwrap(restoreErr))
	}
	assertV09DirectoryEmpty(t, backupRoot)
	assertV09DirectoryEmpty(t, restoreRoot)
	after, err := snapshotV09RecoveryTree(lockedRoot)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("inspected backup changed: err=%v\nbefore=%v\nafter=%v", err, before, after)
	}
}

type v09RecoveryTreeEntry struct {
	Mode    os.FileMode
	Content []byte
}

func snapshotV09RecoveryTree(root string) (map[string]v09RecoveryTreeEntry, error) {
	result := make(map[string]v09RecoveryTreeEntry)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		record := v09RecoveryTreeEntry{Mode: info.Mode()}
		if info.Mode().IsRegular() {
			record.Content, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		result[relative] = record
		return nil
	})
	return result, err
}

func cloneV09RecoveryTree(sourceRoot, destinationRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			return os.Mkdir(destination, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported recovery tree entry %s", relative)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, info.Mode().Perm())
	})
}

func assertV09DirectoryEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("directory not empty: root=%s entries=%v err=%v", root, entries, err)
	}
}

func newV09TestRecovery(
	t *testing.T,
	repository *Repository,
	now time.Time,
	failpoint func(string) error,
) (*Recovery, string, string) {
	t.Helper()
	base := t.TempDir()
	backupRoot, restoreRoot := filepath.Join(base, "backups"), filepath.Join(base, "restores")
	recovery, err := NewRecovery(RecoveryOptions{
		Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
		Now: func() time.Time { return now }, Failpoint: failpoint,
	})
	if err != nil {
		t.Fatalf("new recovery: %v", err)
	}
	t.Cleanup(func() { _ = recovery.Close() })
	return recovery, backupRoot, restoreRoot
}

func authorizeRecoveryCreate(
	t *testing.T,
	repository *Repository,
	state application.CreateGoalState,
) application.CreateGoalState {
	t.Helper()
	principal := testPrincipal(
		t, state.Goal.Actor().String(), state.Goal.Actor().String(), identity.PrincipalKindHuman,
	)
	provisionTestAccess(
		t, repository, principal, state.Goal.Project(), identity.RoleProjectOwner, state.Goal.CreatedAt(),
	)
	state.RequestedBy = principal.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, principal, state.Goal.Project(), identity.PermissionGoalsCreate,
		state.Goal.Project().String(), "authorization:"+state.RequestRef, state.Goal.CreatedAt(),
	)
	return state
}

func recoveryCausalTables(t *testing.T, database *sql.DB) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	for _, table := range []string{"events", "outbox", "action_consumption_receipts", "work_item_fences"} {
		rows, err := database.Query("SELECT * FROM " + quoteSQLiteIdentifier(table))
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for index := range values {
				destinations[index] = &values[index]
			}
			if err := rows.Scan(destinations...); err != nil {
				t.Fatal(err)
			}
			result[table] = append(result[table], fmt.Sprint(values))
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		sort.Strings(result[table])
	}
	return result
}

func readV09Manifest(t *testing.T, root string) backupManifest {
	t.Helper()
	path := findRecoveryFile(t, root, recoveryManifestName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodeCanonicalManifest(content)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func findRecoveryFile(t *testing.T, root, name string) string {
	t.Helper()
	var matches []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == name {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil || len(matches) != 1 {
		t.Fatalf("find %s: matches=%v err=%v", name, matches, err)
	}
	return matches[0]
}

func assertNoRecoveryStages(t *testing.T, root string) {
	t.Helper()
	var stages []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && (strings.HasSuffix(path, ".partial") || strings.HasSuffix(path, ".next")) {
			stages = append(stages, path)
		}
		return err
	})
	if len(stages) != 0 {
		t.Fatalf("recovery stages remain: %v", stages)
	}
}
