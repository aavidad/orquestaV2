package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestLiveProcessCapacityGateV0ReservaSoloHuecoLibreGlobal(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()
	snapshots := map[string]orquestaruntime.ProcessRuntimeSnapshotV0{}
	for _, item := range []struct {
		agentRef   string
		processRef string
		status     orquestaruntime.ProcessRuntimeStatusV0
	}{
		{"agent-ref-capacity-001", "process-ref-capacity-001", orquestaruntime.ProcessRuntimeRunningV0},
		{"agent-ref-capacity-002", "process-ref-capacity-002", orquestaruntime.ProcessRuntimeRunningV0},
		{"agent-ref-capacity-003", "process-ref-capacity-003", orquestaruntime.ProcessRuntimeStoppedV0},
	} {
		if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
			RunID:          "run-capacity-gate-001",
			AgentRequestID: item.agentRef,
			ProcessRef:     item.processRef,
			SessionRef:     "session-ref-" + item.agentRef,
			LaunchRef:      "launch-ref-" + item.agentRef,
			ReadinessRef:   "readiness-ref-" + item.agentRef,
		}); err != nil {
			t.Fatalf("record process: %v", err)
		}
		snapshots[item.processRef] = orquestaruntime.ProcessRuntimeSnapshotV0{
			SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
			ProcessRef:    item.processRef,
			Status:        item.status,
		}
	}
	gate := &LiveProcessCapacityGateV0{
		Registry:       registry,
		SnapshotSource: liveProcessSnapshotSourceForTestV0{snapshots: snapshots},
		Limit:          3,
	}

	first, err := gate.ReserveLiveProcessCapacityV0(context.Background(), LiveProcessCapacityReservationRequestV0{
		RunRef:    "run-capacity-gate-new",
		Requested: 4,
	})
	if err != nil {
		t.Fatalf("ReserveLiveProcessCapacityV0: %v", err)
	}
	if first.Live != 2 || first.Granted != 1 || first.Limit != 3 {
		t.Fatalf("first=%+v", first)
	}
	second, err := gate.ReserveLiveProcessCapacityV0(context.Background(), LiveProcessCapacityReservationRequestV0{
		RunRef:    "run-capacity-gate-new",
		Requested: 1,
	})
	if err != nil {
		t.Fatalf("second ReserveLiveProcessCapacityV0: %v", err)
	}
	if second.Granted != 0 || second.Live != 2 || second.Reserved != 1 {
		t.Fatalf("second=%+v", second)
	}
	if err := gate.ReleaseLiveProcessCapacityV0(context.Background(), first); err != nil {
		t.Fatalf("ReleaseLiveProcessCapacityV0: %v", err)
	}
	third, err := gate.ReserveLiveProcessCapacityV0(context.Background(), LiveProcessCapacityReservationRequestV0{
		RunRef:    "run-capacity-gate-new",
		Requested: 1,
	})
	if err != nil {
		t.Fatalf("third ReserveLiveProcessCapacityV0: %v", err)
	}
	if third.Granted != 1 || third.Reserved != 0 {
		t.Fatalf("third=%+v", third)
	}
}

type liveProcessSnapshotSourceForTestV0 struct {
	snapshots map[string]orquestaruntime.ProcessRuntimeSnapshotV0
}

func (source liveProcessSnapshotSourceForTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return source.snapshots[processRef], nil
}
