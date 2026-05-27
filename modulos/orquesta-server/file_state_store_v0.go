package orquestaserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStateStoreV0 struct {
	mu   sync.Mutex
	path string
}

func NewFileStateStoreV0(path string) (*FileStateStoreV0, error) {
	path = filepath.Clean(path)
	if path == "." || !filepath.IsAbs(path) {
		return nil, fmt.Errorf("orquesta_server_state: path_invalid")
	}
	if filepath.Base(path) != DefaultStateFileV0 && filepath.Ext(path) != ".json" {
		return nil, fmt.Errorf("orquesta_server_state: path_invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("orquesta_server_state: dir_unavailable")
	}
	return &FileStateStoreV0{path: path}, nil
}

func (store *FileStateStoreV0) SaveServerStateV0(
	ctx context.Context,
	state StateV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	state.SchemaVersion = StateSchemaVersionV0
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("orquesta_server_state: json_failed")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	data = append(data, '\n')
	return writeServerDurableFileV0(store.path, data, "orquesta_server_state")
}

func (store *FileStateStoreV0) LoadServerStateV0(ctx context.Context) (StateV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return StateV0{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return StateV0{}, fmt.Errorf("orquesta_server_state: not_found")
	}
	if err != nil {
		return StateV0{}, fmt.Errorf("orquesta_server_state: read_failed")
	}
	var state StateV0
	if err := json.Unmarshal(data, &state); err != nil {
		return StateV0{}, fmt.Errorf("orquesta_server_state: json_invalid")
	}
	if state.SchemaVersion != StateSchemaVersionV0 {
		return StateV0{}, fmt.Errorf("orquesta_server_state: schema_invalid")
	}
	return state, nil
}
