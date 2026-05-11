package orquestadirectorcandidates

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestArquitecturaV0BloqueaImportsYTerminosProhibidos(t *testing.T) {
	files := productionGoFilesV0(t)
	for _, file := range files {
		assertAllowedImportsV0(t, file)
		assertNoForbiddenTermsV0(t, file)
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

func assertAllowedImportsV0(t *testing.T, file string) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, spec := range parsed.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if forbiddenImportV0(path) {
			t.Fatalf("%s imports forbidden package %q", file, path)
		}
	}
}

func forbiddenImportV0(path string) bool {
	for _, banned := range forbiddenImportsV0() {
		if path == banned || strings.HasPrefix(path, banned+"/") {
			return true
		}
	}
	return false
}

func forbiddenImportsV0() []string {
	return []string{
		"database/sql", "net", "os", "path/filepath", "runtime",
		"github.com/go-sql-driver/mysql", "github.com/jackc/pgx", "modernc.org/sqlite",
	}
}

func assertNoForbiddenTermsV0(t *testing.T, file string) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	lower := strings.ToLower(string(data))
	for _, term := range forbiddenTermsV0() {
		if regexp.MustCompile(`\b`+regexp.QuoteMeta(term)+`\b`).FindStringIndex(lower) != nil {
			t.Fatalf("%s contains forbidden term %q", file, term)
		}
	}
}

func forbiddenTermsV0() []string {
	return []string{
		"db", "sqlite", "postgres", "runtime", "provider", "proveedor",
		"model", "modelo", "home", "oauth", "filesystem", "network",
	}
}
