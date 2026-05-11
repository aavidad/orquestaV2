package orquestadirectoragentworkflow

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestApplyDirectorAgentDecisionV0AplicaCloseTaskPorPuertos(t *testing.T) {
	run := directorAgentWorkflowCloseTaskReadyRunForTestV0()
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(reviewWorkflowRequestV0(closeTaskDecisionForWorkflowTestV0())),
		ApplyDirectorAgentDecisionPortsV0{RunStore: store, EventSink: sink},
	)
	if err != nil {
		t.Fatalf("ApplyDirectorAgentDecisionV0 close_task: %v", err)
	}
	if len(result.Issues) != 0 || result.EventsCount != 1 {
		t.Fatalf("result=%+v", result)
	}
	if !directorAgentWorkflowStringInSetV0(result.Run.ClosedTasks, "task-ref-agenda-001") {
		t.Fatalf("closed_tasks=%v", result.Run.ClosedTasks)
	}
	if !directorAgentWorkflowSinkHasEventV0(sink, orquestacoreworkflow.OrchestrationEventTaskClosedV0) {
		t.Fatalf("sink sin TaskClosed: %+v", sink.EventsV0())
	}
}

func directorAgentWorkflowCloseTaskReadyRunForTestV0() orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for index := range phases {
		if phases[index].ID == orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
			phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		}
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           "run-ref-001",
		ProjectRef:      "project-ref-001",
		AppSpecRef:      "spec-ref-001",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases:          phases,
		Tasks:           []string{"task-ref-agenda-001"},
		Deliveries:      []string{"delivery-ref-001"},
		Reviews:         []string{"review-request-ref-001"},
		ReviewResults:   []string{"review-result-ref-001#review_result:accepted#review_request:review-request-ref-001#delivery:delivery-ref-001"},
		AcceptedReviews: []string{"accepted-review-ref-001"},
		LastEventID:     "evt-ready-close-task-001",
		LastSequence:    41,
	}
}
