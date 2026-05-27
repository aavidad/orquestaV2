package main

import (
	"encoding/json"
	"net/http"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func newServerWorkspaceTimelineHTTPHandlerV0(
	source orquestaobservability.WorkspaceTimelineSourcePortV0,
) http.Handler {
	return serverWorkspaceTimelineHTTPHandlerV0{source: source}
}

type serverWorkspaceTimelineHTTPHandlerV0 struct {
	source orquestaobservability.WorkspaceTimelineSourcePortV0
}

func (handler serverWorkspaceTimelineHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != orquestamcp.MCPWorkspaceTimelineEndpointV0 {
		writeServerWorkspaceTimelineHTTPErrorV0(w, http.StatusNotFound, "path", "ruta_no_soportada")
		return
	}
	if handleServerPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setServerPublicHTTPAllowV0(w, http.MethodPost)
		writeServerWorkspaceTimelineHTTPErrorV0(w, http.StatusMethodNotAllowed, "method", "metodo_no_permitido")
		return
	}
	if handler.source == nil {
		writeServerWorkspaceTimelineHTTPErrorV0(w, http.StatusServiceUnavailable, "source", "workspace_timeline_no_configurada")
		return
	}
	input, code := decodeServerWorkspaceTimelineHTTPInputV0(w, r)
	if code != "" {
		writeServerWorkspaceTimelineHTTPErrorV0(w, http.StatusBadRequest, "body", code)
		return
	}
	timeline, err := handler.source.QueryWorkspaceTimelineV0(r.Context(), input)
	if err != nil {
		writeServerWorkspaceTimelineHTTPErrorV0(w, http.StatusBadRequest, "query", orquestaobservability.ErrWorkspaceTimelineQueryInvalidaV0)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if timeline.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", timeline.CorrelationID)
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(timeline)
}

func decodeServerWorkspaceTimelineHTTPInputV0(
	w http.ResponseWriter,
	r *http.Request,
) (orquestaobservability.WorkspaceTimelineQueryV0, string) {
	raw, code := decodeServerPublicHTTPJSONRawV0(w, r, serverPublicHTTPJSONProfileControlV0)
	if code != "" {
		return orquestaobservability.WorkspaceTimelineQueryV0{}, code
	}
	normalized, rpcErr := normalizeWorkspaceTimelineArgumentsV0(raw)
	if rpcErr != nil {
		return orquestaobservability.WorkspaceTimelineQueryV0{}, rpcErr.Data["error_code"]
	}
	var input orquestaobservability.WorkspaceTimelineQueryV0
	if err := json.Unmarshal(normalized, &input); err != nil {
		return orquestaobservability.WorkspaceTimelineQueryV0{}, "workspace_timeline_query_invalida"
	}
	return input, ""
}

func writeServerWorkspaceTimelineHTTPErrorV0(
	w http.ResponseWriter,
	status int,
	field string,
	message string,
) {
	result := orquestamcp.MCPWorkspaceTimelineToolResultV0{
		Estado: orquestamcp.MCPWorkspaceTimelineEstadoErrorV0,
		Errores: []orquestamcp.MCPValidationIssueV0{{
			Code:    "workspace_timeline_http_error",
			Field:   field,
			Message: message,
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
