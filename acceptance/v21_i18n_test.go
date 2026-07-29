package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	v21FixturePath              = "acceptance/fixtures/v21_i18n.json"
	v21ContractBaseGitCommitOID = "8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8"
)

type v21Fixture struct {
	SchemaVersion                  int                 `json:"schema_version"`
	ReceiptSchemaVersion           int                 `json:"receipt_schema_version"`
	ContractID                     string              `json:"contract_id"`
	TrustedBaseGitCommitOID        string              `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string              `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string              `json:"product_delta_sealed_git_commit_oid"`
	SealStatus                     string              `json:"seal_status"`
	SealNote                       string              `json:"seal_note"`
	Command                        string              `json:"command"`
	ExecutionArgv                  []string            `json:"execution_argv"`
	OutputPath                     string              `json:"output_path"`
	ReceiptPath                    string              `json:"receipt_path"`
	CandidateSubjects              []string            `json:"candidate_subjects"`
	ImplementationStatus           string              `json:"implementation_status"`
	LifecycleGate                  string              `json:"lifecycle_gate"`
	PreflightStatus                string              `json:"preflight_status"`
	PreflightNote                  string              `json:"preflight_note"`
	BlockingDependency             string              `json:"blocking_dependency_vertical"`
	DependencyReceiptPath          string              `json:"dependency_receipt_path"`
	DependencyActivation           v21DependencyGate   `json:"dependency_activation_contract"`
	OwnedCapabilityIDs             []string            `json:"owned_capability_ids"`
	DependencyVerticals            []string            `json:"dependency_verticals"`
	ManifestPath                   string              `json:"manifest_path"`
	CatalogPaths                   map[string]string   `json:"catalog_paths"`
	PublicDocumentPairs            []v21PublicDocument `json:"public_document_pairs"`
	ActiveSurfaceIDs               []string            `json:"active_surface_ids"`
	FutureSurfaceIDs               []string            `json:"future_surface_ids"`
	DefaultLocale                  string              `json:"default_locale"`
	FallbackLocale                 string              `json:"fallback_locale"`
	EnabledLocales                 []string            `json:"enabled_locales"`
	BCP47Matrix                    v21BCP47Matrix      `json:"bcp47_matrix"`
	Formatters                     []string            `json:"formatters"`
	CatalogChecks                  []string            `json:"catalog_checks"`
	RequiredCatalogAPI             []string            `json:"required_catalog_api"`
	MachineInvariantFields         []string            `json:"machine_invariant_fields"`
	RequiredBehaviorTests          []string            `json:"required_behavior_tests"`
	BindingE2E                     v21BindingE2E       `json:"binding_e2e"`
	SimplicityBudget               v21SimplicityBudget `json:"simplicity_budget"`
	DeferredOwnership              []string            `json:"deferred_ownership"`
	PSEContract                    v21PSEContract      `json:"pse_contract"`
}

type v21DependencyGate struct {
	Contract            string `json:"contract"`
	Result              string `json:"result"`
	SourceWorktreeState string `json:"source_worktree_state"`
	ProductOID          string `json:"product_oid"`
	SealedOID           string `json:"sealed_oid"`
	EvidenceOID         string `json:"evidence_oid"`
	IntegrationHeadOID  string `json:"integration_head_oid"`
}

type v21PublicDocument struct {
	ID string `json:"id"`
	ES string `json:"es"`
	EN string `json:"en"`
}

type v21BCP47Matrix struct {
	Direct              []string `json:"direct"`
	Regional            []string `json:"regional"`
	UnsupportedFallback []string `json:"unsupported_fallback"`
	Invalid             []string `json:"invalid"`
}

type v21BindingE2E struct {
	Status     string   `json:"status"`
	Surfaces   []string `json:"surfaces"`
	Assertions []string `json:"assertions"`
}

