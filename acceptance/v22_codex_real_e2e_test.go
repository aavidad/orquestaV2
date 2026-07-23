//go:build v22_real_e2e

package acceptance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
)

// The harness owns only temporary files and processes. Every operation that
// changes or reads Orquesta business state uses the official MCP client.
const v22RealDeadline = 12 * time.Minute

type v22Output struct {
	Result struct {
		Data    json.RawMessage `json:"data"`
		Failure *struct {
			Code string `json:"code"`
		} `json:"failure"`
	} `json:"result"`
}
type v22GoalProjection struct {
	Goal struct {
		Ref      string `json:"goal_ref"`
		State    string `json:"state"`
		Revision uint64 `json:"revision"`
		Plan     uint64 `json:"plan_generation"`
		App      uint64 `json:"app_spec_generation"`
		Hash     string `json:"spec_hash"`
	} `json:"goal"`
	ExecutionCount int `json:"execution_count"`
	ArtifactCount  int `json:"artifact_count"`
	WorkItems      []struct {
		Ref          string   `json:"work_item_ref"`
		State        string   `json:"state"`
		Parent       string   `json:"parent_work_item_ref"`
		Handoff      bool     `json:"handoff_required"`
		Execution    string   `json:"execution_ref"`
		Artifacts    []string `json:"artifact_refs"`
		Attestations []string `json:"attestation_refs"`
		Interrupt    string   `json:"interrupt_code"`
	} `json:"work_items"`
	Executions []struct {
		Ref     string `json:"execution_ref"`
		Work    string `json:"work_item_ref"`
		Attempt uint64 `json:"attempt_no"`
		State   string `json:"state"`
		Failure string `json:"failure_code"`
	} `json:"executions"`
	Attestations []struct {
		Tests []struct {
			Ref  string `json:"required_test_ref"`
			Exit int    `json:"exit_code"`
		} `json:"tests"`
	} `json:"attestations"`
	Reviews []struct {
		Ref string `json:"review_ref"`
	} `json:"reviews"`
	Integrations []struct {
		Ref string `json:"integration_ref"`
	} `json:"integration_receipts"`
}
type v22Harness struct {
	t                                     *testing.T
	root, binary, config, state, endpoint string
	server                                *exec.Cmd
	log                                   *os.File
	session                               *sdkmcp.ClientSession
	mu                                    sync.Mutex
	request                               uint64
}

func TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), v22RealDeadline)
	defer cancel()
	h := v22Start(t, ctx)
	defer h.close()
	refs := map[string]string{"A": h.create(ctx, "A", v22PlanA()), "B": h.create(ctx, "B", v22PlanB()), "C": h.create(ctx, "C", v22PlanC()), "D": h.create(ctx, "D", v22PlanD())}
	h.waitRunning(ctx, refs["A"], refs["B"], refs["C"], refs["D"])
	b := h.get(ctx, refs["B"])
	h.call(ctx, "orquesta.goals.control", map[string]any{
		"operation": "stop", "target": "goal", "goal_ref": refs["B"],
		"expected_goal_revision": b.Goal.Revision, "expected_plan_generation": b.Goal.Plan,
		"expected_app_spec_generation": b.Goal.App, "expected_spec_hash": b.Goal.Hash,
		"reason": "V22 public selective stop while A/C/D progress",
	})
	b = h.waitTerminal(ctx, refs["B"])
	if b.Goal.State != "stopped" && b.Goal.State != "cancelled" {
		t.Fatalf("B=%s, want controlled stop", b.Goal.State)
	}

	h.waitExecution(ctx, refs["D"])
	backup := h.backup(ctx)
	h.kill() // non-cooperative server death: SIGKILL, never Runtime.Shutdown.
	h.restore(ctx, backup)
	h.restart(ctx)

	completed := map[string]v22GoalProjection{}
	for _, id := range []string{"A", "C", "D"} {
		completed[id] = h.waitTerminal(ctx, refs[id])
		if completed[id].Goal.State != "succeeded" {
			t.Fatalf("%s state=%s", id, completed[id].Goal.State)
		}
		v22NoContradiction(t, completed[id])
	}
	v22AssertFourGoals(t, completed, b)
}

func TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), v22RealDeadline)
	defer cancel()
	h := v22Start(t, ctx)
	defer h.close()
	v22NoContradiction(t, h.waitTerminal(ctx, h.create(ctx, "census", v22PlanC())))
	h.stop()
	h.census()
}

func v22Start(t *testing.T, ctx context.Context) *v22Harness {
	t.Helper()
	root := t.TempDir()
	codex, err := exec.LookPath("codex")
	if err != nil {
		t.Fatalf("V22 requires real Codex on PATH: %v", err)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("V22 requires Git: %v", err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("V22 requires Go: %v", err)
	}
	bwrap, toolchain := v22AttestorPrerequisites(t, ctx, goTool)
	if err := v22Seed(ctx, root, git); err != nil {
		t.Fatal(err)
	}
	h := &v22Harness{t: t, root: root, binary: filepath.Join(root, "orquesta"), config: filepath.Join(root, "orquesta.toml"), state: filepath.Join(root, "state", "orquesta.sqlite"), endpoint: "http://127.0.0.1:" + v22Port(t) + "/mcp"}
	if err := os.WriteFile(h.config, []byte(v22Config(root, strings.TrimSuffix(h.endpoint, "/mcp")[len("http://127.0.0.1:"):], codex, bwrap, toolchain)), 0o600); err != nil {
		t.Fatal(err)
	}
	build := exec.CommandContext(ctx, goTool, "build", "-mod=vendor", "-trimpath", "-buildvcs=false", "-o", h.binary, "./cmd/orquesta")
	build.Dir = v22Root(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build external cmd/orquesta: %v\n%s", err, out)
	}
	h.launch(ctx)
	return h
}
func (h *v22Harness) launch(ctx context.Context) {
	h.t.Helper()
	log, err := os.OpenFile(filepath.Join(h.root, fmt.Sprintf("server-%d.log", time.Now().UnixNano())), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		h.t.Fatal(err)
	}
	cmd := exec.Command(h.binary, "serve", "--config", h.config)
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		h.t.Fatalf("start external cmd/orquesta: %v", err)
	}
	h.server, h.log = cmd, log
	until := time.Now().Add(30 * time.Second)
	for time.Now().Before(until) {
		if h.connect(ctx) == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
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
func (h *v22Harness) restart(ctx context.Context) {
	h.releaseLog()
	h.server = nil
	h.launch(ctx)
}
func (h *v22Harness) close() { h.stop(); h.census() }
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
		if requireFailure {
			h.t.Fatal("no server to SIGKILL")
		}
		h.releaseLog()
		return
	}
	if err := h.server.Process.Signal(signal); err != nil {
		h.t.Fatalf("server signal %s: %v", signal, err)
	}
	done := make(chan error, 1)
	go func() { done <- h.server.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-time.After(15 * time.Second):
		_ = h.server.Process.Kill()
		err = <-done
	}
	if requireFailure && err == nil {
		h.t.Fatal("SIGKILL unexpectedly clean")
	}
	h.server = nil
	h.releaseLog()
}

func (h *v22Harness) backup(ctx context.Context) application.BackupRef {
	var ref application.BackupRef
	h.recovery(ctx, func(recovery *statesqlite.Recovery) {
		receipt, err := recovery.CreateBackup(ctx)
		if err != nil {
			h.t.Fatalf("V09 online backup: %v", err)
		}
		if _, err = recovery.VerifyBackup(ctx, receipt.Ref); err != nil {
			h.t.Fatalf("V09 verify backup: %v", err)
		}
		ref = receipt.Ref
	})
	return ref
}
func (h *v22Harness) restore(ctx context.Context, backup application.BackupRef) {
	var restored string
	h.recovery(ctx, func(recovery *statesqlite.Recovery) {
		target, err := application.NewRecoveryTargetRef("recovery-target:v22-real")
		if err != nil {
			h.t.Fatal(err)
		}
		if _, err = recovery.RestoreBackup(ctx, backup, target); err != nil {
			h.t.Fatalf("V09 restore backup: %v", err)
		}
		restored, err = recovery.TargetPath(target)
		if err != nil {
			h.t.Fatalf("V09 restored target path: %v", err)
		}
	})
	h.repointState(restored)
}
func (h *v22Harness) recovery(ctx context.Context, run func(*statesqlite.Recovery)) {
	repository, err := statesqlite.Open(ctx, statesqlite.Options{Path: h.state, BusyTimeout: 5 * time.Second, MaxOpenConnections: 8})
	if err != nil {
		h.t.Fatalf("open V09 recovery repository: %v", err)
	}
	defer repository.Close()
	recovery, err := statesqlite.NewRecovery(statesqlite.RecoveryOptions{Repository: repository, BackupRoot: filepath.Join(h.root, "backup"), RestoreRoot: filepath.Join(h.root, "restore")})
	if err != nil {
		h.t.Fatalf("open V09 recovery service: %v", err)
	}
	defer recovery.Close()
	run(recovery)
}

