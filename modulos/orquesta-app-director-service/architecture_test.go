package orquestaappdirectorservice

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAppDirectorServiceArchitectureV0NoImportaLegacyNiDBHardcodeada(t *testing.T) {
	for _, file := range productionGoFilesForServiceTestV0(t) {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range forbiddenServiceSourceFragmentsV0() {
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
			if serviceImportForbiddenV0(path) {
				t.Fatalf("%s importa dependencia prohibida %q", file, path)
			}
		}
	}
}

func productionGoFilesForServiceTestV0(t *testing.T) []string {
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

func forbiddenServiceSourceFragmentsV0() []string {
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

func serviceImportForbiddenV0(path string) bool {
	for _, forbidden := range forbiddenServiceExactImportsV0() {
		if path == forbidden {
			return true
		}
	}
	for _, prefix := range forbiddenServiceImportPrefixesV0() {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, fragment := range forbiddenServiceImportFragmentsV0() {
		if strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}

func forbiddenServiceExactImportsV0() []string {
	return []string{
		"database/sql",
		"net",
		"net/http",
		"os",
		"os/exec",
	}
}

func forbiddenServiceImportPrefixesV0() []string {
	return []string{
		"net/",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-http-gateway",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-operator-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-runtime-",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
	}
}

func forbiddenServiceImportFragmentsV0() []string {
	return []string{
		"codex",
		"modernc.org/sqlite",
		"github.com/mattn/go-sqlite3",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/go-sql-driver/mysql",
	}
}
