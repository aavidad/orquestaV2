package orquestamcp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
		writeMCPArrancarDirectorAppHTTPV0(w, http.StatusMethodNotAllowed, newMCPArrancarDirectorHTTPErrorV0(
			"method",
			MCPPublicErrMethodNotAllowedV0,
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
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPArrancarDirectorAppHTTPV0(w, http.StatusBadRequest, newMCPArrancarDirectorHTTPErrorV0(
			"body",
			code,
			correlationFromArrancarDirectorHTTPV0(r, input),
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPArrancarDirectorAppHTTPV0(
			w,
			http.StatusInternalServerError,
			newMCPArrancarDirectorHTTPExecutorErrorV0(
				err,
				result.Errores,
				correlationFromArrancarDirectorHTTPV0(r, input),
			),
		)
		return
	}
	result = NormalizeMCPArrancarDirectorAppResultV0(
		result,
		correlationFromArrancarDirectorHTTPV0(r, input),
	)
	status := http.StatusOK
	if result.Estado == MCPArrancarDirectorAppEstadoErrorV0 {
		status = http.StatusBadRequest
	} else {
		result.EvidenceRefs = WithMCPConfigProjectionVerifiedEvidenceV0(result.EvidenceRefs, input.RequiredSettings)
	}
	writeMCPArrancarDirectorAppHTTPV0(w, status, result)
}

func newMCPArrancarDirectorHTTPExecutorErrorV0(
	err error,
	publicIssues []MCPValidationIssueV0,
	correlationID string,
) MCPArrancarDirectorAppToolResultV0 {
	field, message := mcpArrancarDirectorHTTPIssueFromPublicIssuesV0(publicIssues)
	if field == "" && message == "" {
		field, message = mcpArrancarDirectorHTTPIssueFromErrorV0(err)
	}
	if field == "" {
		field = "executor"
	}
	if message == "" {
		message = "arrancar_director_error"
	}
	return newMCPArrancarDirectorHTTPErrorV0(
		field,
		message,
		correlationID,
	)
}

func mcpArrancarDirectorHTTPIssueFromPublicIssuesV0(
	issues []MCPValidationIssueV0,
) (string, string) {
	for _, issue := range issues {
		field := strings.TrimSpace(issue.Field)
		message := normalizeMCPArrancarDirectorHTTPErrorTextV0(firstNonEmptyMCPV0(issue.Message, issue.Code))
		if field != "" || message != "" {
			return field, message
		}
	}
	return "", ""
}

func mcpArrancarDirectorHTTPIssueFromErrorV0(err error) (string, string) {
	var coreIssue orquestacionnucleoapp.ErrorV0
	if errors.As(err, &coreIssue) {
		return strings.TrimSpace(coreIssue.Field), normalizeMCPArrancarDirectorHTTPErrorTextV0(
			firstNonEmptyMCPV0(coreIssue.Message, coreIssue.Code, err.Error()),
		)
	}
	var directorIssue orquestaappdirectorservice.AppDirectorServiceIssueV0
	if errors.As(err, &directorIssue) {
		return strings.TrimSpace(directorIssue.Field), normalizeMCPArrancarDirectorHTTPErrorTextV0(err.Error())
	}
	message := normalizeMCPArrancarDirectorHTTPErrorTextV0(err.Error())
	return mcpArrancarDirectorHTTPFieldFromErrorTextV0(message), message
}

func mcpArrancarDirectorHTTPFieldFromErrorTextV0(message string) string {
	const marker = "field="
	if idx := strings.Index(message, marker); idx >= 0 {
		value := message[idx+len(marker):]
		if cut := strings.IndexAny(value, " ,;"); cut >= 0 {
			value = value[:cut]
		}
		return strings.TrimSpace(value)
	}
	return ""
}

func normalizeMCPArrancarDirectorHTTPErrorTextV0(value string) string {
	const maxLength = 300
	normalized := strings.Join(strings.Fields(value), " ")
	if len(normalized) <= maxLength {
		return normalized
	}
	return strings.TrimSpace(normalized[:maxLength])
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
