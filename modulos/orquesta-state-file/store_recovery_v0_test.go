package orquestastatefile

import (
	"context"
	"reflect"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStoreV0RecuperaEstadoTrasRecrearInstancia(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	run := validRunV0()
	event := mustRunStartedEventV0(t, run.RunID)
	task := mustWorkflowTaskV0(t, run.RunID)
	task.ParentTaskRef = "task-ref-parent-state-file-001"
	task.CohortRef = "cohort-state-file-001"
	task.WaveRef = "wave-state-file-001"
	task.DelegationDepth = 1
	task.MaxChildAgents = 6
	task.ChildTaskRefs = []string{"task-ref-child-state-file-001"}
	waitState := stateFileWorkflowTaskWaitStateV0()
	waitState.RunRef = run.RunID
	waitState.ParentTaskRef = task.ParentTaskRef
	planState := stateFileOperationalDirectorPlanStateV0()
	planState.RunRef = run.RunID
	planState.ProjectRef = run.ProjectRef
	planState.ActiveParentTaskRef = task.ParentTaskRef
	planState.PendingAgentRefs = []string{"agent-ref-state-file-001"}
	planState.Steps[0].ParentTaskRef = task.ParentTaskRef
	planState.Steps[0].PendingAgentRefs = []string{"agent-ref-state-file-001"}
	planState.Steps[1].ParentTaskRef = task.ParentTaskRef
	planState.Steps[1].WaitRefs = []string{waitState.WaitRef}
	planState.Steps[1].AgentRefs = []string{"agent-ref-state-file-001"}
	planState.Steps[1].PendingAgentRefs = []string{"agent-ref-state-file-001"}
	record := validAgentProcessRecordV0(run.RunID)

	if err := store.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, run.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("append events: %v", err)
	}
	if err := store.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("save workflow task: %v", err)
	}
	if err := store.SaveWorkflowTaskWaitStateV0(ctx, waitState); err != nil {
		t.Fatalf("save wait state: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, planState); err != nil {
		t.Fatalf("save plan state: %v", err)
	}
	if err := store.RecordAgentProcessV0(ctx, record); err != nil {
		t.Fatalf("record agent process: %v", err)
	}

	recovered := mustStoreV0(t, rootDir)
	assertRecoveredRunV0(t, recovered, run)
	assertRecoveredEventsV0(t, recovered, run.RunID, []orquestacoreworkflow.OrchestrationEventV0{event})
	assertRecoveredTasksV0(t, recovered, run.RunID, []orquestacoreworkflow.WorkflowTaskV0{task})
	assertRecoveredWaitStateV0(t, recovered, waitState)
	assertRecoveredPlanStateV0(t, recovered, planState)
	assertRecoveredAgentProcessV0(t, recovered, record)
}

func TestStoreV0RechazaConflictosDurables(t *testing.T) {
	ctx := context.Background()
	store := mustStoreV0(t, t.TempDir())
	task := mustWorkflowTaskV0(t, "run-state-file-001")
	if err := store.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("save task: %v", err)
	}
	changed := task
	changed.Title = "Cambio incompatible"
	if err := store.SaveWorkflowTaskV0(ctx, changed); err == nil {
		t.Fatal("expected workflow task conflict")
	}

	record := validAgentProcessRecordV0("run-state-file-001")
	if err := store.RecordAgentProcessV0(ctx, record); err != nil {
		t.Fatalf("record process: %v", err)
	}
	changedRecord := record
	changedRecord.ProcessRef = "process-ref-002"
	if err := store.RecordAgentProcessV0(ctx, changedRecord); err == nil {
		t.Fatal("expected process registry conflict")
	}
}

func assertRecoveredRunV0(t *testing.T, store *StoreV0, want orquestacoreworkflow.OrchestrationRunV0) {
	t.Helper()
	got, err := store.LoadRunV0(context.Background(), want.RunID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("run=%+v, want=%+v", got, want)
	}
}

func assertRecoveredEventsV0(
	t *testing.T,
	store *StoreV0,
	runRef string,
	want []orquestacoreworkflow.OrchestrationEventV0,
) {
	t.Helper()
	got, err := store.LoadRunEventsV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events=%+v, want=%+v", got, want)
	}
}

func assertRecoveredTasksV0(
	t *testing.T,
	store *StoreV0,
	runRef string,
	want []orquestacoreworkflow.WorkflowTaskV0,
) {
	t.Helper()
	got, err := store.LoadWorkflowTasksV0(context.Background(), runRef, []string{want[0].TaskID})
	if err != nil {
		t.Fatalf("load tasks: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tasks=%+v, want=%+v", got, want)
	}
}

func assertRecoveredAgentProcessV0(
	t *testing.T,
	store *StoreV0,
	want orquestacionnucleoapp.AgentProcessRecordV0,
) {
	t.Helper()
	got, err := store.ResolveAgentProcessV0(context.Background(), want.RunID, want.AgentRequestID)
	if err != nil {
		t.Fatalf("resolve process: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("record=%+v, want=%+v", got, want)
	}
}

func assertRecoveredWaitStateV0(
	t *testing.T,
	store *StoreV0,
	want orquestacionnucleoapp.WorkflowTaskWaitStateV0,
) {
	t.Helper()
	got, err := store.LoadWorkflowTaskWaitStateV0(context.Background(), want.RunRef, want.WaitRef)
	if err != nil {
		t.Fatalf("load wait state: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wait state=%+v, want=%+v", got, want)
	}
}

func assertRecoveredPlanStateV0(
	t *testing.T,
	store *StoreV0,
	want orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) {
	t.Helper()
	got, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), want.RunRef, want.PlanRef)
	if err != nil {
		t.Fatalf("load plan state: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plan state=%+v, want=%+v", got, want)
	}
}
