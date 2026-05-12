package orquestastatefileoutbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func newOutboxLedgerStateV0() outboxLedgerStateV0 {
	return outboxLedgerStateV0{
		recordsByMessageID: make(map[string]*outboxLedgerRecordV0),
		messageIDByIdemKey: make(map[string]string),
	}
}

func loadOutboxLedgerStateV0(path string) (outboxLedgerStateV0, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newOutboxLedgerStateV0(), nil
	}
	if err != nil {
		return outboxLedgerStateV0{}, fmt.Errorf("file_outbox_ledger: read_failed")
	}
	var snapshot outboxLedgerSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return outboxLedgerStateV0{}, fmt.Errorf("file_outbox_ledger: json_invalid")
	}
	if snapshot.SchemaVersion != fileOutboxLedgerSchemaVersionV0 {
		return outboxLedgerStateV0{}, fmt.Errorf("file_outbox_ledger: schema_invalid")
	}
	return rebuildOutboxLedgerStateV0(snapshot.Records)
}

func persistOutboxLedgerStateV0(path string, state outboxLedgerStateV0) error {
	snapshot := outboxLedgerSnapshotV0{
		SchemaVersion: fileOutboxLedgerSchemaVersionV0,
		Records:       outboxLedgerRecordsForSnapshotV0(state),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("file_outbox_ledger: json_failed")
	}
	return writeOutboxLedgerAtomicV0(path, append(data, '\n'))
}

func writeOutboxLedgerAtomicV0(path string, data []byte) error {
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
	_ = syncOutboxLedgerDirV0(dir)
	return nil
}

func syncOutboxLedgerDirV0(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Sync()
}
