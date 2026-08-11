package sqlite

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsSnapshotSQLiteExactReplayAndConflict(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-snapshot-source")
	service, err := application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(t, system, "request:wizard-snapshot-exact",
		created.Record.State.Revision(), intake.OriginForm)
	first, err := service.ApplyWizardGaps(ctx, request)
	sqliteTestNoError(t, err)
	if !first.EvaluationReplayExact || len(first.EvaluationSnapshot.Bytes) == 0 {
		t.Fatalf("first snapshot=%+v exact=%t", first.EvaluationSnapshot, first.EvaluationReplayExact)
	}
	var storedRef, storedDigest string
	var storedBytes []byte
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT ref,snapshot_digest,snapshot_bytes FROM wizard_gaps_result_snapshots
WHERE input_receipt_ref=?`, first.InputDurability.ReceiptRef).Scan(
		&storedRef, &storedDigest, &storedBytes,
	))
	if storedRef != first.EvaluationSnapshot.Ref || storedDigest != first.EvaluationSnapshot.Digest ||
		!bytes.Equal(storedBytes, first.EvaluationSnapshot.Bytes) {
		t.Fatalf("stored=%q/%q/%d result=%+v", storedRef, storedDigest, len(storedBytes), first.EvaluationSnapshot)
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	service, err = application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	replayed, err := service.ApplyWizardGaps(ctx, request)
	sqliteTestNoError(t, err)
	if replayed.Changed || !replayed.EvaluationReplayExact ||
		!reflect.DeepEqual(replayed.EvaluationSnapshot, first.EvaluationSnapshot) ||
		!reflect.DeepEqual(replayed.Evaluation, first.Evaluation) {
		t.Fatalf("first=%+v replayed=%+v", first, replayed)
	}

	divergent := request
	divergent.Facts.Surface = gaps.SurfaceServerService
	if _, err := service.ApplyWizardGaps(ctx, divergent); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent replay err=%v", err)
	}
	var snapshots int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM wizard_gaps_result_snapshots`,
	).Scan(&snapshots))
	if snapshots != 1 {
		t.Fatalf("snapshots=%d", snapshots)
	}
}

func TestWizardGapsSnapshotFailureRollsBackReceiptAndMutation(t *testing.T) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-snapshot-rollback-source")
	store := &invalidSnapshotStore{Repository: system.repository}
	service, err := application.NewWizardGapsService(store)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(t, system, "request:wizard-snapshot-rollback",
		created.Record.State.Revision(), intake.OriginForm)
	if _, err := service.ApplyWizardGaps(context.Background(), request); err == nil {
		t.Fatal("invalid snapshot accepted")
	}
	assertSQLiteWizardGapsInputEffects(t, system, request.RequestRef, 0, 0, 0)
	var snapshots int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM wizard_gaps_result_snapshots`,
	).Scan(&snapshots))
	if snapshots != 0 {
		t.Fatalf("rolled-back snapshots=%d", snapshots)
	}
	current, err := system.service.GetIntake(context.Background(), application.GetIntakeRequest{
		ActorRef: system.principal.ActorRef, ProjectRef: system.project, StateRef: system.stateRef,
	})
	sqliteTestNoError(t, err)
	if current.State.Revision() != created.Record.State.Revision() {
		t.Fatalf("revision=%d", current.State.Revision())
	}
}

type invalidSnapshotStore struct{ *Repository }

func (store *invalidSnapshotStore) ApplyWizardGapsMutation(
	ctx context.Context,
	reservation application.WizardGapsMutationReservation,
) (application.WizardGapsInputRecord, bool, error) {
	reservation.Input.ResultSnapshot.Bytes = nil
	return store.Repository.ApplyWizardGapsMutation(ctx, reservation)
}
