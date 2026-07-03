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
	"time"
)

func TestMCPAutoprogrammingStatusHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:        MCPAutoprogrammingStatusEstadoOKV0,
			CorrelationID: "corr-autop-status-http-001",
			RunRef:        "run-ref-autop-status-http-001",
		},
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-autop-status-http-001"}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RunRef != "run-ref-autop-status-http-001" ||
		rec.Header().Get("X-Correlation-ID") != "corr-autop-status-http-001" {
		t.Fatalf("input=%+v headers=%v", executor.input, rec.Header())
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0ExecutorNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0TimeoutDevuelveJSONPublico(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{waitForCancel: true}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, bytes.NewBufferString(`{"queue_ref":"global"}`))
	req.Header.Set("X-Correlation-ID", "corr-autop-status-timeout-001")
	rec := httptest.NewRecorder()

	newMCPAutoprogrammingStatusHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout ||
		rec.Header().Get("X-Correlation-ID") != "corr-autop-status-timeout-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_status_timeout" ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_status_timeout") ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "autoprogramming_status_timeout_action") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-status-timeout") ||
		!containsStringMCPTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-autoprogramming-status-timeout") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0ConservaOperatorAdviceNoBloqueante(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado:    MCPAutoprogrammingStatusEstadoOKV0,
			RequestID: "request-ref-status-advice-001",
			RunRef:    "run-ref-status-advice-001",
		},
	}
	body := bytes.NewBufferString(`{
		"run_ref":"run-ref-status-advice-001",
		"operator_advice":[{
			"run":"run-ref-status-advice-alias-001",
			"kind":"pause",
			"text":"esperar confirmacion humana"
		}]
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result mcpAutoprogrammingStatusHTTPResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.OperatorAdvice) != 1 ||
		result.OperatorAdvice[0].TargetRef != "run-ref-status-advice-alias-001" ||
		result.OperatorAdvice[0].Action != "advise" ||
		!result.OperatorAdvice[0].NonBlocking ||
		len(result.Diagnostics) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0AceptaOperatorAdviceTexto(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado: MCPAutoprogrammingStatusEstadoOKV0,
			RunRef: "run-ref-status-advice-text-http-001",
		},
	}
	body := bytes.NewBufferString(`{
		"run_ref":"run-ref-status-advice-text-http-001",
		"operator_advice":"vigilar sin borrar datos validos"
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result mcpAutoprogrammingStatusHTTPResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.OperatorAdvice) != 1 ||
		result.OperatorAdvice[0].Message != "vigilar sin borrar datos validos" ||
		result.OperatorAdvice[0].TargetRef != "run-ref-status-advice-text-http-001" ||
		!result.OperatorAdvice[0].NonBlocking ||
		len(result.Diagnostics) == 0 {
		t.Fatalf("operator_advice=%+v diagnostics=%+v", result.OperatorAdvice, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0AceptaIncludesFlexibles(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado: MCPAutoprogrammingStatusEstadoOKV0,
			RunRef: "run-ref-status-flex-http-001",
		},
	}
	body := bytes.NewBufferString(`{
		"run_ref":"run-ref-status-flex-http-001",
		"include_process_refs":["process_refs"],
		"include_agent_progress":"yes",
		"include_agent_usage":1
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bool(executor.input.IncludeProcessRefs) ||
		!bool(executor.input.IncludeAgentProgress) ||
		!bool(executor.input.IncludeAgentUsage) {
		t.Fatalf("input no normalizado: %+v", executor.input)
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0SerializaEfficiencySummary(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		result: MCPAutoprogrammingStatusToolResultV0{
			Estado: MCPAutoprogrammingStatusEstadoOKV0,
			RunRef: "run-ref-status-efficiency-http-001",
			EfficiencySummary: &MCPAutoprogrammingEfficiencySummaryV0{
				SchemaVersion:               MCPAutoprogrammingEfficiencySummarySchemaVersionV0,
				State:                       "live",
				OperationalHealthPercentage: 100,
				CompletionPercentage:        40,
				AlivePercentage:             100,
				QueueCandidates:             2,
				AgentsInFlight:              1,
			},
		},
	}
	body := bytes.NewBufferString(`{"run_ref":"run-ref-status-efficiency-http-001"}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.EfficiencySummary == nil ||
		result.EfficiencySummary.State != "live" ||
		result.EfficiencySummary.OperationalHealthPercentage != 100 ||
		result.EfficiencySummary.QueueCandidates != 2 {
		t.Fatalf("efficiency_summary=%+v", result.EfficiencySummary)
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0NoMarcaRunningStaleSiHayAgentesVivos(t *testing.T) {
	queue := &fakeMCPAutoprogrammingQueueStatusV0{}
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	executor := MCPAutoprogrammingStatusToolExecutorV0{
		Queue: queue,
		Stats: stats,
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(stats.inputs) != 1 ||
		stats.inputs[0].RunRef != "run-ref-autop-status-001" ||
		!stats.inputs[0].IncludeProcessRefs ||
		!stats.inputs[0].IncludeAgentProgress {
		t.Fatalf("stats inputs=%+v", stats.inputs)
	}
	if result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 ||
		result.QueueHealth.AgentsLive != 1 ||
		result.QueueHealth.RunningStale != 0 ||
		result.QueueHealth.RunningStaleNoProcess != 0 ||
		result.QueueHealth.RunningWithoutRecentStats != 0 ||
		len(result.StaleRunning) != 0 {
		t.Fatalf("queue_health=%+v stale_running=%+v body=%s", result.QueueHealth, result.StaleRunning, rec.Body.String())
	}
	if queue.input.Action != MCPRunQueuePriorityActionRankV0 ||
		!queue.input.IncludeNonExecutable {
		t.Fatalf("queue input=%+v", queue.input)
	}
}

type fakeMCPAutoprogrammingStatusHTTPExecutorV0 struct {
	input         MCPAutoprogrammingStatusToolInputV0
	result        MCPAutoprogrammingStatusToolResultV0
	err           error
	waitForCancel bool
}

func (executor *fakeMCPAutoprogrammingStatusHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPAutoprogrammingStatusToolResultV0, error) {
	executor.input = input
	if executor.waitForCancel {
		<-ctx.Done()
	}
	return executor.result, executor.err
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0ErrorPublico(t *testing.T) {
	result := MCPAutoprogrammingStatusToolResultV0{
		Estado: MCPAutoprogrammingStatusEstadoErrorV0,
		Errores: []MCPValidationIssueV0{{
			Code:    "autoprogramming_status_no_disponible",
			Field:   "ports",
			Message: "estado de autoprogramacion no disponible",
		}},
	}
	encoded, _ := json.Marshal(result)
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{result: result}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, bytes.NewReader(encoded))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPAutoprogrammingStatusHTTPHandlerV0NoPropagaErrorNoCatalogado(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		err: errors.New("status store failed at /root/Trabajo/orquesta token=secret123456 run-ref-status-error-001"),
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingStatusHTTPPathV0, bytes.NewBufferString(`{"run_ref":"run-ref-status-error-001"}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingStatusHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_status_executor_error" ||
		result.Errores[0].Message != "autoprogramming_status_executor_error" ||
		len(result.Diagnostics) != 1 ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-status-executor-error") ||
		!containsStringMCPTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-autoprogramming-status-executor-error") {
		t.Fatalf("error publico incompleto: %+v", result)
	}
	if strings.Contains(rec.Body.String(), "/root/Trabajo") ||
		strings.Contains(rec.Body.String(), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", rec.Body.String())
	}
}

func TestMCPAutoprogrammingStatusTransportV0DevuelvePayloadPublicoSiExecutorFalla(t *testing.T) {
	executor := &fakeMCPAutoprogrammingStatusHTTPExecutorV0{
		err: errors.New("status store failed at /root/Trabajo/orquesta token=secret123456 run-ref-status-transport-error-001"),
	}
	raw, err := json.Marshal(MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-status-transport-error-001",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	output, err := mcpAutoprogrammingStatusTransportHandlerV0(executor)(context.Background(), raw)
	if err != nil {
		t.Fatalf("transport no debe romper JSON-RPC: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_status_executor_error" ||
		result.Errores[0].Message != "autoprogramming_status_executor_error" ||
		len(result.Diagnostics) != 1 ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-status-executor-error") ||
		!containsStringMCPTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-autoprogramming-status-executor-error") {
		t.Fatalf("payload publico incompleto: %+v", result)
	}
	if strings.Contains(string(output), "/root/Trabajo") ||
		strings.Contains(string(output), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", string(output))
	}
}
