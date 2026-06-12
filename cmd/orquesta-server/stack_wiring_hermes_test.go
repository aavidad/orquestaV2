package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"orquesta/modulos/orquesta-app-gateway/inprocesshttp"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
	operatorhermes "orquesta/modulos/orquesta-operator-mcp-hermes"
)

func TestBuildServerAppHandlerV0CableaHermesAPIComoOperatorConnector(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	remoteRefs := map[string]string{}
	hermesHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			t.Fatalf("hermes path=%s want /mcp", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-ref-hermes-stack-test" {
			t.Fatalf("authorization header inesperado")
		}
		var request struct {
			Method string `json:"method"`
			Params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode hermes request: %v", err)
		}
		remoteRefs[request.Params.Name] = hermesConnectorRefFromStackWiringInputV0(t, request.Params.Name, request.Params.Arguments)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(hermesJSONRPCResponseForStackWiringV0(request.Params.Name)))
	})
	oldFactory := newHermesOperatorMCPConnectorV0
	newHermesOperatorMCPConnectorV0 = func(config operatorhermes.HermesOperatorMCPConfigV0) (operator.OperatorMCPConnectorV0, error) {
		config.HTTPClient = &http.Client{Transport: inprocesshttp.TransportV0{Handler: hermesHandler}}
		return operatorhermes.NewHermesOperatorMCPConnectorV0(config)
	}
	t.Cleanup(func() { newHermesOperatorMCPConnectorV0 = oldFactory })
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_HERMES_ENABLED", "1")
	t.Setenv("ORQUESTA_HERMES_BASE_URL", "http://hermes-stack.local")
	t.Setenv("ORQUESTA_HERMES_API_KEY", "token-ref-hermes-stack-test")
	t.Setenv("ORQUESTA_HERMES_STATUS_TOOL", "hermes.status")
	t.Setenv("ORQUESTA_HERMES_BURST_TOOL", "hermes.burst")
	t.Setenv("ORQUESTA_HERMES_OUTBOX_TOOL", "hermes.outbox")
	t.Setenv("ORQUESTA_HERMES_QUERY_TOOL", "hermes.query")
	t.Setenv("ORQUESTA_HERMES_STATUS_CONNECTOR_REF", "hermes-status-ref")
	t.Setenv("ORQUESTA_HERMES_BURST_CONNECTOR_REF", "hermes-burst-ref")
	t.Setenv("ORQUESTA_HERMES_OUTBOX_CONNECTOR_REF", "hermes-outbox-ref")
	t.Setenv("ORQUESTA_HERMES_QUERY_CONNECTOR_REF", "hermes-query-ref")

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

	resources := callMCPStackWiringV0(t, handler, "resources/list", map[string]any{})
	assertMCPListContainsStackWiringV0(t, resources.Result, "resources", orquestamcp.MCPOperatorOperationsResourceNameV0)
	tools := callMCPStackWiringV0(t, handler, "tools/list", map[string]any{})
	for _, name := range []string{
		operator.OperatorMCPStatusToolNameV0,
		operator.OperatorMCPBurstToolNameV0,
		operator.OperatorMCPOutboxToolNameV0,
		operator.OperatorMCPDirectedQueryToolV0,
	} {
		assertMCPListContainsStackWiringV0(t, tools.Result, "tools", name)
	}

	for _, item := range []struct {
		localTool      string
		remoteTool     string
		wantRef        string
		arguments      map[string]any
		wantPayloadKey string
	}{
		{
			localTool:      operator.OperatorMCPStatusToolNameV0,
			remoteTool:     "hermes.status",
			wantRef:        "hermes-status-ref",
			wantPayloadKey: "status",
			arguments: map[string]any{
				"request_ref":          "request-ref-hermes-status",
				"subject_ref":          "run-ref-hermes-status",
				"status_connector_ref": "ignored-local-status-ref",
			},
		},
		{
			localTool:      operator.OperatorMCPBurstToolNameV0,
			remoteTool:     "hermes.burst",
			wantRef:        "hermes-burst-ref",
			wantPayloadKey: "burst",
			arguments: map[string]any{
				"request_ref":         "request-ref-hermes-burst",
				"run_ref":             "run-ref-hermes-burst",
				"burst_connector_ref": "ignored-local-burst-ref",
				"supervision_ref":     "supervision-ref-hermes-burst",
				"max_steps":           3,
			},
		},
		{
			localTool:      operator.OperatorMCPOutboxToolNameV0,
			remoteTool:     "hermes.outbox",
			wantRef:        "hermes-outbox-ref",
			wantPayloadKey: "outbox",
			arguments: map[string]any{
				"request_ref":          "request-ref-hermes-outbox",
				"subject_ref":          "run-ref-hermes-outbox",
				"outbox_connector_ref": "ignored-local-outbox-ref",
				"limit":                5,
			},
		},
		{
			localTool:      operator.OperatorMCPDirectedQueryToolV0,
			remoteTool:     "hermes.query",
			wantRef:        "hermes-query-ref",
			wantPayloadKey: "directed_query",
			arguments: map[string]any{
				"query_ref":           "query-ref-hermes",
				"target_ref":          "director-ref-hermes",
				"query_connector_ref": "ignored-local-query-ref",
				"question":            "Falta alguna decision publica?",
			},
		},
	} {
		response := callMCPStackWiringV0(t, handler, "tools/call", map[string]any{
			"name":      item.localTool,
			"arguments": item.arguments,
		})
		payload := mcpToolPayloadStackWiringV0(t, response.Result)
		if payload["estado"] != "ok" || payload[item.wantPayloadKey] == nil ||
			remoteRefs[item.remoteTool] != item.wantRef {
			t.Fatalf("tool=%s payload=%+v remoteRefs=%+v", item.localTool, payload, remoteRefs)
		}
	}
}

