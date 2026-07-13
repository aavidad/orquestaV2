package orquestastatefile

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestStoreV0LoadRunV0RechazaProyeccionInvalida(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	run := validRunV0()
	run.Status = orquestacoreworkflow.OrchestrationRunStatusV0("estado-roto")
	if err := writeJSONAtomicV0(store.runPathV0(run.RunID), runDocumentV0{
		SchemaVersion: runDocumentSchemaV0,
		RunRef:        run.RunID,
		Run:           run,
	}); err != nil {
		t.Fatalf("write corrupt run: %v", err)
	}

	if _, err := store.LoadRunV0(context.Background(), run.RunID); err == nil {
		t.Fatal("expected invalid run projection error")
	}
}

func TestStoreV0LoadRunEventsV0RechazaEventoInvalido(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	event := mustRunStartedEventV0(t, "run-state-file-load-validation-001")
	event.EventType = "EventoInexistente"
	if err := writeJSONAtomicV0(store.eventsPathV0(event.RunID), eventsDocumentV0{
		SchemaVersion: eventDocumentSchemaV0,
		RunRef:        event.RunID,
		Events:        []orquestacoreworkflow.OrchestrationEventV0{event},
	}); err != nil {
		t.Fatalf("write corrupt events: %v", err)
	}

	if _, err := store.LoadRunEventsV0(context.Background(), event.RunID); err == nil {
		t.Fatal("expected invalid event projection error")
	}
}

func TestStoreV0LoadRunEventsV0RechazaEventosDuplicados(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	event := mustRunStartedEventV0(t, "run-state-file-load-validation-002")
	if err := writeJSONAtomicV0(store.eventsPathV0(event.RunID), eventsDocumentV0{
		SchemaVersion: eventDocumentSchemaV0,
		RunRef:        event.RunID,
		Events: []orquestacoreworkflow.OrchestrationEventV0{
			event,
			event,
		},
	}); err != nil {
		t.Fatalf("write duplicate events: %v", err)
	}

	if _, err := store.LoadRunEventsV0(context.Background(), event.RunID); err == nil {
		t.Fatal("expected duplicate event error")
	}
}

func TestStoreV0LoadRunEventsV0RechazaRecordConRunInconsistente(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	event := mustRunStartedEventV0(t, "run-state-file-load-validation-003")
	if err := store.AppendRunEventsV0(context.Background(), event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	corrupt := event
	corrupt.RunID = "run-state-file-load-validation-otro"
	recordRef := eventRecordRefV0(event, eventPayloadHashV0(compactRawMessageV0(event.Payload)))
	if err := writeJSONAtomicV0(store.eventRecordPathV0(event.RunID, recordRef), eventRecordDocumentV0{
		SchemaVersion: eventRecordDocumentSchemaV0,
		RunRef:        event.RunID,
		RecordRef:     recordRef,
		Event:         corrupt,
	}); err != nil {
		t.Fatalf("write corrupt record: %v", err)
	}

	if _, err := store.LoadRunEventsV0(context.Background(), event.RunID); err == nil {
		t.Fatal("expected inconsistent event ref error")
	}
}

func TestStoreV0LoadRunEventsV0RechazaGapDeSecuenciaTrasReinicio(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	start := mustRunStartedEventV0(t, "run-state-file-load-validation-gap")
	gap := start
	gap.EventID = "evt-run-started-state-file-gap"
	gap.Sequence = 3
	gap.OccurredAt = "2026-05-12T09:01:00Z"
	if err := writeJSONAtomicV0(store.eventsPathV0(start.RunID), eventsDocumentV0{
		SchemaVersion: eventDocumentSchemaV0,
		RunRef:        start.RunID,
		Events:        []orquestacoreworkflow.OrchestrationEventV0{start, gap},
	}); err != nil {
		t.Fatalf("write sequence gap: %v", err)
	}

	if _, err := store.LoadRunEventsV0(context.Background(), start.RunID); err == nil {
		t.Fatal("expected corrupt sequence error")
	}
}
