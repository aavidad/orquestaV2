package orquestaautonomyprogram

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutonomyProgramNoImportaRuntimeNiAdaptadoresV0(t *testing.T) {
	forbidden := []string{"orquesta/cmd", "orquesta/modulos/orquesta-runtime", "orquesta/modulos/orquesta-goal", "orquesta/modulos/orquesta-state-file", "database/sql", "net/http", "os", "os/exec", "path/filepath", "syscall"}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			for _, blocked := range forbidden {
				imported := strings.Trim(spec.Path.Value, "\"")
				if imported == blocked || strings.HasPrefix(imported, blocked+"/") {
					t.Fatalf("%s imports %s", entry.Name(), imported)
				}
			}
		}
	}
}
