package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"
)

const v01SourceIntegrationFixturePath = "acceptance/fixtures/v01_source_integration.json"

var v01SourceIntegrationCandidateSubjects = []string{
	"acceptance/evidence_support_test.go",
	"acceptance/fixtures/v01_source_integration.json",
	"acceptance/v01_source_integration_test.go",
	"product/roadmap.json",
	"product_roadmap_test.go",
}

type v01SourceIntegrationFixture struct {
	SchemaVersion     int      `json:"schema_version"`
	ContractID        string   `json:"contract_id"`
	Command           string   `json:"command"`
	ReceiptPath       string   `json:"receipt_path"`
	CandidateSubjects []string `json:"candidate_subjects"`
}

func TestAcceptanceV01SourceIntegrationReceipt(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v01SourceIntegrationFixture](
		t,
		filepath.Join(repositoryRoot, v01SourceIntegrationFixturePath),
	)
	v01ValidateSourceIntegrationFixture(t, fixture)
	evidenceAssertReceiptV2(t, repositoryRoot, evidenceReceiptV2Expectation{
		Contract: fixture.ContractID, ValidationCommand: fixture.Command,
		ExecutionArgv: []string{
			"go", "test", "-mod=vendor", "-count=1", ".", "-run", "^TestProductRoadmap.*$",
		},
		OutputPath: "product/evidence/v01_source_integration.output.txt", FixturePath: v01SourceIntegrationFixturePath,
		ReceiptPath: fixture.ReceiptPath, CandidateSubjects: fixture.CandidateSubjects,
		ExecutedNotBefore: "2026-07-14T00:00:00+02:00",
		ExpectedGitHead:   "a301a3bbacd80c1ea2d47422a2964339dcd70980",
	})
}

func v01ValidateSourceIntegrationFixture(t *testing.T, fixture v01SourceIntegrationFixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 ||
		fixture.ContractID != "AC-V01-SOURCE-INTEGRATION" ||
		fixture.Command != "go test -mod=vendor -count=1 . -run '^TestProductRoadmap.*$'" ||
		fixture.ReceiptPath != "product/evidence/v01_source_integration.json" {
		t.Fatalf("invalid V01 fixture identity: %+v", fixture)
	}
	if !slices.Equal(fixture.CandidateSubjects, v01SourceIntegrationCandidateSubjects) {
		t.Fatalf(
			"V01 candidate subjects=%v, want exact sorted set %v",
			fixture.CandidateSubjects,
			v01SourceIntegrationCandidateSubjects,
		)
	}
	for _, subject := range fixture.CandidateSubjects {
		if subject == fixture.ReceiptPath || subject == "product/evidence/v01_source_integration.output.txt" {
			t.Fatalf("V01 receipt/output must stay outside candidate subjects: %s", subject)
		}
	}
}
