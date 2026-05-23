package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestEnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0AceptaMarkerEnContextRefsV0(t *testing.T) {
	runRef := "run-app-director-context-marker-plan-state-001"
	taskRef := "task-ref-app-director-context-marker-plan-state-001"
	run := serviceRunForWaitRefsTestV0(runRef, taskRef)
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileImplementationV0,
		Title:           "Tarea marcada por context ref",
		WriteSet:        []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{
			"plan operativo reentrable creado desde context ref",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-app-codex-stack"},
		ContextRefs:   []string{"operational_director.task_source:autoprogramming"},
	}
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()

	next, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-23T10:00:00Z",
			CorrelationID: "corr-app-director-context-marker-plan-state-001",
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
			OperationalPlanStateStore:  planStore,
			OperationalPlanStateWriter: planStore,
		},
	)
	if err != nil {
		t.Fatalf("ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0: %v", err)
	}
	if next.OperationalDirectorPlanRef == "" {
		t.Fatalf("operational_director_plan_ref vacio")
	}
	state, err := planStore.LoadOperationalDirectorPlanStateV0(
		context.Background(),
		runRef,
		next.OperationalDirectorPlanRef,
	)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.TaskRefs, taskRef) ||
		!serviceStringInSetV0(state.RequiredTestRefs, task.RequiredTests[0]) {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
}
