package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
		RequestID:     "request-ref-server-shutdown-http-001",
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

func TestMCPServerShutdownHTTPHandlerV0ActiveGoalsDevuelveConflict(t *testing.T) {
	executor := &fakeMCPServerShutdownHTTPExecutorV0{
		result: MCPServerShutdownToolResultV0{
			Estado:          MCPServerShutdownEstadoOKV0,
			CorrelationID:   "corr-active-goals-http",
			Status:          "active_goals_present",
			ShutdownReady:   false,
			ActiveWorkCount: 1,
			ActiveWorks: []MCPServerShutdownWorkV0{{
				Kind:    "goal_first",
				RunRef:  "run-active-goal-http",
				WorkRef: "goal-active-http",
				Status:  "running",
			}},
		},
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPServerShutdownToolInputV0{
		RequestID:   "req-active-goals-http",
		RequestedBy: "orquesta-director",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPServerShutdownHTTPPathV0, body)

	NewMCPServerShutdownHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict ||
		rec.Header().Get("X-Correlation-ID") != "corr-active-goals-http" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPServerShutdownToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != "active_goals_present" ||
		result.ActiveWorkCount != 1 ||
		len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].RunRef != "run-active-goal-http" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPServerShutdownHTTPHandlerV0BackendStillRunningDevuelveConflict(t *testing.T) {
	executor := &fakeMCPServerShutdownHTTPExecutorV0{
		result: MCPServerShutdownToolResultV0{
			Estado:          MCPServerShutdownEstadoOKV0,
			CorrelationID:   "corr-backend-running-http",
			Status:          "backend_still_running",
			ShutdownReady:   false,
			ActiveWorkCount: 1,
			ActiveWorks: []MCPServerShutdownWorkV0{{
				Kind:    "goal_backend",
				RunRef:  "run-backend-running-http",
				WorkRef: "goal-backend-running-http",
				Status:  "backend_still_running",
			}},
		},
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPServerShutdownToolInputV0{
		RequestID:   "req-backend-running-http",
		RequestedBy: "orquesta-director",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPServerShutdownHTTPPathV0, body)

	NewMCPServerShutdownHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict ||
		rec.Header().Get("X-Correlation-ID") != "corr-backend-running-http" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPServerShutdownToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != "backend_still_running" ||
		result.ActiveWorkCount != 1 ||
		len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].Kind != "goal_backend" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPServerShutdownHTTPHandlerV0NoPropagaErrorNoCatalogado(t *testing.T) {
	executor := &fakeMCPServerShutdownHTTPExecutorV0{
		err: errors.New("payload_invalido: payload en /home/alberto/Trabajo/orquesta/secreto bearer sk-123456789"),
	}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPServerShutdownToolInputV0{
		RequestID:     "req-shutdown-http-error-001",
		CorrelationID: "corr-shutdown-http-error-001",
		Forced:        true,
		RequestedBy:   "orquesta-director",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPServerShutdownHTTPPathV0, body)

	NewMCPServerShutdownHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError ||
		rec.Header().Get("X-Correlation-ID") != "corr-shutdown-http-error-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPServerShutdownToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPServerShutdownEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "server_shutdown_http_error" ||
		result.Errores[0].Message != "server_shutdown_executor_error" ||
		strings.Contains(result.Errores[0].Message, "/home/alberto") ||
		strings.Contains(result.Errores[0].Message, "sk-123456789") {
		t.Fatalf("result=%+v", result)
	}
}

type fakeMCPServerShutdownHTTPExecutorV0 struct {
	input  MCPServerShutdownToolInputV0
	result MCPServerShutdownToolResultV0
	err    error
}

func (executor *fakeMCPServerShutdownHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPServerShutdownToolInputV0,
) (MCPServerShutdownToolResultV0, error) {
	executor.input = input
	return executor.result, executor.err
}
