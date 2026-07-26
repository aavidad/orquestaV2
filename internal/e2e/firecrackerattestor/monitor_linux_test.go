//go:build linux

package firecrackerattestor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestWaitQuiescentAcceptsMissingRunsOnlyAfterStop(t *testing.T) {
	runtimeRoot := t.TempDir()
	cgroupRoot := t.TempDir()
	const parentCgroup = "orquesta-firecracker-attestor-test"
	if err := os.Mkdir(filepath.Join(cgroupRoot, parentCgroup), 0o700); err != nil {
		t.Fatal(err)
	}

	config := validSupervisorConfig()
	config.Candidate.RuntimeRoot = runtimeRoot
	config.Candidate.CgroupRoot = cgroupRoot
	config.Candidate.ParentCgroup = parentCgroup
	config.Candidate.LauncherSocketPath = filepath.Join(
		runtimeRoot, "firecracker-launcher.sock",
	)
	config.StableFor = time.Millisecond
	config.PollInterval = time.Millisecond

	stoppedUnit := &fakeUnit{
		stopped: true, fragmentPath: config.Candidate.UnitPath,
	}
	stopped := UnitIdentity{
		UnitName: config.Candidate.UnitName, FragmentPath: config.Candidate.UnitPath,
		Loaded: true,
	}
	monitor := &LinuxMonitor{config: config, unit: stoppedUnit}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cleanup, err := monitor.WaitQuiescent(ctx, stopped)
	if err != nil {
		t.Fatal(err)
	}
	if !cleanAfterStop(cleanup) {
		t.Fatalf("missing stopped runs not clean: %+v", cleanup)
	}

	activeUnit := &fakeUnit{fragmentPath: config.Candidate.UnitPath}
	active, err := activeUnit.Observe(context.Background(), config.Candidate.UnitName)
	if err != nil {
		t.Fatal(err)
	}
	activeMonitor := &LinuxMonitor{config: config, unit: activeUnit}
	if _, err := activeMonitor.WaitQuiescent(context.Background(), active); err == nil {
		t.Fatal("missing active runs accepted")
	}
}

func TestUnitIdentityMatchesStoppedBaselineWithoutLiveIdentity(t *testing.T) {
	stopped := UnitIdentity{
		UnitName:     "orquesta-firecracker-attestor-" + testDigest + ".service",
		FragmentPath: "/etc/systemd/system/unit.service", Loaded: true,
	}
	if !unitIdentityMatches(stopped, UnitIdentity{
		UnitName: stopped.UnitName, MainPID: 0, Active: false,
		FragmentPath: stopped.FragmentPath, Loaded: true,
		// systemd may retain the last InvocationID after stop.
		InvocationID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}) {
		t.Fatal("stopped unit did not reach stable cleanup state")
	}
	if unitIdentityMatches(stopped, UnitIdentity{
		UnitName: stopped.UnitName, MainPID: 42, Active: false,
		FragmentPath: stopped.FragmentPath, Loaded: true,
	}) {
		t.Fatal("residual MainPID accepted after stop")
	}
}

func TestValidRunIDMatchesLauncherBase32Shape(t *testing.T) {
	// nonce=32 zero bytes -> base32(no padding)=52 lower-case 'a' bytes.
	realVector := "orq-" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if !validRunID(realVector) {
		t.Fatalf("launcher vector rejected: %q len=%d", realVector, len(realVector))
	}
	for _, invalid := range []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"orq-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"orq-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa!",
		"orq-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		if validRunID(invalid) {
			t.Fatalf("invalid run id accepted: %q", invalid)
		}
	}
}

