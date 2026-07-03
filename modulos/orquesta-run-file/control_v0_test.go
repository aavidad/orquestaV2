package orquestarunfile

import (
	"context"
	"errors"
	"reflect"
	"testing"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestRunFileStoreControlPersistsAfterRecreateV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()

	if _, err := store.PutRunControlStateV0(ctx, orquestaruncontrol.RunControlStateV0{
		RunRef:             " run-control-1 ",
		Status:             orquestaruncontrol.RunControlStatusStopRequestedV0,
		CheckpointRecorded: true,
		EvidenceRefs:       []string{"seed-ev"},
	}); err != nil {
		t.Fatalf("PutRunControlStateV0: %v", err)
	}
	completed, err := store.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         " run-control-1 ",
		TargetStatus:   " STOPPED ",
		RequestedBy:    " director ",
		Reason:         " drained ",
		IdempotencyKey: " idem-1 ",
		EvidenceRefs:   []string{" ev-1 ", "ev-1", ""},
	})
	if err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}
	if completed.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!completed.CheckpointRecorded ||
		completed.Forced ||
		completed.Meta.RequestedBy != "director" ||
		!reflect.DeepEqual(completed.EvidenceRefs, []string{"seed-ev", "ev-1"}) {
		t.Fatalf("completed=%+v", completed)
	}

	reopened := mustNewRunFileStoreV0(t, dir)
	read, err := reopened.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: "run-control-1",
	})
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if !reflect.DeepEqual(read, completed) {
		t.Fatalf("read=%+v completed=%+v", read, completed)
	}

	read.EvidenceRefs[0] = "mutated"
	again, err := reopened.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: "run-control-1",
	})
	if err != nil {
		t.Fatalf("ReadRunControlStateV0 again: %v", err)
	}
	if again.EvidenceRefs[0] != "seed-ev" {
		t.Fatalf("state leaked mutable evidence refs: %+v", again)
	}
}

func TestRunFileStoreControlMissingAndInvalidCompletionV0(t *testing.T) {
	store := mustNewRunFileStoreV0(t, t.TempDir())

	_, err := store.ReadRunControlStateV0(context.Background(), orquestaruncontrol.RunControlReadRequestV0{
		RunRef: "run-missing",
	})
	var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
	if err == nil || !errors.As(err, &notFound) || notFound.RunRef != "run-missing" {
		t.Fatalf("err=%v notFound=%+v", err, notFound)
	}

	_, err = store.CompleteRunControlV0(context.Background(), orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-1",
		TargetStatus: orquestaruncontrol.RunControlStatusRunningV0,
	})
	var invalid orquestaruncontrol.RunControlCompletionTargetErrorV0
	if err == nil || !errors.As(err, &invalid) {
		t.Fatalf("err=%v invalid=%+v", err, invalid)
	}
}

func TestRunFileStorePauseResumePersisteEvidenciaCleanupExternoV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	const cleanupEvidence = "evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup"

	paused, err := store.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
		RunRef:       "run-control-cleanup-evidence",
		RequestedBy:  "orquesta-director",
		Reason:       "pausar conservando evidencia de cleanup externo",
		EvidenceRefs: []string{" " + cleanupEvidence + " ", cleanupEvidence, ""},
	})
	if err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}
	if paused.Status != orquestaruncontrol.RunControlStatusPausedV0 ||
		!reflect.DeepEqual(paused.EvidenceRefs, []string{cleanupEvidence}) {
		t.Fatalf("paused=%+v", paused)
	}

	resumed, err := store.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
		RunRef:       "run-control-cleanup-evidence",
		RequestedBy:  "orquesta-director",
		Reason:       "reanudar conservando evidencia de cleanup externo",
		EvidenceRefs: []string{cleanupEvidence},
	})
	if err != nil {
		t.Fatalf("ResumeRunV0: %v", err)
	}
	if resumed.Status != orquestaruncontrol.RunControlStatusRunningV0 ||
		!reflect.DeepEqual(resumed.EvidenceRefs, []string{cleanupEvidence}) {
		t.Fatalf("resumed=%+v", resumed)
	}

	reopened := mustNewRunFileStoreV0(t, dir)
	read, err := reopened.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: "run-control-cleanup-evidence",
	})
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if !reflect.DeepEqual(read, resumed) {
		t.Fatalf("read=%+v resumed=%+v", read, resumed)
	}
}

func TestRunFileStoreRecordRunCheckpointPersisteV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	if _, err := store.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       "run-checkpoint",
		EvidenceRefs: []string{"stop-ref"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	state, err := store.RecordRunCheckpointV0(ctx, orquestaruncontrol.RecordRunCheckpointCommandV0{
		RunRef:       "run-checkpoint",
		RequestedBy:  "preparer",
		Reason:       "ack durable",
		EvidenceRefs: []string{"checkpoint-ref-001"},
	})
	if err != nil {
		t.Fatalf("RecordRunCheckpointV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStopRequestedV0 ||
		!state.CheckpointRecorded ||
		!reflect.DeepEqual(state.EvidenceRefs, []string{"stop-ref", "checkpoint-ref-001"}) {
		t.Fatalf("state=%+v", state)
	}
	reopened := mustNewRunFileStoreV0(t, dir)
	read, err := reopened.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: "run-checkpoint",
	})
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if !reflect.DeepEqual(read, state) {
		t.Fatalf("read=%+v state=%+v", read, state)
	}
}
