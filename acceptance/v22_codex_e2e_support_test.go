package acceptance_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type v22RoadmapDocument struct {
	Verticals           []v22RoadmapVertical   `json:"verticals"`
	AcceptanceContracts []v22RoadmapAcceptance `json:"acceptance_contracts"`
	CapabilityEntries   []v22RoadmapCapability `json:"capability_entries"`
}
type v22RoadmapVertical struct {
	ID                  string   `json:"id"`
	DependsOn           []string `json:"depends_on"`
	AcceptanceContracts []string `json:"acceptance_contracts"`
}
type v22RoadmapAcceptance struct {
	ID         string   `json:"id"`
	Vertical   string   `json:"vertical"`
	Status     string   `json:"status"`
	Assertions []string `json:"assertions"`
}
type v22RoadmapCapability struct {
	ID                  string   `json:"id"`
	Decision            string   `json:"decision"`
	OwnerContext        string   `json:"owner_context"`
	Dependencies        []string `json:"dependencies"`
	AcceptanceContracts []string `json:"acceptance_contracts"`
}

func v22AssertRoadmapBoundary(t *testing.T, root string, fixture v22Fixture) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, "product/roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var roadmap v22RoadmapDocument
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	var vertical v22RoadmapVertical
	for _, candidate := range roadmap.Verticals {
		if candidate.ID == "codex_e2e" {
			vertical = candidate
		}
	}
	dependencies := []string{"recovery_backup", "council", "command_registry", "i18n"}
	if vertical.ID == "" || !reflect.DeepEqual(vertical.DependsOn, dependencies) ||
		!reflect.DeepEqual(vertical.AcceptanceContracts, []string{"AC-V22-CODEX-E2E"}) {
		t.Fatalf("invalid V22 roadmap vertical: %+v", vertical)
	}
	var contract v22RoadmapAcceptance
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V22-CODEX-E2E" {
			contract = candidate
		}
	}
	assertions := []string{
		"public MCP drives plan DAG mailbox workspace tests reviews and close",
		"four concurrent Goals survive selective stop restart backup and shutdown",
		"no terminal contradiction or owned process remains",
	}
	if contract.Vertical != "codex_e2e" || (contract.Status != "planned" && contract.Status != "executable") ||
		!reflect.DeepEqual(contract.Assertions, assertions) {
		t.Fatalf("invalid V22 roadmap acceptance boundary: %+v", contract)
	}
	owned := []string{}
	for _, capability := range roadmap.CapabilityEntries {
		if capability.OwnerContext != "codex_e2e" || capability.Decision != "accept" {
			continue
		}
		owned = append(owned, capability.ID)
		if !reflect.DeepEqual(capability.Dependencies, dependencies) ||
			!reflect.DeepEqual(capability.AcceptanceContracts, vertical.AcceptanceContracts) {
			t.Errorf("V22 capability %s has non-causal ownership", capability.ID)
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, fixture.OwnedCapabilityIDs) {
		t.Fatalf("V22 roadmap ownership=%v want=%v", owned, fixture.OwnedCapabilityIDs)
	}
}

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
