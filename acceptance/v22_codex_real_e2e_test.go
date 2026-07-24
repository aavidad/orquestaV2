//go:build v22_real_e2e && linux

package acceptance_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"net"
	"net/http"
	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Scenario mutations use official MCP; live-state probes are read-only and authorization mutates only isolated restore copies.
const (
	v22RealDeadline       = 12 * time.Minute
	v22SystemdRuntimeMax  = "780s"
	v22SystemdTestTimeout = "765s"
	v22SystemdMarker      = "V22_REAL_E2E_INNER_OK:"
)

var v22SystemdInnerUnit = flag.String("v22-systemd-inner-unit", "", "internal V22 delegated systemd unit")

type v22Output struct {
	Result struct {
		Data    json.RawMessage `json:"data"`
		Failure any             `json:"failure"`
	} `json:"result"`
}
type v22Object map[string]any
type v22GoalProjection = v22Object
type v22ExecutionProjection = v22Object
type v22CouncilIntegration struct {
	ChangeRef, WorkItemRef, ExecutionRef, BaseOID         string
	SubjectDigest, DecisionRef, DecisionDigest, ActionRef string
}

func (o v22Object) object(key string) v22Object { return v22Object(o[key].(map[string]any)) }
func (o v22Object) text(key string) string      { value, _ := o[key].(string); return value }
func (o v22Object) number(key string) uint64    { value, _ := o[key].(float64); return uint64(value) }
func (o v22Object) flag(key string) bool        { value, _ := o[key].(bool); return value }
func (o v22Object) strings(key string) []string {
	raw, _ := o[key].([]any)
	values := make([]string, len(raw))
	for index := range raw {
		values[index], _ = raw[index].(string)
	}
	return values
}
func (o v22Object) objects(key string) []v22Object {
	raw, _ := o[key].([]any)
	values := make([]v22Object, len(raw))
	for index := range raw {
		values[index] = v22Object(raw[index].(map[string]any))
	}
	return values
}

type v22AdmissionIdentity struct{ Message, Admission, SourcePrincipal, SourceExecution, RecipientPrincipal, RecipientExecution string }
type v22MailboxFence struct {
	Admission                                                                v22AdmissionIdentity
	Delivered, Sibling                                                       string
	InitialReceipt                                                           identity.AuthorizationReceipt
	InitialReceiptRef, InitialRequestRef, InitialCopyPath, InitialCopyTarget string
}
type v22IsolatedCopy struct{ Path, Target string }
type v22RestoreExpected struct {
	Project, Goal, Spec, GoalState, Work, Execution, Replaces, ExecutionState, Purpose string
	Plan, App, Attempt, MaxAttempts, WorkItems, Executions                             uint64
}

func v22Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
func v22Decode[T any](path string) T {
	var value T
	if err := json.Unmarshal(v22Must(os.ReadFile(path)), &value); err != nil {
		panic(err)
	}
	return value
}

type v22Harness struct {
	t                                                *testing.T
	root, binary, config, state, endpoint, codexHome string
	server                                           *exec.Cmd
	log                                              *os.File
	session                                          *sdkmcp.ClientSession
	mu                                               sync.Mutex
	request                                          uint64
}

func TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP(t *testing.T) {
	v22InDelegatedUnit(t, "TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP", func() {
		ctx, cancel := context.WithTimeout(context.Background(), v22RealDeadline)
		defer cancel()
		h := v22Start(t, ctx)
		refs := map[string]string{"A": h.create(ctx, "A", v22PlanA()), "B": h.create(ctx, "B", v22PlanB()), "C": h.create(ctx, "C", v22PlanC()), "D": h.create(ctx, "D", v22PlanD())}
		running := h.waitRunning(ctx, refs["A"], refs["B"], refs["C"], refs["D"])
		bProcess := v22WaitProcess(t, h.root, running[refs["B"]].text("execution_ref"))
		dProcess := v22WaitProcess(t, h.root, running[refs["D"]].text("execution_ref"))
		progress := map[string]v22GoalProjection{}
		for _, id := range []string{"A", "C", "D"} {
			progress[id] = h.get(ctx, refs[id])
		}
		b := h.get(ctx, refs["B"])
		bGoal := b.object("goal")
		h.call(ctx, "orquesta.goals.control", map[string]any{"operation": "cancel", "target": "goal", "goal_ref": refs["B"], "expected_goal_revision": bGoal.number("revision"), "expected_plan_generation": bGoal.number("plan_generation"), "expected_app_spec_generation": bGoal.number("app_spec_generation"), "expected_spec_hash": bGoal.text("spec_hash"), "reason": "V22 public selective cancellation while A/C/D progress"})
		b = h.waitCancelled(ctx, refs["B"], running[refs["B"]], bProcess)
		for _, id := range []string{"A", "C"} {
			h.waitProgress(ctx, refs[id], progress[id])
		}
		exactD := running[refs["D"]].text("execution_ref")
		current := h.runningExecution(ctx, refs["D"]).text("execution_ref")
		v22Require(t, current == exactD, "B cancellation disturbed D execution: got=%s want=%s", current, exactD)
		fence := h.assertMailboxArtifactIsolation(ctx, refs["A"])
		admission := fence.Admission
		expectedD := v22RestoreExpectation(t, h.get(ctx, refs["D"]))
		backup := h.backup(ctx)
		h.verifyRestoreCopy(ctx, backup, expectedD)
		current = h.runningExecution(ctx, refs["D"]).text("execution_ref")
		v22Require(t, current == exactD, "D execution changed before crash: got=%s want=%s", current, exactD)
		h.kill() // non-cooperative server death: SIGKILL, never Runtime.Shutdown.
		alive, aliveErr := v22ProcessAlive(dProcess)
		v22Require(t, aliveErr == nil && alive, "D exact process did not survive server SIGKILL: alive=%v err=%v", alive, aliveErr)
		h.restart(ctx)
		h.probeMailboxArtifactIsolation(ctx, refs["A"], fence, "restart")
		h.waitAdopted(ctx, refs["D"], running[refs["D"]], dProcess)
		recovered, recoveredOK := h.completedAdmission(ctx, refs["A"])
		v22Require(t, recoveredOK && recovered == admission, "restart changed completed service admission: got=%+v want=%+v found=%v", recovered, admission, recoveredOK)
		councilIntegration := h.admitAcceptedCouncilIntegration(ctx, refs["C"])
		completed := map[string]v22GoalProjection{}
		for _, id := range []string{"A", "C", "D"} {
			completed[id] = h.waitTerminal(ctx, refs[id])
			state := completed[id].object("goal").text("state")
			v22Require(t, state == "succeeded", "%s state=%s", id, state)
			v22NoContradiction(t, completed[id])
		}
		h.assertSucceededGoal(ctx, refs["A"])
		recovered, recoveredOK = h.completedAdmission(ctx, refs["A"])
		v22Require(t, recoveredOK && recovered == admission, "terminal Goal changed completed admission: got=%+v want=%+v found=%v", recovered, admission, recoveredOK)
		v22AssertAdoptedCompletion(t, completed["D"], exactD)
		v22AssertFourGoals(t, completed, b, admission, councilIntegration)
	})
}

func TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess(t *testing.T) {
	v22InDelegatedUnit(t, "TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess", func() {
		ctx, cancel := context.WithTimeout(context.Background(), v22RealDeadline)
		defer cancel()
		h := v22Start(t, ctx)
		ref := h.create(ctx, "census", v22PlanC())
		councilIntegration := h.admitAcceptedCouncilIntegration(ctx, ref)
		terminal := h.waitTerminal(ctx, ref)
		v22NoContradiction(t, terminal)
		v22AssertProgrammingClosure(t, terminal, councilIntegration)
		h.stop()
		h.census()
	})
}

func TestV22TrustedAttestorToolchainPrerequisite(t *testing.T) {
	root := v22TrustedToolchainRoot(t)
	v22Require(t, filepath.IsAbs(root), "V22 trusted Go toolchain root must be absolute: %q", root)
}

