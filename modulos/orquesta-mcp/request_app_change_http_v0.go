package orquestamcp

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const (
	MCPRequestAppChangeHTTPPathV0        = "/api/v0/apps/change"
	MCPRequestAppChangeHTTPPrefixV0      = "/api/v0/apps/"
	MCPRequestAppChangeHTTPDynamicTailV0 = "/changes"
)

func NewMCPRequestAppChangeHTTPHandlerV0(
	executor MCPTransportRequestAppChangeExecutorV0,
) http.Handler {
	return mcpRequestAppChangeHTTPHandlerV0{executor: executor}
}

type mcpRequestAppChangeHTTPHandlerV0 struct {
	executor MCPTransportRequestAppChangeExecutorV0
}

func (handler mcpRequestAppChangeHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requestAppChangePathSupportedV0(r.URL.Path) {
		writeMCPRequestAppChangeHTTPV0(w, http.StatusNotFound, newMCPRequestAppChangeHTTPErrorV0(
			"path",
			"ruta_no_soportada",
			correlationFromRequestAppChangeHTTPV0(r, MCPRequestAppChangeToolInputV0{}),
		))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPRequestAppChangeHTTPV0(w, http.StatusMethodNotAllowed, newMCPRequestAppChangeHTTPErrorV0(
			"method",
			"metodo_no_permitido",
			correlationFromRequestAppChangeHTTPV0(r, MCPRequestAppChangeToolInputV0{}),
		))
		return
	}
	if handler.executor == nil {
		writeMCPRequestAppChangeHTTPV0(w, http.StatusServiceUnavailable, newMCPRequestAppChangeHTTPErrorV0(
			"executor",
			"request_change_no_configurado",
			correlationFromRequestAppChangeHTTPV0(r, MCPRequestAppChangeToolInputV0{}),
		))
		return
	}
	var input MCPRequestAppChangeToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPRequestAppChangeHTTPV0(w, http.StatusBadRequest, newMCPRequestAppChangeHTTPErrorV0(
			"body",
			"request_body_invalido",
			correlationFromRequestAppChangeHTTPV0(r, input),
		))
		return
	}
	input = requestAppChangeInputWithPathRefV0(input, r.URL.Path)
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPRequestAppChangeHTTPV0(w, http.StatusInternalServerError, newMCPRequestAppChangeHTTPErrorV0(
			"executor",
			"request_change_error",
			correlationFromRequestAppChangeHTTPV0(r, input),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRequestAppChangeEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPRequestAppChangeHTTPV0(w, status, result)
}

func requestAppChangePathSupportedV0(path string) bool {
	return path == MCPRequestAppChangeHTTPPathV0 || requestAppChangePathRefV0(path) != ""
}

func requestAppChangeInputWithPathRefV0(
	input MCPRequestAppChangeToolInputV0,
	path string,
) MCPRequestAppChangeToolInputV0 {
	if strings.TrimSpace(input.AppChangeRequest.AppRef) != "" {
		return input
	}
	input.AppChangeRequest.AppRef = requestAppChangePathRefV0(path)
	return input
}

func requestAppChangePathRefV0(pathValue string) string {
	if !strings.HasPrefix(pathValue, MCPRequestAppChangeHTTPPrefixV0) ||
		!strings.HasSuffix(pathValue, MCPRequestAppChangeHTTPDynamicTailV0) {
		return ""
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(pathValue, MCPRequestAppChangeHTTPPrefixV0), MCPRequestAppChangeHTTPDynamicTailV0)
	if raw == "" || strings.Contains(raw, "/") {
		return ""
	}
	value, err := url.PathUnescape(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func newMCPRequestAppChangeHTTPErrorV0(
	field string,
	message string,
	correlationID string,
) MCPRequestAppChangeToolResultV0 {
	return MCPRequestAppChangeToolResultV0{
		Estado:        MCPRequestAppChangeEstadoErrorV0,
		CorrelationID: strings.TrimSpace(correlationID),
		Errores: []MCPAppChangeIssueV0{{
			Code:  strings.TrimSpace(message),
			Field: strings.TrimSpace(field),
		}},
	}
}

func writeMCPRequestAppChangeHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPRequestAppChangeToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func correlationFromRequestAppChangeHTTPV0(
	r *http.Request,
	input MCPRequestAppChangeToolInputV0,
) string {
	return firstNonEmptyMCPV0(
		r.Header.Get("X-Correlation-ID"),
		input.CorrelationID,
		input.RequestID,
		input.AppChangeRequest.CorrelationID,
		input.AppChangeRequest.RequestID,
		input.AppChangeRequest.ChangeRef,
	)
}
