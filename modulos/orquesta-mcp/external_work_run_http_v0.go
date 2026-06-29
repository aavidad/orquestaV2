package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	MCPExternalWorkRunHTTPPathV0                  = "/api/v0/external-work/run"
	MCPExternalWorkRunHTTPErrorCodeV0             = "external_work_run_http_error"
	MCPExternalWorkRunHTTPNotConfiguredCodeV0     = "external_work_run_no_configurado"
	MCPExternalWorkRunHTTPExecutorErrorCodeV0     = "external_work_run_error"
	MCPExternalWorkRunHTTPTimeoutCodeV0           = "external_work_run_timeout"
	MCPExternalWorkRunHTTPInvalidBodyCodeV0       = MCPPublicErrBodyInvalidV0
	MCPExternalWorkRunHTTPUnsupportedPathCodeV0   = MCPPublicErrPathUnsupportedV0
	MCPExternalWorkRunHTTPUnsupportedMethodCodeV0 = MCPPublicErrMethodNotAllowedV0
)

const defaultMCPExternalWorkRunHTTPResponseTimeoutV0 = 30 * time.Second

func NewMCPExternalWorkRunHTTPHandlerV0(
	executor MCPTransportExternalWorkRunExecutorV0,
) http.Handler {
	return NewMCPExternalWorkRunHTTPHandlerWithResponseTimeoutV0(
		executor,
		defaultMCPExternalWorkRunHTTPResponseTimeoutV0,
	)
}

func NewMCPExternalWorkRunHTTPHandlerWithResponseTimeoutV0(
	executor MCPTransportExternalWorkRunExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	if responseTimeout <= 0 {
		responseTimeout = defaultMCPExternalWorkRunHTTPResponseTimeoutV0
	}
	return newMCPExternalWorkRunHTTPHandlerWithTimeoutV0(
		executor,
		responseTimeout,
	)
}

func newMCPExternalWorkRunHTTPHandlerWithTimeoutV0(
	executor MCPTransportExternalWorkRunExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpExternalWorkRunHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpExternalWorkRunHTTPHandlerV0 struct {
	executor        MCPTransportExternalWorkRunExecutorV0
	responseTimeout time.Duration
}

func (handler mcpExternalWorkRunHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPExternalWorkRunHTTPPathV0 {
		writeMCPExternalWorkRunHTTPV0(w, http.StatusNotFound, newMCPExternalWorkRunHTTPErrorV0(
			r,
			MCPExternalWorkRunToolInputV0{},
			"path",
			MCPExternalWorkRunHTTPUnsupportedPathCodeV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPExternalWorkRunHTTPV0(w, http.StatusMethodNotAllowed, newMCPExternalWorkRunHTTPErrorV0(
			r,
			MCPExternalWorkRunToolInputV0{},
			"method",
			MCPExternalWorkRunHTTPUnsupportedMethodCodeV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPExternalWorkRunHTTPV0(w, http.StatusServiceUnavailable, newMCPExternalWorkRunHTTPErrorV0(
			r,
			MCPExternalWorkRunToolInputV0{},
			"executor",
			MCPExternalWorkRunHTTPNotConfiguredCodeV0,
		))
		return
	}
	var input MCPExternalWorkRunToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileDomainWorkV0); code != "" {
		writeMCPExternalWorkRunHTTPV0(w, http.StatusBadRequest, newMCPExternalWorkRunHTTPErrorV0(
			r,
			input,
			"body",
			code,
		))
		return
	}
	result, err, timedOut := handler.executeExternalWorkRunWithResponseTimeoutV0(r, input)
	if timedOut {
		writeMCPExternalWorkRunHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		writeMCPExternalWorkRunHTTPV0(w, http.StatusInternalServerError, newMCPExternalWorkRunHTTPErrorV0(
			r,
			input,
			"executor",
			MCPExternalWorkRunHTTPExecutorErrorCodeV0,
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPExternalWorkRunEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPExternalWorkRunHTTPV0(w, status, result)
}

type mcpExternalWorkRunHTTPExecutionV0 struct {
	result MCPExternalWorkRunToolResultV0
	err    error
}

func (handler mcpExternalWorkRunHTTPHandlerV0) executeExternalWorkRunWithResponseTimeoutV0(
	r *http.Request,
	input MCPExternalWorkRunToolInputV0,
) (MCPExternalWorkRunToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpExternalWorkRunHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpExternalWorkRunHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPExternalWorkRunHTTPTimeoutResultV0(r, input), nil, true
	}
}

func newMCPExternalWorkRunHTTPErrorV0(
	r *http.Request,
	input MCPExternalWorkRunToolInputV0,
	field string,
	code string,
) MCPExternalWorkRunToolResultV0 {
	return MCPExternalWorkRunToolResultV0{
		Estado:        MCPExternalWorkRunEstadoErrorV0,
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
		Errores: []MCPExternalWorkRunIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	}
}

func newMCPExternalWorkRunHTTPTimeoutResultV0(
	r *http.Request,
	input MCPExternalWorkRunToolInputV0,
) MCPExternalWorkRunToolResultV0 {
	result := newMCPExternalWorkRunHTTPErrorV0(r, input, "executor", MCPExternalWorkRunHTTPTimeoutCodeV0)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = "external-work run excedio la ventana HTTP acotada"
	}
	return result
}

func writeMCPExternalWorkRunHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPExternalWorkRunToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
	flushMCPHTTPResponseV0(w)
}
