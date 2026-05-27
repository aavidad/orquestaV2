package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPRunControlHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPRunControlHTTPExecutorV0{
		result: MCPRunControlToolResultV0{
			Estado:        MCPRunControlEstadoOKV0,
			CorrelationID: "corr-run-control-http-result-001",
			RunRef:        "run-ref-control-http-001",
			Status:        "paused",
		},
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRunControlToolInputV0{
		Action:        "pause",
		RunRef:        "run-ref-control-http-001",
		RequestID:     "request-ref-run-control-http-001",
		CorrelationID: "corr-run-control-http-input-001",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRunControlHTTPPathV0, body)

	NewMCPRunControlHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-result-001" ||
		executor.input.RunRef != "run-ref-control-http-001" {
		t.Fatalf("status=%d headers=%v input=%+v body=%s", rec.Code, rec.Header(), executor.input, rec.Body.String())
	}
}

func TestMCPRunControlHTTPHandlerV0MetodoYExecutorNil(t *testing.T) {
	for _, tc := range []struct {
		name    string
		method  string
		handler http.Handler
		want    int
	}{
		{name: "metodo", method: http.MethodGet, handler: NewMCPRunControlHTTPHandlerV0(&fakeMCPRunControlHTTPExecutorV0{}), want: http.StatusMethodNotAllowed},
		{name: "nil", method: http.MethodPost, handler: NewMCPRunControlHTTPHandlerV0(nil), want: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, MCPRunControlHTTPPathV0, bytes.NewBufferString(`{}`))
			req.Header.Set("X-Correlation-ID", "corr-run-control-http-error-001")

			tc.handler.ServeHTTP(rec, req)

			if rec.Code != tc.want ||
				rec.Header().Get("X-Correlation-ID") != "corr-run-control-http-error-001" {
				t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
			}
		})
	}
}

type fakeMCPRunControlHTTPExecutorV0 struct {
	input  MCPRunControlToolInputV0
	result MCPRunControlToolResultV0
}

func (executor *fakeMCPRunControlHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolResultV0, error) {
	executor.input = input
	return executor.result, nil
}
