package orquestaappcodexstack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppCodexStackNoImportaLegacyCmdNiDBV0(t *testing.T) {
	root := "."
	for _, path := range goFilesForArchitectureTestV0(t, root) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(data)
		for _, forbidden := range []string{
			`"orquesta/` + `cmd"`,
			`"orquesta/` + `db"`,
			`github.com/mattn/go-` + `sqlite3`,
			`github.com/jackc/` + `pgx`,
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contiene import prohibido %s", path, forbidden)
			}
		}
	}
}

func goFilesForArchitectureTestV0(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
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