func v22InDelegatedUnit(t *testing.T, testName string, run func()) {
	t.Helper()
	if *v22SystemdInnerUnit == "" {
		t.Parallel()
		v22RunInDelegatedUnit(t, testName)
		return
	}
	v22RequireDelegatedSystemdHarness(t, *v22SystemdInnerUnit)
	v22ConstrainInnerEnvironment(t)
	t.Cleanup(func() { fmt.Printf("%s%s:%s\n", v22SystemdMarker, testName, *v22SystemdInnerUnit) })
	run()
}
func v22RunInDelegatedUnit(t *testing.T, testName string) {
	t.Helper()
	systemdRun := v22Must(exec.LookPath("systemd-run"))
	executable := v22Must(os.Executable())
	environment := v22SystemdEnvironment(t)
	unit := v22RandomSystemdUnit(t)
	t.Cleanup(func() { v22StopAndCollectSystemdUnit(t, unit) })
	args := []string{"--user", "--wait", "--collect", "--pipe", "--service-type=exec", "--same-dir", "--unit=" + unit,
		"--property=Delegate=yes", "--property=DelegateSubgroup=runner",
		"--property=MemoryAccounting=yes", "--property=TasksAccounting=yes",
		"--property=KillMode=control-group", "--property=SendSIGKILL=yes",
		"--property=RuntimeMaxSec=" + v22SystemdRuntimeMax, "--property=TimeoutStopSec=15s",
		"--property=LimitNOFILE=16384", "--property=LimitFSIZE=536870912",
		"--property=LimitAS=8589934592", "--property=LimitCPU=120"}
	args = append(args, environment...)
	args = append(args, executable, "-test.run=^"+testName+"$", "-test.count=1",
		"-test.timeout="+v22SystemdTestTimeout, "-v22-systemd-inner-unit="+unit)
	if testing.Verbose() {
		args = append(args, "-test.v=true")
	}
	command := exec.Command(systemdRun, args...)
	command.Stdin = os.Stdin
	output, runErr := command.CombinedOutput()
	_, _ = os.Stdout.Write(output)
	v22RequireSystemdUnitCollected(t, unit)
	v22Require(t, runErr == nil, "V22 delegated inner test failed: unit=%s err=%v", unit, runErr)
	marker, count := v22SystemdMarker+testName+":"+unit, 0
	for _, line := range bytes.Split(output, []byte("\n")) {
		if string(line) == marker {
			count++
		}
	}
	v22Require(t, count == 1, "V22 delegated inner marker count=%d unit=%s", count, unit)
}
func v22RandomSystemdUnit(t *testing.T) string {
	t.Helper()
	value := make([]byte, 12)
	v22Require(t, func() bool { _, err := rand.Read(value); return err == nil }(), "V22 random systemd unit")
	return "orquesta-v22-" + hex.EncodeToString(value) + ".service"
}
func v22SystemdEnvironment(t *testing.T) []string {
	t.Helper()
	root := t.TempDir()
	v22Require(t, os.Chmod(root, 0o700) == nil, "secure V22 outer temp root")
	home := v22Must(os.UserHomeDir())
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	codexInfo := v22Must(os.Lstat(codexHome))
	codexStat, codexOK := codexInfo.Sys().(*syscall.Stat_t)
	v22Require(t, filepath.IsAbs(codexHome) && filepath.Clean(codexHome) == codexHome &&
		codexInfo.IsDir() && codexInfo.Mode()&os.ModeSymlink == 0 && codexOK &&
		codexStat.Uid == uint32(os.Geteuid()) && codexInfo.Mode().Perm()&0o002 == 0,
		"V22 source Codex home unsafe")
	for _, path := range []string{home, codexHome} {
		v22Require(t, filepath.IsAbs(path) && !strings.ContainsRune(path, 0), "V22 outer environment path invalid: %q", path)
	}
	directories := map[string]string{
		"GOCACHE": filepath.Join(root, "go-cache"), "GOTMPDIR": filepath.Join(root, "go-tmp"),
		"GOMODCACHE": filepath.Join(root, "go-mod-cache"), "GOPATH": filepath.Join(root, "go-path"),
		"TMPDIR": filepath.Join(root, "tmp"),
	}
	for _, path := range directories {
		v22Require(t, os.Mkdir(path, 0o700) == nil, "create V22 outer environment path %s", path)
	}
	pathValue := os.Getenv("PATH")
	v22Require(t, pathValue != "" && !strings.ContainsRune(pathValue, 0), "V22 outer PATH unavailable")
	values := [][2]string{
		{"PATH", pathValue}, {"HOME", home}, {"CODEX_HOME", codexHome},
		{"GOCACHE", directories["GOCACHE"]}, {"GOMODCACHE", directories["GOMODCACHE"]},
		{"GOPATH", directories["GOPATH"]}, {"GOTMPDIR", directories["GOTMPDIR"]},
		{"GOTOOLCHAIN", "auto"}, {"GOENV", "off"}, {"TMPDIR", directories["TMPDIR"]},
	}
	args := make([]string, 0, len(values))
	for _, value := range values {
		args = append(args, "--setenv="+value[0]+"="+value[1])
	}
	for _, name := range []string{"GOFLAGS", "GOMAXPROCS"} {
		if value, found := os.LookupEnv(name); found {
			v22Require(t, value != "" && !strings.ContainsRune(value, 0), "V22 outer %s invalid", name)
			args = append(args, "--setenv="+name+"="+value)
		}
	}
	return args
}
func v22ConstrainInnerEnvironment(t *testing.T) {
	t.Helper()
	names := []string{"PATH", "HOME", "CODEX_HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOFLAGS", "GOMAXPROCS", "GOTOOLCHAIN", "GOENV"}
	values := map[string]string{}
	for _, name := range names {
		if value, found := os.LookupEnv(name); found {
			v22Require(t, value != "" && !strings.ContainsRune(value, 0), "V22 inner %s invalid", name)
			values[name] = value
		}
	}
	for _, required := range []string{"PATH", "HOME", "CODEX_HOME", "TMPDIR", "GOTMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOTOOLCHAIN", "GOENV"} {
		_, found := values[required]
		v22Require(t, found, "V22 inner %s unavailable", required)
	}
	os.Clearenv()
	for _, name := range names {
		if value, found := values[name]; found {
			v22Require(t, os.Setenv(name, value) == nil, "restore V22 inner %s", name)
		}
	}
}
func v22RequireDelegatedSystemdHarness(t *testing.T, unit string) {
	t.Helper()
	properties := v22SystemdProperties(t, unit)
	want := map[string]string{
		"Delegate": "yes", "DelegateSubgroup": "runner", "MemoryAccounting": "yes", "TasksAccounting": "yes",
		"KillMode": "control-group", "SendSIGKILL": "yes",
		"LimitNOFILE": "16384", "LimitNOFILESoft": "16384",
		"LimitFSIZE": "536870912", "LimitFSIZESoft": "536870912",
		"LimitAS": "8589934592", "LimitASSoft": "8589934592",
		"LimitCPU": "120", "LimitCPUSoft": "120",
		"RuntimeMaxUSec": "13min", "TimeoutStopUSec": "15s",
	}
	for property, value := range want {
		v22Require(t, properties[property] == value, "V22 systemd property %s=%q want=%q", property, properties[property], value)
	}
	mainPID, err := strconv.Atoi(properties["MainPID"])
	v22Require(t, err == nil && mainPID == os.Getpid(), "V22 systemd MainPID=%q pid=%d", properties["MainPID"], os.Getpid())
	control := properties["ControlGroup"]
	v22Require(t, control != "" && strings.HasPrefix(control, "/user.slice/") && filepath.Clean(control) == control && filepath.Base(control) == unit, "V22 systemd ControlGroup=%q", control)
	self := v22SelfCgroup(t)
	v22Require(t, self == control+"/runner", "V22 delegated runner=%q want=%q", self, control+"/runner")
	var filesystem syscall.Statfs_t
	v22Require(t, syscall.Statfs("/sys/fs/cgroup", &filesystem) == nil && uint64(filesystem.Type) == 0x63677270, "V22 cgroup v2 unavailable")
	parent := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(control, "/"))
	runner := filepath.Join(parent, "runner")
	v22RequireCgroupOwner(t, parent)
	v22RequireCgroupOwner(t, runner)
	v22Require(t, os.Chmod(parent, 0o700) == nil, "secure V22 delegated cgroup parent")
	info := v22Must(os.Lstat(parent))
	v22Require(t, info.Mode() == os.ModeDir|0o700, "V22 delegated cgroup mode=%v", info.Mode())
	v22RequireCgroupProcesses(t, parent, nil)
	v22RequireCgroupProcesses(t, runner, []int{os.Getpid()})
	v22EnableExactCgroupControllers(t, parent)
}
func v22SystemdProperties(t *testing.T, unit string) map[string]string {
	t.Helper()
	systemctl := v22Must(exec.LookPath("systemctl"))
	requested := "ControlGroup,MainPID,Delegate,DelegateSubgroup,MemoryAccounting,TasksAccounting,KillMode,SendSIGKILL,RuntimeMaxUSec,TimeoutStopUSec," +
		"LimitNOFILE,LimitNOFILESoft,LimitFSIZE,LimitFSIZESoft,LimitAS,LimitASSoft,LimitCPU,LimitCPUSoft"
	output, err := exec.Command(systemctl, "--user", "show", unit, "--property="+requested).CombinedOutput()
	v22Require(t, err == nil, "inspect V22 systemd unit %s: %v: %s", unit, err, output)
	properties := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		name, value, ok := strings.Cut(line, "=")
		if ok {
			properties[name] = value
		}
	}
	return properties
}
func v22RequireSystemdUnitCollected(t *testing.T, unit string) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for {
		state := v22SystemdUnitLoadState(t, unit)
		if state == "not-found" {
			return
		}
		v22Require(t, time.Now().Before(until), "V22 systemd unit not collected: unit=%s state=%s", unit, state)
		time.Sleep(25 * time.Millisecond)
	}
}
func v22StopAndCollectSystemdUnit(t *testing.T, unit string) {
	t.Helper()
	if v22SystemdUnitLoadState(t, unit) == "not-found" {
		return
	}
	systemctl := v22Must(exec.LookPath("systemctl"))
	output, err := exec.Command(systemctl, "--user", "stop", unit).CombinedOutput()
	v22Require(t, err == nil, "stop exact V22 systemd unit: unit=%s err=%v output=%s", unit, err, output)
	v22RequireSystemdUnitCollected(t, unit)
}
func v22SystemdUnitLoadState(t *testing.T, unit string) string {
	t.Helper()
	systemctl := v22Must(exec.LookPath("systemctl"))
	output, err := exec.Command(systemctl, "--user", "show", unit, "--property=LoadState", "--value").CombinedOutput()
	v22Require(t, err == nil, "inspect V22 systemd collection: unit=%s err=%v output=%s", unit, err, output)
	state := strings.TrimSpace(string(output))
	v22Require(t, state != "", "V22 systemd collection state missing: unit=%s", unit)
	return state
}
func v22SelfCgroup(t *testing.T) string {
	t.Helper()
	content := v22Must(os.ReadFile("/proc/self/cgroup"))
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	v22Require(t, len(content) <= 4096 && len(lines) == 1 && strings.HasPrefix(lines[0], "0::/"), "V22 unified cgroup invalid: %q", content)
	return strings.TrimPrefix(lines[0], "0::")
}
func v22RequireCgroupOwner(t *testing.T, path string) {
	t.Helper()
	info := v22Must(os.Lstat(path))
	stat, ok := info.Sys().(*syscall.Stat_t)
	v22Require(t, ok && int(stat.Uid) == os.Geteuid(), "V22 cgroup owner invalid: path=%s", path)
}
func v22RequireCgroupProcesses(t *testing.T, root string, want []int) {
	t.Helper()
	content := v22Must(os.ReadFile(filepath.Join(root, "cgroup.procs")))
	fields := strings.Fields(string(content))
	v22Require(t, len(fields) == len(want), "V22 cgroup processes %s=%v want=%v", root, fields, want)
	for index, value := range fields {
		pid, err := strconv.Atoi(value)
		v22Require(t, err == nil && pid == want[index], "V22 cgroup process %s[%d]=%q want=%d", root, index, value, want[index])
	}
}
func v22EnableExactCgroupControllers(t *testing.T, parent string) {
	t.Helper()
	want := map[string]bool{"cpu": true, "memory": true, "pids": true}
	available := strings.Fields(string(v22Must(os.ReadFile(filepath.Join(parent, "cgroup.controllers")))))
	for controller := range want {
		found := false
		for _, value := range available {
			found = found || value == controller
		}
		v22Require(t, found, "V22 cgroup controller unavailable: %s in %v", controller, available)
	}
	control := filepath.Join(parent, "cgroup.subtree_control")
	for _, controller := range strings.Fields(string(v22Must(os.ReadFile(control)))) {
		if !want[controller] {
			v22Require(t, os.WriteFile(control, []byte("-"+controller), 0o600) == nil, "disable V22 cgroup controller %s", controller)
		}
	}
	v22Require(t, os.WriteFile(control, []byte("+cpu +memory +pids"), 0o600) == nil, "enable V22 delegated cgroup controllers")
	enabled := strings.Join(strings.Fields(string(v22Must(os.ReadFile(control)))), " ")
	v22Require(t, enabled == "cpu memory pids", "V22 delegated cgroup controllers=%q", enabled)
}

