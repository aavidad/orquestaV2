package orquestadecisioncouncil

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecisionCouncilV0NoImportaAdaptadoresNiProveedores(t *testing.T) {
	forbidden := []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"modernc.org/sqlite",
		"orquesta/cmd",
		"orquesta/internal",
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-web",
		"orquesta/modulos/orquesta-persistence",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		assertCouncilFileImportsAllowedV0(t, entry.Name(), forbidden)
		assertCouncilFileDoesNotPersistTranscriptsV0(t, entry.Name())
	}
}

func assertCouncilFileImportsAllowedV0(t *testing.T, path string, forbidden []string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports %s: %v", path, err)
	}
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		for _, blocked := range forbidden {
			if blockedCouncilImportV0(importPath, blocked) {
				t.Fatalf("%s importa adaptador prohibido %q", path, importPath)
			}
		}
	}
}

func blockedCouncilImportV0(importPath string, blocked string) bool {
	return importPath == blocked ||
		strings.HasPrefix(importPath, blocked+"/") ||
		strings.HasPrefix(importPath, blocked+"-")
}

func assertCouncilFileDoesNotPersistTranscriptsV0(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lower := strings.ToLower(string(content))
	if strings.Contains(lower, "transcript") {
		t.Fatalf("%s contiene transcript; use source refs opacas", path)
	}
}
