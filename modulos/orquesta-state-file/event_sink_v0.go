package orquestastatefile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
		if err := validateStateFileWorkflowEventV0(event); err != nil {
			return err
		}
		if normalizeRefV0(event.RunID) != runRef {
			return fmt.Errorf("orquesta_state_file: event_run_mismatch")
		}
	}
	payloads := make([]json.RawMessage, 0, len(events))
	for _, event := range events {
		payloads = append(payloads, event.Payload)
	}
	if err := store.events.validateAppendV0(payloads); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	index, err := store.loadEventIndexV0(runRef)
	if err != nil {
		return err
	}
	if len(index.Order)+len(events) > store.events.MaxRunEvents {
		return storeErrorV0("events.budget", "events_run_limit_exceeded")
	}
	if err := store.ensureEventAnchorDocumentV0(runRef); err != nil {
		return err
	}
	for _, event := range cloneEventsV0(events) {
		before := len(index.Order)
		if err := store.addEventRecordToIndexV0(&index, event); err != nil {
			return err
		}
		if len(index.Order) == before {
			continue
		}
	}
	return store.writeEventIndexV0(runRef, index)
}

func (store *StoreV0) LoadRunEventsV0(
	ctx context.Context,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	runRef = normalizeRefV0(runRef)
	limit := store.events.MaxRunEvents
	var out []orquestacoreworkflow.OrchestrationEventV0
	cursor := ""
	for {
		remaining := limit - len(out)
		if remaining <= 0 {
			return nil, storeErrorV0("events.budget", "events_full_history_budget_exceeded")
		}
		pageLimit := maxEventLogPageLimitV0
		if remaining < pageLimit {
			pageLimit = remaining
		}
		result, err := store.LoadRunEventsPageV0(ctx, orquestacionnucleoapp.RunEventPageRequestV0{
			RunRef: runRef,
			Limit:  pageLimit,
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		if result.Total > limit || len(out)+len(result.Events) > limit {
			return nil, storeErrorV0("events.budget", "events_full_history_budget_exceeded")
		}
		out = append(out, result.Events...)
		if !result.HasMore {
			break
		}
		if result.NextCursor == "" || result.NextCursor == cursor {
			return nil, storeErrorV0("events.cursor", "cursor invalido")
		}
		cursor = result.NextCursor
	}
	if err := validateLoadedRunEventsV0(runRef, out); err != nil {
		return nil, err
	}
	return cloneEventsV0(out), nil
}

func (store *StoreV0) LoadRunEventsPageV0(
	ctx context.Context,
	request orquestacionnucleoapp.RunEventPageRequestV0,
) (orquestacionnucleoapp.RunEventPageResultV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.RunEventPageResultV0{}, err
	}
	runRef := normalizeRefV0(request.RunRef)
	limit := boundedEventPageLimitV0(request.Limit)
	offset, err := decodeRunEventCursorV0(request.Cursor)
	if err != nil {
		return orquestacionnucleoapp.RunEventPageResultV0{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	index, err := store.loadEventIndexV0(runRef)
	if err != nil {
		if isNotExistV0(err) {
			return orquestacionnucleoapp.RunEventPageResultV0{}, nil
		}
		return orquestacionnucleoapp.RunEventPageResultV0{}, err
	}
	records := sortedIndexRecordsV0(index)
	if offset > len(records) {
		offset = len(records)
	}
	end := offset + limit
	if end > len(records) {
		end = len(records)
	}
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, end-offset)
	for _, recordRef := range records[offset:end] {
		event, err := loadEventRecordV0(store.eventRecordPathV0(runRef, recordRef), runRef, recordRef)
		if err != nil {
			if isNotExistV0(err) {
				return orquestacionnucleoapp.RunEventPageResultV0{}, missingEventRecordErrorV0(recordRef)
			}
			return orquestacionnucleoapp.RunEventPageResultV0{}, err
		}
		events = append(events, event)
	}
	result := orquestacionnucleoapp.RunEventPageResultV0{Events: cloneEventsV0(events), Total: len(records)}
	if end < len(records) {
		result.HasMore = true
		result.NextCursor = encodeRunEventCursorV0(end)
	}
	return result, nil
}

func (store *StoreV0) ensureEventAnchorDocumentV0(runRef string) error {
	path := store.eventsPathV0(runRef)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !isNotExistV0(err) {
		return err
	}
	return writeJSONAtomicV0(path, eventsDocumentV0{
		SchemaVersion: eventDocumentSchemaV0,
		RunRef:        runRef,
	})
}

func validateEventsDocumentV0(document eventsDocumentV0, expectedRunRef string) error {
	if document.SchemaVersion != eventDocumentSchemaV0 {
		return storeErrorV0("events.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef {
		return storeErrorV0("events.ref", "ref inconsistente")
	}
	return validateLoadedRunEventsV0(expectedRunRef, document.Events)
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
