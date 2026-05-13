package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPServerShutdownHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPServerShutdownHTTPExecutorV0{
		result: MCPServerShutdownToolResultV0{
			Estado:        MCPServerShutdownEstadoOKV0,
			CorrelationID: "corr-server-shutdown-http-result-001",
			ShutdownReady: true,
			RunsRequested: 1,
		},
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPServerShutdownToolInputV0{
		Forced:        true,
		CorrelationID: "corr-server-shutdown-http-input-001",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPServerShutdownHTTPPathV0, body)

	NewMCPServerShutdownHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-server-shutdown-http-result-001" ||
		!executor.input.Forced {
		t.Fatalf("status=%d headers=%v input=%+v body=%s", rec.Code, rec.Header(), executor.input, rec.Body.String())
	}
}

func TestMCPServerShutdownHTTPHandlerV0MetodoYExecutorNil(t *testing.T) {
	for _, tc := range []struct {
		name    string
		method  string
		handler http.Handler
		want    int
	}{
		{name: "metodo", method: http.MethodGet, handler: NewMCPServerShutdownHTTPHandlerV0(&fakeMCPServerShutdownHTTPExecutorV0{}), want: http.StatusMethodNotAllowed},
		{name: "nil", method: http.MethodPost, handler: NewMCPServerShutdownHTTPHandlerV0(nil), want: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, MCPServerShutdownHTTPPathV0, bytes.NewBufferString(`{}`))
			req.Header.Set("X-Correlation-ID", "corr-server-shutdown-http-error-001")

			tc.handler.ServeHTTP(rec, req)

			if rec.Code != tc.want ||
				rec.Header().Get("X-Correlation-ID") != "corr-server-shutdown-http-error-001" {
				t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
			}
		})
	}
}

type fakeMCPServerShutdownHTTPExecutorV0 struct {
	input  MCPServerShutdownToolInputV0
	result MCPServerShutdownToolResultV0
}

func (executor *fakeMCPServerShutdownHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPServerShutdownToolInputV0,
) (MCPServerShutdownToolResultV0, error) {
	executor.input = input
	return executor.result, nil
}
