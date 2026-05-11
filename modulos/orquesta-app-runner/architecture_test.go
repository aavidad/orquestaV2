package orquestaapprunner

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAppRunnerArchitectureV0NoImportaLegacyNiDBHardcodeada(t *testing.T) {
	for _, file := range productionGoFilesForRunnerTestV0(t) {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range forbiddenRunnerSourceFragmentsV0() {
			if strings.Contains(strings.ToLower(string(src)), forbidden) {
				t.Fatalf("%s contiene fragmento prohibido %q", file, forbidden)
			}
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("import path %s: %v", spec.Path.Value, err)
			}
			if runnerImportForbiddenV0(path) {
				t.Fatalf("%s importa dependencia prohibida %q", file, path)
			}
		}
	}
}

func productionGoFilesForRunnerTestV0(t *testing.T) []string {
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

func forbiddenRunnerSourceFragmentsV0() []string {
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

func runnerImportForbiddenV0(path string) bool {
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
