package orquestaruntimecodexdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	fileCodexProgressStateSchemaVersionV0 = "orquesta.runtime.codex_delivery.progress_state.file.v0"
	fileCodexProgressStateNameV0          = "codex_progress_state_v0.json"
)

type FileCodexProgressStateStoreV0 struct {
	mu      sync.Mutex
	path    string
	records map[string]codexProgressStateRecordV0
}

type fileCodexProgressStateSnapshotV0 struct {
	SchemaVersion string                           `json:"schema_version"`
	Records       []fileCodexProgressStateRecordV0 `json:"records"`
}

type fileCodexProgressStateRecordV0 struct {
	Key   string                     `json:"key"`
	State codexProgressStateRecordV0 `json:"state"`
}

func NewFileCodexProgressStateStoreV0(dir string) (*FileCodexProgressStateStoreV0, error) {
	dir = filepath.Clean(dir)
	if dir == "." || dir == "" || !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("codex_progress_state_file_store: dir_invalid")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("codex_progress_state_file_store: dir_unavailable")
	}
	path := filepath.Join(dir, fileCodexProgressStateNameV0)
	records, err := loadFileCodexProgressStateV0(path)
	if err != nil {
		return nil, err
	}
	return &FileCodexProgressStateStoreV0{path: path, records: records}, nil
}

func (store *FileCodexProgressStateStoreV0) ObserveCodexProgressV0(
	ctx context.Context,
	sample CodexProgressSampleV0,
) (CodexProgressObservationStateV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return CodexProgressObservationStateV0{}, err
	}
	sample = normalizeCodexProgressSampleV0(sample)
	if err := validateCodexProgressSampleV0(sample); err != nil {
		return CodexProgressObservationStateV0{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureRecordsLockedV0()
	key := codexProgressSampleKeyV0(sample)
	record := store.records[key]
	if !record.acceptsSampleV0(sample) {
		return record.observationStateV0(sample.Signature, false), nil
	}
	next := record.nextHeartbeatV0(sample)
	if err := store.persistNextV0(key, next); err != nil {
		return CodexProgressObservationStateV0{}, err
	}
	return next.observationStateV0(sample.Signature, true), nil
}

func (store *FileCodexProgressStateStoreV0) MarkCodexProgressReportedV0(
	ctx context.Context,
	mark CodexProgressReportMarkV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	mark = normalizeCodexProgressReportMarkV0(mark)
	if err := validateCodexProgressReportMarkV0(mark); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureRecordsLockedV0()
	key := codexProgressReportMarkKeyV0(mark)
	record := store.records[key]
	record.ReportedSignature = mark.Signature
	record.ReportedStatus = mark.Status
	return store.persistNextV0(key, record)
}

func (store *FileCodexProgressStateStoreV0) ensureRecordsLockedV0() {
	if store.records == nil {
		store.records = map[string]codexProgressStateRecordV0{}
	}
}

func (store *FileCodexProgressStateStoreV0) persistNextV0(
	key string,
	record codexProgressStateRecordV0,
) error {
	next := make(map[string]codexProgressStateRecordV0, len(store.records)+1)
	for existingKey, existing := range store.records {
		next[existingKey] = existing
	}
	next[key] = record
	if err := persistFileCodexProgressStateV0(store.path, next); err != nil {
		return err
	}
	store.records = next
	return nil
}

func loadFileCodexProgressStateV0(path string) (map[string]codexProgressStateRecordV0, error) {
	data, err := readCodexDeliveryFileSnapshotBytesV0(path, "codex_progress_state_file_store")
	if errors.Is(err, os.ErrNotExist) {
		return map[string]codexProgressStateRecordV0{}, nil
	}
	if err != nil {
		return nil, err
	}
	var snapshot fileCodexProgressStateSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("codex_progress_state_file_store: json_invalid")
	}
	if snapshot.SchemaVersion != fileCodexProgressStateSchemaVersionV0 {
		return nil, fmt.Errorf("codex_progress_state_file_store: schema_invalid")
	}
	if len(snapshot.Records) > codexDeliveryFileSnapshotMaxRecordsV0 {
		return nil, fmt.Errorf("codex_progress_state_file_store: records_limit_exceeded")
	}
	records := map[string]codexProgressStateRecordV0{}
	for _, record := range snapshot.Records {
		if record.Key == "" {
			return nil, fmt.Errorf("codex_progress_state_file_store: record_invalid")
		}
		records[record.Key] = record.State
	}
	return records, nil
}

func persistFileCodexProgressStateV0(
	path string,
	records map[string]codexProgressStateRecordV0,
) error {
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	snapshot := fileCodexProgressStateSnapshotV0{
		SchemaVersion: fileCodexProgressStateSchemaVersionV0,
		Records:       make([]fileCodexProgressStateRecordV0, 0, len(keys)),
	}
	for _, key := range keys {
		snapshot.Records = append(snapshot.Records, fileCodexProgressStateRecordV0{
			Key:   key,
			State: records[key],
		})
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("codex_progress_state_file_store: json_failed")
	}
	data = append(data, '\n')
	return writeCodexDeliveryDurableFileV0(path, data, "codex_progress_state_file_store")
}
