package acceptance_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	v19CandidateSubjectCount   = 131
	v19CandidateSubjectsSHA256 = "9a3c3a55e9e0a2c5f6bada899d35f4cf9b9e3fbc1b52bc707053577661f23d85"
)

func TestV19PSELifecycleIsExact(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v19Fixture](t, v19FixtureRepositoryPath(repositoryRoot))
	v19AssertLifecycleEnvelope(t, fixture)

	switch fixture.ImplementationStatus {
	case "implemented_unsealed":
		if fixture.LifecycleGate != "V19_IMPLEMENTED_UNSEALED" ||
			fixture.ProductDeltaSealedGitCommitOID != "" ||
			fixture.SealStatus != "p_implemented_unsealed_pending_seal" ||
			fixture.SealNote != "P has product and tests only; S manifest and E receipt/output are absent and no PASS is implied." {
			t.Fatalf("V19 P state is not exact: %+v", fixture)
		}
		v19AssertEvidencePresence(t, repositoryRoot, false, false)
		v19AssertRoadmapLifecycle(t, repositoryRoot, "planned", "declared")
	case "sealed_unexecuted":
		if fixture.LifecycleGate != "V19_SEALED_UNEXECUTED" ||
			fixture.ProductDeltaSealedGitCommitOID == "" ||
			fixture.SealStatus != "s_product_delta_sealed_pending_execution" {
			t.Fatalf("V19 S declaration is not exact: %+v", fixture)
		}
		hasExecution := v19ExecutionEvidencePresent(t, repositoryRoot)
		v19AssertEvidencePresence(t, repositoryRoot, true, hasExecution)
		v19AssertSealManifest(t, repositoryRoot, fixture)
		if hasExecution {
			v19AssertRoadmapLifecycle(t, repositoryRoot, "executable", "accredited")
		} else {
			v19AssertRoadmapLifecycle(t, repositoryRoot, "planned", "declared")
		}
	default:
		t.Fatalf("V19 fixture implementation_status=%q; only P/S declarations are valid", fixture.ImplementationStatus)
	}
}

func v19ExecutionEvidencePresent(t *testing.T, repositoryRoot string) bool {
	t.Helper()
	present := make([]bool, 0, 2)
	for _, path := range []string{v19ReceiptPath, v19OutputPath} {
		_, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(path)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		present = append(present, err == nil)
	}
	if present[0] != present[1] {
		t.Fatal("V19 execution evidence must contain both receipt and output or neither")
	}
	return present[0]
}

// TestV19EReceiptLifecycle is deliberately independent of fixture mutation:
// the receipt is bound to the sealed S fixture blob, so E derives from a valid
// V3 receipt/output plus the roadmap transition rather than rewriting S.
func TestV19EReceiptLifecycle(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v19Fixture](t, v19FixtureRepositoryPath(repositoryRoot))
	if fixture.ImplementationStatus != "sealed_unexecuted" {
		return // P has no receipt by definition; S is the only receipt source.
	}
	if _, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(fixture.ReceiptPath))); errors.Is(err, os.ErrNotExist) {
		return // exact S: execution has not happened yet.
	} else if err != nil {
		t.Fatal(err)
	}
	v19AssertEvidencePresence(t, repositoryRoot, true, true)
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{
		Contract: "AC-V19-COUNCIL", FixturePath: "acceptance/" + v19FixturePath, ReceiptPath: fixture.ReceiptPath,
		ExecutedNotBefore: "2026-07-23T00:00:00Z", TrustedBaseGitCommitOID: fixture.TrustedBaseGitCommitOID,
	})
	v19AssertReceiptBindsSealManifest(t, repositoryRoot, fixture)
	v19AssertRoadmapLifecycle(t, repositoryRoot, "executable", "accredited")
}

func v19AssertReceiptBindsSealManifest(t *testing.T, repositoryRoot string, fixture v19Fixture) {
	t.Helper()
	receipt := evidenceDecodeStrictJSON[evidenceReceiptV3](t, filepath.Join(repositoryRoot, filepath.FromSlash(fixture.ReceiptPath)))
	if err := evidenceGitAncestor(repositoryRoot, fixture.ProductDeltaSealedGitCommitOID, receipt.SealedSource.GitCommitOID); err != nil {
		t.Fatalf("V19 executed S commit does not descend from sealed P: %v", err)
	}
	sealedEntry, err := evidenceGitBlobAt(repositoryRoot, receipt.SealedSource.GitCommitOID, v19SealManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(v19SealManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sealedEntry.Content, current) {
		t.Fatal("V19 receipt sealed tree does not bind current Council seal manifest")
	}
}

func v19AssertLifecycleEnvelope(t *testing.T, fixture v19Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.TrustedBaseGitCommitOID != fixture.ProductDeltaBaseGitCommitOID ||
		fixture.SealStatus == "" || fixture.SealNote == "" {
		t.Fatalf("invalid V19 P/S/E envelope: %+v", fixture)
	}
}

func v19AssertEvidencePresence(t *testing.T, repositoryRoot string, wantSeal, wantExecution bool) {
	t.Helper()
	for _, expectation := range []struct {
		path string
		want bool
	}{
		{v19SealManifestPath, wantSeal},
		{v19ReceiptPath, wantExecution},
		{v19OutputPath, wantExecution},
	} {
		_, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(expectation.path)))
		if expectation.want && err != nil {
			t.Fatalf("V19 expected evidence %s: %v", expectation.path, err)
		}
		if !expectation.want && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("V19 unexpected evidence %s: %v", expectation.path, err)
		}
	}
}

