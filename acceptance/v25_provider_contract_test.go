package acceptance_test

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"orquesta/internal/application"
)

const v25ProviderContractFixturePath = "acceptance/fixtures/v25_provider_contract.json"

type v25ProviderContractFixture struct {
	SchemaVersion              int                       `json:"schema_version"`
	ContractID                 string                    `json:"contract_id"`
	Status                     string                    `json:"status"`
	CapabilitiesAccredited     []string                  `json:"capabilities_accredited"`
	RelatedAcceptanceContract  v25AcceptanceContract     `json:"related_acceptance_contract"`
	RelatedRoadmapCapabilities []v25RoadmapCapability    `json:"related_roadmap_capabilities"`
	FoundationFiles            []string                  `json:"foundation_files"`
	AuthorityRules             []string                  `json:"authority_rules"`
	ContractCases              []string                  `json:"contract_cases"`
	ForbiddenImportBoundaries  []string                  `json:"forbidden_import_boundaries"`
	Deferred                   []string                  `json:"deferred"`
	RequiredTests              []v25ProviderRequiredTest `json:"required_tests"`
}

type v25AcceptanceContract struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type v25RoadmapCapability struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	EvidenceRefs []string `json:"evidence_refs"`
}

type v25ProviderRequiredTest struct {
	Command            string `json:"command"`
	NamePrefix         string `json:"name_prefix"`
	RejectNoTestsToRun bool   `json:"reject_no_tests_to_run"`
}

type v25ProviderCatalogQuerySurface interface {
	ObserveProviderCatalog(context.Context) (application.ProviderCatalog, error)
	RouteProviderModel(context.Context, application.ProviderRouteRequest) (application.ProviderRouteDecision, error)
}

var _ v25ProviderCatalogQuerySurface = (*application.Orchestrator)(nil)

func TestAcceptanceV25ProviderContractFoundation(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v25ProviderContractFixture](
		t, filepath.Join(root, filepath.FromSlash(v25ProviderContractFixturePath)),
	)
	assertV25ProviderFoundationEnvelope(t, root, fixture)
	assertV25ProviderSourceBoundary(t, root, fixture)
	assertV25ProviderRoadmapRemainsUnaccredited(t, root, fixture)
	assertV25ProviderObserverHasNoLifecycleAuthority(t)
	assertV25ProviderQueryOnlyComposition(t, root)
}

func assertV25ProviderFoundationEnvelope(t *testing.T, root string, fixture v25ProviderContractFixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ContractID != "FOUNDATION-V25-PROVIDER-CONTRACT" ||
		fixture.Status != "query_only_unwired_to_launch" || len(fixture.CapabilitiesAccredited) != 0 {
		t.Fatalf("invalid V25 foundation envelope: %+v", fixture)
	}
	if fixture.RelatedAcceptanceContract != (v25AcceptanceContract{ID: "AC-V25-PROVIDER-ADAPTERS", Status: "planned"}) {
		t.Fatalf("V25 acceptance must remain planned: %+v", fixture.RelatedAcceptanceContract)
	}
	capabilityIDs := make([]string, 0, len(fixture.RelatedRoadmapCapabilities))
	for _, capability := range fixture.RelatedRoadmapCapabilities {
		capabilityIDs = append(capabilityIDs, capability.ID)
	}
	assertV25ProviderStrings(t, capabilityIDs, []string{
		"ORC-07", "ORC-26", "ORC-27", "ORC-29", "AGT-02", "AGT-09", "AGT-11", "AGT-12",
	})
	assertV25ProviderStrings(t, fixture.FoundationFiles, []string{
		"internal/ports/provider_contract.go",
		"internal/application/provider_catalog.go",
		"internal/application/provider_routing.go",
		"internal/application/orchestrator.go",
		"internal/application/provider_contract_test.go",
		"internal/application/provider_catalog_orchestrator_test.go",
		"acceptance/v25_provider_contract_test.go",
		"acceptance/fixtures/v25_provider_contract.json",
	})
	assertV25ProviderStrings(t, fixture.AuthorityRules, []string{
		"providers_observe_only",
		"application_decides_routes",
		"fallback_is_explicit",
		"unknown_fails_closed",
		"provider_facts_never_write_goal_lifecycle",
		"orchestrator_clock_is_canonical",
		"sources_normalized_once_at_composition",
		"query_only_unwired_to_launch",
	})
	assertV25ProviderStrings(t, fixture.ContractCases, []string{
		"provider_absent",
		"quota_unknown",
		"quota_exhausted",
		"isolated_provider_failure",
		"no_inferred_model_or_capability_parity",
		"ambiguous_composition_rejected",
		"canonical_clock_refresh",
		"query_does_not_write_lifecycle",
	})
	assertV25ProviderStrings(t, fixture.Deferred, []string{
		"concrete provider adapters",
		"provider-specific configuration and bootstrap",
		"cost latency confidence and sample scoring policy",
		"adapters composition and gates required to accredit related roadmap capabilities",
		"durable integration with the existing agent lifecycle",
		"real provider end-to-end evidence",
		"V25 acceptance accreditation",
	})
	if len(fixture.RequiredTests) != 2 ||
		fixture.RequiredTests[0] != (v25ProviderRequiredTest{
			Command:            "go test -mod=vendor -count=1 ./internal/application -run '^TestProvider'",
			NamePrefix:         "TestProvider",
			RejectNoTestsToRun: true,
		}) || fixture.RequiredTests[1] != (v25ProviderRequiredTest{
		Command:            "go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV25ProviderContractFoundation$'",
		NamePrefix:         "TestAcceptanceV25ProviderContractFoundation",
		RejectNoTestsToRun: true,
	}) {
		t.Fatalf("invalid V25 focused test inventory: %+v", fixture.RequiredTests)
	}
	for _, name := range fixture.FoundationFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatalf("foundation file %q: %v", name, err)
		}
	}
}

