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

func TestMCPTransportV0AutoprogrammingStatusPublicaDiagnosticosConfiguradosDesdeBindings(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		AutoprogrammingStatusDiagnostics: []MCPAutoprogrammingDiagnosticV0{{
			Code:         "codex_goal_backend_degraded",
			Scope:        "app_goal",
			Message:      "codex goal backend degradado: codex_app_server_auth_missing",
			EvidenceRefs: []string{"evidence-ref-server-codex-goal-backend-degraded-app_goal"},
		}},
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{},
	)
	if err != nil {
		t.Fatalf("call autoprogramming status: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "codex_goal_backend_degraded") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}
