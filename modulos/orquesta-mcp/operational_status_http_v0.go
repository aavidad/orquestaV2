package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const MCPOperationalStatusHTTPPathV0 = "/api/v0/operational-status/query"

func NewMCPOperationalStatusHTTPHandlerV0(
	source orquestaobservability.OperationalStatusQuerySourceV0,
) http.Handler {
	return mcpOperationalStatusHTTPHandlerV0{source: source}
}

type mcpOperationalStatusHTTPHandlerV0 struct {
	source orquestaobservability.OperationalStatusQuerySourceV0
}

func (handler mcpOperationalStatusHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPOperationalStatusHTTPPathV0 {
		writeMCPOperationalStatusIssuesV0(w, http.StatusNotFound, r.Header.Get("X-Correlation-ID"), "path", MCPPublicErrPathUnsupportedV0)
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPOperationalStatusIssuesV0(w, http.StatusMethodNotAllowed, r.Header.Get("X-Correlation-ID"), "method", MCPPublicErrMethodNotAllowedV0)
		return
	}
	if handler.source == nil {
		writeMCPOperationalStatusIssuesV0(w, http.StatusServiceUnavailable, r.Header.Get("X-Correlation-ID"), "source", orquestaobservability.ErrProyeccionNoDisponibleV0)
		return
	}
	var query orquestaobservability.OperationalStatusQueryV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &query); code != "" {
		writeMCPOperationalStatusIssuesV0(w, http.StatusBadRequest, r.Header.Get("X-Correlation-ID"), "body", code)
		return
	}
	diagnostic, err := handler.source.QueryOperationalStatusV0(query)
	if err != nil {
		writeMCPOperationalStatusErrorV0(w, statusForOperationalStatusErrorV0(err), query.CorrelationID, err)
		return
	}
	writeMCPOperationalStatusDiagnosticV0(w, http.StatusOK, diagnostic)
}

func writeMCPOperationalStatusDiagnosticV0(
	w http.ResponseWriter,
	status int,
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if strings.TrimSpace(diagnostic.CorrelationID) != "" {
		w.Header().Set("X-Correlation-ID", strings.TrimSpace(diagnostic.CorrelationID))
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(diagnostic)
}

func writeMCPOperationalStatusErrorV0(w http.ResponseWriter, status int, correlationID string, err error) {
	var issues []orquestaobservability.OperationalStatusValidationIssueV0
	if validation, ok := err.(orquestaobservability.OperationalStatusValidationErrorV0); ok {
		issues = validation.Issues
	}
	if len(issues) == 0 {
		issues = []orquestaobservability.OperationalStatusValidationIssueV0{{
			Code:  orquestaobservability.ErrOperationalStatusQueryInvalidaV0,
			Field: "source",
		}}
	}
	writeMCPOperationalStatusValidationV0(w, status, correlationID, issues)
}

func writeMCPOperationalStatusIssuesV0(w http.ResponseWriter, status int, correlationID string, field string, code string) {
	writeMCPOperationalStatusValidationV0(
		w,
		status,
		correlationID,
		[]orquestaobservability.OperationalStatusValidationIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	)
}

func writeMCPOperationalStatusValidationV0(
	w http.ResponseWriter,
	status int,
	correlationID string,
	issues []orquestaobservability.OperationalStatusValidationIssueV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if strings.TrimSpace(correlationID) != "" {
		w.Header().Set("X-Correlation-ID", strings.TrimSpace(correlationID))
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(orquestaobservability.OperationalStatusValidationErrorV0{Issues: issues})
}

func statusForOperationalStatusErrorV0(err error) int {
	if orquestaobservability.HasOperationalStatusIssueV0(err, orquestaobservability.ErrProyeccionNoDisponibleV0) ||
		orquestaobservability.HasOperationalStatusIssueV0(err, orquestaobservability.ErrDiagnosticoNoDisponibleV0) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}