func (h *v22Harness) repointState(next string) {
	content, err := os.ReadFile(h.config)
	if err != nil {
		h.t.Fatal(err)
	}
	oldJSON, _ := json.Marshal(h.state)
	nextJSON, _ := json.Marshal(next)
	updated := strings.Replace(string(content), string(oldJSON), string(nextJSON), 1)
	if updated == string(content) {
		h.t.Fatal("V22 config lacks current state path")
	}
	if err := os.WriteFile(h.config, []byte(updated), 0o600); err != nil {
		h.t.Fatal(err)
	}
	h.state = next
}

func (h *v22Harness) create(ctx context.Context, id string, plan map[string]any) string {
	r := h.call(ctx, "orquesta.goals.create", map[string]any{"statement": "V22 " + id + " real Codex acceptance", "confirm": true, "plan": plan})
	var data struct {
		Goal struct {
			Ref string `json:"goal_ref"`
		} `json:"goal"`
	}
	v22Data(h.t, r, &data)
	if data.Goal.Ref == "" {
		h.t.Fatalf("create %s omitted goal_ref", id)
	}
	return data.Goal.Ref
}
func (h *v22Harness) call(ctx context.Context, tool string, payload map[string]any) v22Output {
	h.mu.Lock()
	h.request++
	ref := fmt.Sprintf("request:v22-real:%d", h.request)
	h.mu.Unlock()
	result, err := h.session.CallTool(ctx, &sdkmcp.CallToolParams{Name: tool, Arguments: map[string]any{"version": "1", "project_ref": "project:v22-real", "request_ref": ref, "payload": payload}})
	if err != nil {
		h.t.Fatalf("MCP %s: %v", tool, err)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		h.t.Fatal(err)
	}
	var out v22Output
	if err := json.Unmarshal(raw, &out); err != nil {
		h.t.Fatalf("MCP %s decode: %v", tool, err)
	}
	if result.IsError || out.Result.Failure != nil {
		code := ""
		if out.Result.Failure != nil {
			code = out.Result.Failure.Code
		}
		h.t.Fatalf("MCP %s rejected public operation: %s", tool, code)
	}
	return out
}
func (h *v22Harness) get(ctx context.Context, ref string) v22GoalProjection {
	r := h.call(ctx, "orquesta.goals.get", map[string]any{"goal_ref": ref})
	var g v22GoalProjection
	v22Data(h.t, r, &g)
	return g
}
func (h *v22Harness) waitExecution(ctx context.Context, ref string) {
	for {
		if h.get(ctx, ref).ExecutionCount > 0 {
			return
		}
		v22Wait(h.t, ctx, "execution")
	}
}
func (h *v22Harness) waitRunning(ctx context.Context, refs ...string) {
	for {
		n := 0
		for _, ref := range refs {
			g := h.get(ctx, ref)
			if g.Goal.State == "running" && g.ExecutionCount > 0 {
				n++
			}
		}
		if n == len(refs) {
			return
		}
		v22Wait(h.t, ctx, "parallel Goals")
	}
}
func (h *v22Harness) waitTerminal(ctx context.Context, ref string) v22GoalProjection {
	for {
		g := h.get(ctx, ref)
		switch g.Goal.State {
		case "succeeded", "failed", "cancelled", "stopped":
			return g
		}
		v22Wait(h.t, ctx, "terminal Goal")
	}
}
func (h *v22Harness) census() {
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		if len(v22PIDs(h.root)) == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	if p := v22PIDs(h.root); len(p) > 0 {
		h.t.Fatalf("owned process survives cleanup: %v", p)
	}
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
	if len(r.Result.Data) == 0 {
		t.Fatal("MCP response omitted data")
	}
	if err := json.Unmarshal(r.Result.Data, target); err != nil {
		t.Fatalf("decode MCP data: %v", err)
	}
}
func v22Wait(t *testing.T, ctx context.Context, what string) {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatalf("wait %s: %v", what, ctx.Err())
	case <-time.After(300 * time.Millisecond):
	}
}

