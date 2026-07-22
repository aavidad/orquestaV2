package bootstrap

import (
	"context"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/adapters/attestor/bubblewrap"
	"orquesta/internal/adapters/system/local"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestDisabledTestAttestorKeepsNonCodeCompositionOperational(t *testing.T) {
	snapshot, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	composition, err := openBuildTestAttestor(snapshot, local.Clock{}, nil)
	if err != nil || composition.attestor != nil || composition.policy != (application.TestAttestationPolicy{}) {
		t.Fatalf("disabled attestor composition = %+v err=%v", composition, err)
	}
}

func TestBuildOrchestratorWiresDistinctAttestClaimLease(t *testing.T) {
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte(`[scheduler]
claim_lease = "2m"
attest_test_claim_lease = "20m"
`)})
	if err != nil {
		t.Fatal(err)
	}
	dependencies := buildOrchestratorDependencies(
		buildSetup{snapshot: snapshot}, nil, nil, nil, nil, ports.AgentCapabilities{}, nil,
		buildTestAttestorComposition{},
	)
	if dependencies.ClaimLease != 2*time.Minute || dependencies.AttestTestClaimLease != 20*time.Minute {
		t.Fatalf("claim lease wiring normal=%s attest=%s", dependencies.ClaimLease, dependencies.AttestTestClaimLease)
	}
}

func TestBuildRejectsPartialTestAttestorConfigBeforeEffects(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[identity]\n", `[test_attestor]
provider = "bubblewrap"

[identity]
`)
	var factoryCalls atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		AgentFactory: func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			factoryCalls.Add(1)
			return countingFactory(&atomic.Int64{})(snapshot, clock)
		},
	})
	if runtime != nil || !config.HasErrorCode(err, config.ErrorCrossValidation) {
		t.Fatalf("partial attestor config = %v, %v", runtime, err)
	}
	if factoryCalls.Load() != 0 {
		t.Fatalf("partial attestor config reached agent factory %d times", factoryCalls.Load())
	}
	assertNoCompositionState(t, root)
}

func TestBuildRejectsUnavailableTrustedAttestorInputsAndShutsAgent(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[identity]\n", `[repository.local]
seed_path = `+strconv.Quote(filepath.Join(root, "repository"))+`

[workspace.local]
root = `+strconv.Quote(filepath.Join(root, "workspaces"))+`

[test_attestor]
provider = "bubblewrap"
timeout = "3s"

[test_attestor.bubblewrap]
command = `+strconv.Quote(filepath.Join(root, "missing", "bwrap"))+`

[test_attestor.go]
toolchain_root = `+strconv.Quote(filepath.Join(root, "missing", "go"))+`

[test_attestor.resources]
cgroup_root = `+strconv.Quote(filepath.Join(root, "missing", "cgroup"))+`

[identity]
`)
	replaceTestConfigValue(t, configPath, "[scheduler]\n", `[scheduler]
attest_test_claim_lease = "6s"
`)
	var factoryCalls atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		AgentFactory: func(snapshot config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			factoryCalls.Add(1)
			return &v16WorkspaceAgent{now: clock.Now, writes: map[string]v16Write{},
				launches: &atomic.Int64{}, requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest)}, nil
		},
	})
	if runtime != nil || bubblewrap.ErrorCode(err) != bubblewrap.CodeBinaryUnsafe {
		t.Fatalf("unsafe attestor config = %v, %v", runtime, err)
	}
	if factoryCalls.Load() != 1 {
		t.Fatalf("unsafe attestor config factory calls=%d", factoryCalls.Load())
	}
}
