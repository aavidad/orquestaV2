package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestWorkflowTaskWaitAgentRefsV0FiltraPorOlaYCohorte(t *testing.T) {
	runRef := "run-workflow-task-wait-refs-001"
	first := workflowTaskForWaitRefsTestV0(runRef, "task-wait-a", "wave-01", "cohort-a")
	second := workflowTaskForWaitRefsTestV0(runRef, "task-wait-b", "wave-01", "cohort-a")
	other := workflowTaskForWaitRefsTestV0(runRef, "task-wait-c", "wave-02", "cohort-b")
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{first.TaskID, second.TaskID, other.TaskID}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(first, second, other),
		run,
		WorkflowTaskWaitFilterV0{WaveRef: " wave-01 ", CohortRef: "cohort-a"},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	want := []string{
		WorkflowTaskAgentRequestRefV0(first.TaskID),
		WorkflowTaskAgentRequestRefV0(second.TaskID),
	}
	if !sameStringsForTestV0(refs, want) {
		t.Fatalf("refs=%v want=%v", refs, want)
	}
}

func TestBuildWorkflowTaskWaitSnapshotV0IncluyePendientesAcotados(t *testing.T) {
	runRef := "run-workflow-task-wait-snapshot-001"
	first := workflowTaskForWaitRefsTestV0(runRef, "task-wait-snapshot-a", "wave-01", "cohort-a")
	second := workflowTaskForWaitRefsTestV0(runRef, "task-wait-snapshot-b", "wave-01", "cohort-a")
	unrequested := workflowTaskForWaitRefsTestV0(runRef, "task-wait-snapshot-unrequested", "wave-01", "cohort-a")
	other := workflowTaskForWaitRefsTestV0(runRef, "task-wait-snapshot-c", "wave-02", "cohort-b")
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{first.TaskID, second.TaskID, unrequested.TaskID, other.TaskID}
	run.StartedAgents = []string{
		WorkflowTaskAgentRequestRefV0(first.TaskID),
		WorkflowTaskAgentRequestRefV0(second.TaskID),
		WorkflowTaskAgentRequestRefV0(other.TaskID),
		"agent-ref-ajeno-vivo",
	}
	run.DeliveredAgents = []string{WorkflowTaskAgentRequestRefV0(second.TaskID)}

	snapshot, err := BuildWorkflowTaskWaitSnapshotV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(first, second, unrequested, other),
		run,
		WorkflowTaskWaitFilterV0{WaveRef: "wave-01", CohortRef: "cohort-a"},
	)
	if err != nil {
		t.Fatalf("BuildWorkflowTaskWaitSnapshotV0: %v", err)
	}
	if !sameStringsForTestV0(snapshot.TaskRefs, []string{first.TaskID, second.TaskID, unrequested.TaskID}) {
		t.Fatalf("task_refs=%v", snapshot.TaskRefs)
	}
	if !sameStringsForTestV0(snapshot.AgentRefs, []string{
		WorkflowTaskAgentRequestRefV0(first.TaskID),
		WorkflowTaskAgentRequestRefV0(second.TaskID),
		WorkflowTaskAgentRequestRefV0(unrequested.TaskID),
	}) {
		t.Fatalf("agent_refs=%v", snapshot.AgentRefs)
	}
	if !sameStringsForTestV0(snapshot.PendingAgentRefs, []string{
		WorkflowTaskAgentRequestRefV0(first.TaskID),
		WorkflowTaskAgentRequestRefV0(unrequested.TaskID),
	}) {
		t.Fatalf("pending_agent_refs=%v", snapshot.PendingAgentRefs)
	}
}

func TestWorkflowTaskWaitAgentRefsV0FiltraPorParentTask(t *testing.T) {
	runRef := "run-workflow-task-wait-parent-001"
	child := workflowTaskForWaitRefsTestV0(runRef, "task-child-a", "wave-02", "cohort-a")
	child.ParentTaskRef = "task-parent-001"
	other := workflowTaskForWaitRefsTestV0(runRef, "task-child-b", "wave-02", "cohort-a")
	other.ParentTaskRef = "task-parent-002"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{child.TaskID, other.TaskID}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(child, other),
		run,
		WorkflowTaskWaitFilterV0{ParentTaskRef: "task-parent-001"},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	want := []string{WorkflowTaskAgentRequestRefV0(child.TaskID)}
	if !sameStringsForTestV0(refs, want) {
		t.Fatalf("refs=%v want=%v", refs, want)
	}
}

