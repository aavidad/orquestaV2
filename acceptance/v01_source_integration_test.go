package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"
)

const v01SourceIntegrationFixturePath = "acceptance/fixtures/v01_source_integration.json"

var v01SourceIntegrationCandidateSubjects = []string{
	"acceptance/evidence_protocol_test.go",
	"acceptance/evidence_support_test.go",
	"acceptance/fixtures/v01_source_integration.json",
	"acceptance/v01_source_integration_test.go",
	"product/roadmap.json",
	"product_roadmap_test.go",
}

type v01SourceIntegrationFixture struct {
	SchemaVersion           int      `json:"schema_version"`
	ReceiptSchemaVersion    int      `json:"receipt_schema_version"`
	ContractID              string   `json:"contract_id"`
	TrustedBaseGitCommitOID string   `json:"trusted_base_git_commit_oid"`
	Command                 string   `json:"command"`
	ExecutionArgv           []string `json:"execution_argv"`
	OutputPath              string   `json:"output_path"`
	ReceiptPath             string   `json:"receipt_path"`
	CandidateSubjects       []string `json:"candidate_subjects"`
}

func TestAcceptanceV01SourceIntegrationReceipt(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	evidenceAssertReceiptV3(t, repositoryRoot, evidenceReceiptV3Expectation{
		Contract:                "AC-V01-SOURCE-INTEGRATION",
		FixturePath:             v01SourceIntegrationFixturePath,
		ReceiptPath:             "product/evidence/v01_source_integration.json",
		ExecutedNotBefore:       "2026-07-14T00:00:00+02:00",
		TrustedBaseGitCommitOID: "a301a3bbacd80c1ea2d47422a2964339dcd70980",
	})
}

func TestAcceptanceV01SourceIntegrationContract(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v01SourceIntegrationFixture](t, filepath.Join(repositoryRoot, v01SourceIntegrationFixturePath))
	v01ValidateSourceIntegrationFixture(t, fixture)
}

func v01ValidateSourceIntegrationFixture(t *testing.T, fixture v01SourceIntegrationFixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V01-SOURCE-INTEGRATION" ||
		fixture.TrustedBaseGitCommitOID != "a301a3bbacd80c1ea2d47422a2964339dcd70980" ||
		fixture.Command != "go test -mod=vendor -count=1 . -run '^TestProductRoadmap.*$'" ||
		!slices.Equal(fixture.ExecutionArgv, []string{"go", "test", "-mod=vendor", "-count=1", ".", "-run", "^TestProductRoadmap.*$"}) ||
		fixture.OutputPath != "product/evidence/v01_source_integration.output.txt" ||
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
