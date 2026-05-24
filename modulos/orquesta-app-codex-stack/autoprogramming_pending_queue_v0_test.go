package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackAutoprogrammingPrepareRunAPIV0NoRelanzaAgentePendienteEnTicksGlobalesV0(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-pending-queue-api-001",
		CorrelationID:          "corr-autoprogramming-pending-queue-api-001",
		OccurredAt:             "2026-05-22T11:35:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || prepared.RunRef == "" || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}

	first, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 first: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(first); len(got) != 1 ||
		got[0] != prepared.RunRef || runtime.launchCountV0() != 1 {
		t.Fatalf("first executions=%v launches=%d result=%+v", got, runtime.launchCountV0(), first)
	}

	second, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 second: %v", err)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if runtime.launchCountV0() != 1 ||
		len(compactStringsV0(run.StartedAgents)) != 1 ||
		!autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) ||
		!drainRunHasPendingExternalAgentsV0(run, nil) {
		t.Fatalf("second=%+v run=%+v launches=%d wait=%v", second, run, runtime.launchCountV0(), prepared.WaitAgentRefs)
	}
}
