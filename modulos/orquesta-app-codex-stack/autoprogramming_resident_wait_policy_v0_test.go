package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingResidentModeV0PermiteOverrideEsperaAmpliaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackRunSupervisorExecutorV0(&stack)

	result, err := executor.Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:        "request-autoprogramming-resident-wait-policy-001",
		CorrelationID:    "corr-autoprogramming-resident-wait-policy-001",
		ResidentMode:     true,
		MaxExternalWaits: 3,
	})
	if err != nil {
		t.Fatalf("Execute resident: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-resident-external-wait-policy") {
		t.Fatalf("result=%+v", result)
	}
}

func TestAutoprogrammingResidentModeV0BloqueaOverrideEsperaDescontroladaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackRunSupervisorExecutorV0(&stack)

	result, err := executor.Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:        "request-autoprogramming-resident-wait-policy-002",
		CorrelationID:    "corr-autoprogramming-resident-wait-policy-002",
		ResidentMode:     true,
		MaxExternalWaits: 71,
	})
	if err != nil {
		t.Fatalf("Execute resident: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "resident_external_wait_budget_incompatible" ||
		result.Errores[0].Field != "max_external_waits" ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-resident-external-wait-policy") ||
		len(result.NextActions) == 0 {
		t.Fatalf("result=%+v", result)
	}
}
