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
		!reflect.DeepEqual(completed.EvidenceRefs, []string{"ev-1"}) {
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
	if again.EvidenceRefs[0] != "ev-1" {
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
