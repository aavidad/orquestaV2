package orquesta_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestNeutralOrchestrationPackagesDoNotImportProductAdapters(t *testing.T) {
	for _, boundary := range []struct {
		pkg       string
		forbidden []string
	}{
		{
			pkg: "orquesta/modulos/orquesta-core-workflow",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-run-file",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-domain-work",
			forbidden: []string{
				"orquesta/modulos/",
				"net/http",
				"os",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-orchestration-core",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-run-file",
				"orquesta/modulos/orquesta-runtime-codex",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-external-work-run",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-director-operativo",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-document-plan-expander",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-run-file",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-domain-work-memory",
			forbidden: []string{
				"database/sql",
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-run-file",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-app-change",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-app-change-director-source",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
	} {
		imports := packageImportsForBoundaryTest(t, boundary.pkg)
		forbiddenImports := append([]string(nil), forbiddenNeutralDBImportsV0()...)
		forbiddenImports = append(forbiddenImports, boundary.forbidden...)
		for _, forbidden := range forbiddenImports {
			for _, imported := range imports {
				if strings.Contains(imported, forbidden) {
					t.Fatalf("%s imports forbidden adapter %s", boundary.pkg, imported)
				}
			}
		}
	}
}

func packageImportsForBoundaryTest(t *testing.T, pkg string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list %s: %v\n%s", pkg, err, string(output))
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	imports := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			imports = append(imports, trimmed)
		}
	}
	return imports
}

func forbiddenNeutralDBImportsV0() []string {
	return []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
	}
}
