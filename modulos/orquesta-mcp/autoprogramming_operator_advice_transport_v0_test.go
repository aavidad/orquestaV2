package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPAutoprogrammingStatusTransportV0ConservaOperatorAdviceV0(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingStatusToolNameV0, map[string]any{
		"request_id": "request-ref-status-advice-transport-001",
		"run_ref":    "run-ref-status-advice-transport-001",
		"operator_advice": []map[string]any{{
			"run":  "run-ref-status-advice-alias-transport-001",
			"kind": "pause",
			"text": "esperar revision humana",
		}},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result mcpAutoprogrammingStatusTransportResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertAutoprogrammingAdviceTransportV0(t, result.OperatorAdvice, "run-ref-status-advice-alias-transport-001")
	if len(result.Diagnostics) == 0 {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}

func TestMCPAutoprogrammingSuperviseTransportV0ConservaOperatorAdviceV0(t *testing.T) {
	supervisor := &fakeAutoprogrammingAdviceSupervisorV0{}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{RunSupervisor: supervisor}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingSuperviseToolNameV0, map[string]any{
		"request_id": "request-ref-supervise-advice-transport-001",
		"run_ref":    "run-ref-supervise-advice-transport-001",
		"max_ticks":  1,
		"operator_advice": []map[string]any{{
			"subject_ref": "run-ref-supervise-advice-alias-transport-001",
			"advice":      "consultar operador",
		}},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if supervisor.input.RunRef != "run-ref-supervise-advice-transport-001" || supervisor.input.MaxTicks != 1 {
		t.Fatalf("input=%+v", supervisor.input)
	}
	var result mcpAutoprogrammingSuperviseTransportResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertAutoprogrammingAdviceTransportV0(t, result.OperatorAdvice, "run-ref-supervise-advice-alias-transport-001")
	if len(result.Diagnostics) == 0 {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}

type fakeAutoprogrammingAdviceSupervisorV0 struct {
	input MCPRunSupervisorToolInputV0
}

func (fake *fakeAutoprogrammingAdviceSupervisorV0) Execute(
	_ context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	fake.input = input
	return NewMCPRunSupervisorOKResultV0(input, input.RunRef, "tick_done", input.MaxTicks, MCPRunSupervisorSnapshotV0{}, nil), nil
}

func assertAutoprogrammingAdviceTransportV0(
	t *testing.T,
	advice []MCPAutoprogrammingOperatorAdviceV0,
	wantTarget string,
) {
	t.Helper()
	if len(advice) != 1 ||
		advice[0].TargetRef != wantTarget ||
		advice[0].Action != "advise" ||
		!advice[0].NonBlocking {
		t.Fatalf("operator_advice=%+v", advice)
	}
}
