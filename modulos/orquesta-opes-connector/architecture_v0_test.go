package orquestaopesconnector

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureV0NoImportaDBLegacyNiRuntime(t *testing.T) {
	for _, file := range productionOPESConnectorFilesV0(t) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			for _, forbidden := range forbiddenOPESConnectorImportsV0() {
				if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
					t.Fatalf("%s importa %s", file, path)
				}
			}
		}
	}
}

func productionOPESConnectorFilesV0(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return files
}

func forbiddenOPESConnectorImportsV0() []string {
	return []string{
		"database/sql",
		"os/exec",
		"orquesta/cmd",
		"orquesta/db",
	}
}
