package artifactsapp

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileRepository struct {
	mu   sync.Mutex
	path string
}

func NewFileRepository(path string) *FileRepository {
	return &FileRepository{path: filepath.Clean(path)}
}

func (r *FileRepository) SaveArtifact(artifact Artifact) error {
	if r == nil || r.path == "" {
		return errors.New("path de artifacts obligatorio")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	encoded, err := json.Marshal(artifact)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return nil
}

func (r *FileRepository) ListArtifacts(filter ListFilter) ([]Artifact, error) {
	if r == nil || r.path == "" {
		return nil, errors.New("path de artifacts obligatorio")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	items, err := r.readAllLocked()
	if err != nil {
		return nil, err
	}
	out := make([]Artifact, 0, len(items))
	for _, item := range items {
		if matchesFilter(item, filter) {
			out = append(out, item)
		}
	}
	sortArtifacts(out)
	return out, nil
}

func (r *FileRepository) readAllLocked() ([]Artifact, error) {
	f, err := os.Open(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	items := make([]Artifact, 0)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item Artifact
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return items, nil
}