func hermesConnectorRefFromStackWiringInputV0(t *testing.T, tool string, raw json.RawMessage) string {
	t.Helper()
	switch tool {
	case "hermes.status":
		var input operator.OperatorStatusQueryV0
		if err := json.Unmarshal(raw, &input); err != nil {
			t.Fatalf("decode status args: %v", err)
		}
		return input.StatusConnectorRef
	case "hermes.burst":
		var input operator.OperatorSupervisedBurstRequestV0
		if err := json.Unmarshal(raw, &input); err != nil {
			t.Fatalf("decode burst args: %v", err)
		}
		return input.BurstConnectorRef
	case "hermes.outbox":
		var input operator.OperatorPendingOutboxQueryV0
		if err := json.Unmarshal(raw, &input); err != nil {
			t.Fatalf("decode outbox args: %v", err)
		}
		return input.OutboxConnectorRef
	case "hermes.query":
		var input operator.OperatorDirectedQueryV0
		if err := json.Unmarshal(raw, &input); err != nil {
			t.Fatalf("decode query args: %v", err)
		}
		return input.QueryConnectorRef
	default:
		t.Fatalf("tool remoto inesperado: %s", tool)
		return ""
	}
}

func hermesJSONRPCResponseForStackWiringV0(tool string) string {
	payloads := map[string]string{
		"hermes.status": `{\"estado\":\"ok\",\"status\":{\"status\":\"healthy\",\"evidence_refs\":[\"evidence-ref-hermes-stack-status\"]}}`,
		"hermes.burst":  `{\"estado\":\"ok\",\"burst\":{\"burst_ref\":\"burst-ref-hermes-stack\",\"executed_steps\":3,\"final_action\":\"continue\"}}`,
		"hermes.outbox": `{\"estado\":\"ok\",\"outbox\":{\"pending_count\":1,\"items\":[{\"message_ref\":\"message-ref-hermes-stack\",\"kind\":\"director_question\"}]}}`,
		"hermes.query":  `{\"estado\":\"ok\",\"directed_query\":{\"accepted\":true,\"answer_ref\":\"answer-ref-hermes-stack\",\"next_action\":\"wait\"}}`,
	}
	payload := payloads[tool]
	if payload == "" {
		payload = `{\"estado\":\"error\",\"error_code\":\"operator_mcp_port_error\"}`
	}
	return `{"jsonrpc":"2.0","id":"hermes-stack-test","result":{"content":[{"type":"text","mimeType":"application/json","text":"` + payload + `"}]}}`
}
