package orquesta_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestStatusSurfaceBudgetTPer502V0(t *testing.T) {
	root := findRepoRootForStatusSurfaceBudgetTPer502V0(t)
	got := collectStatusSurfaceEndpointsTPer502V0(t, root)
	want := []string{
		"/api/v0/apps/director/goal/observe",
		"/api/v0/autoprogramming/goal/observe",
		"/api/v0/autoprogramming/goals/observe-active",
		"/api/v0/autoprogramming/status",
		"/api/v0/codebase/status",
		"/api/v0/director/stats",
		"/api/v0/director/stats/nope",
		"/api/v0/domain-work/status",
		"/api/v0/external-work/observe",
		"/api/v0/operational-status/query",
		"/api/v0/queue/global-status",
		"/api/v0/resident-director/control",
		"/api/v0/runs/control",
		"/api/v0/runs/control?token=secret&run_ref=run-ref-001",
		"/api/v0/server/readiness",
		"/api/v0/server/status",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf(
			"la superficie de status solo puede crecer con decision documentada: lee docs/informe_pericial_claude_orquesta_2026-07-03.md R5\nwant=%v\ngot=%v",
			want,
			got,
		)
	}
}

func collectStatusSurfaceEndpointsTPer502V0(t *testing.T, root string) []string {
	t.Helper()
	endpointLiteral := regexp.MustCompile(`"/api/v0/[^"]*"`)
	interesting := regexp.MustCompile(`(?i)(status|stats|readiness|observe|health|cockpit|control)`)
	seen := map[string]struct{}{}
	for _, base := range []string{"cmd", "modulos"} {
		err := filepath.WalkDir(filepath.Join(root, base), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, match := range endpointLiteral.FindAllString(string(content), -1) {
				endpoint := strings.Trim(match, `"`)
				if interesting.MatchString(strings.ToLower(endpoint)) {
					seen[endpoint] = struct{}{}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", base, err)
		}
	}
	endpoints := make([]string, 0, len(seen))
	for endpoint := range seen {
		endpoints = append(endpoints, endpoint)
	}
	sort.Strings(endpoints)
	return endpoints
}

func findRepoRootForStatusSurfaceBudgetTPer502V0(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if hasDirTPer502V0(dir, "cmd") && hasDirTPer502V0(dir, "modulos") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root no encontrado desde %s", dir)
		}
		dir = parent
	}
}

func hasDirTPer502V0(root string, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}
