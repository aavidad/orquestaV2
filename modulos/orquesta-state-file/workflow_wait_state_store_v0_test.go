package orquestastatefile

import (
	"context"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStoreV0RecuperaWorkflowTaskWaitStateV0(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileWorkflowTaskWaitStateV0()
	if err := store.SaveWorkflowTaskWaitStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveWorkflowTaskWaitStateV0: %v", err)
	}
	recovered := mustStoreV0(t, rootDir)
	got, err := recovered.LoadWorkflowTaskWaitStateV0(context.Background(), state.RunRef, state.WaitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if got.WaitRef != state.WaitRef || len(got.PendingAgentRefs) != 1 {
		t.Fatalf("state=%+v", got)
	}
}

func TestStoreV0RechazaWorkflowTaskWaitStateConRefInternaInconsistente(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileWorkflowTaskWaitStateV0()
	document := workflowWaitStateDocumentV0{
		SchemaVersion: workflowWaitDocumentSchemaV0,
		RunRef:        state.RunRef,
		WaitRef:       state.WaitRef,
		State:         state,
	}
	document.State.RunRef = "run-state-file-distinto"
	if err := writeJSONAtomicV0(store.workflowWaitPathV0(state.RunRef, state.WaitRef), document); err != nil {
		t.Fatalf("writeJSONAtomicV0: %v", err)
	}
	if _, err := store.LoadWorkflowTaskWaitStateV0(context.Background(), state.RunRef, state.WaitRef); err == nil {
		t.Fatal("err=nil, want ref interna inconsistente")
	}
}

func stateFileWorkflowTaskWaitStateV0() orquestacionnucleoapp.WorkflowTaskWaitStateV0 {
	return orquestacionnucleoapp.WorkflowTaskWaitStateV0{
		SchemaVersion:    orquestacionnucleoapp.WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:          "wait-ref-state-file-001",
		RunRef:           "run-state-file-001",
		ReasonCode:       orquestacionnucleoapp.WorkflowTaskWaitReasonCohortInProgressV0,
		CohortRef:        "cohort-state-file-001",
		WaveRef:          "wave-state-file-001",
		TaskRefs:         []string{"task-ref-state-file-001"},
		AgentRefs:        []string{"agent-ref-state-file-001"},
		PendingAgentRefs: []string{"agent-ref-state-file-001"},
		Status:           orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0,
		MaxExternalWaits: 2,
		CorrelationID:    "corr-state-file-wait-001",
		EvidenceRefs:     []string{"evidence-ref-state-file-wait-001"},
		ObservedAt:       "2026-05-17T13:20:00Z",
	}
}
