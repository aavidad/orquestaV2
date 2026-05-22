package orquestastatefile

import (
	"context"
	"testing"
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
