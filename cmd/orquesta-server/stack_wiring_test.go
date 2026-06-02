package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestBuildStackFromEnvV0UsaConectoresDurablesFileBased(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}

	assertServerStackDurableStoresV0(t, stack)
}

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

func TestBuildServerAppHandlerV0CableaHermesAPIComoOperatorConnector(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	remoteRefs := map[string]string{}
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer hermes.Close()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_HERMES_ENABLED", "1")
	t.Setenv("ORQUESTA_HERMES_BASE_URL", hermes.URL)
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

func TestBuildStackFromEnvV0CableaDomainWorkFileOptIn(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.DomainWork == nil {
		t.Fatalf("DomainWork file no cableado")
	}
	if stack.DomainDelivery.Enabled {
		t.Fatalf("DomainDelivery no debe activarse con domain-work-file sin submitter")
	}
	result, err := stack.DomainWork.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "request-ref-stack-file-001",
			CorrelationID:  "corr-stack-file-001",
			IdempotencyKey: "idem-stack-file-001",
			RequestedBy:    "test",
			DomainRef:      "dominio-demo",
			WorkKind:       "generate_content_package",
			Objective:      "crear job generico desde stack",
		},
	})
	if err != nil {
		t.Fatalf("DomainWork.Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestBuildStackFromEnvV0CableaOPESFallbackParaDomainWorkYDeliveryV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "http://127.0.0.1:18082")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.DomainWork == nil {
		t.Fatalf("DomainWork OPES fallback no cableado")
	}
	if !stack.DomainDelivery.Enabled {
		t.Fatalf("DomainDelivery debe activarse con OPES_BASE_URL fallback")
	}
}

func TestBuildStackFromEnvV0CableaHTTPNeutralParaDomainWorkYDeliveryV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", "http://127.0.0.1:18083")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE", "smoke_local")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if stack.DomainWork == nil {
		t.Fatalf("DomainWork HTTP neutral no cableado")
	}
	if !stack.DomainDelivery.Enabled {
		t.Fatalf("DomainDelivery debe activarse con HTTP neutral opt-in")
	}
}

func TestBuildStackFromEnvV0NoCableaRunnerLocalPorDefectoPeroAceptaReceiptsACK(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", "")
	t.Setenv("ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS", "")
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
	localRunner, err := requiredTestRunnerFromEnvV0(config, stack.Stores.RequiredTestEvidenceStore)
	if err != nil {
		t.Fatalf("requiredTestRunnerFromEnvV0: %v", err)
	}
	if localRunner != nil {
		t.Fatalf("runner local de comandos debe quedar apagado por defecto")
	}
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatalf("runner de receipts ACK debe quedar disponible por defecto")
	}
}

func TestBuildStackFromEnvV0CableaRequiredTestRunnerOptIn(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "1")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", filepath.Join(projectDir, "go-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", filepath.Join(t.TempDir(), "test-output"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_ENV", "")
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
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatalf("RequiredTestRunner opt-in no cableado")
	}
}

func assertServerStackDurableStoresV0(
	t *testing.T,
	stack orquestaappcodexstack.StackV0,
) {
	t.Helper()
	assertTypeV0[*orquestastatefile.StoreV0](t, "RunStore", stack.Stores.RunStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "EventSink", stack.Stores.EventSink)
	assertTypeV0[*orquestastatefile.StoreV0](t, "TaskStore", stack.Stores.TaskStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "WaitStateStore", stack.Stores.WaitStateStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "OperationalPlanStateWriter", stack.Stores.OperationalPlanStateWriter)
	assertTypeV0[*orquestastatefile.StoreV0](t, "OperationalPlanStateStore", stack.Stores.OperationalPlanStateStore)
	assertTypeV0[*orquestastatefile.StoreV0](t, "ProcessRegistry", stack.Stores.ProcessRegistry)
	assertTypeV0[*orquestapersistence.FileOutboxLedgerV0](t, "OutboxLedger", stack.Stores.OutboxLedger)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "AppChangeStore", stack.Stores.AppChangeStore)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "RunControl", stack.Stores.RunControl)
	assertTypeV0[*orquestarunfile.RunFileStoreV0](t, "RunQueue", stack.Stores.RunQueue)
	assertTypeV0[*orquestaruntimecodexdelivery.FileCodexReceiptDescriptorStoreV0](t, "ReceiptStore", stack.Stores.ReceiptStore)
	assertTypeV0[*orquestaruntimecodexdelivery.FileCodexProgressStateStoreV0](t, "ProgressState", stack.Stores.ProgressState)
}

func assertTypeV0[T any](t *testing.T, name string, value any) {
	t.Helper()
	if _, ok := value.(T); !ok {
		t.Fatalf("%s usa %T, debe usar %T", name, value, *new(T))
	}
}
