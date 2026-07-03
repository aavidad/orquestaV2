package orquesta_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectorV2FreezeFileBudget(t *testing.T) {
	const freezeMessage = "espina V2 congelada; si el aumento es un fix aprobado, actualiza el mapa en el mismo commit y enlaza el bug"

	expected := map[string]int{
		"modulos/orquesta-director-cycle":        11,
		"modulos/orquesta-director-runner":       8,
		"modulos/orquesta-director-scheduler":    34,
		"modulos/orquesta-director-cycle-outbox": 5,
		"modulos/orquesta-director-tick-input":   10,
	}

	for modulePath, expectedCount := range expected {
		actualCount, err := productionGoFileCount(modulePath)
		if err != nil {
			t.Fatalf("%s: %s: %v", modulePath, freezeMessage, err)
		}
		if actualCount != expectedCount {
			t.Fatalf("%s: %s (actual=%d esperado=%d)", modulePath, freezeMessage, actualCount, expectedCount)
		}
	}
}

func productionGoFileCount(modulePath string) (int, error) {
	count := 0
	err := filepath.WalkDir(modulePath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			count++
		}
		return nil
	})
	return count, err
}