type v21SimplicityBudget struct {
	I18NProductLOC         int    `json:"i18n_product_loc"`
	BindingIntegrationLOC  int    `json:"binding_integration_loc"`
	CatalogManifestDataLOC int    `json:"catalog_manifest_data_loc"`
	PublicDocumentLOC      int    `json:"public_document_loc"`
	VendorModule           string `json:"vendor_module"`
	VendorDeltaFileMax     int    `json:"vendor_delta_file_max"`
	VendorDeltaLOCMax      int    `json:"vendor_delta_loc_max"`
	MaximumEnabledLocales  int    `json:"maximum_enabled_locales"`
	MaximumActiveSurfaces  int    `json:"maximum_active_surfaces"`
	MaximumFutureSurfaces  int    `json:"maximum_future_surfaces"`
	MaximumCatalogKeys     int    `json:"maximum_catalog_keys"`
}

type v21PSEContract struct {
	P string `json:"P"`
	S string `json:"S"`
	E string `json:"E"`
}

type v21Manifest struct {
	SchemaVersion   int                         `json:"schema_version"`
	DefaultLocale   string                      `json:"default_locale"`
	FallbackLocale  string                      `json:"fallback_locale"`
	EnabledLocales  []string                    `json:"enabled_locales"`
	Catalogs        []v21ManifestCatalog        `json:"catalogs"`
	Surfaces        []v21ManifestSurface        `json:"surfaces"`
	PublicDocuments []v21ManifestPublicDocument `json:"public_documents"`
}

type v21ManifestCatalog struct {
	Locale string `json:"locale"`
	Path   string `json:"path"`
}

type v21ManifestSurface struct {
	ID            string   `json:"id"`
	State         string   `json:"state"`
	KeySources    []string `json:"key_sources"`
	LiteralPolicy string   `json:"literal_policy"`
}

type v21ManifestPublicDocument struct {
	ID      string            `json:"id"`
	Locales map[string]string `json:"locales"`
}

func TestV21PreflightContractIsStructurallyValid(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	v21AssertFixture(t, root, fixture)
	v21AssertRoadmapBoundary(t, root, fixture)
}

func TestAcceptanceV21I18N(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v21Fixture](t, filepath.Join(root, v21FixturePath))
	v21AssertFixture(t, root, fixture)
	v21AssertRoadmapBoundary(t, root, fixture)

	manifestPath := filepath.Join(root, filepath.FromSlash(fixture.ManifestPath))
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			t.Fatal("V21_PRODUCT_PENDING: canonical i18n manifest is absent on the accredited V20 base")
		}
		t.Fatal(err)
	}
	manifest := evidenceDecodeStrictJSON[v21Manifest](t, manifestPath)
	v21AssertManifest(t, root, fixture, manifest)
	v21AssertCatalogSources(t, root, fixture, manifest)
	v21AssertRequiredProductTests(t, root, fixture)
	v21AssertRequiredCatalogAPI(t, root, fixture)
}

