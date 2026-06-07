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
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestBuildServerAppHandlerV0WorkspaceTimelineEventsDesdeEventReaderV0(t *testing.T) {
	runRef := "run-ref-timeline-events-001"
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.NotFoundHandler(),
		Stores: orquestaappcodexstack.StoresV0{
			EventSink: fakeWorkspaceTimelineEventReaderV0{events: []orquestacoreworkflow.OrchestrationEventV0{
				{
					EventID:        "event-ref-agent-started-001",
					EventType:      orquestacoreworkflow.OrchestrationEventAgentStartedV0,
					RunID:          runRef,
					Sequence:       1,
					OccurredAt:     time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339),
					PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
					Payload:        json.RawMessage(`{"agent_request_id":"agent-ref-timeline-events-001","task_ref":"task-ref-timeline-events-001","evidence_refs":["evidence-ref-events-001","/home/alberto/raw","client_secret=abc"]}`),
				},
				{
					EventID:        "event-ref-unknown-001",
					EventType:      "ExternalPayloadWithPrompt",
					RunID:          runRef,
					Sequence:       2,
					OccurredAt:     time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339),
					PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
					Payload:        json.RawMessage(`{"summary":"prompt secreto /home/alberto","task_ref":"/home/alberto/task"}`),
				},
			}},
		},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	query := orquestaobservability.WorkspaceTimelineQueryV0{
		SchemaVersion: orquestaobservability.WorkspaceTimelineQuerySchemaVersionV0,
		RequestID:     "request-ref-cmd-workspace-timeline-events-001",
		CorrelationID: "corr-cmd-workspace-timeline-events-001",
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0,
		},
		Locale:     "es-ES",
		Scope:      orquestaobservability.WorkspaceTimelineScopeWorkspaceV0,
		RunRef:     runRef,
		Page:       orquestaobservability.WorkspaceTimelinePageRequestV0{Limit: 10},
		Sources:    []string{orquestaobservability.WorkspaceTimelineSourceEventsV0},
		TimeWindow: orquestaobservability.OperationalStatusTimeWindowV0{Preset: "last_hour"},
	}
	body, _ := json.Marshal(query)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPWorkspaceTimelineEndpointV0, bytes.NewReader(body))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	rawBody := rec.Body.String()
	if strings.Contains(rawBody, "/home/alberto") ||
		strings.Contains(rawBody, "client_secret") ||
		strings.Contains(rawBody, "prompt secreto") {
		t.Fatalf("payload sensible expuesto: %s", rawBody)
	}
	var timeline orquestaobservability.WorkspaceTimelineV0
	if err := json.Unmarshal(rec.Body.Bytes(), &timeline); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(timeline.Items) != 2 ||
		timeline.Sources[0].Status != orquestaobservability.WorkspaceTimelineSourceAvailableV0 {
		t.Fatalf("timeline=%+v", timeline)
	}
	if timeline.Items[0].Category != orquestaobservability.WorkspaceTimelineCategoryRuntimeV0 ||
		timeline.Items[0].Refs.AgentRef != "agent-ref-timeline-events-001" ||
		timeline.Items[0].Refs.TaskRef != "task-ref-timeline-events-001" ||
		len(timeline.Items[0].EvidenceRefs) != 1 {
		t.Fatalf("item conocido=%+v", timeline.Items[0])
	}
	if timeline.Items[1].Category != orquestaobservability.WorkspaceTimelineCategoryEventV0 ||
		timeline.Items[1].SummaryKey != "workspace.timeline.events.generic" ||
		timeline.Items[1].Refs.TaskRef != "" {
		t.Fatalf("item desconocido=%+v", timeline.Items[1])
	}
}

type fakeWorkspaceTimelineEventReaderV0 struct {
	events []orquestacoreworkflow.OrchestrationEventV0
}

func (reader fakeWorkspaceTimelineEventReaderV0) AppendRunEventsV0(
	_ context.Context,
	_ string,
	_ []orquestacoreworkflow.OrchestrationEventV0,
) error {
	return nil
}

func (reader fakeWorkspaceTimelineEventReaderV0) LoadRunEventsV0(
	_ context.Context,
	_ string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	return append([]orquestacoreworkflow.OrchestrationEventV0(nil), reader.events...), nil
}
