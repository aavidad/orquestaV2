package orquestaobservability

import (
	"context"
	"testing"
	"time"
)

func validWorkspaceTimelineQueryV0() WorkspaceTimelineQueryV0 {
	return WorkspaceTimelineQueryV0{
		SchemaVersion: WorkspaceTimelineQuerySchemaVersionV0,
		RequestID:     "request-ref-workspace-timeline-001",
		CorrelationID: "corr-workspace-timeline-001",
		Consumer: OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: OperationalStatusConsumerWebChannelV0,
		},
		Locale:     "es-ES",
		Scope:      WorkspaceTimelineScopeWorkspaceV0,
		AgentRef:   "agent-ref-codex-001",
		ProjectRef: "project-ref-orquesta-001",
		TaskRef:    "task-ref-001",
		TimeWindow: OperationalStatusTimeWindowV0{Preset: "last_hour"},
		Page:       WorkspaceTimelinePageRequestV0{Limit: 10},
		Sources: []string{
			WorkspaceTimelineSourceEventsV0,
			WorkspaceTimelineSourceAuditV0,
			WorkspaceTimelineSourceRunQueueV0,
			WorkspaceTimelineSourceRuntimeProgressV0,
			WorkspaceTimelineSourceGitStatsV0,
			WorkspaceTimelineSourceUsageCostV0,
		},
	}
}

func TestWorkspaceTimelineQueryV0ValidaScopeWorkspaceYSources(t *testing.T) {
	query := validWorkspaceTimelineQueryV0()
	if err := ValidateWorkspaceTimelineQueryV0(query); err != nil {
		t.Fatalf("query valida: %v", err)
	}
	query.Scope = "proyecto"
	if !HasOperationalStatusIssueV0(ValidateWorkspaceTimelineQueryV0(query), ErrScopeNoSoportadoV0) {
		t.Fatalf("scope no detectado")
	}
	query = validWorkspaceTimelineQueryV0()
	query.Sources = []string{WorkspaceTimelineSourceEventsV0, WorkspaceTimelineSourceEventsV0}
	if !HasOperationalStatusIssueV0(ValidateWorkspaceTimelineQueryV0(query), ErrWorkspaceTimelineQueryInvalidaV0) {
		t.Fatalf("source duplicada no detectada")
	}
}

func TestWorkspaceTimelineUnavailableReaderV0DeclaraSourcesNotAvailable(t *testing.T) {
	query := validWorkspaceTimelineQueryV0()
	reader := NewWorkspaceTimelineUnavailableReaderV0(func() time.Time {
		return time.Date(2026, 5, 25, 16, 30, 0, 0, time.UTC)
	})
	timeline, err := reader.QueryWorkspaceTimelineV0(context.Background(), query)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(timeline.Items) != 0 || len(timeline.Sources) != len(query.Sources) {
		t.Fatalf("timeline inesperada: %+v", timeline)
	}
	for _, source := range timeline.Sources {
		if source.Status != WorkspaceTimelineSourceNotAvailableV0 ||
			source.ReasonCode != "source_port_not_bound" {
			t.Fatalf("source no declarada como ausente: %+v", source)
		}
	}
	if !ValidateWorkspaceTimelinePrivacyOKForTestV0(timeline.Privacy) {
		t.Fatalf("privacy no compacta: %+v", timeline.Privacy)
	}
}

func ValidateWorkspaceTimelinePrivacyOKForTestV0(privacy DiagnosticoPrivacyV0) bool {
	return !privacy.ContainsSecret &&
		!privacy.ContainsTranscript &&
		!privacy.ContainsPrompt &&
		!privacy.ContainsCompletion &&
		!privacy.ContainsConnectionDetail
}
