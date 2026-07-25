package config

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestTestAttestorRegistryIsMinimalAndDisabledByDefault(t *testing.T) {
	snapshot, err := Resolve(ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	keys := []Key{KeyTestAttestorProvider, KeyTestAttestorMaxSubjectBytes,
		KeyTestAttestorTimeout,
		KeyTestAttestorMaxConcurrentRuns, KeyTestAttestorBubblewrapCommand, KeyTestAttestorGoToolchainRoot,
		KeyTestAttestorMicroVMLauncherSocket, KeyTestAttestorMicroVMExpectedAssetDigest,
		KeyTestAttestorMicroVMGuestMemoryMiB,
		KeyTestAttestorCgroupRoot, KeyTestAttestorMemoryMaxBytes, KeyTestAttestorPIDsMax,
		KeyTestAttestorCPUQuotaMicros}
	if snapshot.TestAttestorProvider() != "disabled" || snapshot.SchedulerExecutionTimeout() != 45*time.Minute ||
		snapshot.TestAttestorTimeout() != 15*time.Minute ||
		snapshot.TestAttestorMaxSubjectBytes() != 512<<20 || snapshot.TestAttestorMaxConcurrentRuns() != 2 ||
		snapshot.TestAttestorMicroVMGuestMemoryMiB() != 4096 ||
		snapshot.TestAttestorMemoryMaxBytes() != 5<<30 || snapshot.TestAttestorPIDsMax() != 512 {
		t.Fatalf("unsafe attestor defaults: %+v", snapshot)
	}
	for _, key := range keys {
		metadata, found := snapshot.Metadata(key)
		if !found || !metadata.RestartRequired || metadata.Scope != "attestor" {
			t.Fatalf("missing attestor metadata for %s: %+v", key, metadata)
		}
	}
	guestMetadata, _ := snapshot.Metadata(KeyTestAttestorMicroVMGuestMemoryMiB)
	if guestMetadata.Minimum == nil || *guestMetadata.Minimum != 128 {
		t.Fatalf("microVM guest minimum=%v want=128 MiB", guestMetadata.Minimum)
	}
	count := 0
	registry, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range registry.keys {
		if metadata, _ := snapshot.Metadata(definition.Key); metadata.Scope == "attestor" {
			count++
		}
	}
	if count != len(keys) {
		t.Fatalf("attestor key count=%d want=%d", count, len(keys))
	}
}

func TestBubblewrapConfigurationIsOneValidatedUnit(t *testing.T) {
	document := []byte(`[repository.local]
seed_path = "/repo"
[runtime]
max_output_bytes = 2097152
[test_attestor]
provider = "bubblewrap"
timeout = "2m"
max_subject_bytes = 16777216
max_concurrent_runs = 3
[test_attestor.bubblewrap]
command = "/usr/bin/bwrap"
[test_attestor.go]
toolchain_root = "/usr/local/go"
[test_attestor.resources]
cgroup_root = "/sys/fs/cgroup/orquesta"
memory_max_bytes = 2147483648
pids_max = 128
cpu_quota_micros = 200000
[scheduler]
attest_test_claim_lease = "3m"
`)
	snapshot, err := Resolve(ResolveOptions{TOML: document})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.TestAttestorProvider() != "bubblewrap" || snapshot.TestAttestorTimeout() != 2*time.Minute || snapshot.RuntimeMaxOutputBytes() != 2<<20 ||
		snapshot.TestAttestorMaxConcurrentRuns() != 3 || snapshot.TestAttestorCgroupRoot() != "/sys/fs/cgroup/orquesta" {
		t.Fatalf("attestor config lost: %+v", snapshot)
	}
	for name, mutation := range map[string]string{
		"missing command":      "",
		"relative command":     "bwrap",
		"noncanonical command": "/usr/../usr/bin/bwrap",
	} {
		t.Run(name, func(t *testing.T) {
			line := "command = \"/usr/bin/bwrap\""
			replacement := ""
			if mutation != "" {
				replacement = "command = \"" + mutation + "\""
			}
			bad := []byte(strings.Replace(string(document), line, replacement, 1))
			if _, err := Resolve(ResolveOptions{TOML: bad}); err == nil {
				t.Fatal("unsafe attestor config accepted")
			}
		})
	}
	maximumSharedQuota := []byte(strings.Replace(
		string(document), "cpu_quota_micros = 200000", "cpu_quota_micros = 10000000", 1,
	))
	if _, err := Resolve(ResolveOptions{TOML: maximumSharedQuota}); err != nil {
		t.Fatalf("bubblewrap lost its shared CPU quota range: %v", err)
	}
}

func TestMicroVMConfigurationIsOneValidatedUnit(t *testing.T) {
	document := []byte(`[repository.local]
seed_path = "/repo"
[runtime]
max_output_bytes = 2097152
[test_attestor]
provider = "microvm"
timeout = "2m"
max_subject_bytes = 16777216
max_concurrent_runs = 3
[test_attestor.microvm]
launcher_socket = "/run/orquesta/firecracker-launcher.sock"
expected_asset_digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
guest_memory_mib = 3072
[test_attestor.resources]
memory_max_bytes = 4294967296
pids_max = 128
cpu_quota_micros = 200000
[scheduler]
attest_test_claim_lease = "3m"
`)
	snapshot, err := Resolve(ResolveOptions{TOML: document})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.TestAttestorProvider() != "microvm" ||
		snapshot.TestAttestorMicroVMLauncherSocket() != "/run/orquesta/firecracker-launcher.sock" ||
		snapshot.TestAttestorMicroVMExpectedAssetDigest() != strings.Repeat("a", 64) ||
		snapshot.TestAttestorMicroVMGuestMemoryMiB() != 3072 ||
		snapshot.TestAttestorBubblewrapCommand() != "" || snapshot.TestAttestorGoToolchainRoot() != "" ||
		snapshot.TestAttestorCgroupRoot() != "" {
		t.Fatalf("microvm config lost: %+v", snapshot)
	}
	withoutSocket := []byte(strings.Replace(
		string(document), `launcher_socket = "/run/orquesta/firecracker-launcher.sock"`+"\n", "", 1,
	))
	if _, err := Resolve(ResolveOptions{TOML: withoutSocket}); err == nil ||
		!HasErrorCode(err, ErrorCrossValidation) {
		t.Fatalf("microvm config without launcher socket accepted: %v", err)
	}
	withoutAssetDigest := []byte(strings.Replace(
		string(document), `expected_asset_digest = "`+strings.Repeat("a", 64)+`"`+"\n", "", 1,
	))
	if _, err := Resolve(ResolveOptions{TOML: withoutAssetDigest}); err == nil ||
		!HasErrorCode(err, ErrorCrossValidation) {
		t.Fatalf("microvm config without expected asset digest accepted: %v", err)
	}
	for name, testCase := range map[string]struct {
		document string
		code     ErrorCode
	}{
		"guest below minimum": {
			document: strings.Replace(string(document), "guest_memory_mib = 3072", "guest_memory_mib = 127", 1),
			code:     ErrorValueInvalid,
		},
		"guest exceeds cgroup margin": {
			document: strings.Replace(string(document), "memory_max_bytes = 4294967296", "memory_max_bytes = 4294967295", 1),
			code:     ErrorCrossValidation,
		},
		"guest cannot hold subject working set": {
			document: strings.Replace(string(document), "max_subject_bytes = 16777216", "max_subject_bytes = 1073741824", 1),
			code:     ErrorCrossValidation,
		},
		"relative socket": {
			document: strings.Replace(string(document), "/run/orquesta/firecracker-launcher.sock", "launcher.sock", 1),
			code:     ErrorCrossValidation,
		},
		"asset digest malformed": {
			document: strings.Replace(string(document), strings.Repeat("a", 64), "AAAA", 1),
			code:     ErrorCrossValidation,
		},
		"quota exceeds Firecracker vCPU limit": {
			document: strings.Replace(string(document), "cpu_quota_micros = 200000", "cpu_quota_micros = 3200001", 1),
			code:     ErrorCrossValidation,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Resolve(ResolveOptions{TOML: []byte(testCase.document)}); err == nil ||
				!HasErrorCode(err, testCase.code) {
				t.Fatalf("unsafe microvm config accepted: %v", err)
			}
		})
	}
	maximumMicroVMQuota := []byte(strings.Replace(
		string(document), "cpu_quota_micros = 200000", "cpu_quota_micros = 3200000", 1,
	))
	if _, err := Resolve(ResolveOptions{TOML: maximumMicroVMQuota}); err != nil {
		t.Fatalf("canonical maximum microVM CPU quota rejected: %v", err)
	}
}

