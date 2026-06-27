package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPPreviewDirectorAppHTTPPathV0 = "/api/v0/apps/director/preview"

func NewMCPPreviewDirectorAppHTTPHandlerV0(
	executor MCPTransportPreviewDirectorAppExecutorV0,
) http.Handler {
	return mcpPreviewDirectorAppHTTPHandlerV0{executor: executor}
}

type mcpPreviewDirectorAppHTTPHandlerV0 struct {
	executor MCPTransportPreviewDirectorAppExecutorV0
}

func (handler mcpPreviewDirectorAppHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPPreviewDirectorAppHTTPPathV0 {
		writeMCPPreviewDirectorAppHTTPV0(w, http.StatusNotFound, newMCPPreviewDirectorHTTPErrorV0(
			"path",
			MCPPublicErrPathUnsupportedV0,
			correlationFromArrancarDirectorHTTPV0(r, MCPArrancarDirectorAppToolInputV0{}),
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPPreviewDirectorAppHTTPV0(w, http.StatusMethodNotAllowed, newMCPPreviewDirectorHTTPErrorV0(
			"method",
			MCPPublicErrMethodNotAllowedV0,
			correlationFromArrancarDirectorHTTPV0(r, MCPArrancarDirectorAppToolInputV0{}),
		))
		return
	}
	if handler.executor == nil {
		writeMCPPreviewDirectorAppHTTPV0(w, http.StatusServiceUnavailable, newMCPPreviewDirectorHTTPErrorV0(
			"executor",
			"preview_director_no_configurado",
			correlationFromArrancarDirectorHTTPV0(r, MCPArrancarDirectorAppToolInputV0{}),
		))
		return
	}
	var input MCPArrancarDirectorAppToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPPreviewDirectorAppHTTPV0(w, http.StatusBadRequest, newMCPPreviewDirectorHTTPErrorV0(
			"body",
			code,
			correlationFromArrancarDirectorHTTPV0(r, input),
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPPreviewDirectorAppHTTPV0(
			w,
			http.StatusInternalServerError,
			newMCPPreviewDirectorHTTPExecutorErrorV0(
				err,
				result.Errores,
				correlationFromArrancarDirectorHTTPV0(r, input),
			),
		)
		return
	}
	status := http.StatusOK
	if result.Estado == MCPArrancarDirectorAppEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPPreviewDirectorAppHTTPV0(w, status, result)
}

func newMCPPreviewDirectorHTTPExecutorErrorV0(
	err error,
	publicIssues []MCPValidationIssueV0,
	correlationID string,
) MCPPreviewDirectorAppToolResultV0 {
	field, message := mcpArrancarDirectorHTTPIssueFromPublicIssuesV0(publicIssues)
	if field == "" && message == "" {
		field, message = mcpArrancarDirectorHTTPIssueFromErrorV0(err)
	}
	if field == "" {
		field = "executor"
	}
	if message == "" {
		message = "preview_director_error"
	}
	return newMCPPreviewDirectorHTTPErrorV0(field, message, correlationID)
}

func newMCPPreviewDirectorHTTPErrorV0(
	field string,
	message string,
	correlationID string,
) MCPPreviewDirectorAppToolResultV0 {
	return MCPPreviewDirectorAppToolResultV0{
		Estado:        MCPArrancarDirectorAppEstadoErrorV0,
		CorrelationID: strings.TrimSpace(correlationID),
		Errores: []MCPValidationIssueV0{{
			Code:    "preview_director_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func writeMCPPreviewDirectorAppHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPPreviewDirectorAppToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
