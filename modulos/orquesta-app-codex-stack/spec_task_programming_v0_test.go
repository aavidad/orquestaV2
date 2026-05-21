package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgrammingTaskV0PropagaRequiredTestsYContratoGoCompleto(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		TaskID:   "task-programacion-api-001",
		RunID:    "run-programacion-api-001",
		PhaseID:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:    "Crear API Go",
		Summary:  "Implementar API de agenda.",
		WriteSet: []string{"cmd/server", "internal/agenda"},
		AcceptanceCriteria: []string{
			"go.mod e imports de modulo",
			"sin imports relativos ../",
		},
		ParentTaskRef:   "task-ref-parent-001",
		CohortRef:       "cohort-ref-recursive-001",
		WaveRef:         "wave-ref-recursive-001",
		DelegationDepth: 2,
		MaxChildAgents:  6,
		ChildTaskRefs:   []string{"task-ref-child-002"},
		RequiredTests:   []string{"go test ./..."},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:agenda-api:v0",
			FunctionName: "CrearAPI",
		}},
	}
	resolver := CodexLaunchSpecResolverV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}

	got, err := resolver.agentTaskV0(context.Background(), orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   task.RunID,
		PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef: task.TaskID,
	}, "programacion")
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	if len(got.RequiredTests) != 1 || got.RequiredTests[0] != "go test ./..." {
		t.Fatalf("required_tests=%v", got.RequiredTests)
	}
	if got.ParentTaskRef != "task-ref-parent-001" ||
		got.CohortRef != "cohort-ref-recursive-001" ||
		got.WaveRef != "wave-ref-recursive-001" ||
		got.DelegationDepth != 2 ||
		got.MaxChildAgents != 6 ||
		len(got.ChildTaskRefs) != 1 ||
		got.ChildTaskRefs[0] != "task-ref-child-002" {
		t.Fatalf("linaje no propagado: %+v", got)
	}
	for _, want := range []string{
		"contrato de esta tarea completa",
		"go.mod",
		"cmd/server",
		"imports de modulo",
		"sin imports relativos ../",
	} {
		if !strings.Contains(got.Objective, want) && !codexStackDoneCriteriaContainsForTestV0(got.DoneCriteria, want) {
			t.Fatalf("contrato Go completo no contiene %q: objective=%s criteria=%v", want, got.Objective, got.DoneCriteria)
		}
	}
	if strings.Contains(got.Objective, "microtarea") {
		t.Fatalf("objective no debe pedir microtareas: %s", got.Objective)
	}
}

func TestProgrammingTaskV0TrabajoExternoUsaUnidadTrabajoNoMicrotareaMinima(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		TaskID:  "task-programacion-opes-001",
		RunID:   "run-programacion-opes-001",
		PhaseID: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:   "Redactar capitulo OPES",
		Summary: "Redactar unidad editorial amplia con paquete de dominio suficiente.",
		WriteSet: []string{
			"external/opes/draft_content_block",
		},
		AcceptanceCriteria: []string{
			"usar paquete de dominio suficiente",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:app-change:opes:v0",
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	resolver := CodexLaunchSpecResolverV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}

	got, err := resolver.agentTaskV0(context.Background(), orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   task.RunID,
		PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef: task.TaskID,
	}, "programacion")
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	if !strings.Contains(got.Objective, "unidad de trabajo externa") ||
		strings.Contains(got.Objective, "microtarea") {
		t.Fatalf("objective=%s", got.Objective)
	}
}
