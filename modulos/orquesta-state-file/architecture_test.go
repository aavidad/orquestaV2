package orquestastatefile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateFileModuleDoesNotImportForbiddenPeripheryV0(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob go files: %v", err)
	}
	forbidden := []string{
		`"database/sql"`,
		`"orquesta/cmd`,
		`"orquesta/db`,
		`"orquesta/internal`,
		`go-sqlite3`,
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, needle := range forbidden {
			if strings.Contains(string(data), needle) {
				t.Fatalf("%s imports forbidden dependency %q", file, needle)
			}
		}
	}
}
