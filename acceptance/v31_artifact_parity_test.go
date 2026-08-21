package acceptance_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	artifactFilesystem "orquesta/internal/adapters/artifact/filesystem"
	artifactS3 "orquesta/internal/adapters/artifact/s3"
	"orquesta/internal/goal"
	"orquesta/internal/testsupport/artifactcontract"
)

const (
	v31ArtifactParityFixturePath = "acceptance/fixtures/v31_artifact_parity.json"
	v31ArtifactParitySuitePath   = "internal/testsupport/artifactcontract"
)

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
		fixture.SuitePath != v31ArtifactParitySuitePath ||
		!reflect.DeepEqual(fixture.Adapters, []string{"filesystem_project_aware", "s3_fake_injected"}) {
		t.Fatalf("invalid V31 artifact parity identity: %+v", fixture)
	}
	wantInvariants := []string{
		"content_addressed_put_exact_replay_and_media_type_conflict",
		"opaque_project_isolation",
		"store_reinstantiation_over_same_backend",
		"concurrent_idempotency_and_single_conflict_winner",
	}
	if !reflect.DeepEqual(fixture.SharedInvariants, wantInvariants) {
		t.Fatalf("shared artifact contract drift: %v", fixture.SharedInvariants)
	}
	backends := map[string]func(*testing.T) artifactcontract.ParityBackend{
		"filesystem_project_aware": newV31FilesystemParityBackend,
		"s3_fake_injected":         newV31S3ParityBackend,
	}
	if len(backends) != len(fixture.Adapters) {
		t.Fatalf("V31 parity backend count=%d fixture count=%d", len(backends), len(fixture.Adapters))
	}
	for _, adapter := range fixture.Adapters {
		openBackend, found := backends[adapter]
		if !found {
			t.Fatalf("V31 parity adapter %q has no executable acceptance", adapter)
		}
		t.Run(adapter, func(t *testing.T) {
			artifactcontract.RunParity(t, openBackend(t))
		})
	}
}

func TestV31ArtifactParityFoundationDoesNotAccreditOPS13(t *testing.T) {
	fixture := readV31ArtifactParityFixture(t)
	wantDeferred := []string{
		"real_s3_compatible_backend",
		"real_process_client_backend_restart_recovery",
		"real_s3_compatible_backend_receipt",
		"postgres_s3_multihost_e2e",
		"sealed_candidate_receipt",
	}
	if fixture.Status != "implemented_foundation_not_accredited" || !reflect.DeepEqual(fixture.DeferredGates, wantDeferred) {
		t.Fatalf("V31 artifact parity foundation overclaims OPS-13: %+v", fixture)
	}
}

func newV31FilesystemParityBackend(t *testing.T) artifactcontract.ParityBackend {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return artifactcontract.ParityBackend{
		Open: func(test testing.TB, project goal.ProjectRef) artifactcontract.Handle {
			test.Helper()
			store, err := artifactFilesystem.OpenForProject(root, project)
			if err != nil {
				test.Fatal(err)
			}
			var once sync.Once
			var closeErr error
			closeFn := func() error {
				once.Do(func() { closeErr = store.Close() })
				return closeErr
			}
			test.Cleanup(func() { _ = closeFn() })
			return artifactcontract.Handle{Store: store, Close: closeFn}
		},
		LogicalObjectCount: func(test testing.TB) int {
			test.Helper()
			count := 0
			err := filepath.WalkDir(root, func(name string, entry os.DirEntry, err error) error {
				if err == nil && !entry.IsDir() && strings.HasSuffix(name, ".blob") {
					count++
				}
				return err
			})
			if err != nil {
				test.Fatal(err)
			}
			return count
		},
		AssertProjectRefOpaque: func(test testing.TB, project goal.ProjectRef) {
			test.Helper()
			leakedPath := ""
			err := filepath.WalkDir(root, func(name string, _ os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if strings.Contains(name, project.String()) {
					leakedPath = name
					return filepath.SkipAll
				}
				return nil
			})
			if err != nil {
				test.Fatal(err)
			}
			if leakedPath != "" {
				test.Fatalf("project ref leaked in filesystem path %q", leakedPath)
			}
		},
	}
}

func newV31S3ParityBackend(t *testing.T) artifactcontract.ParityBackend {
	t.Helper()
	client := newV31S3Client()
	return artifactcontract.ParityBackend{
		Open: func(test testing.TB, project goal.ProjectRef) artifactcontract.Handle {
			test.Helper()
			store, err := artifactS3.New(client, artifactS3.Options{
				Bucket: "v31-parity-artifacts", ProjectRef: project, MaxObjectBytes: 8192,
				ReconcileTimeout: 50 * time.Millisecond,
			})
			if err != nil {
				test.Fatal(err)
			}
			return artifactcontract.Handle{Store: store}
		},
		LogicalObjectCount: func(testing.TB) int { return client.count() },
		AssertProjectRefOpaque: func(test testing.TB, project goal.ProjectRef) {
			test.Helper()
			for _, locator := range client.locators() {
				if strings.Contains(locator.Key, project.String()) {
					test.Fatalf("project ref leaked in object key %q", locator.Key)
				}
			}
		},
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
