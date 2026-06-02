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
	var remoteTool string
	var remoteConnectorRef string
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
		remoteTool = request.Params.Name
		var input operator.OperatorStatusQueryV0
		if err := json.Unmarshal(request.Params.Arguments, &input); err != nil {
			t.Fatalf("decode hermes arguments: %v", err)
		}
		remoteConnectorRef = input.StatusConnectorRef
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jsonrpc":"2.0",
			"id":"hermes-stack-test",
			"result":{
				"content":[{
					"type":"text",
					"mimeType":"application/json",
					"text":"{\"estado\":\"ok\",\"status\":{\"status\":\"healthy\",\"evidence_refs\":[\"evidence-ref-hermes-stack-test\"]}}"
				}]
			}
		}`))
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
	t.Setenv("ORQUESTA_HERMES_STATUS_CONNECTOR_REF", "hermes-status-ref")

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
	body := bytes.NewBufferString(`{
		"jsonrpc":"2.0",
		"id":"mcp-hermes-stack-test",
		"method":"tools/call",
		"params":{
			"name":"orquesta.operator.status.query.v0",
			"arguments":{
				"request_ref":"request-ref-hermes-stack-test",
				"subject_ref":"run-ref-hermes-stack-test",
				"status_connector_ref":"ignored-local-ref"
			}
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, mcpRealHTTPPathV0, body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Error  *mcpJSONRPCErrorV0 `json:"error,omitempty"`
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("rpc error=%+v", response.Error)
	}
	if len(response.Result.Content) != 1 {
		t.Fatalf("content inesperado: %+v", response.Result.Content)
	}
	var payload struct {
		Estado string `json:"estado"`
		Status struct {
			Status string `json:"status"`
		} `json:"status"`
	}
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &payload); err != nil {
		t.Fatalf("decode payload: %v text=%s", err, response.Result.Content[0].Text)
	}
	if payload.Estado != "ok" ||
		payload.Status.Status != "healthy" ||
		remoteTool != "hermes.status" ||
		remoteConnectorRef != "hermes-status-ref" {
		t.Fatalf("payload=%+v remoteTool=%s remoteConnectorRef=%s", payload, remoteTool, remoteConnectorRef)
	}
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
