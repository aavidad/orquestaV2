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

func TestOperationalClosureTaskClassifierV0NoConfundeDomainWorkTaskSourceV0(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		TaskID:          "task-ref-app-change-classifier-001",
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		AcceptanceCriteria: []string{
			"operational_director.task_source: director_decision",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:app-change:classifier:v0",
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	if codexStackWorkflowTaskLooksOperationalDirectorV0(task) {
		t.Fatalf("domain_work externo no debe retenerse como cierre operativo: %+v", task)
	}
}
