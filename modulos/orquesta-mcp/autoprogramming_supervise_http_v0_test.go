package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMCPAutoprogrammingSuperviseDescriptorV0DeclaraEvidenciaEnErrores(t *testing.T) {
	descriptor := MCPAutoprogrammingSuperviseDescriptorV0()
	if !strings.Contains(descriptor.Output, "error:{errores_publicos,evidence_refs?") ||
		!strings.Contains(descriptor.Output, "operation_ref?") ||
		!strings.Contains(descriptor.Output, "repair_run_refs?") ||
		!strings.Contains(descriptor.Output, "idempotency_key?") ||
		!strings.Contains(descriptor.Output, "next_actions?") ||
		!strings.Contains(descriptor.Output, "diagnostics?") {
		t.Fatalf("descriptor autoprogramming.supervise debe declarar evidencia en errores: %+v", descriptor)
	}
}

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

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0AceptaAliasesOperativos(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-autop-supervise-http-aliases-001",
		},
	}
	body := bytes.NewBufferString(`{
		"run_ref":"run-ref-autop-supervise-http-aliases-001",
		"queue_ref":"global",
		"max_ticks":3,
		"max_dispatches":6,
		"max_outbox":12,
		"include_process_refs":true
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RunRef != "run-ref-autop-supervise-http-aliases-001" ||
		executor.input.QueueRef != "global" ||
		executor.input.MaxTicks != 3 ||
		executor.input.MaxDispatchesPerWait != 6 ||
		executor.input.MaxRunsPerTick != 0 ||
		executor.input.MaxExecutions != 0 ||
		executor.input.MaxOutboxPerCycle != 12 {
		t.Fatalf("input=%+v", executor.input)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0AliasDispatchesAmpliaColaV0(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
		},
	}
	body := bytes.NewBufferString(`{
		"queue_ref":"global",
		"max_dispatches":6,
		"max_outbox":12
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.QueueRef != "global" ||
		executor.input.MaxDispatchesPerWait != 6 ||
		executor.input.MaxRunsPerTick != 6 ||
		executor.input.MaxExecutions != 6 ||
		executor.input.MaxOutboxPerCycle != 12 {
		t.Fatalf("input=%+v", executor.input)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0NoCancelaSupervisorPorCierreHTTP(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-autop-supervise-http-cancel-001",
		},
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-autop-supervise-http-cancel-001","max_ticks":1}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingSuperviseHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.ctxErr != nil {
		t.Fatalf("executor recibio contexto cancelado: %v", executor.ctxErr)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0DevuelveAcceptedSiExecutorSigueVivo(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		delay: 25 * time.Millisecond,
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-autop-supervise-http-background-001",
		},
	}
	body := bytes.NewBufferString(`{
		"request_id":"request-ref-autop-supervise-http-background-001",
		"run_ref":"run-ref-autop-supervise-http-background-001",
		"idempotency_key":"idem-autop-supervise-http-background-001"
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSuperviseHTTPPathV0, body)
	rec := httptest.NewRecorder()

	newMCPAutoprogrammingSuperviseHTTPHandlerWithTimeoutV0(
		executor,
		time.Millisecond,
	).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !rec.Flushed {
		t.Fatalf("accepted_background no hizo flush de la respuesta HTTP")
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoOKV0 ||
		result.StopReason != "accepted_background" ||
		result.OperationRef == "" ||
		!strings.Contains(result.OperationRef, "idem-autop-supervise-http-background-001") ||
		!hasMCPRunSupervisorNextActionForTestV0(result.NextActions, "poll_autoprogramming_status") ||
		!hasMCPRunSupervisorNextActionForTestV0(result.NextActions, "poll_queue_global_status") ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_supervise_background_accepted") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0BodyVacioNoSeCuelga(t *testing.T) {
	executor := &fakeMCPAutoprogrammingSuperviseHTTPExecutorV0{
		delay: 25 * time.Millisecond,
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
		},
	}
	req := httptest.NewRequest(
		http.MethodPost,
		MCPAutoprogrammingSuperviseHTTPPathV0,
		bytes.NewBufferString(`{}`),
	)
	rec := httptest.NewRecorder()

	newMCPAutoprogrammingSuperviseHTTPHandlerWithTimeoutV0(
		executor,
		time.Millisecond,
	).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.StopReason != "accepted_background" ||
		result.OperationRef != "operation-ref-autoprogramming-supervise-queue" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_supervise_background_accepted") ||
		!hasMCPRunSupervisorNextActionForTestV0(result.NextActions, "poll_queue_global_status") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSuperviseHTTPHandlerV0NoDuplicaOperacionActiva(t *testing.T) {
	executor := &blockingMCPAutoprogrammingSuperviseHTTPExecutorV0{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	handler := newMCPAutoprogrammingSuperviseHTTPHandlerWithTimeoutV0(executor, time.Millisecond)
	body := `{
		"request_id":"request-ref-autop-supervise-http-dedupe-001",
		"run_ref":"run-ref-autop-supervise-http-dedupe-001",
		"idempotency_key":"idem-autop-supervise-http-dedupe-001"
	}`
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(
		http.MethodPost,
		MCPAutoprogrammingSuperviseHTTPPathV0,
		bytes.NewBufferString(body),
	))
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	select {
	case <-executor.started:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("executor no arranco")
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(
		http.MethodPost,
		MCPAutoprogrammingSuperviseHTTPPathV0,
		bytes.NewBufferString(body),
	))
	close(executor.release)

	if second.Code != http.StatusAccepted {
		t.Fatalf("second status=%d body=%s", second.Code, second.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(second.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if executor.callsV0() != 1 ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_supervise_operation_already_running") {
		t.Fatalf("calls=%d result=%+v", executor.callsV0(), result)
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
	if result.OperationRef == "" ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-supervise-execute-error") ||
		!containsStringMCPTestV0(result.EvidenceRefs, result.OperationRef) ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_supervise_executor_error") ||
		!containsStringMCPTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-autoprogramming-supervise-execute-error") {
		t.Fatalf("error debe conservar evidencia/diagnostico: %+v", result)
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
	if result.OperationRef == "" ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-supervise-execute-error") ||
		!containsStringMCPTestV0(result.EvidenceRefs, result.OperationRef) ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_supervise_executor_error") ||
		!containsStringMCPTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-autoprogramming-supervise-execute-error") {
		t.Fatalf("error debe conservar evidencia/diagnostico: %+v", result)
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

type blockingMCPAutoprogrammingSuperviseHTTPExecutorV0 struct {
	mu      sync.Mutex
	once    sync.Once
	calls   int
	started chan struct{}
	release chan struct{}
}

func (executor *blockingMCPAutoprogrammingSuperviseHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	executor.mu.Lock()
	executor.calls++
	executor.once.Do(func() { close(executor.started) })
	executor.mu.Unlock()
	<-executor.release
	return MCPRunSupervisorToolResultV0{
		Estado: MCPRunSupervisorEstadoOKV0,
		RunRef: input.RunRef,
	}, nil
}

func (executor *blockingMCPAutoprogrammingSuperviseHTTPExecutorV0) callsV0() int {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.calls
}

type fakeMCPAutoprogrammingSuperviseHTTPExecutorV0 struct {
	input  MCPRunSupervisorToolInputV0
	ctxErr error
	result MCPRunSupervisorToolResultV0
	err    error
	delay  time.Duration
}

func (executor *fakeMCPAutoprogrammingSuperviseHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	executor.input = input
	executor.ctxErr = ctx.Err()
	if executor.delay > 0 {
		time.Sleep(executor.delay)
	}
	if executor.err != nil {
		return executor.result, executor.err
	}
	return executor.result, nil
}
