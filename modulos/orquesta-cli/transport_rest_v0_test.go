package orquestacli

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewCLIRESTClientConfigV0NormalizaBaseYTimeout(t *testing.T) {
	config, err := newCLIRESTClientConfigV0("https://api.example.test/", 0, "/api/v0/test")
	if err != nil {
		t.Fatalf("newCLIRESTClientConfigV0: %v", err)
	}
	if config.BaseURL != "https://api.example.test" {
		t.Fatalf("base_url inesperada: %q", config.BaseURL)
	}
	if config.Timeout != CliDefaultTimeoutRESTV0 {
		t.Fatalf("timeout inesperado: %v", config.Timeout)
	}
	if config.Endpoint != "/api/v0/test" {
		t.Fatalf("endpoint inesperado: %q", config.Endpoint)
	}
}

func TestPrepareCLIRESTRequestV0SeteaHeadersCanonicos(t *testing.T) {
	req, err := prepareCLIRESTRequestV0(context.Background(), "https://api.example.test", "/api/v0/test", "corr-123", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("prepareCLIRESTRequestV0: %v", err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("method inesperado: %s", req.Method)
	}
	if req.URL.String() != "https://api.example.test/api/v0/test" {
		t.Fatalf("url inesperada: %s", req.URL.String())
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type inesperado: %q", got)
	}
	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("accept inesperado: %q", got)
	}
	if got := req.Header.Get(CliCorrelationHeaderV0); got != "corr-123" {
		t.Fatalf("correlation header inesperado: %q", got)
	}
}

func TestHTTPClientFromConfigV0RespetaClienteInyectado(t *testing.T) {
	custom := &http.Client{Timeout: 3 * time.Second}
	got := httpClientFromConfigV0(cliRESTClientConfigV0{HTTPClient: custom}, time.Second)
	if got != custom {
		t.Fatalf("se esperaba reutilizar el cliente inyectado")
	}
}
