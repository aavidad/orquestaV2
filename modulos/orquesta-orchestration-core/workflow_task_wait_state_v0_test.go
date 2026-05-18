package orquestacionnucleoapp

import (
	"context"
	"testing"
)

func TestInMemoryWorkflowTaskWaitStateStoreV0GuardaYRecupera(t *testing.T) {
	store := NewInMemoryWorkflowTaskWaitStateStoreV0()
	state := WorkflowTaskWaitStateV0{
		SchemaVersion:    WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:          "wait-ref-nucleo-001",
		RunRef:           "run-ref-nucleo-wait-state-001",
		ReasonCode:       WorkflowTaskWaitReasonCohortInProgressV0,
		CohortRef:        "cohort-a",
		WaveRef:          "wave-01",
		TaskRefs:         []string{"task-a"},
		AgentRefs:        []string{"agent-a"},
		PendingAgentRefs: []string{"agent-a"},
		Status:           WorkflowTaskWaitStateStatusWaitingV0,
		MaxExternalWaits: 2,
		CorrelationID:    "corr-nucleo-wait-state-001",
		EvidenceRefs:     []string{"evidence-ref-nucleo-wait-state-001"},
		ObservedAt:       "2026-05-17T13:00:00Z",
	}
	if err := store.SaveWorkflowTaskWaitStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveWorkflowTaskWaitStateV0: %v", err)
	}
	got, err := store.LoadWorkflowTaskWaitStateV0(context.Background(), state.RunRef, state.WaitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if got.WaitRef != state.WaitRef ||
		got.ReasonCode != WorkflowTaskWaitReasonCohortInProgressV0 ||
		len(got.PendingAgentRefs) != 1 {
		t.Fatalf("state=%+v", got)
	}
}

func TestWorkflowTaskWaitStateV0RequiereScope(t *testing.T) {
	_, err := NewWorkflowTaskWaitStateV0(WorkflowTaskWaitStateV0{
		SchemaVersion: WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:       "wait-ref-nucleo-no-scope-001",
		RunRef:        "run-ref-nucleo-no-scope-001",
		ReasonCode:    WorkflowTaskWaitReasonCohortInProgressV0,
		Status:        WorkflowTaskWaitStateStatusWaitingV0,
		ObservedAt:    "2026-05-17T13:01:00Z",
	})
	if err == nil {
		t.Fatal("err=nil, want scope error")
	}
}

func TestWorkflowTaskWaitStateV0RechazaReasonCodeDesconocido(t *testing.T) {
	_, err := NewWorkflowTaskWaitStateV0(WorkflowTaskWaitStateV0{
		SchemaVersion: WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:       "wait-ref-nucleo-reason-001",
		RunRef:        "run-ref-nucleo-reason-001",
		ReasonCode:    "typo_reason",
		CohortRef:     "cohort-reason",
		Status:        WorkflowTaskWaitStateStatusWaitingV0,
		ObservedAt:    "2026-05-17T13:02:00Z",
	})
	if err == nil {
		t.Fatal("err=nil, want reason_code error")
	}
}
