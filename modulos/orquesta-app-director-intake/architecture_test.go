package orquestaappdirectorintake

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAppDirectorIntakeArchitectureV0NoImportaLegacyNiDBHardcodeada(t *testing.T) {
	for _, file := range productionGoFilesForDirectorIntakeTestV0(t) {
		src := readDirectorIntakeSourceFileV0(t, file)
		for _, forbidden := range forbiddenDirectorIntakeSourceFragmentsV0() {
			if strings.Contains(strings.ToLower(string(src)), forbidden) {
				t.Fatalf("%s contiene fragmento prohibido %q", file, forbidden)
			}
		}
		for _, path := range importsForDirectorIntakeSourceFileV0(t, file) {
			if directorIntakeImportForbiddenV0(path) {
				t.Fatalf("%s importa dependencia prohibida %q", file, path)
			}
		}
	}
}

func TestAppDirectorIntakeNeutralCoreV0NoImportaFactory(t *testing.T) {
	for _, file := range []string{
		"normalize_v0.go",
		"prepare_v0.go",
		"task_plan_v0.go",
		"task_ref_v0.go",
		"task_v0.go",
		"validation_v0.go",
		"workflow_v0.go",
	} {
		for _, path := range importsForDirectorIntakeSourceFileV0(t, file) {
			if path == "orquesta/modulos/orquesta-factory" {
				t.Fatalf("%s importa factory; usa AppDirectorInputSpecV0 o el adapter dedicado", file)
			}
		}
	}
}

func productionGoFilesForDirectorIntakeTestV0(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Clean(name))
	}
	if len(files) == 0 {
		t.Fatalf("no hay ficheros go de produccion")
	}
	return files
}

func readDirectorIntakeSourceFileV0(t *testing.T, file string) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	return src
}

func importsForDirectorIntakeSourceFileV0(t *testing.T, file string) []string {
	t.Helper()
	src := readDirectorIntakeSourceFileV0(t, file)
	parsed, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("import path %s: %v", spec.Path.Value, err)
		}
		imports = append(imports, path)
	}
	return imports
}

func forbiddenDirectorIntakeSourceFragmentsV0() []string {
	return []string{
		"database/sql",
		"postgres",
		"sqlite",
		"mysql",
		"oauth",
		"ollama",
		"vllm",
		" codex",
		" claude",
		" gemini",
	}
}

func directorIntakeImportForbiddenV0(path string) bool {
	if path == "database/sql" {
		return true
	}
	if strings.HasPrefix(path, "orquesta/cmd") || strings.HasPrefix(path, "orquesta/db") {
		return true
	}
	for _, fragment := range []string{
		"modernc.org/sqlite",
		"github.com/mattn/go-sqlite3",
		"github.com/jackc/pgx",
		"github.com/go-sql-driver/mysql",
	} {
		if strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}
