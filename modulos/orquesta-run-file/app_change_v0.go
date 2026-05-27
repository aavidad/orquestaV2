package orquestarunfile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
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
	if len(snapshot.Records) > runFileSnapshotMaxRecordsV0 {
		return nil, fmt.Errorf("orquesta_run_file: app_change_records_limit_exceeded")
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
	record.Request.ExternalWork = copyRunFileAppChangeExternalWorkV0(record.Request.ExternalWork)
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
	request.ExternalWork = normalizeRunFileAppChangeExternalWorkV0(request.ExternalWork)
	if request.RequestID == "" {
		request.RequestID = request.ChangeRef
	}
	if request.CorrelationID == "" {
		request.CorrelationID = request.RequestID
	}
	return request
}

func normalizeRunFileAppChangeExternalWorkV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) *orquestaappchange.AppChangeExternalWorkV0 {
	if work == nil {
		return nil
	}
	out := &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    strings.TrimSpace(work.ProjectRef),
		JobRef:        strings.TrimSpace(work.JobRef),
		InterfaceRefs: compactRunFileStringsOrEmptyV0(work.InterfaceRefs),
		WorkKind:      strings.TrimSpace(work.WorkKind),
		WorkRefs:      compactRunFileStringsOrEmptyV0(work.WorkRefs),
		InputFields:   normalizeRunFileDomainWorkFieldsV0(work.InputFields),
	}
	if out.ProjectRef == "" &&
		out.JobRef == "" &&
		len(out.InterfaceRefs) == 0 &&
		out.WorkKind == "" &&
		len(out.WorkRefs) == 0 &&
		len(out.InputFields) == 0 {
		return nil
	}
	return out
}

func copyRunFileAppChangeExternalWorkV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) *orquestaappchange.AppChangeExternalWorkV0 {
	return normalizeRunFileAppChangeExternalWorkV0(work)
}

func normalizeRunFileDomainWorkFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	seen := map[string]struct{}{}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields))
	for _, field := range fields {
		normalized := orquestadomainwork.DomainWorkFieldV0{
			Name:      strings.TrimSpace(field.Name),
			Value:     strings.TrimSpace(field.Value),
			Values:    compactRunFileStringsV0(field.Values),
			ValueJSON: normalizeRunFileDomainWorkFieldJSONV0(field.ValueJSON),
		}
		if normalized.Name == "" ||
			(normalized.Value == "" && len(normalized.Values) == 0 && len(normalized.ValueJSON) == 0) {
			continue
		}
		key := normalized.Name + "\x00" + normalized.Value + "\x00" +
			strings.Join(normalized.Values, "\x00") + "\x00" + string(normalized.ValueJSON)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	if out == nil {
		return []orquestadomainwork.DomainWorkFieldV0{}
	}
	return out
}

func normalizeRunFileDomainWorkFieldJSONV0(
	value json.RawMessage,
) json.RawMessage {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	var raw any
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return append(json.RawMessage(nil), trimmed...)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return append(json.RawMessage(nil), trimmed...)
	}
	return append(json.RawMessage(nil), data...)
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
