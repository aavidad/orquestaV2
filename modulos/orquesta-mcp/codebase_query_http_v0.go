package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const defaultMCPCodebaseQueryHTTPResponseTimeoutV0 = 5 * time.Second

func NewMCPCodebaseQueryHTTPHandlerV0(
	executor MCPTransportCodebaseQueryExecutorV0,
) http.Handler {
	return NewMCPCodebaseQueryHTTPHandlerWithResponseTimeoutV0(
		executor,
		defaultMCPCodebaseQueryHTTPResponseTimeoutV0,
	)
}

func NewMCPCodebaseQueryHTTPHandlerWithResponseTimeoutV0(
	executor MCPTransportCodebaseQueryExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	if responseTimeout <= 0 {
		responseTimeout = defaultMCPCodebaseQueryHTTPResponseTimeoutV0
	}
	return mcpCodebaseQueryHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpCodebaseQueryHTTPHandlerV0 struct {
	executor        MCPTransportCodebaseQueryExecutorV0
	responseTimeout time.Duration
}

func (handler mcpCodebaseQueryHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPCodebaseQueryHTTPPathV0 {
		writeMCPCodebaseQueryHTTPV0(w, http.StatusNotFound, newMCPCodebaseQueryHTTPErrorV0(
			r,
			MCPCodebaseQueryToolInputV0{},
			"path",
			MCPCodebaseQueryHTTPUnsupportedPathCodeV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPCodebaseQueryHTTPV0(w, http.StatusMethodNotAllowed, newMCPCodebaseQueryHTTPErrorV0(
			r,
			MCPCodebaseQueryToolInputV0{},
			"method",
			MCPCodebaseQueryHTTPUnsupportedMethodCodeV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPCodebaseQueryHTTPV0(w, http.StatusServiceUnavailable, newMCPCodebaseQueryHTTPErrorV0(
			r,
			MCPCodebaseQueryToolInputV0{},
			"executor",
			MCPCodebaseQueryHTTPNotConfiguredCodeV0,
		))
		return
	}
	var input MCPCodebaseQueryToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		writeMCPCodebaseQueryHTTPV0(w, http.StatusBadRequest, newMCPCodebaseQueryHTTPErrorV0(
			r,
			input,
			"body",
			code,
		))
		return
	}
	result, err, timedOut := handler.executeCodebaseQueryWithResponseTimeoutV0(r, input)
	if timedOut {
		writeMCPCodebaseQueryHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		writeMCPCodebaseQueryHTTPV0(w, http.StatusInternalServerError, newMCPCodebaseQueryHTTPErrorV0(
			r,
			input,
			"executor",
			MCPCodebaseQueryHTTPExecutorErrorCodeV0,
		))
		return
	}
	status := http.StatusOK
	if result.Estado == orquestacontext.CodeContextEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPCodebaseQueryHTTPV0(w, status, result)
}

type mcpCodebaseQueryHTTPExecutionV0 struct {
	result MCPCodebaseQueryToolResultV0
	err    error
}

func (handler mcpCodebaseQueryHTTPHandlerV0) executeCodebaseQueryWithResponseTimeoutV0(
	r *http.Request,
	input MCPCodebaseQueryToolInputV0,
) (MCPCodebaseQueryToolResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		result, err := handler.executor.Execute(r.Context(), input)
		return result, err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpCodebaseQueryHTTPExecutionV0, 1)
	go func() {
		result, err := handler.executor.Execute(ctx, input)
		done <- mcpCodebaseQueryHTTPExecutionV0{result: result, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return execution.result, execution.err, false
	case <-timer.C:
		return newMCPCodebaseQueryHTTPErrorV0(
			r,
			input,
			"executor",
			orquestacontext.ErrCodeContextProveedorTimeoutV0,
		), nil, true
	}
}

func newMCPCodebaseQueryHTTPErrorV0(
	r *http.Request,
	input MCPCodebaseQueryToolInputV0,
	field string,
	code string,
) MCPCodebaseQueryToolResultV0 {
	return orquestacontext.CodeContextResultV0{
		SchemaVersion: orquestacontext.CodeContextResultSchemaVersionV0,
		Estado:        orquestacontext.CodeContextEstadoErrorV0,
		RequestRef:    strings.TrimSpace(input.RequestRef),
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestRef),
		RepositoryRef: strings.TrimSpace(input.RepositoryRef),
		WorktreeRef:   strings.TrimSpace(input.WorktreeRef),
		CommitRef:     strings.TrimSpace(input.CommitRef),
		QueryKind:     strings.TrimSpace(input.QueryKind),
		IndexerPolicy: orquestacontext.CodeContextProviderPolicyCentralOnlyV0,
		Issues: []orquestacontext.CodeContextIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	}
}

func writeMCPCodebaseQueryHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPCodebaseQueryToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
	flushMCPHTTPResponseV0(w)
}
