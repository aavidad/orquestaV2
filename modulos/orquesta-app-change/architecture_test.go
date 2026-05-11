package orquestaappchange

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArquitecturaAppChangeNoImportaLegacyDBRuntimeV0(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source := string(data)
		for _, forbidden := range []string{`"orquesta/cmd"`, `"orquesta/db"`, `"database/sql"`, `"os/exec"`} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s importa %s", file, forbidden)
			}
		}
	}
}
