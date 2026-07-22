package sqlite

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
)

const v09RecoveryCrashExitCode = 86

var v09RecoveryCrashStages = []string{
	"after_first_backup_step",
	"after_backup_sync",
	"after_backup_publish",
	"after_restore_sync",
	"after_restore_publish",
}

// TestV09RecoverySurvivesProcessCrashAtEveryBoundary uses a real child
// process. os.Exit bypasses Backup.Finish, deferred cleanup and Close, so the
// parent proves restart recovery rather than ordinary error unwinding.
func TestV09RecoverySurvivesProcessCrashAtEveryBoundary(t *testing.T) {
	if raceEnabled {
		t.Skip("multiprocess crash matrix is covered by the normal gate; focal races cover V17")
	}
	for _, stage := range v09RecoveryCrashStages {
		stage := stage
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			databasePath := filepath.Join(root, "state", "orquesta.sqlite")
			backupRoot := filepath.Join(root, "backups")
			restoreRoot := filepath.Join(root, "restores")
			at := time.Date(2026, 7, 15, 4, 30, 0, 0, time.UTC)
			suffix := "v09-crash-" + strings.ReplaceAll(stage, "_", "-")

			repository := openV09CrashRepository(t, databasePath, at)
			created := authorizeRecoveryCreate(t, repository, newCreateFixture(
				t, suffix, "request:"+suffix, "fingerprint:"+suffix, "actor:v09-crash", "project:v09-crash",
			))
			if _, _, err := repository.CreateGoal(ctx, created); err != nil {
				t.Fatalf("seed crash fixture: %v", err)
			}

			var backupRef application.BackupRef
			if strings.HasPrefix(stage, "after_restore_") {
				stable, err := NewRecovery(RecoveryOptions{
					Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
					Now: func() time.Time { return at },
				})
				if err != nil {
					t.Fatalf("open stable recovery: %v", err)
				}
				receipt, createErr := stable.CreateBackup(ctx)
				closeErr := stable.Close()
				if createErr != nil || closeErr != nil {
					t.Fatalf("prepare stable backup: create=%v close=%v", createErr, closeErr)
				}
				backupRef = receipt.Ref
			}
			if err := repository.Close(); err != nil {
				t.Fatalf("close state before crash: %v", err)
			}

			command := exec.Command(
				os.Args[0], "-test.run=^TestV09RecoveryCrashProcessHelper$", "-test.count=1", "--",
				"v09-recovery-crash-helper", stage, databasePath, backupRoot, restoreRoot,
				backupRef.String(), at.Format(time.RFC3339Nano),
			)
			command.Env = []string{}
			output, crashErr := command.CombinedOutput()
			var exitError *exec.ExitError
			if !errors.As(crashErr, &exitError) || exitError.ExitCode() != v09RecoveryCrashExitCode {
				t.Fatalf("stage %s did not crash at failpoint: %v\n%s", stage, crashErr, output)
			}

			repository = openV09CrashRepository(t, databasePath, at)
			t.Cleanup(func() { _ = repository.Close() })
			recovered, err := NewRecovery(RecoveryOptions{
				Repository: repository, BackupRoot: backupRoot, RestoreRoot: restoreRoot,
				Now: func() time.Time { return at },
			})
			if err != nil {
				t.Fatalf("reopen recovery after %s: %v", stage, err)
			}
			t.Cleanup(func() { _ = recovered.Close() })
			assertV09NoRecoveryResidue(t, backupRoot, restoreRoot)

			if strings.HasPrefix(stage, "after_backup_") || stage == "after_first_backup_step" {
				receipt, err := recovered.CreateBackup(ctx)
				if err != nil {
					t.Fatalf("retry backup after %s: %v", stage, err)
				}
				if _, err := recovered.VerifyBackup(ctx, receipt.Ref); err != nil {
					t.Fatalf("verify recovered backup after %s: %v", stage, err)
				}
				assertV09CompleteBackupSet(t, recovered, backupRoot)
			} else {
				target, err := application.NewRecoveryTargetRef("recovery-target:" + suffix)
				sqliteTestNoError(t, err)
				if stage == "after_restore_sync" {
					if _, err := recovered.RestoreBackup(ctx, backupRef, target); err != nil {
						t.Fatalf("retry restore after prepublish crash: %v", err)
					}
				}
				targetPath, err := recovered.TargetPath(target)
				sqliteTestNoError(t, err)
				assertV09NoRecoveryResidue(t, backupRoot, restoreRoot)
				restored := openV09CrashRepository(t, targetPath, at)
				if _, err := restored.GetGoal(ctx, created.Goal.Ref()); err != nil {
					_ = restored.Close()
					t.Fatalf("complete restored state missing after %s: %v", stage, err)
				}
				if err := restored.Close(); err != nil {
					t.Fatalf("close restored state: %v", err)
				}
			}
			assertV09NoRecoveryResidue(t, backupRoot, restoreRoot)
		})
	}
}

