package acceptance_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

const v19FixturePath = "fixtures/v19_council.json"

type v19Fixture struct {
	SchemaVersion              int      `json:"schema_version"`
	ContractID                 string   `json:"contract_id"`
	Vertical                   string   `json:"vertical"`
	OwnedCapabilityIDs         []string `json:"owned_capability_ids"`
	AwaitingDependency         string   `json:"awaiting_dependency"`
	AwaitingAcceptanceContract string   `json:"awaiting_acceptance_contract"`
	ImplementationStatus       string   `json:"implementation_status"`
	Policies                   []string `json:"policies"`
	Invariants                 []string `json:"invariants"`
	E2ECases                   []struct {
		Policy          string   `json:"policy"`
		IsolatedRuntime bool     `json:"isolated_runtime"`
		Assertions      []string `json:"assertions"`
	} `json:"e2e_cases"`
	RedGate string `json:"red_gate"`
}

func TestAcceptanceV19Council(t *testing.T) {
	fixture := loadV19Fixture(t)
	if fixture.SchemaVersion != 1 || fixture.ContractID != "AC-V19-COUNCIL" || fixture.Vertical != "council" {
		t.Fatalf("invalid V19 fixture identity: %+v", fixture)
	}
	wantCapabilities := []string{"EVD-07", "GOV-11", "GOV-13", "GOV-14", "STG-06", "STG-08"}
	if got := sortedV19(fixture.OwnedCapabilityIDs); !reflect.DeepEqual(got, wantCapabilities) {
		t.Fatalf("V19 capability ownership=%v want=%v", got, wantCapabilities)
	}
	if fixture.AwaitingDependency != "independent_reviews" || fixture.AwaitingAcceptanceContract != "AC-V18-INDEPENDENT-REVIEWS" || fixture.ImplementationStatus != "awaiting_product" {
		t.Fatalf("V19 dependency contract=%+v", fixture)
	}
	if got := sortedV19(fixture.Policies); !reflect.DeepEqual(got, []string{"auto", "required", "skip_by_operator"}) {
		t.Fatalf("V19 policies=%v", got)
	}
	if len(fixture.Invariants) != 4 || fixture.RedGate != "V19_PRODUCT_PENDING" {
		t.Fatalf("V19 invariant/red-gate contract invalid")
	}
	if len(fixture.E2ECases) != 3 {
		t.Fatalf("V19 E2E cases=%d want=3", len(fixture.E2ECases))
	}
	casePolicies := make([]string, 0, len(fixture.E2ECases))
	for _, scenario := range fixture.E2ECases {
		casePolicies = append(casePolicies, scenario.Policy)
		t.Run(scenario.Policy, func(t *testing.T) {
			if !scenario.IsolatedRuntime || len(scenario.Assertions) != 2 {
				t.Fatalf("V19 %s is not an isolated two-assertion E2E", scenario.Policy)
			}
		})
	}
	if got := sortedV19(casePolicies); !reflect.DeepEqual(got, []string{"auto", "required", "skip_by_operator"}) {
		t.Fatalf("V19 E2E policies=%v", got)
	}
	t.Fatalf("%s: product V19 is absent; awaiting_dependency=%s (%s)", fixture.RedGate, fixture.AwaitingDependency, fixture.AwaitingAcceptanceContract)
}

func loadV19Fixture(t *testing.T) v19Fixture {
	t.Helper()
	content, err := os.ReadFile(v19FixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture v19Fixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func sortedV19(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
