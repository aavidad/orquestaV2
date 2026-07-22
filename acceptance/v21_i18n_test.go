package acceptance_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

const v21FixturePath = "fixtures/v21_i18n.json"

type v21Fixture struct {
	SchemaVersion              int      `json:"schema_version"`
	ContractID                 string   `json:"contract_id"`
	Vertical                   string   `json:"vertical"`
	OwnedCapabilityIDs         []string `json:"owned_capability_ids"`
	AwaitingDependency         string   `json:"awaiting_dependency"`
	AwaitingAcceptanceContract string   `json:"awaiting_acceptance_contract"`
	ImplementationStatus       string   `json:"implementation_status"`
	CatalogOwnerSurfaces       []string `json:"catalog_owner_surfaces"`
	DefaultLocale              string   `json:"default_locale"`
	FallbackLocale             string   `json:"fallback_locale"`
	BCP47Examples              []string `json:"bcp47_examples"`
	Formatters                 []string `json:"formatters"`
	CatalogChecks              []string `json:"catalog_checks"`
	BindingE2E                 struct {
		Status     string   `json:"status"`
		Dependency string   `json:"dependency"`
		Surfaces   []string `json:"surfaces"`
		Assertions []string `json:"assertions"`
	} `json:"binding_e2e"`
	RedGate string `json:"red_gate"`
}

func TestAcceptanceV21I18N(t *testing.T) {
	fixture := loadV21Fixture(t)
	if fixture.SchemaVersion != 1 || fixture.ContractID != "AC-V21-I18N" || fixture.Vertical != "i18n" {
		t.Fatalf("invalid V21 fixture identity: %+v", fixture)
	}
	if !reflect.DeepEqual(fixture.OwnedCapabilityIDs, []string{"UI-18"}) {
		t.Fatalf("V21 capability ownership=%v", fixture.OwnedCapabilityIDs)
	}
	if fixture.AwaitingDependency != "command_registry" || fixture.AwaitingAcceptanceContract != "AC-V20-COMMAND-REGISTRY" || fixture.ImplementationStatus != "awaiting_product" {
		t.Fatalf("V21 dependency contract=%+v", fixture)
	}
	if fixture.DefaultLocale != "es" || fixture.FallbackLocale != "es" {
		t.Fatalf("V21 default/fallback=%q/%q", fixture.DefaultLocale, fixture.FallbackLocale)
	}
	assertV21Set(t, "catalog owner surfaces", fixture.CatalogOwnerSurfaces, []string{"cli", "errors", "notifications", "prompts", "public_docs", "web", "wizard"})
	assertV21Set(t, "BCP-47 examples", fixture.BCP47Examples, []string{"ca-ES-valencia", "en", "es", "zh-Hans"})
	assertV21Set(t, "formatters", fixture.Formatters, []string{"currency", "date", "number", "plural", "timezone"})
	assertV21Set(t, "catalog checks", fixture.CatalogChecks, []string{"extract_public_keys", "locale_parity", "machine_codes_invariant", "missing_key_fails", "unused_key_fails"})
	if fixture.BindingE2E.Status != "awaiting_dependency" || fixture.BindingE2E.Dependency != "command_registry" || len(fixture.BindingE2E.Assertions) != 2 {
		t.Fatalf("V21 deferred binding E2E=%+v", fixture.BindingE2E)
	}
	assertV21Set(t, "deferred binding surfaces", fixture.BindingE2E.Surfaces, []string{"cli", "http", "mcp"})
	if fixture.RedGate != "V21_PRODUCT_PENDING" {
		t.Fatalf("V21 red gate=%q", fixture.RedGate)
	}
	t.Fatalf("%s: product V21 is absent; awaiting_dependency=%s (%s)", fixture.RedGate, fixture.AwaitingDependency, fixture.AwaitingAcceptanceContract)
}

func loadV21Fixture(t *testing.T) v21Fixture {
	t.Helper()
	content, err := os.ReadFile(v21FixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture v21Fixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func assertV21Set(t *testing.T, name string, got, want []string) {
	t.Helper()
	actual := append([]string(nil), got...)
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("V21 %s=%v want=%v", name, actual, want)
	}
}
