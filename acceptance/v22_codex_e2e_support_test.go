package acceptance_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func v22MissingRequiredProductTests(t *testing.T, root string, fixture v22Fixture) []string {
	t.Helper()
	found := map[string]bool{}
	for _, relative := range []string{"acceptance", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, relative), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return err
			}
			tree, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			for _, declaration := range tree.Decls {
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
					found[function.Name.Name] = true
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	missing := []string{}
	for _, required := range fixture.RequiredBehaviorTests {
		if !found[required] {
			missing = append(missing, required)
		}
	}
	sort.Strings(missing)
	return missing
}

func v22AssertRealE2ETestSources(t *testing.T, root string, fixture v22Fixture) {
	t.Helper()
	path := filepath.Join(root, "acceptance", "v22_codex_real_e2e_test.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "//go:build v22_real_e2e") ||
		strings.Contains(source, ".Skip(") || strings.Contains(source, ".Skipf(") ||
		strings.Contains(source, ".SkipNow(") {
		t.Fatal("V22 real E2E tag/skip contract violated")
	}
	tree, err := parser.ParseFile(token.NewFileSet(), path, content, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, declaration := range tree.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
			found[function.Name.Name] = true
		}
	}
	for _, required := range fixture.RequiredRealE2ETests {
		if !found[required] {
			t.Errorf("V22 real E2E source lacks %s", required)
		}
	}
}
