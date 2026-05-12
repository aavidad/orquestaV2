package orquestarunfile

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureRunFileNoForbiddenImportsV0(t *testing.T) {
	for _, file := range productionRunFileGoFilesV0(t) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if forbiddenRunFileImportV0(path) {
				t.Fatalf("%s imports forbidden package %q", file, path)
			}
		}
	}
}

func productionRunFileGoFilesV0(t *testing.T) []string {
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

func forbiddenRunFileImportV0(path string) bool {
	for _, banned := range forbiddenRunFileImportsV0() {
		if path == banned || strings.HasPrefix(path, banned+"/") {
			return true
		}
	}
	return false
}

func forbiddenRunFileImportsV0() []string {
	return []string{
		"orquesta/cmd",
		"orquesta/db",
		"database/sql",
		"net",
		"os/exec",
		"runtime",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"modernc.org/sqlite",
	}
}