func v22Start(t *testing.T, ctx context.Context) *v22Harness {
	t.Helper()
	root := t.TempDir()
	codex := v22Must(exec.LookPath("codex"))
	git := v22Must(exec.LookPath("git"))
	goTool := v22Must(exec.LookPath("go"))
	v22Require(t, os.Chmod(root, 0o700) == nil, "secure V22 harness root")
	bwrap, toolchain := v22AttestorPrerequisites(t)
	v22Seed(t, ctx, root, git)
	h := &v22Harness{t: t, root: root, binary: filepath.Join(root, "orquesta"), config: filepath.Join(root, "orquesta.toml"), state: filepath.Join(root, "state", "orquesta.sqlite"), endpoint: "http://127.0.0.1:" + v22Port(t) + "/mcp", codexHome: v22PrivateCodexHome(t, root)}
	v22Require(t, os.WriteFile(h.config, []byte(v22Config(root, strings.TrimSuffix(h.endpoint, "/mcp")[len("http://127.0.0.1:"):], codex, bwrap, toolchain, v22ConfiguredCgroupRoot(t))), 0o600) == nil, "write V22 config")
	t.Cleanup(h.close) // Registered before any owned process can be launched.
	build := exec.CommandContext(ctx, goTool, "build", "-mod=vendor", "-trimpath", "-buildvcs=false", "-o", h.binary, "./cmd/orquesta")
	build.Dir = evidenceRepositoryRoot(t)
	out, err := build.CombinedOutput()
	v22Require(t, err == nil, "build external cmd/orquesta: %v\n%s", err, out)
	h.launch(ctx)
	return h
}
func (h *v22Harness) launch(ctx context.Context) {
	h.t.Helper()
	log := v22Must(os.OpenFile(filepath.Join(h.root, fmt.Sprintf("server-%d.log", time.Now().UnixNano())), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600))
	cmd := exec.Command(h.binary, "serve", "--config", h.config)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout, cmd.Stderr = log, log
	cmd.Env = v22EnvironmentWithCodexHome(h.codexHome)
	v22Require(h.t, cmd.Start() == nil, "start external cmd/orquesta")
	h.server, h.log = cmd, log
	for until := time.Now().Add(30 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		if h.connect(ctx) == nil {
			return
		}
	}
	out, _ := os.ReadFile(log.Name())
	h.t.Fatalf("external MCP server unavailable: %s", out)
}
func (h *v22Harness) connect(ctx context.Context) error {
	if h.session != nil {
		_ = h.session.Close()
		h.session = nil
	}
	token, err := os.ReadFile(filepath.Join(h.root, "secrets", "local-owner.token"))
	if err != nil {
		return err
	}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "orquesta-v22-real-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{Endpoint: h.endpoint, HTTPClient: &http.Client{Transport: v22Bearer{strings.TrimSpace(string(token))}}}, nil)
	if err == nil {
		h.session = session
	}
	return err
}
func (h *v22Harness) restart(ctx context.Context) { h.releaseLog(); h.server = nil; h.launch(ctx) }
func (h *v22Harness) close()                      { h.stop(); h.census() }
func (h *v22Harness) releaseLog() {
	if h.log != nil {
		_ = h.log.Close()
		h.log = nil
	}
}
func (h *v22Harness) disconnect() {
	if h.session != nil {
		_ = h.session.Close()
		h.session = nil
	}
}
func (h *v22Harness) stop() { h.finish(os.Interrupt, false) }
func (h *v22Harness) kill() { h.finish(os.Kill, true) }
func (h *v22Harness) finish(signal os.Signal, requireFailure bool) {
	h.disconnect()
	if h.server == nil || h.server.Process == nil {
		v22Require(h.t, !requireFailure, "no server to SIGKILL")
		h.releaseLog()
		return
	}
	v22Require(h.t, h.server.Process.Signal(signal) == nil, "server signal %s", signal)
	done := make(chan error, 1)
	go func() { done <- h.server.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-time.After(15 * time.Second):
		_ = h.server.Process.Kill()
		err = <-done
	}
	v22Require(h.t, !requireFailure || err != nil, "SIGKILL unexpectedly clean")
	h.server = nil
	h.releaseLog()
}

