package orquestamcp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestDecodeMCPHTTPJSONResponseV0AplicaLimiteContentTypeYTrailing(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "content type", contentType: "text/html", body: `{"ok":true}`, want: "mcp_response_content_type"},
		{name: "trailing", contentType: "application/json", body: `{"ok":true} {}`, want: "mcp_response_trailing_data"},
		{name: "too large", contentType: "application/json", body: strings.Repeat(" ", int(mcpHTTPResponseMaxBytesV0)+1), want: "mcp_response_too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target map[string]any
			err := decodeMCPHTTPJSONResponseV0(mcpResponseForTestV0(tt.contentType, tt.body), &target)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("error=%v want=%s", err, tt.want)
			}
		})
	}
}

func TestDecodeMCPHTTPJSONResponseV0AceptaLegacyTextPlainJSON(t *testing.T) {
	var target map[string]any
	err := decodeMCPHTTPJSONResponseV0(mcpResponseForTestV0("text/plain; charset=utf-8", `{"ok":true}`), &target)
	if err != nil {
		t.Fatalf("error=%v", err)
	}
	if target["ok"] != true {
		t.Fatalf("target=%v", target)
	}
}

func TestMCPNuevaAppToolExecutorV0NoPropagaBodyNo2XX(t *testing.T) {
	executor, err := newLocalMCPNuevaAppToolExecutorV0(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(MCPNuevaAppToolCorrelationHeaderV0, "corr-http-500")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`secret_token=abc local_path=/tmp/private`))
	}))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = executor.Execute(newMCPTestContextV0(), minimalMCPNuevaAppInputV0())

	if err == nil {
		t.Fatalf("error esperado")
	}
	got := err.Error()
	if !strings.Contains(got, "status 500") || !strings.Contains(got, "corr-http-500") {
		t.Fatalf("error publico sin status/correlation: %q", got)
	}
	if strings.Contains(got, "secret_token") || strings.Contains(got, "local_path") || strings.Contains(got, "/tmp/private") {
		t.Fatalf("error filtra body: %q", got)
	}
}

func TestMCPNuevaAppToolExecutorV0LimitaError400(t *testing.T) {
	executor, err := newLocalMCPNuevaAppToolExecutorV0(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strings.Repeat(" ", int(mcpHTTPResponseMaxBytesV0)+1)))
	}))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = executor.Execute(newMCPTestContextV0(), minimalMCPNuevaAppInputV0())

	if err == nil || !strings.Contains(err.Error(), "mcp_response_too_large") {
		t.Fatalf("error limite esperado: %v", err)
	}
}

func mcpResponseForTestV0(contentType string, body string) *http.Response {
	header := http.Header{}
	if strings.TrimSpace(contentType) != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}

func newMCPTestContextV0() context.Context {
	return context.Background()
}

func minimalMCPNuevaAppInputV0() MCPNuevaAppToolInputV0 {
	return MCPNuevaAppToolInputV0{
		RequestID:     "req-http-redaction",
		CorrelationID: "corr-http-redaction",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			Locale:        "es",
			Nombre:        "Agenda",
			Objetivo:      "Coordinar ensayos",
			TipoApp:       "web",
		},
	}
}
