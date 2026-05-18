package orquestastatefile

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestStoreV0AppendRunEventsV0EsIdempotentePorEventIDYPayload(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	event := mustStateFileRunStartedEventV0(t, "evt-state-file-idempotent-001", "project-ref-state-file-001")

	if err := store.AppendRunEventsV0(ctx, event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	recovered := mustStoreV0(t, rootDir)
	if err := recovered.AppendRunEventsV0(ctx, event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("append retry: %v", err)
	}

	got, err := recovered.LoadRunEventsV0(ctx, event.RunID)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("events len=%d, want 1", len(got))
	}
}

func TestStoreV0AppendRunEventsV0RechazaMismoEventIDConPayloadDistinto(t *testing.T) {
	ctx := context.Background()
	store := mustStoreV0(t, t.TempDir())
	first := mustStateFileRunStartedEventV0(t, "evt-state-file-conflict-001", "project-ref-state-file-001")
	changed := mustStateFileRunStartedEventV0(t, "evt-state-file-conflict-001", "project-ref-state-file-002")

	if err := store.AppendRunEventsV0(ctx, first.RunID, []orquestacoreworkflow.OrchestrationEventV0{first}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, changed.RunID, []orquestacoreworkflow.OrchestrationEventV0{changed}); err == nil {
		t.Fatal("expected conflict")
	}

	got, err := store.LoadRunEventsV0(ctx, first.RunID)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("events len=%d, want 1", len(got))
	}
}

func mustStateFileRunStartedEventV0(
	t *testing.T,
	eventID string,
	projectRef string,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	event, err := orquestacoreworkflow.NewRunStartedEventV0(
		orquestacoreworkflow.OrchestrationEventMetaV0{
			EventID:    eventID,
			RunID:      "run-state-file-event-sink-001",
			Sequence:   1,
			OccurredAt: "2026-05-17T10:00:00Z",
		},
		orquestacoreworkflow.RunStartedPayloadV0{
			ProjectRef: projectRef,
			AppSpecRef: "appspec-ref-state-file-001",
		},
	)
	if err != nil {
		t.Fatalf("run started event: %v", err)
	}
	return event
}
