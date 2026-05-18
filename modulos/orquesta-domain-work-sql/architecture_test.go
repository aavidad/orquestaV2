package orquestadomainworksql_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDomainWorkSQLArchitectureV0NoImportaProductoRuntimeNiDrivers(t *testing.T) {
	cmd := exec.Command(
		"go",
		"list",
		"-f",
		"{{join .Imports \"\\n\"}}",
		"orquesta/modulos/orquesta-domain-work-sql",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, string(output))
	}
	forbidden := []string{
		"net",
		"net/http",
		"os",
		"os/exec",
		"runtime",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-file",
		"orquesta/modulos/orquesta-domain-work-memory",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
	}
	imports := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, imported := range imports {
		for _, blocked := range forbidden {
			if imported == blocked || strings.HasPrefix(imported, blocked+"/") {
				t.Fatalf("import prohibido %q", imported)
			}
		}
	}
}
