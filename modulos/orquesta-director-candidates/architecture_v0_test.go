package orquestadirectorcandidates

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

func TestArquitecturaV0BloqueaImportsYTerminosProhibidos(t *testing.T) {
	files := productionGoFilesV0(t)
	for _, file := range files {
		parsed := parseProductionGoFileV0(t, file)
		assertAllowedImportsV0(t, file, parsed)
		assertNoForbiddenLiteralsV0(t, file, parsed)
	}
}

func productionGoFilesV0(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk package: %v", err)
	}
	return files
}

func parseProductionGoFileV0(t *testing.T, file string) *ast.File {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	return parsed
}

func assertAllowedImportsV0(t *testing.T, file string, parsed *ast.File) {
	t.Helper()
	for _, spec := range parsed.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if orquestarails.ArchitectureImportForbiddenV0(path, importPolicyV0()) {
			t.Fatalf("%s imports forbidden package %q", file, path)
		}
	}
}

func importPolicyV0() orquestarails.ArchitectureImportPolicyV0 {
	return orquestarails.ArchitectureImportPolicyV0{
		ExactImports: []string{"database/sql", "net", "os", "path/filepath", "runtime"},
		ImportFragments: []string{
			"github.com/go-sql-driver/mysql", "github.com/jackc/pgx", "modernc.org/sqlite",
		},
	}
}

func assertNoForbiddenLiteralsV0(t *testing.T, file string, parsed *ast.File) {
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
		if orquestarails.ArchitectureSourceLiteralForbiddenV0("orquesta-director-candidates", value) {
			t.Fatalf("%s contiene literal sensible o adaptador concreto %q", file, value)
		}
		return true
	})
}