func v19AssertSealManifest(t *testing.T, repositoryRoot string, fixture v19Fixture) {
	t.Helper()
	v19AssertSealRepositoryBindings(t, repositoryRoot, fixture)
}

func v19AssertRoadmapLifecycle(t *testing.T, repositoryRoot, contractStatus, capabilityStatus string) {
	t.Helper()
	fixture := evidenceDecodeStrictJSON[v19Fixture](t, v19FixtureRepositoryPath(repositoryRoot))
	content, err := os.ReadFile(filepath.Join(repositoryRoot, "product/roadmap.json"))
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
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V19-COUNCIL" {
			continue
		}
		if contract.Status != contractStatus {
			t.Fatalf("V19 roadmap contract status=%q want=%q", contract.Status, contractStatus)
		}
		if contractStatus == "planned" && (contract.TestRef != "planned:acceptance/v19_council_test.go" ||
			contract.Command != "planned:go test -mod=vendor -count=1 . ./acceptance -run '^TestAcceptanceV19Council$'" ||
			contract.Fixture != "planned:fixtures/v19_council" || contract.Receipt != "") {
			t.Fatalf("V19 planned roadmap contract metadata invalid: %+v", contract)
		}
		if contractStatus == "executable" && (contract.TestRef != "acceptance/v19_council_test.go" ||
			contract.Command != fixture.Command || contract.Fixture != "acceptance/fixtures/v19_council.json" ||
			contract.Receipt != v19ReceiptPath) {
			t.Fatalf("V19 executable roadmap contract metadata invalid: %+v", contract)
		}
		goto capabilities
	}
	t.Fatal("missing V19 roadmap contract")

capabilities:
	wantIDs := map[string]bool{"EVD-07": false, "GOV-11": false, "GOV-13": false, "GOV-14": false, "STG-06": false, "STG-08": false}
	wantEvidence := []string{"acceptance/v19_council_test.go", "acceptance/fixtures/v19_council.json", v19ReceiptPath}
	for _, entry := range roadmap.CapabilityEntries {
		if entry.OwnerContext != "council" {
			continue
		}
		if _, found := wantIDs[entry.ID]; !found || wantIDs[entry.ID] {
			t.Fatalf("unexpected or duplicate V19 council capability %q", entry.ID)
		}
		wantIDs[entry.ID] = true
		if entry.Status != capabilityStatus {
			t.Fatalf("V19 council capability status=%q want=%q", entry.Status, capabilityStatus)
		}
		if capabilityStatus == "declared" && len(entry.EvidenceRefs) != 0 {
			t.Fatalf("P/S council capability has evidence refs: %#v", entry)
		}
		if capabilityStatus == "accredited" && !reflect.DeepEqual(entry.EvidenceRefs, wantEvidence) {
			t.Fatalf("E council capability %s evidence=%v want=%v", entry.ID, entry.EvidenceRefs, wantEvidence)
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Fatalf("missing V19 council capability %s", id)
		}
	}
}

func TestV19CandidateSubjectsDeclareExactBaseThroughP2Delta(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v19Fixture](t, v19FixtureRepositoryPath(repositoryRoot))
	if len(fixture.CandidateSubjects) != v19CandidateSubjectCount {
		t.Fatalf("V19 P2 candidate count=%d want=%d", len(fixture.CandidateSubjects), v19CandidateSubjectCount)
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath); err != nil {
		t.Fatal(err)
	}
	for _, subject := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(subject))); err != nil {
			t.Fatalf("V19 candidate subject %q is not readable: %v", subject, err)
		}
	}
	digest := sha256.Sum256([]byte(strings.Join(fixture.CandidateSubjects, "\n") + "\n"))
	if got := hex.EncodeToString(digest[:]); got != v19CandidateSubjectsSHA256 {
		t.Fatalf("V19 P2 candidate subjects digest=%s want=%s", got, v19CandidateSubjectsSHA256)
	}
	if fixture.ProductDeltaSealedGitCommitOID != "" {
		evidenceAssertCandidateDelta(t, "V19", repositoryRoot, fixture.ProductDeltaBaseGitCommitOID,
			fixture.ProductDeltaSealedGitCommitOID, fixture.CandidateSubjects)
		return
	}
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
		for _, path := range v19NonEmptyLines(output) {
			unique[filepath.ToSlash(path)] = struct{}{}
		}
	}
	dirty := make([]string, 0, len(unique))
	for path := range unique {
		dirty = append(dirty, path)
	}
	sort.Strings(dirty)
	if !reflect.DeepEqual(dirty, fixture.CandidateSubjects) {
		t.Fatalf("V19_GATE_PRE_P_SUBJECTS: dirty=%v fixture=%v", dirty, fixture.CandidateSubjects)
	}
}