func v22AssertFourGoals(t *testing.T, done map[string]v22GoalProjection, b v22GoalProjection) {
	t.Helper()
	if b.Goal.State != "stopped" && b.Goal.State != "cancelled" {
		t.Fatalf("B=%s", b.Goal.State)
	}
	for _, e := range b.Executions {
		if e.State == "running" {
			t.Fatalf("B running execution: %+v", e)
		}
	}
	c := done["C"]
	if len(c.Attestations) == 0 || len(c.Reviews) == 0 || len(c.Integrations) == 0 {
		t.Fatalf("C lacks public test/review/integration receipts: %+v", c)
	}
	for _, a := range c.Attestations {
		for _, test := range a.Tests {
			if test.Exit != 0 {
				t.Fatalf("C test %s exit=%d", test.Ref, test.Exit)
			}
		}
	}
	handoff := false
	for _, w := range done["A"].WorkItems {
		if w.Handoff {
			handoff = true
			if w.Parent == "" || w.Execution == "" || len(w.Artifacts) == 0 {
				t.Fatalf("A mailbox handoff lacks public causal evidence: %+v", w)
			}
		}
	}
	if !handoff {
		t.Fatal("A missing parent/child handoff")
	}
}
func v22NoContradiction(t *testing.T, g v22GoalProjection) {
	t.Helper()
	if g.Goal.State != "succeeded" || g.ExecutionCount != len(g.Executions) || g.ArtifactCount == 0 {
		t.Fatalf("terminal projection contradiction: %+v", g)
	}
	seen := map[string]bool{}
	for _, e := range g.Executions {
		if e.Ref == "" || e.Work == "" || e.Attempt == 0 || e.State != "succeeded" || e.Failure != "" || seen[e.Ref] {
			t.Fatalf("execution contradiction: %+v", e)
		}
		seen[e.Ref] = true
	}
	for _, w := range g.WorkItems {
		if w.State != "succeeded" || w.Execution == "" || len(w.Artifacts) == 0 || len(w.Attestations) == 0 || w.Interrupt != "" {
			t.Fatalf("work item contradiction: %+v", w)
		}
	}
}

func v22PlanA() map[string]any {
	return v22Plan("a", v22Item("parent", "Stay running; poll execution-bound mailbox.list/get, mark delivered, consume and acknowledge; read only delivered artifact refs with artifacts.read; never call goals.get; then produce parent artifact.", "phase:a", []string{"docs/v22-parent.md"}, false, nil, "artifact"), v22Item("child", "Produce the child artifact and let the declared post-artifact delivery continuation admit it to the parent mailbox.", "phase:a", []string{"docs/v22-child.md"}, true, "parent", "artifact"))
}
func v22PlanB() map[string]any {
	return v22Plan("b", v22Item("stop", "Remain working until public selective stop; do not affect another Goal.", "phase:b", []string{"docs/v22-stop.md"}, false, nil, "artifact"))
}
func v22PlanC() map[string]any {
	return v22Plan("c", v22Item("program", "Use isolated Git workspace: add a minimal go.mod and v22_marker.txt, run the required Go test, obtain review, then integrate.", "phase:c", []string{"go.mod", "v22_marker.txt"}, false, nil, "evidence_bundle"))
}
func v22PlanD() map[string]any {
	return v22Plan("d", v22Item("recover", "Remain live across crash, resume exactly once after restart, and produce recovery artifact.", "phase:d", []string{"docs/v22-recover.md"}, false, nil, "artifact"))
}
func v22Plan(key string, items ...map[string]any) map[string]any {
	return map[string]any{"phases": []any{v22Phase(key)}, "work_items": items}
}
func v22Phase(key string) map[string]any {
	return map[string]any{"ref": "phase-instance:" + key, "key": "phase:" + key, "template_ref": "phase-template:program"}
}
func v22Item(key, objective, phase string, write []string, handoff bool, parent any, output string) map[string]any {
	item := map[string]any{"key": key, "objective": objective, "phase": phase, "role": "role:codex", "dependencies": []string{}, "write_set": write, "handoff_required": handoff, "output_contract": output, "council_policy": "required"}
	if parent != nil {
		item["parent"] = parent
	}
	if key == "program" {
		item["required_tests"] = []any{map[string]any{"ref": "required-test:v22-c", "tool_ref": "tool:go", "arguments": []string{"test", "./..."}, "working_directory": "."}}
	}
	return item
}

