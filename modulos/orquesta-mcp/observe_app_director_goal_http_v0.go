package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	MCPObserveAppDirectorGoalHTTPPathV0        = "/api/v0/apps/director/goal/observe"
	MCPObserveAppDirectorGoalHTTPTimeoutCodeV0 = "observe_app_director_goal_timeout"
)

const (
	defaultMCPObserveAppDirectorGoalHTTPResponseTimeoutV0 = 2 * time.Second
	defaultMCPObserveAppDirectorGoalHTTPSnapshotTimeoutV0 = 75 * time.Millisecond
)

func NewMCPObserveAppDirectorGoalHTTPHandlerV0(
	executor MCPTransportObserveAppDirectorGoalExecutorV0,
) http.Handler {
	return newMCPObserveAppDirectorGoalHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPObserveAppDirectorGoalHTTPResponseTimeoutV0,
	)
}

func newMCPObserveAppDirectorGoalHTTPHandlerWithTimeoutV0(
	executor MCPTransportObserveAppDirectorGoalExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpObserveAppDirectorGoalHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpObserveAppDirectorGoalHTTPHandlerV0 struct {
	executor        MCPTransportObserveAppDirectorGoalExecutorV0
	responseTimeout time.Duration
}

func (handler mcpObserveAppDirectorGoalHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPObserveAppDirectorGoalHTTPPathV0 {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusNotFound, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			MCPObserveAppDirectorGoalToolInputV0{},
			"path",
			MCPPublicErrPathUnsupportedV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusMethodNotAllowed, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			MCPObserveAppDirectorGoalToolInputV0{},
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusServiceUnavailable, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			MCPObserveAppDirectorGoalToolInputV0{},
			"executor",
			"observe_app_director_goal_no_configurado",
		))
		return
	}
	var input MCPObserveAppDirectorGoalToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, newMCPObserveAppDirectorGoalHTTPErrorV0(r, input, "body", code))
		return
	}
	if strings.TrimSpace(input.RunRef) == "" {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, newMCPObserveAppDirectorGoalHTTPErrorV0(r, input, "run_ref", "run_ref_requerido"))
		return
	}
	result, err, timedOut := handler.executeObserveAppDirectorGoalWithResponseTimeoutV0(r, input)
	if timedOut {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		if result.Estado == MCPObserveAppDirectorGoalEstadoErrorV0 && len(result.Errores) > 0 {
			result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
			writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, result)
			return
		}
		if publicResult, ok := NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(input, err); ok {
			publicResult.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), publicResult.CorrelationID, input.CorrelationID, input.RequestID)
			writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, publicResult)
			return
		}
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusInternalServerError, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			input,
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("observe_app_director_goal_error", err),
		))
		return
	}
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
	status := http.StatusOK
	if result.Estado == MCPObserveAppDirectorGoalEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPObserveAppDirectorGoalHTTPV0(w, status, result)
}

type mcpObserveAppDirectorGoalHTTPExecutionV0 struct {
	result MCPObserveAppDirectorGoalToolResultV0
	err    error
}

func (handler mcpObserveAppDirectorGoalHTTPHandlerV0) executeObserveAppDirectorGoalWithResponseTimeoutV0(
	r *http.Request,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpObserveAppDirectorGoalHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpObserveAppDirectorGoalHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return handler.newMCPObserveAppDirectorGoalHTTPTimeoutResultV0(r, input), nil, true
	}
}

func (handler mcpObserveAppDirectorGoalHTTPHandlerV0) newMCPObserveAppDirectorGoalHTTPTimeoutResultV0(
	r *http.Request,
	input MCPObserveAppDirectorGoalToolInputV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	timeoutResult := newMCPObserveAppDirectorGoalHTTPTimeoutResultV0(r, input)
	snapshotter, ok := handler.executor.(MCPTransportObserveAppDirectorGoalTimeoutSnapshotExecutorV0)
	if !ok || snapshotter == nil {
		return timeoutResult
	}
	snapshotCtx, cancel := context.WithTimeout(r.Context(), defaultMCPObserveAppDirectorGoalHTTPSnapshotTimeoutV0)
	defer cancel()
	done := make(chan mcpObserveAppDirectorGoalHTTPExecutionV0, 1)
	go func() {
		result, err := snapshotter.ObserveAppDirectorGoalTimeoutSnapshotV0(snapshotCtx, input)
		done <- mcpObserveAppDirectorGoalHTTPExecutionV0{result: result, err: err}
	}()
	select {
	case execution := <-done:
		if execution.err != nil {
			return timeoutResult
		}
		execution.result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), execution.result.CorrelationID, input.CorrelationID, input.RequestID)
		return NewMCPObserveAppDirectorGoalTimeoutResultWithPartialV0(timeoutResult, execution.result)
	case <-snapshotCtx.Done():
		return timeoutResult
	}
}

func newMCPObserveAppDirectorGoalHTTPErrorV0(
	r *http.Request,
	input MCPObserveAppDirectorGoalToolInputV0,
	field string,
	message string,
) MCPObserveAppDirectorGoalToolResultV0 {
	result := NewMCPObserveAppDirectorGoalErrorResultV0(input, "observe_app_director_goal_http_error", field, message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func newMCPObserveAppDirectorGoalHTTPTimeoutResultV0(
	r *http.Request,
	input MCPObserveAppDirectorGoalToolInputV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	result := NewMCPObserveAppDirectorGoalErrorResultV0(
		input,
		MCPObserveAppDirectorGoalHTTPTimeoutCodeV0,
		"executor",
		"observe_app_director_goal excedio la ventana HTTP acotada",
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
	return result
}

func writeMCPObserveAppDirectorGoalHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPObserveAppDirectorGoalToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
	flushMCPHTTPResponseV0(w)
}
