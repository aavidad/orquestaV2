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

	microtaskRequest := validDirectorAgentWorkflowMicrotaskRequestForTestV0()
	microtaskRequest.Decision.CreateMicrotask.Task.ParentTaskRef = "task-ref-parent-001"
	microtaskRequest.Decision.CreateMicrotask.Task.CohortRef = "cohort-ref-recursive-001"
	microtaskRequest.Decision.CreateMicrotask.Task.WaveRef = "wave-ref-recursive-001"
	microtaskRequest.Decision.CreateMicrotask.Task.ContextRefs = []string{
		"context-ref-scope-001",
		"context-ref-policy-001",
	}
	microtaskRequest.Decision.CreateMicrotask.Task.DelegationDepth = 2
	microtaskRequest.Decision.CreateMicrotask.Task.MaxChildAgents = 4
	microtaskRequest.Decision.CreateMicrotask.Task.ChildTaskRefs = []string{
		"task-ref-child-001",
		"task-ref-child-002",
	}
	requests := []DirectorAgentWorkflowCommandRequestV0{
		validDirectorAgentWorkflowOpenVoteRequestForTestV0(),
		validDirectorAgentWorkflowVoteRequestForTestV0(),
		validDirectorAgentWorkflowAcceptRequestForTestV0(),
		validDirectorAgentWorkflowOpenPlanningRequestForTestV0(),
		validDirectorAgentWorkflowContractRequestForTestV0(),
		microtaskRequest,
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
	if tasks[0].ParentTaskRef != "task-ref-parent-001" ||
		tasks[0].CohortRef != "cohort-ref-recursive-001" ||
		tasks[0].WaveRef != "wave-ref-recursive-001" ||
		len(tasks[0].ContextRefs) != 2 ||
		tasks[0].ContextRefs[0] != "context-ref-scope-001" ||
		tasks[0].ContextRefs[1] != "context-ref-policy-001" ||
		tasks[0].DelegationDepth != 2 ||
		tasks[0].MaxChildAgents != 4 ||
		len(tasks[0].ChildTaskRefs) != 2 ||
		tasks[0].ChildTaskRefs[0] != "task-ref-child-001" ||
		tasks[0].ChildTaskRefs[1] != "task-ref-child-002" {
		t.Fatalf("linaje no materializado: %+v", tasks[0])
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
