package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPTransportV0AutoprogrammingStatusQuedaOptInSinPuertos(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{},
	)
	if err != nil {
		t.Fatalf("call autoprogramming status unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.Tool != MCPAutoprogrammingStatusToolNameV0 ||
		result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("autoprogramming status debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}
