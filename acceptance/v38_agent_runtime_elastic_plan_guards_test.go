package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func v38AssertNoPrematureEvidence(t *testing.T, evidenceRoot string) {
	t.Helper()
	err := filepath.WalkDir(evidenceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("enlace no permitido en evidencias: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("entrada no regular en evidencias: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".json" {
			var record any
			if err := json.Unmarshal(data, &record); err != nil {
				return fmt.Errorf("evidencia JSON ilegible %s: %w", path, err)
			}
			normalized, err := json.Marshal(record)
			if err != nil {
				return err
			}
			data = append(data, normalized...)
		}
		relative, err := filepath.Rel(evidenceRoot, path)
		if err != nil {
			return err
		}
		if v38IsPrematureEvidence(relative, data) {
			return fmt.Errorf("V38 planificada anticipa evidencia en %q", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func v38IsPrematureEvidence(path string, data []byte) bool {
	lowerPath := strings.ToLower(filepath.ToSlash(path))
	lowerData := bytes.ToLower(data)
	for _, marker := range []string{"v38", "agent_runtime_elastic"} {
		if strings.Contains(lowerPath, marker) || bytes.Contains(lowerData, []byte(marker)) {
			return true
		}
	}
	return false
}
