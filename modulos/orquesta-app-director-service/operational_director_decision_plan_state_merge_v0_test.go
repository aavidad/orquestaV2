package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestEnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0FusionaDecisionTardiaV0(t *testing.T) {
	runRef := "run-app-director-decision-plan-state-merge-001"
	planRef := defaultOperationalDirectorDecisionPlanRefV0(runRef)
	firstTask := serviceDirectorDecisionWorkflowTaskForMergeTestV0(runRef, "task-ref-director-decision-merge-first-001", "go test ./first")
	nextTask := serviceDirectorDecisionWorkflowTaskForMergeTestV0(runRef, "task-ref-director-decision-merge-next-001", "go test ./next")
	run := serviceRunForWaitRefsTestV0(runRef, firstTask.TaskID, nextTask.TaskID)
	state, err := directorDecisionOperationalPlanStateV0(
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-24T10:00:00Z",
			CorrelationID: "corr-director-decision-merge-first",
		},
		run,
		planRef,
		[]orquestacoreworkflow.WorkflowTaskV0{firstTask},
	)
	if err != nil {
		t.Fatalf("directorDecisionOperationalPlanStateV0: %v", err)
	}
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)

	next, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-24T10:05:00Z",
			CorrelationID: "corr-director-decision-merge-next",
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(firstTask, nextTask),
			OperationalPlanStateStore:  store,
			OperationalPlanStateWriter: store,
		},
	)
	if err != nil {
		t.Fatalf("ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0: %v", err)
	}
	if next.OperationalDirectorPlanRef != planRef {
		t.Fatalf("plan_ref=%q want %q", next.OperationalDirectorPlanRef, planRef)
	}
	merged := serviceLoadPlanStateForMergeTestV0(t, store, runRef, planRef)
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, merged, "step-wait-subagents")
	firstAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstTask.TaskID)
	nextAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(nextTask.TaskID)
	if merged.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceSameStringSetV0(waitStep.TaskRefs, []string{firstTask.TaskID, nextTask.TaskID}) ||
		!serviceSameStringSetV0(waitStep.AgentRefs, []string{firstAgent, nextAgent}) ||
		!serviceSameStringSetV0(merged.PendingAgentRefs, []string{firstAgent, nextAgent}) ||
		!serviceStringInSetV0(merged.RequiredTestRefs, firstTask.RequiredTests[0]) ||
		!serviceStringInSetV0(merged.RequiredTestRefs, nextTask.RequiredTests[0]) {
		t.Fatalf("merged=%+v waitStep=%+v", merged, waitStep)
	}

	replayed, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-24T10:10:00Z",
			CorrelationID: "corr-director-decision-merge-replay",
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(firstTask, nextTask),
			OperationalPlanStateStore:  store,
			OperationalPlanStateWriter: store,
		},
	)
	if err != nil {
		t.Fatalf("ensure replay: %v", err)
	}
	reloaded := serviceLoadPlanStateForMergeTestV0(t, store, runRef, planRef)
	reloadedWait := serviceOperationalDirectorPlanStateStepForTestV0(t, reloaded, "step-wait-subagents")
	if replayed.OperationalDirectorPlanRef != planRef ||
		reloaded.UpdatedAt != merged.UpdatedAt ||
		!serviceSameStringSetV0(reloadedWait.WaitRefs, waitStep.WaitRefs) ||
		!serviceSameStringSetV0(reloadedWait.TaskRefs, waitStep.TaskRefs) {
		t.Fatalf("replay no idempotente: merged=%+v reloaded=%+v", merged, reloaded)
	}
}

func TestEnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0FusionNoReabrePendienteEntregadoV0(t *testing.T) {
	runRef := "run-app-director-decision-plan-state-merge-delivered-001"
	planRef := defaultOperationalDirectorDecisionPlanRefV0(runRef)
	firstTask := serviceDirectorDecisionWorkflowTaskForMergeTestV0(runRef, "task-ref-director-decision-merge-delivered-first", "go test ./first")
	nextTask := serviceDirectorDecisionWorkflowTaskForMergeTestV0(runRef, "task-ref-director-decision-merge-delivered-next", "go test ./next")
	firstAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstTask.TaskID)
	nextAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(nextTask.TaskID)
	run := serviceRunForWaitRefsTestV0(runRef, firstTask.TaskID, nextTask.TaskID)
	run.DeliveredAgents = []string{firstAgent}
	state, err := directorDecisionOperationalPlanStateV0(
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-24T11:00:00Z",
			CorrelationID: "corr-director-decision-merge-delivered-first",
		},
		run,
		planRef,
		[]orquestacoreworkflow.WorkflowTaskV0{firstTask},
	)
	if err != nil {
		t.Fatalf("directorDecisionOperationalPlanStateV0: %v", err)
	}
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)

	_, err = ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-24T11:05:00Z",
			CorrelationID: "corr-director-decision-merge-delivered-next",
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(firstTask, nextTask),
			OperationalPlanStateStore:  store,
			OperationalPlanStateWriter: store,
		},
	)
	if err != nil {
		t.Fatalf("ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0: %v", err)
	}
	merged := serviceLoadPlanStateForMergeTestV0(t, store, runRef, planRef)
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, merged, "step-wait-subagents")
	if !serviceSameStringSetV0(waitStep.TaskRefs, []string{nextTask.TaskID}) ||
		!serviceSameStringSetV0(waitStep.AgentRefs, []string{nextAgent}) ||
		serviceStringInSetV0(merged.PendingAgentRefs, firstAgent) {
		t.Fatalf("merged reabrio agente entregado: merged=%+v waitStep=%+v", merged, waitStep)
	}
}

func TestClosedOperationalDirectorPlanStateContinueResultV0UsaPlanRefPorDefectoV0(t *testing.T) {
	runRef := "run-app-director-decision-plan-state-closed-default"
	planRef := defaultOperationalDirectorDecisionPlanRefV0(runRef)
	run := serviceRunForWaitRefsTestV0(runRef, "task-ref-director-decision-closed-default")
	run.Status = orquestacoreworkflow.OrchestrationRunStatusClosedV0
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0
	state.ClosureReason = "operational-closure-succeeded"
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)

	result, closed, err := closedOperationalDirectorPlanStateContinueResultV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			CorrelationID: "corr-director-decision-closed-default",
		},
		StartAppDirectorPortsV0{
			RunStore:                  orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			OperationalPlanStateStore: store,
		},
	)
	if err != nil || !closed {
		t.Fatalf("closed=%v err=%v result=%+v", closed, err, result)
	}
	if result.Status != ContinueAppDirectorStatusContinuedV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		result.Run.RunID != runRef ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("result=%+v", result)
	}
}

func serviceDirectorDecisionWorkflowTaskForMergeTestV0(
	runRef string,
	taskRef string,
	requiredTest string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Tarea de decision tardia",
		Summary:            "Fusionar task marcada por decision tardia.",
		WriteSet:           []string{"modulos/orquesta-app-director-service"},
		AcceptanceCriteria: []string{"operational_director.task_source: director_decision"},
		RequiredTests:      []string{requiredTest},
	}
}

func serviceLoadPlanStateForMergeTestV0(
	t *testing.T,
	store orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0,
	runRef string,
	planRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	t.Helper()
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	return state
}

func serviceSameStringSetV0(got []string, want []string) bool {
	got = compactServiceRefsV0(got)
	want = compactServiceRefsV0(want)
	if len(got) != len(want) {
		return false
	}
	for _, value := range want {
		if !serviceStringInSetV0(got, value) {
			return false
		}
	}
	return true
}
