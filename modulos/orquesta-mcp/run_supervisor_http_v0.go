package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPRunSupervisorHTTPPathV0 = "/api/v0/runs/supervise"

func NewMCPRunSupervisorHTTPHandlerV0(
	executor MCPTransportRunSupervisorExecutorV0,
) http.Handler {
	return mcpRunSupervisorHTTPHandlerV0{executor: executor}
}

type mcpRunSupervisorHTTPHandlerV0 struct {
	executor MCPTransportRunSupervisorExecutorV0
}

func (handler mcpRunSupervisorHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPRunSupervisorHTTPPathV0 {
		writeMCPRunSupervisorHTTPV0(w, http.StatusNotFound, newMCPRunSupervisorHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "path", "ruta_no_soportada"))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPRunSupervisorHTTPV0(w, http.StatusMethodNotAllowed, newMCPRunSupervisorHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "method", "metodo_no_permitido"))
		return
	}
	if handler.executor == nil {
		writeMCPRunSupervisorHTTPV0(w, http.StatusServiceUnavailable, newMCPRunSupervisorHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "executor", "run_supervisor_no_configurado"))
		return
	}
	var input MCPRunSupervisorToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPRunSupervisorHTTPV0(w, http.StatusBadRequest, newMCPRunSupervisorHTTPErrorV0(r, input, "body", "request_body_invalido"))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		if result.Estado == MCPRunSupervisorEstadoErrorV0 && len(result.Errores) > 0 {
			writeMCPRunSupervisorHTTPV0(w, http.StatusInternalServerError, result)
			return
		}
		writeMCPRunSupervisorHTTPV0(w, http.StatusInternalServerError, newMCPRunSupervisorHTTPErrorV0(
			r,
			input,
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("run_supervisor_error", err),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRunSupervisorEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPRunSupervisorHTTPV0(w, status, result)
}

func newMCPRunSupervisorHTTPErrorV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
	field string,
	message string,
) MCPRunSupervisorToolResultV0 {
	result := NewMCPRunSupervisorErrorResultV0(input, "run_supervisor_http_error", field, message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func writeMCPRunSupervisorHTTPV0(
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