func TestMicroVMHost128GiBProfileForSixteenRunsValidates(t *testing.T) {
	document := []byte(`[repository.local]
seed_path = "/repo"
[runtime]
max_output_bytes = 67108864
[test_attestor]
provider = "microvm"
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
`)
	snapshot, err := Resolve(ResolveOptions{TOML: document})
	if err != nil {
		t.Fatalf("128 GiB host profile rejected: %v", err)
	}
	if snapshot.TestAttestorProvider() != "microvm" ||
		snapshot.TestAttestorMaxConcurrentRuns() != 16 ||
		snapshot.TestAttestorMicroVMGuestMemoryMiB() != 4096 ||
		snapshot.TestAttestorMemoryMaxBytes() != 5<<30 ||
		snapshot.TestAttestorPIDsMax() != 512 ||
		snapshot.TestAttestorCPUQuotaMicros() != 200000 ||
		snapshot.TestAttestorMaxSubjectBytes() != 512<<20 ||
		snapshot.RuntimeMaxOutputBytes() != 64<<20 {
		t.Fatalf("128 GiB host profile lost: %+v", snapshot)
	}
}

func TestMicroVMCapacityRejectsOverflowAndUnsatisfiableCombinations(t *testing.T) {
	policy := testMicroVMPolicy(t)
	const (
		guest4GiB  = int64(4096)
		cgroup5GiB = int64(5 << 30)
	)
	for name, values := range map[string][4]int64{
		"guest byte conversion overflow": {
			math.MaxInt64, math.MaxInt64, 1, 1,
		},
		"cgroup headroom overflow": {
			math.MaxInt64 / (1 << 20), math.MaxInt64, 1, 1,
		},
		"snapshot working set overflow": {
			guest4GiB, cgroup5GiB, math.MaxInt64/2 + 1, 1,
		},
		"output addition overflow": {
			guest4GiB, cgroup5GiB, 1, math.MaxInt64,
		},
		"cgroup margin unsatisfiable": {
			guest4GiB, cgroup5GiB - 1, 512 << 20, 64 << 20,
		},
		"guest capacity unsatisfiable": {
			guest4GiB, cgroup5GiB, 1 << 30, 64 << 20,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if validMicroVMCapacity(values[0], values[1], values[2], values[3], policy) {
				t.Fatal("unsafe microVM capacity accepted")
			}
		})
	}
}

