package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const MCPAutoprogrammingStatusHTTPPathV0 = "/api/v0/autoprogramming/status"
const defaultMCPAutoprogrammingStatusHTTPResponseTimeoutV0 = 2 * time.Second

func NewMCPAutoprogrammingStatusHTTPHandlerV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
) http.Handler {
	return newMCPAutoprogrammingStatusHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPAutoprogrammingStatusHTTPResponseTimeoutV0,
	)
}

func newMCPAutoprogrammingStatusHTTPHandlerWithTimeoutV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpAutoprogrammingStatusHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpAutoprogrammingStatusHTTPHandlerV0 struct {
	executor        MCPTransportAutoprogrammingStatusExecutorV0
	responseTimeout time.Duration
}

type mcpAutoprogrammingStatusHTTPInputV0 struct {
	MCPAutoprogrammingStatusToolInputV0
	TelemetryFlags string                                 `json:"telemetry_flags,omitempty"`
	OperatorAdvice mcpAutoprogrammingOperatorAdviceListV0 `json:"operator_advice,omitempty"`
}

type mcpAutoprogrammingStatusHTTPResultV0 struct {
	MCPAutoprogrammingStatusToolResultV0
	OperatorAdvice []MCPAutoprogrammingOperatorAdviceV0 `json:"operator_advice,omitempty"`
}

func (handler mcpAutoprogrammingStatusHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingStatusHTTPPathV0 {
		writeMCPAutoprogrammingStatusHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingStatusHTTPErrorV0(r, MCPAutoprogrammingStatusToolInputV0{}, "path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPAutoprogrammingStatusHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingStatusHTTPErrorV0(r, MCPAutoprogrammingStatusToolInputV0{}, "method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	var input mcpAutoprogrammingStatusHTTPInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		writeMCPAutoprogrammingStatusHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingStatusHTTPErrorV0(r, input.MCPAutoprogrammingStatusToolInputV0, "body", code))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingStatusHTTPResultV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingStatusHTTPErrorV0(r, input.MCPAutoprogrammingStatusToolInputV0, "executor", "autoprogramming_status_no_configurado"), input.OperatorAdvice)
		return
	}
	result, err, timedOut := handler.executeStatusWithResponseTimeoutV0(r, input.MCPAutoprogrammingStatusToolInputV0)
	if timedOut {
		writeMCPAutoprogrammingStatusHTTPResultV0(w, http.StatusGatewayTimeout, result, input.OperatorAdvice)
		return
	}
	if err != nil {
		result := newMCPAutoprogrammingStatusExecutorErrorResultV0(
			input.MCPAutoprogrammingStatusToolInputV0,
			publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_status_executor_error", err),
		)
		result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID)
		writeMCPAutoprogrammingStatusHTTPResultV0(w, http.StatusInternalServerError, result, input.OperatorAdvice)
		return
	}
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingStatusEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingStatusHTTPResultV0(w, status, result, input.OperatorAdvice)
}

type mcpAutoprogrammingStatusHTTPExecutionV0 struct {
	result MCPAutoprogrammingStatusToolResultV0
	err    error
}

func (handler mcpAutoprogrammingStatusHTTPHandlerV0) executeStatusWithResponseTimeoutV0(
	r *http.Request,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPAutoprogrammingStatusToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpAutoprogrammingStatusHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpAutoprogrammingStatusHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPAutoprogrammingStatusTimeoutResultV0(r, input), nil, true
	}
}

func newMCPAutoprogrammingStatusExecutorErrorResultV0(
	input MCPAutoprogrammingStatusToolInputV0,
	message string,
) MCPAutoprogrammingStatusToolResultV0 {
	result := newMCPAutoprogrammingStatusBaseV0(input)
	result.Estado = MCPAutoprogrammingStatusEstadoErrorV0
	result.Errores = []MCPValidationIssueV0{{
		Code:    "autoprogramming_status_executor_error",
		Field:   "executor",
		Message: strings.TrimSpace(firstNonEmptyMCPV0(message, "autoprogramming_status_executor_error")),
	}}
	result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
		"autoprogramming_status_executor_error",
		"executor",
		result.Errores[0].Message,
	))
	return result
}

func newMCPAutoprogrammingStatusTimeoutResultV0(
	r *http.Request,
	input MCPAutoprogrammingStatusToolInputV0,
) MCPAutoprogrammingStatusToolResultV0 {
	result := newMCPAutoprogrammingStatusBaseV0(input)
	result.Estado = MCPAutoprogrammingStatusEstadoErrorV0
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID)
	result.Errores = []MCPValidationIssueV0{{
		Code:    "autoprogramming_status_timeout",
		Field:   "executor",
		Message: "consulta de estado excedio la ventana HTTP acotada",
	}}
	result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
		"autoprogramming_status_timeout",
		"executor",
		"consulta de estado cancelada por timeout HTTP; no relanzar goal ni usar loop legacy sin diagnostico acotado",
	))
	result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
		"autoprogramming_status_timeout_action",
		"operator",
		"acciones: poll_autoprogramming_status_with_run_ref, observe_active_goals_once_with_operation_ref, inspect_goal_backend_snapshot",
	))
	return result
}

func newMCPAutoprogrammingStatusHTTPErrorV0(
	r *http.Request,
	input MCPAutoprogrammingStatusToolInputV0,
	field string,
	message string,
) MCPAutoprogrammingStatusToolResultV0 {
	result := newMCPAutoprogrammingStatusBaseV0(input)
	result.Estado = MCPAutoprogrammingStatusEstadoErrorV0
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	result.Errores = []MCPValidationIssueV0{{
		Code:    "autoprogramming_status_http_error",
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	}}
	return result
}

func writeMCPAutoprogrammingStatusHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingStatusToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func writeMCPAutoprogrammingStatusHTTPResultV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingStatusToolResultV0,
	advice mcpAutoprogrammingOperatorAdviceListV0,
) {
	normalizedAdvice := advice.normalizedMCPV0(
		firstNonEmptyMCPV0(result.RunRef, result.QueueRef, result.RequestID, result.CorrelationID),
	)
	if len(normalizedAdvice) == 0 {
		writeMCPAutoprogrammingStatusHTTPV0(w, status, result)
		return
	}
	payload := mcpAutoprogrammingStatusHTTPResultV0{
		MCPAutoprogrammingStatusToolResultV0: result,
		OperatorAdvice:                       normalizedAdvice,
	}
	payload.Diagnostics = append(payload.Diagnostics, mcpAutoprogrammingDiagnosticV0(
		"operator_advice_recorded_non_blocking",
		"operator_advice",
		"consejo de operador registrado sin bloquear estado de autoprogramacion",
	))
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
