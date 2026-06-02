package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
	operatorhermes "orquesta/modulos/orquesta-operator-mcp-hermes"
)

const (
	envHermesRealSmokeConfirmV0      = "ORQUESTA_HERMES_REAL_SMOKE_CONFIRM"
	envHermesRealSmokeBurstConfirmV0 = "ORQUESTA_HERMES_REAL_SMOKE_BURST_CONFIRM"
)

func TestHermesOperatorRealSmokeStatusOutboxQueryV0(t *testing.T) {
	endpoint := newHermesOperatorRealSmokeEndpointV0(t)
	assertHermesOperatorRealSmokeSurfaceV0(t, endpoint)

	status := callHermesOperatorRealSmokeToolV0(t, endpoint, operator.OperatorMCPStatusToolNameV0, map[string]any{
		"request_ref":          "request-ref-hermes-real-smoke-status-001",
		"subject_ref":          "run-ref-hermes-real-smoke-status-001",
		"status_connector_ref": hermesRealSmokeEnvOrDefaultV0(envHermesStatusConnectorRefV0, "status-connector-ref-hermes-real-smoke-001"),
		"include_sections":     []string{"summary"},
	})
	if status.Status == nil {
		t.Fatalf("status payload ausente")
	}

	outbox := callHermesOperatorRealSmokeToolV0(t, endpoint, operator.OperatorMCPOutboxToolNameV0, map[string]any{
		"request_ref":          "request-ref-hermes-real-smoke-outbox-001",
		"subject_ref":          "run-ref-hermes-real-smoke-outbox-001",
		"outbox_connector_ref": hermesRealSmokeEnvOrDefaultV0(envHermesOutboxConnectorRefV0, "outbox-connector-ref-hermes-real-smoke-001"),
		"limit":                5,
	})
	if outbox.Outbox == nil {
		t.Fatalf("outbox payload ausente")
	}

	query := callHermesOperatorRealSmokeToolV0(t, endpoint, operator.OperatorMCPDirectedQueryToolV0, map[string]any{
		"query_ref":           "query-ref-hermes-real-smoke-001",
		"target_ref":          "operator-ref-hermes-real-smoke-001",
		"query_connector_ref": hermesRealSmokeEnvOrDefaultV0(envHermesQueryConnectorRefV0, "query-connector-ref-hermes-real-smoke-001"),
		"question":            "Smoke Hermes API/MCP: confirma estado publico sin efectos externos.",
	})
	if query.DirectedQuery == nil {
		t.Fatalf("directed_query payload ausente")
	}
}

func TestHermesOperatorRealSmokeSupervisedBurstV0(t *testing.T) {
	requireHermesOperatorRealSmokeConfirmedV0(t)
	if strings.TrimSpace(os.Getenv(envHermesRealSmokeBurstConfirmV0)) != "1" {
		t.Skipf("set %s=1 para ejecutar supervised_burst real", envHermesRealSmokeBurstConfirmV0)
	}
	endpoint := newHermesOperatorRealSmokeEndpointV0(t)

	burst := callHermesOperatorRealSmokeToolV0(t, endpoint, operator.OperatorMCPBurstToolNameV0, map[string]any{
		"request_ref":         "request-ref-hermes-real-smoke-burst-001",
		"run_ref":             "run-ref-hermes-real-smoke-burst-001",
		"burst_connector_ref": hermesRealSmokeEnvOrDefaultV0(envHermesBurstConnectorRefV0, "burst-connector-ref-hermes-real-smoke-001"),
		"supervision_ref":     "supervision-ref-hermes-real-smoke-001",
		"max_steps":           1,
		"evidence_refs":       []string{"evidence-ref-hermes-real-smoke-burst-confirmed-001"},
	})
	if burst.Burst == nil {
		t.Fatalf("burst payload ausente")
	}
}

func newHermesOperatorRealSmokeEndpointV0(t *testing.T) string {
	t.Helper()
	requireHermesOperatorRealSmokeConfirmedV0(t)
	baseURL := strings.TrimSpace(os.Getenv(envHermesBaseURLV0))
	if baseURL == "" {
		t.Fatalf("set %s para ejecutar el smoke real Hermes API/MCP", envHermesBaseURLV0)
	}
	handler := newHermesOperatorRealSmokeHandlerV0(t, baseURL)
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)
	return server.URL + mcpRealHTTPPathV0
}

