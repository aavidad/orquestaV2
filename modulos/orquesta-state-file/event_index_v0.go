package orquestastatefile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const runEventCursorPrefixV0 = "run_event_cursor_v0:"

type eventIndexDocumentV0 struct {
	SchemaVersion string                        `json:"schema_version"`
	RunRef        string                        `json:"run_ref"`
	Order         []string                      `json:"order"`
	EventsByID    map[string]eventIndexEntryV0  `json:"events_by_id,omitempty"`
	Records       map[string]eventIndexRecordV0 `json:"records,omitempty"`
}

type eventIndexEntryV0 struct {
	RecordRef   string `json:"record_ref"`
	PayloadHash string `json:"payload_hash"`
}

type eventIndexRecordV0 struct {
	EventID     string `json:"event_id,omitempty"`
	PayloadHash string `json:"payload_hash"`
	Sequence    int64  `json:"sequence"`
	OccurredAt  string `json:"occurred_at,omitempty"`
}

type eventRecordDocumentV0 struct {
	SchemaVersion string                                    `json:"schema_version"`
	RunRef        string                                    `json:"run_ref"`
	RecordRef     string                                    `json:"record_ref"`
	Event         orquestacoreworkflow.OrchestrationEventV0 `json:"event"`
}

func (store *StoreV0) loadEventIndexV0(runRef string) (eventIndexDocumentV0, error) {
	index, ok, err := readJSONFileV0[eventIndexDocumentV0](store.eventIndexPathV0(runRef))
	if err != nil {
		return eventIndexDocumentV0{}, err
	}
	if ok {
		return validateEventIndexDocumentV0(index, runRef)
	}
	return store.bootstrapEventIndexV0(runRef)
}

func (store *StoreV0) bootstrapEventIndexV0(runRef string) (eventIndexDocumentV0, error) {
	index := eventIndexDocumentV0{
		SchemaVersion: eventIndexDocumentSchemaV0,
		RunRef:        runRef,
		EventsByID:    map[string]eventIndexEntryV0{},
		Records:       map[string]eventIndexRecordV0{},
	}
	document, ok, err := readJSONFileV0[eventsDocumentV0](store.eventsPathV0(runRef))
	if err != nil {
		return eventIndexDocumentV0{}, err
	}
	if !ok {
		return index, nil
	}
	if err := validateEventsDocumentV0(document, runRef); err != nil {
		return eventIndexDocumentV0{}, err
	}
	for _, event := range document.Events {
		if err := store.addEventRecordToIndexV0(&index, event); err != nil {
			return eventIndexDocumentV0{}, err
		}
	}
	return index, store.writeEventIndexV0(runRef, index)
}

func validateEventIndexDocumentV0(
	index eventIndexDocumentV0,
	runRef string,
) (eventIndexDocumentV0, error) {
	if index.SchemaVersion != eventIndexDocumentSchemaV0 {
		return eventIndexDocumentV0{}, storeErrorV0("events.index.schema_version", "schema_version invalida")
	}
	if index.RunRef != runRef {
		return eventIndexDocumentV0{}, storeErrorV0("events.index.ref", "ref inconsistente")
	}
	if index.EventsByID == nil {
		index.EventsByID = map[string]eventIndexEntryV0{}
	}
	if index.Records == nil {
		index.Records = map[string]eventIndexRecordV0{}
	}
	return index, nil
}

func (store *StoreV0) addEventRecordToIndexV0(
	index *eventIndexDocumentV0,
	event orquestacoreworkflow.OrchestrationEventV0,
) error {
	event.Payload = compactRawMessageV0(event.Payload)
	payloadHash := eventPayloadHashV0(event.Payload)
	eventID := normalizeRefV0(event.EventID)
	if entry, ok := index.EventsByID[eventID]; ok && eventID != "" {
		if entry.PayloadHash == payloadHash {
			return nil
		}
		return storeErrorV0("events.idempotency", "event_id conflictivo")
	}
	recordRef := eventRecordRefV0(event, payloadHash)
	if _, ok := index.Records[recordRef]; ok {
		return nil
	}
	if len(index.Order)+1 > store.events.MaxRunEvents {
		return storeErrorV0("events.budget", "events_run_limit_exceeded")
	}
	next := cloneEventIndexDocumentV0(*index)
	next.Order = append(next.Order, recordRef)
	next.Records[recordRef] = eventIndexRecordV0{
		EventID:     eventID,
		PayloadHash: payloadHash,
		Sequence:    event.Sequence,
		OccurredAt:  normalizeRefV0(event.OccurredAt),
	}
	if eventID != "" {
		next.EventsByID[eventID] = eventIndexEntryV0{RecordRef: recordRef, PayloadHash: payloadHash}
	}
	if err := store.validateEventIndexSizeV0(next); err != nil {
		return err
	}
	if err := writeJSONAtomicV0(store.eventRecordPathV0(index.RunRef, recordRef), eventRecordDocumentV0{
		SchemaVersion: eventRecordDocumentSchemaV0,
		RunRef:        index.RunRef,
		RecordRef:     recordRef,
		Event:         event,
	}); err != nil {
		return err
	}
	*index = next
	return nil
}

