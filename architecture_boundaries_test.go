package orquesta_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNeutralOrchestrationPackagesDoNotImportProductAdapters(t *testing.T) {
	for _, boundary := range []struct {
		pkg       string
		forbidden []string
	}{
		{
			pkg: "orquesta/modulos/orquesta-core",
			forbidden: []string{
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-domain-work-file",
				"orquesta/modulos/orquesta-domain-work-sql",
				"orquesta/modulos/orquesta-factory",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-run-file",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-tool-capability-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os/exec",
			},
		},
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
			pkg: "orquesta/modulos/orquesta-tool-capability",
			forbidden: []string{
				"database/sql",
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-app-codex-stack",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-orchestration-core",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-goal",
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
				"os",
				"os/exec",
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
			pkg: "orquesta/modulos/orquesta-estado-vivo",
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
				"orquesta/modulos/orquesta-runtime-codex",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os",
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
			pkg: "orquesta/modulos/orquesta-document-extraction",
			forbidden: []string{
				"database/sql",
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-document-extraction-csv",
				"orquesta/modulos/orquesta-document-extraction-fake",
				"orquesta/modulos/orquesta-document-extraction-json",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-opes-",
				"orquesta/modulos/orquesta-run-file",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os",
				"os/exec",
			},
		},
		{
			pkg: "orquesta/modulos/orquesta-data-ingestion",
			forbidden: []string{
				"database/sql",
				"orquesta/cmd",
				"orquesta/db",
				"orquesta/modulos/orquesta-mcp",
				"orquesta/modulos/orquesta-runtime",
				"orquesta/modulos/orquesta-state-file",
				"orquesta/modulos/orquesta-web",
				"net/http",
				"os",
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

func TestNeutralOrchestrationPackagesDoNotDependOnProductAdapters(t *testing.T) {
	for _, pkg := range []string{
		"orquesta/modulos/orquesta-core",
		"orquesta/modulos/orquesta-core-workflow",
		"orquesta/modulos/orquesta-domain-work",
		"orquesta/modulos/orquesta-estado-vivo",
		"orquesta/modulos/orquesta-goal",
		"orquesta/modulos/orquesta-orchestration-core",
	} {
		deps := packageDepsForBoundaryTest(t, pkg)
		for _, forbidden := range []string{
			"orquesta/modulos/orquesta-app-codex-stack",
			"orquesta/modulos/orquesta-data-ingestion",
			"orquesta/modulos/orquesta-document-extraction",
			"orquesta/modulos/orquesta-opes-",
			"orquesta/modulos/orquesta-presentation-extraction",
			"orquesta/modulos/orquesta-runtime-codex",
			"orquesta/modulos/orquesta-tool-capability",
		} {
			for _, dep := range deps {
				if strings.Contains(dep, forbidden) {
					t.Fatalf("%s depends on forbidden product adapter %s", pkg, dep)
				}
			}
		}
	}
}

func TestCodexGoalAdapterDoesNotImportServerStorageOrShell(t *testing.T) {
	imports := packageImportsForBoundaryTest(t, "orquesta/modulos/orquesta-runtime-codex-goal")
	for _, forbidden := range []string{
		"net/http",
		"os",
		"os/exec",
	} {
		for _, imported := range imports {
			if imported == forbidden {
				t.Fatalf("orquesta-runtime-codex-goal imports forbidden dependency %s", imported)
			}
		}
	}
	deps := packageDepsForBoundaryTest(t, "orquesta/modulos/orquesta-runtime-codex-goal")
	for _, forbidden := range []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-file",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
	} {
		for _, dep := range deps {
			if strings.Contains(dep, forbidden) {
				t.Fatalf("orquesta-runtime-codex-goal depends on forbidden adapter dependency %s", dep)
			}
		}
	}
}

func TestCodexAppServerAdapterDoesNotImportCmdOrProductStorage(t *testing.T) {
	deps := packageDepsForBoundaryTest(t, "orquesta/modulos/orquesta-runtime-codex-appserver")
	for _, forbidden := range []string{
		"database/sql",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/mattn/go-sqlite3",
		"modernc.org/sqlite",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-file",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
	} {
		for _, dep := range deps {
			if strings.Contains(dep, forbidden) {
				t.Fatalf("orquesta-runtime-codex-appserver depends on forbidden adapter dependency %s", dep)
			}
		}
	}
	for _, dep := range deps {
		if dep == "orquesta/modulos/orquesta-server" {
			t.Fatalf("orquesta-runtime-codex-appserver depends on composition server config %s", dep)
		}
	}
}

func TestDirectorV2NeutralPackagesDoNotDependOnRuntimeOrProductAdapters(t *testing.T) {
	for _, pkg := range []string{
		"orquesta/modulos/orquesta-agent-progress",
		"orquesta/modulos/orquesta-director",
		"orquesta/modulos/orquesta-director-cycle",
		"orquesta/modulos/orquesta-director-cycle-outbox",
		"orquesta/modulos/orquesta-director-runner",
		"orquesta/modulos/orquesta-director-scheduler",
		"orquesta/modulos/orquesta-director-supervised-burst",
		"orquesta/modulos/orquesta-director-supervisor",
		"orquesta/modulos/orquesta-director-tick-input",
	} {
		deps := packageDepsForBoundaryTest(t, pkg)
		forbiddenDeps := append([]string(nil), forbiddenNeutralDBImportsV0()...)
		forbiddenDeps = append(forbiddenDeps,
			"net/http",
			"orquesta/cmd",
			"orquesta/db",
			"orquesta/modulos/orquesta-app-codex-stack",
			"orquesta/modulos/orquesta-mcp",
			"orquesta/modulos/orquesta-opes-",
			"orquesta/modulos/orquesta-runtime",
			"orquesta/modulos/orquesta-runtime-codex",
			"os/exec",
		)
		for _, forbidden := range forbiddenDeps {
			for _, dep := range deps {
				if strings.Contains(dep, forbidden) {
					t.Fatalf("%s depends on forbidden Director V2 dependency %s", pkg, dep)
				}
			}
		}
	}
}

func TestNeutralOrchestrationPackagesDoNotDependOnFactoryOrHTTP(t *testing.T) {
	for _, pkg := range []string{
		"orquesta/modulos/orquesta-app-change",
		"orquesta/modulos/orquesta-app-change-director-source",
		"orquesta/modulos/orquesta-core",
		"orquesta/modulos/orquesta-core-concurrency",
		"orquesta/modulos/orquesta-core-leases",
		"orquesta/modulos/orquesta-core-replanner",
		"orquesta/modulos/orquesta-core-workflow",
		"orquesta/modulos/orquesta-director",
		"orquesta/modulos/orquesta-director-operativo",
		"orquesta/modulos/orquesta-document-plan-expander",
		"orquesta/modulos/orquesta-domain-work",
		"orquesta/modulos/orquesta-domain-work-memory",
		"orquesta/modulos/orquesta-estado-vivo",
		"orquesta/modulos/orquesta-external-work-run",
		"orquesta/modulos/orquesta-goal",
		"orquesta/modulos/orquesta-orchestration-core",
		"orquesta/modulos/orquesta-run-control",
		"orquesta/modulos/orquesta-run-queue",
		"orquesta/modulos/orquesta-runtime",
	} {
		deps := packageDepsForBoundaryTest(t, pkg)
		for _, forbidden := range []string{
			"orquesta/modulos/orquesta-factory",
			"net/http",
		} {
			for _, dep := range deps {
				if dep == forbidden {
					t.Fatalf("%s depends on forbidden neutral-boundary dependency %s", pkg, dep)
				}
			}
		}
	}
}

func TestFactoryDoesNotDependOnHTTPAdapter(t *testing.T) {
	deps := packageDepsForBoundaryTest(t, "orquesta/modulos/orquesta-factory")
	for _, forbidden := range []string{
		"net/http",
		"orquesta/modulos/orquesta-factory-http",
	} {
		for _, dep := range deps {
			if dep == forbidden {
				t.Fatalf("orquesta-factory depends on forbidden HTTP boundary dependency %s", dep)
			}
		}
	}
}

func TestNeutralCoreDoesNotDependTransitivelyOnAdapters(t *testing.T) {
	deps := packageDepsForBoundaryTest(t, "orquesta/modulos/orquesta-core")
	for _, forbidden := range []string{
		"database/sql",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-file",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-factory",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
		"net/http",
		"os/exec",
	} {
		for _, dep := range deps {
			if strings.Contains(dep, forbidden) {
				t.Fatalf("orquesta-core depends on forbidden adapter %s", dep)
			}
		}
	}
}

func TestNeutralOrchestrationProductionCodeDoesNotHardcodeLocalPaths(t *testing.T) {
	for _, dir := range []string{
		"modulos/orquesta-core-workflow",
		"modulos/orquesta-domain-work",
		"modulos/orquesta-tool-capability",
		"modulos/orquesta-estado-vivo",
		"modulos/orquesta-goal",
		"modulos/orquesta-orchestration-core",
	} {
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, forbidden := range []string{
				"/home/alberto/",
				"/Users/alberto/",
				`C:\Users\`,
			} {
				if strings.Contains(string(content), forbidden) {
					t.Fatalf("%s hardcodes local path marker %q", path, forbidden)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
}

func TestCausalVerdictAuthorityCallersRemainExplicitV0(t *testing.T) {
	approved := map[string]bool{
		"modulos/orquesta-app-codex-stack/goal_first_resident_rework_v0.go":   false,
		"modulos/orquesta-estado-vivo/proyeccion_v0.go":                       false,
		"modulos/orquesta-mcp/observe_app_director_goal_estado_vivo_v0.go":    false,
		"modulos/orquesta-run-coordinator/external_work_reconciliation_v0.go": false,
	}

	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		callsAuthority := false
		ast.Inspect(file, func(node ast.Node) bool {
			if callsAuthority {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				if fun.Name == "DerivarVeredictoCausalV0" {
					callsAuthority = true
				}
			case *ast.SelectorExpr:
				if fun.Sel.Name == "DerivarVeredictoCausalV0" {
					callsAuthority = true
				}
			}
			return !callsAuthority
		})
		if !callsAuthority {
			return nil
		}

		path = filepath.ToSlash(strings.TrimPrefix(path, "./"))
		if _, ok := approved[path]; !ok {
			t.Errorf("DerivarVeredictoCausalV0 has unapproved production caller %s", path)
			return nil
		}
		approved[path] = true
		return nil
	})
	if err != nil {
		t.Fatalf("scan production Go callers: %v", err)
	}
	for path, found := range approved {
		if !found {
			t.Errorf("approved DerivarVeredictoCausalV0 caller missing from %s", path)
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

func packageDepsForBoundaryTest(t *testing.T, pkg string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list deps %s: %v\n%s", pkg, err, string(output))
	}
	return nonEmptyBoundaryLinesV0(string(output))
}

func nonEmptyBoundaryLinesV0(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
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
