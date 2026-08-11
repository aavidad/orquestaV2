package sqlite

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

func TestV23WizardGapsSnapshotRecoveryRejectsTamper(t *testing.T) {
	t.Run("coherent bytes and digest", func(t *testing.T) {
		system, request, result := seedWizardGapsSnapshotRecovery(t, "coherent")
		foreign, err := gaps.Evaluate(gaps.Input{Facts: gaps.Facts{Surface: gaps.SurfaceServerService}})
		sqliteTestNoError(t, err)
		encoded, digest, err := gaps.MarshalResultSnapshot(foreign)
		sqliteTestNoError(t, err)
		ref := "wizard-gaps-result-snapshot:" + canonicalFingerprint(
			"orquesta.wizard.gaps.result-snapshot-binding.v1",
			result.InputDurability.ReceiptRef,
			digest,
		)
		rewriteRecoveryTrigger(t, system.repository.db,
			"wizard_gaps_result_snapshots_immutable_update", func() {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_result_snapshots SET ref=?,snapshot_digest=?,snapshot_bytes=?
WHERE input_receipt_ref=?`, ref, digest, encoded, result.InputDurability.ReceiptRef)
			})
		requireWizardGapsSnapshotRecoveryError(t, system,
			"sqlite.recovery_v23_wizard_gaps_snapshot_invalid")
		assertWizardGapsSnapshotRestartRejects(t, system, request)
	})

	for _, test := range []struct {
		name, column, value string
	}{
		{name: "revision", column: "expected_revision", value: "2"},
		{name: "catalog", column: "evaluator_version", value: "'v2'"},
	} {
		t.Run(test.name, func(t *testing.T) {
			system, _, result := seedWizardGapsSnapshotRecovery(t, test.name)
			rewriteRecoveryTrigger(t, system.repository.db,
				"wizard_gaps_result_snapshots_immutable_update", func() {
					mustV10Exec(t, system.repository.db,
						"UPDATE wizard_gaps_result_snapshots SET "+test.column+"="+test.value+
							" WHERE input_receipt_ref=?", result.InputDurability.ReceiptRef)
				})
			requireWizardGapsSnapshotRecoveryError(t, system,
				"sqlite.recovery_v23_wizard_gaps_snapshot_binding_invalid")
		})
	}

	t.Run("legacy without snapshot", func(t *testing.T) {
		system, request, first := seedWizardGapsSnapshotRecovery(t, "legacy")
		rewriteRecoveryTrigger(t, system.repository.db,
			"wizard_gaps_result_snapshots_immutable_delete", func() {
				mustV10Exec(t, system.repository.db,
					`DELETE FROM wizard_gaps_result_snapshots WHERE input_receipt_ref=?`,
					first.InputDurability.ReceiptRef)
			})
		if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
			t.Fatalf("legacy recovery: %s", sqliteTestErrorChain(err))
		}
		sqliteTestNoError(t, system.repository.Close())
		system.repository = openSQLiteIntakeTestRepository(t, system.path)
		service, err := application.NewWizardGapsService(system.repository)
		sqliteTestNoError(t, err)
		replayed, err := service.ApplyWizardGaps(context.Background(), request)
		sqliteTestNoError(t, err)
		if replayed.EvaluationReplayExact || replayed.EvaluationSnapshot.Ref != "" ||
			!reflect.DeepEqual(replayed.Evaluation, first.Evaluation) {
			t.Fatalf("legacy replay snapshot=%+v exact=%t", replayed.EvaluationSnapshot,
				replayed.EvaluationReplayExact)
		}
	})
}

func seedWizardGapsSnapshotRecovery(
	t *testing.T,
	suffix string,
) (*sqliteIntakeTestSystem, application.ApplyWizardGapsRequest, application.ApplyWizardGapsResult) {
	t.Helper()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-snapshot-recovery-source:"+suffix)
	service, err := application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(t, system, "request:wizard-snapshot-recovery:"+suffix,
		created.Record.State.Revision(), intake.OriginForm)
	result, err := service.ApplyWizardGaps(context.Background(), request)
	sqliteTestNoError(t, err)
	return system, request, result
}

func requireWizardGapsSnapshotRecoveryError(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	want string,
) {
	t.Helper()
	_, _, err := validateRecoveryDatabase(context.Background(), system.repository.db)
	if err == nil || !recoveryErrorContains(err, want) {
		t.Fatalf("recovery err=%s want=%s", sqliteTestErrorChain(err), want)
	}
}

func assertWizardGapsSnapshotRestartRejects(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	request application.ApplyWizardGapsRequest,
) {
	t.Helper()
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	service, err := application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	if _, err := service.ApplyWizardGaps(context.Background(), request); err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "wizard_gaps_result_snapshot") {
		t.Fatalf("corrupt restart err=%s", sqliteTestErrorChain(err))
	}
}