func (h *v22Harness) backup(ctx context.Context) application.BackupRef {
	var ref application.BackupRef
	h.recovery(ctx, func(recovery *statesqlite.Recovery) {
		receipt := v22Must(recovery.CreateBackup(ctx))
		v22Must(recovery.VerifyBackup(ctx, receipt.Ref))
		ref = receipt.Ref
	})
	return ref
}
func (h *v22Harness) verifyRestoreCopy(ctx context.Context, backup application.BackupRef, want v22RestoreExpected) {
	var restored string
	h.recovery(ctx, func(recovery *statesqlite.Recovery) {
		target := v22Must(application.NewRecoveryTargetRef("recovery-target:v22-real"))
		v22Must(recovery.RestoreBackup(ctx, backup, target))
		restored = v22Must(recovery.TargetPath(target))
	})
	copyRepository := v22Must(statesqlite.Open(ctx, statesqlite.Options{Path: restored, BusyTimeout: 5 * time.Second, MaxOpenConnections: 1}))
	defer copyRepository.Close()
	goalRef := mustGoalRef(want.Goal)
	executionRef := v22Must(goal.NewExecutionRef(want.Execution))
	workItemRef := v22Must(goal.NewWorkItemRef(want.Work))
	record := v22Must(copyRepository.GetGoal(ctx, goalRef))
	v22Require(h.t, h.state != restored && record.Goal.Ref() == goalRef && record.Goal.Project().String() == want.Project &&
		string(record.Goal.State()) == want.GoalState && record.Goal.PlanGeneration() == goal.PlanGeneration(want.Plan) &&
		record.Goal.AppSpec().Generation() == goal.AppSpecGeneration(want.App) && record.Goal.SpecHash() == want.Spec &&
		uint64(record.Goal.WorkItemCount()) == want.WorkItems && uint64(len(record.Executions)) == want.Executions,
		"restored D aggregate differs from pre-backup tuple: state=%q restored=%q goal=%+v want=%+v", h.state, restored, record.Goal, want)
	execution := record.Executions[0]
	item, itemOK := record.Goal.WorkItem(execution.WorkItemRef)
	aggregateExecution, bound := item.Execution()
	v22Require(h.t, itemOK && bound && aggregateExecution == executionRef && item.Ref() == workItemRef &&
		execution.Ref == executionRef && execution.GoalRef == goalRef && execution.WorkItemRef == workItemRef &&
		string(execution.State) == want.ExecutionState && execution.AttemptNo == want.Attempt &&
		execution.MaxExecutionAttempts == want.MaxAttempts && execution.PlanGeneration == goal.PlanGeneration(want.Plan) &&
		execution.AppSpecGeneration == goal.AppSpecGeneration(want.App) && execution.SpecHash == want.Spec &&
		execution.ReplacesExecutionRef.String() == want.Replaces && string(execution.Purpose) == want.Purpose,
		"restored D execution differs from pre-backup tuple: item=%+v execution=%+v want=%+v", item, execution, want)
}
func (h *v22Harness) recovery(ctx context.Context, run func(*statesqlite.Recovery)) {
	repository := v22Must(statesqlite.Open(ctx, statesqlite.Options{Path: h.state, BusyTimeout: 5 * time.Second, MaxOpenConnections: 8}))
	defer repository.Close()
	recovery := v22Must(statesqlite.NewRecovery(statesqlite.RecoveryOptions{Repository: repository, BackupRoot: filepath.Join(h.root, "backup"), RestoreRoot: filepath.Join(h.root, "restore")}))
	defer recovery.Close()
	run(recovery)
}
func v22RestoreExpectation(t *testing.T, projection v22GoalProjection) v22RestoreExpected {
	t.Helper()
	view, executions := projection.object("goal"), projection.objects("executions")
	v22Require(t, view.text("state") == "running" && len(executions) == 1, "D pre-backup projection not exact/running: %+v", projection)
	execution := executions[0]
	expected := v22RestoreExpected{Project: view.text("project_ref"), Goal: view.text("goal_ref"), Spec: view.text("spec_hash"),
		GoalState: view.text("state"), Work: execution.text("work_item_ref"), Execution: execution.text("execution_ref"),
		Replaces: execution.text("replaces_execution_ref"), ExecutionState: execution.text("state"), Purpose: execution.text("purpose"),
		Plan: view.number("plan_generation"), App: view.number("app_spec_generation"), Attempt: execution.number("attempt_no"),
		MaxAttempts: execution.number("max_attempts"), WorkItems: view.number("work_item_count"), Executions: projection.number("execution_count")}
	v22Require(t, expected.Project != "" && expected.Goal != "" && expected.Spec != "" && expected.Work != "" && expected.Execution != "" &&
		expected.Plan > 0 && expected.App > 0 && expected.Attempt > 0 && expected.MaxAttempts > 0 && expected.WorkItems == 1 && expected.Executions == 1,
		"D pre-backup tuple incomplete: %+v", expected)
	return expected
}

