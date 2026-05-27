package orquestamcp

import (
	"errors"
	"testing"
)

func TestMCPPublicErrorCatalogV0CubreCodigosCompartidosYHTTPV0(t *testing.T) {
	for _, code := range []string{
		MCPPublicErrMethodNotAllowedV0,
		MCPPublicErrBodyInvalidV0,
		MCPPublicErrBodyTooLargeV0,
		MCPPublicErrTransportUnboundV0,
		MCPAutoprogrammingPrepareRunHTTPNotConfiguredCodeV0,
		MCPAutoprogrammingPrepareRunHTTPExecutorErrorCodeV0,
		MCPDomainWorkHTTPNotConfiguredCodeV0,
		MCPDomainWorkHTTPExecutorErrorCodeV0,
		MCPExternalWorkRunHTTPNotConfiguredCodeV0,
		MCPExternalWorkRunHTTPExecutorErrorCodeV0,
		"director_stats_executor_error",
		"server_shutdown_executor_error",
		"run_supervisor_execute_error",
		"run_supervisor_error",
		"workspace_timeline_http_error",
		"workspace_timeline_no_configurada",
		"workspace_timeline_no_disponible",
	} {
		if !MCPPublicErrorCodeKnownV0(code) {
			t.Fatalf("codigo MCP fuera de catalogo: %s", code)
		}
	}
}

func TestMCPPublicErrorCodeFromErrorV0AllowlistYFallbackV0(t *testing.T) {
	if got := publicMCPErrorCodeFromErrorV0(errors.New(MCPPublicErrTransportUnboundV0), MCPPublicErrExecutorV0); got != MCPPublicErrTransportUnboundV0 {
		t.Fatalf("allowlist=%q", got)
	}
	if got := publicMCPErrorCodeFromErrorV0(errors.New("db password=/tmp/private"), MCPPublicErrExecutorV0); got != MCPPublicErrExecutorV0 {
		t.Fatalf("fallback=%q", got)
	}
}

func TestPublicMCPExecutorErrorMessageFromErrorV0NoPropagaNoCatalogadosV0(t *testing.T) {
	if got := publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_status_executor_error", errors.New(MCPPublicErrTransportUnboundV0)); got != "autoprogramming_status_executor_error: "+MCPPublicErrTransportUnboundV0 {
		t.Fatalf("catalogado=%q", got)
	}
	if got := publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_status_executor_error", errors.New("db password=/tmp/private")); got != "autoprogramming_status_executor_error" {
		t.Fatalf("no catalogado=%q", got)
	}
}
