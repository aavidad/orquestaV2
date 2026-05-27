package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPAutoprogrammingSuperviseHTTPPathV0 = "/api/v0/autoprogramming/supervise"

func NewMCPAutoprogrammingSuperviseHTTPHandlerV0(
	executor MCPTransportRunSupervisorExecutorV0,
) http.Handler {
	return mcpAutoprogrammingSuperviseHTTPHandlerV0{executor: executor}
}

type mcpAutoprogrammingSuperviseHTTPHandlerV0 struct {
	executor MCPTransportRunSupervisorExecutorV0
}

type mcpAutoprogrammingSuperviseHTTPInputV0 struct {
	MCPRunSupervisorToolInputV0
	OperatorAdvice mcpAutoprogrammingOperatorAdviceListV0 `json:"operator_advice,omitempty"`
}

type mcpAutoprogrammingSuperviseHTTPResultV0 struct {
	MCPRunSupervisorToolResultV0
	OperatorAdvice []MCPAutoprogrammingOperatorAdviceV0 `json:"operator_advice,omitempty"`
	Diagnostics    []MCPAutoprogrammingDiagnosticV0     `json:"diagnostics,omitempty"`
}

func (handler mcpAutoprogrammingSuperviseHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingSuperviseHTTPPathV0 {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	var input mcpAutoprogrammingSuperviseHTTPInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, input.MCPRunSupervisorToolInputV0, "body", code))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingSuperviseHTTPResultV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, input.MCPRunSupervisorToolInputV0, "executor", "autoprogramming_supervise_no_configurado"), input.OperatorAdvice)
		return
	}
	result, err := handler.executor.Execute(r.Context(), input.MCPRunSupervisorToolInputV0)
	if err != nil {
		if result.Estado == MCPRunSupervisorEstadoErrorV0 && len(result.Errores) > 0 {
			writeMCPAutoprogrammingSuperviseHTTPResultV0(w, http.StatusInternalServerError, result, input.OperatorAdvice)
			return
		}
		writeMCPAutoprogrammingSuperviseHTTPResultV0(w, http.StatusInternalServerError, newMCPAutoprogrammingSuperviseExecutorErrorV0(r, input.MCPRunSupervisorToolInputV0, err), input.OperatorAdvice)
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRunSupervisorEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingSuperviseHTTPResultV0(w, status, result, input.OperatorAdvice)
}

func newMCPAutoprogrammingSuperviseHTTPErrorV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
	field string,
	message string,
) MCPRunSupervisorToolResultV0 {
	result := NewMCPRunSupervisorErrorResultV0(input, "autoprogramming_supervise_http_error", field, strings.TrimSpace(message))
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	return result
}

func newMCPAutoprogrammingSuperviseExecutorErrorV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
	err error,
) MCPRunSupervisorToolResultV0 {
	message := publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_supervise_executor_error", err)
	result := NewMCPRunSupervisorErrorResultV0(input, "autoprogramming_supervise_executor_error", "executor", message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	return result
}

func writeMCPAutoprogrammingSuperviseHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPRunSupervisorToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func writeMCPAutoprogrammingSuperviseHTTPResultV0(
	w http.ResponseWriter,
	status int,
	result MCPRunSupervisorToolResultV0,
	advice mcpAutoprogrammingOperatorAdviceListV0,
) {
	normalizedAdvice := advice.normalizedMCPV0(
		firstNonEmptyMCPV0(result.RunRef, result.RequestID, result.CorrelationID),
	)
	if len(normalizedAdvice) == 0 {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, status, result)
		return
	}
	diagnostics := append([]MCPAutoprogrammingDiagnosticV0(nil), result.Diagnostics...)
	diagnostics = append(diagnostics, mcpAutoprogrammingDiagnosticV0(
		"operator_advice_recorded_non_blocking",
		"operator_advice",
		"consejo de operador registrado sin bloquear supervisor",
	))
	payloadResult := result
	payloadResult.Diagnostics = nil
	payload := mcpAutoprogrammingSuperviseHTTPResultV0{
		MCPRunSupervisorToolResultV0: payloadResult,
		OperatorAdvice:               normalizedAdvice,
		Diagnostics:                  diagnostics,
	}
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
