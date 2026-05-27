package orquestastatefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const stateFileJSONMaxBytesV0 int64 = 16 * 1024 * 1024

func readJSONFileV0[T any](path string) (T, bool, error) {
	var out T
	data, err := readJSONFileBytesV0(path)
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

func readJSONFileBytesV0(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("orquesta_state_file: read_failed")
	}
	if info.Size() > stateFileJSONMaxBytesV0 {
		return nil, fmt.Errorf("orquesta_state_file: size_limit_exceeded")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, stateFileJSONMaxBytesV0+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > stateFileJSONMaxBytesV0 {
		return nil, fmt.Errorf("orquesta_state_file: size_limit_exceeded")
	}
	return data, nil
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
