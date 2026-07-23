//go:build linux && (v17_real_e2e || v18_real_e2e)

package bootstrap

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	v17RealBubblewrap       = "/usr/bin/bwrap"
	v17SystemdRuntimeMax    = "150s"
	v17SystemdStopTimeout   = "10s"
	realE2ETestTimeout      = "145s"
	realE2ESystemdUnitBytes = 12
)

type realE2ESystemdSpec struct {
	gate       string
	unitPrefix string
	testName   string
	innerFlag  string
	marker     string
}

func runRealE2EInDelegatedUnit(t *testing.T, spec realE2ESystemdSpec) {
	t.Helper()
	systemdRun, err := exec.LookPath("systemd-run")
	if err != nil {
		t.Fatalf("%s_SYSTEMD_UNAVAILABLE: %v", spec.gate, err)
	}
	executable, err := os.Executable()
	v17NoError(t, err)
	unit := randomRealE2ESystemdUnit(t, spec.unitPrefix)
	args := []string{
		"--user", "--wait", "--collect", "--pipe", "--service-type=exec", "--same-dir",
		"--unit=" + unit,
		"--property=Delegate=yes", "--property=DelegateSubgroup=runner",
		"--property=MemoryAccounting=yes", "--property=TasksAccounting=yes",
		"--property=RuntimeMaxSec=" + v17SystemdRuntimeMax,
		"--property=TimeoutStopSec=" + v17SystemdStopTimeout,
		"--property=LimitNOFILE=8192", "--property=LimitFSIZE=536870912",
		"--property=LimitAS=4294967296", "--property=LimitCPU=30",
		executable,
		"-test.run=^" + spec.testName + "$", "-test.count=1", "-test.timeout=" + realE2ETestTimeout,
		"-" + spec.innerFlag + "=" + unit,
	}
	if testing.Verbose() {
		args = append(args, "-test.v=true")
	}
	command := exec.Command(systemdRun, args...)
	command.Stdin = os.Stdin
	output, runErr := command.CombinedOutput()
	_, _ = os.Stdout.Write(output)
	v17RequireSystemdUnitCollected(t, unit)
	if runErr != nil {
		t.Fatalf("%s_SYSTEMD_INNER_FAILED: unit=%s err=%v", spec.gate, unit, runErr)
	}
	if !bytes.Contains(output, []byte(spec.marker+unit)) {
		t.Fatalf("%s_SYSTEMD_INNER_NOT_EXECUTED: unit=%s", spec.gate, unit)
	}
}

func randomRealE2ESystemdUnit(t *testing.T, prefix string) string {
	t.Helper()
	random := make([]byte, realE2ESystemdUnitBytes)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	return prefix + hex.EncodeToString(random) + ".service"
}