func TestCandidateUnitPathsMatchInstallerNames(t *testing.T) {
	candidate := Candidate{
		UnitName:   "orquesta-firecracker-attestor-" + testDigest + ".service",
		UnitPath:   "/etc/systemd/system/orquesta-firecracker-attestor-" + testDigest + ".service",
		UnitSHA256: testDigest,
		PrimitivesUnitPath: "/etc/systemd/system/orquesta-firecracker-primitives-" +
			testDigest + ".service",
		PrimitivesUnitSHA256: testDigest,
	}
	if !validCandidateUnitPaths(candidate) {
		t.Fatal("canonical installer unit names rejected")
	}
	candidate.PrimitivesUnitPath = "/etc/systemd/system/orquesta-firecracker-attestor-primitives-" +
		testDigest + ".service"
	if validCandidateUnitPaths(candidate) {
		t.Fatal("non-installer primitives unit prefix accepted")
	}
	installer, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "scripts", "install_firecracker_attestor_launcher.sh",
	))
	if err != nil {
		t.Fatal(err)
	}
	const renderedName = `primitives_unit_name="orquesta-firecracker-primitives-$primitives_unit_sha.service"`
	if !strings.Contains(string(installer), renderedName) {
		t.Fatal("installer and E2E primitives unit names drifted")
	}
}

func TestTreeHasNoSocketsEnforcesOneGlobalEntryLimit(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"a", "b"} {
		path := filepath.Join(root, directory)
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"one", "two"} {
			if err := os.WriteFile(filepath.Join(path, name), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	handle, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if clean, err := treeHasNoSockets(handle, 0, 3); err == nil || clean {
		t.Fatalf("global tree limit bypassed: clean=%v err=%v", clean, err)
	}
}

func TestTrustedAncestorStatRejectsNonRootAndWritableParents(t *testing.T) {
	base := unix.Stat_t{
		Mode: unix.S_IFDIR | 0o755, Uid: 0, Gid: 0,
	}
	if !trustedAncestorStat(&base) {
		t.Fatal("safe root ancestor rejected")
	}
	nonRoot := base
	nonRoot.Uid = 1000
	if trustedAncestorStat(&nonRoot) {
		t.Fatal("non-root ancestor accepted")
	}
	writable := base
	writable.Mode = unix.S_IFDIR | 0o777
	if trustedAncestorStat(&writable) {
		t.Fatal("group/world-writable ancestor accepted")
	}
}

func TestVerifyTrustedAncestorsRejectsUntrustedTemporaryParent(t *testing.T) {
	parent := t.TempDir()
	path := filepath.Join(parent, "candidate")
	if err := os.WriteFile(path, []byte("candidate"), 0o500); err != nil {
		t.Fatal(err)
	}
	if VerifyTrustedAncestors(path) == nil {
		t.Fatal("non-root temporary ancestor accepted")
	}
}

func TestReadVirtualFileAcceptsProcZeroSizedControls(t *testing.T) {
	process, err := openDirectoryNoLinks("/proc", strconv.Itoa(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	defer process.Close()
	content, err := readVirtualFileAt(process, "status", maxControlBytes)
	if err != nil || len(content) == 0 {
		t.Fatalf("virtual proc file not read: bytes=%d err=%v", len(content), err)
	}
}

func TestProcessInParentCgroupIncludesDirectAndDescendantOnly(t *testing.T) {
	parent := "/orquesta-firecracker-attestor"
	for _, content := range []string{
		"0::/orquesta-firecracker-attestor\n",
		"0::/orquesta-firecracker-attestor/orq-abc\n",
	} {
		if !processInParentCgroup([]byte(content), parent) {
			t.Fatalf("owned cgroup process not detected: %q", content)
		}
	}
	for _, content := range []string{
		"0::/orquesta-firecracker-attestor-other/orq-abc\n",
		"1:name=systemd:/orquesta-firecracker-attestor/orq-abc\n",
		"0::/other\n",
	} {
		if processInParentCgroup([]byte(content), parent) {
			t.Fatalf("foreign cgroup process accepted: %q", content)
		}
	}
}

func TestUnitIdentityMatchesLiveBaselineRequiresExactPIDAndInvocation(t *testing.T) {
	live := UnitIdentity{
		UnitName: "orquesta-firecracker-attestor-" + testDigest + ".service",
		MainPID:  42, InvocationID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Active: true, FragmentPath: "/etc/systemd/system/unit.service",
		Loaded: true,
	}
	if !unitIdentityMatches(live, live) {
		t.Fatal("exact live identity rejected")
	}
	drifted := live
	drifted.InvocationID = "cccccccccccccccccccccccccccccccc"
	if unitIdentityMatches(live, drifted) {
		t.Fatal("InvocationID drift accepted")
	}
}