func requireHermesOperatorRealSmokeConfirmedV0(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv(envHermesRealSmokeConfirmV0)) != "1" {
		t.Skipf("set %s=1 para ejecutar el smoke real Hermes API/MCP", envHermesRealSmokeConfirmV0)
	}
}

func newHermesOperatorRealSmokeHandlerV0(t *testing.T, baseURL string) http.Handler {
	t.Helper()
	t.Setenv(envHermesEnabledV0, "1")
	t.Setenv(envHermesBaseURLV0, baseURL)

	connector, err := newHermesOperatorRealSmokeConnectorV0()
	if err != nil {
		t.Fatalf("hermes connector: %v", err)
	}
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{
		OperatorConnector: connector,
	})
	if err != nil {
		t.Fatalf("mcp handler: %v", err)
	}
	return handler
}

func newHermesOperatorRealSmokeConnectorV0() (operator.OperatorMCPConnectorV0, error) {
	config := hermesOperatorEnvConfigFromEnvV0()
	return operatorhermes.NewHermesOperatorMCPConnectorV0(operatorhermes.HermesOperatorMCPConfigV0{
		BaseURL:          config.BaseURL,
		MCPPath:          config.MCPPath,
		APIKey:           config.APIKey,
		ToolNames:        config.ToolNames,
		ConnectorRefs:    config.ConnectorRefs,
		Timeout:          time.Duration(config.TimeoutSeconds) * time.Second,
		HTTPClient:       hermesOperatorRealSmokeHTTPClientV0(),
		MaxRequestBytes:  int64(config.MaxRequestBytes),
		MaxResponseBytes: int64(config.MaxResponseBytes),
	})
}

func hermesOperatorRealSmokeHTTPClientV0() *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func assertHermesOperatorRealSmokeSurfaceV0(t *testing.T, endpoint string) {
	t.Helper()
	var resources mcpResourceListResultV0
	callMCPJSONRPCTestV0(t, endpoint, "resources/list", map[string]any{}, &resources)
	if !hermesRealSmokeResourceExistsV0(resources.Resources, orquestamcp.MCPOperatorOperationsResourceNameV0) {
		t.Fatalf("resources/list no expone %s: %+v", orquestamcp.MCPOperatorOperationsResourceNameV0, resources.Resources)
	}

	var tools mcpToolListResultV0
	callMCPJSONRPCTestV0(t, endpoint, "tools/list", map[string]any{}, &tools)
	for _, name := range []string{
		operator.OperatorMCPStatusToolNameV0,
		operator.OperatorMCPOutboxToolNameV0,
		operator.OperatorMCPDirectedQueryToolV0,
		operator.OperatorMCPBurstToolNameV0,
	} {
		if mcpToolByNameTestV0(tools.Tools, name) == nil {
			t.Fatalf("tools/list no expone %s: %+v", name, tools.Tools)
		}
	}
}

func callHermesOperatorRealSmokeToolV0(
	t *testing.T,
	endpoint string,
	toolName string,
	arguments map[string]any,
) orquestamcp.MCPOperatorToolResultV0 {
	t.Helper()
	var result mcpToolCallResultV0
	callMCPJSONRPCTestV0(t, endpoint, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": arguments,
	}, &result)
	if result.IsError || len(result.Content) != 1 {
		t.Fatalf("tool=%s result inesperado: is_error=%t content_count=%d", toolName, result.IsError, len(result.Content))
	}
	var payload orquestamcp.MCPOperatorToolResultV0
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("tool=%s decode payload: %v", toolName, err)
	}
	if payload.Estado != orquestamcp.MCPOperatorToolEstadoOKV0 {
		t.Fatalf("tool=%s estado=%s error_code=%s issues=%+v", toolName, payload.Estado, payload.ErrorCode, payload.Issues)
	}
	return payload
}

func hermesRealSmokeResourceExistsV0(resources []mcpResourceDescriptorV0, name string) bool {
	for _, resource := range resources {
		if resource.Name == name {
			return true
		}
	}
	return false
}

func hermesRealSmokeEnvOrDefaultV0(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