func v22Config(root, port, codex, bwrap, toolchain string) string {
	q := func(v string) string { b, _ := json.Marshal(v); return string(b) }
	return fmt.Sprintf(`[server]
listen = "127.0.0.1:%s"
shutdown_timeout = "10s"
[state.sqlite]
path = %s
[artifact.filesystem]
root = %s
[credentials.local]
path = %s
[runtime]
provider = "codex"
max_output_bytes = 1048576
[runtime.codex]
command = %s
timeout = "10m"
max_concurrent_executions = 4
work_root = %s
[workspace.local]
root = %s
[repository.local]
seed_path = %s
target_ref = "refs/heads/main"
[test_attestor]
provider = "bubblewrap"
timeout = "2m"
[test_attestor.bubblewrap]
command = %s
[test_attestor.go]
toolchain_root = %s
[test_attestor.resources]
cgroup_root = "/sys/fs/cgroup"
[identity]
local_token_path = %s
[project]
default = "project:v22-real"
[scheduler]
poll_interval = "50ms"
observation_interval = "100ms"
execution_timeout = "11m"
attest_test_claim_lease = "5m"
[config]
effective_path = %s
`, port, q(filepath.Join(root, "state", "orquesta.sqlite")), q(filepath.Join(root, "artifacts")), q(filepath.Join(root, "secrets", "credentials.json")), q(codex), q(filepath.Join(root, "work")), q(filepath.Join(root, "workspaces")), q(filepath.Join(root, "seed")), q(bwrap), q(toolchain), q(filepath.Join(root, "secrets", "local-owner.token")), q(filepath.Join(root, "effective.json")))
}
func v22AttestorPrerequisites(t *testing.T, ctx context.Context, goTool string) (string, string) {
	t.Helper()
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Fatalf("V22 requires real bubblewrap test attestor: %v", err)
	}
	output, err := exec.CommandContext(ctx, goTool, "env", "GOROOT").Output()
	if err != nil {
		t.Fatalf("V22 resolve Go toolchain root: %v", err)
	}
	toolchain := strings.TrimSpace(string(output))
	if !filepath.IsAbs(bwrap) || !filepath.IsAbs(toolchain) {
		t.Fatalf("V22 invalid bwrap/GOROOT: %q %q", bwrap, toolchain)
	}
	return bwrap, toolchain
}
func v22Seed(ctx context.Context, root, git string) error {
	seed := filepath.Join(root, "seed")
	for _, args := range [][]string{{"init", "--initial-branch=main", seed}, {"-C", seed, "config", "user.email", "v22@example.invalid"}, {"-C", seed, "config", "user.name", "V22 E2E"}} {
		if out, err := exec.CommandContext(ctx, git, args...).CombinedOutput(); err != nil {
			return fmt.Errorf("git %v: %w: %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("V22 real workspace\n"), 0o600); err != nil {
		return err
	}
	if out, err := exec.CommandContext(ctx, git, "-C", seed, "add", "README.md").CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	if out, err := exec.CommandContext(ctx, git, "-C", seed, "commit", "-m", "V22 seed").CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}
func v22Root(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func v22Port(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}
func v22PIDs(root string) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var pids []int
	for _, entry := range entries {
		var pid int
		if _, err := fmt.Sscanf(entry.Name(), "%d", &pid); err != nil || pid == os.Getpid() {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err == nil && strings.Contains(string(b), root) {
			pids = append(pids, pid)
		}
	}
	sort.Ints(pids)
	return pids
}
