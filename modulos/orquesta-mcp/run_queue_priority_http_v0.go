package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPRunQueuePriorityHTTPPathV0 = "/api/v0/runs/queue/priority"

func NewMCPRunQueuePriorityHTTPHandlerV0(
	executor MCPTransportRunQueuePriorityExecutorV0,
) http.Handler {
	return mcpRunQueuePriorityHTTPHandlerV0{executor: executor}
}

type mcpRunQueuePriorityHTTPHandlerV0 struct {
	executor MCPTransportRunQueuePriorityExecutorV0
}

func (handler mcpRunQueuePriorityHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPRunQueuePriorityHTTPPathV0 {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusNotFound, newMCPRunQueuePriorityHTTPErrorV0(r, MCPRunQueuePriorityToolInputV0{}, "path", "ruta_no_soportada"))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusMethodNotAllowed, newMCPRunQueuePriorityHTTPErrorV0(r, MCPRunQueuePriorityToolInputV0{}, "method", "metodo_no_permitido"))
		return
	}
	if handler.executor == nil {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusServiceUnavailable, newMCPRunQueuePriorityHTTPErrorV0(r, MCPRunQueuePriorityToolInputV0{}, "executor", "run_queue_no_configurado"))
		return
	}
	var input MCPRunQueuePriorityToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusBadRequest, newMCPRunQueuePriorityHTTPErrorV0(r, input, "body", "request_body_invalido"))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusInternalServerError, newMCPRunQueuePriorityHTTPErrorV0(r, input, "executor", "run_queue_error"))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRunQueuePriorityEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPRunQueuePriorityHTTPV0(w, status, result)
}

func newMCPRunQueuePriorityHTTPErrorV0(
	r *http.Request,
	input MCPRunQueuePriorityToolInputV0,
	field string,
	message string,
) MCPRunQueuePriorityToolResultV0 {
	result := newMCPRunQueuePriorityErrorV0(input, "run_queue_http_error", field, message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func writeMCPRunQueuePriorityHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPRunQueuePriorityToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
