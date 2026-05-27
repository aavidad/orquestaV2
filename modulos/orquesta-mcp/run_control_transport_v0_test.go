package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPTransportV0RunControlQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPRunControlToolNameV0, MCPRunControlToolInputV0{})
	if err != nil {
		t.Fatalf("call run control unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 ||
		result.Tool != MCPRunControlToolNameV0 {
		t.Fatalf("run control debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0RunControlInvocaExecutor(t *testing.T) {
	port := &fakeMCPRunControlPortV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunControl: NewMCPRunControlToolExecutorV0(port),
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPRunControlToolNameV0,
		MCPRunControlToolInputV0{
			RequestID: "request-ref-run-control-transport-001",
			Action:    "pause",
			RunRef:    "run-ref-transport-001",
		},
	)
	if err != nil {
		t.Fatalf("call run control: %v", err)
	}
	var result MCPRunControlToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != "paused" ||
		port.pause.RunRef != "run-ref-transport-001" {
		t.Fatalf("result=%+v port=%+v", result, port)
	}
}