func (h *v22Harness) create(ctx context.Context, id string, plan map[string]any) string {
	r := h.call(ctx, "orquesta.goals.create", map[string]any{"statement": "V22 " + id + " real Codex acceptance", "confirm": true, "plan": plan})
	var data v22Object
	v22Data(h.t, r, &data)
	ref := data.object("goal").text("goal_ref")
	v22Require(h.t, ref != "", "create %s omitted goal_ref", id)
	return ref
}
func (h *v22Harness) call(ctx context.Context, tool string, payload map[string]any) v22Output {
	h.mu.Lock()
	h.request++
	ref := fmt.Sprintf("request:v22-real:%d", h.request)
	h.mu.Unlock()
	result, err := h.session.CallTool(ctx, &sdkmcp.CallToolParams{Name: tool, Arguments: map[string]any{"version": "1", "project_ref": "project:v22-real", "request_ref": ref, "payload": payload}})
	v22Require(h.t, err == nil, "MCP %s: %v", tool, err)
	raw := v22Must(json.Marshal(result.StructuredContent))
	var out v22Output
	v22Require(h.t, json.Unmarshal(raw, &out) == nil, "MCP %s decode", tool)
	v22Require(h.t, !result.IsError && out.Result.Failure == nil, "MCP %s rejected public operation: %+v", tool, out.Result.Failure)
	return out
}
func (h *v22Harness) get(ctx context.Context, ref string) v22GoalProjection {
	r := h.call(ctx, "orquesta.goals.get", map[string]any{"goal_ref": ref})
	var g v22GoalProjection
	v22Data(h.t, r, &g)
	return g
}
func (h *v22Harness) admitAcceptedCouncilIntegration(ctx context.Context, raw string) v22CouncilIntegration {
	repository := v22Must(statesqlite.Open(ctx, statesqlite.Options{Path: h.state, BusyTimeout: 5 * time.Second, MaxOpenConnections: 1}))
	defer repository.Close()
	goalRef := mustGoalRef(raw)
	var accepted v22CouncilIntegration
	for {
		record := v22Must(repository.GetGoal(ctx, goalRef))
		for _, decision := range record.CouncilDecisions {
			if decision.Decision.Outcome != council.OutcomeAccepted {
				continue
			}
			v22Require(h.t, accepted.DecisionRef == "" && len(record.CouncilDecisions) == 1,
				"multiple V22 Council decisions: %+v", record.CouncilDecisions)
			for _, round := range record.CouncilRounds {
				if round.Ref == decision.RoundRef && round.SubjectDigest == decision.SubjectDigest {
					v22Require(h.t, round.GoalRef == goalRef && round.ChangeSetRef != "" &&
						decision.Ref != "" && decision.DecisionDigest != "" &&
						decision.Decision.SubjectDigest == string(decision.SubjectDigest),
						"accepted V22 Council decision lacks exact round: round=%+v decision=%+v", round, decision)
					for _, change := range record.ChangeSets {
						if change.Ref.String() != round.ChangeSetRef {
							continue
						}
						v22Require(h.t, accepted.ChangeRef == "" && change.GoalRef == goalRef &&
							change.WorkItemRef == round.WorkItemRef &&
							round.Subject.GoalRef == raw &&
							round.Subject.WorkItemRef == change.WorkItemRef.String() &&
							round.Subject.ChangeSetRef == change.Ref.String() &&
							round.Subject.SpecHash == change.SpecHash &&
							round.Subject.PlanGeneration == uint64(change.PlanGeneration) &&
							round.Subject.AppSpecGeneration == uint64(change.AppSpecGeneration),
							"accepted V22 Council round/change mismatch: round=%+v change=%+v", round, change)
						accepted = v22CouncilIntegration{
							ChangeRef: change.Ref.String(), WorkItemRef: change.WorkItemRef.String(),
							ExecutionRef: change.ExecutionRef.String(), BaseOID: change.BaseOID,
							SubjectDigest: string(decision.SubjectDigest),
							DecisionRef:   decision.Ref, DecisionDigest: string(decision.DecisionDigest),
						}
					}
					v22RequireAcceptedCouncilFacts(h.t, record, decision)
				}
			}
			v22Require(h.t, accepted.ChangeRef != "", "accepted V22 Council decision has no matching round: %+v", decision)
		}
		if accepted.DecisionRef != "" {
			break
		}
		v22Require(h.t, !record.Goal.IsTerminal(), "V22 Goal became terminal before accepted Council decision: %+v", record.Goal)
		v22Wait(h.t, ctx, "accepted V22 Council decision")
	}
	record := v22Must(repository.GetGoal(ctx, goalRef))
	v22Require(h.t, !record.Goal.IsTerminal(), "V22 Goal became terminal before public integration: %+v", record.Goal)
	changeFound := false
	for _, change := range record.ChangeSets {
		changeFound = changeFound || change.Ref.String() == accepted.ChangeRef &&
			change.GoalRef == goalRef && change.WorkItemRef.String() == accepted.WorkItemRef &&
			change.ExecutionRef.String() == accepted.ExecutionRef
	}
	v22Require(h.t, changeFound, "accepted V22 Council change disappeared before public integration: %+v", accepted)
	response := h.call(ctx, "orquesta.changes.list", map[string]any{"limit": 100})
	var data v22Object
	v22Data(h.t, response, &data)
	var pending v22Object
	for _, change := range data.objects("changes") {
		if change.text("goal_ref") == raw && change.text("change_ref") == accepted.ChangeRef {
			v22Require(h.t, pending == nil, "duplicate V22 pending change via MCP: %+v", change)
			pending = change
		}
	}
	v22Require(h.t, pending != nil, "accepted V22 Council change omitted from synchronous public pending list: %+v", accepted)
	v22Require(h.t, pending.text("work_item_ref") == accepted.WorkItemRef &&
		pending.text("execution_ref") == accepted.ExecutionRef &&
		pending.text("status") == "" && pending.text("target_oid") == accepted.BaseOID &&
		pending.text("target_oid") != "",
		"accepted fresh V22 Council change is not publicly integrable: %+v", pending)
	response = h.call(ctx, "orquesta.changes.integrate", map[string]any{
		"goal_ref": raw, "change_ref": accepted.ChangeRef, "expected_target_oid": pending.text("target_oid"),
	})
	var integrated v22Object
	v22Data(h.t, response, &integrated)
	accepted.ActionRef = integrated.text("action_ref")
	v22Require(h.t, accepted.ActionRef != "", "V22 public integration omitted action_ref")
	v22RequireCouncilIntegrationIntent(h.t, v22Must(repository.GetGoal(ctx, goalRef)), accepted)
	return accepted
}
func v22RequireAcceptedCouncilFacts(t *testing.T, record application.GoalRecord, decision application.CouncilDecisionRecord) {
	t.Helper()
	purposes := map[council.Role]application.ExecutionPurpose{
		council.RoleProposer: application.ExecutionPurposeCouncilProposer,
		council.RoleCritic:   application.ExecutionPurposeCouncilCritic,
		council.RoleArbiter:  application.ExecutionPurposeCouncilArbiter,
	}
	seen := map[council.Role]bool{}
	for _, fact := range record.CouncilFacts {
		if fact.SubjectDigest != string(decision.SubjectDigest) {
			continue
		}
		purpose, known := purposes[fact.Role]
		v22Require(t, known && !seen[fact.Role] && fact.ExecutionRef != "" && fact.ExecutionAttempt > 0 &&
			fact.LaunchReceiptRef != "" && fact.ExternalRef != "" && fact.ArtifactRef != "" &&
			fact.ArtifactDigest != "" && fact.IdempotencyKey != "",
			"invalid/duplicate V22 Council fact for accepted subject: %+v", fact)
		seen[fact.Role] = true
		exactExecution := false
		for _, execution := range record.Executions {
			exactExecution = exactExecution || execution.Ref.String() == fact.ExecutionRef &&
				execution.AttemptNo == fact.ExecutionAttempt && execution.Purpose == purpose &&
				execution.State == application.ExecutionSucceeded &&
				execution.CouncilSubjectDigest == decision.SubjectDigest &&
				execution.LaunchReceiptRef == fact.LaunchReceiptRef &&
				execution.ExternalRef == fact.ExternalRef
		}
		v22Require(t, exactExecution, "V22 Council fact lacks exact successful execution: %+v", fact)
	}
	v22Require(t, len(seen) == 3 && seen[council.RoleProposer] && seen[council.RoleCritic] && seen[council.RoleArbiter],
		"accepted V22 Council subject lacks exact proposer/critic/arbiter facts: %+v", seen)
}
func v22RequireCouncilIntegrationIntent(t *testing.T, record application.GoalRecord, accepted v22CouncilIntegration) {
	t.Helper()
	found := 0
	for _, intent := range record.EffectIntents {
		if intent.ActionRef != accepted.ActionRef {
			continue
		}
		found++
		resolution := intent.CouncilResolution
		v22Require(t, intent.ActionKind == application.ActionIntegrateChange &&
			intent.Subject.GoalRef.String() == record.Goal.Ref().String() &&
			intent.Subject.WorkItemRef.String() == accepted.WorkItemRef &&
			intent.Subject.ExecutionRef.String() == accepted.ExecutionRef &&
			resolution != nil && string(resolution.SubjectDigest) == accepted.SubjectDigest &&
			resolution.DecisionRef == accepted.DecisionRef &&
			string(resolution.DecisionDigest) == accepted.DecisionDigest &&
			resolution.SkipRef == "" && resolution.SkipDigest == "",
			"V22 public integration action lacks exact Council resolution: intent=%+v accepted=%+v", intent, accepted)
	}
	v22Require(t, found == 1, "V22 public integration action intent count=%d want=1: %+v", found, accepted)
}
func (h *v22Harness) waitRunning(ctx context.Context, refs ...string) map[string]v22ExecutionProjection {
	for {
		running := map[string]v22ExecutionProjection{}
		for _, ref := range refs {
			g := h.get(ctx, ref)
			goalView := g.object("goal")
			v22Require(h.t, !v22Terminal(goalView.text("state")), "Goal %s became terminal before exact running observation: %s", ref, goalView.text("state"))
			for _, execution := range g.objects("executions") {
				if goalView.text("state") == "running" && execution.text("state") == "running" && execution.text("execution_ref") != "" && execution.text("work_item_ref") != "" && execution.number("attempt_no") > 0 && execution.number("plan_generation") == goalView.number("plan_generation") && execution.number("app_spec_generation") == goalView.number("app_spec_generation") {
					running[ref] = execution
					break
				}
			}
		}
		if len(running) == len(refs) {
			return running
		}
		v22Wait(h.t, ctx, "parallel Goals")
	}
}
func v22AssertAdoptedCompletion(t *testing.T, d v22GoalProjection, exact string) {
	t.Helper()
	for _, execution := range d.objects("executions") {
		if execution.text("execution_ref") == exact && execution.text("purpose") == "work" && execution.text("state") == "succeeded" {
			return
		}
	}
	t.Fatalf("restart did not complete the pre-crash D execution %s: %+v", exact, d.objects("executions"))
}
func (h *v22Harness) runningExecution(ctx context.Context, ref string) v22ExecutionProjection {
	return h.waitRunning(ctx, ref)[ref]
}
func (h *v22Harness) waitAdopted(ctx context.Context, ref string, want v22ExecutionProjection, process v22ProcessRecord) {
	for {
		g := h.get(ctx, ref)
		for _, execution := range g.objects("executions") {
			if execution.text("execution_ref") == want.text("execution_ref") && execution.text("state") == "running" && execution.text("replaces_execution_ref") == "" {
				alive, err := v22ProcessAlive(process)
				v22Require(h.t, err == nil && alive, "restart did not adopt exact D process: alive=%v err=%v", alive, err)
				return
			}
		}
		v22Wait(h.t, ctx, "D exact execution adoption")
	}
}
func (h *v22Harness) waitProgress(ctx context.Context, ref string, before v22GoalProjection) {
	for {
		after := h.get(ctx, ref)
		state := after.object("goal").text("state")
		v22Require(h.t, state != "failed" && state != "cancelled" && state != "stopped", "independent Goal damaged by B cancellation: %+v", after.object("goal"))
		if after.object("goal").number("revision") > before.object("goal").number("revision") || after.number("artifact_count") > before.number("artifact_count") || v22ExecutionStates(after) != v22ExecutionStates(before) {
			return
		}
		v22Wait(h.t, ctx, "independent Goal progress")
	}
}
func (h *v22Harness) waitTerminal(ctx context.Context, ref string) v22GoalProjection {
	for {
		g := h.get(ctx, ref)
		if v22Terminal(g.object("goal").text("state")) {
			return g
		}
		v22Wait(h.t, ctx, "terminal Goal")
	}
}
func v22Terminal(state string) bool {
	return state == "succeeded" || state == "failed" || state == "cancelled" || state == "stopped"
}
func (h *v22Harness) assertMailboxArtifactIsolation(ctx context.Context, goalRaw string) v22MailboxFence {
	for {
		admission, ok := h.completedAdmission(ctx, goalRaw)
		items := map[string]v22Object{}
		for _, item := range h.get(ctx, goalRaw).objects("work_items") {
			items[item.text("execution_ref")] = item
		}
		source, recipient := items[admission.SourceExecution], items[admission.RecipientExecution]
		var sibling string
		for execution, item := range items {
			if execution != admission.SourceExecution && execution != admission.RecipientExecution && len(item.strings("artifact_refs")) == 1 {
				sibling = item.strings("artifact_refs")[0]
			}
		}
		if ok && len(source.strings("artifact_refs")) == 1 && recipient.text("state") == "running" && sibling != "" {
			fence := v22MailboxFence{Admission: admission, Delivered: source.strings("artifact_refs")[0], Sibling: sibling}
			return h.probeMailboxArtifactIsolation(ctx, goalRaw, fence, "initial")
		}
		v22Wait(h.t, ctx, "A completed admission, delivered/sibling artifacts and live recipient")
	}
}
func (h *v22Harness) probeMailboxArtifactIsolation(ctx context.Context, goalRaw string, fence v22MailboxFence, phase string) v22MailboxFence {
	deliveredRequest, siblingRequest := "authorization:v22-real:"+phase+":delivered", "authorization:v22-real:"+phase+":sibling"
	v22Require(h.t, h.liveAuthorizationReceiptCount(ctx, deliveredRequest, siblingRequest) == 0, "%s authorization probe already mutated live DB", phase)
	isolated := h.isolatedStateCopy(ctx, phase)
	isolatedRepository := v22Must(statesqlite.Open(ctx, statesqlite.Options{Path: isolated.Path, BusyTimeout: 5 * time.Second, MaxOpenConnections: 1}))
	defer isolatedRepository.Close()
	admission, delivered, sibling := fence.Admission, fence.Delivered, fence.Sibling
	recipientRef := v22Must(goal.NewExecutionRef(admission.RecipientExecution))
	authority, err := isolatedRepository.ExecutionSessionAuthority(ctx, recipientRef, "execution_token")
	v22Require(h.t, err == nil && authority.ServicePrincipal.Ref.String() == admission.RecipientPrincipal, "recipient authority drift: %+v admission=%+v err=%v", authority, admission, err)
	record := v22Must(isolatedRepository.GetGoal(ctx, mustGoalRef(goalRaw)))
	sourceExact, recipientExact := false, false
	for _, execution := range record.Executions {
		if execution.Ref.String() == admission.SourceExecution {
			source := v22Must(application.DeriveExecutionSessionAuthority(application.ExecutionSessionRequest(record.Goal, execution), "execution_token"))
			sourceExact = source.ServicePrincipal.Ref.String() == admission.SourcePrincipal
		}
		if execution.Ref == recipientRef {
			expected := v22Must(application.DeriveExecutionSessionAuthority(application.ExecutionSessionRequest(record.Goal, execution), "execution_token"))
			recipientExact = authority == expected
		}
	}
	v22Require(h.t, sourceExact && recipientExact, "%s exact execution authority drift: authority=%+v admission=%+v", phase, authority, admission)
	authorize := func(requestRef, artifact string) (identity.AuthorizationReceipt, error) {
		request := v22Must(identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{RequestRef: requestRef, Principal: authority.ServicePrincipal, ProjectRef: record.Goal.Project(), Permission: identity.PermissionArtifactsRead, ResourceRef: artifact, RequestedAt: time.Now().UTC()}))
		return isolatedRepository.Authorize(ctx, request)
	}
	receipt, deliveredErr := authorize(deliveredRequest, delivered)
	_, siblingErr := authorize(siblingRequest, sibling)
	v22Require(h.t, delivered != sibling && deliveredErr == nil && receipt.Decision().Outcome() == identity.AuthorizationAllowed &&
		receipt.Decision().Role() == identity.RoleExecutionService && receipt.Decision().Request().RequestRef() == deliveredRequest &&
		errors.Is(siblingErr, application.ErrForbidden), "%s artifact delivery fence delivered=%s sibling=%s receipt=%+v errors=%v/%v", phase, delivered, sibling, receipt, deliveredErr, siblingErr)
	v22Require(h.t, isolatedRepository.Close() == nil && h.liveAuthorizationReceiptCount(ctx, deliveredRequest, siblingRequest) == 0,
		"%s isolated authorization leaked into live DB", phase)
	if phase == "initial" {
		fence.InitialReceipt, fence.InitialReceiptRef, fence.InitialRequestRef = receipt, receipt.Ref(), deliveredRequest
		fence.InitialCopyPath, fence.InitialCopyTarget = isolated.Path, isolated.Target
		v22Require(h.t, fence.InitialReceiptRef != "" && fence.InitialReceipt.Decision().Request().RequestRef() == fence.InitialRequestRef,
			"initial authorization receipt/request not exact: %+v", fence)
	} else {
		v22Require(h.t, fence.InitialReceipt.Ref() == fence.InitialReceiptRef &&
			fence.InitialReceipt.Decision().Request().RequestRef() == fence.InitialRequestRef &&
			deliveredRequest != fence.InitialRequestRef && receipt.Ref() != fence.InitialReceiptRef &&
			receipt.Decision().Request().RequestRef() == deliveredRequest &&
			isolated.Path != fence.InitialCopyPath && isolated.Target != fence.InitialCopyTarget,
			"restart authorization reused initial receipt/request: initial=%+v new=%+v request=%s", fence.InitialReceipt, receipt, deliveredRequest)
	}
	return fence
}
func (h *v22Harness) isolatedStateCopy(ctx context.Context, phase string) v22IsolatedCopy {
	var isolated v22IsolatedCopy
	h.recovery(ctx, func(recovery *statesqlite.Recovery) {
		receipt := v22Must(recovery.CreateBackup(ctx))
		v22Must(recovery.VerifyBackup(ctx, receipt.Ref))
		target := v22Must(application.NewRecoveryTargetRef("recovery-target:v22-auth-" + phase))
		v22Must(recovery.RestoreBackup(ctx, receipt.Ref, target))
		isolated = v22IsolatedCopy{Path: v22Must(recovery.TargetPath(target)), Target: target.String()}
	})
	v22Require(h.t, isolated.Path != "" && isolated.Path != h.state && isolated.Target != "",
		"%s authorization copy not isolated: state=%q copy=%+v", phase, h.state, isolated)
	return isolated
}
func (h *v22Harness) liveAuthorizationReceiptCount(ctx context.Context, first, second string) int {
	database := v22Must(sql.Open("sqlite", h.state))
	defer database.Close()
	var count int
	err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM authorization_receipts WHERE request_ref=? OR request_ref=?`, first, second).Scan(&count)
	v22Require(h.t, err == nil, "read live authorization receipts: %v", err)
	return count
}
func (h *v22Harness) completedAdmission(ctx context.Context, goalRef string) (v22AdmissionIdentity, bool) {
	database := v22Must(sql.Open("sqlite", h.state))
	defer database.Close()
	var found v22AdmissionIdentity
	err := database.QueryRowContext(ctx, `SELECT envelope.ref,admission.ref,envelope.source_principal_ref,envelope.source_execution_ref,envelope.recipient_principal_ref,envelope.recipient_execution_ref FROM mailbox_envelopes envelope JOIN mailbox_admission_receipts admission ON admission.mailbox_message_ref=envelope.ref JOIN outbox action ON action.kind='admit_mailbox' AND action.goal_ref=envelope.goal_ref AND action.work_item_ref=envelope.child_work_item_ref AND action.execution_ref=envelope.source_execution_ref WHERE envelope.goal_ref=? AND action.completed_at IS NOT NULL`, goalRef).Scan(&found.Message, &found.Admission, &found.SourcePrincipal, &found.SourceExecution, &found.RecipientPrincipal, &found.RecipientExecution)
	if errors.Is(err, sql.ErrNoRows) {
		return found, false
	}
	v22Require(h.t, err == nil, "read completed admission: %v", err)
	return found, true
}
func mustGoalRef(raw string) goal.GoalRef { return v22Must(goal.NewGoalRef(raw)) }
func (h *v22Harness) assertSucceededGoal(ctx context.Context, raw string) {
	repository := v22Must(statesqlite.Open(ctx, statesqlite.Options{Path: h.state, BusyTimeout: 5 * time.Second, MaxOpenConnections: 1}))
	defer repository.Close()
	ref := mustGoalRef(raw)
	record := v22Must(repository.GetGoal(ctx, ref))
	v22Require(h.t, record.Goal.Ref() == ref && record.Goal.State() == goal.GoalStateSucceeded && record.Goal.IsTerminal(), "restarted A Goal not terminal/succeeded: %+v", record.Goal)
}
func (h *v22Harness) census() {
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		if len(h.liveProcesses()) == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	if records := h.liveProcesses(); len(records) > 0 {
		for _, record := range records {
			v22KillExact(record)
		}
		h.t.Errorf("owned exact processes survived cleanup and were killed: %+v", records)
	}
}
func (h *v22Harness) liveProcesses() []v22ProcessRecord {
	records, live := v22ProcessRecords(h.t, h.root), []v22ProcessRecord{}
	for _, record := range records {
		alive, err := v22ProcessGroupAlive(record)
		v22Require(h.t, err == nil, "process group liveness: %v", err)
		if alive {
			live = append(live, record)
		}
	}
	return live
}

type v22Bearer struct{ token string }

func (b v22Bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.Header = r.Header.Clone()
	c.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(c)
}
func v22Data(t *testing.T, r v22Output, target any) {
	t.Helper()
	v22Require(t, len(r.Result.Data) > 0, "MCP response omitted data")
	v22Require(t, json.Unmarshal(r.Result.Data, target) == nil, "decode MCP data")
}
func v22Wait(t *testing.T, ctx context.Context, what string) {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatalf("wait %s: %v", what, ctx.Err())
	case <-time.After(300 * time.Millisecond):
	}
}

func v22AssertFourGoals(t *testing.T, done map[string]v22GoalProjection, b v22GoalProjection, admission v22AdmissionIdentity, councilIntegration v22CouncilIntegration) {
	t.Helper()
	state := b.object("goal").text("state")
	v22Require(t, state == "cancelled", "B=%s", state)
	v22AssertMailboxClosure(t, done["A"], admission)
	v22AssertProgrammingClosure(t, done["C"], councilIntegration)
	work := 0
	for _, execution := range done["D"].objects("executions") {
		if execution.text("purpose") == "work" {
			work++
			v22Require(t, execution.text("replaces_execution_ref") == "", "D duplicated/replaced work after restart: %+v", execution)
		}
	}
	v22Require(t, work == 1, "D work execution count=%d want exactly one", work)
}
func v22NoContradiction(t *testing.T, g v22GoalProjection) {
	t.Helper()
	goalView, executions := g.object("goal"), g.objects("executions")
	v22Require(t, goalView.text("state") == "succeeded" && g.number("execution_count") == uint64(len(executions)) && g.number("artifact_count") > 0, "terminal projection contradiction: %+v", g)
	seen := map[string]bool{}
	for _, execution := range executions {
		ref := execution.text("execution_ref")
		v22Require(t, ref != "" && execution.text("work_item_ref") != "" && execution.number("attempt_no") > 0 && execution.number("plan_generation") == goalView.number("plan_generation") && execution.number("app_spec_generation") == goalView.number("app_spec_generation") && execution.text("state") == "succeeded" && execution.text("failure_code") == "" && !seen[ref], "execution contradiction: %+v", execution)
		seen[ref] = true
	}
	for _, work := range g.objects("work_items") {
		v22Require(t, work.text("state") == "succeeded" && work.text("execution_ref") != "" && len(work.strings("artifact_refs")) > 0 && len(work.strings("attestation_refs")) > 0 && work.text("interrupt_code") == "", "work item contradiction: %+v", work)
	}
}

func (h *v22Harness) waitCancelled(ctx context.Context, ref string, execution v22ExecutionProjection, process v22ProcessRecord) v22GoalProjection {
	h.t.Helper()
	terminal := h.waitTerminal(ctx, ref)
	v22Require(h.t, terminal.object("goal").text("state") == "cancelled", "B=%s, want controlled cancellation", terminal.object("goal").text("state"))
	v22AssertProcessGone(h.t, process)
	v22AssertStopEvidence(h.t, process)
	for {
		b := h.get(ctx, ref)
		goalView, executionSettled := b.object("goal"), true
		v22Require(h.t, goalView.text("state") == "cancelled", "B changed after cancellation: %+v", goalView)
		for _, current := range b.objects("executions") {
			if current.text("execution_ref") == execution.text("execution_ref") && current.text("state") == "running" {
				executionSettled = false
			}
		}
		for _, control := range b.objects("controls") {
			if executionSettled && control.text("operation") == "cancel" && control.text("target") == "goal" && control.text("status") == "confirmed" && control.text("receipt_ref") != "" && control.number("plan_generation") == goalView.number("plan_generation") && control.number("app_spec_generation") == goalView.number("app_spec_generation") {
				return b
			}
		}
		v22Wait(h.t, ctx, "B exact confirmed cancellation")
	}
}

func v22AssertMailboxClosure(t *testing.T, a v22GoalProjection, want v22AdmissionIdentity) {
	t.Helper()
	var parent, child v22Object
	items := map[string]v22Object{}
	for _, item := range a.objects("work_items") {
		items[item.text("work_item_ref")] = item
		if item.flag("handoff_required") {
			child = item
		}
	}
	if child != nil {
		parent = items[child.text("parent_work_item_ref")]
	}
	v22Require(t, parent != nil && child != nil && parent.text("execution_ref") != "" && child.text("execution_ref") != "" && parent.text("execution_ref") != child.text("execution_ref") && len(parent.strings("artifact_refs")) > 0 && len(child.strings("artifact_refs")) > 0, "A lacks exact admitted/consumed/acknowledged handoff closure: parent=%+v child=%+v", parent, child)
	receipts := a.objects("mailbox_receipts")
	v22Require(t, len(receipts) == 1, "A mailbox receipts=%d want exactly one and zero orphans: %+v", len(receipts), receipts)
	receipt := receipts[0]
	admission, consumption, acknowledgement := receipt.text("admission_ref"), receipt.text("consumption_ref"), receipt.text("acknowledgement_ref")
	v22Require(t, receipt.text("message_ref") == want.Message && admission == want.Admission && receipt.text("state") == "acknowledged" && receipt.text("outcome") == "acknowledged" && receipt.text("source_principal_ref") == want.SourcePrincipal && receipt.text("source_execution_ref") == want.SourceExecution && receipt.text("recipient_principal_ref") == want.RecipientPrincipal && receipt.text("recipient_execution_ref") == want.RecipientExecution && want.SourceExecution == child.text("execution_ref") && want.RecipientExecution == parent.text("execution_ref") && want.SourcePrincipal != want.RecipientPrincipal && admission != "" && consumption != "" && acknowledgement != "" && admission != consumption && admission != acknowledgement && consumption != acknowledgement, "A public mailbox receipt is not exact admission/consume/ACK evidence: %+v", receipt)
	for _, execution := range a.objects("executions") {
		v22Require(t, !execution.flag("recipient_mailbox_retired"), "A contains retired/orphan recipient mailbox: %+v", execution)
	}
}

func v22AssertProgrammingClosure(t *testing.T, c v22GoalProjection, councilIntegration v22CouncilIntegration) {
	t.Helper()
	reviews, integrations := c.objects("reviews"), c.objects("integration_receipts")
	v22Require(t, len(reviews) == 2 && len(integrations) == 1, "C review/integration cardinality=%d/%d want 2/1", len(reviews), len(integrations))
	first, second, integration := reviews[0], reviews[1], integrations[0]
	roles := map[string]bool{first.text("role"): true, second.text("role"): true}
	v22Require(t, roles["primary"] && roles["adversarial"] && first.text("verdict") == "approve" && second.text("verdict") == "approve" && first.text("change_ref") != "" && first.text("change_ref") == second.text("change_ref") && first.text("subject_digest") != "" && first.text("subject_digest") == second.text("subject_digest") && first.text("work_item_ref") == second.text("work_item_ref") && first.text("reviewer_execution_ref") != second.text("reviewer_execution_ref") && first.number("reviewer_execution_attempt") > 0 && second.number("reviewer_execution_attempt") > 0, "C reviews do not bind primary+adversarial to one exact subject: %+v %+v", first, second)
	v22Require(t, councilIntegration.DecisionRef != "" && councilIntegration.ActionRef != "" &&
		integration.text("change_ref") == councilIntegration.ChangeRef &&
		integration.text("change_ref") == first.text("change_ref") && integration.text("status") == "integrated" && integration.text("integration_ref") != "" && integration.text("target_before_oid") != "" && integration.text("target_after_oid") != "" && integration.text("tree_oid") != "" && integration.text("conflict_digest") == "", "C integration is not causal to accepted Council decision and approved change: council=%+v integration=%+v", councilIntegration, integration)
	passed := false
	for _, attestation := range c.objects("attestations") {
		if attestation.text("work_item_ref") != first.text("work_item_ref") || attestation.text("change_ref") != first.text("change_ref") || attestation.text("verdict") != "passed" || attestation.text("attestation_ref") == "" || attestation.text("execution_ref") == "" {
			continue
		}
		for _, test := range attestation.objects("tests") {
			passed = passed || test.text("required_test_ref") == "required-test:v22-c" && test.number("exit_code") == 0 && test.text("output_digest") != ""
		}
	}
	v22Require(t, passed, "C reviews/integration lack causal required-test PASS: %+v", c.objects("attestations"))
}

func v22ExecutionStates(g v22GoalProjection) string {
	var value strings.Builder
	for _, execution := range g.objects("executions") {
		fmt.Fprintf(&value, "%s=%s;", execution.text("execution_ref"), execution.text("state"))
	}
	return value.String()
}

func v22PlanA() map[string]any {
	parent := v22Item("parent", "Poll the execution-bound mailbox, deliver/consume/ACK its child, read only delivered artifact refs, then remain live for 20 seconds before producing the parent artifact.", "phase:a", nil, false, nil, "artifact")
	parent["dependencies"] = []string{"sibling"}
	return v22Plan("a",
		v22Item("sibling", "Produce a distinct independent artifact containing V22-SIBLING-NOT-DELIVERED.", "phase:a", nil, false, nil, "artifact"),
		parent,
		v22Item("child", "Produce a distinct child artifact containing V22-CHILD-DELIVERED and let post-artifact delivery admit it to parent.", "phase:a", nil, true, "parent", "artifact"),
	)
}
func v22PlanB() map[string]any {
	return v22Plan("b", v22Item("stop", "Remain working until public selective cancellation; do not affect another Goal.", "phase:b", nil, false, nil, "artifact"))
}
func v22PlanC() map[string]any {
	return v22Plan("c", v22Item("program", "Use isolated Git workspace: add a minimal go.mod, v22_marker.go with a Marker function, and v22_marker_test.go that verifies it; run the required Go test, obtain review, then integrate.", "phase:c", []string{"go.mod", "v22_marker.go", "v22_marker_test.go"}, false, nil, "evidence_bundle"))
}
func v22PlanD() map[string]any {
	return v22Plan("d", v22Item("recover", "Remain live across crash, resume exactly once after restart, and produce recovery artifact.", "phase:d", nil, false, nil, "artifact"))
}
func v22Plan(key string, items ...map[string]any) map[string]any {
	return map[string]any{"phases": []any{v22Phase(key)}, "work_items": items}
}
func v22Phase(key string) map[string]any {
	return map[string]any{"ref": "phase-instance:" + key, "key": "phase:" + key, "template_ref": "phase-template:program"}
}
func v22Item(key, objective, phase string, write []string, handoff bool, parent any, output string) map[string]any {
	item := map[string]any{"key": key, "objective": objective, "phase": phase, "role": "role:codex", "dependencies": []string{}, "write_set": []string{}, "handoff_required": handoff, "output_contract": output}
	if parent != nil {
		item["parent"] = parent
	}
	if key == "program" {
		item["write_set"] = write
		item["council_policy"] = "auto"
		item["required_tests"] = []any{map[string]any{"ref": "required-test:v22-c", "tool_ref": "tool:go", "arguments": []string{"test", "./..."}, "working_directory": "."}}
	}
	return item
}

func v22Config(root, port, codex, bwrap, toolchain, cgroupRoot string) string {
	q := func(v string) string { b, _ := json.Marshal(v); return string(b) }
	template := "[server]\nlisten = \"127.0.0.1:%s\"\nshutdown_timeout = \"10s\"\n[state.sqlite]\npath = %s\n[artifact.filesystem]\nroot = %s\n[credentials.local]\npath = %s\n[runtime]\nprovider = \"codex\"\nmax_output_bytes = 1048576\n[runtime.codex]\ncommand = %s\ntimeout = \"10m\"\nmax_concurrent_executions = 4\nwork_root = %s\n[workspace.local]\nroot = %s\n[repository.local]\nseed_path = %s\ntarget_ref = \"refs/heads/main\"\n[test_attestor]\nprovider = \"bubblewrap\"\ntimeout = \"5m\"\nmax_concurrent_runs = 4\n[test_attestor.bubblewrap]\ncommand = %s\n[test_attestor.go]\ntoolchain_root = %s\n[test_attestor.resources]\ncgroup_root = %s\n[identity]\nlocal_token_path = %s\n[project]\ndefault = \"project:v22-real\"\n[scheduler]\npoll_interval = \"50ms\"\nobservation_interval = \"100ms\"\nexecution_timeout = \"11m\"\nattest_test_claim_lease = \"6m\"\n[config]\neffective_path = %s\n"
	return fmt.Sprintf(template, port, q(filepath.Join(root, "state", "orquesta.sqlite")), q(filepath.Join(root, "artifacts")), q(filepath.Join(root, "secrets", "credentials.json")), q(codex), q(filepath.Join(root, "work")), q(filepath.Join(root, "workspaces")), q(filepath.Join(root, "seed")), q(bwrap), q(toolchain), q(cgroupRoot), q(filepath.Join(root, "secrets", "local-owner.token")), q(filepath.Join(root, "effective.json")))
}
func v22AttestorPrerequisites(t *testing.T) (string, string) {
	t.Helper()
	bwrap, err := exec.LookPath("bwrap")
	v22Require(t, err == nil, "V22 requires real bubblewrap test attestor: %v", err)
	toolchain := v22TrustedToolchainRoot(t)
	v22Require(t, filepath.IsAbs(bwrap) && filepath.IsAbs(toolchain), "V22 invalid bwrap/GOROOT: %q %q", bwrap, toolchain)
	return bwrap, toolchain
}
func v22TrustedToolchainRoot(t *testing.T) string {
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
	t.Fatal("V22_GATE_REAL_E2E_TOOLCHAIN_UNSAFE: root-owned immutable Go toolchain unavailable")
	return ""
}
func v22PrivateCodexHome(t *testing.T, root string) string {
	t.Helper()
	source := os.Getenv("CODEX_HOME")
	if source == "" {
		source = filepath.Join(v22Must(os.UserHomeDir()), ".codex")
	}
	target := filepath.Join(root, "codex-home")
	v22Require(t, os.Mkdir(target, 0o700) == nil, "create private V22 Codex home")
	v22CopyPrivateCodexAuth(t, filepath.Join(source, "auth.json"), filepath.Join(target, "auth.json"))
	directory := v22Must(os.Open(target))
	v22Require(t, directory.Sync() == nil && directory.Close() == nil, "sync private V22 Codex home")
	return target
}
func v22CopyPrivateCodexAuth(t *testing.T, sourcePath, targetPath string) {
	t.Helper()
	const maximum = int64(1 << 20)
	sourceFD, err := syscall.Open(sourcePath, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	v22Require(t, err == nil, "open private V22 Codex auth source")
	source := os.NewFile(uintptr(sourceFD), "v22-codex-auth-source")
	defer source.Close()
	before := v22Must(source.Stat())
	beforeStat, beforeOK := before.Sys().(*syscall.Stat_t)
	v22Require(t, beforeOK && before.Mode().IsRegular() && beforeStat.Uid == uint32(os.Geteuid()) &&
		beforeStat.Nlink == 1 && before.Mode().Perm()&0o077 == 0 && before.Mode().Perm()&0o400 != 0 &&
		before.Size() > 0 && before.Size() <= maximum, "unsafe private V22 Codex auth source")
	content := v22Must(io.ReadAll(io.LimitReader(source, maximum+1)))
	defer clear(content)
	var document map[string]json.RawMessage
	v22Require(t, json.Valid(content) && json.Unmarshal(content, &document) == nil && len(document) > 0, "private V22 Codex auth source invalid")
	after := v22Must(source.Stat())
	afterStat, afterOK := after.Sys().(*syscall.Stat_t)
	v22Require(t, int64(len(content)) == before.Size() && afterOK && os.SameFile(before, after) &&
		afterStat.Nlink == beforeStat.Nlink && afterStat.Ctim == beforeStat.Ctim && after.Size() == before.Size(),
		"private V22 Codex auth source changed")
	targetFD, err := syscall.Open(targetPath, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	v22Require(t, err == nil, "create private V22 Codex auth copy")
	target := os.NewFile(uintptr(targetFD), "v22-codex-auth-target")
	written, writeErr := target.Write(content)
	syncErr := target.Sync()
	info, statErr := target.Stat()
	v22Require(t, statErr == nil, "stat private V22 Codex auth copy")
	stat, ok := info.Sys().(*syscall.Stat_t)
	closeErr := target.Close()
	v22Require(t, writeErr == nil && written == len(content) && syncErr == nil && ok &&
		info.Mode().IsRegular() && info.Mode().Perm() == 0o600 && stat.Uid == uint32(os.Geteuid()) &&
		stat.Nlink == 1 && info.Size() == int64(len(content)) && closeErr == nil, "persist private V22 Codex auth copy")
}
func v22EnvironmentWithCodexHome(home string) []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "CODEX_HOME=") {
			environment = append(environment, entry)
		}
	}
	return append(environment, "CODEX_HOME="+home)
}
func v22ConfiguredCgroupRoot(t *testing.T) string {
	t.Helper()
	content := v22Must(os.ReadFile("/proc/self/cgroup"))
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		if strings.HasPrefix(line, "0::/") {
			self := strings.TrimPrefix(line, "0::")
			v22Require(t, filepath.Base(self) == "runner", "V22 delegated cgroup runner missing: %q", self)
			root := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(filepath.Dir(self), "/"))
			v22Require(t, os.Chmod(root, 0o700) == nil, "secure V22 delegated cgroup root")
			v22Require(t, os.WriteFile(filepath.Join(root, "cgroup.subtree_control"), []byte("+cpu +memory +pids"), 0o600) == nil, "enable V22 delegated cgroup controllers")
			return root
		}
	}
	t.Fatal("V22 unified cgroup unavailable")
	return ""
}
func v22Seed(t *testing.T, ctx context.Context, root, git string) {
	seed := filepath.Join(root, "seed")
	run := func(args ...string) {
		out, err := exec.CommandContext(ctx, git, args...).CombinedOutput()
		v22Require(t, err == nil, "git %v: %v: %s", args, err, out)
	}
	for _, args := range [][]string{{"init", "--initial-branch=main", seed}, {"-C", seed, "config", "user.email", "v22@example.invalid"}, {"-C", seed, "config", "user.name", "V22 E2E"}} {
		run(args...)
	}
	for _, path := range []string{seed, filepath.Join(seed, ".git")} {
		v22Require(t, os.Chmod(path, 0o700) == nil, "secure V22 seed Git path %s", path)
	}
	v22Require(t, os.WriteFile(filepath.Join(seed, "README.md"), []byte("V22 real workspace\n"), 0o600) == nil, "write V22 seed")
	run("-C", seed, "add", "README.md")
	run("-C", seed, "commit", "-m", "V22 seed")
}
func v22Port(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	v22Require(t, err == nil, "listen V22 port: %v", err)
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}
