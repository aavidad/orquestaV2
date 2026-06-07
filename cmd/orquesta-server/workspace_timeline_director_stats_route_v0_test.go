package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestBuildServerAppHandlerV0WorkspaceTimelineDirectorStatsDesdeRunRefV0(t *testing.T) {
	stats := &fakeWorkspaceTimelineDirectorStatsV0{observedAt: time.Now().UTC()}
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.NotFoundHandler(),
		MCPTransportBindings: orquestamcp.MCPTransportBindingsV0{
			DirectorStats: stats,
		},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	query := orquestaobservability.WorkspaceTimelineQueryV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineQuerySchemaVersionV0,
		RequestID:     "request-ref-cmd-workspace-timeline-director-stats-001",
		CorrelationID: "corr-cmd-workspace-timeline-director-stats-001",
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0,
		},
		Locale:     "es-ES",
		Scope:      orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		RunRef:     "run-ref-timeline-director-stats-001",
		Page:       orquestaobservability.WorkspaceTimelinePageRequestV0{Limit: 10},
		Sources:    []string{orquestaobservability.WorkspaceTimelineSourceDirectorStatsV0},
		TimeWindow: orquestaobservability.OperationalStatusTimeWindowV0{Preset: "last_hour"},
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
	if timeline.RunRef != query.RunRef ||
		timeline.Sources[0].Source != orquestaobservability.WorkspaceTimelineSourceDirectorStatsV0 ||
		timeline.Sources[0].Status != orquestaobservability.WorkspaceTimelineSourceAvailableV0 ||
		len(timeline.Items) < 3 {
		t.Fatalf("timeline=%+v", timeline)
	}
	if timeline.Items[0].Source != orquestaobservability.WorkspaceTimelineSourceDirectorStatsV0 ||
		timeline.Items[0].Refs.RunRef != query.RunRef ||
		timeline.Items[0].Transcript.Class != orquestaobservability.WorkspaceTimelineTranscriptMetadataV0 {
		t.Fatalf("snapshot=%+v", timeline.Items[0])
	}
	if stats.input.RunRef != query.RunRef ||
		!stats.input.IncludeAgentProgress ||
		!stats.input.IncludeAgentUsage {
		t.Fatalf("input=%+v", stats.input)
	}
}

func TestBuildServerAppHandlerV0WorkspaceTimelineDirectorStatsNoDisponibleV0(t *testing.T) {
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.NotFoundHandler(),
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := []byte(`{
		"schema_version":"workspace_timeline_query.v0",
		"request_id":"request-ref-cmd-workspace-timeline-director-stats-missing",
		"correlation_id":"corr-cmd-workspace-timeline-director-stats-missing",
		"scope":"workspace",
		"run_ref":"run-ref-timeline-director-stats-missing",
		"page":{"limit":5},
		"sources":["director_stats"]
	}`)
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
	if len(timeline.Items) != 0 ||
		len(timeline.Sources) != 1 ||
		timeline.Sources[0].Status != orquestaobservability.WorkspaceTimelineSourceNotAvailableV0 {
		t.Fatalf("timeline=%+v", timeline)
	}
}

type fakeWorkspaceTimelineDirectorStatsV0 struct {
	observedAt time.Time
	input      orquestamcp.MCPDirectorStatsToolInputV0
}

func (stats *fakeWorkspaceTimelineDirectorStatsV0) Execute(
	_ context.Context,
	input orquestamcp.MCPDirectorStatsToolInputV0,
) (orquestamcp.MCPDirectorStatsToolResultV0, error) {
	stats.input = input
	observedAt := stats.observedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	observed := observedAt.Format(time.RFC3339)
	if input.OccurredAt != "" {
		observed = input.OccurredAt
	}
	runRef := input.RunRef
	decisionContext := orquestaobservability.DirectorDecisionContextV0{
		SchemaVersion: orquestaobservability.DirectorDecisionContextSchemaVersionV0,
		RunRef:        runRef,
		ObservedAt:    observed,
		CurrentPhase:  "phase-redaccion",
		Progress: orquestaobservability.DirectorDecisionProgressV0{
			PercentComplete: 50,
			TasksTotal:      2,
			TasksClosed:     1,
			TasksOpen:       1,
			TasksObserved:   2,
			ObservedAgents:  1,
		},
		Lifecycle: orquestaobservability.DirectorDecisionLifecycleV0{
			AgentsRequested: 1,
			AgentsStarted:   1,
			AgentsRunning:   1,
			AgentsInFlight:  1,
		},
		Closure: orquestaobservability.DirectorDecisionClosureV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
			Blocked:     true,
			BlockedBy:   []string{orquestacionnucleoapp.DirectorClosureBlockedByRevisionFinalV0},
			BlockerRefs: []string{"blocker-ref-review-001"},
		},
		ReworkReplan: orquestaobservability.DirectorDecisionReworkReplanV0{
			ReworkRequests:    1,
			ReworkRequestRefs: []string{"rework-ref-001"},
		},
		Tasks: []orquestaobservability.DirectorDecisionTaskV0{{
			TaskRef:        "task-ref-timeline-001",
			Status:         "running",
			AgentRequestID: "agent-ref-timeline-001",
			EvidenceRefs:   []string{"evidence-ref-task-001"},
		}},
		Agents: []orquestaobservability.DirectorDecisionAgentV0{{
			AgentRequestID: "agent-ref-timeline-001",
			Status:         "running",
			Running:        true,
			InFlight:       true,
		}},
		Activity: []orquestaobservability.DirectorDecisionActivityV0{
			{
				ActivityRef: "activity-ref-director-phase-001",
				Kind:        orquestaobservability.DirectorDecisionActivityPhaseCurrentV0,
				OccurredAt:  observed,
				SourceRef:   runRef,
				SummaryKey:  "director.decision_context.activity.phase_current",
			},
			{
				ActivityRef:    "activity-ref-director-task-001",
				Kind:           orquestaobservability.DirectorDecisionActivityTaskProgressV0,
				OccurredAt:     observed,
				SourceRef:      "task-ref-timeline-001",
				AgentRequestID: "agent-ref-timeline-001",
				TaskRef:        "task-ref-timeline-001",
				SummaryKey:     "director.decision_context.activity.task_progress",
			},
		},
		Privacy: orquestaobservability.NewDiagnosticoPrivacyMetadataOnlyV0(),
	}
	return orquestamcp.MCPDirectorStatsToolResultV0{
		Estado: orquestamcp.MCPDirectorStatsEstadoOKV0,
		RunRef: runRef,
		Stats: &orquestacionnucleoapp.DirectorRunStatsV0{
			SchemaVersion: orquestacionnucleoapp.DirectorRunStatsSchemaVersionV0,
			RunRef:        runRef,
			ProjectRef:    "project-ref-timeline-director",
			Status:        "running",
			CurrentPhase:  "phase-redaccion",
			Counts:        orquestacionnucleoapp.DirectorRunStatsCountsV0{TasksTotal: 2, TasksClosed: 1, TasksOpen: 1},
			Closure:       orquestacionnucleoapp.DirectorClosureStatsV0{Status: orquestacionnucleoapp.DirectorClosureStatusBlockedV0, Blocked: true},
		},
		DecisionContext: &decisionContext,
	}, nil
}
