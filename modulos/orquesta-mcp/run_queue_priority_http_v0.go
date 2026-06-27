package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const MCPRunQueuePriorityHTTPPathV0 = "/api/v0/runs/queue/priority"
const defaultMCPRunQueuePriorityHTTPResponseTimeoutV0 = 2 * time.Second

func NewMCPRunQueuePriorityHTTPHandlerV0(
	executor MCPTransportRunQueuePriorityExecutorV0,
) http.Handler {
	return newMCPRunQueuePriorityHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPRunQueuePriorityHTTPResponseTimeoutV0,
	)
}

func newMCPRunQueuePriorityHTTPHandlerWithTimeoutV0(
	executor MCPTransportRunQueuePriorityExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpRunQueuePriorityHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpRunQueuePriorityHTTPHandlerV0 struct {
	executor        MCPTransportRunQueuePriorityExecutorV0
	responseTimeout time.Duration
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
	input.Action = action
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
	result, err, timedOut := handler.executeRunQueuePriorityWithResponseTimeoutV0(r, input, action)
	if timedOut {
		writeMCPRunQueuePriorityHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
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

type mcpRunQueuePriorityHTTPExecutionV0 struct {
	result MCPRunQueuePriorityToolResultV0
	err    error
}

func (handler mcpRunQueuePriorityHTTPHandlerV0) executeRunQueuePriorityWithResponseTimeoutV0(
	r *http.Request,
	input MCPRunQueuePriorityToolInputV0,
	action string,
) (MCPRunQueuePriorityToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 || strings.EqualFold(strings.TrimSpace(action), MCPRunQueuePriorityActionSetV0) {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpRunQueuePriorityHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpRunQueuePriorityHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPRunQueuePriorityTimeoutResultV0(r, input), nil, true
	}
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

func newMCPRunQueuePriorityTimeoutResultV0(
	r *http.Request,
	input MCPRunQueuePriorityToolInputV0,
) MCPRunQueuePriorityToolResultV0 {
	result := newMCPRunQueuePriorityErrorV0(
		input,
		"run_queue_priority_timeout",
		"executor",
		"consulta de cola excedio la ventana HTTP acotada",
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get(MCPPublicCorrelationHeaderV0), result.CorrelationID)
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
