package orquestarunfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func readJSONSnapshotV0(path string, target any) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("orquesta_run_file: read_failed")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return true, fmt.Errorf("orquesta_run_file: json_invalid")
	}
	return true, nil
}

func writeAtomicJSONSnapshotV0(path string, snapshot any) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("orquesta_run_file: json_failed")
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("orquesta_run_file: temp_failed")
	}
	tmpName := tmp.Name()
	keepTemp := false
	defer removeTempFileV0(tmpName, &keepTemp)
	if err := writeTempFileV0(tmp, data); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("orquesta_run_file: rename_failed")
	}
	keepTemp = true
	return syncDirV0(filepath.Dir(path))
}

func writeTempFileV0(tmp *os.File, data []byte) error {
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("orquesta_run_file: chmod_failed")
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("orquesta_run_file: write_failed")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("orquesta_run_file: sync_failed")
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("orquesta_run_file: close_failed")
	}
	return nil
}

func removeTempFileV0(path string, keep *bool) {
	if keep != nil && *keep {
		return
	}
	_ = os.Remove(path)
}

func syncDirV0(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("orquesta_run_file: dir_sync_failed")
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("orquesta_run_file: dir_sync_failed")
	}
	return nil
}