func v17NoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func v17RequireSystemdUnitCollected(t *testing.T, unit string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		output, err := exec.Command("systemctl", "--user", "show", unit, "--property=LoadState", "--value").CombinedOutput()
		state := strings.TrimSpace(string(output))
		if err != nil || state == "" || state == "not-found" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("V17_GATE_REAL_E2E_SYSTEMD_UNIT_NOT_COLLECTED: unit=%s state=%s err=%v", unit, state, err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func v17RequireDelegatedSystemdHarness(t *testing.T, unit string) {
	t.Helper()
	properties := v17SystemdProperties(t, unit)
	want := map[string]string{
		"Delegate": "yes", "DelegateSubgroup": "runner",
		"MemoryAccounting": "yes", "TasksAccounting": "yes",
		"LimitNOFILE": "8192", "LimitNOFILESoft": "8192",
		"LimitFSIZE": "536870912", "LimitFSIZESoft": "536870912",
		"LimitAS": "4294967296", "LimitASSoft": "4294967296",
		"LimitCPU": "30", "LimitCPUSoft": "30",
	}
	for property, value := range want {
		if properties[property] != value {
			t.Fatalf("V17_GATE_REAL_E2E_SYSTEMD_PROPERTY_INVALID: %s=%q want=%q", property, properties[property], value)
		}
	}
	mainPID, err := strconv.Atoi(properties["MainPID"])
	if err != nil || mainPID != os.Getpid() {
		t.Fatalf("V17_GATE_REAL_E2E_SYSTEMD_MAIN_PID_INVALID: got=%q pid=%d err=%v", properties["MainPID"], os.Getpid(), err)
	}
	controlGroup := properties["ControlGroup"]
	if controlGroup == "" || !strings.HasPrefix(controlGroup, "/user.slice/") ||
		filepath.Clean(controlGroup) != controlGroup || filepath.Base(controlGroup) != unit {
		t.Fatalf("V17_GATE_REAL_E2E_SYSTEMD_CONTROL_GROUP_INVALID: %q", controlGroup)
	}
	self := v17SelfCgroup(t)
	if self != controlGroup+"/runner" {
		t.Fatalf("V17_GATE_REAL_E2E_SYSTEMD_RUNNER_INVALID: self=%q control=%q", self, controlGroup)
	}
	var filesystem syscall.Statfs_t
	if err := syscall.Statfs("/sys/fs/cgroup", &filesystem); err != nil || uint64(filesystem.Type) != 0x63677270 {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_V2_REQUIRED: type=%#x err=%v", filesystem.Type, err)
	}
	parent := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(controlGroup, "/"))
	runner := filepath.Join(parent, "runner")
	v17RequireCgroupOwner(t, parent)
	v17RequireCgroupOwner(t, runner)
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_PARENT_MODE: %v", err)
	}
	info, err := os.Lstat(parent)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_PARENT_MODE: mode=%v err=%v", info.Mode(), err)
	}
	v17RequireCgroupProcesses(t, parent, nil)
	v17RequireCgroupProcesses(t, runner, []int{os.Getpid()})
	v17EnableExactCgroupControllers(t, parent)
}

func v17SystemdProperties(t *testing.T, unit string) map[string]string {
	t.Helper()
	requested := "ControlGroup,MainPID,Delegate,DelegateSubgroup,MemoryAccounting,TasksAccounting," +
		"LimitNOFILE,LimitNOFILESoft,LimitFSIZE,LimitFSIZESoft,LimitAS,LimitASSoft,LimitCPU,LimitCPUSoft"
	output, err := exec.Command("systemctl", "--user", "show", unit, "--property="+requested).CombinedOutput()
	if err != nil {
		t.Fatalf("V17_GATE_REAL_E2E_SYSTEMD_INSPECTION_FAILED: %v: %s", err, output)
	}
	properties := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		name, value, ok := strings.Cut(line, "=")
		if ok {
			properties[name] = value
		}
	}
	return properties
}

func v17SelfCgroup(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || len(content) > 4096 {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_UNAVAILABLE: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "0::/") {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_V2_REQUIRED: %q", content)
	}
	return strings.TrimPrefix(lines[0], "0::")
}

func v17RequireCgroupOwner(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	v17NoError(t, err)
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_OWNER_INVALID: path=%s uid=%d euid=%d", path, stat.Uid, os.Geteuid())
	}
}

func v17RequireCgroupProcesses(t *testing.T, root string, want []int) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, "cgroup.procs"))
	v17NoError(t, err)
	var got []int
	for _, value := range strings.Fields(string(content)) {
		pid, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		got = append(got, pid)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_PROCESSES_INVALID: root=%s got=%v want=%v", root, got, want)
	}
}

func v17EnableExactCgroupControllers(t *testing.T, parent string) {
	t.Helper()
	want := map[string]bool{"cpu": true, "memory": true, "pids": true}
	available, err := os.ReadFile(filepath.Join(parent, "cgroup.controllers"))
	v17NoError(t, err)
	availableSet := make(map[string]bool)
	for _, controller := range strings.Fields(string(available)) {
		availableSet[controller] = true
	}
	for controller := range want {
		if !availableSet[controller] {
			t.Fatalf("V17_GATE_REAL_E2E_CGROUP_CONTROLLER_UNAVAILABLE: %s in %q", controller, available)
		}
	}
	controlPath := filepath.Join(parent, "cgroup.subtree_control")
	current, err := os.ReadFile(controlPath)
	v17NoError(t, err)
	for _, controller := range strings.Fields(string(current)) {
		if !want[controller] {
			if err := os.WriteFile(controlPath, []byte("-"+controller), 0o600); err != nil {
				t.Fatalf("V17_GATE_REAL_E2E_CGROUP_CONTROLLER_DISABLE_FAILED: %s: %v", controller, err)
			}
		}
	}
	if err := os.WriteFile(controlPath, []byte("+cpu +memory +pids"), 0o600); err != nil {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_CONTROLLER_ENABLE_FAILED: %v", err)
	}
	enabled, err := os.ReadFile(controlPath)
	if err != nil || strings.Join(strings.Fields(string(enabled)), " ") != "cpu memory pids" {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_CONTROLLERS_INVALID: %q err=%v", enabled, err)
	}
}

