package orquestadirectoragentworkflow

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestApplyDirectorAgentDecisionV0PlanificaMicrotareaDesdeCadenaDirector(t *testing.T) {
	run := directorAgentWorkflowBrainstormRunForTestV0(t, "run-ref-001")
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ports := ApplyDirectorAgentDecisionPortsV0{
		RunStore:  store,
		EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		TaskStore: taskStore,
	}

	requests := []DirectorAgentWorkflowCommandRequestV0{
		validDirectorAgentWorkflowOpenVoteRequestForTestV0(),
		validDirectorAgentWorkflowVoteRequestForTestV0(),
		validDirectorAgentWorkflowAcceptRequestForTestV0(),
		validDirectorAgentWorkflowOpenPlanningRequestForTestV0(),
		validDirectorAgentWorkflowContractRequestForTestV0(),
		validDirectorAgentWorkflowMicrotaskRequestForTestV0(),
	}
	var result ApplyDirectorAgentDecisionResultV0
	for _, request := range requests {
		applied, err := ApplyDirectorAgentDecisionV0(
			context.Background(),
			ApplyDirectorAgentDecisionRequestV0(request),
			ports,
		)
		if err != nil {
			t.Fatalf("ApplyDirectorAgentDecisionV0 %s: %v", request.Decision.CommandType, err)
		}
		if len(applied.Issues) != 0 {
			t.Fatalf("issues en %s: %+v", request.Decision.CommandType, applied.Issues)
		}
		result = applied
	}

	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	if !directorAgentWorkflowStringInSetV0(result.Run.Tasks, "task-ref-agenda-001") {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(
		context.Background(),
		"run-ref-001",
		[]string{"task-ref-agenda-001"},
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(tasks) != 1 || tasks[0].PhaseID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("stored tasks=%+v", tasks)
	}
}

func validDirectorAgentWorkflowOpenPlanningRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-open-planning-001",
		RunID:         "run-ref-001",
		PhaseID:       "votacion_y_decision",
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-director-open-planning-001",
		Summary:       "Abrir planificacion de microtareas.",
		EvidenceRefs:  []string{"evidence-ref-open-planning-001"},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: "planificacion_microtareas",
			Reason:  "Preparar microtareas compactas.",
		},
	}
	return request
}