func v21AssertFixture(t *testing.T, root string, fixture v21Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V21-I18N" ||
		fixture.TrustedBaseGitCommitOID != v21ContractBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v21ContractBaseGitCommitOID ||
		fixture.Command == "" || len(fixture.ExecutionArgv) != 3 ||
		fixture.ExecutionArgv[0] != "sh" || fixture.ExecutionArgv[1] != "-c" ||
		fixture.Command != "sh -c '"+fixture.ExecutionArgv[2]+"'" ||
		fixture.OutputPath != "product/evidence/v21_i18n.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v21_i18n.json" ||
		fixture.SealNote == "" || fixture.PreflightNote == "" ||
		fixture.PSEContract.P == "" || fixture.PSEContract.S == "" || fixture.PSEContract.E == "" {
		t.Fatalf("invalid V21 identity/PSE envelope: %+v", fixture)
	}
	if fixture.BlockingDependency != "none" ||
		fixture.DependencyReceiptPath != "product/evidence/v20_command_registry.json" ||
		fixture.DependencyActivation != (v21DependencyGate{
			Contract: "AC-V20-COMMAND-REGISTRY", Result: "PASS",
			SourceWorktreeState: "detached_clean",
			ProductOID:          "a3da82278a3ce947b4425665d92d961c99b2baaf",
			SealedOID:           "7f27685d992c9a8f5f308d94bbfc3e9828009d84",
			EvidenceOID:         "7b1bbaf475743c16c5749b6e4e039e23b0a8354b",
			IntegrationHeadOID:  v21ContractBaseGitCommitOID,
		}) {
		t.Fatalf("invalid V21 dependency activation: %+v", fixture)
	}
	if !reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"UI-18"}) ||
		!reflect.DeepEqual(fixture.DependencyVerticals, []string{"command_registry"}) ||
		fixture.ManifestPath != "internal/i18n/manifest.json" ||
		!reflect.DeepEqual(fixture.CatalogPaths, map[string]string{
			"es": "internal/i18n/catalogs/es.json",
			"en": "internal/i18n/catalogs/en.json",
		}) {
		t.Fatalf("invalid V21 ownership/paths: %+v", fixture)
	}
	assertV21ExactSet(t, "active surfaces", fixture.ActiveSurfaceIDs,
		[]string{"cli", "command_registry", "http", "mcp", "public_docs"})
	assertV21ExactSet(t, "future surfaces", fixture.FutureSurfaceIDs,
		[]string{"notifications", "prompts", "web", "wizard"})
	if fixture.DefaultLocale != "es" || fixture.FallbackLocale != "es" ||
		!reflect.DeepEqual(fixture.EnabledLocales, []string{"es", "en"}) {
		t.Fatalf("invalid V21 locale ownership: %+v", fixture)
	}
	assertV21ExactSet(t, "formatters", fixture.Formatters,
		[]string{"currency", "date", "number", "plural", "timezone"})
	assertV21ExactSet(t, "catalog checks", fixture.CatalogChecks, []string{
		"duplicate_json_keys_fail", "extract_typed_public_keys", "locale_parity",
		"missing_key_fails", "placeholder_sets_match", "public_document_parity", "unused_key_fails",
	})
	assertV21ExactSet(t, "binding surfaces", fixture.BindingE2E.Surfaces, []string{"cli", "http", "mcp"})
	wantBindingStatus := "implemented"
	if fixture.ImplementationStatus == "development_unsealed" {
		wantBindingStatus = "product_pending"
	}
	if fixture.BindingE2E.Status != wantBindingStatus || len(fixture.BindingE2E.Assertions) != 4 ||
		len(fixture.PublicDocumentPairs) != 1 || fixture.PublicDocumentPairs[0] != (v21PublicDocument{
		ID: "orquesta.quickstart", ES: "docs/public/es/README.md", EN: "docs/public/en/README.md",
	}) {
		t.Fatalf("invalid V21 binding/docs contract: %+v", fixture)
	}
	if fixture.SimplicityBudget != (v21SimplicityBudget{
		I18NProductLOC: 900, BindingIntegrationLOC: 500, CatalogManifestDataLOC: 800,
		PublicDocumentLOC: 250, VendorModule: "golang.org/x/text", VendorDeltaFileMax: 40,
		VendorDeltaLOCMax: 12000, MaximumEnabledLocales: 2, MaximumActiveSurfaces: 5,
		MaximumFutureSurfaces: 4, MaximumCatalogKeys: 256,
	}) {
		t.Fatalf("invalid V21 simplicity budget: %+v", fixture.SimplicityBudget)
	}
	if len(fixture.RequiredBehaviorTests) != 12 || !sort.StringsAreSorted(fixture.RequiredBehaviorTests) ||
		len(fixture.RequiredCatalogAPI) != 7 || !sort.StringsAreSorted(fixture.RequiredCatalogAPI) ||
		len(fixture.MachineInvariantFields) != 8 || !sort.StringsAreSorted(fixture.MachineInvariantFields) {
		t.Fatalf("invalid V21 executable ratchets: %+v", fixture)
	}
	if _, err := evidenceGitCanonicalCommit(root, fixture.TrustedBaseGitCommitOID); err != nil {
		t.Fatalf("invalid V21 trusted base: %v", err)
	}
	evidenceAssertReceiptV3(t, root, evidenceReceiptV3Expectation{
		Contract: "AC-V20-COMMAND-REGISTRY", FixturePath: "acceptance/fixtures/v20_command_registry.json",
		ReceiptPath: fixture.DependencyReceiptPath, ExecutedNotBefore: "2026-07-23T00:00:00Z",
		TrustedBaseGitCommitOID: "42692512b0048f116d660a53fbe6650e243bb4e3",
	})
}

