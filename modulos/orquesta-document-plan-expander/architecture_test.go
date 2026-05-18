package orquestadocumentplanexpander_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDocumentPlanExpanderArchitectureV0NoImportaAdaptadoresProducto(t *testing.T) {
	cmd := exec.Command(
		"go",
		"list",
		"-f",
		"{{join .Imports \"\\n\"}}",
		"orquesta/modulos/orquesta-document-plan-expander",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, string(output))
	}
	forbidden := []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
		"net/http",
		"os",
	}
	imports := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, imported := range imports {
		for _, blocked := range forbidden {
			if blocked == "os" {
				if imported == blocked {
					t.Fatalf("import prohibido %q", imported)
				}
				continue
			}
			if strings.Contains(imported, blocked) {
				t.Fatalf("import prohibido %q", imported)
			}
		}
	}
}
