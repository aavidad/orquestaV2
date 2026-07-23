package acceptance_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestV18CandidateSubjectsCoverLifecycleDelta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v18Fixture](t, filepath.Join(repositoryRoot, v18FixturePath))
	if fixture.ProductDeltaSealedGitCommitOID == "" {
		v18AssertPrePSubjects(t, repositoryRoot, fixture)
		return
	}
	evidenceAssertCandidateDelta(t, "V18", repositoryRoot, fixture.ProductDeltaBaseGitCommitOID,
		fixture.ProductDeltaSealedGitCommitOID, fixture.CandidateSubjects)
}

func TestV18CandidateSubjectsRetainGenericReceipt(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v18Fixture](t, filepath.Join(repositoryRoot, v18FixturePath))
	if !containsV18Subject(fixture.CandidateSubjects, "product/evidence/real_codex_mcp_e2e.json") {
		t.Fatal("V18 candidate must retain the renewed generic real Codex MCP receipt")
	}
	if containsV18Subject(fixture.CandidateSubjects, fixture.ReceiptPath) ||
		containsV18Subject(fixture.CandidateSubjects, fixture.OutputPath) {
		t.Fatal("V18 receipt/output must remain outside the sealed candidate")
	}
}

func TestV18ReceiptV3Lifecycle(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v18Fixture](t, filepath.Join(repositoryRoot, v18FixturePath))
	_, receiptErr := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.ReceiptPath)))
	_, outputErr := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.OutputPath)))
	if errors.Is(receiptErr, os.ErrNotExist) {
		if !errors.Is(outputErr, os.ErrNotExist) {
			t.Fatalf("V18 output exists without its receipt: %v", outputErr)
		}
		return
	}
	if receiptErr != nil || outputErr != nil {
		t.Fatalf("V18 E evidence pair is incomplete: receipt=%v output=%v", receiptErr, outputErr)
	}
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{
		Contract: "AC-V18-INDEPENDENT-REVIEWS", FixturePath: v18FixturePath,
		ReceiptPath: fixture.ReceiptPath, ExecutedNotBefore: "2026-07-23T00:00:00Z",
		TrustedBaseGitCommitOID: v18ContractBaseGitCommitOID,
	})
}

func TestV18RoadmapAndEvidenceLifecycle(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v18Fixture](t, filepath.Join(repositoryRoot, v18FixturePath))
	_, receiptErr := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.ReceiptPath)))
	receiptPresent := receiptErr == nil
	if receiptErr != nil && !errors.Is(receiptErr, os.ErrNotExist) {
		t.Fatal(receiptErr)
	}
	content, err := os.ReadFile(filepath.Join(repositoryRoot, "product", "roadmap.json"))
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
			Status       string   `json:"status"`
			EvidenceRefs []string `json:"evidence_refs"`
		} `json:"capability_entries"`
	}
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V18-INDEPENDENT-REVIEWS" {
			continue
		}
		if !receiptPresent {
			if contract.Status != "planned" || contract.Receipt != "" ||
				contract.TestRef != "planned:acceptance/v18_independent_reviews_test.go" ||
				contract.Fixture != "planned:fixtures/v18_independent_reviews" ||
				!strings.HasPrefix(contract.Command, "planned:") {
				t.Fatalf("V18 roadmap is invalid before E: %+v", contract)
			}
		} else if contract.Status != "executable" || contract.TestRef != "acceptance/v18_independent_reviews_test.go" ||
			contract.Fixture != v18FixturePath || contract.Receipt != fixture.ReceiptPath || contract.Command != fixture.Command {
			t.Fatalf("V18 roadmap is invalid after E: %+v", contract)
		}
		goto capabilities
	}
	t.Fatal("missing V18 roadmap contract")

capabilities:
	want := map[string]bool{"EVD-06": false, "GOV-12": false, "STG-13": false, "STG-14": false, "STG-16": false}
	for _, entry := range roadmap.CapabilityEntries {
		if _, owned := want[entry.ID]; !owned {
			continue
		}
		if !receiptPresent && (entry.Status != "declared" || len(entry.EvidenceRefs) != 0) {
			t.Fatalf("V18 capability %s is invalid before E: %+v", entry.ID, entry)
		}
		wantEvidence := []string{
			"acceptance/v18_independent_reviews_test.go",
			v18FixturePath,
			fixture.ReceiptPath,
		}
		if receiptPresent && (entry.Status != "accredited" || !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence)) {
			t.Fatalf("V18 capability %s is invalid after E: %+v", entry.ID, entry)
		}
		want[entry.ID] = true
	}
	for id, seen := range want {
		if !seen {
			t.Fatalf("missing V18 owned capability %s", id)
		}
	}
}

