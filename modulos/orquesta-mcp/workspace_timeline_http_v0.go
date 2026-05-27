package orquestamcp

import (
	"encoding/json"
	"net/http"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func NewMCPWorkspaceTimelineHTTPHandlerV0(
	source orquestaobservability.WorkspaceTimelineSourcePortV0,
) http.Handler {
	return mcpWorkspaceTimelineHTTPHandlerV0{source: source}
}

type mcpWorkspaceTimelineHTTPHandlerV0 struct {
	source orquestaobservability.WorkspaceTimelineSourcePortV0
}

func (handler mcpWorkspaceTimelineHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPWorkspaceTimelineEndpointV0 {
		writeMCPWorkspaceTimelineErrorHTTPV0(w, r, http.StatusNotFound, "path", MCPPublicErrPathUnsupportedV0)
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPWorkspaceTimelineErrorHTTPV0(w, r, http.StatusMethodNotAllowed, "method", MCPPublicErrMethodNotAllowedV0)
		return
	}
	if handler.source == nil {
		writeMCPWorkspaceTimelineErrorHTTPV0(w, r, http.StatusServiceUnavailable, "source", "workspace_timeline_no_configurada")
		return
	}
	var input orquestaobservability.WorkspaceTimelineQueryV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPWorkspaceTimelineErrorHTTPV0(w, r, http.StatusBadRequest, "body", code)
		return
	}
	timeline, err := handler.source.QueryWorkspaceTimelineV0(r.Context(), input)
	if err != nil {
		writeMCPWorkspaceTimelineErrorHTTPV0(w, r, http.StatusBadRequest, "query", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if timeline.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", timeline.CorrelationID)
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(timeline)
}

func writeMCPWorkspaceTimelineErrorHTTPV0(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	field string,
	message string,
) {
	result := MCPWorkspaceTimelineToolResultV0{
		Estado:        MCPWorkspaceTimelineEstadoErrorV0,
		CorrelationID: r.Header.Get("X-Correlation-ID"),
		Errores: []MCPValidationIssueV0{{
			Code:    "workspace_timeline_http_error",
			Field:   field,
			Message: publicMCPErrorMessageFromTextV0("workspace_timeline_http_error", message),
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
