package orquestamcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
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
		operations:      newMCPObserveHTTPAsyncCoordinatorV0[MCPObserveAppDirectorGoalToolResultV0](),
	}
}

type mcpObserveAppDirectorGoalHTTPHandlerV0 struct {
	executor        MCPTransportObserveAppDirectorGoalExecutorV0
	responseTimeout time.Duration
	operations      *mcpObserveHTTPAsyncCoordinatorV0[MCPObserveAppDirectorGoalToolResultV0]
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
		w.Header().Set("Location", r.URL.Path)
		w.Header().Set("Retry-After", "1")
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusAccepted, result)
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

const mcpObserveHTTPAsyncExecutionMaxRuntimeV0 = time.Hour

type mcpObserveHTTPAsyncOperationV0[T any] struct {
	done   chan struct{}
	result T
	err    error
}

type mcpObserveHTTPAsyncCoordinatorV0[T any] struct {
	mu         sync.Mutex
	operations map[string]*mcpObserveHTTPAsyncOperationV0[T]
}

func newMCPObserveHTTPAsyncCoordinatorV0[T any]() *mcpObserveHTTPAsyncCoordinatorV0[T] {
	return &mcpObserveHTTPAsyncCoordinatorV0[T]{operations: map[string]*mcpObserveHTTPAsyncOperationV0[T]{}}
}

func (coordinator *mcpObserveHTTPAsyncCoordinatorV0[T]) getOrStartV0(
	key string,
	execute func() (T, error),
) *mcpObserveHTTPAsyncOperationV0[T] {
	coordinator.mu.Lock()
	if operation, ok := coordinator.operations[key]; ok {
		coordinator.mu.Unlock()
		return operation
	}
	operation := &mcpObserveHTTPAsyncOperationV0[T]{done: make(chan struct{})}
	coordinator.operations[key] = operation
	coordinator.mu.Unlock()

	go func() {
		operation.result, operation.err = execute()
		coordinator.mu.Lock()
		close(operation.done)
		coordinator.mu.Unlock()
	}()
	return operation
}

func (coordinator *mcpObserveHTTPAsyncCoordinatorV0[T]) forgetCompletedV0(
	key string,
	operation *mcpObserveHTTPAsyncOperationV0[T],
) {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.operations[key] == operation {
		delete(coordinator.operations, key)
	}
}

func mcpObserveHTTPAsyncOperationKeyV0(runRef, _ string, _ string) string {
	// Solo puede existir una observacion activa por run, aunque un cliente
	// reintente con request_id o correlation_id diferentes.
	return strings.TrimSpace(runRef)
}

func mcpObserveHTTPAsyncOperationRefV0(runRef string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(runRef)))
	return "operation-ref-observe-goal-" + hex.EncodeToString(digest[:16])
}

func (handler mcpObserveAppDirectorGoalHTTPHandlerV0) executeObserveAppDirectorGoalWithResponseTimeoutV0(
	r *http.Request,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error, bool) {
	if handler.operations == nil {
		handler.operations = newMCPObserveHTTPAsyncCoordinatorV0[MCPObserveAppDirectorGoalToolResultV0]()
	}
	operationKey := mcpObserveHTTPAsyncOperationKeyV0(input.RunRef, input.RequestID, input.CorrelationID)
	operation := handler.operations.getOrStartV0(
		operationKey,
		func() (MCPObserveAppDirectorGoalToolResultV0, error) {
			// El trabajo de observacion actualiza estado durable. El deadline o la
			// desconexion del cliente HTTP solo acota la respuesta, nunca ese trabajo.
			executionCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), mcpObserveHTTPAsyncExecutionMaxRuntimeV0)
			defer cancel()
			return handler.executor.Execute(executionCtx, input)
		},
	)
	timeout := handler.responseTimeout
	if timeout <= 0 {
		<-operation.done
		handler.operations.forgetCompletedV0(operationKey, operation)
		return operation.result, operation.err, false
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-operation.done:
		handler.operations.forgetCompletedV0(operationKey, operation)
		return operation.result, operation.err, false
	case <-r.Context().Done():
		return mcpObserveHTTPAcceptedResultV0(handler.newMCPObserveAppDirectorGoalHTTPTimeoutResultV0(r, input), input.RunRef), nil, true
	case <-timer.C:
		return mcpObserveHTTPAcceptedResultV0(handler.newMCPObserveAppDirectorGoalHTTPTimeoutResultV0(r, input), input.RunRef), nil, true
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
	snapshotCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), defaultMCPObserveAppDirectorGoalHTTPSnapshotTimeoutV0)
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
		"observe_app_director_goal sigue ejecutandose fuera de la ventana HTTP acotada",
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
	result.Partial = true
	result.RecommendedAction = "observe_later"
	result.Summary = "observacion aceptada en segundo plano; conservar request_id y run_ref para consultar el mismo resultado sin relanzarla"
	result.EvidenceRefs = compactStringsMCPV0([]string{
		"evidence-ref-observe-app-director-goal-timeout",
		strings.TrimSpace(input.RunRef),
	})
	return result
}

func mcpObserveHTTPAcceptedResultV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	runRef string,
) MCPObserveAppDirectorGoalToolResultV0 {
	result.Estado = MCPObserveAppDirectorGoalEstadoOKV0
	result.OperationRef = mcpObserveHTTPAsyncOperationRefV0(runRef)
	result.Errores = nil
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
