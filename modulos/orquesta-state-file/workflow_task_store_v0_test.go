package orquestastatefile

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestStoreV0RechazaWorkflowTaskConRefInternaInconsistente(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	task := mustWorkflowTaskV0(t, "run-state-file-001")
	document := workflowTaskDocumentV0{
		SchemaVersion: workflowTaskDocumentSchemaV0,
		RunRef:        task.RunID,
		TaskRef:       task.TaskID,
		Task:          task,
	}
	document.Task.RunID = "run-state-file-distinto"
	if err := writeJSONAtomicV0(store.workflowTaskPathV0(task.RunID, task.TaskID), document); err != nil {
		t.Fatalf("writeJSONAtomicV0: %v", err)
	}
	if _, err := store.LoadWorkflowTasksV0(context.Background(), task.RunID, []string{task.TaskID}); err == nil {
		t.Fatal("err=nil, want ref interna inconsistente")
	}
}

func TestStoreV0LoadWorkflowTasksByParentV0RecuperaHijosTrasRecrearInstancia(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	runRef := "run-state-file-by-parent-001"
	parent := mustWorkflowTaskV0(t, runRef)
	parent.TaskID = "task-ref-parent-state-file-by-parent"
	childA := mustWorkflowTaskV0(t, runRef)
	childA.TaskID = "task-ref-child-state-file-by-parent-a"
	childA.Title = "Hijo A persistido"
	childA.WriteSet = []string{"app/child_a.go"}
	childA.ParentTaskRef = parent.TaskID
	childB := mustWorkflowTaskV0(t, runRef)
	childB.TaskID = "task-ref-child-state-file-by-parent-b"
	childB.Title = "Hijo B persistido"
	childB.WriteSet = []string{"app/child_b.go"}
	childB.ParentTaskRef = " " + parent.TaskID + " "
	sibling := mustWorkflowTaskV0(t, runRef)
	sibling.TaskID = "task-ref-child-state-file-by-parent-sibling"
	sibling.WriteSet = []string{"app/sibling.go"}
	sibling.ParentTaskRef = "task-ref-parent-distinto"
	otherRunChild := mustWorkflowTaskV0(t, "run-state-file-by-parent-other")
	otherRunChild.TaskID = "task-ref-child-state-file-by-parent-other-run"
	otherRunChild.ParentTaskRef = parent.TaskID

	store := mustStoreV0(t, rootDir)
	for _, task := range []orquestacoreworkflow.WorkflowTaskV0{parent, childB, sibling, otherRunChild, childA} {
		if err := store.SaveWorkflowTaskV0(ctx, task); err != nil {
			t.Fatalf("SaveWorkflowTaskV0(%s): %v", task.TaskID, err)
		}
	}

	recovered := mustStoreV0(t, rootDir)
	got, err := recovered.LoadWorkflowTasksByParentV0(ctx, runRef, " "+parent.TaskID+" ")
	if err != nil {
		t.Fatalf("LoadWorkflowTasksByParentV0: %v", err)
	}
	if len(got) != 2 || got[0].TaskID != childA.TaskID || got[1].TaskID != childB.TaskID {
		t.Fatalf("children=%+v", got)
	}
	if got[0].ParentTaskRef != parent.TaskID || got[1].ParentTaskRef != parent.TaskID {
		t.Fatalf("parent refs no normalizadas: %+v", got)
	}
	if _, err := recovered.LoadWorkflowTasksByParentV0(ctx, runRef, " "); err == nil {
		t.Fatal("expected parent_task_ref requerido")
	}
}
