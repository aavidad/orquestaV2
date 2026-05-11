package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPArrancarDirectorAppHTTPPathV0 = "/api/v0/apps/director"

func NewMCPArrancarDirectorAppHTTPHandlerV0(
	executor MCPTransportArrancarDirectorAppExecutorV0,
) http.Handler {
	return mcpArrancarDirectorAppHTTPHandlerV0{executor: executor}
}

type mcpArrancarDirectorAppHTTPHandlerV0 struct {
	executor MCPTransportArrancarDirectorAppExecutorV0
}

func (handler mcpArrancarDirectorAppHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPArrancarDirectorAppHTTPPathV0 {
		writeMCPArrancarDirectorAppHTTPV0(w, http.StatusNotFound, newMCPArrancarDirectorHTTPErrorV0(
			"path",
			"ruta_no_soportada",
			correlationFromArrancarDirectorHTTPV0(r, MCPArrancarDirectorAppToolInputV0{}),
		))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPArrancarDirectorAppHTTPV0(w, http.StatusMethodNotAllowed, newMCPArrancarDirectorHTTPErrorV0(
			"method",
			"metodo_no_permitido",
			correlationFromArrancarDirectorHTTPV0(r, MCPArrancarDirectorAppToolInputV0{}),
		))
		return
	}
	if handler.executor == nil {
		writeMCPArrancarDirectorAppHTTPV0(w, http.StatusServiceUnavailable, newMCPArrancarDirectorHTTPErrorV0(
			"executor",
			"arrancar_director_no_configurado",
			correlationFromArrancarDirectorHTTPV0(r, MCPArrancarDirectorAppToolInputV0{}),
		))
		return
	}
	var input MCPArrancarDirectorAppToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPArrancarDirectorAppHTTPV0(w, http.StatusBadRequest, newMCPArrancarDirectorHTTPErrorV0(
			"body",
			"request_body_invalido",
			correlationFromArrancarDirectorHTTPV0(r, input),
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPArrancarDirectorAppHTTPV0(w, http.StatusInternalServerError, newMCPArrancarDirectorHTTPErrorV0(
			"executor",
			"arrancar_director_error",
			correlationFromArrancarDirectorHTTPV0(r, input),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPArrancarDirectorAppEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPArrancarDirectorAppHTTPV0(w, status, result)
}

func newMCPArrancarDirectorHTTPErrorV0(
	field string,
	message string,
	correlationID string,
) MCPArrancarDirectorAppToolResultV0 {
	return MCPArrancarDirectorAppToolResultV0{
		Estado:        MCPArrancarDirectorAppEstadoErrorV0,
		CorrelationID: strings.TrimSpace(correlationID),
		Errores: []MCPValidationIssueV0{{
			Code:    "arrancar_director_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func writeMCPArrancarDirectorAppHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPArrancarDirectorAppToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func correlationFromArrancarDirectorHTTPV0(
	r *http.Request,
	input MCPArrancarDirectorAppToolInputV0,
) string {
	return firstNonEmptyMCPV0(
		r.Header.Get("X-Correlation-ID"),
		input.CorrelationID,
		input.RequestID,
		input.AppSpecRequest.RequestID,
	)
}
