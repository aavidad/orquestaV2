package orquestaruncontrol

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestArchitectureV0BlocksAdaptersAndExternalTerms(t *testing.T) {
	files := productionGoFilesV0(t)
	for _, file := range files {
		assertAllowedProductionImportsV0(t, file)
		assertNoForbiddenProductionTermsV0(t, file)
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

func assertAllowedProductionImportsV0(t *testing.T, file string) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, spec := range parsed.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if forbiddenProductionImportV0(path) {
			t.Fatalf("%s imports forbidden package %q", file, path)
		}
	}
}

func forbiddenProductionImportV0(path string) bool {
	for _, banned := range forbiddenProductionImportsV0() {
		if path == banned || strings.HasPrefix(path, banned+"/") {
			return true
		}
	}
	return false
}

func forbiddenProductionImportsV0() []string {
	return []string{
		"database/sql", "net", "os", "path/filepath", "runtime",
		"github.com/go-sql-driver/mysql", "github.com/jackc/pgx", "modernc.org/sqlite",
	}
}

func assertNoForbiddenProductionTermsV0(t *testing.T, file string) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	lower := strings.ToLower(string(data))
	for _, term := range forbiddenProductionTermsV0() {
		if regexp.MustCompile(`\b`+regexp.QuoteMeta(term)+`\b`).FindStringIndex(lower) != nil {
			t.Fatalf("%s contains forbidden term %q", file, term)
		}
	}
}

func forbiddenProductionTermsV0() []string {
	return []string{
		"database", "sqlite", "postgres", "mysql", "sql", "http", "mcp", "runtime",
	}
}
