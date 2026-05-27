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

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0ConservaOperatorAdviceSinPuerto(t *testing.T) {
	body := bytes.NewBufferString(`{
		"request_id":"request-ref-supervise-advice-001",
		"operator_advice":[{
			"subject_ref":"run-ref-supervise-advice-001",
			"advice":"consultar al operador"
		}]
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result mcpAutoprogrammingSuperviseHTTPResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.OperatorAdvice) != 1 ||
		result.OperatorAdvice[0].TargetRef != "run-ref-supervise-advice-001" ||
		!result.OperatorAdvice[0].NonBlocking ||
		len(result.Diagnostics) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0AceptaOperatorAdviceTextoV0(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-supervise-advice-text-http-001",
		},
	}
	body := bytes.NewBufferString(`{
		"run_ref":"run-ref-supervise-advice-text-http-001",
		"max_ticks":1,
		"operator_advice":"HTTP fallback cron diagnose; no destructive actions."
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result mcpAutoprogrammingSuperviseHTTPResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.OperatorAdvice) != 1 ||
		result.OperatorAdvice[0].Message != "HTTP fallback cron diagnose; no destructive actions." ||
		result.OperatorAdvice[0].TargetRef != "run-ref-supervise-advice-text-http-001" ||
		!result.OperatorAdvice[0].NonBlocking ||
		len(result.Diagnostics) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSuperviseTransportV0AceptaOperatorAdviceTextoV0(t *testing.T) {
	supervisor := &fakeAutoprogrammingAdviceSupervisorV0{}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{RunSupervisor: supervisor}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingSuperviseToolNameV0, map[string]any{
		"run_ref":         "run-ref-supervise-advice-text-transport-001",
		"max_ticks":       1,
		"operator_advice": "Cron operador: avanzar cola con 1-2 ticks no residentes.",
	})
	if err != nil {
		t.Fatalf("call tool no debe romper JSON-RPC: %v", err)
	}
	var result mcpAutoprogrammingSuperviseTransportResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.OperatorAdvice) != 1 ||
		result.OperatorAdvice[0].Message != "Cron operador: avanzar cola con 1-2 ticks no residentes." ||
		result.OperatorAdvice[0].TargetRef != "run-ref-supervise-advice-text-transport-001" ||
		!result.OperatorAdvice[0].NonBlocking ||
		len(result.Diagnostics) == 0 {
		t.Fatalf("operator_advice=%+v diagnostics=%+v", result.OperatorAdvice, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0DevuelvePayloadPublicoSiExecutorFalla(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		err: errors.New("payload.task requerido"),
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-autop-supervise-http-error-001","max_ticks":1}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 || len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_supervise_executor_error" ||
		result.Errores[0].Message != "autoprogramming_supervise_executor_error" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0PreservaPayloadPublicoDelExecutorConDiagnostics(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-autop-supervise-http-public-error-001",
		CorrelationID: "corr-autop-supervise-http-public-error-001",
		RunRef:        "run-ref-autop-supervise-http-public-error-001",
	}
	result := NewMCPRunSupervisorErrorResultV0(
		input,
		"delivery_ack_ingestion_failed",
		"payload.agent_ref",
		"delivery_ack_ingestion_failed field=payload.agent_ref next_action=ingest_late_ack_or_reconcile_stopped_agent",
	)
	result.Diagnostics = []MCPAutoprogrammingDiagnosticV0{{
		Code:         "drain_observation_apply_failed",
		Scope:        "run:run-ref-autop-supervise-http-public-error-001/agent:agent-ref-001",
		Message:      "field=payload.agent_ref cause=transicion_invalida",
		EvidenceRefs: []string{"delivery-ref-001", "agent-ref-001"},
	}}
	result.NextActions = []string{"ingest_late_ack_or_reconcile_stopped_agent"}
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		result: result,
		err:    errors.New("internal stack error must not hide public diagnostics"),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(got.Errores) != 1 ||
		got.Errores[0].Code != "delivery_ack_ingestion_failed" ||
		len(got.Diagnostics) != 1 ||
		got.Diagnostics[0].Code != "drain_observation_apply_failed" ||
		len(got.NextActions) != 1 ||
		got.NextActions[0] != "ingest_late_ack_or_reconcile_stopped_agent" {
		t.Fatalf("payload publico/diagnostics perdidos: %+v", got)
	}
}

func TestMCPAutoprogrammingSuperviseTransportV0DevuelvePayloadPublicoSiExecutorFalla(t *testing.T) {
	transport := newFakeMCPTransportV0()
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		err: errors.New("payload.task requerido"),
	}
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{RunSupervisor: executor}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingSuperviseToolNameV0, MCPRunSupervisorToolInputV0{
		RunRef:   "run-ref-autop-supervise-error-001",
		MaxTicks: 1,
	})
	if err != nil {
		t.Fatalf("call tool no debe romper JSON-RPC: %v", err)
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 || len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_supervise_executor_error" ||
		result.Errores[0].Message != "autoprogramming_supervise_executor_error" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSuperviseTransportV0PreservaPayloadPublicoDelExecutorConDiagnostics(t *testing.T) {
	transport := newFakeMCPTransportV0()
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-autop-supervise-transport-public-error-001",
		CorrelationID: "corr-autop-supervise-transport-public-error-001",
		RunRef:        "run-ref-autop-supervise-transport-public-error-001",
	}
	result := NewMCPRunSupervisorErrorResultV0(
		input,
		"delivery_ack_ingestion_failed",
		"payload.agent_ref",
		"delivery_ack_ingestion_failed field=payload.agent_ref next_action=ingest_late_ack_or_reconcile_stopped_agent",
	)
	result.Diagnostics = []MCPAutoprogrammingDiagnosticV0{{
		Code:         "drain_observation_apply_failed",
		Scope:        "run:run-ref-autop-supervise-transport-public-error-001/agent:agent-ref-001",
		Message:      "field=payload.agent_ref cause=transicion_invalida",
		EvidenceRefs: []string{"delivery-ref-001", "agent-ref-001"},
	}}
	result.NextActions = []string{"ingest_late_ack_or_reconcile_stopped_agent"}
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		result: result,
		err:    errors.New("internal stack error must not hide public diagnostics"),
	}
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{RunSupervisor: executor}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingSuperviseToolNameV0, input)
	if err != nil {
		t.Fatalf("call tool no debe romper JSON-RPC: %v", err)
	}
	var got MCPRunSupervisorToolResultV0
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(got.Errores) != 1 ||
		got.Errores[0].Code != "delivery_ack_ingestion_failed" ||
		len(got.Diagnostics) != 1 ||
		got.Diagnostics[0].Code != "drain_observation_apply_failed" ||
		len(got.NextActions) != 1 ||
		got.NextActions[0] != "ingest_late_ack_or_reconcile_stopped_agent" {
		t.Fatalf("payload publico/diagnostics perdidos: %+v", got)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0RechazaBodyConTrailingData(t *testing.T) {
	body := bytes.NewBufferString(`{"run_ref":"run-ref-trailing","max_ticks":1} {"extra":true}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(&fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "request_body_trailing_data") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeMCPAutoprogrammingSuperviseHTTPExecutorV0 struct {
	input  MCPRunSupervisorToolInputV0
	result MCPRunSupervisorToolResultV0
	err    error
}

func (executor *fakeMCPAutoprogrammingSuperviseHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	executor.input = input
	if executor.err != nil {
		return executor.result, executor.err
	}
	return executor.result, nil
}
