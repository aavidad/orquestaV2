package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const MCPRunSupervisorHTTPPathV0 = "/api/v0/runs/supervise"

const defaultMCPRunSupervisorHTTPResponseTimeoutV0 = 15 * time.Second

func NewMCPRunSupervisorHTTPHandlerV0(
	executor MCPTransportRunSupervisorExecutorV0,
) http.Handler {
	return newMCPRunSupervisorHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPRunSupervisorHTTPResponseTimeoutV0,
	)
}

func newMCPRunSupervisorHTTPHandlerWithTimeoutV0(
	executor MCPTransportRunSupervisorExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpRunSupervisorHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpRunSupervisorHTTPHandlerV0 struct {
	executor        MCPTransportRunSupervisorExecutorV0
	responseTimeout time.Duration
}

func (handler mcpRunSupervisorHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPRunSupervisorHTTPPathV0 {
		writeMCPRunSupervisorHTTPV0(w, http.StatusNotFound, newMCPRunSupervisorHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPRunSupervisorHTTPV0(w, http.StatusMethodNotAllowed, newMCPRunSupervisorHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	if handler.executor == nil {
		writeMCPRunSupervisorHTTPV0(w, http.StatusServiceUnavailable, newMCPRunSupervisorHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "executor", "run_supervisor_no_configurado"))
		return
	}
	var input MCPRunSupervisorToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPRunSupervisorHTTPV0(w, http.StatusBadRequest, newMCPRunSupervisorHTTPErrorV0(r, input, "body", code))
		return
	}
	result, err, acceptedBackground := handler.executeSupervisorWithResponseTimeoutV0(r, input)
	if acceptedBackground {
		writeMCPRunSupervisorHTTPV0(w, http.StatusAccepted, result)
		return
	}
	if err != nil {
		if result.Estado == MCPRunSupervisorEstadoErrorV0 && len(result.Errores) > 0 {
			writeMCPRunSupervisorHTTPV0(w, http.StatusInternalServerError, result)
			return
		}
		writeMCPRunSupervisorHTTPV0(w, http.StatusInternalServerError, newMCPRunSupervisorHTTPErrorV0(
			r,
			input,
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("run_supervisor_error", err),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRunSupervisorEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPRunSupervisorHTTPV0(w, status, result)
}

type mcpRunSupervisorHTTPExecutionV0 struct {
	result MCPRunSupervisorToolResultV0
	err    error
}

func (handler mcpRunSupervisorHTTPHandlerV0) executeSupervisorWithResponseTimeoutV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(runSupervisorExecutionContextV0(r), input)
		return result, err, false
	}
	done := make(chan mcpRunSupervisorHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(runSupervisorExecutionContextV0(r), input)
		done <- mcpRunSupervisorHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPRunSupervisorAcceptedBackgroundV0(r, input), nil, true
	}
}

func newMCPRunSupervisorAcceptedBackgroundV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
) MCPRunSupervisorToolResultV0 {
	result := NewMCPRunSupervisorOKResultV0(
		input,
		input.RunRef,
		"accepted_background",
		0,
		MCPRunSupervisorSnapshotV0{
			Status:       "accepted_background",
			SessionRef:   strings.TrimSpace(input.RunRef),
			EvidenceRefs: []string{"evidence-ref-run-supervisor-background-accepted"},
		},
		nil,
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	result.OperationRef = mcpRunSupervisorOperationRefV0(input)
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, result.OperationRef))
	result.NextActions = []string{
		"poll_director_stats_or_run_queue",
		"do_not_relaunch_same_run_ref_while_operation_ref_pending",
	}
	result.Diagnostics = []MCPAutoprogrammingDiagnosticV0{{
		Code:         "run_supervisor_background_accepted",
		Scope:        mcpRunSupervisorOperationScopeV0(input),
		Message:      "supervision sigue en segundo plano; conservar operation_ref y consultar estado por run_ref/cola",
		EvidenceRefs: result.EvidenceRefs,
	}}
	return result
}

func mcpRunSupervisorOperationScopeV0(input MCPRunSupervisorToolInputV0) string {
	if runRef := strings.TrimSpace(input.RunRef); runRef != "" {
		return "run:" + runRef
	}
	if queueRef := strings.TrimSpace(input.QueueRef); queueRef != "" {
		return "queue:" + queueRef
	}
	return "queue"
}

func mcpRunSupervisorOperationRefV0(input MCPRunSupervisorToolInputV0) string {
	base := firstNonEmptyMCPV0(
		input.IdempotencyKey,
		input.RequestID,
		input.CorrelationID,
		input.RunRef,
		input.QueueRef,
		"queue",
	)
	return "operation-ref-run-supervisor-" + safeMCPRunSupervisorOperationRefPartV0(base)
}

func safeMCPRunSupervisorOperationRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '.' || r == '-'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "queue"
	}
	return out
}

func newMCPRunSupervisorHTTPErrorV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
	field string,
	message string,
) MCPRunSupervisorToolResultV0 {
	result := NewMCPRunSupervisorErrorResultV0(input, "run_supervisor_http_error", field, message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func writeMCPRunSupervisorHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPRunSupervisorToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
