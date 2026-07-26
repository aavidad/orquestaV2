package sqlite

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestV23DossierRecoveryAcceptsDurableContentAndHistoricalAuthorization(t *testing.T) {
	system := newSQLiteIntakeDossierTestSystem(t)
	_, _ = system.prepare(t, "request:intake-dossier:recovery-valid")
	_, err := system.repository.db.Exec(`
UPDATE project_memberships
SET revision=revision+1, status='revoked', revoked_by_ref=?, revoked_at=?
WHERE principal_ref=? AND project_ref=?`,
		system.principal.Ref.String(), system.now.Add(time.Minute).UnixNano(),
		system.principal.Ref.String(), system.project.String(),
	)
	sqliteTestNoError(t, err)

	transaction, err := system.repository.db.BeginTx(context.Background(), nil)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	if err := validateRecoveryV23Dossiers(context.Background(), transaction); err != nil {
		t.Fatalf("valid dossier after authorization revocation: %s", intakeTestErrorChain(err))
	}
}

func TestV23DossierRecoveryRejectsTamperedCanonicalSnapshot(t *testing.T) {
	system := newSQLiteIntakeDossierTestSystem(t)
	_, result := system.prepare(t, "request:intake-dossier:recovery-snapshot")
	rewriteRecoveryTrigger(
		t, system.repository.db, "intake_dossiers_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE intake_dossiers
SET snapshot_json=json_set(snapshot_json, '$.objective', 'alterado')
WHERE ref=?`, result.Record.Dossier.Ref())
		},
	)
	requireV23DossierRecoveryError(
		t, system, "sqlite.recovery_v23_intake_dossier_snapshot_invalid",
	)
}

func TestV23DossierRecoveryRejectsTamperedCausalBinding(t *testing.T) {
	system := newSQLiteIntakeDossierTestSystem(t)
	_, result := system.prepare(t, "request:intake-dossier:recovery-binding")
	rewriteRecoveryTrigger(
		t, system.repository.db, "intake_dossiers_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE intake_dossiers
SET plan_digest=state_digest
WHERE ref=?`, result.Record.Dossier.Ref())
		},
	)
	requireV23DossierRecoveryError(
		t, system, "sqlite.recovery_v23_intake_dossier_binding_invalid",
	)
}

func TestV23DossierRecoveryRejectsRewrittenGenerationFingerprint(t *testing.T) {
	system := newSQLiteIntakeDossierTestSystem(t)
	_, result := system.prepare(t, "request:intake-dossier:recovery-fingerprint")
	rewriteRecoveryTrigger(
		t, system.repository.db,
		"intake_dossier_generation_receipts_immutable_update",
		func() {
			mustV10Exec(t, system.repository.db, `
UPDATE intake_dossier_generation_receipts
SET request_fingerprint=?
WHERE ref=?`, strings.Repeat("0", 64), result.Record.Receipt.Ref)
		},
	)
	requireV23DossierRecoveryError(
		t, system, "sqlite.recovery_v23_intake_dossier_snapshot_invalid",
	)
}

func TestV23DossierBackupRestorePreservesLogicalContent(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeDossierTestSystem(t)
	_, result := system.prepare(t, "request:intake-dossier:backup")
	recovery, _, _ := newV09TestRecovery(
		t, system.repository, system.now.Add(time.Minute), nil,
	)
	backup, err := recovery.CreateBackup(ctx)
	sqliteTestNoError(t, err)
	if _, err := recovery.VerifyBackup(ctx, backup.Ref); err != nil {
		t.Fatalf("verify dossier backup: %v", err)
	}
	target, err := application.NewRecoveryTargetRef("recovery-target:v23-dossier")
	sqliteTestNoError(t, err)
	if _, err := recovery.RestoreBackup(ctx, backup.Ref, target); err != nil {
		t.Fatalf("restore dossier backup: %v", err)
	}
	targetPath, err := recovery.TargetPath(target)
	sqliteTestNoError(t, err)
	restored := openRawV10TestDatabase(t, targetPath)
	defer restored.Close()
	if _, _, err := validateRecoveryDatabase(ctx, restored); err != nil {
		t.Fatalf("restored dossier invalid: %s", intakeTestErrorChain(err))
	}
	record, err := readIntakeDossierRecordByReceipt(
		ctx, restored, result.Record.Receipt.Ref,
	)
	sqliteTestNoError(t, err)
	if record.Receipt != result.Record.Receipt ||
		!reflect.DeepEqual(
			application.SnapshotIntakeDossier(record.Dossier),
			application.SnapshotIntakeDossier(result.Record.Dossier),
		) {
		t.Fatal("backup restore changed immutable dossier")
	}
}

func requireV23DossierRecoveryError(
	t *testing.T,
	system *sqliteIntakeDossierTestSystem,
	code string,
) {
	t.Helper()
	if _, _, err := validateRecoveryDatabase(
		context.Background(), system.repository.db,
	); err == nil || !recoveryErrorContains(err, code) {
		t.Fatalf("invalid dossier passed recovery: %s", intakeTestErrorChain(err))
	}
}
