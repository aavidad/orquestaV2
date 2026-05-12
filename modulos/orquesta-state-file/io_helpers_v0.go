package orquestastatefile

import (
	"encoding/json"
	"os"
)

func encodeAndSyncJSONV0(file *os.File, value any) error {
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	return file.Sync()
}

func removeTempIfNeededV0(tmpName string, committed *bool) {
	if committed != nil && *committed {
		return
	}
	_ = os.Remove(tmpName)
}

func syncDirV0(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Sync()
}
