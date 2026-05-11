package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPRunControlHTTPPathV0 = "/api/v0/runs/control"

func NewMCPRunControlHTTPHandlerV0(
	executor MCPTransportRunControlExecutorV0,
) http.Handler {
	return mcpRunControlHTTPHandlerV0{executor: executor}
}

type mcpRunControlHTTPHandlerV0 struct {
	executor MCPTransportRunControlExecutorV0
}

func (handler mcpRunControlHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPRunControlHTTPPathV0 {
		writeMCPRunControlHTTPV0(w, http.StatusNotFound, newMCPRunControlHTTPErrorV0(r, MCPRunControlToolInputV0{}, "path", "ruta_no_soportada"))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPRunControlHTTPV0(w, http.StatusMethodNotAllowed, newMCPRunControlHTTPErrorV0(r, MCPRunControlToolInputV0{}, "method", "metodo_no_permitido"))
		return
	}
	if handler.executor == nil {
		writeMCPRunControlHTTPV0(w, http.StatusServiceUnavailable, newMCPRunControlHTTPErrorV0(r, MCPRunControlToolInputV0{}, "executor", "run_control_no_configurado"))
		return
	}
	var input MCPRunControlToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPRunControlHTTPV0(w, http.StatusBadRequest, newMCPRunControlHTTPErrorV0(r, input, "body", "request_body_invalido"))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPRunControlHTTPV0(w, http.StatusInternalServerError, newMCPRunControlHTTPErrorV0(r, input, "executor", "run_control_error"))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRunControlEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPRunControlHTTPV0(w, status, result)
}

func newMCPRunControlHTTPErrorV0(
	r *http.Request,
	input MCPRunControlToolInputV0,
	field string,
	message string,
) MCPRunControlToolResultV0 {
	result := newMCPRunControlErrorV0(input, "run_control_http_error", field)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func writeMCPRunControlHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPRunControlToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
