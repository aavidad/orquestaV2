package orquestaappcodexstack

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeAppStackDurableFileV0(path string, data []byte, prefix string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("%s: temp_failed", prefix)
	}
	tmpName := tmp.Name()
	keepTemp := false
	defer removeAppStackTempV0(tmpName, &keepTemp)
	if err := writeAppStackTempV0(tmp, data, prefix); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("%s: rename_failed", prefix)
	}
	keepTemp = true
	return syncAppStackDirV0(filepath.Dir(path), prefix)
}

func writeAppStackTempV0(tmp *os.File, data []byte, prefix string) error {
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

func removeAppStackTempV0(path string, keep *bool) {
	if keep != nil && *keep {
		return
	}
	_ = os.Remove(path)
}

func syncAppStackDirV0(dir string, prefix string) error {
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
