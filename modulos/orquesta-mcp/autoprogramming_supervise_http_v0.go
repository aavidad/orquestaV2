package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const MCPAutoprogrammingSuperviseHTTPPathV0 = "/api/v0/autoprogramming/supervise"

const defaultMCPAutoprogrammingSuperviseHTTPResponseTimeoutV0 = 15 * time.Second

func NewMCPAutoprogrammingSuperviseHTTPHandlerV0(
	executor MCPTransportRunSupervisorExecutorV0,
) http.Handler {
	return newMCPAutoprogrammingSuperviseHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPAutoprogrammingSuperviseHTTPResponseTimeoutV0,
	)
}

func newMCPAutoprogrammingSuperviseHTTPHandlerWithTimeoutV0(
	executor MCPTransportRunSupervisorExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpAutoprogrammingSuperviseHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpAutoprogrammingSuperviseHTTPHandlerV0 struct {
	executor        MCPTransportRunSupervisorExecutorV0
	responseTimeout time.Duration
}

type mcpAutoprogrammingSuperviseHTTPInputV0 struct {
	MCPRunSupervisorToolInputV0
	OperatorAdvice mcpAutoprogrammingOperatorAdviceListV0 `json:"operator_advice,omitempty"`
}

type mcpAutoprogrammingSuperviseHTTPResultV0 struct {
	MCPRunSupervisorToolResultV0
	OperatorAdvice []MCPAutoprogrammingOperatorAdviceV0 `json:"operator_advice,omitempty"`
	Diagnostics    []MCPAutoprogrammingDiagnosticV0     `json:"diagnostics,omitempty"`
}

func (handler mcpAutoprogrammingSuperviseHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingSuperviseHTTPPathV0 {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, MCPRunSupervisorToolInputV0{}, "method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	var input mcpAutoprogrammingSuperviseHTTPInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, input.MCPRunSupervisorToolInputV0, "body", code))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingSuperviseHTTPResultV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingSuperviseHTTPErrorV0(r, input.MCPRunSupervisorToolInputV0, "executor", "autoprogramming_supervise_no_configurado"), input.OperatorAdvice)
		return
	}
	result, err, acceptedBackground := handler.executeSupervisorWithResponseTimeoutV0(r, input.MCPRunSupervisorToolInputV0)
	if acceptedBackground {
		writeMCPAutoprogrammingSuperviseHTTPResultV0(w, http.StatusAccepted, result, input.OperatorAdvice)
		return
	}
	if err != nil {
		if result.Estado == MCPRunSupervisorEstadoErrorV0 && len(result.Errores) > 0 {
			writeMCPAutoprogrammingSuperviseHTTPResultV0(w, http.StatusInternalServerError, result, input.OperatorAdvice)
			return
		}
		writeMCPAutoprogrammingSuperviseHTTPResultV0(w, http.StatusInternalServerError, newMCPAutoprogrammingSuperviseExecutorErrorV0(r, input.MCPRunSupervisorToolInputV0, err), input.OperatorAdvice)
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRunSupervisorEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingSuperviseHTTPResultV0(w, status, result, input.OperatorAdvice)
}

type mcpAutoprogrammingSuperviseExecutionV0 struct {
	result MCPRunSupervisorToolResultV0
	err    error
}

func (handler mcpAutoprogrammingSuperviseHTTPHandlerV0) executeSupervisorWithResponseTimeoutV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(runSupervisorExecutionContextV0(r), input)
		return result, err, false
	}
	done := make(chan mcpAutoprogrammingSuperviseExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(runSupervisorExecutionContextV0(r), input)
		done <- mcpAutoprogrammingSuperviseExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPAutoprogrammingSuperviseAcceptedBackgroundV0(r, input), nil, true
	}
}

func newMCPAutoprogrammingSuperviseAcceptedBackgroundV0(
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
			EvidenceRefs: []string{"evidence-ref-autoprogramming-supervise-background-accepted"},
		},
		nil,
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	result.OperationRef = mcpAutoprogrammingSuperviseOperationRefV0(input)
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, result.OperationRef))
	result.NextActions = []string{
		"poll_autoprogramming_status",
		"do_not_relaunch_same_run_ref_while_operation_ref_pending",
	}
	result.Diagnostics = []MCPAutoprogrammingDiagnosticV0{{
		Code:         "autoprogramming_supervise_background_accepted",
		Scope:        mcpAutoprogrammingSuperviseOperationScopeV0(input),
		Message:      "supervision sigue en segundo plano; conservar operation_ref y consultar estado por run_ref/cola",
		EvidenceRefs: result.EvidenceRefs,
	}}
	return result
}

func mcpAutoprogrammingSuperviseOperationScopeV0(input MCPRunSupervisorToolInputV0) string {
	if runRef := strings.TrimSpace(input.RunRef); runRef != "" {
		return "run:" + runRef
	}
	if queueRef := strings.TrimSpace(input.QueueRef); queueRef != "" {
		return "queue:" + queueRef
	}
	return "queue"
}

func mcpAutoprogrammingSuperviseOperationRefV0(input MCPRunSupervisorToolInputV0) string {
	base := firstNonEmptyMCPV0(
		input.IdempotencyKey,
		input.RequestID,
		input.CorrelationID,
		input.RunRef,
		input.QueueRef,
		"queue",
	)
	return "operation-ref-autoprogramming-supervise-" + safeMCPAutoprogrammingOperationRefPartV0(base)
}

func safeMCPAutoprogrammingOperationRefPartV0(value string) string {
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

func newMCPAutoprogrammingSuperviseHTTPErrorV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
	field string,
	message string,
) MCPRunSupervisorToolResultV0 {
	result := NewMCPRunSupervisorErrorResultV0(input, "autoprogramming_supervise_http_error", field, strings.TrimSpace(message))
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	return result
}

func newMCPAutoprogrammingSuperviseExecutorErrorV0(
	r *http.Request,
	input MCPRunSupervisorToolInputV0,
	err error,
) MCPRunSupervisorToolResultV0 {
	message := publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_supervise_executor_error", err)
	result := NewMCPRunSupervisorErrorResultV0(input, "autoprogramming_supervise_executor_error", "executor", message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	return result
}

func writeMCPAutoprogrammingSuperviseHTTPV0(
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

func writeMCPAutoprogrammingSuperviseHTTPResultV0(
	w http.ResponseWriter,
	status int,
	result MCPRunSupervisorToolResultV0,
	advice mcpAutoprogrammingOperatorAdviceListV0,
) {
	normalizedAdvice := advice.normalizedMCPV0(
		firstNonEmptyMCPV0(result.RunRef, result.RequestID, result.CorrelationID),
	)
	if len(normalizedAdvice) == 0 {
		writeMCPAutoprogrammingSuperviseHTTPV0(w, status, result)
		return
	}
	diagnostics := append([]MCPAutoprogrammingDiagnosticV0(nil), result.Diagnostics...)
	diagnostics = append(diagnostics, mcpAutoprogrammingDiagnosticV0(
		"operator_advice_recorded_non_blocking",
		"operator_advice",
		"consejo de operador registrado sin bloquear supervisor",
	))
	payloadResult := result
	payloadResult.Diagnostics = nil
	payload := mcpAutoprogrammingSuperviseHTTPResultV0{
		MCPRunSupervisorToolResultV0: payloadResult,
		OperatorAdvice:               normalizedAdvice,
		Diagnostics:                  diagnostics,
	}
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
