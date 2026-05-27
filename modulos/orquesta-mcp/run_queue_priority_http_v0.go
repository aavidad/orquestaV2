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
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusNotFound, newMCPRunQueuePriorityHTTPErrorV0(r, MCPRunQueuePriorityToolInputV0{}, "path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusMethodNotAllowed, newMCPRunQueuePriorityHTTPErrorV0(r, MCPRunQueuePriorityToolInputV0{}, "method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	if handler.executor == nil {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusServiceUnavailable, newMCPRunQueuePriorityHTTPErrorV0(r, MCPRunQueuePriorityToolInputV0{}, "executor", "run_queue_no_configurado"))
		return
	}
	var input MCPRunQueuePriorityToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusBadRequest, newMCPRunQueuePriorityHTTPErrorV0(r, input, "body", code))
		return
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action == "" {
		action = MCPRunQueuePriorityActionRankV0
	}
	input, issues := normalizeMCPRunQueuePriorityIdentityV0(
		input,
		r.Header.Get(MCPPublicCorrelationHeaderV0),
		MCPPublicMutationHeaderIdempotencyKeyV0(r.Header.Get),
		action == MCPRunQueuePriorityActionSetV0,
	)
	if len(issues) > 0 {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusBadRequest, newMCPRunQueuePriorityErrorV0(input, issues[0].Code, issues[0].Field, issues[0].Code))
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
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get(MCPPublicCorrelationHeaderV0), input.CorrelationID, input.RequestID)
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
		w.Header().Set(MCPPublicCorrelationHeaderV0, result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
