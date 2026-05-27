package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestMCPRealTransportV0BloqueaResourceSobrePresupuestoSinEcoV0(t *testing.T) {
	handler := newMCPBudgetTestHandlerV0(t, func(registry *mcpRealTransportRegistryV0) {
		err := registry.RegisterResourceV0(orquestamcp.MCPTransportResourceEnvelopeV0{
			Name:         "orquesta.test.large_resource.v0",
			URI:          "orquesta://test/large-resource/v0",
			ContentType:  "application/json",
			OutputBudget: smallMCPBudgetForTestV0(orquestamcp.MCPTransportResourcePayloadBlockedV0),
			Handler: func(context.Context) (json.RawMessage, error) {
				return json.RawMessage(`{"payload":"` + strings.Repeat("/home/alberto/token", 8) + `"}`), nil
			},
		})
		if err != nil {
			t.Fatalf("register resource: %v", err)
		}
	})

	rpc := postMCPJSONRPCRawBodyTestV0(t, handler, `{
		"jsonrpc":"2.0",
		"id":"request-ref-budget-resource",
		"method":"resources/read",
		"params":{"name":"orquesta.test.large_resource.v0"}
	}`, "")
	assertMCPBudgetRPCErrorTestV0(t, rpc, orquestamcp.MCPTransportResourcePayloadBlockedV0)
}

func TestMCPRealTransportV0BloqueaToolSobrePresupuestoSinEcoV0(t *testing.T) {
	handler := newMCPBudgetTestHandlerV0(t, func(registry *mcpRealTransportRegistryV0) {
		err := registry.RegisterToolV0(orquestamcp.MCPTransportToolEnvelopeV0{
			Name:         "orquesta.test.large_tool.v0",
			OutputBudget: smallMCPBudgetForTestV0(orquestamcp.MCPTransportToolPayloadBlockedV0),
			Handler: func(context.Context, json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"result":"` + strings.Repeat("raw prompt token ", 8) + `"}`), nil
			},
		})
		if err != nil {
			t.Fatalf("register tool: %v", err)
		}
	})

	rpc := postMCPJSONRPCRawBodyTestV0(t, handler, `{
		"jsonrpc":"2.0",
		"id":"request-ref-budget-tool",
		"method":"tools/call",
		"params":{"name":"orquesta.test.large_tool.v0","arguments":{"secret":"dont_echo"}}
	}`, "")
	assertMCPBudgetRPCErrorTestV0(t, rpc, orquestamcp.MCPTransportToolPayloadBlockedV0)
}

func newMCPBudgetTestHandlerV0(
	t *testing.T,
	register func(*mcpRealTransportRegistryV0),
) http.Handler {
	t.Helper()
	registry := &mcpRealTransportRegistryV0{
		resourcesByName: map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		resourcesByURI:  map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		tools:           map[string]orquestamcp.MCPTransportToolEnvelopeV0{},
	}
	register(registry)
	mux := http.NewServeMux()
	mux.HandleFunc(mcpRealHTTPPathV0, registry.serveJSONRPCV0)
	return mux
}

func smallMCPBudgetForTestV0(code string) orquestamcp.MCPTransportOutputBudgetV0 {
	return orquestamcp.MCPTransportOutputBudgetV0{
		MaxBytes:  24,
		Mode:      orquestamcp.MCPTransportOutputModeFullV0,
		Freshness: orquestamcp.MCPTransportOutputFreshnessLiveV0,
		Redaction: orquestamcp.MCPTransportOutputRedactionPublicV0,
		Overflow:  code,
	}
}

func assertMCPBudgetRPCErrorTestV0(
	t *testing.T,
	rpc mcpJSONRPCRawResponseV0,
	wantCode string,
) {
	t.Helper()
	encoded, _ := json.Marshal(rpc)
	text := strings.ToLower(string(encoded))
	if rpc.Error == nil ||
		rpc.Error.Message != orquestamcp.MCPTransportOutputTooLargeV0 ||
		rpc.Error.Data["error_code"] != wantCode ||
		rpc.Error.Data["output_mode"] != orquestamcp.MCPTransportOutputModeBlockedV0 {
		t.Fatalf("rpc=%s", string(encoded))
	}
	for _, forbidden := range []string{"/home/", "token", "prompt", "dont_echo", "secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("rpc filtra %q: %s", forbidden, text)
		}
	}
}
