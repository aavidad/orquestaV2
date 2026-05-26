package orquestaweb

import (
	"context"
	"net/http"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestRESTWorkspaceTimelineClientV0ConsumeMismoEndpointMCP(t *testing.T) {
	source, err := orquestaobservability.NewWorkspaceTimelineMemoryAdapterV0(
		[]orquestaobservability.WorkspaceTimelineV0{validWebWorkspaceTimelineForClientV0()},
	)
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	server := newWebHTTPTestServerV0(t, orquestamcp.NewMCPWorkspaceTimelineHTTPHandlerV0(source))
	defer server.Close()
	client := NewRESTWorkspaceTimelineClientV0(server.URL, time.Second)
	view, err := client.ConsultarWorkspaceTimeline(context.Background(), validWebWorkspaceTimelineQueryForClientV0())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if view.SchemaVersion != WebWorkspaceTimelineSchemaV0 ||
		len(view.Items) != 1 ||
		view.Items[0].AgentRef != "agent-ref-web-client-001" ||
		!view.PrivacyOK {
		t.Fatalf("view: %+v", view)
	}
}

func TestRESTWorkspaceTimelineClientV0AceptaContextNil(t *testing.T) {
	source, err := orquestaobservability.NewWorkspaceTimelineMemoryAdapterV0(
		[]orquestaobservability.WorkspaceTimelineV0{validWebWorkspaceTimelineForClientV0()},
	)
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	server := newWebHTTPTestServerV0(t, orquestamcp.NewMCPWorkspaceTimelineHTTPHandlerV0(source))
	defer server.Close()

	client := NewRESTWorkspaceTimelineClientV0(server.URL, time.Second)
	view, err := client.ConsultarWorkspaceTimeline(nil, validWebWorkspaceTimelineQueryForClientV0())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if view.SchemaVersion != WebWorkspaceTimelineSchemaV0 {
		t.Fatalf("view=%+v", view)
	}
}

func TestRESTWorkspaceTimelineClientV0ClasificaErrorHTTP(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := NewRESTWorkspaceTimelineClientV0(server.URL, time.Second)
	_, err := client.ConsultarWorkspaceTimeline(context.Background(), validWebWorkspaceTimelineQueryForClientV0())
	if !IsWebWorkspaceTimelineClientErrorCodeV0(err, WebWorkspaceTimelineErrTransporteV0) {
		t.Fatalf("err=%v", err)
	}
}

func validWebWorkspaceTimelineQueryForClientV0() orquestaobservability.WorkspaceTimelineQueryV0 {
	return NewWebWorkspaceTimelineQueryV0(WebWorkspaceTimelineQueryInputV0{
		RequestID:     "request-ref-web-client-timeline-001",
		CorrelationID: "corr-web-client-timeline-001",
		AgentRef:      "agent-ref-web-client-001",
		Preset:        "last_hour",
		Limit:         5,
		Sources:       []string{orquestaobservability.WorkspaceTimelineSourceRunQueueV0},
	})
}

func validWebWorkspaceTimelineForClientV0() orquestaobservability.WorkspaceTimelineV0 {
	return orquestaobservability.WorkspaceTimelineV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineSchemaVersionV0,
		TimelineID:    "timeline-ref-web-client-001",
		GeneratedAt:   "2026-05-25T10:00:00Z",
		CorrelationID: "corr-web-client-timeline-001",
		Scope:         orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		AgentRef:      "agent-ref-web-client-001",
		Filters: orquestaobservability.WorkspaceTimelineFiltersV0{
			AgentRef: "agent-ref-web-client-001",
		},
		TimeWindow: orquestaobservability.OperationalStatusTimeWindowV0{Preset: "last_hour"},
		Page:       orquestaobservability.WorkspaceTimelinePageV0{Limit: 5},
		Sources: []orquestaobservability.WorkspaceTimelineSourceStatusV0{{
			Source: orquestaobservability.WorkspaceTimelineSourceRunQueueV0,
			Status: orquestaobservability.WorkspaceTimelineSourceAvailableV0,
		}},
		Items: []orquestaobservability.WorkspaceTimelineItemV0{{
			EventRef:   "event-ref-web-client-timeline-001",
			OccurredAt: "2026-05-25T10:00:00Z",
			Source:     orquestaobservability.WorkspaceTimelineSourceRunQueueV0,
			Category:   orquestaobservability.WorkspaceTimelineCategoryQueueV0,
			SummaryKey: "workspace.timeline.run_queue",
			Summary:    "Run visible en cola.",
			Refs: orquestaobservability.WorkspaceTimelineRefsV0{
				AgentRef: "agent-ref-web-client-001",
			},
			Transcript: orquestaobservability.WorkspaceTranscriptClassificationV0{
				Class:      orquestaobservability.WorkspaceTimelineTranscriptNoneV0,
				Confidence: 1,
				Compact:    true,
			},
		}},
		Privacy: orquestaobservability.DiagnosticoPrivacyV0{},
	}
}