func TestWorkflowTaskWaitSnapshotV0ParentTaskRefAcotaSubarbol(t *testing.T) {
	runRef := "run-workflow-task-wait-parent-subtree-001"
	parent := workflowTaskForWaitRefsTestV0(runRef, "task-parent-subtree-001", "wave-01", "cohort-root")
	childA := workflowTaskForWaitRefsTestV0(runRef, "task-child-subtree-a", "wave-02", "cohort-subtree")
	childA.ParentTaskRef = parent.TaskID
	childB := workflowTaskForWaitRefsTestV0(runRef, "task-child-subtree-b", "wave-02", "cohort-subtree")
	childB.ParentTaskRef = parent.TaskID
	closedChild := workflowTaskForWaitRefsTestV0(runRef, "task-child-subtree-closed", "wave-02", "cohort-subtree")
	closedChild.ParentTaskRef = parent.TaskID
	sibling := workflowTaskForWaitRefsTestV0(runRef, "task-child-sibling", "wave-02", "cohort-subtree")
	sibling.ParentTaskRef = "task-parent-other"
	otherRoot := workflowTaskForWaitRefsTestV0(runRef, "task-root-other", "wave-01", "cohort-root")
	parent.ChildTaskRefs = []string{childA.TaskID, childB.TaskID, closedChild.TaskID}

	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{
		parent.TaskID,
		childA.TaskID,
		childB.TaskID,
		closedChild.TaskID,
		sibling.TaskID,
		otherRoot.TaskID,
	}
	run.ClosedTasks = []string{closedChild.TaskID}
	run.StartedAgents = []string{
		WorkflowTaskAgentRequestRefV0(parent.TaskID),
		WorkflowTaskAgentRequestRefV0(childA.TaskID),
		WorkflowTaskAgentRequestRefV0(childB.TaskID),
		WorkflowTaskAgentRequestRefV0(closedChild.TaskID),
		WorkflowTaskAgentRequestRefV0(sibling.TaskID),
		WorkflowTaskAgentRequestRefV0(otherRoot.TaskID),
		"agent-ref-ajeno-vivo",
	}
	run.DeliveredAgents = []string{WorkflowTaskAgentRequestRefV0(childB.TaskID)}

	snapshot, err := BuildWorkflowTaskWaitSnapshotV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(parent, childA, childB, closedChild, sibling, otherRoot),
		run,
		WorkflowTaskWaitFilterV0{ParentTaskRef: " " + parent.TaskID + " "},
	)
	if err != nil {
		t.Fatalf("BuildWorkflowTaskWaitSnapshotV0: %v", err)
	}
	if snapshot.Filter.ParentTaskRef != parent.TaskID {
		t.Fatalf("parent_task_ref=%q want=%q", snapshot.Filter.ParentTaskRef, parent.TaskID)
	}
	if !sameStringsForTestV0(snapshot.TaskRefs, []string{childA.TaskID, childB.TaskID}) {
		t.Fatalf("task_refs=%v", snapshot.TaskRefs)
	}
	if !sameStringsForTestV0(snapshot.AgentRefs, []string{
		WorkflowTaskAgentRequestRefV0(childA.TaskID),
		WorkflowTaskAgentRequestRefV0(childB.TaskID),
	}) {
		t.Fatalf("agent_refs=%v", snapshot.AgentRefs)
	}
	if !sameStringsForTestV0(snapshot.PendingAgentRefs, []string{WorkflowTaskAgentRequestRefV0(childA.TaskID)}) {
		t.Fatalf("pending_agent_refs=%v", snapshot.PendingAgentRefs)
	}
}

func TestWorkflowTaskWaitAgentRefsV0FiltroVacioNoEsperaTodo(t *testing.T) {
	runRef := "run-workflow-task-wait-empty-001"
	task := workflowTaskForWaitRefsTestV0(runRef, "task-wait-empty", "wave-01", "cohort-a")
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{task.TaskID}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(task),
		run,
		WorkflowTaskWaitFilterV0{},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("refs=%v", refs)
	}
}

