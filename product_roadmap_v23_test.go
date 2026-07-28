package orquesta_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

const roadmapV23ContractID = "AC-V23-WIZARD"

type roadmapV23Fixture struct {
	ContractID     string                   `json:"contract_id"`
	Classification roadmapV23Classification `json:"capability_classification"`
	SealStatus     string                   `json:"seal_status"`
	ReceiptPath    string                   `json:"receipt_path"`
}

type roadmapV23Classification struct {
	Candidate []string `json:"candidate"`
	Partial   []string `json:"partial"`
	Pending   []string `json:"pending"`
	Rejected  []string `json:"rejected"`
}

func TestProductRoadmapV23ScopeIsExhaustiveAndDeclared(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV23Fixture(t)
	wantIDs := roadmapV23ExpectedCapabilityIDs()
	linkedIDs := make([]string, 0, len(wantIDs))
	for _, entry := range index.document.CapabilityEntries {
		if reflect.DeepEqual(entry.AcceptanceContracts, []string{roadmapV23ContractID}) {
			linkedIDs = append(linkedIDs, entry.ID)
		}
	}
	sort.Strings(linkedIDs)
	if !reflect.DeepEqual(linkedIDs, wantIDs) {
		t.Fatalf("capabilities linked to %s=%v want exact=%v",
			roadmapV23ContractID, linkedIDs, wantIDs)
	}

	classified := roadmapV23ClassifiedIDs(t, fixture.Classification)
	if !reflect.DeepEqual(classified, wantIDs) {
		t.Fatalf("fixture classification=%v want exact roadmap scope=%v", classified, wantIDs)
	}

	rejected := roadmapSetOf("WIZ-12", "WIZ-14")
	wizard := index.verticals["wizard"]
	for _, id := range wantIDs {
		entry := index.entries[id]
		if entry.OwnerContext != wizard.ID ||
			!reflect.DeepEqual(entry.Dependencies, wizard.DependsOn) ||
			!reflect.DeepEqual(entry.AcceptanceContracts, wizard.AcceptanceContracts) ||
			entry.Status != "declared" || len(entry.EvidenceRefs) != 0 {
			t.Errorf("V23 capability %s is promoted, evidenced, or causally detached: %#v", id, entry)
		}
		if _, isRejected := rejected[id]; isRejected {
			if entry.Decision != "reject" || entry.ReleaseTarget != "excluded" ||
				entry.CutoverRequired {
				t.Errorf("rejected V23 capability %s has invalid release semantics: %#v", id, entry)
			}
			continue
		}
		if entry.Decision != "accept" || entry.ReleaseTarget != "total_v1" ||
			!entry.CutoverRequired {
			t.Errorf("accepted V23 capability %s has invalid release semantics: %#v", id, entry)
		}
	}
}

func TestProductRoadmapV23DependenciesAreReadyWithoutSealingWizard(t *testing.T) {
	index := readRoadmapTestIndex(t)
	assertRoadmapV22Lifecycle(t, index, readRoadmapV22Fixture(t))

	v22 := index.verticals["codex_e2e"]
	if v22.Sequence != 22 ||
		!reflect.DeepEqual(v22.DependsOn,
			[]string{"recovery_backup", "council", "command_registry", "i18n"}) ||
		!reflect.DeepEqual(v22.AcceptanceContracts, []string{"AC-V22-CODEX-E2E"}) {
		t.Fatalf("V22 vertical has drifted dependencies: %#v", v22)
	}
	wizard := index.verticals["wizard"]
	if wizard.Sequence != 23 ||
		!reflect.DeepEqual(wizard.DependsOn,
			[]string{"intent_appspec", "goal_dag_phases", "command_registry", "i18n"}) ||
		!reflect.DeepEqual(wizard.AcceptanceContracts, []string{roadmapV23ContractID}) {
		t.Fatalf("V23 Wizard vertical has drifted dependencies: %#v", wizard)
	}
	for _, dependencyID := range wizard.DependsOn {
		dependency, found := index.verticals[dependencyID]
		if !found || dependency.Sequence >= wizard.Sequence ||
			len(dependency.AcceptanceContracts) != 1 {
			t.Fatalf("V23 dependency %q is missing or non-causal: %#v", dependencyID, dependency)
		}
		contract := index.contracts[dependency.AcceptanceContracts[0]]
		if contract.Status != "executable" || contract.Receipt == "" {
			t.Fatalf("V23 dependency %q is not accredited by an executable receipt: %#v",
				dependencyID, contract)
		}
		if _, err := os.Stat(contract.Receipt); err != nil {
			t.Fatalf("V23 dependency %q receipt %q: %v", dependencyID, contract.Receipt, err)
		}
	}
}

func TestProductRoadmapV23CandidateVocabularyRemainsLocalAndUnsealed(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV23Fixture(t)
	contract := index.contracts[roadmapV23ContractID]
	if contract.Status != "planned" ||
		contract.TestRef != "planned:acceptance/v23_wizard_test.go" ||
		contract.Command != "planned:go test -mod=vendor -count=1 . ./internal/... ./cmd/orquesta -run '^TestAcceptance$'" ||
		contract.Fixture != "planned:fixtures/v23_wizard" ||
		contract.Receipt != "" {
		t.Fatalf("V23 roadmap contract was promoted or drifted: %#v", contract)
	}
	if fixture.ContractID != roadmapV23ContractID ||
		fixture.SealStatus != "not_sealed" || fixture.ReceiptPath != "" {
		t.Fatalf("V23 fixture claims a seal or receipt: %+v", fixture)
	}
	for _, localStatus := range []string{"candidate", "partial", "pending", "rejected"} {
		for _, canonicalStatus := range index.document.StatusVocabulary {
			if localStatus == canonicalStatus {
				t.Fatalf("local V23 classification %q leaked into roadmap status vocabulary", localStatus)
			}
		}
	}
	for _, id := range fixture.Classification.Candidate {
		entry := index.entries[id]
		if entry.Status != "declared" || len(entry.EvidenceRefs) != 0 {
			t.Errorf("local candidate %s promoted roadmap state: %#v", id, entry)
		}
	}
	if _, err := os.Lstat("product/evidence/v23_wizard.json"); !os.IsNotExist(err) {
		if err == nil {
			t.Fatal("product/evidence/v23_wizard.json exists before V23 is sealed")
		}
		t.Fatalf("inspect V23 evidence path: %v", err)
	}
}

func readRoadmapV23Fixture(t *testing.T) roadmapV23Fixture {
	t.Helper()
	content, err := os.ReadFile("acceptance/fixtures/v23_wizard.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture roadmapV23Fixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func roadmapV23ExpectedCapabilityIDs() []string {
	ids := make([]string, 0, 29)
	for number := 1; number <= 25; number++ {
		ids = append(ids, "WIZ-"+roadmapTwoDigits(number))
	}
	ids = append(ids, "STG-01", "STG-03", "STG-07", "UI-05")
	sort.Strings(ids)
	return ids
}

func roadmapV23ClassifiedIDs(t *testing.T, classification roadmapV23Classification) []string {
	t.Helper()
	seen := make(map[string]string, 29)
	for class, ids := range map[string][]string{
		"candidate": classification.Candidate,
		"partial":   classification.Partial,
		"pending":   classification.Pending,
		"rejected":  classification.Rejected,
	} {
		for _, id := range ids {
			if previous, duplicate := seen[id]; duplicate {
				t.Fatalf("capability %s appears in both %s and %s", id, previous, class)
			}
			seen[id] = class
		}
	}
	if len(seen) != 29 {
		t.Fatalf("fixture classifies %d unique V23 IDs, want 29", len(seen))
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
