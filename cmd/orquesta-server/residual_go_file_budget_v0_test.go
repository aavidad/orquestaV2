package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const residualGoFileBudgetMaxLinesV0 = 300

var residualGoFileBudgetBaselineV0 = map[string]int{
	"cmd/orquesta-server/codex_director_wave_prompts_v0.go":                          313,
	"cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go":                319,
	"cmd/orquesta-server/idle_self_improvement_federated_backlog_v0.go":              311,
	"cmd/orquesta-server/idle_self_improvement_stack_v0.go":                          333,
	"cmd/orquesta-server/mcp_real_transport_v0.go":                                   593,
	"cmd/orquesta-server/stack.go":                                                   352,
	"modulos/orquesta-core-workflow/run_state_validation_v0.go":                      303,
	"modulos/orquesta-core-workflow/work_items_validation_v0.go":                     306,
	"modulos/orquesta-director/agent_progress_supervisor_helpers_v0.go":              307,
	"modulos/orquesta-external-work-run/service_v0.go":                               305,
	"modulos/orquesta-runtime-codex-delivery/progress_source_v0.go":                  312,
	"modulos/orquesta-runtime-codex-delivery/progress_state_v0.go":                   310,
	"modulos/orquesta-runtime-codex-delivery/source_review_gate_project_files_v0.go": 326,
	"modulos/orquesta-runtime-codex-delivery/source_v0.go":                           314,
}

var residualGoFileBudgetScopesV0 = []string{
	"cmd/orquesta-server",
	"modulos/orquesta-app-gateway",
	"modulos/orquesta-core-workflow",
	"modulos/orquesta-director",
	"modulos/orquesta-external-work-run",
	"modulos/orquesta-run-coordinator",
	"modulos/orquesta-runtime-codex-delivery",
	"modulos/orquesta-runtime-required-test",
}

func TestResidualGoFileBudgetT90V0(t *testing.T) {
	repoRoot := findRepoRootForResidualGoFileBudgetTestV0(t)
	seen := map[string]bool{}
	var issues []string
	for _, scope := range residualGoFileBudgetScopesV0 {
		scopePath := filepath.Join(repoRoot, filepath.FromSlash(scope))
		err := filepath.WalkDir(scopePath, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel := relativeSlashPathForResidualGoFileBudgetTestV0(t, repoRoot, path)
			lines := countFileLinesForResidualGoFileBudgetTestV0(t, path)
			baseline, baselined := residualGoFileBudgetBaselineV0[rel]
			if lines > residualGoFileBudgetMaxLinesV0 && !baselined {
				issues = append(issues, rel+": nuevo fichero >300 sin baseline")
				return nil
			}
			if baselined {
				seen[rel] = true
				if lines > baseline {
					issues = append(issues, rel+": crece de baseline")
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", scope, err)
		}
	}
	for rel := range residualGoFileBudgetBaselineV0 {
		if !seen[rel] {
			issues = append(issues, rel+": baseline sin fichero actual")
		}
	}
	sort.Strings(issues)
	if len(issues) > 0 {
		t.Fatalf("baseline T90 roto: %s", strings.Join(issues, "; "))
	}
}

func findRepoRootForResidualGoFileBudgetTestV0(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod no encontrado desde %s", dir)
		}
		dir = parent
	}
}

func relativeSlashPathForResidualGoFileBudgetTestV0(t *testing.T, root string, path string) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("rel %s: %v", path, err)
	}
	return filepath.ToSlash(rel)
}

func countFileLinesForResidualGoFileBudgetTestV0(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lines := bytes.Count(data, []byte{'\n'})
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}
