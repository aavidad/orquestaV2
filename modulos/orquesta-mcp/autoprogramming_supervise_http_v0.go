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

func (handler mcpAutoprogrammingSuperviseHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingSuperviseHTTPPathV0 {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "path", "ruta_no_soportada"))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "method", "metodo_no_permitido"))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "executor", "autoprogramming_supervise_no_configurado"))
		return
	}
	var input MCPRunSupervisorToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, input, "body", "request_body_invalido"))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusInternalServerError, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, input, "executor", "autoprogramming_supervise_error"))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRunSupervisorEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingSuperviseHTTPV0(w, status, result)
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
