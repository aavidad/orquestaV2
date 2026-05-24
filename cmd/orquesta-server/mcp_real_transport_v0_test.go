package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestMCPRealTransportSmokeOptInV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	summary, err := runMCPRealSmokeWithHandlerV0(ctx, handler)
	if err != nil {
		t.Fatalf("smoke: %v", err)
	}
	if summary.Status != "completed" ||
		!summary.ResourceRead ||
		!summary.SelfImprovement ||
		summary.OperatorErrorCode != operator.ErrOperatorMCPPortUnavailableV0 {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestMCPRealSmokeCommandV0RequiereConfirmacion(t *testing.T) {
	t.Setenv(mcpRealSmokeConfirmEnvV0, "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := mcpRealSmokeCommandV0(&stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), mcpRealSmokeConfirmEnvV0) {
		t.Fatalf("stderr=%s", stderr.String())
	}
}

func TestMCPRealTransportV0ExponeJSONRPCListasCompactas(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var resources mcpResourceListResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "resources/list", map[string]any{}, &resources)
	if len(resources.Resources) == 0 {
		t.Fatalf("resources vacio")
	}
	var tools mcpToolListResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/list", map[string]any{}, &tools)
	if len(tools.Tools) == 0 {
		t.Fatalf("tools vacio")
	}
	if tools.Tools[0].InputSchema["type"] != "object" {
		t.Fatalf("inputSchema no compatible MCP: %+v", tools.Tools[0].InputSchema)
	}
	statusTool := mcpToolByNameTestV0(tools.Tools, "orquesta.status.v0")
	if statusTool == nil ||
		!strings.Contains(statusTool.Description, "No internal refs required") {
		t.Fatalf("status tool no autodescriptivo: %+v", statusTool)
	}
	operatorTool := mcpToolByNameTestV0(tools.Tools, "orquesta.operator.status.query.v0")
	if operatorTool == nil {
		t.Fatalf("operator status no registrado")
	}
	required, _ := operatorTool.InputSchema["required"].([]any)
	if len(required) == 0 {
		t.Fatalf("operator status sin required schema: %+v", operatorTool.InputSchema)
	}
}

func TestMCPRealTransportV0HandshakeCompatibleClienteMCP(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var init mcpInitializeResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "hermes-probe",
			"version": "0",
		},
	}, &init)
	if init.ProtocolVersion == "" || init.ServerInfo.Name != "orquesta-mcp" {
		t.Fatalf("initialize inesperado: %+v", init)
	}

	var pong map[string]any
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "ping", map[string]any{}, &pong)
}

func callMCPJSONRPCTestV0(t *testing.T, endpoint string, method string, params any, output any) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": mcpJSONRPCVersionV0,
		"id":      "request-ref-mcp-real-test",
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	response, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	var rpc mcpJSONRPCRawResponseV0
	if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
		t.Fatalf("decode rpc: %v", err)
	}
	if rpc.Error != nil {
		t.Fatalf("rpc error=%+v", rpc.Error)
	}
	if err := json.Unmarshal(rpc.Result, output); err != nil {
		t.Fatalf("decode result: %v", err)
	}
}

func mcpToolByNameTestV0(tools []mcpToolDescriptorV0, name string) *mcpToolDescriptorV0 {
	for idx := range tools {
		if tools[idx].Name == name {
			return &tools[idx]
		}
	}
	return nil
}