func loadEventRecordV0(path string, runRef string, recordRef string) (orquestacoreworkflow.OrchestrationEventV0, error) {
	document, ok, err := readJSONFileV0[eventRecordDocumentV0](path)
	if err != nil || !ok {
		return orquestacoreworkflow.OrchestrationEventV0{}, err
	}
	if document.SchemaVersion != eventRecordDocumentSchemaV0 ||
		document.RunRef != runRef ||
		document.RecordRef != recordRef {
		return orquestacoreworkflow.OrchestrationEventV0{}, storeErrorV0("events.record.ref", "ref inconsistente")
	}
	if normalizeRefV0(document.Event.RunID) != runRef {
		return orquestacoreworkflow.OrchestrationEventV0{}, storeErrorV0("events.record.ref", "event_ref_inconsistent")
	}
	if err := validateStateFileWorkflowEventV0(document.Event); err != nil {
		return orquestacoreworkflow.OrchestrationEventV0{}, err
	}
	return document.Event, nil
}

func decodeRunEventCursorV0(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	if !strings.HasPrefix(cursor, runEventCursorPrefixV0) {
		return 0, storeErrorV0("events.cursor", "cursor invalido")
	}
	offset, err := strconv.Atoi(strings.TrimPrefix(cursor, runEventCursorPrefixV0))
	if err != nil || offset < 0 {
		return 0, storeErrorV0("events.cursor", "cursor invalido")
	}
	return offset, nil
}

func encodeRunEventCursorV0(offset int) string {
	return runEventCursorPrefixV0 + strconv.Itoa(offset)
}

func eventRecordRefV0(event orquestacoreworkflow.OrchestrationEventV0, payloadHash string) string {
	base := strings.Join([]string{
		normalizeRefV0(event.RunID),
		normalizeRefV0(event.EventID),
		strconv.FormatInt(event.Sequence, 10),
		payloadHash,
	}, "|")
	sum := sha256.Sum256([]byte(base))
	return "event-record-ref-" + hex.EncodeToString(sum[:])[:32]
}

func eventPayloadHashV0(payload json.RawMessage) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func sortedIndexRecordsV0(index eventIndexDocumentV0) []string {
	records := append([]string(nil), index.Order...)
	if len(records) == 0 && len(index.Records) > 0 {
		for recordRef := range index.Records {
			records = append(records, recordRef)
		}
		sort.SliceStable(records, func(i, j int) bool {
			return index.Records[records[i]].Sequence < index.Records[records[j]].Sequence
		})
	}
	return records
}

func isNotExistV0(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func missingEventRecordErrorV0(recordRef string) error {
	if strings.TrimSpace(recordRef) == "" {
		return storeErrorV0("events.record", "event_record_missing")
	}
	return storeErrorV0("events.record", fmt.Sprintf("event_record_missing:%s", recordRef))
}

func (store *StoreV0) writeEventIndexV0(runRef string, index eventIndexDocumentV0) error {
	if err := store.validateEventIndexSizeV0(index); err != nil {
		return err
	}
	return writeJSONAtomicV0(store.eventIndexPathV0(runRef), index)
}

func (store *StoreV0) validateEventIndexSizeV0(index eventIndexDocumentV0) error {
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if int64(len(data)) > store.events.MaxEventSnapshotBytes {
		return storeErrorV0("events.budget", "events_index_snapshot_limit_exceeded")
	}
	return nil
}

func cloneEventIndexDocumentV0(index eventIndexDocumentV0) eventIndexDocumentV0 {
	out := index
	out.Order = append([]string(nil), index.Order...)
	out.EventsByID = make(map[string]eventIndexEntryV0, len(index.EventsByID))
	for key, value := range index.EventsByID {
		out.EventsByID[key] = value
	}
	out.Records = make(map[string]eventIndexRecordV0, len(index.Records))
	for key, value := range index.Records {
		out.Records[key] = value
	}
	return out
}