func v21AssertManifest(t *testing.T, root string, fixture v21Fixture, manifest v21Manifest) {
	t.Helper()
	if manifest.SchemaVersion != 1 || manifest.DefaultLocale != fixture.DefaultLocale ||
		manifest.FallbackLocale != fixture.FallbackLocale ||
		!reflect.DeepEqual(manifest.EnabledLocales, fixture.EnabledLocales) {
		t.Fatalf("invalid V21 manifest identity: %+v", manifest)
	}
	wantCatalogs := []v21ManifestCatalog{
		{Locale: "en", Path: "catalogs/en.json"},
		{Locale: "es", Path: "catalogs/es.json"},
	}
	gotCatalogs := append([]v21ManifestCatalog(nil), manifest.Catalogs...)
	sort.Slice(gotCatalogs, func(i, j int) bool { return gotCatalogs[i].Locale < gotCatalogs[j].Locale })
	if !reflect.DeepEqual(gotCatalogs, wantCatalogs) {
		t.Fatalf("V21 manifest catalogs=%v want=%v", gotCatalogs, wantCatalogs)
	}
	manifestDirectory := filepath.Dir(fixture.ManifestPath)
	for _, catalog := range manifest.Catalogs {
		resolved := filepath.ToSlash(filepath.Clean(filepath.Join(manifestDirectory, catalog.Path)))
		if resolved != fixture.CatalogPaths[catalog.Locale] {
			t.Fatalf("V21 manifest catalog %s resolves to %q want %q",
				catalog.Locale, resolved, fixture.CatalogPaths[catalog.Locale])
		}
		v21AssertSafeExistingPath(t, root, resolved)
	}
	active, future := []string{}, []string{}
	seen := map[string]struct{}{}
	activePolicies := map[string]string{
		"cli": "catalog_only", "command_registry": "registry_keys_catalog_values",
		"http": "machine_envelope_catalog_presenter", "mcp": "catalog_only",
		"public_docs": "localized_document_bundle",
		"prompts":     "catalog_only",
		"wizard":      "catalog_only",
	}
	surfaceIDs := make(map[string]struct{}, len(manifest.Surfaces))
	for _, surface := range manifest.Surfaces {
		surfaceIDs[surface.ID] = struct{}{}
	}
	for _, surface := range manifest.Surfaces {
		wantPolicy := activePolicies[surface.ID]
		if surface.State == "future" {
			wantPolicy = "catalog_required_before_activation"
		}
		if strings.TrimSpace(surface.ID) != surface.ID || surface.ID == "" ||
			wantPolicy == "" || surface.LiteralPolicy != wantPolicy {
			t.Fatalf("invalid V21 surface: %+v", surface)
		}
		if _, duplicate := seen[surface.ID]; duplicate {
			t.Fatalf("duplicate V21 surface %q", surface.ID)
		}
		seen[surface.ID] = struct{}{}
		switch surface.State {
		case "active":
			if len(surface.KeySources) == 0 {
				t.Fatalf("active V21 surface %q has no typed key source", surface.ID)
			}
			for _, source := range surface.KeySources {
				switch {
				case strings.HasPrefix(source, "catalog:") && strings.TrimPrefix(source, "catalog:") != "":
				case strings.HasPrefix(source, "surface:"):
					if _, ok := surfaceIDs[strings.TrimPrefix(source, "surface:")]; !ok {
						t.Fatalf("V21 surface %q has unknown dependency %q", surface.ID, source)
					}
				case source == "public_documents" && surface.ID == "public_docs":
				default:
					t.Fatalf("V21 surface %q has invalid typed source %q", surface.ID, source)
				}
			}
			active = append(active, surface.ID)
		case "future":
			if len(surface.KeySources) != 0 {
				t.Fatalf("future V21 surface %q claims nonexistent key sources", surface.ID)
			}
			future = append(future, surface.ID)
		default:
			t.Fatalf("invalid V21 surface state: %+v", surface)
		}
	}
	for _, historical := range fixture.ActiveSurfaceIDs {
		if !containsV21String(active, historical) {
			t.Fatalf("V21 active surface %q regressed: active=%v", historical, active)
		}
	}
	for _, successorOwned := range active {
		if !containsV21String(fixture.ActiveSurfaceIDs, successorOwned) &&
			!containsV21String(fixture.FutureSurfaceIDs, successorOwned) {
			t.Fatalf("unknown successor surface %q activated: active=%v", successorOwned, active)
		}
	}
	for _, remaining := range future {
		if !containsV21String(fixture.FutureSurfaceIDs, remaining) {
			t.Fatalf("unknown V21 future surface %q: future=%v", remaining, future)
		}
	}

	gotDocs := make([]v21PublicDocument, 0, len(manifest.PublicDocuments))
	for _, document := range manifest.PublicDocuments {
		gotDocs = append(gotDocs, v21PublicDocument{
			ID: document.ID, ES: document.Locales["es"], EN: document.Locales["en"],
		})
	}
	sort.Slice(gotDocs, func(i, j int) bool { return gotDocs[i].ID < gotDocs[j].ID })
	if !reflect.DeepEqual(gotDocs, fixture.PublicDocumentPairs) {
		t.Fatalf("V21 manifest public docs=%v want=%v", gotDocs, fixture.PublicDocumentPairs)
	}
	for _, document := range gotDocs {
		v21AssertSafeExistingPath(t, root, document.ES)
		v21AssertSafeExistingPath(t, root, document.EN)
	}
}