func TestWorkflowTaskWaitAgentRefsV0FiltroVacioPermiteStoreNil(t *testing.T) {
	runRef := "run-workflow-task-wait-empty-nil-store-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{"task-wait-empty-nil-store"}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		nil,
		run,
		WorkflowTaskWaitFilterV0{},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("refs=%v", refs)
	}
}

func TestWorkflowTaskWaitAgentRefsV0FiltroNoVacioRequiereStore(t *testing.T) {
	run := mustActiveProgrammingRunV0(t, "run-workflow-task-wait-nil-store-001")

	_, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		nil,
		run,
		WorkflowTaskWaitFilterV0{WaveRef: "wave-01"},
	)
	if err == nil {
		t.Fatal("WorkflowTaskWaitAgentRefsV0 err=nil, want error")
	}
}

func TestWorkflowTaskWaitAgentRefsV0DeduplicaRunTasksYRefs(t *testing.T) {
	runRef := "run-workflow-task-wait-dedup-001"
	task := workflowTaskForWaitRefsTestV0(runRef, "task-wait-dedup", "wave-01", "cohort-a")
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{" ", task.TaskID, " " + task.TaskID + " ", task.TaskID}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(task),
		run,
		WorkflowTaskWaitFilterV0{WaveRef: "wave-01"},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	want := []string{WorkflowTaskAgentRequestRefV0(task.TaskID)}
	if !sameStringsForTestV0(refs, want) {
		t.Fatalf("refs=%v want=%v", refs, want)
	}
}

func TestWorkflowTaskWaitAgentRefsV0NoUsaTasksFueraDelRun(t *testing.T) {
	runRef := "run-workflow-task-wait-run-filter-001"
	inRun := workflowTaskForWaitRefsTestV0(runRef, "task-wait-run-filter-a", "wave-02", "cohort-b")
	outsideRunTasks := workflowTaskForWaitRefsTestV0(runRef, "task-wait-run-filter-b", "wave-01", "cohort-a")
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{inRun.TaskID}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(inRun, outsideRunTasks),
		run,
		WorkflowTaskWaitFilterV0{WaveRef: "wave-01", CohortRef: "cohort-a"},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("refs=%v, want empty", refs)
	}
}

func TestWorkflowTaskWaitAgentRefsV0OmiteTasksCerradas(t *testing.T) {
	runRef := "run-workflow-task-wait-closed-001"
	open := workflowTaskForWaitRefsTestV0(runRef, "task-wait-open", "wave-01", "cohort-a")
	closed := workflowTaskForWaitRefsTestV0(runRef, "task-wait-closed", "wave-01", "cohort-a")
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{open.TaskID, closed.TaskID}
	run.ClosedTasks = []string{closed.TaskID}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(open, closed),
		run,
		WorkflowTaskWaitFilterV0{WaveRef: "wave-01", CohortRef: "cohort-a"},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	want := []string{WorkflowTaskAgentRequestRefV0(open.TaskID)}
	if !sameStringsForTestV0(refs, want) {
		t.Fatalf("refs=%v want=%v", refs, want)
	}
}

func TestWorkflowTaskWaitAgentRefsV0SinCoincidencias(t *testing.T) {
	runRef := "run-workflow-task-wait-no-match-001"
	task := workflowTaskForWaitRefsTestV0(runRef, "task-wait-no-match", "wave-02", "cohort-b")
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{task.TaskID}

	refs, err := WorkflowTaskWaitAgentRefsV0(
		context.Background(),
		NewInMemoryWorkflowTaskStoreV0(task),
		run,
		WorkflowTaskWaitFilterV0{WaveRef: "wave-01", CohortRef: "cohort-a"},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("refs=%v, want empty", refs)
	}
}

func workflowTaskForWaitRefsTestV0(
	runRef string,
	taskRef string,
	waveRef string,
	cohortRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	task := workflowTaskForCandidateProviderTestV0(runRef, taskRef, []string{"docs/" + taskRef})
	task.WaveRef = waveRef
	task.CohortRef = cohortRef
	return task
}
