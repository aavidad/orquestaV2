package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const defaultMCPCodebaseStatusHTTPResponseTimeoutV0 = 5 * time.Second

func NewMCPCodebaseStatusHTTPHandlerV0(
	executor MCPTransportCodebaseStatusExecutorV0,
) http.Handler {
	return NewMCPCodebaseStatusHTTPHandlerWithResponseTimeoutV0(
		executor,
		defaultMCPCodebaseStatusHTTPResponseTimeoutV0,
	)
}

func NewMCPCodebaseStatusHTTPHandlerWithResponseTimeoutV0(
	executor MCPTransportCodebaseStatusExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	if responseTimeout <= 0 {
		responseTimeout = defaultMCPCodebaseStatusHTTPResponseTimeoutV0
	}
	return mcpCodebaseStatusHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpCodebaseStatusHTTPHandlerV0 struct {
	executor        MCPTransportCodebaseStatusExecutorV0
	responseTimeout time.Duration
}

func (handler mcpCodebaseStatusHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPCodebaseStatusHTTPPathV0 {
		writeMCPCodebaseStatusHTTPV0(w, http.StatusNotFound, newMCPCodebaseStatusHTTPErrorV0(
			r,
			MCPCodebaseStatusToolInputV0{},
			"path",
			MCPCodebaseStatusHTTPUnsupportedPathCodeV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPCodebaseStatusHTTPV0(w, http.StatusMethodNotAllowed, newMCPCodebaseStatusHTTPErrorV0(
			r,
			MCPCodebaseStatusToolInputV0{},
			"method",
			MCPCodebaseStatusHTTPUnsupportedMethodCodeV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPCodebaseStatusHTTPV0(w, http.StatusServiceUnavailable, newMCPCodebaseStatusHTTPErrorV0(
			r,
			MCPCodebaseStatusToolInputV0{},
			"executor",
			MCPCodebaseStatusHTTPNotConfiguredCodeV0,
		))
		return
	}
	var input MCPCodebaseStatusToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		writeMCPCodebaseStatusHTTPV0(w, http.StatusBadRequest, newMCPCodebaseStatusHTTPErrorV0(
			r,
			input,
			"body",
			code,
		))
		return
	}
	result, err, timedOut := handler.executeCodebaseStatusWithResponseTimeoutV0(r, input)
	if timedOut {
		writeMCPCodebaseStatusHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		writeMCPCodebaseStatusHTTPV0(w, http.StatusInternalServerError, newMCPCodebaseStatusHTTPErrorV0(
			r,
			input,
			"executor",
			MCPCodebaseStatusHTTPExecutorErrorCodeV0,
		))
		return
	}
	status := http.StatusOK
	if result.Estado == orquestacontext.CodeContextToolingEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPCodebaseStatusHTTPV0(w, status, result)
}

type mcpCodebaseStatusHTTPExecutionV0 struct {
	result MCPCodebaseStatusToolResultV0
	err    error
}

func (handler mcpCodebaseStatusHTTPHandlerV0) executeCodebaseStatusWithResponseTimeoutV0(
	r *http.Request,
	input MCPCodebaseStatusToolInputV0,
) (MCPCodebaseStatusToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpCodebaseStatusHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpCodebaseStatusHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPCodebaseStatusHTTPErrorV0(
			r,
			input,
			"executor",
			orquestacontext.ErrCodeContextProveedorTimeoutV0,
		), nil, true
	}
}

func newMCPCodebaseStatusHTTPErrorV0(
	r *http.Request,
	input MCPCodebaseStatusToolInputV0,
	field string,
	code string,
) MCPCodebaseStatusToolResultV0 {
	return orquestacontext.CodeContextToolingStatusV0{
		SchemaVersion: orquestacontext.CodeContextToolingStatusSchemaVersionV0,
		Estado:        orquestacontext.CodeContextToolingEstadoErrorV0,
		RequestRef:    strings.TrimSpace(input.RequestRef),
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestRef),
		RepositoryRef: strings.TrimSpace(input.RepositoryRef),
		ObservedAt:    strings.TrimSpace(input.ObservedAt),
		Issues: []orquestacontext.CodeContextIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	}
}

func writeMCPCodebaseStatusHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPCodebaseStatusToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
	flushMCPHTTPResponseV0(w)
}
