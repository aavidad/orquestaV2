package orquestapersistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func NewFileOutboxLedgerV0(dir string) (*FileOutboxLedgerV0, error) {
	path, err := fileOutboxLedgerPathV0(dir)
	if err != nil {
		return nil, fmt.Errorf("file_outbox_ledger: dir_invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("file_outbox_ledger: dir_unavailable")
	}
	state, err := loadFileOutboxLedgerStateV0(path)
	if err != nil {
		return nil, err
	}
	return &FileOutboxLedgerV0{path: path, state: state}, nil
}

func fileOutboxLedgerPathV0(dir string) (string, error) {
	dir = trimV0(dir)
	if dir == "" {
		return "", filepath.ErrBadPattern
	}
	return filepath.Join(filepath.Clean(dir), fileOutboxLedgerNameV0), nil
}

func newFileOutboxLedgerStateV0() fileOutboxLedgerStateV0 {
	return fileOutboxLedgerStateV0{
		recordsByMessageID: make(map[string]*fileOutboxLedgerRecordV0),
		messageIDByIdemKey: make(map[string]string),
	}
}

func loadFileOutboxLedgerStateV0(path string) (fileOutboxLedgerStateV0, error) {
	state, migrated, err := loadFileOutboxLedgerStateWithMigrationV0(path)
	if err != nil {
		return fileOutboxLedgerStateV0{}, err
	}
	if migrated {
		_ = persistFileOutboxLedgerStateV0(path, state)
	}
	return state, nil
}

func loadFileOutboxLedgerStateWithMigrationV0(path string) (fileOutboxLedgerStateV0, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newFileOutboxLedgerStateV0(), false, nil
	}
	if err != nil {
		return fileOutboxLedgerStateV0{}, false, fmt.Errorf("file_outbox_ledger: read_failed")
	}
	var snapshot fileOutboxLedgerSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fileOutboxLedgerStateV0{}, false, fmt.Errorf("file_outbox_ledger: json_invalid")
	}
	switch snapshot.SchemaVersion {
	case FileOutboxLedgerSchemaVersionV0:
		state, err := rebuildFileOutboxLedgerStateV0(snapshot.Records)
		return state, false, err
	case legacyFileOutboxLedgerSchemaVersionV0:
		state, err := rebuildFileOutboxLedgerStateV0(migrateLegacyFileOutboxLedgerRecordsV0(snapshot.Records))
		return state, true, err
	default:
		return fileOutboxLedgerStateV0{}, false, fmt.Errorf("file_outbox_ledger: schema_invalid")
	}
}

func migrateLegacyFileOutboxLedgerRecordsV0(
	records []fileOutboxLedgerRecordV0,
) []fileOutboxLedgerRecordV0 {
	migrated := make([]fileOutboxLedgerRecordV0, 0, len(records))
	for _, record := range records {
		if record.Ack != nil && trimV0(record.Ack.Status) == "" {
			ack := *record.Ack
			ack.Status = OutboxDispatchStatusDispatchedV0
			record.Ack = &ack
		}
		migrated = append(migrated, record)
	}
	return migrated
}

func persistFileOutboxLedgerStateV0(path string, state fileOutboxLedgerStateV0) error {
	snapshot := fileOutboxLedgerSnapshotV0{
		SchemaVersion: FileOutboxLedgerSchemaVersionV0,
		Records:       fileOutboxLedgerRecordsForSnapshotV0(state),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("file_outbox_ledger: json_failed")
	}
	return writeFileOutboxLedgerAtomicV0(path, append(data, '\n'))
}

func writeFileOutboxLedgerAtomicV0(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("file_outbox_ledger: mkdir_failed")
	}
	tmp, err := os.CreateTemp(dir, ".outbox_ledger_*.tmp")
	if err != nil {
		return fmt.Errorf("file_outbox_ledger: write_failed")
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("file_outbox_ledger: write_failed")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("file_outbox_ledger: sync_failed")
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("file_outbox_ledger: close_failed")
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("file_outbox_ledger: rename_failed")
	}
	return syncFileOutboxLedgerDirV0(dir)
}

func syncFileOutboxLedgerDirV0(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Sync()
}
