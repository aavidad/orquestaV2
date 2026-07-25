package config

import (
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
		KeyTestAttestorCgroupRoot, KeyTestAttestorMemoryMaxBytes, KeyTestAttestorPIDsMax,
		KeyTestAttestorCPUQuotaMicros}
	if snapshot.TestAttestorProvider() != "disabled" || snapshot.SchedulerExecutionTimeout() != 45*time.Minute ||
		snapshot.TestAttestorTimeout() != 15*time.Minute ||
		snapshot.TestAttestorMaxSubjectBytes() != 512<<20 || snapshot.TestAttestorMaxConcurrentRuns() != 2 ||
		snapshot.TestAttestorMemoryMaxBytes() != 2<<30 || snapshot.TestAttestorPIDsMax() != 256 {
		t.Fatalf("unsafe attestor defaults: %+v", snapshot)
	}
	for _, key := range keys {
		metadata, found := snapshot.Metadata(key)
		if !found || !metadata.RestartRequired || metadata.Scope != "attestor" {
			t.Fatalf("missing attestor metadata for %s: %+v", key, metadata)
		}
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
