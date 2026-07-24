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

func v22Require(t *testing.T, ok bool, message string, values ...any) {
	t.Helper(); if !ok { t.Fatalf(message, values...) }
}

func v22MissingRequiredProductTests(t *testing.T, root string, fixture v22Fixture) []string {
	t.Helper(); found := v22TestNames(t, filepath.Join(root, "acceptance"), filepath.Join(root, "internal"))
	missing := []string{}
	for _, required := range fixture.RequiredBehaviorTests {
		if !found[required] { missing = append(missing, required) }
	}
	sort.Strings(missing); return missing
}

func v22AssertRealE2ETestSources(t *testing.T, root string, fixture v22Fixture) {
	t.Helper(); path := filepath.Join(root, "acceptance", "v22_codex_real_e2e_test.go"); content, err := os.ReadFile(path)
	v22Require(t, err == nil, "read V22 E2E source: %v", err)
	source := string(content)
	v22Require(t, strings.Contains(source, "//go:build v22_real_e2e") && !strings.Contains(source, ".Skip(") && !strings.Contains(source, ".Skipf(") && !strings.Contains(source, ".SkipNow("), "V22 real E2E tag/skip contract violated")
	found := v22TestNames(t, path)
	for _, required := range fixture.RequiredRealE2ETests {
		if !found[required] { t.Errorf("V22 real E2E source lacks %s", required) }
	}
}

func v22TestNames(t *testing.T, roots ...string) map[string]bool {
	t.Helper(); found := map[string]bool{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") { return err }
			tree, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil { return err }
			for _, declaration := range tree.Decls {
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil { found[function.Name.Name] = true }
			}
			return nil
		})
		v22Require(t, err == nil, "walk V22 test sources: %v", err)
	}
	return found
}
