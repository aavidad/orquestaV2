package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingResidentModeV0NormalizaBacklogDurableV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackRunSupervisorExecutorV0(&stack)

	result, err := executor.Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-autoprogramming-resident-001",
		CorrelationID: "corr-autoprogramming-resident-001",
		ResidentMode:  true,
		MaxTicks:      1,
	})
	if err != nil {
		t.Fatalf("Execute resident: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.StopReason == "" ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-resident-mode") {
		t.Fatalf("result=%+v", result)
	}
}

func TestAutoprogrammingResidentModeV0TomaRunPreparadaDeColaV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-resident-queue-001",
		CorrelationID:          "corr-autoprogramming-resident-queue-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	})

	result := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-autoprogramming-resident-supervise-001",
		CorrelationID: "corr-autoprogramming-resident-queue-001",
		ResidentMode:  true,
		MaxTicks:      1,
	})
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != prepared.RunRef ||
		runtime.launchCountV0() != 1 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-resident-backlog-queue") {
		t.Fatalf("result=%+v prepared=%+v launches=%d", result, prepared, runtime.launchCountV0())
	}
}
