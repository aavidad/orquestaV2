package orquestamcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	MCPAutoprogrammingObserveGoalHTTPPathV0        = "/api/v0/autoprogramming/goal/observe"
	MCPAutoprogrammingObserveGoalHTTPTimeoutCodeV0 = "autoprogramming_observe_goal_timeout"
)

const defaultMCPAutoprogrammingObserveGoalHTTPResponseTimeoutV0 = 2 * time.Second

func NewMCPAutoprogrammingObserveGoalHTTPHandlerV0(
	executor MCPTransportAutoprogrammingObserveGoalExecutorV0,
) http.Handler {
	return newMCPAutoprogrammingObserveGoalHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPAutoprogrammingObserveGoalHTTPResponseTimeoutV0,
	)
}

func newMCPAutoprogrammingObserveGoalHTTPHandlerWithTimeoutV0(
	executor MCPTransportAutoprogrammingObserveGoalExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpAutoprogrammingObserveGoalHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpAutoprogrammingObserveGoalHTTPHandlerV0 struct {
	executor        MCPTransportAutoprogrammingObserveGoalExecutorV0
	responseTimeout time.Duration
}

func (handler mcpAutoprogrammingObserveGoalHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingObserveGoalHTTPPathV0 {
		writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingObserveGoalHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveGoalToolInputV0{},
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
		writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingObserveGoalHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveGoalToolInputV0{},
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingObserveGoalHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveGoalToolInputV0{},
			"executor",
			"autoprogramming_observe_goal_no_configurado",
		))
		return
	}
	var input MCPAutoprogrammingObserveGoalToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingObserveGoalHTTPErrorV0(r, input, "body", code))
		return
	}
	if strings.TrimSpace(input.RunRef) == "" {
		writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingObserveGoalHTTPErrorV0(r, input, "run_ref", "run_ref_requerido"))
		return
	}
	result, err, timedOut := handler.executeAutoprogrammingObserveGoalWithResponseTimeoutV0(r, input)
	if timedOut || isMCPAutoprogrammingObserveGoalHTTPRecoverableTimeoutV0(err) {
		if !timedOut {
			result = handler.newMCPAutoprogrammingObserveGoalHTTPTimeoutResultV0(r, input)
		}
		writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		if result.Estado == MCPAutoprogrammingObserveGoalEstadoErrorV0 && len(result.Errores) > 0 {
			result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
			writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusBadRequest, result)
			return
		}
		if publicResult, ok := NewMCPAutoprogrammingObserveGoalErrorResultFromErrorV0(input, err); ok {
			publicResult.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), publicResult.CorrelationID, input.CorrelationID, input.RequestID)
			writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusBadRequest, publicResult)
			return
		}
		writeMCPAutoprogrammingObserveGoalHTTPV0(w, http.StatusInternalServerError, newMCPAutoprogrammingObserveGoalHTTPErrorV0(
			r,
			input,
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_observe_goal_error", err),
		))
		return
	}
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingObserveGoalEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingObserveGoalHTTPV0(w, status, result)
}

func isMCPAutoprogrammingObserveGoalHTTPRecoverableTimeoutV0(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

type mcpAutoprogrammingObserveGoalHTTPExecutionV0 struct {
	result MCPAutoprogrammingObserveGoalToolResultV0
	err    error
}

func (handler mcpAutoprogrammingObserveGoalHTTPHandlerV0) executeAutoprogrammingObserveGoalWithResponseTimeoutV0(
	r *http.Request,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) (MCPAutoprogrammingObserveGoalToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpAutoprogrammingObserveGoalHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpAutoprogrammingObserveGoalHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		cancel()
		return handler.newMCPAutoprogrammingObserveGoalHTTPTimeoutResultV0(r, input), nil, true
	}
}

func (handler mcpAutoprogrammingObserveGoalHTTPHandlerV0) newMCPAutoprogrammingObserveGoalHTTPTimeoutResultV0(
	r *http.Request,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) MCPAutoprogrammingObserveGoalToolResultV0 {
	timeoutResult := newMCPAutoprogrammingObserveGoalHTTPTimeoutResultV0(r, input)
	snapshotter, ok := handler.executor.(MCPTransportObserveAppDirectorGoalTimeoutSnapshotExecutorV0)
	if !ok || snapshotter == nil {
		return timeoutResult
	}
	snapshotCtx, cancel := context.WithTimeout(r.Context(), defaultMCPObserveAppDirectorGoalHTTPSnapshotTimeoutV0)
	defer cancel()
	done := make(chan mcpAutoprogrammingObserveGoalHTTPExecutionV0, 1)
	go func() {
		result, err := snapshotter.ObserveAppDirectorGoalTimeoutSnapshotV0(snapshotCtx, MCPObserveAppDirectorGoalToolInputV0{
			RequestID:     strings.TrimSpace(input.RequestID),
			CorrelationID: strings.TrimSpace(input.CorrelationID),
			RunRef:        strings.TrimSpace(input.RunRef),
			OccurredAt:    strings.TrimSpace(input.OccurredAt),
			RequestedBy:   strings.TrimSpace(input.RequestedBy),
		})
		done <- mcpAutoprogrammingObserveGoalHTTPExecutionV0{result: result, err: err}
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

func newMCPAutoprogrammingObserveGoalHTTPErrorV0(
	r *http.Request,
	input MCPAutoprogrammingObserveGoalToolInputV0,
	field string,
	message string,
) MCPAutoprogrammingObserveGoalToolResultV0 {
	result := NewMCPAutoprogrammingObserveGoalErrorResultV0(input, "autoprogramming_observe_goal_http_error", field, message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func newMCPAutoprogrammingObserveGoalHTTPTimeoutResultV0(
	r *http.Request,
	input MCPAutoprogrammingObserveGoalToolInputV0,
) MCPAutoprogrammingObserveGoalToolResultV0 {
	result := NewMCPAutoprogrammingObserveGoalErrorResultV0(
		input,
		MCPAutoprogrammingObserveGoalHTTPTimeoutCodeV0,
		"executor",
		"autoprogramming observe_goal excedio la ventana HTTP acotada",
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
	result.Partial = true
	result.RecommendedAction = "observe_later"
	result.Summary = "observacion de autoprogramacion incompleta por timeout HTTP; conservar run_ref y reintentar observe acotado antes de replanificar o relanzar"
	result.EvidenceRefs = compactStringsMCPV0([]string{
		"evidence-ref-autoprogramming-observe-goal-timeout",
		strings.TrimSpace(input.RunRef),
	})
	return result
}

func writeMCPAutoprogrammingObserveGoalHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingObserveGoalToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
	flushMCPHTTPResponseV0(w)
}
