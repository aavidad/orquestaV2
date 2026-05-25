package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestBuildServerAppHandlerV0ExponeWorkspaceTimeline(t *testing.T) {
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	query := orquestaobservability.WorkspaceTimelineQueryV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineQuerySchemaVersionV0,
		RequestID:     "request-ref-cmd-workspace-timeline-001",
		CorrelationID: "corr-cmd-workspace-timeline-001",
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0,
		},
		Locale:     "es-ES",
		Scope:      orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		TimeWindow: orquestaobservability.OperationalStatusTimeWindowV0{Preset: "last_hour"},
		Page:       orquestaobservability.WorkspaceTimelinePageRequestV0{Limit: 5},
		Sources:    []string{orquestaobservability.WorkspaceTimelineSourceAuditV0},
	}
	body, _ := json.Marshal(query)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPWorkspaceTimelineEndpointV0, bytes.NewReader(body))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var timeline orquestaobservability.WorkspaceTimelineV0
	if err := json.Unmarshal(rec.Body.Bytes(), &timeline); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if timeline.Sources[0].Status != orquestaobservability.WorkspaceTimelineSourceNotAvailableV0 {
		t.Fatalf("timeline inesperada: %+v", timeline)
	}
}
