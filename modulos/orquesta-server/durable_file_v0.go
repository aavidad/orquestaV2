package orquestaserver

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeServerDurableFileV0(path string, data []byte, prefix string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("%s: temp_failed", prefix)
	}
	tmpName := tmp.Name()
	keepTemp := false
	defer removeServerTempV0(tmpName, &keepTemp)
	if err := writeServerTempV0(tmp, data, prefix); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("%s: rename_failed", prefix)
	}
	keepTemp = true
	return syncServerDirV0(filepath.Dir(path), prefix)
}

func writeServerTempV0(tmp *os.File, data []byte, prefix string) error {
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s: chmod_failed", prefix)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s: write_failed", prefix)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s: sync_failed", prefix)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%s: close_failed", prefix)
	}
	return nil
}

func appendServerDurableFileV0(path string, data []byte, prefix string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("%s: open_failed", prefix)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("%s: write_failed", prefix)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("%s: sync_failed", prefix)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("%s: close_failed", prefix)
	}
	return syncServerDirV0(filepath.Dir(path), prefix)
}

func removeServerTempV0(path string, keep *bool) {
	if keep != nil && *keep {
		return
	}
	_ = os.Remove(path)
}

func syncServerDirV0(dir string, prefix string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("%s: dir_sync_failed", prefix)
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("%s: dir_sync_failed", prefix)
	}
	return nil
}
