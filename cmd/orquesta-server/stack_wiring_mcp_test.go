package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestBuildServerAppHandlerV0MontaMCPNativoV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":"mcp-test","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`)
	req := httptest.NewRequest(http.MethodPost, mcpRealHTTPPathV0, body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response mcpJSONRPCResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("rpc error=%+v", response.Error)
	}
}

type mcpStackWiringResponseV0 struct {
	Error  *mcpJSONRPCErrorV0 `json:"error,omitempty"`
	Result json.RawMessage    `json:"result,omitempty"`
}

func callMCPStackWiringV0(t *testing.T, handler http.Handler, method string, params map[string]any) mcpStackWiringResponseV0 {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      "mcp-hermes-stack-test",
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, mcpRealHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("method=%s status=%d body=%s", method, rec.Code, rec.Body.String())
	}
	var response mcpStackWiringResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("method=%s rpc error=%+v", method, response.Error)
	}
	return response
}

func assertMCPListContainsStackWiringV0(t *testing.T, raw json.RawMessage, field string, name string) {
	t.Helper()
	var result map[string][]struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode %s list: %v", field, err)
	}
	for _, item := range result[field] {
		if item.Name == name {
			return
		}
	}
	t.Fatalf("%s no contiene %s: %+v", field, name, result[field])
}

func mcpToolPayloadStackWiringV0(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content inesperado: %+v", result.Content)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("decode payload: %v text=%s", err, result.Content[0].Text)
	}
	return payload
}
