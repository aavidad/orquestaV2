package orquesta_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var roadmapV20OwnedCapabilities = []string{"GOV-17", "UI-02"}

type roadmapV20Fixture struct {
	ImplementationStatus string   `json:"implementation_status"`
	Command              string   `json:"command"`
	ReceiptPath          string   `json:"receipt_path"`
	OutputPath           string   `json:"output_path"`
	OwnedCapabilityIDs   []string `json:"owned_capability_ids"`
}

func TestProductRoadmapV20ScopeAndLifecycleContract(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV20Fixture(t)
	if _, err := os.Stat("product/evidence/v19_council.json"); err != nil {
		t.Fatalf("V20 dependency receipt is not accredited: %v", err)
	}
	assertRoadmapV20Lifecycle(t, index, fixture)

	var owned []string
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == "command_registry" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	if !reflect.DeepEqual(owned, roadmapV20OwnedCapabilities) ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, roadmapV20OwnedCapabilities) {
		t.Fatalf("V20 accepted ownership=%v fixture=%v want=%v",
			owned, fixture.OwnedCapabilityIDs, roadmapV20OwnedCapabilities)
	}

	vertical := index.verticals["command_registry"]
	for _, id := range roadmapV20OwnedCapabilities {
		entry := index.entries[id]
		if !reflect.DeepEqual(entry.Dependencies, vertical.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, vertical.AcceptanceContracts) {
			t.Errorf("V20 capability %s has invalid causal ownership: %#v", id, entry)
		}
	}
	successor := index.contracts["AC-V21-I18N"]
	if _, err := os.Stat("product/evidence/v21_i18n.json"); os.IsNotExist(err) {
		if successor.Status != "planned" || successor.Receipt != "" {
			t.Fatalf("V20 cannot promote successor V21 before its receipt: %#v", successor)
		}
	} else if err != nil {
		t.Fatal(err)
	} else if successor.Status != "executable" ||
		successor.Receipt != "product/evidence/v21_i18n.json" {
		t.Fatalf("V20 sees a V21 receipt without exact roadmap promotion: %#v", successor)
	}
}

func TestV20EvidenceBelongsOnlyToCommandRegistryCapabilities(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV20Fixture(t)
	receiptPresent := roadmapV20ReceiptPresent(t, fixture)
	wantEvidence := []string{
		"acceptance/v20_command_registry_test.go",
		"acceptance/fixtures/v20_command_registry.json",
		fixture.ReceiptPath,
	}
	evidence := roadmapSetOf(append(append([]string(nil), wantEvidence...), fixture.OutputPath)...)
	assertRoadmapEvidenceExclusive(t, index.document.CapabilityEntries,
		roadmapSetOf(roadmapV20OwnedCapabilities...), evidence,
		func(t *testing.T, entry roadmapEntry) {
			if receiptPresent && (entry.Status != "accredited" ||
				!reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
				t.Errorf("owned V20 capability %s lacks exact post-E evidence: %#v", entry.ID, entry)
			}
			if !receiptPresent && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
				t.Errorf("owned V20 capability %s has premature evidence: %#v", entry.ID, entry)
			}
		})
}

func TestV20AcceptanceCommandRunsOwnedConsumersAndExactE2E(t *testing.T) {
	fixture := readRoadmapV20Fixture(t)
	contract := roadmapAcceptanceContract{ID: "AC-V20-COMMAND-REGISTRY", Command: fixture.Command}
	assertRoadmapCommandContains(t, contract, nil,
		"./scripts/check_rebuild_write_set.sh 42692512b0048f116d660a53fbe6650e243bb4e3",
		"git diff --check 42692512b0048f116d660a53fbe6650e243bb4e3 HEAD --",
		"./acceptance", "./internal/...", "./cmd/...", "./sdk/...",
		"timeout --kill-after=10s", "-race",
		"TestConcurrentReplayAppendsOneExactTerminalWithoutRawData",
		"TestCommandDispatcherCrashAfterMutatingHandlerReplaysApplicationReceiptWithoutSecondEffect",
		"TestSQLiteCommandAuditConcurrentExactReplayConverges",
		"TestRealHTTPMCPCLIAndSDKParityEndToEnd",
		"TestRealCommandAuditSurvivesRestartAndLifecycleRemainsApplicationOwned",
		"\\\"Action\\\":\\\"run\\\"", "\\\"Action\\\":\\\"pass\\\"", "GOFLAGS=-mod=vendor go vet",
	)
	if roadmapCommandHasArgument(fixture.Command, "./...") {
		t.Fatalf("V20 gate opens the legacy/global package universe: %q", fixture.Command)
	}
}

func assertRoadmapV20Lifecycle(t *testing.T, index roadmapTestIndex, fixture roadmapV20Fixture) {
	t.Helper()
	receiptPresent := roadmapV20ReceiptPresent(t, fixture)
	contract := index.contracts["AC-V20-COMMAND-REGISTRY"]
	assertions := []string{
		"one command definition generates HTTP MCP and CLI bindings",
		"auth schema i18n and error codes are semantically identical",
		"no interface writes lifecycle directly",
	}
	if receiptPresent {
		if contract.Status != "executable" ||
			contract.TestRef != "acceptance/v20_command_registry_test.go" ||
			contract.Command != fixture.Command ||
			contract.Fixture != "acceptance/fixtures/v20_command_registry.json" ||
			contract.Receipt != fixture.ReceiptPath ||
			!reflect.DeepEqual(contract.Assertions, assertions) {
			t.Fatalf("V20 roadmap contract is not exact post-E: %#v", contract)
		}
	} else if contract.Status != "planned" ||
		contract.TestRef != "planned:acceptance/v20_command_registry_test.go" ||
		contract.Command != "planned:go test -mod=vendor -count=1 . ./internal/... ./cmd/orquesta -run '^TestAcceptance$'" ||
		contract.Fixture != "planned:fixtures/v20_command_registry" || contract.Receipt != "" ||
		!reflect.DeepEqual(contract.Assertions, assertions) {
		t.Fatalf("V20 roadmap contract must remain non-executable before E: %#v", contract)
	}

	wantEvidence := []string{
		"acceptance/v20_command_registry_test.go",
		"acceptance/fixtures/v20_command_registry.json",
		fixture.ReceiptPath,
	}
	for _, id := range roadmapV20OwnedCapabilities {
		entry := index.entries[id]
		if receiptPresent && (entry.Status != "accredited" ||
			!reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
			t.Fatalf("V20 capability %s is not accredited exactly: %#v", id, entry)
		}
		if !receiptPresent && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
			t.Fatalf("V20 capability %s has premature evidence: %#v", id, entry)
		}
	}
}

func readRoadmapV20Fixture(t *testing.T) roadmapV20Fixture {
	t.Helper()
	content, err := os.ReadFile("acceptance/fixtures/v20_command_registry.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture roadmapV20Fixture
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Command == "" || fixture.ReceiptPath != "product/evidence/v20_command_registry.json" ||
		fixture.OutputPath != "product/evidence/v20_command_registry.output.txt" ||
		!reflect.DeepEqual(fixture.OwnedCapabilityIDs, roadmapV20OwnedCapabilities) {
		t.Fatalf("invalid V20 roadmap fixture: %+v", fixture)
	}
	return fixture
}

func roadmapV20ReceiptPresent(t *testing.T, fixture roadmapV20Fixture) bool {
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
		t.Fatal("V20 receipt/output evidence pair is incomplete")
	}
	return present[0]
}
