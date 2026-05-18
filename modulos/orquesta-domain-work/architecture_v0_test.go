package orquestadomainwork

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureV0NoImportaAdaptadoresNiInfraestructura(t *testing.T) {
	for _, file := range productionDomainWorkFilesV0(t) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			for _, forbidden := range forbiddenDomainWorkImportsV0() {
				if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
					t.Fatalf("%s importa %s", file, path)
				}
			}
		}
	}
}

func productionDomainWorkFilesV0(t *testing.T) []string {
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
		t.Fatalf("walk: %v", err)
	}
	return files
}

func forbiddenDomainWorkImportsV0() []string {
	return []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
		"net",
		"os",
		"os/exec",
		"path/filepath",
		"runtime",
		"orquesta/cmd",
		"orquesta/db",
	}
}
