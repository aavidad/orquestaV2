package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPTransportV0DirectorStatsQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(
		context.Background(),
		MCPDirectorStatsToolNameV0,
		MCPDirectorStatsToolInputV0{RunRef: "run-ref-mcp-director-stats-unbound-001"},
	)
	if err != nil {
		t.Fatalf("call director stats unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("director stats debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0DirectorStatsEntradaInvalidaDevuelvePayloadPublico(t *testing.T) {
	handler := mcpDirectorStatsTransportHandlerV0(nil)
	output, err := handler(
		context.Background(),
		json.RawMessage(`{"run_ref":"run-ref-mcp-director-stats-invalid-001","include_agent_progress":["summary"]}`),
	)
	if err != nil {
		t.Fatalf("entrada invalida no debe escapar como error MCP opaco: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode invalid input: %v", err)
	}
	if result.Tool != MCPDirectorStatsToolNameV0 || result.ErrorCode != MCPTransportToolInputInvalidV0 {
		t.Fatalf("payload invalido inesperado: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}
