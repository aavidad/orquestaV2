package orquestaserver

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureV0ServerNoImportaAdaptadoresConcretos(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			for _, forbidden := range forbiddenServerImportsV0() {
				if strings.Contains(path, forbidden) {
					t.Fatalf("%s importa %s", file, path)
				}
			}
		}
	}
}

func forbiddenServerImportsV0() []string {
	return []string{
		"/cmd",
		"/db",
		"database/sql",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-runtime-codex",
		"orquesta/modulos/orquesta-web",
		"orquesta/modulos/orquesta-mcp",
	}
}
