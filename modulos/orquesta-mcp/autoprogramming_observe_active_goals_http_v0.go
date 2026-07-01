package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultMCPAutoprogrammingObserveActiveGoalsHTTPResponseTimeoutV0 = 2 * time.Second

func NewMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0(
	executor MCPTransportAutoprogrammingObserveActiveGoalsExecutorV0,
) http.Handler {
	return newMCPAutoprogrammingObserveActiveGoalsHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPAutoprogrammingObserveActiveGoalsHTTPResponseTimeoutV0,
	)
}

func newMCPAutoprogrammingObserveActiveGoalsHTTPHandlerWithTimeoutV0(
	executor MCPTransportAutoprogrammingObserveActiveGoalsExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpAutoprogrammingObserveActiveGoalsHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
		operations:      newMCPAutoprogrammingObserveActiveGoalsOperationStoreV0(),
	}
}

type mcpAutoprogrammingObserveActiveGoalsHTTPHandlerV0 struct {
	executor        MCPTransportAutoprogrammingObserveActiveGoalsExecutorV0
	responseTimeout time.Duration
	operations      *mcpAutoprogrammingObserveActiveGoalsOperationStoreV0
}

func (handler mcpAutoprogrammingObserveActiveGoalsHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingObserveActiveGoalsHTTPPathV0 {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
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
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
			"executor",
			"autoprogramming_observe_active_goals_no_configurado",
		))
		return
	}
	var input MCPAutoprogrammingObserveActiveGoalsToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(r, input, "body", code))
		return
	}
	result, err, acceptedBackground := handler.executeObserveActiveGoalsWithResponseTimeoutV0(r, input)
	if acceptedBackground {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusAccepted, result)
		return
	}
	if err != nil {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusInternalServerError, mcpAutoprogrammingObserveActiveGoalsErrorV0(
			input,
			"autoprogramming_observe_active_goals_execute_error",
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_observe_active_goals_execute_error", err),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingObserveActiveGoalsEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, status, result)
}

type mcpAutoprogrammingObserveActiveGoalsHTTPExecutionV0 struct {
	result MCPAutoprogrammingObserveActiveGoalsToolResultV0
	err    error
}

type mcpAutoprogrammingObserveActiveGoalsOperationStoreV0 struct {
	mu     sync.Mutex
	active map[string]bool
}

func newMCPAutoprogrammingObserveActiveGoalsOperationStoreV0() *mcpAutoprogrammingObserveActiveGoalsOperationStoreV0 {
	return &mcpAutoprogrammingObserveActiveGoalsOperationStoreV0{active: map[string]bool{}}
}

func (store *mcpAutoprogrammingObserveActiveGoalsOperationStoreV0) beginV0(operationRef string) bool {
	if store == nil {
		return true
	}
	operationRef = strings.TrimSpace(operationRef)
	if operationRef == "" {
		return true
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.active[operationRef] {
		return false
	}
	store.active[operationRef] = true
	return true
}

func (store *mcpAutoprogrammingObserveActiveGoalsOperationStoreV0) finishV0(operationRef string) {
	if store == nil {
		return
	}
	operationRef = strings.TrimSpace(operationRef)
	if operationRef == "" {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.active, operationRef)
}

func (handler mcpAutoprogrammingObserveActiveGoalsHTTPHandlerV0) executeObserveActiveGoalsWithResponseTimeoutV0(
	r *http.Request,
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) (MCPAutoprogrammingObserveActiveGoalsToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(runSupervisorExecutionContextV0(r), input)
		return result, err, false
	}
	operationRef := mcpAutoprogrammingObserveActiveGoalsOperationRefV0(input)
	if !handler.operations.beginV0(operationRef) {
		return newMCPAutoprogrammingObserveActiveGoalsAlreadyRunningV0(r, input), nil, true
	}
	done := make(chan mcpAutoprogrammingObserveActiveGoalsHTTPExecutionV0, 1)
	ctx, cancel := context.WithCancel(runSupervisorExecutionContextV0(r))
	go func() {
		defer handler.operations.finishV0(operationRef)
		defer cancel()
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpAutoprogrammingObserveActiveGoalsHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		cancel()
		return newMCPAutoprogrammingObserveActiveGoalsAcceptedBackgroundV0(r, input), nil, true
	}
}

func newMCPAutoprogrammingObserveActiveGoalsAlreadyRunningV0(
	r *http.Request,
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) MCPAutoprogrammingObserveActiveGoalsToolResultV0 {
	result := newMCPAutoprogrammingObserveActiveGoalsAcceptedBackgroundV0(r, input)
	result.Diagnostics = append(result.Diagnostics, MCPAutoprogrammingDiagnosticV0{
		Code:         "autoprogramming_observe_active_goals_operation_already_running",
		Scope:        mcpAutoprogrammingObserveActiveGoalsOperationScopeV0(input),
		Message:      "operation_ref ya tiene una observacion batch activa; no se duplica la pasada",
		EvidenceRefs: result.EvidenceRefs,
	})
	return result
}

func newMCPAutoprogrammingObserveActiveGoalsAcceptedBackgroundV0(
	r *http.Request,
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) MCPAutoprogrammingObserveActiveGoalsToolResultV0 {
	result := newMCPAutoprogrammingObserveActiveGoalsBaseV0(input)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	result.OperationRef = mcpAutoprogrammingObserveActiveGoalsOperationRefV0(input)
	result.EvidenceRefs = compactStringsMCPV0([]string{
		"evidence-ref-autoprogramming-observe-active-goals-background-accepted",
		result.OperationRef,
	})
	result.NextActions = []string{
		"poll_autoprogramming_status",
		"do_not_relaunch_same_observe_active_goals_operation_ref",
	}
	result.Diagnostics = []MCPAutoprogrammingDiagnosticV0{{
		Code:         "autoprogramming_observe_active_goals_background_accepted",
		Scope:        mcpAutoprogrammingObserveActiveGoalsOperationScopeV0(input),
		Message:      "observacion de goals activos sigue en segundo plano; conservar operation_ref y consultar autoprogramming/status",
		EvidenceRefs: result.EvidenceRefs,
	}}
	return result
}

func mcpAutoprogrammingObserveActiveGoalsOperationScopeV0(
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) string {
	runRefs := compactStringsMCPV0(input.RunRefs)
	if len(runRefs) == 1 {
		return "run:" + runRefs[0]
	}
	if len(runRefs) > 1 {
		return "runs"
	}
	return "goals"
}

func mcpAutoprogrammingObserveActiveGoalsOperationRefV0(
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) string {
	base := firstNonEmptyMCPV0(
		input.RequestID,
		input.CorrelationID,
		strings.Join(compactStringsMCPV0(input.RunRefs), "-"),
		"goals",
	)
	return "operation-ref-autoprogramming-observe-active-goals-" + safeMCPAutoprogrammingOperationRefPartV0(base)
}

func newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
	r *http.Request,
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
	field string,
	message string,
) MCPAutoprogrammingObserveActiveGoalsToolResultV0 {
	result := mcpAutoprogrammingObserveActiveGoalsErrorV0(
		input,
		"autoprogramming_observe_active_goals_http_error",
		field,
		message,
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	return result
}

func writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingObserveActiveGoalsToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