func v17RequireRealAttestorHost(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 || os.Getegid() == 0 {
		t.Fatal("V17_GATE_REAL_E2E_IDENTITY_UNSAFE: real gate requires non-root; no validated local non-root reexec pattern exists")
	}
	toolchain := v17TrustedToolchainRoot(t)
	for _, path := range []string{v17RealBubblewrap, toolchain, filepath.Join(toolchain, "bin", "go")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("V17_GATE_REAL_E2E_INPUT_UNAVAILABLE: %s: %v", path, err)
		}
	}
}

func v17TrustedToolchainRoot(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{"/usr/local/go", runtime.GOROOT()} {
		root, rootErr := os.Lstat(candidate)
		binary, binaryErr := os.Lstat(filepath.Join(candidate, "bin", "go"))
		if rootErr != nil || binaryErr != nil {
			continue
		}
		rootStat, rootOK := root.Sys().(*syscall.Stat_t)
		binaryStat, binaryOK := binary.Sys().(*syscall.Stat_t)
		if root.IsDir() && binary.Mode().IsRegular() &&
			rootOK && binaryOK && rootStat.Uid == 0 && binaryStat.Uid == 0 &&
			root.Mode().Perm()&0o022 == 0 && binary.Mode().Perm()&0o022 == 0 && binary.Mode().Perm()&0o111 != 0 {
			return candidate
		}
	}
	t.Fatal("V17_GATE_REAL_E2E_TOOLCHAIN_UNSAFE: root-owned immutable Go toolchain unavailable")
	return ""
}

func v17ConfiguredCgroupRoot(t *testing.T) string {
	t.Helper()
	self := v17SelfCgroup(t)
	if filepath.Base(self) != "runner" {
		t.Fatalf("V17_GATE_REAL_E2E_CGROUP_RUNNER_MISSING: %q", self)
	}
	return filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(filepath.Dir(self), "/"))
}

func v17RequirePrivateMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, want); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != want || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("private fixture mode %s=%v want=%#o err=%v", path, info, want, err)
	}
}

func requireNoRealE2EOwnedProcesses(t *testing.T) {
	t.Helper()
	self := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(v17SelfCgroup(t), "/"))
	v17RequireCgroupProcesses(t, self, []int{os.Getpid()})
	parent := filepath.Dir(self)
	entries, err := os.ReadDir(parent)
	v17NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "run-") {
			t.Fatalf("REAL_E2E_OWNED_CGROUP_REMAINED: %s", filepath.Join(parent, entry.Name()))
		}
	}
}

func configureBubblewrapTestAttestor(t *testing.T, path string) {
	t.Helper()
	replaceTestConfigValue(t, path, "shutdown_timeout = \"2s\"", "shutdown_timeout = \"3s\"")
	replaceTestConfigValue(t, path, "claim_lease = \"10s\"", `claim_lease = "45s"
attest_test_claim_lease = "45s"`)
	replaceTestConfigValue(t, path, "max_execution_attempts = 3", "max_execution_attempts = 1")
	replaceTestConfigValue(t, path, "execution_timeout = \"10s\"", "execution_timeout = \"60s\"")
	replaceTestConfigValue(t, path, "[identity]\n", fmt.Sprintf(`[test_attestor]
provider = "bubblewrap"
timeout = "30s"
max_subject_bytes = 536870912

[test_attestor.bubblewrap]
command = %s

[test_attestor.go]
toolchain_root = %s

[test_attestor.resources]
cgroup_root = %s
memory_max_bytes = 2147483648
pids_max = 256
cpu_quota_micros = 200000

[identity]
`, strconv.Quote(v17RealBubblewrap), strconv.Quote(v17TrustedToolchainRoot(t)),
		strconv.Quote(v17ConfiguredCgroupRoot(t))))
	v17RequirePrivateMode(t, path, 0o600)
}
