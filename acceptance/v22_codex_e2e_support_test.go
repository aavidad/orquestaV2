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
	TestRef    string   `json:"test_ref"`
	Command    string   `json:"command"`
	Fixture    string   `json:"fixture"`
	Receipt    string   `json:"receipt"`
	Assertions []string `json:"assertions"`
}

type v22RoadmapCapability struct {
	ID                  string   `json:"id"`
	Decision            string   `json:"decision"`
	OwnerContext        string   `json:"owner_context"`
	Dependencies        []string `json:"dependencies"`
	AcceptanceContracts []string `json:"acceptance_contracts"`
	Status              string   `json:"status"`
	EvidenceRefs        []string `json:"evidence_refs"`
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
			break
		}
	}
	if vertical.ID == "" ||
		!reflect.DeepEqual(vertical.DependsOn,
			[]string{"recovery_backup", "council", "command_registry", "i18n"}) ||
		!reflect.DeepEqual(vertical.AcceptanceContracts, []string{"AC-V22-CODEX-E2E"}) {
		t.Fatalf("invalid V22 roadmap vertical: %+v", vertical)
	}
	var contract v22RoadmapAcceptance
	for _, candidate := range roadmap.AcceptanceContracts {
		if candidate.ID == "AC-V22-CODEX-E2E" {
			contract = candidate
			break
		}
	}
	if contract.ID == "" || contract.Vertical != "codex_e2e" ||
		!reflect.DeepEqual(contract.Assertions, []string{
			"public MCP drives plan DAG mailbox workspace tests reviews and close",
			"four concurrent Goals survive selective stop restart backup and shutdown",
			"no terminal contradiction or owned process remains",
		}) {
		t.Fatalf("invalid V22 roadmap acceptance boundary: %+v", contract)
	}
	if contract.Status != "planned" && contract.Status != "executable" {
		t.Fatalf("invalid V22 roadmap lifecycle state: %+v", contract)
	}
	owned := make([]string, 0, len(fixture.OwnedCapabilityIDs))
	for _, capability := range roadmap.CapabilityEntries {
		if capability.OwnerContext != "codex_e2e" || capability.Decision != "accept" {
			continue
		}
		owned = append(owned, capability.ID)
		if !reflect.DeepEqual(capability.Dependencies, vertical.DependsOn) ||
			!reflect.DeepEqual(capability.AcceptanceContracts, vertical.AcceptanceContracts) {
			t.Errorf("V22 capability %s has non-causal ownership: %+v", capability.ID, capability)
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, fixture.OwnedCapabilityIDs) {
		t.Fatalf("V22 roadmap ownership=%v want=%v", owned, fixture.OwnedCapabilityIDs)
	}
}

func v22MissingRequiredProductTests(t *testing.T, root string, fixture v22Fixture) []string {
	t.Helper()
	found := map[string]string{}
	for _, relative := range []string{"acceptance", "internal"} {
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		err := filepath.WalkDir(absolute, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			tree, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			for _, declaration := range tree.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Name == nil ||
					!strings.HasPrefix(function.Name.Name, "Test") {
					continue
				}
				source, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				found[function.Name.Name] = filepath.ToSlash(source)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	missing := make([]string, 0, len(fixture.RequiredBehaviorTests))
	for _, required := range fixture.RequiredBehaviorTests {
		if _, exists := found[required]; !exists {
			missing = append(missing, required)
		}
	}
	sort.Strings(missing)
	return missing
}

func v22AssertRealE2ETestSources(t *testing.T, root string, fixture v22Fixture) {
	t.Helper()
	required := make(map[string]struct{}, len(fixture.RequiredRealE2ETests))
	for _, name := range fixture.RequiredRealE2ETests {
		required[name] = struct{}{}
	}
	path := filepath.Join(root, "acceptance", "v22_codex_real_e2e_test.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("V22 real E2E source: %v", err)
	}
	if !strings.Contains(string(content), "//go:build v22_real_e2e") {
		t.Error("V22 real E2E lacks exact opt-in build tag")
	}
	if strings.Contains(string(content), ".Skip(") ||
		strings.Contains(string(content), ".Skipf(") ||
		strings.Contains(string(content), ".SkipNow(") {
		t.Error("V22 real E2E may not skip missing runtime, credential or preconditions")
	}
	tree, err := parser.ParseFile(token.NewFileSet(), path, content, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]struct{}{}
	for _, declaration := range tree.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil {
			if _, wanted := required[function.Name.Name]; wanted {
				found[function.Name.Name] = struct{}{}
			}
		}
	}
	if !reflect.DeepEqual(v22StringSet(found), fixture.RequiredRealE2ETests) {
		t.Fatalf("V22 real E2E source tests=%v want=%v",
			v22StringSet(found), fixture.RequiredRealE2ETests)
	}
}

func v22StringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
