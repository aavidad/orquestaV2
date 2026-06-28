package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMCPExternalWorkRunHTTPHandlerV0TimeoutDevuelveJSONPublico(t *testing.T) {
	executor := &blockingMCPExternalWorkRunHTTPExecutorV0{done: make(chan struct{})}
	body := bytes.NewBufferString(`{
		"request_id":"request-ref-external-work-run-timeout-001",
		"correlation_id":"corr-external-work-run-timeout-input-001",
		"director_execution_mode":"goal_first"
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPExternalWorkRunHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-external-work-run-timeout-header-001")

	newMCPExternalWorkRunHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout ||
		rec.Header().Get("X-Correlation-ID") != "corr-external-work-run-timeout-header-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPExternalWorkRunHTTPTimeoutCodeV0 {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-executor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor no recibio cancelacion tras timeout HTTP")
	}
}

type blockingMCPExternalWorkRunHTTPExecutorV0 struct {
	done chan struct{}
}

func (executor *blockingMCPExternalWorkRunHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPExternalWorkRunToolInputV0,
) (MCPExternalWorkRunToolResultV0, error) {
	<-ctx.Done()
	close(executor.done)
	return MCPExternalWorkRunToolResultV0{
		Estado:                MCPExternalWorkRunEstadoOKV0,
		DirectorExecutionMode: input.DirectorExecutionMode,
	}, nil
}
