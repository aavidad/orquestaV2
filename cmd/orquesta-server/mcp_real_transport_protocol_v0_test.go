package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestMCPRealTransportV0RechazaJSONRPCEstrictoV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "version", body: `{"id":"request-ref-1","method":"ping"}`, want: "mcp_jsonrpc_version_invalid"},
		{name: "method", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":" "}`, want: "mcp_method_required"},
		{name: "id_bool", body: `{"jsonrpc":"2.0","id":true,"method":"ping"}`, want: "mcp_id_invalid"},
		{name: "id_object", body: `{"jsonrpc":"2.0","id":{"x":1},"method":"ping"}`, want: "mcp_id_invalid"},
		{name: "id_too_large", body: `{"jsonrpc":"2.0","id":"` + strings.Repeat("a", mcpJSONRPCMaxIDStringBytesV0+1) + `","method":"ping"}`, want: "mcp_id_invalid"},
		{name: "batch", body: `[{"jsonrpc":"2.0","id":"request-ref-1","method":"ping"}]`, want: "mcp_batch_unsupported"},
		{name: "notification", body: `{"jsonrpc":"2.0","method":"ping"}`, want: "mcp_notification_unsupported"},
		{name: "notification_with_id", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":"notifications/initialized"}`, want: "mcp_notification_id_not_allowed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rpc := postMCPJSONRPCRawBodyTestV0(t, handler, tc.body, "")
			if rpc.Error == nil || rpc.Error.Data["error_code"] != tc.want {
				t.Fatalf("rpc=%+v want=%s", rpc.Error, tc.want)
			}
		})
	}
}

func TestMCPRealTransportV0InitializedNotificationAceptadaV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	response := postMCPJSONRPCHTTPTestV0(t, handler, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, "")
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if response.StatusCode != http.StatusAccepted || strings.TrimSpace(string(body)) != "" {
		t.Fatalf("status=%d body=%s", response.StatusCode, string(body))
	}
}

func TestMCPRealTransportV0ValidaAcceptYParamsEstrictoV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, tc := range []struct {
		name   string
		accept string
		body   string
		want   string
	}{
		{name: "accept", accept: "text/html", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":"ping"}`, want: "request_accept_invalido"},
		{name: "params_budget", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":"resources/read","params":{"name":"` + strings.Repeat("a", mcpJSONRPCReadMaxParamsBytesV0+1) + `"}}`, want: "mcp_params_too_large"},
		{name: "resource_unknown", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":"resources/read","params":{"name":"orquesta.status.v0","extra":1}}`, want: "mcp_resource_params_invalid"},
		{name: "resource_ambiguous", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":"resources/read","params":{"name":"orquesta.status.v0","uri":"orquesta://status"}}`, want: "mcp_resource_params_invalid"},
		{name: "tool_unknown", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":"tools/call","params":{"name":"orquesta.status.v0","extra":1}}`, want: "mcp_tool_params_invalid"},
		{name: "tool_arguments_array", body: `{"jsonrpc":"2.0","id":"request-ref-1","method":"tools/call","params":{"name":"orquesta.status.v0","arguments":[]}}`, want: "mcp_tool_params_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rpc := postMCPJSONRPCRawBodyTestV0(t, handler, tc.body, tc.accept)
			if rpc.Error == nil || rpc.Error.Data["error_code"] != tc.want {
				t.Fatalf("rpc=%+v want=%s", rpc.Error, tc.want)
			}
		})
	}
}

func postMCPJSONRPCRawBodyTestV0(
	t *testing.T,
	handler http.Handler,
	body string,
	accept string,
) mcpJSONRPCRawResponseV0 {
	t.Helper()
	response := postMCPJSONRPCHTTPTestV0(t, handler, body, accept)
	defer response.Body.Close()
	var rpc mcpJSONRPCRawResponseV0
	if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
		t.Fatalf("decode rpc: %v", err)
	}
	return rpc
}

func postMCPJSONRPCHTTPTestV0(
	t *testing.T,
	handler http.Handler,
	body string,
	accept string,
) *http.Response {
	t.Helper()
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)
	request, err := http.NewRequest(http.MethodPost, server.URL+mcpRealHTTPPathV0, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	client := commandHTTPClientWithRedirectPolicyV0(2*time.Second, server.URL)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", response.StatusCode)
	}
	return response
}
