package orquestaappchangedirectorsource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArquitecturaAppChangeDirectorSourceNoImportaAdaptadoresConcretosV0(t *testing.T) {
	root := "."
	for _, file := range mustGoFilesForArchitectureTestV0(t, root) {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(data)
		for _, forbidden := range []string{
			`"database/sql"`,
			`"github.com/go-sql-driver/mysql"`,
			`"github.com/jackc/pgx"`,
			`"github.com/lib/pq"`,
			`"github.com/mattn/go-sqlite3"`,
			`"modernc.org/sqlite"`,
			`"orquesta/cmd"`,
			`"orquesta/db"`,
			`"os/exec"`,
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contiene dependencia prohibida %q", file, forbidden)
			}
		}
	}
}

func mustGoFilesForArchitectureTestV0(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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
