package orquesta_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var roadmapV21OwnedCapabilities = []string{"UI-18"}

type roadmapV21Fixture struct {
	ImplementationStatus string   `json:"implementation_status"`
	Command              string   `json:"command"`
	ReceiptPath          string   `json:"receipt_path"`
	OutputPath           string   `json:"output_path"`
	OwnedCapabilityIDs   []string `json:"owned_capability_ids"`
}

func TestProductRoadmapV21ScopeAndLifecycleContract(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV21Fixture(t)
	if _, err := os.Stat("product/evidence/v20_command_registry.json"); err != nil {
		t.Fatalf("V21 dependency receipt is not accredited: %v", err)
	}
	assertRoadmapV21Lifecycle(t, index, fixture)

	var owned []string
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == "i18n" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, roadmapV21OwnedCapabilities) ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, roadmapV21OwnedCapabilities) {
		t.Fatalf("V21 accepted ownership=%v fixture=%v want=%v",
			owned, fixture.OwnedCapabilityIDs, roadmapV21OwnedCapabilities)
	}
	vertical := index.verticals["i18n"]
	entry := index.entries["UI-18"]
	if !reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) ||
		!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) {
		t.Fatalf("V21 capability has invalid causal ownership: %#v", entry)
	}
	successor := index.contracts["AC-V22-CODEX-E2E"]
	if _, err := os.Stat("product/evidence/v22_codex_e2e.json"); os.IsNotExist(err) {
		if successor.Status != "planned" || successor.Receipt != "" {
			t.Fatalf("V21 cannot promote successor V22 before its receipt: %#v", successor)
		}
	} else if err != nil {
		t.Fatal(err)
	} else if successor.Status != "executable" ||
		successor.TestRef != "acceptance/v22_codex_e2e_test.go" ||
		successor.Fixture != "acceptance/fixtures/v22_codex_e2e.json" ||
		successor.Receipt != "product/evidence/v22_codex_e2e.json" {
		t.Fatalf("V21 sees a V22 receipt without exact roadmap promotion: %#v", successor)
	}
}

func TestV21EvidenceBelongsOnlyToI18NCapability(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV21Fixture(t)
	receiptPresent := roadmapV21ReceiptPresent(t, fixture)
	wantEvidence := []string{
		"acceptance/v21_i18n_test.go",
		"acceptance/fixtures/v21_i18n.json",
		fixture.ReceiptPath,
	}
	evidence := roadmapSetOf(append(append([]string(nil), wantEvidence...), fixture.OutputPath)...)
	assertRoadmapEvidenceExclusive(t, index.document.CapabilityEntries,
		roadmapSetOf(roadmapV21OwnedCapabilities...), evidence,
		func(t *testing.T, entry roadmapEntry) {
			if receiptPresent && (entry.Status != "accredited" ||
				!reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
				t.Errorf("UI-18 lacks exact post-E evidence: %#v", entry)
			}
			if !receiptPresent && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
				t.Errorf("UI-18 has premature evidence: %#v", entry)
			}
		})
}

func TestV21AcceptanceCommandRunsOwnedConsumersAndRealE2E(t *testing.T) {
	fixture := readRoadmapV21Fixture(t)
	contract := roadmapAcceptanceContract{ID: "AC-V21-I18N", Command: fixture.Command}
	assertRoadmapCommandContains(t, contract, nil,
		"./scripts/check_rebuild_write_set.sh 8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8",
		"git diff --check 8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8 HEAD --",
		"./acceptance", "./internal/i18n", "./internal/interfaces/httpapi",
		"./internal/interfaces/mcp", "./internal/interfaces/cli", "./internal/bootstrap",
		"./cmd/orquesta", "./sdk/commands", "-race",
		"TestCatalogConcurrentReadsAreRaceFree",
		"TestRealHTTPMCPCLIAndI18NParityEndToEnd",
		"\\\"Action\\\":\\\"run\\\"", "\\\"Action\\\":\\\"pass\\\"",
		"GOFLAGS=-mod=vendor go vet",
	)
	if roadmapCommandHasArgument(fixture.Command, "./...") {
		t.Fatalf("V21 gate opens the legacy/global package universe: %q", fixture.Command)
	}
}

func assertRoadmapV21Lifecycle(t *testing.T, index roadmapTestIndex, fixture roadmapV21Fixture) {
	t.Helper()
	receiptPresent := roadmapV21ReceiptPresent(t, fixture)
	contract := index.contracts["AC-V21-I18N"]
	assertions := []string{
		"all public keys have locale parity and Spanish fallback",
		"BCP-47 plurals dates numbers currency and timezone pass",
		"machine codes never change by locale",
	}
	if receiptPresent {
		if contract.Status != "executable" ||
			contract.TestRef != "acceptance/v21_i18n_test.go" ||
			contract.Command != fixture.Command ||
			contract.Fixture != "acceptance/fixtures/v21_i18n.json" ||
			contract.Receipt != fixture.ReceiptPath ||
			!reflect.DeepEqual(contract.Assertions, assertions) {
			t.Fatalf("V21 roadmap contract is not exact post-E: %#v", contract)
		}
	} else if contract.Status != "planned" ||
		contract.TestRef != "planned:acceptance/v21_i18n_test.go" ||
		contract.Command != "planned:go test -mod=vendor -count=1 . ./internal/... ./cmd/orquesta -run '^TestAcceptance$'" ||
		contract.Fixture != "planned:fixtures/v21_i18n" || contract.Receipt != "" ||
		!reflect.DeepEqual(contract.Assertions, assertions) {
		t.Fatalf("V21 roadmap contract must remain non-executable before E: %#v", contract)
	}
}

func readRoadmapV21Fixture(t *testing.T) roadmapV21Fixture {
	t.Helper()
	content, err := os.ReadFile("acceptance/fixtures/v21_i18n.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture roadmapV21Fixture
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Command == "" || fixture.ReceiptPath != "product/evidence/v21_i18n.json" ||
		fixture.OutputPath != "product/evidence/v21_i18n.output.txt" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, roadmapV21OwnedCapabilities) {
		t.Fatalf("invalid V21 roadmap fixture: %+v", fixture)
	}
	return fixture
}

func roadmapV21ReceiptPresent(t *testing.T, fixture roadmapV21Fixture) bool {
	t.Helper()
	present := make([]bool, 0, 2)
	for _, path := range []string{fixture.ReceiptPath, fixture.OutputPath} {
		_, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		present = append(present, err == nil)
	}
	if present[0] != present[1] {
		t.Fatal("V21 receipt/output evidence pair is incomplete")
	}
	return present[0]
}
