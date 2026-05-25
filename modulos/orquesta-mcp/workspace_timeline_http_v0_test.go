package orquestamcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestMCPWorkspaceTimelineHTTPHandlerV0UsaPuertoCompartido(t *testing.T) {
	query := mcpWorkspaceTimelineHTTPQueryTestV0()
	body, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPWorkspaceTimelineEndpointV0, bytes.NewReader(body))
	reader := orquestaobservability.NewWorkspaceTimelineUnavailableReaderV0(func() time.Time {
		return time.Date(2026, 5, 25, 16, 40, 0, 0, time.UTC)
	})
	NewMCPWorkspaceTimelineHTTPHandlerV0(reader).ServeHTTP(rec, req)
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

func TestMCPWorkspaceTimelineHTTPHandlerV0RechazaMetodoYPuertoNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPWorkspaceTimelineEndpointV0, nil)
	rec := httptest.NewRecorder()
	reader := orquestaobservability.NewWorkspaceTimelineUnavailableReaderV0(time.Now)
	NewMCPWorkspaceTimelineHTTPHandlerV0(reader).ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status=%d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, MCPWorkspaceTimelineEndpointV0, bytes.NewBufferString(`{}`))
	rec = httptest.NewRecorder()
	NewMCPWorkspaceTimelineHTTPHandlerV0(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil status=%d", rec.Code)
	}
}

func mcpWorkspaceTimelineHTTPQueryTestV0() orquestaobservability.WorkspaceTimelineQueryV0 {
	return orquestaobservability.WorkspaceTimelineQueryV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineQuerySchemaVersionV0,
		RequestID:     "request-ref-workspace-timeline-http-001",
		CorrelationID: "corr-workspace-timeline-http-001",
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-mcp",
			Channel: orquestaobservability.OperationalStatusConsumerMCPChannelV0,
		},
		Locale:     "es-ES",
		Scope:      orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		TimeWindow: orquestaobservability.OperationalStatusTimeWindowV0{Preset: "last_hour"},
		Page:       orquestaobservability.WorkspaceTimelinePageRequestV0{Limit: 5},
		Sources:    []string{orquestaobservability.WorkspaceTimelineSourceAuditV0},
	}
}
