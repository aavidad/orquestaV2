package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPDirectorStatsHTTPPathV0 = "/api/v0/director/stats"

func NewMCPDirectorStatsHTTPHandlerV0(
	executor MCPTransportDirectorStatsExecutorV0,
) http.Handler {
	return mcpDirectorStatsHTTPHandlerV0{executor: executor}
}

type mcpDirectorStatsHTTPHandlerV0 struct {
	executor MCPTransportDirectorStatsExecutorV0
}

func (handler mcpDirectorStatsHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPDirectorStatsHTTPPathV0 {
		writeMCPDirectorStatsHTTPV0(w, http.StatusNotFound, newMCPDirectorStatsHTTPErrorV0(
			"path",
			MCPPublicErrPathUnsupportedV0,
			correlationFromDirectorStatsHTTPV0(r, MCPDirectorStatsToolInputV0{}),
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPDirectorStatsHTTPV0(w, http.StatusMethodNotAllowed, newMCPDirectorStatsHTTPErrorV0(
			"method",
			MCPPublicErrMethodNotAllowedV0,
			correlationFromDirectorStatsHTTPV0(r, MCPDirectorStatsToolInputV0{}),
		))
		return
	}
	if handler.executor == nil {
		writeMCPDirectorStatsHTTPV0(w, http.StatusServiceUnavailable, newMCPDirectorStatsHTTPErrorV0(
			"executor",
			"director_stats_no_configurado",
			correlationFromDirectorStatsHTTPV0(r, MCPDirectorStatsToolInputV0{}),
		))
		return
	}
	var input MCPDirectorStatsToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPDirectorStatsHTTPV0(w, http.StatusBadRequest, newMCPDirectorStatsHTTPErrorV0(
			"body",
			code,
			correlationFromDirectorStatsHTTPV0(r, input),
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		if result.Estado == MCPDirectorStatsEstadoErrorV0 && len(result.Errores) > 0 {
			writeMCPDirectorStatsHTTPV0(w, http.StatusInternalServerError, result)
			return
		}
		writeMCPDirectorStatsHTTPV0(w, http.StatusInternalServerError, newMCPDirectorStatsHTTPErrorV0(
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("director_stats_executor_error", err),
			correlationFromDirectorStatsHTTPV0(r, input),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPDirectorStatsEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPDirectorStatsHTTPV0(w, status, result)
}

func newMCPDirectorStatsHTTPErrorV0(
	field string,
	message string,
	correlationID string,
) MCPDirectorStatsToolResultV0 {
	return MCPDirectorStatsToolResultV0{
		Estado:        MCPDirectorStatsEstadoErrorV0,
		CorrelationID: strings.TrimSpace(correlationID),
		Errores: []MCPValidationIssueV0{{
			Code:    "director_stats_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func writeMCPDirectorStatsHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPDirectorStatsToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func correlationFromDirectorStatsHTTPV0(
	r *http.Request,
	input MCPDirectorStatsToolInputV0,
) string {
	return firstNonEmptyMCPV0(
		r.Header.Get("X-Correlation-ID"),
		input.CorrelationID,
		input.RequestID,
		input.RunRef,
	)
}
