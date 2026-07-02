package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const MCPRunControlHTTPPathV0 = "/api/v0/runs/control"
const defaultMCPRunControlHTTPResponseTimeoutV0 = 2 * time.Second

func NewMCPRunControlHTTPHandlerV0(
	executor MCPTransportRunControlExecutorV0,
) http.Handler {
	return newMCPRunControlHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPRunControlHTTPResponseTimeoutV0,
	)
}

func newMCPRunControlHTTPHandlerWithTimeoutV0(
	executor MCPTransportRunControlExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpRunControlHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpRunControlHTTPHandlerV0 struct {
	executor        MCPTransportRunControlExecutorV0
	responseTimeout time.Duration
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
	result, err, timedOut := handler.executeRunControlWithResponseTimeoutV0(r, input)
	if timedOut {
		writeMCPRunControlHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		writeMCPRunControlHTTPV0(w, http.StatusInternalServerError, newMCPRunControlHTTPErrorV0(r, input, "executor", "run_control_error"))
		return
	}
	status := mcpRunControlHTTPStatusFromResultV0(result)
	writeMCPRunControlHTTPV0(w, status, result)
}

func mcpRunControlHTTPStatusFromResultV0(result MCPRunControlToolResultV0) int {
	if result.Estado != MCPRunControlEstadoErrorV0 {
		return http.StatusOK
	}
	for _, issue := range result.Errores {
		if strings.TrimSpace(issue.Code) == "control_not_propagated_to_goal_backend" {
			return http.StatusConflict
		}
	}
	return http.StatusBadRequest
}

type mcpRunControlHTTPExecutionV0 struct {
	result MCPRunControlToolResultV0
	err    error
}

func (handler mcpRunControlHTTPHandlerV0) executeRunControlWithResponseTimeoutV0(
	r *http.Request,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpRunControlHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpRunControlHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPRunControlTimeoutResultV0(r, input), nil, true
	}
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

func newMCPRunControlTimeoutResultV0(
	r *http.Request,
	input MCPRunControlToolInputV0,
) MCPRunControlToolResultV0 {
	result := newMCPRunControlErrorV0(input, "run_control_timeout", "executor")
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get(MCPPublicCorrelationHeaderV0), result.CorrelationID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = "control de run excedio la ventana HTTP acotada"
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
	flushMCPHTTPResponseV0(w)
}
