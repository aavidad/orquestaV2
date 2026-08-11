package acceptance_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/application"
)

const v25CodexProviderCatalogFixturePath = "acceptance/fixtures/v25_codex_provider_catalog.json"

type v25CodexProviderCatalogFixture struct {
	SchemaVersion              int                       `json:"schema_version"`
	ContractID                 string                    `json:"contract_id"`
	Status                     string                    `json:"status"`
	CapabilitiesAccredited     []string                  `json:"capabilities_accredited"`
	RelatedAcceptanceContract  v25AcceptanceContract     `json:"related_acceptance_contract"`
	RelatedRoadmapCapabilities []v25RoadmapCapability    `json:"related_roadmap_capabilities"`
	ProviderRef                string                    `json:"provider_ref"`
	FoundationFiles            []string                  `json:"foundation_files"`
	ObservedFacts              []string                  `json:"observed_facts"`
	FailClosedFacts            []string                  `json:"fail_closed_facts"`
	AuthorityRules             []string                  `json:"authority_rules"`
	ForbiddenImports           []string                  `json:"forbidden_imports"`
	Deferred                   []string                  `json:"deferred"`
	RequiredTests              []v25ProviderRequiredTest `json:"required_tests"`
}

var _ application.ProviderCatalogSource = (*codex.ProviderCatalogSource)(nil)

func TestAcceptanceV25CodexProviderCatalogFoundation(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v25CodexProviderCatalogFixture](
		t, filepath.Join(root, filepath.FromSlash(v25CodexProviderCatalogFixturePath)),
	)
	if fixture.SchemaVersion != 1 || fixture.ContractID != "FOUNDATION-V25-CODEX-PROVIDER-CATALOG" ||
		fixture.Status != "foundation_only_unwired" || len(fixture.CapabilitiesAccredited) != 0 ||
		fixture.ProviderRef != codex.ProviderRef ||
		fixture.RelatedAcceptanceContract != (v25AcceptanceContract{ID: "AC-V25-PROVIDER-ADAPTERS", Status: "planned"}) {
		t.Fatalf("invalid Codex provider catalog foundation: %+v", fixture)
	}
	assertV25ProviderStrings(t, v25CodexCapabilityIDs(fixture.RelatedRoadmapCapabilities), []string{"AGT-02", "AGT-12"})
	assertV25ProviderStrings(t, fixture.FoundationFiles, []string{
		"internal/adapters/agent/codex/provider_catalog.go",
		"internal/adapters/agent/codex/provider_catalog_test.go",
		"acceptance/v25_codex_provider_catalog_test.go",
		"acceptance/fixtures/v25_codex_provider_catalog.json",
	})
	assertV25ProviderStrings(t, fixture.ObservedFacts, []string{
		"provider_ref_is_stable_codex_identity",
		"model_ref_is_exact_configured_or_logical_default",
		"local_adapter_shutdown_is_observed_as_unavailable",
	})
	assertV25ProviderStrings(t, fixture.FailClosedFacts, []string{
		"live_local_adapter_does_not_prove_remote_availability",
		"provider_quota_is_unknown",
		"provider_usage_is_unknown",
		"model_capabilities_are_not_inferred",
		"model_reasoning_efforts_are_not_inferred",
	})
	assertV25ProviderStrings(t, fixture.AuthorityRules, []string{
		"source_is_read_only",
		"source_does_not_write_goal_lifecycle",
		"observation_window_is_explicit_not_defaulted",
		"application_remains_routing_authority",
	})
	assertV25ProviderStrings(t, fixture.Deferred, []string{
		"canonical configuration and bootstrap composition",
		"trustworthy live provider availability observation",
		"provider quota observation integration",
		"provider advertised model capability discovery",
		"isolated real Codex provider catalog end-to-end evidence",
		"V25 acceptance accreditation",
	})
	if len(fixture.RequiredTests) != 2 ||
		fixture.RequiredTests[0] != (v25ProviderRequiredTest{
			Command:            "go test -mod=vendor -count=1 ./internal/adapters/agent/codex -run '^TestCodexProviderCatalog'",
			NamePrefix:         "TestCodexProviderCatalog",
			RejectNoTestsToRun: true,
		}) || fixture.RequiredTests[1] != (v25ProviderRequiredTest{
		Command:            "go test -mod=vendor -count=1 ./acceptance -run '^TestAcceptanceV25CodexProviderCatalogFoundation$'",
		NamePrefix:         "TestAcceptanceV25CodexProviderCatalogFoundation",
		RejectNoTestsToRun: true,
	}) {
		t.Fatalf("invalid Codex provider test inventory: %+v", fixture.RequiredTests)
	}

	assertV25CodexProviderSourceBoundary(t, root, fixture)
	assertV25CodexProviderSourceMethods(t)
	assertV25ProviderRoadmapRemainsUnaccredited(t, root, v25ProviderContractFixture{
		RelatedAcceptanceContract:  fixture.RelatedAcceptanceContract,
		RelatedRoadmapCapabilities: fixture.RelatedRoadmapCapabilities,
	})
}

func v25CodexCapabilityIDs(capabilities []v25RoadmapCapability) []string {
	result := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		result = append(result, capability.ID)
	}
	return result
}

func assertV25CodexProviderSourceBoundary(
	t *testing.T,
	root string,
	fixture v25CodexProviderCatalogFixture,
) {
	t.Helper()
	parsed, err := parser.ParseFile(
		token.NewFileSet(), filepath.Join(root, "internal/adapters/agent/codex/provider_catalog.go"), nil, parser.ImportsOnly,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range parsed.Imports {
		path := imported.Path.Value[1 : len(imported.Path.Value)-1]
		for _, forbidden := range fixture.ForbiddenImports {
			if path == forbidden {
				t.Fatalf("Codex provider catalog imports forbidden authority %s", path)
			}
		}
	}
}

func assertV25CodexProviderSourceMethods(t *testing.T) {
	t.Helper()
	typeOfSource := reflect.TypeOf((*codex.ProviderCatalogSource)(nil))
	methods := make([]string, 0, typeOfSource.NumMethod())
	for index := 0; index < typeOfSource.NumMethod(); index++ {
		methods = append(methods, typeOfSource.Method(index).Name)
	}
	sort.Strings(methods)
	if !reflect.DeepEqual(methods, []string{"ObserveProviderCatalog", "ProviderRef"}) {
		t.Fatalf("Codex provider catalog gained authority: %v", methods)
	}
}
