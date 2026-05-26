package orquestadirectoroperativo

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectorOperativoNoImportaAdaptadoresConcretos(t *testing.T) {
	forbidden := []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
		"net/http",
		"os/exec",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-file",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		assertFileDoesNotImportForbiddenV0(t, entry.Name(), forbidden)
	}
}

func TestDirectorOperativoDocsLocalesNoReabrenCierresVigentes(t *testing.T) {
	paths := []string{
		"README.md",
		filepath.Join("docs", "contratos.md"),
		filepath.Join("docs", "pruebas.md"),
		filepath.Join("docs", "tareas.md"),
	}
	var combined strings.Builder
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		combined.Write(content)
		combined.WriteByte('\n')
	}
	docs := combined.String()
	for _, required := range []string{
		"CODEX-WAVE-REAL",
		"CODEX-RECURSION-REAL",
		"OPES temporal real de derivados/cierre",
	} {
		if !strings.Contains(docs, required) {
			t.Fatalf("docs locales no declaran estado vigente %q", required)
		}
	}
	for _, stale := range []string{
		"quedan pendientes wait por cohorte/ola",
		"recursion Codex real con parent/child refs",
		"smoke con Codex real amplio/recursivo",
	} {
		if strings.Contains(docs, stale) {
			t.Fatalf("docs locales reabren pendiente stale %q", stale)
		}
	}
}

func assertFileDoesNotImportForbiddenV0(t *testing.T, path string, forbidden []string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports %s: %v", path, err)
	}
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		for _, blocked := range forbidden {
			if importPath == blocked || strings.HasPrefix(importPath, blocked+"/") ||
				strings.Contains(importPath, blocked) {
				t.Fatalf("%s importa dependencia prohibida %q", path, importPath)
			}
		}
	}
}