func testMicroVMPolicy(t *testing.T) registryCrossValidatorDefinition {
	t.Helper()
	registry, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, validator := range registry.crossValidators {
		if validator.ID == "test_attestor_provider_requirements" {
			return validator
		}
	}
	t.Fatal("canonical microVM policy is missing")
	return registryCrossValidatorDefinition{}
}

func TestDisabledProviderIgnoresIncompleteMicroVMConfiguration(t *testing.T) {
	snapshot, err := Resolve(ResolveOptions{TOML: []byte(
		"[test_attestor.microvm]\nlauncher_socket = \"launcher.sock\"\n",
	)})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.TestAttestorProvider() != "disabled" ||
		snapshot.TestAttestorMicroVMLauncherSocket() != "launcher.sock" {
		t.Fatalf("disabled provider semantics changed: %+v", snapshot)
	}
}

func TestTestAttestorTimeoutOrdering(t *testing.T) {
	base := `[repository.local]
seed_path = "/repo"
[runtime]
max_output_bytes = 2097152
[runtime.codex]
timeout = "1s"
supervisor_start_timeout = "500ms"
[server]
shutdown_timeout = "15s"
[test_attestor]
provider = "bubblewrap"
timeout = "30s"
max_subject_bytes = 16777216
[test_attestor.bubblewrap]
command = "/usr/bin/bwrap"
[test_attestor.go]
toolchain_root = "/usr/local/go"
[test_attestor.resources]
cgroup_root = "/sys/fs/cgroup/orquesta"
memory_max_bytes = 2147483648
pids_max = 128
cpu_quota_micros = 200000
[scheduler]
attest_test_claim_lease = "60s"
execution_timeout = "90s"
`
	if _, err := Resolve(ResolveOptions{TOML: []byte(base)}); err != nil {
		t.Fatalf("valid timeout ordering rejected: %v", err)
	}
	for name, mutation := range map[string]string{
		"timeout reaches lease":  strings.Replace(base, `timeout = "30s"`, `timeout = "60s"`, 1),
		"cleanup margin missing": strings.Replace(base, `timeout = "30s"`, `timeout = "50s"`, 1),
		"lease reaches execution": strings.Replace(base, `attest_test_claim_lease = "60s"`,
			`attest_test_claim_lease = "90s"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Resolve(ResolveOptions{TOML: []byte(mutation)})
			if err == nil || !HasErrorCode(err, ErrorCrossValidation) {
				t.Fatalf("unsafe timeout ordering accepted: %v", err)
			}
		})
	}
}

func TestOptionalPathRejectsWhitespaceAndNULWithoutAliases(t *testing.T) {
	for _, document := range []string{"[repository.local]\nseed_path = \" /repo\"\n",
		"[test_attestor.bubblewrap]\ncommand = \"/usr/bin/bwrap \"\n",
		"[test_attestor.microvm]\nlauncher_socket = \"/run/launcher\\u0000tail\"\n",
		"[test_attestor.go]\ntoolchain_root = \"/usr/local/go\\u0000tail\"\n"} {
		if _, err := ParseExplicit([]byte(document)); err == nil || !HasErrorCode(err, ErrorValueInvalid) {
			t.Fatalf("unsafe optional path accepted: %q err=%v", document, err)
		}
	}
	if aliases := Aliases(); len(aliases) != 0 {
		t.Fatalf("unexpected aliases: %+v", aliases)
	}
}

func TestCodexReasoningRegistryAcceptsXHighAndUltraAndRejectsUnknown(t *testing.T) {
	for _, effort := range []string{"xhigh", "ultra"} {
		snapshot := resolveTOML(t, "[runtime.codex]\nreasoning = \""+effort+"\"", nil)
		if snapshot.RuntimeCodexReasoning() != effort {
			t.Fatalf("reasoning=%q want=%q", snapshot.RuntimeCodexReasoning(), effort)
		}
	}
	for _, effort := range []string{"max", "XHIGH", "ultra "} {
		_, err := Resolve(ResolveOptions{TOML: []byte("[runtime.codex]\nreasoning = \"" + effort + "\"\n")})
		assertConfigError(t, err, ErrorValueInvalid, KeyRuntimeCodexReasoning)
	}
}
