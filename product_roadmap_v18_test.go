package orquesta_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type roadmapV18Fixture struct {
	ReceiptSchemaVersion         int      `json:"receipt_schema_version"`
	ContractID                   string   `json:"contract_id"`
	TrustedBaseGitCommitOID      string   `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID string   `json:"product_delta_base_git_commit_oid"`
	Command                      string   `json:"command"`
	ExecutionArgv                []string `json:"execution_argv"`
	OutputPath                   string   `json:"output_path"`
	ReceiptPath                  string   `json:"receipt_path"`
	CandidateSubjects            []string `json:"candidate_subjects"`
	OwnedCapabilityIDs           []string `json:"owned_capability_ids"`
	DependencyVerticals          []string `json:"dependency_verticals"`
	RoadmapAssertions            []string `json:"roadmap_assertions"`
}

func TestProductRoadmapV18ScopeAndExecutableContract(t *testing.T) {
	assertRoadmapV18Lifecycle(t, readRoadmapTestIndex(t), readRoadmapV18Fixture(t))
}

func TestV18EvidenceBelongsOnlyToIndependentReviewCapabilities(t *testing.T) {
	index, fixture := readRoadmapTestIndex(t), readRoadmapV18Fixture(t)
	_, receiptErr := os.Stat(fixture.ReceiptPath)
	receiptExists := receiptErr == nil
	if receiptErr != nil && !os.IsNotExist(receiptErr) {
		t.Fatal(receiptErr)
	}
	owned := roadmapSetOf(fixture.OwnedCapabilityIDs...)
	evidence := roadmapSetOf("acceptance/v18_independent_reviews_test.go", "acceptance/fixtures/v18_independent_reviews.json", fixture.ReceiptPath, fixture.OutputPath)
	wantEvidence := []string{"acceptance/v18_independent_reviews_test.go", "acceptance/fixtures/v18_independent_reviews.json", fixture.ReceiptPath}
	assertRoadmapEvidenceExclusive(t, index.document.CapabilityEntries, owned, evidence, func(t *testing.T, entry roadmapEntry) {
		if receiptExists && (entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
			t.Errorf("owned V18 capability %s has invalid post-E evidence: status=%q evidence=%v", entry.ID, entry.Status, entry.EvidenceRefs)
		}
		if !receiptExists && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
			t.Errorf("owned V18 capability %s has premature pre-E evidence: status=%q evidence=%v", entry.ID, entry.Status, entry.EvidenceRefs)
		}
	})
}

func assertRoadmapV18Lifecycle(t *testing.T, index roadmapTestIndex, fixture roadmapV18Fixture) {
	t.Helper()
	if fixture.ReceiptSchemaVersion != 3 || fixture.ContractID != "AC-V18-INDEPENDENT-REVIEWS" || fixture.TrustedBaseGitCommitOID != "4428f46dd6b48659a4fb871a66cb72927f41cb93" || fixture.ProductDeltaBaseGitCommitOID != "e311a97e4fdcafcf81c6114dbd904af4ed9293ad" || fixture.Command == "" || len(fixture.ExecutionArgv) != 3 || fixture.OutputPath == "" || fixture.ReceiptPath == "" || len(fixture.CandidateSubjects) == 0 || !reflect.DeepEqual(fixture.DependencyVerticals, []string{"controls", "workspace_git", "test_attestor"}) || len(fixture.RoadmapAssertions) != 3 {
		t.Fatalf("invalid V18 roadmap fixture: %+v", fixture)
	}
	var owned []string
	for _, entry := range index.document.CapabilityEntries {
		if entry.OwnerContext == "independent_reviews" && entry.Decision == "accept" {
			owned = append(owned, entry.ID)
		}
	}
	sort.Strings(owned)
	wantOwned := append([]string(nil), fixture.OwnedCapabilityIDs...)
	sort.Strings(wantOwned)
	if !reflect.DeepEqual(owned, wantOwned) {
		t.Fatalf("V18 accepted ownership = %v, want exact %v", owned, wantOwned)
	}
	_, receiptErr := os.Stat(fixture.ReceiptPath)
	receiptExists := receiptErr == nil
	if receiptErr != nil && !os.IsNotExist(receiptErr) {
		t.Fatal(receiptErr)
	}
	contract := index.contracts[fixture.ContractID]
	for _, id := range fixture.OwnedCapabilityIDs {
		entry := index.entries[id]
		if !reflect.DeepEqual(entry.Dependencies, fixture.DependencyVerticals) {
			t.Errorf("V18 capability %s dependencies=%v, want=%v", id, entry.Dependencies, fixture.DependencyVerticals)
		}
		if receiptExists && entry.Status != "accredited" {
			t.Errorf("V18 capability %s lacks post-E accreditation: %#v", id, entry)
		}
		if !receiptExists && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
			t.Errorf("V18 capability %s is not exactly deferred before E: %#v", id, entry)
		}
	}
	if receiptExists {
		if contract.Status != "executable" || contract.TestRef != "acceptance/v18_independent_reviews_test.go" || contract.Command != fixture.Command || contract.Fixture != "acceptance/fixtures/v18_independent_reviews.json" || contract.Receipt != fixture.ReceiptPath || !reflect.DeepEqual(contract.Assertions, fixture.RoadmapAssertions) {
			t.Fatalf("invalid V18 post-E executable contract: %#v", contract)
		}
	} else if contract.Status != "planned" || !strings.HasPrefix(contract.TestRef, "planned:") || !strings.HasPrefix(contract.Command, "planned:") || !strings.HasPrefix(contract.Fixture, "planned:") || contract.Receipt != "" {
		t.Fatalf("invalid V18 pre-E planned contract: %#v", contract)
	}
}

func readRoadmapV18Fixture(t *testing.T) roadmapV18Fixture {
	t.Helper()
	content, err := os.ReadFile("acceptance/fixtures/v18_independent_reviews.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture roadmapV18Fixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}
