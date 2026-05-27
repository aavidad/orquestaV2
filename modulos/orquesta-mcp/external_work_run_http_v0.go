package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	MCPExternalWorkRunHTTPPathV0                  = "/api/v0/external-work/run"
	MCPExternalWorkRunHTTPErrorCodeV0             = "external_work_run_http_error"
	MCPExternalWorkRunHTTPNotConfiguredCodeV0     = "external_work_run_no_configurado"
	MCPExternalWorkRunHTTPExecutorErrorCodeV0     = "external_work_run_error"
	MCPExternalWorkRunHTTPInvalidBodyCodeV0       = MCPPublicErrBodyInvalidV0
	MCPExternalWorkRunHTTPUnsupportedPathCodeV0   = MCPPublicErrPathUnsupportedV0
	MCPExternalWorkRunHTTPUnsupportedMethodCodeV0 = MCPPublicErrMethodNotAllowedV0
)

func NewMCPExternalWorkRunHTTPHandlerV0(
	executor MCPTransportExternalWorkRunExecutorV0,
) http.Handler {
	return mcpExternalWorkRunHTTPHandlerV0{executor: executor}
}

type mcpExternalWorkRunHTTPHandlerV0 struct {
	executor MCPTransportExternalWorkRunExecutorV0
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
	result, err := handler.executor.Execute(r.Context(), input)
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
}
