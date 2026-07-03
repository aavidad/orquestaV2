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

func TestMCPRunSupervisorHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado:        MCPRunSupervisorEstadoOKV0,
			CorrelationID: "corr-run-supervisor-http-001",
			RunRef:        "run-ref-supervisor-http-001",
			StopReason:    "max_ticks",
			Ticks:         2,
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-http-001",
		CorrelationID: "corr-run-supervisor-http-001",
		RunRef:        "run-ref-supervisor-http-001",
		MaxTicks:      2,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RunRef != "run-ref-supervisor-http-001" || executor.input.MaxTicks != 2 {
		t.Fatalf("input=%+v", executor.input)
	}
	if rec.Header().Get("X-Correlation-ID") != "corr-run-supervisor-http-001" {
		t.Fatalf("correlation header=%s", rec.Header().Get("X-Correlation-ID"))
	}
}

func TestMCPRunSupervisorHTTPHandlerV0NoCancelaSupervisorPorCierreHTTP(t *testing.T) {
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-supervisor-http-cancel-001",
		},
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-supervisor-http-cancel-001","max_ticks":1}`)
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.ctxErr != nil {
		t.Fatalf("executor recibio contexto cancelado: %v", executor.ctxErr)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0DevuelveAcceptedSiExecutorSigueVivo(t *testing.T) {
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		delay: 25 * time.Millisecond,
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-supervisor-http-background-001",
		},
	}
	body := bytes.NewBufferString(`{
		"request_id":"request-ref-supervisor-http-background-001",
		"correlation_id":"corr-supervisor-http-background-001",
		"idempotency_key":"idem-supervisor-http-background-001",
		"run_ref":"run-ref-supervisor-http-background-001",
		"max_ticks":20
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	newMCPRunSupervisorHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

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
		!strings.Contains(result.OperationRef, "idem-supervisor-http-background-001") ||
		len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != "run_supervisor_background_accepted" ||
		!hasMCPRunSupervisorNextActionForTestV0(result.NextActions, "poll_director_stats_or_run_queue") ||
		!hasMCPRunSupervisorNextActionForTestV0(result.NextActions, "poll_queue_global_status") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0ClienteRealRecibeCuerpoSinColgar(t *testing.T) {
	executor := &blockingMCPRunSupervisorHTTPExecutorV0{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	handler := newMCPRunSupervisorHTTPHandlerWithTimeoutV0(executor, time.Millisecond)
	server := httptest.NewServer(handler)
	defer server.Close()

	body := bytes.NewBufferString(`{
		"request_id":"request-ref-supervisor-http-real-client-001",
		"correlation_id":"corr-supervisor-http-real-client-001",
		"idempotency_key":"idem-supervisor-http-real-client-001",
		"run_ref":"run-ref-supervisor-http-real-client-001",
		"max_ticks":20
	}`)
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+MCPRunSupervisorHTTPPathV0,
		body,
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-supervisor-http-real-client-001")

	client := &http.Client{Timeout: time.Second}
	started := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("client.Do elapsed=%s err=%v", elapsed, err)
	}
	defer resp.Body.Close()
	close(executor.release)

	if resp.StatusCode != http.StatusAccepted || elapsed > time.Second {
		t.Fatalf("status=%d elapsed=%s", resp.StatusCode, elapsed)
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoOKV0 ||
		result.StopReason != "accepted_background" ||
		result.OperationRef == "" ||
		result.Last.Status != "accepted_background" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "run_supervisor_background_accepted") ||
		!hasMCPRunSupervisorNextActionForTestV0(result.NextActions, "poll_queue_global_status") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0BodyVacioNoSeCuelga(t *testing.T) {
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		delay: 25 * time.Millisecond,
		result: MCPRunSupervisorToolResultV0{
			Estado: MCPRunSupervisorEstadoOKV0,
		},
	}
	req := httptest.NewRequest(
		http.MethodPost,
		MCPRunSupervisorHTTPPathV0,
		bytes.NewBufferString(`{}`),
	)
	rec := httptest.NewRecorder()

	newMCPRunSupervisorHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.StopReason != "accepted_background" ||
		result.OperationRef != "operation-ref-run-supervisor-queue" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "run_supervisor_background_accepted") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0NoDuplicaOperacionActiva(t *testing.T) {
	executor := &blockingMCPRunSupervisorHTTPExecutorV0{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	handler := newMCPRunSupervisorHTTPHandlerWithTimeoutV0(executor, time.Millisecond)
	body := `{
		"request_id":"request-ref-supervisor-http-dedupe-001",
		"run_ref":"run-ref-supervisor-http-dedupe-001",
		"idempotency_key":"idem-supervisor-http-dedupe-001"
	}`
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(
		http.MethodPost,
		MCPRunSupervisorHTTPPathV0,
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
		MCPRunSupervisorHTTPPathV0,
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
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "run_supervisor_operation_already_running") {
		t.Fatalf("calls=%d result=%+v", executor.callsV0(), result)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0PropagaErrorPublicoDelExecutor(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-http-error-001",
		CorrelationID: "corr-run-supervisor-http-error-001",
		RunRef:        "run-ref-supervisor-http-error-001",
	}
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		result: NewMCPRunSupervisorErrorResultV0(
			input,
			"run_supervisor_execute_error",
			"executor",
			"delivery_ack_ingestion_failed",
		),
		err: errors.New("internal stack error must not hide public payload"),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_supervisor_execute_error" ||
		result.Errores[0].Message != "delivery_ack_ingestion_failed" {
		t.Fatalf("error publico perdido: %+v", result)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0RespetaOKConErrorPostEntrega(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID: "request-ref-run-supervisor-http-post-delivery-001",
		RunRef:    "run-ref-supervisor-http-post-delivery-001",
	}
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		result: MCPRunSupervisorToolResultV0{
			Estado:     MCPRunSupervisorEstadoOKV0,
			RunRef:     input.RunRef,
			StopReason: "delivered_with_post_delivery_supervisor_error",
			Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
				Code: "delivered_with_post_delivery_supervisor_error",
			}},
		},
		err: errors.New("post delivery supervisor error already classified"),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoOKV0 ||
		result.StopReason != "delivered_with_post_delivery_supervisor_error" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "delivered_with_post_delivery_supervisor_error") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunSupervisorHTTPHandlerV0NoPropagaErrorNoCatalogado(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-http-cause-001",
		CorrelationID: "corr-run-supervisor-http-cause-001",
		RunRef:        "run-ref-supervisor-http-cause-001",
	}
	executor := &fakeMCPRunSupervisorHTTPExecutorV0{
		err: errors.New("codex_worktree_verification: ack_files_mismatch en /root/Trabajo/orquesta con token=secret123456"),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_supervisor_execute_error" ||
		result.Errores[0].Message != "run_supervisor_execute_error" {
		t.Fatalf("error publico inesperado: %+v", result)
	}
	if result.CorrelationID != input.CorrelationID ||
		result.OperationRef == "" ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-supervisor-execute-error") ||
		!containsStringMCPTestV0(result.EvidenceRefs, result.OperationRef) ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "run_supervisor_execute_error") ||
		!containsStringMCPTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-run-supervisor-execute-error") {
		t.Fatalf("error debe conservar correlacion/evidencia/diagnostico: %+v", result)
	}
	if strings.Contains(rec.Body.String(), "/root/Trabajo") ||
		strings.Contains(rec.Body.String(), "secret123456") {
		t.Fatalf("error filtra datos operativos: %s", rec.Body.String())
	}
}

func TestMCPRunSupervisorHTTPHandlerV0ExecutorNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, MCPRunSupervisorHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPRunSupervisorHTTPHandlerV0SoloPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPRunSupervisorHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPRunSupervisorHTTPHandlerV0(&fakeMCPRunSupervisorHTTPExecutorV0{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != mcpPublicHTTPAllowHeaderV0(http.MethodPost) {
		t.Fatalf("status=%d allow=%s", rec.Code, rec.Header().Get("Allow"))
	}
}

type blockingMCPRunSupervisorHTTPExecutorV0 struct {
	mu      sync.Mutex
	once    sync.Once
	calls   int
	started chan struct{}
	release chan struct{}
}

func hasMCPRunSupervisorNextActionForTestV0(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func (executor *blockingMCPRunSupervisorHTTPExecutorV0) Execute(
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

func (executor *blockingMCPRunSupervisorHTTPExecutorV0) callsV0() int {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.calls
}

type fakeMCPRunSupervisorHTTPExecutorV0 struct {
	input  MCPRunSupervisorToolInputV0
	ctxErr error
	result MCPRunSupervisorToolResultV0
	err    error
	delay  time.Duration
}

func (executor *fakeMCPRunSupervisorHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	executor.input = input
	executor.ctxErr = ctx.Err()
	if executor.delay > 0 {
		time.Sleep(executor.delay)
	}
	if executor.result.Estado == "" {
		executor.result = MCPRunSupervisorToolResultV0{Estado: MCPRunSupervisorEstadoOKV0}
	}
	return executor.result, executor.err
}
