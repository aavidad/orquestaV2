package orquestastatefile

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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

func TestStoreV0AppendRunEventsV0RechazaPayloadGrandeAntesDePersistir(t *testing.T) {
	ctx := context.Background()
	store := mustStoreV0(t, t.TempDir())
	event := orquestacoreworkflow.OrchestrationEventV0{
		EventID:        "evt-state-file-large-payload-001",
		EventType:      orquestacoreworkflow.OrchestrationEventRunStartedV0,
		RunID:          "run-state-file-event-sink-001",
		Sequence:       1,
		OccurredAt:     "2026-05-26T10:00:00Z",
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        json.RawMessage(strings.Repeat("{", 4097)),
	}

	if err := store.AppendRunEventsV0(ctx, event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err == nil {
		t.Fatal("expected payload budget error")
	}
	got, err := store.LoadRunEventsV0(ctx, event.RunID)
	if err != nil {
		t.Fatalf("load events after rejected append: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("events len=%d, want 0", len(got))
	}
}

func TestStoreV0AppendRunEventsV0UsaIndiceDurableYRegistros(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	event := mustStateFileRunStartedEventV0(t, "evt-state-file-index-001", "project-ref-state-file-001")

	if err := store.AppendRunEventsV0(ctx, event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := os.Stat(store.eventIndexPathV0(event.RunID)); err != nil {
		t.Fatalf("event index missing: %v", err)
	}
	page, err := store.LoadRunEventsPageV0(ctx, orquestacionnucleoapp.RunEventPageRequestV0{
		RunRef: event.RunID,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("load page: %v", err)
	}
	if len(page.Events) != 1 || page.Events[0].EventID != event.EventID {
		t.Fatalf("page events=%v, want indexed event", page.Events)
	}
}

func TestStoreV0AppendRunEventsV0AplicaPresupuestoPorAppendYRun(t *testing.T) {
	ctx := context.Background()
	store, err := NewStoreV0(ConfigV0{RootDir: t.TempDir(), MaxAppendEvents: 1, MaxRunEvents: 1})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	first := mustStateFileRunStartedEventV0(t, "evt-state-file-budget-001", "project-ref-state-file-001")
	second := mustStateFileRunStartedEventV0(t, "evt-state-file-budget-002", "project-ref-state-file-002")
	if err := store.AppendRunEventsV0(ctx, first.RunID, []orquestacoreworkflow.OrchestrationEventV0{first, second}); err == nil {
		t.Fatal("expected append budget error")
	}
	if err := store.AppendRunEventsV0(ctx, first.RunID, []orquestacoreworkflow.OrchestrationEventV0{first}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, second.RunID, []orquestacoreworkflow.OrchestrationEventV0{second}); err == nil {
		t.Fatal("expected run budget error")
	}
}

func TestStoreV0AppendRunEventsV0AplicaPresupuestoDeIndice(t *testing.T) {
	ctx := context.Background()
	store, err := NewStoreV0(ConfigV0{RootDir: t.TempDir(), MaxEventSnapshotBytes: 64})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	event := mustStateFileRunStartedEventV0(t, "evt-state-file-index-budget-001", "project-ref-state-file-001")
	if err := store.AppendRunEventsV0(ctx, event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err == nil {
		t.Fatal("expected index budget error")
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
