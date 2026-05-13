package orquestamcp

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPDomainWorkV0NoImportaOPESNiConectorREST(t *testing.T) {
	files, err := filepath.Glob("domain_work_*_v0.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no hay ficheros domain_work_*_v0.go")
	}
	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imported := range parsed.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if strings.Contains(path, "orquesta-opes-connector") ||
				strings.Contains(path, "rest_client") {
				t.Fatalf("%s importa adaptador prohibido: %s", file, path)
			}
		}
	}
}
