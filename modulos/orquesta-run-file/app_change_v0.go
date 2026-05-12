package orquestarunfile

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

const runFileAppChangeSchemaVersionV0 = "orquesta.run_file.app_change.v0"

type runFileAppChangeSnapshotV0 struct {
	SchemaVersion string                                `json:"schema_version"`
	Records       []orquestaappchange.AppChangeRecordV0 `json:"records"`
}

func (store *RunFileStoreV0) SaveAppChangeRequestV0(
	ctx context.Context,
	record orquestaappchange.AppChangeRecordV0,
) error {
	if err := runFileAppChangeContextErrV0(ctx); err != nil {
		return err
	}
	record = normalizeRunFileAppChangeRecordV0(record)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	next := copyRunFileAppChangeRecordsV0(store.appChangeRecords)
	next = saveRunFileAppChangeRecordV0(next, record)
	if err := persistRunFileAppChangeV0(store.appChangePath, next); err != nil {
		return err
	}
	store.appChangeRecords = next
	return nil
}

func (store *RunFileStoreV0) ListAppChangeRecordsV0(
	ctx context.Context,
	filter orquestaappchange.AppChangeRecordFilterV0,
) ([]orquestaappchange.AppChangeRecordV0, error) {
	if err := runFileAppChangeContextErrV0(ctx); err != nil {
		return nil, err
	}
	runRef := strings.TrimSpace(filter.RunRef)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	out := make([]orquestaappchange.AppChangeRecordV0, 0, len(store.appChangeRecords))
	for _, record := range store.appChangeRecords {
		if runRef != "" && record.Request.RunRef != runRef {
			continue
		}
		out = append(out, copyRunFileAppChangeRecordV0(record))
	}
	return out, nil
}

func saveRunFileAppChangeRecordV0(
	records []orquestaappchange.AppChangeRecordV0,
	record orquestaappchange.AppChangeRecordV0,
) []orquestaappchange.AppChangeRecordV0 {
	for index, existing := range records {
		if runFileAppChangeRecordKeyV0(existing) == runFileAppChangeRecordKeyV0(record) {
			records[index] = copyRunFileAppChangeRecordV0(record)
			return records
		}
	}
	return append(records, copyRunFileAppChangeRecordV0(record))
}

func loadRunFileAppChangeV0(
	path string,
) ([]orquestaappchange.AppChangeRecordV0, error) {
	var snapshot runFileAppChangeSnapshotV0
	found, err := readJSONSnapshotV0(path, &snapshot)
	if err != nil {
		return nil, err
	}
	if !found {
		return []orquestaappchange.AppChangeRecordV0{}, nil
	}
	if snapshot.SchemaVersion != runFileAppChangeSchemaVersionV0 {
		return nil, fmt.Errorf("orquesta_run_file: app_change_schema_invalid")
	}
	return copyRunFileAppChangeRecordsV0(snapshot.Records), nil
}

func persistRunFileAppChangeV0(
	path string,
	records []orquestaappchange.AppChangeRecordV0,
) error {
	snapshot := runFileAppChangeSnapshotV0{
		SchemaVersion: runFileAppChangeSchemaVersionV0,
		Records:       copyRunFileAppChangeRecordsV0(records),
	}
	return writeAtomicJSONSnapshotV0(path, snapshot)
}

func copyRunFileAppChangeRecordsV0(
	records []orquestaappchange.AppChangeRecordV0,
) []orquestaappchange.AppChangeRecordV0 {
	out := make([]orquestaappchange.AppChangeRecordV0, 0, len(records))
	for _, record := range records {
		out = append(out, copyRunFileAppChangeRecordV0(record))
	}
	return out
}

func copyRunFileAppChangeRecordV0(
	record orquestaappchange.AppChangeRecordV0,
) orquestaappchange.AppChangeRecordV0 {
	record = normalizeRunFileAppChangeRecordV0(record)
	record.Request.CurrentStateRefs = append([]string(nil), record.Request.CurrentStateRefs...)
	record.Request.Scope = append([]string(nil), record.Request.Scope...)
	record.Request.AcceptanceCriteria = append([]string(nil), record.Request.AcceptanceCriteria...)
	record.Request.Constraints = append([]string(nil), record.Request.Constraints...)
	record.Request.AllowedWriteSet = append([]string(nil), record.Request.AllowedWriteSet...)
	record.Request.MetadataRefs = append([]string(nil), record.Request.MetadataRefs...)
	return record
}

func normalizeRunFileAppChangeRecordV0(
	record orquestaappchange.AppChangeRecordV0,
) orquestaappchange.AppChangeRecordV0 {
	return orquestaappchange.AppChangeRecordV0{
		Request:     normalizeRunFileAppChangeRequestV0(record.Request),
		ReceivedAt:  strings.TrimSpace(record.ReceivedAt),
		RequestedBy: strings.TrimSpace(record.RequestedBy),
	}
}

func normalizeRunFileAppChangeRequestV0(
	request orquestaappchange.AppChangeRequestV0,
) orquestaappchange.AppChangeRequestV0 {
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	if request.SchemaVersion == "" {
		request.SchemaVersion = orquestaappchange.AppChangeRequestSchemaV0
	}
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.AppRef = strings.TrimSpace(request.AppRef)
	request.ChangeRef = strings.TrimSpace(request.ChangeRef)
	request.ActorRef = strings.TrimSpace(request.ActorRef)
	request.Locale = strings.TrimSpace(request.Locale)
	request.UserIntent = strings.TrimSpace(request.UserIntent)
	request.TargetArea = strings.TrimSpace(request.TargetArea)
	request.CurrentStateRefs = compactRunFileStringsOrEmptyV0(request.CurrentStateRefs)
	request.Scope = compactRunFileStringsOrEmptyV0(request.Scope)
	request.AcceptanceCriteria = compactRunFileStringsOrEmptyV0(request.AcceptanceCriteria)
	request.Constraints = compactRunFileStringsOrEmptyV0(request.Constraints)
	request.AllowedWriteSet = compactRunFileStringsOrEmptyV0(request.AllowedWriteSet)
	request.MetadataRefs = compactRunFileStringsOrEmptyV0(request.MetadataRefs)
	if request.RequestID == "" {
		request.RequestID = request.ChangeRef
	}
	if request.CorrelationID == "" {
		request.CorrelationID = request.RequestID
	}
	return request
}

func runFileAppChangeRecordKeyV0(
	record orquestaappchange.AppChangeRecordV0,
) string {
	request := normalizeRunFileAppChangeRequestV0(record.Request)
	return request.RunRef + "\x00" + request.ChangeRef
}

func runFileAppChangeContextErrV0(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
