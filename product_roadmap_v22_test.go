package orquesta_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var roadmapV22OwnedCapabilities = []string{"AGT-01", "AGT-03", "EVD-09", "STG-11"}

type roadmapV22Fixture struct {
	ImplementationStatus string   `json:"implementation_status"`
	Command              string   `json:"command"`
	ReceiptPath          string   `json:"receipt_path"`
	OutputPath           string   `json:"output_path"`
	OwnedCapabilityIDs   []string `json:"owned_capability_ids"`
}

func TestProductRoadmapV22ScopeAndLifecycleContract(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV22Fixture(t)
	if _, err := os.Stat("product/evidence/v21_i18n.json"); err != nil {
		t.Fatalf("V22 dependency receipt is not accredited: %v", err)
	}
	assertRoadmapV22Lifecycle(t, index, fixture)

	vertical := index.verticals["codex_e2e"]
	if !reflect.DeepEqual(vertical.DependsOn,
		[]string{"recovery_backup", "council", "command_registry", "i18n"}) ||
		!reflect.DeepEqual(vertical.AcceptanceContracts, []string{"AC-V22-CODEX-E2E"}) {
		t.Fatalf("V22 vertical has invalid causal boundary: %#v", vertical)
	}
	var owned []string
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == "codex_e2e" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
			if !reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) ||
				!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) {
				t.Errorf("V22 capability %s has invalid causal ownership: %#v", entry.ID, entry)
			}
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, roadmapV22OwnedCapabilities) ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, roadmapV22OwnedCapabilities) {
		t.Fatalf("V22 accepted ownership=%v fixture=%v want=%v",
			owned, fixture.OwnedCapabilityIDs, roadmapV22OwnedCapabilities)
	}
	successor := index.contracts["AC-V23-WIZARD"]
	if successor.Status != "planned" || successor.Receipt != "" {
		t.Fatalf("V22 must not promote successor V23: %#v", successor)
	}
}

func TestV22EvidenceBelongsOnlyToCodexE2ECapabilities(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV22Fixture(t)
	receiptPresent := roadmapV22ReceiptPresent(t, fixture)
	wantEvidence := []string{
		"acceptance/v22_codex_e2e_test.go",
		"acceptance/fixtures/v22_codex_e2e.json",
		fixture.ReceiptPath,
	}
	evidence := roadmapSetOf(append(append([]string(nil), wantEvidence...), fixture.OutputPath)...)
	assertRoadmapEvidenceExclusive(t, index.document.CapabilityEntries,
		roadmapSetOf(roadmapV22OwnedCapabilities...), evidence,
		func(t *testing.T, entry roadmapEntry) {
			if receiptPresent && (entry.Status != "accredited" ||
				!reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
				t.Errorf("%s lacks exact V22 post-E evidence: %#v", entry.ID, entry)
			}
			if !receiptPresent && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
				t.Errorf("%s has premature V22 evidence: %#v", entry.ID, entry)
			}
		})
}

func TestV22AcceptanceCommandCannotPassWithoutOwnedRealTests(t *testing.T) {
	fixture := readRoadmapV22Fixture(t)
	contract := roadmapAcceptanceContract{ID: "AC-V22-CODEX-E2E", Command: fixture.Command}
	assertRoadmapCommandContains(t, contract, nil,
		"./scripts/check_rebuild_write_set.sh 665ef446a32a7f0512255640fa99b2e7edd9f29d",
		"git diff --check 665ef446a32a7f0512255640fa99b2e7edd9f29d HEAD --",
		"./acceptance", "./internal/...", "./internal/adapters/agent/codex",
		"./internal/adapters/state/sqlite",
		"./internal/bootstrap", "./cmd/...", "./sdk/...", "-race",
		"TestCodexChildDeliveryUsesSameExecutionServicePrincipalAfterArtifactPersistence",
		"TestCodexRuntimeUsesCatalogOwnedPrompt",
		"TestExecutionServicePrincipalExactScopeRevocationAndRestart",
		"TestProductionBuildCreatesDurableExecutionAuthorityResolver",
		"-tags=v22_real_e2e",
		"TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP",
		"TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess",
		"\\\"Action\\\":\\\"run\\\"", "\\\"Action\\\":\\\"pass\\\"",
		"GOFLAGS=-mod=vendor go vet",
	)
	if roadmapCommandHasArgument(fixture.Command, "./...") {
		t.Fatalf("V22 gate opens the legacy/global package universe: %q", fixture.Command)
	}
}

func assertRoadmapV22Lifecycle(t *testing.T, index roadmapTestIndex, fixture roadmapV22Fixture) {
	t.Helper()
	receiptPresent := roadmapV22ReceiptPresent(t, fixture)
	contract := index.contracts["AC-V22-CODEX-E2E"]
	assertions := []string{
		"public MCP drives plan DAG mailbox workspace tests reviews and close",
		"four concurrent Goals survive selective stop restart backup and shutdown",
		"no terminal contradiction or owned process remains",
	}
	if receiptPresent {
		if contract.Status != "executable" ||
			contract.TestRef != "acceptance/v22_codex_e2e_test.go" ||
			contract.Command != fixture.Command ||
			contract.Fixture != "acceptance/fixtures/v22_codex_e2e.json" ||
			contract.Receipt != fixture.ReceiptPath ||
			!reflect.DeepEqual(contract.Assertions, assertions) {
			t.Fatalf("V22 roadmap contract is not exact post-E: %#v", contract)
		}
		return
	}
	if contract.Status != "planned" ||
		contract.TestRef != "planned:acceptance/v22_codex_e2e_test.go" ||
		contract.Command != "planned:"+fixture.Command ||
		contract.Fixture != "planned:acceptance/fixtures/v22_codex_e2e.json" ||
		contract.Receipt != "" ||
		!reflect.DeepEqual(contract.Assertions, assertions) {
		t.Fatalf("V22 planned contract is unsafe or does not bind its exact future gate: %#v", contract)
	}
}

func readRoadmapV22Fixture(t *testing.T) roadmapV22Fixture {
	t.Helper()
	content, err := os.ReadFile("acceptance/fixtures/v22_codex_e2e.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture roadmapV22Fixture
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Command == "" ||
		fixture.ReceiptPath != "product/evidence/v22_codex_e2e.json" ||
		fixture.OutputPath != "product/evidence/v22_codex_e2e.output.txt" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, roadmapV22OwnedCapabilities) {
		t.Fatalf("invalid V22 roadmap fixture: %+v", fixture)
	}
	return fixture
}

func roadmapV22ReceiptPresent(t *testing.T, fixture roadmapV22Fixture) bool {
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
		t.Fatal("V22 receipt/output evidence pair is incomplete")
	}
	return present[0]
}
