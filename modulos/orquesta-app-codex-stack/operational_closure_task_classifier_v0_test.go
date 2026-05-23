package orquestaappcodexstack

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestOperationalClosureTaskClassifierV0AceptaMarkerEnContextRefsV0(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		TaskID:      "task-autoprogramming-classifier-001",
		ContextRefs: []string{autoprogrammingBridgeOperationalTaskSourceRefV0},
	}
	if !codexStackWorkflowTaskLooksOperationalDirectorV0(task) {
		t.Fatalf("context_refs=%v no reconocidos como senal operacional", task.ContextRefs)
	}
}
