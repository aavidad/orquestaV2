package orquestadirectorsupervisor

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectorSupervisorV0NoImportaAdaptadoresOperativos(t *testing.T) {
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
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		assertSupervisorFileImportsAllowedV0(t, entry.Name(), forbidden)
	}
}

func TestDirectorSupervisorV0NoEsImportadoPorModulosInferiores(t *testing.T) {
	lowerModules := []string{
		"../orquesta-director-cycle",
		"../orquesta-director-cycle-outbox",
		"../orquesta-director-runner",
		"../orquesta-director-scheduler",
		"../orquesta-director-tick-input",
	}
	for _, dir := range lowerModules {
		assertModuleDoesNotImportSupervisorV0(t, dir)
	}
}

func assertSupervisorFileImportsAllowedV0(t *testing.T, path string, forbidden []string) {
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

func assertModuleDoesNotImportSupervisorV0(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if importPath == "orquesta/modulos/orquesta-director-supervisor" {
				t.Fatalf("%s importa supervisor desde modulo inferior", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}
