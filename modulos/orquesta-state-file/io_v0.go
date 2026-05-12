package orquestastatefile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func readJSONFileV0[T any](path string) (T, bool, error) {
	var out T
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, false, nil
	}
	if err != nil {
		return out, false, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, true, err
	}
	return out, true, nil
}

func writeJSONAtomicV0(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	committed := false
	defer removeTempIfNeededV0(tmpName, &committed)
	if err := encodeAndSyncJSONV0(tmp, value); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	committed = true
	return syncDirV0(filepath.Dir(path))
}
