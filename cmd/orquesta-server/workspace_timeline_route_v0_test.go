package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestBuildServerAppHandlerV0WorkspaceTimelineAPIAceptaLast30mComoStringV0(t *testing.T) {
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.NotFoundHandler(),
		MCPTransportBindings: orquestamcp.MCPTransportBindingsV0{
			RunQueuePriority: fakeWorkspaceTimelineQueueV0{now: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := []byte(`{
		"schema_version":"workspace_timeline_query.v0",
		"request_id":"request-ref-cmd-workspace-timeline-last30m",
		"correlation_id":"corr-cmd-workspace-timeline-last30m",
		"scope":"workspace",
		"time_window":"last_30m",
		"page":{"limit":10},
		"sources":["run_queue"]
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
	if timeline.TimeWindow.Preset != "last_30m" ||
		timeline.Counters["tasks_closed"] != 1 ||
		timeline.Counters["events"] != 2 {
		t.Fatalf("timeline=%+v", timeline)
	}
}

func TestBuildServerAppHandlerV0WorkspaceTimelineAPIErrorPublicoSaneadoV0(t *testing.T) {
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.NotFoundHandler(),
		MCPTransportBindings: orquestamcp.MCPTransportBindingsV0{
			RunQueuePriority: fakeWorkspaceTimelineQueueV0{now: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := []byte(`{
		"schema_version":"workspace_timeline_query.v0",
		"request_id":"request-ref-cmd-workspace-timeline-invalid",
		"correlation_id":"corr-cmd-workspace-timeline-invalid",
		"task_ref":"/home/alberto/prompts/raw.txt",
		"scope":"workspace",
		"time_window":123,
		"page":{"limit":10},
		"sources":["run_queue"]
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPWorkspaceTimelineEndpointV0, bytes.NewReader(body))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest ||
		strings.Contains(rec.Body.String(), "/home/alberto") ||
		!strings.Contains(rec.Body.String(), "workspace_timeline_time_window_invalid") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBuildServerAppHandlerV0WorkspaceTimelineAPIAcotaBodyComoControlPlaneV0(t *testing.T) {
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.NotFoundHandler(),
		MCPTransportBindings: orquestamcp.MCPTransportBindingsV0{
			RunQueuePriority: fakeWorkspaceTimelineQueueV0{now: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	body := `{"schema_version":"workspace_timeline_query.v0","request_id":"request-ref-cmd-workspace-timeline-large","correlation_id":"corr-cmd-workspace-timeline-large","scope":"workspace","time_window":"last_30m","page":{"limit":10},"sources":["run_queue"],"ignored":"` +
		strings.Repeat("a", int(serverPublicHTTPJSONControlMaxBytesV0)+1) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPWorkspaceTimelineEndpointV0, strings.NewReader(body))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "request_body_too_large") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeWorkspaceTimelineQueueV0 struct {
	now time.Time
}

func (queue fakeWorkspaceTimelineQueueV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunQueuePriorityToolInputV0,
) (orquestamcp.MCPRunQueuePriorityToolResultV0, error) {
	now := queue.now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return orquestamcp.MCPRunQueuePriorityToolResultV0{
		Estado:        orquestamcp.MCPRunQueuePriorityEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Action:        orquestamcp.MCPRunQueuePriorityActionRankV0,
		Count:         3,
		Ranked: []orquestamcp.MCPRunQueueRankedCandidateCompactV0{
			{Rank: 1, RunRef: "run-ref-timeline-closed-recent", AppRef: "app-ref-timeline", Status: "closed", PriorityScore: 80, UpdatedAt: now.Add(-10 * time.Minute).Format(time.RFC3339)},
			{Rank: 2, RunRef: "run-ref-timeline-ready-recent", AppRef: "app-ref-timeline", Status: "ready", PriorityScore: 70, UpdatedAt: now.Add(-5 * time.Minute).Format(time.RFC3339)},
			{Rank: 3, RunRef: "run-ref-timeline-closed-old", AppRef: "app-ref-timeline", Status: "closed", PriorityScore: 60, UpdatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339)},
		},
	}, nil
}
