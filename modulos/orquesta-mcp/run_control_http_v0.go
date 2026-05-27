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
		writeMCPRunControlHTTPV0(w, http.StatusNotFound, newMCPRunControlHTTPErrorV0(r, MCPRunControlToolInputV0{}, "path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPRunControlHTTPV0(w, http.StatusMethodNotAllowed, newMCPRunControlHTTPErrorV0(r, MCPRunControlToolInputV0{}, "method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	if handler.executor == nil {
		writeMCPRunControlHTTPV0(w, http.StatusServiceUnavailable, newMCPRunControlHTTPErrorV0(r, MCPRunControlToolInputV0{}, "executor", "run_control_no_configurado"))
		return
	}
	var input MCPRunControlToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPRunControlHTTPV0(w, http.StatusBadRequest, newMCPRunControlHTTPErrorV0(r, input, "body", code))
		return
	}
	input, issues := normalizeMCPRunControlIdentityV0(
		input,
		r.Header.Get(MCPPublicCorrelationHeaderV0),
		MCPPublicMutationHeaderIdempotencyKeyV0(r.Header.Get),
	)
	if len(issues) > 0 {
		writeMCPRunControlHTTPV0(w, http.StatusBadRequest, newMCPRunControlErrorV0(input, issues[0].Code, issues[0].Field))
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
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get(MCPPublicCorrelationHeaderV0), input.CorrelationID, input.RequestID)
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
		w.Header().Set(MCPPublicCorrelationHeaderV0, result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
