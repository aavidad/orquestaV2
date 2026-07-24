package acceptance_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestV22SimplicityBudgetMeasuresPhysicalDelta(t *testing.T) {
	root, totals := evidenceRepositoryRoot(t), map[string]int{}
	for path, added := range v22AddedLines(t, root) {
		class := v22DeltaClass(path)
		totals[class] += added
		if class == "product" && added > 500 {
			t.Logf("V22 advisory: product %s added LOC=%d target=500", path, added)
		}
	}
	if totals["product"] > 3000 {
		t.Logf("V22 advisory: product added LOC=%d target=3000", totals["product"])
	}
	if totals["test"] > 4200 {
		t.Logf("V22 advisory: test/harness added LOC=%d target=4200", totals["test"])
	}
	v22AssertSingleAuthorities(t, root)
}

func v22AddedLines(t *testing.T, root string) map[string]int {
	t.Helper()
	result := map[string]int{}
	output, err := evidenceGit(root, "diff", "--numstat", v22ContractBaseGitCommitOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if row == "" {
			continue
		}
		fields := strings.SplitN(row, "\t", 3)
		if len(fields) != 3 || fields[0] == "-" {
			t.Fatalf("V22 unmeasurable numstat %q", row)
		}
		added, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatal(err)
		}
		result[filepath.ToSlash(fields[2])] = added
	}
	output, err = evidenceGit(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range strings.Fields(string(output)) {
		result[filepath.ToSlash(path)] = v22PhysicalLines(t, filepath.Join(root, path))
	}
	return result
}

func v22DeltaClass(path string) string {
	if strings.HasPrefix(path, "product/evidence/") {
		return ""
	}
	if strings.HasSuffix(path, "_test.go") || strings.HasPrefix(path, "acceptance/") || strings.Contains(path, "/fixtures/") || strings.HasPrefix(path, "scripts/") {
		return "test"
	}
	if strings.HasPrefix(path, "internal/") || strings.HasPrefix(path, "cmd/") || strings.HasPrefix(path, "sdk/") || strings.HasPrefix(path, "config/") || strings.HasSuffix(path, ".sql") || strings.HasSuffix(path, ".json") {
		return "product"
	}
	return ""
}

func TestV22SimplicityBudgetExcludesOnlyPostSealEvidence(t *testing.T) {
	if v22DeltaClass("product/evidence/v22_codex_e2e.json") != "" || v22DeltaClass("product/roadmap.json") != "product" {
		t.Fatal("V22 budget must exclude only post-seal evidence")
	}
}

func v22PhysicalLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Count(string(data), "\n")
	if len(data) != 0 && data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}

func v22AssertSingleAuthorities(t *testing.T, root string) {
	t.Helper()
	state, err := os.ReadDir(filepath.Join(root, "internal/adapters/state"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state) != 1 || !state[0].IsDir() || state[0].Name() != "sqlite" {
		t.Errorf("V22 top-level state stores=%v want [sqlite]", state)
	}
	for _, check := range []struct{ dir, name string }{{"internal/application", "Orchestrator"}, {"internal/bootstrap", "scheduler"}, {"internal/adapters/agent/codex", "Adapter"}} {
		if got := v22NamedStructs(t, filepath.Join(root, check.dir), check.name); got != 1 {
			t.Errorf("V22 %s.%s declarations=%d want=1", check.dir, check.name, got)
		}
	}
	paths, err := filepath.Glob(filepath.Join(root, "internal/commands", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Errorf("V22 command registry count=%d want=1", len(paths))
	}
}

func v22NamedStructs(t *testing.T, root, name string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range general.Specs {
				typed, ok := spec.(*ast.TypeSpec)
				if !ok || typed.Name.Name != name {
					continue
				}
				if _, ok := typed.Type.(*ast.StructType); ok {
					count++
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return count
}
