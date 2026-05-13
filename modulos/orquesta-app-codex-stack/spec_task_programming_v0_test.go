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
		RequiredTests: []string{"go test ./..."},
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
	for _, want := range []string{
		"go.mod",
		"cmd/server",
		"imports de modulo",
		"sin imports relativos ../",
	} {
		if !strings.Contains(got.Objective, want) && !codexStackDoneCriteriaContainsForTestV0(got.DoneCriteria, want) {
			t.Fatalf("contrato Go completo no contiene %q: objective=%s criteria=%v", want, got.Objective, got.DoneCriteria)
		}
	}
}
