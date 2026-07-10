package orquestatoolcapability

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedToolConsumerDoesNotRequireOrquestaRuntimeV0(t *testing.T) {
	for _, file := range productionToolCapabilityFilesV0(t) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			for _, forbidden := range []string{
				"orquesta/modulos/orquesta-orchestration-core",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-runtime-codex",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-tool-capability-file",
				"net/http", "os", "os/exec",
			} {
				if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
					t.Fatalf("embedded SDK imports forbidden dependency %s from %s", importPath, file)
				}
			}
		}
	}
}

func productionToolCapabilityFilesV0(t *testing.T) []string {
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
