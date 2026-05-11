package orquestadirectorsupervisedburst

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectorSupervisedBurstV0NoImportaAdaptadoresOperativos(t *testing.T) {
	forbidden := []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"modernc.org/sqlite",
		"orquesta/cmd",
		"orquesta/internal",
		"orquesta/modulos/orquesta-persistence",
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-web",
		"time",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		assertBurstFileImportsAllowedV0(t, entry.Name(), forbidden)
	}
}

func assertBurstFileImportsAllowedV0(t *testing.T, path string, forbidden []string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports %s: %v", path, err)
	}
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		for _, blocked := range forbidden {
			if importPath == blocked || strings.HasPrefix(importPath, blocked+"/") {
				t.Fatalf("%s importa adaptador prohibido %q", path, importPath)
			}
		}
	}
}
