package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	for _, resource := range resources.Resources {
		if !orquestamcp.ValidateMCPResourceDescriptorSourceV0(resource.DescriptorSource) {
			t.Fatalf("resource %s sin descriptor_source valido: %+v", resource.Name, resource.DescriptorSource)
		}
		assertMCPRealDescriptorSourceNoSensitiveTestV0(t, resource.DescriptorSource)
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

func TestMCPRealTransportV0RechazaBodyTooLargeYTrailingV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, tc := range []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "content_type", contentType: "text/plain", body: `{"jsonrpc":"2.0","id":1,"method":"ping"}`, want: "request_content_type_invalido"},
		{name: "trailing", body: `{"jsonrpc":"2.0","id":1,"method":"ping"} {}`, want: "request_body_trailing_data"},
		{name: "too_large", body: `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"x":"` + strings.Repeat("a", mcpJSONRPCMaxBodyBytesV0+1) + `"}}`, want: "request_body_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, mcpRealHTTPPathV0, strings.NewReader(tc.body))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Header().Get("X-Orquesta-Control-Plane-Header-Policy") != "control-plane-mcp-v0" {
				t.Fatalf("security headers=%v", rec.Header())
			}
			if !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("body=%s want=%s", rec.Body.String(), tc.want)
			}
		})
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

func TestMCPRealTransportV0ToleraArgumentsComoStringJSONObjectV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var result mcpToolCallResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name":      "orquesta.status.v0",
		"arguments": `{"include_recent_errors":true}`,
	}, &result)
	if len(result.Content) == 0 {
		t.Fatalf("tool result inesperado: %+v", result)
	}
}

func TestMCPRealTransportV0ArgumentsStringInvalidoDevuelveInvalidParamsV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	rpc := callMCPJSONRPCRawTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name":      "orquesta.status.v0",
		"arguments": "no es json",
	})
	if rpc.Error == nil ||
		rpc.Error.Code != -32602 ||
		rpc.Error.Message != "mcp_invalid_params" ||
		rpc.Error.Data["error_code"] != "mcp_tool_arguments_string_not_json_object" {
		t.Fatalf("rpc error inesperado: %+v", rpc.Error)
	}
}

func TestMCPRealTransportV0WorkspaceTimelineAceptaLast30mComoStringV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{
		WorkspaceTimeline: newServerWorkspaceTimelineSourceV0(orquestamcp.MCPTransportBindingsV0{
			RunQueuePriority: fakeWorkspaceTimelineQueueV0{now: time.Now().UTC()},
		}),
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var result mcpToolCallResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name": orquestamcp.MCPWorkspaceTimelineToolNameV0,
		"arguments": map[string]any{
			"schema_version": "workspace_timeline_query.v0",
			"request_id":     "request-ref-mcp-workspace-timeline-last30m",
			"correlation_id": "corr-mcp-workspace-timeline-last30m",
			"scope":          "workspace",
			"time_window":    "last_30m",
			"page":           map[string]any{"limit": 10},
			"sources":        []string{"run_queue"},
		},
	}, &result)
	if result.IsError || len(result.Content) != 1 {
		t.Fatalf("result=%+v", result)
	}
	var toolResult orquestamcp.MCPWorkspaceTimelineToolResultV0
	if err := json.Unmarshal([]byte(result.Content[0].Text), &toolResult); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if toolResult.Estado != orquestamcp.MCPWorkspaceTimelineEstadoOKV0 ||
		toolResult.Timeline == nil ||
		toolResult.Timeline.TimeWindow.Preset != "last_30m" ||
		toolResult.Timeline.Counters["tasks_closed"] != 1 ||
		toolResult.Timeline.Counters["events"] != 2 {
		t.Fatalf("toolResult=%+v", toolResult)
	}
}

func TestMCPRealTransportV0WorkspaceTimelineEntradaInvalidaNoHandlerErrorV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{
		WorkspaceTimeline: newServerWorkspaceTimelineSourceV0(orquestamcp.MCPTransportBindingsV0{
			RunQueuePriority: fakeWorkspaceTimelineQueueV0{now: time.Now().UTC()},
		}),
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	rpc := callMCPJSONRPCRawTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name": orquestamcp.MCPWorkspaceTimelineToolNameV0,
		"arguments": map[string]any{
			"schema_version": "workspace_timeline_query.v0",
			"request_id":     "request-ref-mcp-workspace-timeline-invalid",
			"correlation_id": "corr-mcp-workspace-timeline-invalid",
			"task_ref":       "/home/alberto/prompts/raw.txt",
			"scope":          "workspace",
			"time_window":    123,
			"page":           map[string]any{"limit": 10},
			"sources":        []string{"run_queue"},
		},
	})
	encoded, _ := json.Marshal(rpc)
	if rpc.Error == nil ||
		rpc.Error.Message == "mcp_tool_handler_error" ||
		rpc.Error.Data["error_code"] != "workspace_timeline_time_window_invalid" ||
		strings.Contains(string(encoded), "/home/alberto") {
		t.Fatalf("rpc=%s", string(encoded))
	}
}

func callMCPJSONRPCTestV0(t *testing.T, endpoint string, method string, params any, output any) {
	t.Helper()
	rpc := callMCPJSONRPCRawTestV0(t, endpoint, method, params)
	if rpc.Error != nil {
		t.Fatalf("rpc error=%+v", rpc.Error)
	}
	if err := json.Unmarshal(rpc.Result, output); err != nil {
		t.Fatalf("decode result: %v", err)
	}
}

func callMCPJSONRPCRawTestV0(t *testing.T, endpoint string, method string, params any) mcpJSONRPCRawResponseV0 {
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
	return rpc
}

func mcpToolByNameTestV0(tools []mcpToolDescriptorV0, name string) *mcpToolDescriptorV0 {
	for idx := range tools {
		if tools[idx].Name == name {
			return &tools[idx]
		}
	}
	return nil
}
