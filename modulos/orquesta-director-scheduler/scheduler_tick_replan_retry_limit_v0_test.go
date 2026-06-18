package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func schedulerReplanRefForTaskV0(replanRef, taskRef, action string) string {
	return replanRef + "#source:source-ref#task:" + taskRef + "#action:" + action + "#followups:f1"
}

// Tras alcanzar el limite de replans, una accion resolutiva (retry_task) se escala
// a decision (NeedsDirector) registrando la decision pero SIN ejecutar followups.
func TestBuildDirectorSchedulerTickV0ReplanRetryLimitEscalaADirectorV0(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)
	taskRef := "task-ref-scheduler-001"
	input.Snapshot.ReplanRefs = []string{
		schedulerReplanRefForTaskV0("replan-prev-1", taskRef, "retry_task"),
		schedulerReplanRefForTaskV0("replan-prev-2", taskRef, "retry_task"),
		schedulerReplanRefForTaskV0("replan-prev-3", taskRef, "split_task"),
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusNeedsDirectorV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
	)
}

// Por debajo del limite, el replan resolutivo procede normal (no escala).
func TestBuildDirectorSchedulerTickV0ReplanBajoLimiteNoEscalaV0(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)
	taskRef := "task-ref-scheduler-001"
	input.Snapshot.ReplanRefs = []string{
		schedulerReplanRefForTaskV0("replan-prev-1", taskRef, "retry_task"),
		schedulerReplanRefForTaskV0("replan-prev-2", taskRef, "retry_task"),
	}

	plan := mustSchedulerTickPlanV0(t, input)

	// Con 2 replans previos (< 3) el flujo normal de replan procede: NO debe
	// quedarse solo en RecordReplanDecision por el guard.
	if plan.Status == SchedulerTickStatusNeedsDirectorV0 && len(plan.Commands) == 1 {
		t.Fatalf("no debe escalar por debajo del limite: status=%s commands=%d", plan.Status, len(plan.Commands))
	}
}

func TestSchedulerReplanCountForTaskV0(t *testing.T) {
	refs := []string{
		schedulerReplanRefForTaskV0("r1", "task-a", "retry_task"),
		schedulerReplanRefForTaskV0("r2", "task-b", "retry_task"),
		schedulerReplanRefForTaskV0("r3", "task-a", "split_task"),
		"ref-malformada-sin-task",
	}
	if got := schedulerReplanCountForTaskV0(refs, "task-a"); got != 2 {
		t.Fatalf("count task-a=%d want 2", got)
	}
	if got := schedulerReplanCountForTaskV0(refs, "task-b"); got != 1 {
		t.Fatalf("count task-b=%d want 1", got)
	}
}

func TestSchedulerReplanRetryLimitNoCapaTerminalesV0(t *testing.T) {
	snapshot := RunSchedulingSnapshotV0{ReplanRefs: []string{
		schedulerReplanRefForTaskV0("r1", "task-a", "retry_task"),
		schedulerReplanRefForTaskV0("r2", "task-a", "retry_task"),
		schedulerReplanRefForTaskV0("r3", "task-a", "retry_task"),
	}}
	// ask_director y abort_task no se capan aunque se supere el limite.
	if schedulerReplanRetryLimitExceededV0(snapshot, "task-a", orquestacoreworkflow.ReplanDecisionActionAskDirectorV0) {
		t.Fatalf("ask_director no debe capar")
	}
	if schedulerReplanRetryLimitExceededV0(snapshot, "task-a", orquestacoreworkflow.ReplanDecisionActionAbortTaskV0) {
		t.Fatalf("abort_task no debe capar")
	}
	// retry_task sí cae bajo el cap al superar el limite.
	if !schedulerReplanRetryLimitExceededV0(snapshot, "task-a", orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) {
		t.Fatalf("retry_task debe capar al superar el limite")
	}
}