func v18ValidationShellBody() string {
	return strings.Join([]string{
		"git diff --check " + v18ProductDeltaBaseGitCommitOID + " HEAD --",
		"go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV17ScopeAndExecutableContract|TestV17EvidenceBelongsOnlyToTestAttestorCapabilities|TestProductRoadmapV18ScopeAndExecutableContract|TestV18EvidenceBelongsOnlyToIndependentReviewCapabilities|TestAcceptanceV16WorkspaceGit|TestAcceptanceV16WorkspaceGitReceipt|TestAcceptanceV17TestAttestor|TestAcceptanceV17TestAttestorReceipt|TestAcceptanceV18IndependentReviews|TestV18CandidateSubjectsCoverLifecycleDelta|TestV18CandidateSubjectsRetainGenericReceipt|TestV18ReceiptV3Lifecycle|TestV18RoadmapAndEvidenceLifecycle|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons)$\"",
		"go test -mod=vendor -count=1 ./internal/review ./internal/application ./internal/adapters/state/sqlite ./internal/bootstrap ./internal/goal ./internal/ports ./internal/interfaces/mcp ./internal/adapters/agent/codex ./internal/adapters/artifact/filesystem ./internal/adapters/attestor/bubblewrap ./internal/adapters/workspace/gitlocal ./cmd/orquesta",
		"timeout --kill-after=10s 240s go test -mod=vendor -race -count=1 -timeout=210s ./internal/application ./internal/adapters/state/sqlite ./internal/bootstrap -run \"^(TestConcurrentReviewClaimsKeepOneDecisionPerRole|TestReviewCrashFrontiersReplayWithoutDuplicateLaunchOrDecision|TestTerminalAttestationReceiptPreventsRestartRerun)$\"",
		"e2e_events=$(CGO_ENABLED=0 go test -mod=vendor -tags=v18_real_e2e -json -count=1 -timeout=180s ./internal/bootstrap -run \"^TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd$\" 2>&1); e2e_status=$?; printf '%s\\n' \"$e2e_events\"; [ \"$e2e_status\" -eq 0 ] && printf '%s\\n' \"$e2e_events\" | grep -F \"\\\"Action\\\":\\\"run\\\"\" | grep -F \"\\\"Package\\\":\\\"orquesta/internal/bootstrap\\\"\" | grep -F \"\\\"Test\\\":\\\"TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd\\\"\" >/dev/null && printf '%s\\n' \"$e2e_events\" | grep -F \"\\\"Action\\\":\\\"pass\\\"\" | grep -F \"\\\"Package\\\":\\\"orquesta/internal/bootstrap\\\"\" | grep -F \"\\\"Test\\\":\\\"TestRealGitSQLiteFilesystemCASBubblewrapIndependentReviewsEndToEnd\\\"\" >/dev/null",
		"GOFLAGS=-mod=vendor go vet ./internal/review ./internal/application ./internal/adapters/state/sqlite ./internal/bootstrap ./internal/goal ./internal/ports ./internal/interfaces/mcp ./internal/adapters/agent/codex ./internal/adapters/artifact/filesystem ./internal/adapters/attestor/bubblewrap ./internal/adapters/workspace/gitlocal ./cmd/orquesta",
	}, " && ")
}

func v18AssertPrePSubjects(t *testing.T, repositoryRoot string, fixture v18Fixture) {
	t.Helper()
	tracked, err := evidenceGit(repositoryRoot, "diff", "--name-only", fixture.ProductDeltaBaseGitCommitOID, "--")
	if err != nil {
		t.Fatal(err)
	}
	untracked, err := evidenceGit(repositoryRoot, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		t.Fatal(err)
	}
	unique := map[string]struct{}{}
	for _, output := range [][]byte{tracked, untracked} {
		for _, relative := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if relative != "" {
				unique[filepath.ToSlash(relative)] = struct{}{}
			}
		}
	}
	dirty := make([]string, 0, len(unique))
	for relative := range unique {
		dirty = append(dirty, relative)
	}
	sort.Strings(dirty)
	if !reflect.DeepEqual(dirty, fixture.CandidateSubjects) {
		t.Fatalf("V18_GATE_PRE_P_SUBJECTS: dirty delta differs from fixture:\ndirty=%v\nfixture=%v", dirty, fixture.CandidateSubjects)
	}
}

func containsV18Subject(subjects []string, wanted string) bool {
	for _, subject := range subjects {
		if subject == wanted {
			return true
		}
	}
	return false
}
