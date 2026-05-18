package orquestastatefile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type eventsDocumentV0 struct {
	SchemaVersion string                                      `json:"schema_version"`
	RunRef        string                                      `json:"run_ref"`
	Events        []orquestacoreworkflow.OrchestrationEventV0 `json:"events"`
}

func (store *StoreV0) AppendRunEventsV0(
	ctx context.Context,
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	runRef = normalizeRefV0(runRef)
	for _, event := range events {
		if normalizeRefV0(event.RunID) != runRef {
			return fmt.Errorf("orquesta_state_file: event_run_mismatch")
		}
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[eventsDocumentV0](store.eventsPathV0(runRef))
	if err != nil {
		return err
	}
	if !ok {
		document = eventsDocumentV0{SchemaVersion: eventDocumentSchemaV0, RunRef: runRef}
	}
	if err := validateEventsDocumentV0(document, runRef); err != nil {
		return err
	}
	for _, event := range cloneEventsV0(events) {
		if duplicate, err := eventAlreadyPersistedV0(document.Events, event); err != nil {
			return err
		} else if duplicate {
			continue
		}
		document.Events = append(document.Events, event)
	}
	return writeJSONAtomicV0(store.eventsPathV0(runRef), document)
}

func eventAlreadyPersistedV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	event orquestacoreworkflow.OrchestrationEventV0,
) (bool, error) {
	eventID := normalizeRefV0(event.EventID)
	if eventID == "" {
		return false, nil
	}
	runRef := normalizeRefV0(event.RunID)
	for _, existing := range events {
		if normalizeRefV0(existing.RunID) != runRef || normalizeRefV0(existing.EventID) != eventID {
			continue
		}
		if bytes.Equal(compactRawMessageV0(existing.Payload), event.Payload) {
			return true, nil
		}
		return false, storeErrorV0("events.idempotency", "event_id conflictivo")
	}
	return false, nil
}

func (store *StoreV0) LoadRunEventsV0(
	ctx context.Context,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runRef = normalizeRefV0(runRef)
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[eventsDocumentV0](store.eventsPathV0(runRef))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	if err := validateEventsDocumentV0(document, runRef); err != nil {
		return nil, err
	}
	return cloneEventsV0(document.Events), nil
}

func validateEventsDocumentV0(document eventsDocumentV0, expectedRunRef string) error {
	if document.SchemaVersion != eventDocumentSchemaV0 {
		return storeErrorV0("events.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef {
		return storeErrorV0("events.ref", "ref inconsistente")
	}
	return nil
}

func cloneEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) []orquestacoreworkflow.OrchestrationEventV0 {
	if len(events) == 0 {
		return nil
	}
	out := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(events))
	for _, event := range events {
		event.Payload = compactRawMessageV0(event.Payload)
		event.Payload = append(event.Payload[:0:0], event.Payload...)
		out = append(out, event)
	}
	return out
}

func compactRawMessageV0(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return nil
	}
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, payload); err != nil {
		return append(payload[:0:0], payload...)
	}
	return append(json.RawMessage(nil), buffer.Bytes()...)
}
