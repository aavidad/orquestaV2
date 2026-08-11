package acceptance_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const v31ArtifactParityFixturePath = "acceptance/fixtures/v31_artifact_parity.json"

type v31ArtifactParityFixture struct {
	SchemaVersion    int      `json:"schema_version"`
	FixtureID        string   `json:"fixture_id"`
	ContractID       string   `json:"contract_id"`
	CapabilityIDs    []string `json:"capability_ids"`
	Status           string   `json:"status"`
	SuitePath        string   `json:"suite_path"`
	Adapters         []string `json:"adapters"`
	SharedInvariants []string `json:"shared_invariants"`
	DeferredGates    []string `json:"deferred_gates"`
}

func TestAcceptanceV31ArtifactParityFoundation(t *testing.T) {
	fixture := readV31ArtifactParityFixture(t)
	if fixture.SchemaVersion != 1 || fixture.FixtureID != "v31_artifact_parity_foundation" ||
		fixture.ContractID != "AC-V31-POSTGRES-S3-MULTIHOST" ||
		!reflect.DeepEqual(fixture.CapabilityIDs, []string{"OPS-13"}) ||
		fixture.Status != "implemented_foundation_not_accredited" ||
		fixture.SuitePath != "internal/adapters/artifact/contracttest" ||
		!reflect.DeepEqual(fixture.Adapters, []string{"filesystem_project_aware", "s3_fake_injected"}) {
		t.Fatalf("invalid V31 artifact parity identity: %+v", fixture)
	}
	wantInvariants := []string{
		"content_addressed_put",
		"exact_replay_and_media_type_conflict",
		"content_and_causal_metadata_tamper_rejection",
		"bounded_read_and_expected_size_validation",
		"opaque_project_isolation",
		"restart_recovery",
		"concurrent_idempotency_and_single_conflict_winner",
	}
	if !reflect.DeepEqual(fixture.SharedInvariants, wantInvariants) {
		t.Fatalf("shared artifact contract drift: %v", fixture.SharedInvariants)
	}
}

func TestV31ArtifactParityFoundationDoesNotAccreditOPS13(t *testing.T) {
	fixture := readV31ArtifactParityFixture(t)
	wantDeferred := []string{
		"real_s3_compatible_backend",
		"real_s3_compatible_backend_receipt",
		"postgres_s3_multihost_e2e",
		"sealed_candidate_receipt",
	}
	if fixture.Status != "implemented_foundation_not_accredited" || !reflect.DeepEqual(fixture.DeferredGates, wantDeferred) {
		t.Fatalf("V31 artifact parity foundation overclaims OPS-13: %+v", fixture)
	}
}

func readV31ArtifactParityFixture(t *testing.T) v31ArtifactParityFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(evidenceRepositoryRoot(t), v31ArtifactParityFixturePath))
	if err != nil {
		t.Fatal(err)
	}
	var fixture v31ArtifactParityFixture
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("trailing fixture data: %v", err)
	}
	canonical, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, append(canonical, '\n')) {
		t.Fatal("V31 artifact parity fixture is not canonical strict JSON")
	}
	return fixture
}
