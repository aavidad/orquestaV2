package orquestaopesbridge

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureV0BridgeNoImportaRuntimeServidorNiHTTPDirecto(t *testing.T) {
	for _, file := range productionOPESBridgeFilesV0(t) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			for _, forbidden := range forbiddenOPESBridgeImportsV0() {
				if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
					t.Fatalf("%s importa %s", file, path)
				}
			}
		}
	}
}

func productionOPESBridgeFilesV0(t *testing.T) []string {
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

func forbiddenOPESBridgeImportsV0() []string {
	return []string{
		"database/sql",
		"net/http",
		"os/exec",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-app-director-service",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-runtime-codex",
		"orquesta/modulos/orquesta-runtime-codex-goal",
		"orquesta/modulos/orquesta-server",
		"orquesta/modulos/orquesta-web",
	}
}
