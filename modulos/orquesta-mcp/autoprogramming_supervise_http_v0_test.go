package orquestamcp

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado:        MCPRunSupervisorEstadoOKV0,
			CorrelationID: "corr-autop-supervise-http-001",
			RunRef:        "run-ref-autop-supervise-http-001",
			Ticks:         1,
		},
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-autop-supervise-http-001","max_ticks":1}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RunRef != "run-ref-autop-supervise-http-001" ||
		executor.input.MaxTicks != 1 ||
		rec.Header().Get("X-Correlation-ID") != "corr-autop-supervise-http-001" {
		t.Fatalf("input=%+v headers=%v", executor.input, rec.Header())
	}
}

func TestMCPAutoprogrammingSuperviseTransportV0QuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingSuperviseToolNameV0, MCPRunSupervisorToolInputV0{})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 300)
}

type fakeMCPAutoprogrammingSuperviseHTTPExecutorV0 struct {
	input  MCPRunSupervisorToolInputV0
	result MCPRunSupervisorToolResultV0
}

func (executor *fakeMCPAutoprogrammingSuperviseHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	executor.input = input
	return executor.result, nil
}
