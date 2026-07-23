package acceptance_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	v19CandidateSubjectCount   = 113
	v19CandidateSubjectsSHA256 = "d3ff227abcad64b5eb17ca265e6444268ee514fbc22895d8d74176ba1c77aa37"
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
		v19AssertEvidencePresence(t, repositoryRoot, true, false)
		v19AssertSealManifest(t, repositoryRoot)
		v19AssertRoadmapLifecycle(t, repositoryRoot, "planned", "declared")
	default:
		t.Fatalf("V19 fixture implementation_status=%q; only P/S declarations are valid", fixture.ImplementationStatus)
	}
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
	v19AssertRoadmapLifecycle(t, repositoryRoot, "executable", "accredited")
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

func v19AssertSealManifest(t *testing.T, repositoryRoot string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(v19SealManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SchemaVersion int    `json:"schema_version"`
		ManifestKind  string `json:"manifest_kind"`
		ContractID    string `json:"contract_id"`
		SealState     string `json:"seal_state"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || manifest.ManifestKind != "orquesta.v19_council.seal_manifest.v1" ||
		manifest.ContractID != "AC-V19-COUNCIL" || manifest.SealState != "sealed_unexecuted" {
		t.Fatalf("invalid V19 S manifest: %+v", manifest)
	}
}

func v19AssertRoadmapLifecycle(t *testing.T, repositoryRoot, contractStatus, capabilityStatus string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repositoryRoot, "product/roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var roadmap struct {
		AcceptanceContracts []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"acceptance_contracts"`
		CapabilityEntries []struct {
			OwnerContext string   `json:"owner_context"`
			Status       string   `json:"status"`
			EvidenceRefs []string `json:"evidence_refs"`
		} `json:"capability_entries"`
	}
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID == "AC-V19-COUNCIL" && contract.Status == contractStatus {
			goto capabilities
		}
	}
	t.Fatalf("V19 roadmap contract status is not %q", contractStatus)

capabilities:
	count := 0
	for _, entry := range roadmap.CapabilityEntries {
		if entry.OwnerContext != "council" {
			continue
		}
		count++
		if entry.Status != capabilityStatus {
			t.Fatalf("V19 council capability status=%q want=%q", entry.Status, capabilityStatus)
		}
		if capabilityStatus == "declared" && len(entry.EvidenceRefs) != 0 {
			t.Fatalf("P/S council capability has evidence refs: %#v", entry)
		}
	}
	if count != 6 {
		t.Fatalf("V19 council capability count=%d want=6", count)
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
}
