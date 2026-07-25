//go:build linux

package bootstrap

import (
	"os"
	"strings"
	"testing"
	"time"

	"orquesta/internal/config"
)

func TestBubblewrapTestAttestorWiringUsesObjectStreamAndCanonicalLimits(t *testing.T) {
	content, err := os.ReadFile("runtime.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, required := range []string{
		"bubblewrap.New(config)",
		"TestAttestorTimeout()",
		"TestAttestorMaxSubjectBytes()",
		"TestAttestorMaxConcurrentRuns()",
		"RuntimeMaxOutputBytes()",
		"closer: attestor",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("production wiring missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"WorkspacePathResolver: workspace",
		"MaterializeVerifiedTestWorkspace(",
		"bubblewrap.Prepare(",
	} {
		if strings.Contains(source, forbidden) {
			t.Errorf("unsafe worktree authority remains: %q", forbidden)
		}
	}
}

func TestMicroVMTestAttestorWiringUsesAuthenticatedLauncherAndCanonicalLimits(t *testing.T) {
	content, err := os.ReadFile("runtime.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, required := range []string{
		`"orquesta/internal/adapters/attestor/firecrackerclient"`,
		`"orquesta/internal/adapters/attestor/firecrackerlauncher"`,
		"firecrackerlauncher.NewClient(",
		"firecrackerclient.New(firecrackerConfig(",
		"TestAttestorMicroVMExpectedAssetDigest()",
		"TestAttestorMicroVMLauncherSocket()",
		"TestAttestorMicroVMGuestMemoryMiB()",
		"CleanupTimeout: snapshot.ServerShutdownTimeout()",
		"closer: attestor",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("microVM production wiring missing %q", required)
		}
	}
}

func TestMicroVMTestAttestorConfigProjectsCanonicalProfile(t *testing.T) {
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte(`
[repository.local]
seed_path = "/repo"
[runtime]
max_output_bytes = 67108864
[test_attestor]
provider = "microvm"
timeout = "2m"
max_subject_bytes = 536870912
max_concurrent_runs = 16
[test_attestor.microvm]
launcher_socket = "/run/orquesta/firecracker-launcher.sock"
expected_asset_digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
guest_memory_mib = 4096
[test_attestor.resources]
memory_max_bytes = 5368709120
pids_max = 512
cpu_quota_micros = 200000
[scheduler]
attest_test_claim_lease = "3m"
`)})
	if err != nil {
		t.Fatal(err)
	}
	projected := firecrackerConfig(snapshot, nil, nil, time.Now)
	if projected.ExpectedAssetDigest != strings.Repeat("a", 64) ||
		projected.Limits.Timeout != 2*time.Minute ||
		projected.Limits.CleanupTimeout != snapshot.ServerShutdownTimeout() ||
		projected.Limits.MaxOutputBytes != 64<<20 ||
		projected.Limits.MaxSubjectBytes != 512<<20 ||
		projected.Limits.MaxConcurrentRuns != 16 ||
		projected.Limits.GuestMemoryMiB != 4096 ||
		projected.Limits.MemoryMaxBytes != 5<<30 ||
		projected.Limits.PIDsMax != 512 ||
		projected.Limits.CPUQuotaMicros != 200000 {
		t.Fatalf("canonical microVM profile projected incorrectly: %+v", projected)
	}
}
