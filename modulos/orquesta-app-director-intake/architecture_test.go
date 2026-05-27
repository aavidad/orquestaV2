package orquestaappdirectorintake

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

func TestAppDirectorIntakeArchitectureV0NoImportaLegacyNiDBHardcodeada(t *testing.T) {
	for _, file := range productionGoFilesForDirectorIntakeTestV0(t) {
		parsed := parseDirectorIntakeSourceFileV0(t, file)
		for _, path := range importsForDirectorIntakeSourceFileV0(t, parsed) {
			if orquestarails.ArchitectureImportForbiddenV0(path, directorIntakeImportPolicyV0()) {
				t.Fatalf("%s importa dependencia prohibida %q", file, path)
			}
		}
		assertDirectorIntakeSourceLiteralsSafeV0(t, file, parsed)
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
		parsed := parseDirectorIntakeSourceFileV0(t, file)
		for _, path := range importsForDirectorIntakeSourceFileV0(t, parsed) {
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

func parseDirectorIntakeSourceFileV0(t *testing.T, file string) *ast.File {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Clean(file), nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return parsed
}

func importsForDirectorIntakeSourceFileV0(t *testing.T, parsed *ast.File) []string {
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

func assertDirectorIntakeSourceLiteralsSafeV0(t *testing.T, file string, parsed *ast.File) {
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
		if orquestarails.ArchitectureSourceLiteralForbiddenV0("orquesta-app-director-intake", value) {
			t.Fatalf("%s contiene literal sensible o adaptador concreto %q", file, value)
		}
		return true
	})
}

func directorIntakeImportPolicyV0() orquestarails.ArchitectureImportPolicyV0 {
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
