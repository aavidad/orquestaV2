package orquestaapprunner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func TestAppRunnerArchitectureV0NoImportaLegacyNiDBHardcodeada(t *testing.T) {
	for _, file := range productionGoFilesForRunnerTestV0(t) {
		parsed := parseRunnerSourceFileV0(t, file)
		for _, path := range runnerImportsV0(t, parsed) {
			if orquestarails.ArchitectureImportForbiddenV0(path, runnerImportPolicyV0()) {
				t.Fatalf("%s importa dependencia prohibida %q", file, path)
			}
		}
		assertRunnerSourceLiteralsSafeV0(t, file, parsed)
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

func parseRunnerSourceFileV0(t *testing.T, file string) *ast.File {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return parsed
}

func runnerImportsV0(t *testing.T, parsed *ast.File) []string {
	t.Helper()
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

func assertRunnerSourceLiteralsSafeV0(t *testing.T, file string, parsed *ast.File) {
	t.Helper()
	ast.Inspect(parsed, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		if orquestarails.ArchitectureSourceLiteralForbiddenV0("orquesta-app-runner", value) {
			t.Fatalf("%s contiene literal sensible o adaptador concreto %q", file, value)
		}
		return true
	})
}

func runnerImportPolicyV0() orquestarails.ArchitectureImportPolicyV0 {
	return orquestarails.ArchitectureImportPolicyV0{
		ExactImports:   []string{"database/sql"},
		ImportPrefixes: []string{"orquesta/cmd", "orquesta/db"},
		ImportFragments: []string{
			"modernc.org/sqlite",
			"github.com/mattn/go-sqlite3",
			"github.com/jackc/pgx",
			"github.com/go-sql-driver/mysql",
		},
	}
}