func v21AssertRoadmapBoundary(t *testing.T, root string, fixture v21Fixture) {
	t.Helper()
	receiptPresent := v21ExecutionEvidencePresent(t, root, fixture)
	content, err := os.ReadFile(filepath.Join(root, "product", "roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var roadmap struct {
		AcceptanceContracts []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			TestRef string `json:"test_ref"`
			Command string `json:"command"`
			Fixture string `json:"fixture"`
			Receipt string `json:"receipt"`
		} `json:"acceptance_contracts"`
		CapabilityEntries []struct {
			ID           string   `json:"id"`
			OwnerContext string   `json:"owner_context"`
			Status       string   `json:"status"`
			EvidenceRefs []string `json:"evidence_refs"`
		} `json:"capability_entries"`
	}
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V21-I18N" {
			continue
		}
		found = true
		if !receiptPresent {
			if contract.Status != "planned" || contract.TestRef != "planned:acceptance/v21_i18n_test.go" ||
				contract.Command != "planned:go test -mod=vendor -count=1 . ./internal/... ./cmd/orquesta -run '^TestAcceptance$'" ||
				contract.Fixture != "planned:fixtures/v21_i18n" || contract.Receipt != "" {
				t.Fatalf("invalid planned V21 roadmap boundary: %+v", contract)
			}
		} else if contract.Status != "executable" ||
			contract.TestRef != "acceptance/v21_i18n_test.go" ||
			contract.Command != fixture.Command || contract.Fixture != v21FixturePath ||
			contract.Receipt != fixture.ReceiptPath {
			t.Fatalf("invalid executable V21 roadmap boundary: %+v", contract)
		}
	}
	if !found {
		t.Fatal("AC-V21-I18N missing")
	}
	wantEvidence := []string{"acceptance/v21_i18n_test.go", v21FixturePath, fixture.ReceiptPath}
	count := 0
	for _, capability := range roadmap.CapabilityEntries {
		if capability.ID != "UI-18" {
			continue
		}
		count++
		if capability.OwnerContext != "i18n" {
			t.Fatalf("invalid V21 capability owner: %+v", capability)
		}
		if !receiptPresent && (capability.Status != "declared" || len(capability.EvidenceRefs) != 0) {
			t.Fatalf("V21 capability has premature evidence: %+v", capability)
		}
		if receiptPresent && (capability.Status != "accredited" ||
			!reflect.DeepEqual(capability.EvidenceRefs, wantEvidence)) {
			t.Fatalf("V21 capability lacks exact E evidence: %+v", capability)
		}
	}
	if count != 1 {
		t.Fatalf("V21 owned capability count=%d want=1", count)
	}
}

func assertV21ExactSet(t *testing.T, name string, got, want []string) {
	t.Helper()
	actual := append([]string(nil), got...)
	expected := append([]string(nil), want...)
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("V21 %s=%v want=%v", name, actual, expected)
	}
}

func containsV21String(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
