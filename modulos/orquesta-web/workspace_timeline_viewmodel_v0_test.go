package orquestaweb

import (
	"testing"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestNewWebWorkspaceTimelineQueryV0DefaultsWorkspaceYSources(t *testing.T) {
	query := NewWebWorkspaceTimelineQueryV0(WebWorkspaceTimelineQueryInputV0{
		RequestID:     "request-ref-web-workspace-timeline-001",
		CorrelationID: "corr-web-workspace-timeline-001",
		Preset:        "last_hour",
	})
	if query.Scope != orquestaobservability.WorkspaceTimelineScopeWorkspaceV0 ||
		query.Consumer.Channel != orquestaobservability.OperationalStatusConsumerWebChannelV0 ||
		len(query.Sources) != 6 ||
		query.Page.Limit != 20 {
		t.Fatalf("query inesperada: %+v", query)
	}
	if err := orquestaobservability.ValidateWorkspaceTimelineQueryV0(query); err != nil {
		t.Fatalf("query invalida: %v", err)
	}
}

func TestNewWebWorkspaceTimelineV0ProyectaCompacto(t *testing.T) {
	timeline := orquestaobservability.WorkspaceTimelineV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineSchemaVersionV0,
		TimelineID:    "timeline-ref-web-workspace-001",
		GeneratedAt:   "2026-05-25T16:45:00Z",
		CorrelationID: "corr-web-workspace-timeline-001",
		Scope:         orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		Filters: orquestaobservability.WorkspaceTimelineFiltersV0{
			AgentRef: "agent-ref-web-001",
		},
		TimeWindow: orquestaobservability.OperationalStatusTimeWindowV0{Preset: "last_hour"},
		Page:       orquestaobservability.WorkspaceTimelinePageV0{Limit: 10},
		Sources: []orquestaobservability.WorkspaceTimelineSourceStatusV0{{
			Source:     orquestaobservability.WorkspaceTimelineSourceAuditV0,
			Status:     orquestaobservability.WorkspaceTimelineSourceNotAvailableV0,
			ReasonCode: "source_port_not_bound",
		}},
		Items: []orquestaobservability.WorkspaceTimelineItemV0{{
			EventRef:   "event-ref-web-workspace-001",
			OccurredAt: "2026-05-25T16:44:00Z",
			Source:     orquestaobservability.WorkspaceTimelineSourceAuditV0,
			Category:   orquestaobservability.WorkspaceTimelineCategoryRuntimeV0,
			SummaryKey: "workspace.timeline.audit",
			Summary:    "Evento auditado compacto.",
			Refs: orquestaobservability.WorkspaceTimelineRefsV0{
				AgentRef: "agent-ref-web-001",
			},
			Transcript: orquestaobservability.WorkspaceTranscriptClassificationV0{
				Class:      "metadata_only",
				Confidence: 1,
				Compact:    true,
			},
		}},
		Privacy: orquestaobservability.DiagnosticoPrivacyV0{},
	}
	view := NewWebWorkspaceTimelineV0("es-ES", timeline)
	if view.SchemaVersion != WebWorkspaceTimelineSchemaV0 ||
		len(view.Items) != 1 ||
		!view.PrivacyOK ||
		view.Items[0].TranscriptClassification != "metadata_only" {
		t.Fatalf("view inesperada: %+v", view)
	}
}