// TestV09RecoveryCrashProcessHelper is entered only by the parent test above.
func TestV09RecoveryCrashProcessHelper(t *testing.T) {
	arguments := flag.Args()
	if len(arguments) == 0 || arguments[0] != "v09-recovery-crash-helper" {
		return
	}
	if len(arguments) != 7 {
		t.Fatalf("invalid crash helper arguments: %q", arguments)
	}
	stage, databasePath := arguments[1], arguments[2]
	backupRoot, restoreRoot := arguments[3], arguments[4]
	backupRefValue, timeValue := arguments[5], arguments[6]
	at, err := time.Parse(time.RFC3339Nano, timeValue)
	sqliteTestNoError(t, err)
	repository := openV09CrashRepository(t, databasePath, at)
	recovery, err := NewRecovery(RecoveryOptions{
		Repository:  repository,
		BackupRoot:  backupRoot,
		RestoreRoot: restoreRoot,
		Now:         func() time.Time { return at },
		Failpoint: func(current string) error {
			if current == stage {
				os.Exit(v09RecoveryCrashExitCode)
			}
			return nil
		},
	})
	sqliteTestNoError(t, err)
	if strings.HasPrefix(stage, "after_restore_") {
		backupRef, err := application.NewBackupRef(backupRefValue)
		sqliteTestNoError(t, err)
		target, err := application.NewRecoveryTargetRef(
			"recovery-target:v09-crash-" + strings.ReplaceAll(stage, "_", "-"),
		)
		sqliteTestNoError(t, err)
		_, err = recovery.RestoreBackup(context.Background(), backupRef, target)
	} else {
		_, err = recovery.CreateBackup(context.Background())
	}
	if err != nil {
		t.Fatalf("operation returned before crash failpoint %s: %v", stage, err)
	}
	t.Fatalf("configured crash failpoint %s was not reached", stage)
}

func openV09CrashRepository(t *testing.T, path string, at time.Time) *Repository {
	t.Helper()
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		Now: func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("open crash repository: %v", err)
	}
	return repository
}

func assertV09CompleteBackupSet(t *testing.T, recovery *Recovery, backupRoot string) {
	t.Helper()
	entries, err := os.ReadDir(backupRoot)
	if err != nil || len(entries) == 0 {
		t.Fatalf("complete backup set missing: entries=%v err=%v", entries, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		decoded, decodeErr := hex.DecodeString(name)
		if !entry.IsDir() || len(name) != 64 || decodeErr != nil || hex.EncodeToString(decoded) != name {
			t.Fatalf("non-final backup entry survived recovery: %s", name)
		}
		files, err := os.ReadDir(filepath.Join(backupRoot, name))
		if err != nil || len(files) != 2 || files[0].Name() == files[1].Name() {
			t.Fatalf("backup %s is incomplete: files=%v err=%v", name, files, err)
		}
		ref, err := application.NewBackupRef(backupRefPrefix + name)
		sqliteTestNoError(t, err)
		if _, err := recovery.VerifyBackup(context.Background(), ref); err != nil {
			t.Fatalf("published backup %s is not verifiable: %v", name, err)
		}
	}
}

func assertV09NoRecoveryResidue(t *testing.T, roots ...string) {
	t.Helper()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == root {
				return nil
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".next") || strings.HasSuffix(name, ".partial") ||
				strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, "-wal") ||
				strings.HasSuffix(name, "-shm") || strings.HasSuffix(name, "-journal") {
				return errors.New("recovery residue: " + path)
			}
			return nil
		})
		sqliteTestNoError(t, err)
	}
}