func assertV25ProviderSourceBoundary(t *testing.T, root string, fixture v25ProviderContractFixture) {
	t.Helper()
	for _, relative := range []string{
		"internal/ports/provider_contract.go",
		"internal/application/provider_catalog.go",
		"internal/application/provider_routing.go",
	} {
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, relative), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range parsed.Imports {
			path := imported.Path.Value[1 : len(imported.Path.Value)-1]
			for _, forbidden := range fixture.ForbiddenImportBoundaries {
				if len(path) >= len(forbidden) && path[:len(forbidden)] == forbidden {
					t.Fatalf("%s imports forbidden boundary %s", relative, path)
				}
			}
		}
	}
}

func assertV25ProviderRoadmapRemainsUnaccredited(
	t *testing.T,
	root string,
	fixture v25ProviderContractFixture,
) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "product/roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var roadmap struct {
		AcceptanceContracts []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"acceptance_contracts"`
		CapabilityEntries []v25RoadmapCapability `json:"capability_entries"`
	}
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	foundContract := false
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID == fixture.RelatedAcceptanceContract.ID {
			foundContract = true
			if contract.Status != "planned" {
				t.Fatalf("V25 acceptance unexpectedly accredited: %+v", contract)
			}
		}
	}
	if !foundContract {
		t.Fatal("AC-V25-PROVIDER-ADAPTERS missing from roadmap")
	}
	wanted := make(map[string]v25RoadmapCapability, len(fixture.RelatedRoadmapCapabilities))
	for _, capability := range fixture.RelatedRoadmapCapabilities {
		wanted[capability.ID] = capability
	}
	for _, capability := range roadmap.CapabilityEntries {
		if expected, found := wanted[capability.ID]; found {
			if capability.Status != expected.Status || len(capability.EvidenceRefs) != 0 {
				t.Fatalf("V25 capability %s unexpectedly accredited: %+v", capability.ID, capability)
			}
			delete(wanted, capability.ID)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("V25 roadmap capabilities missing: %v", wanted)
	}
}

func assertV25ProviderObserverHasNoLifecycleAuthority(t *testing.T) {
	t.Helper()
	contract := reflect.TypeOf((*application.ProviderCatalogSource)(nil)).Elem()
	actual := make([]string, 0, contract.NumMethod())
	for index := 0; index < contract.NumMethod(); index++ {
		actual = append(actual, contract.Method(index).Name)
	}
	assertV25ProviderStrings(t, actual, []string{"ObserveProviderCatalog", "ProviderRef"})
}

func assertV25ProviderQueryOnlyComposition(t *testing.T, root string) {
	t.Helper()
	dependencies := reflect.TypeOf(application.Dependencies{})
	field, found := dependencies.FieldByName("ProviderCatalogSources")
	if !found || field.Type != reflect.TypeOf([]application.ProviderCatalogSource(nil)) {
		t.Fatalf("provider catalog dependency missing or invalid: %+v found=%t", field, found)
	}
	processing, err := os.ReadFile(filepath.Join(root, "internal/application/processing.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ProviderCatalog", "RouteProviderModel"} {
		if strings.Contains(string(processing), forbidden) {
			t.Fatalf("V25 query-only catalog wired into processing through %q", forbidden)
		}
	}
}

func assertV25ProviderStrings(t *testing.T, actual, expected []string) {
	t.Helper()
	actual = append([]string(nil), actual...)
	expected = append([]string(nil), expected...)
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("values=%v want=%v", actual, expected)
	}
}
