package orquestaserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRuntimeV0HTTPServerConfiguraLimitesDeRecursosV0(t *testing.T) {
	runtime := newControlPlaneRuntimeForTestV0(t, ConfigV0{
		Addr:     "127.0.0.1:8787",
		StateDir: t.TempDir(),
		HTTPResourceLimits: HTTPResourceLimitsV0{
			ReadHeaderTimeout: 3 * time.Second,
			ReadTimeout:       4 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       6 * time.Second,
			MaxHeaderBytes:    8192,
		},
	})

	server := runtime.httpServerV0()
	if server.ReadHeaderTimeout != 3*time.Second ||
		server.ReadTimeout != 4*time.Second ||
		server.WriteTimeout != 5*time.Second ||
		server.IdleTimeout != 6*time.Second ||
		server.MaxHeaderBytes != 8192 {
		t.Fatalf("limites http no aplicados: %+v", server)
	}
}

func TestServerOperationalStatusV0RechazaJSONTrailingDataV0(t *testing.T) {
	runtime := newControlPlaneRuntimeForTestV0(t, ConfigV0{
		Addr:     "127.0.0.1:8787",
		StateDir: t.TempDir(),
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/operational-status/query", strings.NewReader(`{} {}`))

	runtime.HandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "request_body_trailing_data") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
