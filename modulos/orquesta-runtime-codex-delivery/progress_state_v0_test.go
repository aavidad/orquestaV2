package orquestaruntimecodexdelivery

import (
	"context"
	"testing"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexProgressStateV0MuestrasSinCambioAumentanNoProgressSinRepeticion(t *testing.T) {
	store := NewInMemoryCodexProgressStateStoreV0()
	sample := codexProgressStateSampleForTestV0("firma-repetida")

	first := codexProgressStateObserveForTestV0(t, store, sample)
	if first.Current.RepeatedActionCount != 0 || first.Previous != nil {
		t.Fatalf("first=%+v", first)
	}

	second := codexProgressStateObserveForTestV0(t, store, sample)
	if second.Current.RepeatedActionCount != 0 ||
		second.Current.ProgressCounter != 1 ||
		second.Previous == nil {
		t.Fatalf("second=%+v", second)
	}

	third := codexProgressStateObserveForTestV0(t, store, sample)
	if third.Current.RepeatedActionCount != 0 ||
		third.Current.ProgressCounter != 1 ||
		third.Previous == nil {
		t.Fatalf("third=%+v", third)
	}
}

func TestCodexProgressStateV0SilencioSostenidoReportaStalledNoLoop(t *testing.T) {
	store := NewInMemoryCodexProgressStateStoreV0()
	sample := codexProgressStateSampleForTestV0("firma-loop")
	_ = codexProgressStateObserveForTestV0(t, store, sample)
	_ = codexProgressStateObserveForTestV0(t, store, sample)
	state := codexProgressStateObserveForTestV0(t, store, sample)

	report, issues := orquestaruntime.BuildAgentProgressReportFromHeartbeatV0(
		"agent-progress-report-ref-state-loop-001",
		codexProgressStateSnapshotForTestV0(state.Current.ProcessRef),
		state.Previous,
		state.Current,
		orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: 2,
			LoopAfterRepeatedActions:    2,
		},
	)
	if len(issues) > 0 {
		t.Fatalf("report issues=%+v", issues)
	}
	if report.Status != orquestaruntime.AgentStalledV0 ||
		report.RepeatedActionCount != 0 ||
		report.NoProgressTicks != 2 {
		t.Fatalf("report=%+v", report)
	}
}

func TestCodexProgressStateV0ActividadConMismaAccionEscalaABucle(t *testing.T) {
	store := NewInMemoryCodexProgressStateStoreV0()
	sample := codexProgressStateSampleForTestV0("firma-accion-001")
	sample.ActionSignature = "action-sig-diff-progress-state-001"

	_ = codexProgressStateObserveForTestV0(t, store, sample)
	sample.Signature = "firma-accion-002"
	second := codexProgressStateObserveForTestV0(t, store, sample)
	sample.Signature = "firma-accion-003"
	third := codexProgressStateObserveForTestV0(t, store, sample)

	report, issues := orquestaruntime.BuildAgentProgressReportFromHeartbeatV0(
		"agent-progress-report-ref-state-repeated-action-001",
		codexProgressStateSnapshotForTestV0(third.Current.ProcessRef),
		third.Previous,
		third.Current,
		orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: 99,
			LoopAfterRepeatedActions:    2,
		},
	)
	if len(issues) > 0 {
		t.Fatalf("report issues=%+v", issues)
	}
	if second.Current.RepeatedActionCount != 1 ||
		report.Status != orquestaruntime.AgentLoopDetectedV0 ||
		report.RepeatedActionCount != 2 ||
		report.NoProgressTicks != 0 {
		t.Fatalf("second=%+v report=%+v", second.Current, report)
	}
}

func codexProgressStateSampleForTestV0(signature string) CodexProgressSampleV0 {
	return CodexProgressSampleV0{
		RunID:          "run-ref-progress-state-001",
		AgentRequestID: "agent-ref-progress-state-001",
		ProcessRef:     "process-ref-progress-state-001",
		Signature:      signature,
		EvidenceRefs:   []string{"evidence-ref-progress-state-001"},
		ObservedAt:     time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC),
	}
}

func codexProgressStateObserveForTestV0(
	t *testing.T,
	store *InMemoryCodexProgressStateStoreV0,
	sample CodexProgressSampleV0,
) CodexProgressObservationStateV0 {
	t.Helper()
	got, err := store.ObserveCodexProgressV0(context.Background(), sample)
	if err != nil {
		t.Fatalf("ObserveCodexProgressV0: %v", err)
	}
	if !got.SampleAccepted {
		t.Fatalf("sample no aceptada: %+v", got)
	}
	return got
}

func codexProgressStateSnapshotForTestV0(processRef string) orquestaruntime.ProcessRuntimeSnapshotV0 {
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    processRef,
		SessionRef:    "session-ref-progress-state-001",
		LaunchRef:     "launch-ref-progress-state-001",
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
}
