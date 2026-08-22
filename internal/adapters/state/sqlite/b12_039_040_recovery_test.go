package sqlite

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/application"
)

// V40 is a read-only projection, not a schema migration. Backup/restore must
// preserve the V32 authority, V39 runtime supplement, attempt and accepted
// receipt from which the same authority is derived after restart.
func TestB12BackupRestoreRecomputesDerivedV40Authority(t *testing.T) {
	system, claim, attempt, authority, runtime := seedV40PreparedLaunch(t, "backup-restore")
	receipt := receiptV40Launch(t, system, claim, attempt)
	want := v40ExpectedHistoricalAuthority(attempt, receipt, authority, runtime)

	now := time.Date(2026, 8, 22, 3, 0, 0, 0, time.UTC)
	recovery, _, _ := newV09TestRecovery(t, system.repository, now, nil)
	backup, err := recovery.CreateBackup(context.Background())
	if err != nil {
		t.Fatalf("create B12 backup: %s", sqliteTestErrorChain(err))
	}
	targetRef, err := application.NewRecoveryTargetRef("recovery-target:b12-derived-v40")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovery.RestoreBackup(context.Background(), backup.Ref, targetRef); err != nil {
		t.Fatalf("restore B12 backup: %s", sqliteTestErrorChain(err))
	}
	targetPath, err := recovery.TargetPath(targetRef)
	if err != nil {
		t.Fatal(err)
	}
	restored := openSQLiteV15Repository(t, targetPath, system.clock.Now)
	defer restored.Close()

	got, err := restored.ResolveAgentHistoricalRuntimeAuthority(context.Background(), want.Key)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("restored derived authority=%+v want=%+v err=%v", got, want, err)
	}
	var version int
	if err := restored.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 39 {
		t.Fatalf("derived V40 created schema version %d, want 39", version)
	}
}
