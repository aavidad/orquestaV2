package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPAutoprogrammingStatusHTTPPathV0 = "/api/v0/autoprogramming/status"

func NewMCPAutoprogrammingStatusHTTPHandlerV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
) http.Handler {
	return mcpAutoprogrammingStatusHTTPHandlerV0{executor: executor}
}

type mcpAutoprogrammingStatusHTTPHandlerV0 struct {
	executor MCPTransportAutoprogrammingStatusExecutorV0
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
	result, err := handler.executor.Execute(r.Context(), input.MCPAutoprogrammingStatusToolInputV0)
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
