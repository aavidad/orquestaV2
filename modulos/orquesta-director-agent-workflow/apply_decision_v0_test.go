package orquestadirectoragentworkflow

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestApplyDirectorAgentDecisionV0AplicaBrainstormPorPuertos(t *testing.T) {
	run := directorAgentWorkflowBrainstormRunForTestV0(t, "run-ref-001")
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowRequestForTestV0()),
		ApplyDirectorAgentDecisionPortsV0{
			RunStore:  store,
			EventSink: sink,
		},
	)
	if err != nil {
		t.Fatalf("ApplyDirectorAgentDecisionV0: %v", err)
	}
	if len(result.Issues) != 0 || result.EventsCount != 1 {
		t.Fatalf("result=%+v", result)
	}
	if !directorAgentWorkflowStringInSetV0(result.Run.Brainstorms, "brainstorm-ref-director-001") {
		t.Fatalf("brainstorms=%v", result.Run.Brainstorms)
	}
	if !directorAgentWorkflowSinkHasEventV0(sink, orquestacoreworkflow.OrchestrationEventBrainstormRequestedV0) {
		t.Fatalf("sink sin BrainstormRequested: %+v", sink.EventsV0())
	}
}

func TestApplyDirectorAgentDecisionV0AplicaContratoYMicrotareaPorPuertos(t *testing.T) {
	run := directorAgentWorkflowPlanningRunForTestV0(t, "run-ref-001")
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ports := ApplyDirectorAgentDecisionPortsV0{
		RunStore:  store,
		EventSink: sink,
		TaskStore: taskStore,
	}

	contractResult, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowContractRequestForTestV0()),
		ports,
	)
	if err != nil {
		t.Fatalf("ApplyDirectorAgentDecisionV0 contrato: %v", err)
	}
	if len(contractResult.Issues) != 0 || contractResult.EventsCount != 1 {
		t.Fatalf("contract result=%+v", contractResult)
	}
	if !directorAgentWorkflowStringInSetV0(contractResult.Run.FunctionContracts, "contract:function:agenda:v0") {
		t.Fatalf("function_contracts=%v", contractResult.Run.FunctionContracts)
	}

	taskResult, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowMicrotaskRequestForTestV0()),
		ports,
	)
	if err != nil {
		t.Fatalf("ApplyDirectorAgentDecisionV0 microtarea: %v", err)
	}
	if len(taskResult.Issues) != 0 || taskResult.EventsCount != 1 {
		t.Fatalf("task result=%+v", taskResult)
	}
	if !directorAgentWorkflowStringInSetV0(taskResult.Run.Tasks, "task-ref-agenda-001") {
		t.Fatalf("tasks=%v", taskResult.Run.Tasks)
	}
	storedTasks, err := taskStore.LoadWorkflowTasksV0(
		context.Background(),
		"run-ref-001",
		[]string{"task-ref-agenda-001"},
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 || storedTasks[0].PhaseID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("stored tasks=%+v", storedTasks)
	}
	if !directorAgentWorkflowSinkHasEventV0(sink, orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0) {
		t.Fatalf("sink sin MicrotaskCreated: %+v", sink.EventsV0())
	}
}

func TestApplyDirectorAgentDecisionV0ValidaPuertos(t *testing.T) {
	result, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowRequestForTestV0()),
		ApplyDirectorAgentDecisionPortsV0{},
	)
	if err != nil {
		t.Fatalf("no debe devolver error tecnico: %v", err)
	}
	requireDirectorAgentWorkflowIssueV0(t, result.Issues, "director_agent_workflow_required")
}

func TestApplyDirectorAgentDecisionV0RequiereTaskStoreParaMicrotarea(t *testing.T) {
	run := directorAgentWorkflowPlanningRunForTestV0(t, "run-ref-001")
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ports := ApplyDirectorAgentDecisionPortsV0{RunStore: store, EventSink: sink}

	if _, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowContractRequestForTestV0()),
		ports,
	); err != nil {
		t.Fatalf("preparar contrato: %v", err)
	}

	result, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowMicrotaskRequestForTestV0()),
		ports,
	)
	if err != nil {
		t.Fatalf("no debe devolver error tecnico: %v", err)
	}
	requireDirectorAgentWorkflowIssueV0(t, result.Issues, "director_agent_workflow_required")
}

func directorAgentWorkflowBrainstormRunForTestV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	var run orquestacoreworkflow.OrchestrationRunV0
	run = directorAgentWorkflowApplyForTestV0(t, run, directorAgentWorkflowStartRunCommandV0(t, runRef))
	run = directorAgentWorkflowApplyForTestV0(t, run, directorAgentWorkflowOpenBrainstormCommandV0(t, runRef))
	return run
}

func directorAgentWorkflowStartRunCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-start-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-start-" + runRef,
			OccurredAt:     "2026-05-09T23:00:00Z",
		},
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-001",
			AppSpecRef: "spec-ref-001",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func directorAgentWorkflowOpenBrainstormCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-brainstorm-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-open-brainstorm-" + runRef,
			OccurredAt:     "2026-05-09T23:01:00Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: "brainstorming_arquitectura",
			Reason:  "Preparar decision inicial.",
		},
	)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	return command
}

func directorAgentWorkflowApplyForTestV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle command: %v", err)
	}
	next, err := applyDirectorAgentWorkflowEventsV0(run, result.Events)
	if err != nil {
		t.Fatalf("apply events: %v", err)
	}
	return next
}

func directorAgentWorkflowStringInSetV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func directorAgentWorkflowSinkHasEventV0(
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	eventType string,
) bool {
	for _, event := range sink.EventsV0() {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
