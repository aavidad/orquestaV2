package orquestarunmemory

import (
	"context"
	"reflect"
	"testing"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestRunMemoryStoreRecordRunCheckpointPreservaEstadoV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()
	_, err := store.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       "run-checkpoint",
		RequestedBy:  "operator",
		Reason:       "shutdown",
		EvidenceRefs: []string{"evidence-stop"},
	})
	if err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	state, err := store.RecordRunCheckpointV0(ctx, orquestaruncontrol.RecordRunCheckpointCommandV0{
		RunRef:       "run-checkpoint",
		RequestedBy:  "checkpoint-preparer",
		Reason:       "ack durable",
		EvidenceRefs: []string{"checkpoint-ref-001"},
	})
	if err != nil {
		t.Fatalf("RecordRunCheckpointV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStopRequestedV0 ||
		!state.CheckpointRecorded ||
		!reflect.DeepEqual(state.EvidenceRefs, []string{"evidence-stop", "checkpoint-ref-001"}) {
		t.Fatalf("state=%+v", state)
	}
}
