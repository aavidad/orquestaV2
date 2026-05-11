package orquestaruncoordinator

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureV0BlocksAdaptersAndStackImports(t *testing.T) {
	for _, file := range productionGoFilesV0(t) {
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
		"database/sql",
		"net/http",
		"os",
		"path/filepath",
		"runtime",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"modernc.org/sqlite",
	}
}
