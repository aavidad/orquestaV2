package orquestastatefile

import (
	"context"
	"strings"
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
	childA.WaveRef = "wave-ref-state-file-by-parent"
	childA.CohortRef = "cohort-ref-state-file-by-parent"
	childA.DelegationDepth = 2
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
	index, ok, err := readJSONFileV0[workflowTaskParentIndexDocumentV0](recovered.workflowTaskParentIndexPathV0(runRef))
	if err != nil || !ok {
		t.Fatalf("indice parent no persistido: ok=%v err=%v", ok, err)
	}
	entry := index.Tasks[childA.TaskID]
	if entry.RunRef != runRef ||
		entry.TaskRef != childA.TaskID ||
		entry.ParentTaskRef != parent.TaskID ||
		entry.WaveRef != childA.WaveRef ||
		entry.CohortRef != childA.CohortRef ||
		entry.DelegationDepth != childA.DelegationDepth {
		t.Fatalf("entrada indice incompleta: %+v", entry)
	}
	if _, err := recovered.LoadWorkflowTasksByParentV0(ctx, runRef, " "); err == nil {
		t.Fatal("expected parent_task_ref requerido")
	}
}

func TestStoreV0LoadWorkflowTasksByParentV0UsaIndiceSinRebuildIlimitado(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	runRef := "run-state-file-by-parent-index-001"
	parent := mustWorkflowTaskV0(t, runRef)
	parent.TaskID = "task-ref-parent-state-file-index"
	childA := mustWorkflowTaskV0(t, runRef)
	childA.TaskID = "task-ref-child-state-file-index-a"
	childA.ParentTaskRef = parent.TaskID
	childB := mustWorkflowTaskV0(t, runRef)
	childB.TaskID = "task-ref-child-state-file-index-b"
	childB.ParentTaskRef = parent.TaskID

	store := mustStoreV0(t, rootDir)
	for _, task := range []orquestacoreworkflow.WorkflowTaskV0{parent, childA, childB} {
		if err := store.SaveWorkflowTaskV0(ctx, task); err != nil {
			t.Fatalf("SaveWorkflowTaskV0(%s): %v", task.TaskID, err)
		}
	}
	recovered, err := NewStoreV0(ConfigV0{
		RootDir:                                  rootDir,
		MaxWorkflowTaskParentIndexRebuildEntries: 1,
	})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	got, err := recovered.LoadWorkflowTasksByParentV0(ctx, runRef, parent.TaskID)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksByParentV0: %v", err)
	}
	if len(got) != 2 || got[0].TaskID != childA.TaskID || got[1].TaskID != childB.TaskID {
		t.Fatalf("children=%+v", got)
	}
}

func TestStoreV0LoadWorkflowTasksByParentV0BloqueaRebuildSinPresupuesto(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	runRef := "run-state-file-by-parent-rebuild-budget"
	parent := mustWorkflowTaskV0(t, runRef)
	parent.TaskID = "task-ref-parent-state-file-rebuild-budget"
	childA := mustWorkflowTaskV0(t, runRef)
	childA.TaskID = "task-ref-child-state-file-rebuild-budget-a"
	childA.ParentTaskRef = parent.TaskID
	childB := mustWorkflowTaskV0(t, runRef)
	childB.TaskID = "task-ref-child-state-file-rebuild-budget-b"
	childB.ParentTaskRef = parent.TaskID
	store := mustStoreV0(t, rootDir)
	for _, task := range []orquestacoreworkflow.WorkflowTaskV0{parent, childA, childB} {
		document := workflowTaskDocumentV0{
			SchemaVersion: workflowTaskDocumentSchemaV0,
			RunRef:        task.RunID,
			TaskRef:       task.TaskID,
			Task:          task,
		}
		if err := writeJSONAtomicV0(store.workflowTaskPathV0(task.RunID, task.TaskID), document); err != nil {
			t.Fatalf("write task document: %v", err)
		}
	}
	recovered, err := NewStoreV0(ConfigV0{
		RootDir:                                  rootDir,
		MaxWorkflowTaskParentIndexRebuildEntries: 1,
	})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	_, err = recovered.LoadWorkflowTasksByParentV0(ctx, runRef, parent.TaskID)
	if err == nil || !strings.Contains(err.Error(), "workflow_task_parent_index_budget_exhausted") {
		t.Fatalf("err=%v, want workflow_task_parent_index_budget_exhausted", err)
	}
}
