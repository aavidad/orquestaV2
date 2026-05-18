package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestInMemoryEventSinkV0AppendRunEventsEsIdempotentePorEventIDYPayload(t *testing.T) {
	ctx := context.Background()
	sink := NewInMemoryEventSinkV0()
	event := mustMemoryRunStartedEventV0(t, "evt-memory-001", "project-ref-memory-001")

	if err := sink.AppendRunEventsV0(ctx, event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := sink.AppendRunEventsV0(ctx, event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("append retry: %v", err)
	}

	got, err := sink.LoadRunEventsV0(ctx, event.RunID)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("events len=%d, want 1", len(got))
	}
}

func TestInMemoryEventSinkV0AppendRunEventsRechazaMismoEventIDConPayloadDistinto(t *testing.T) {
	ctx := context.Background()
	sink := NewInMemoryEventSinkV0()
	first := mustMemoryRunStartedEventV0(t, "evt-memory-conflict-001", "project-ref-memory-001")
	changed := mustMemoryRunStartedEventV0(t, "evt-memory-conflict-001", "project-ref-memory-002")

	if err := sink.AppendRunEventsV0(ctx, first.RunID, []orquestacoreworkflow.OrchestrationEventV0{first}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := sink.AppendRunEventsV0(ctx, changed.RunID, []orquestacoreworkflow.OrchestrationEventV0{changed}); err == nil {
		t.Fatal("expected conflict")
	}

	got, err := sink.LoadRunEventsV0(ctx, first.RunID)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("events len=%d, want 1", len(got))
	}
}

func mustMemoryRunStartedEventV0(
	t *testing.T,
	eventID string,
	projectRef string,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	event, err := orquestacoreworkflow.NewRunStartedEventV0(
		orquestacoreworkflow.OrchestrationEventMetaV0{
			EventID:    eventID,
			RunID:      "run-memory-event-sink-001",
			Sequence:   1,
			OccurredAt: "2026-05-17T10:00:00Z",
		},
		orquestacoreworkflow.RunStartedPayloadV0{
			ProjectRef: projectRef,
			AppSpecRef: "appspec-ref-memory-001",
		},
	)
	if err != nil {
		t.Fatalf("run started event: %v", err)
	}
	event.Payload = cloneRawMessageForMemoryTestV0(t, event.Payload)
	return event
}

func cloneRawMessageForMemoryTestV0(t *testing.T, payload json.RawMessage) json.RawMessage {
	t.Helper()
	out := append(json.RawMessage(nil), payload...)
	if !json.Valid(out) {
		t.Fatal("invalid event payload")
	}
	return out
}
