package orquestamcp

import (
	"encoding/json"
	"net/http"
)

func NewMCPDomainWorkHTTPHandlerV0(
	executor MCPDomainWorkExecutorPortV0,
) http.Handler {
	return mcpDomainWorkHTTPHandlerV0{executor: executor}
}

type mcpDomainWorkHTTPHandlerV0 struct {
	executor MCPDomainWorkExecutorPortV0
}

func (handler mcpDomainWorkHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPDomainWorkHTTPPathV0 {
		writeMCPDomainWorkHTTPV0(w, http.StatusNotFound, newMCPDomainWorkHTTPErrorV0(
			r,
			MCPDomainWorkToolInputV0{},
			"path",
			MCPDomainWorkHTTPUnsupportedPathCodeV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPDomainWorkHTTPV0(w, http.StatusMethodNotAllowed, newMCPDomainWorkHTTPErrorV0(
			r,
			MCPDomainWorkToolInputV0{},
			"method",
			MCPDomainWorkHTTPUnsupportedMethodCodeV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPDomainWorkHTTPV0(w, http.StatusServiceUnavailable, newMCPDomainWorkHTTPErrorV0(
			r,
			MCPDomainWorkToolInputV0{},
			"executor",
			MCPDomainWorkHTTPNotConfiguredCodeV0,
		))
		return
	}
	var input MCPDomainWorkToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileDomainWorkV0); code != "" {
		writeMCPDomainWorkHTTPV0(w, http.StatusBadRequest, newMCPDomainWorkHTTPErrorV0(
			r,
			input,
			"body",
			code,
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPDomainWorkHTTPV0(w, http.StatusInternalServerError, newMCPDomainWorkHTTPErrorV0(
			r,
			input,
			"executor",
			MCPDomainWorkHTTPExecutorErrorCodeV0,
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPDomainWorkEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPDomainWorkHTTPV0(w, status, result)
}

func newMCPDomainWorkHTTPErrorV0(
	r *http.Request,
	input MCPDomainWorkToolInputV0,
	field string,
	code string,
) MCPDomainWorkToolResultV0 {
	result := newMCPDomainWorkErrorV0(input, MCPDomainWorkHTTPErrorCodeV0, field, code)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Code = code
		result.Errores[0].Message = code
	}
	return result
}

func writeMCPDomainWorkHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPDomainWorkToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
