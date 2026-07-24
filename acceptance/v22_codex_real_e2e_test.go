//go:build v22_real_e2e && linux

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
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/goal"
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
type v22ExecutionProjection struct {
	Ref            string `json:"execution_ref"`
	Work           string `json:"work_item_ref"`
	Replaces       string `json:"replaces_execution_ref"`
	State          string `json:"state"`
	Purpose        string `json:"purpose"`
	Failure        string `json:"failure_code"`
	Attempt        uint64 `json:"attempt_no"`
	Plan           uint64 `json:"plan_generation"`
	App            uint64 `json:"app_spec_generation"`
	MailboxRetired bool   `json:"recipient_mailbox_retired"`
}
type v22WorkProjection struct {
	Ref          string   `json:"work_item_ref"`
	State        string   `json:"state"`
	Parent       string   `json:"parent_work_item_ref"`
	Handoff      bool     `json:"handoff_required"`
	Execution    string   `json:"execution_ref"`
	Artifacts    []string `json:"artifact_refs"`
	Attestations []string `json:"attestation_refs"`
	Interrupt    string   `json:"interrupt_code"`
}
type v22MailboxReceiptProjection struct {
	MessageRef            string `json:"message_ref"`
	State                 string `json:"state"`
	SourcePrincipalRef    string `json:"source_principal_ref"`
	SourceExecutionRef    string `json:"source_execution_ref"`
	RecipientPrincipalRef string `json:"recipient_principal_ref"`
	RecipientExecutionRef string `json:"recipient_execution_ref"`
	AdmissionRef          string `json:"admission_ref"`
	ConsumptionRef        string `json:"consumption_ref"`
	AcknowledgementRef    string `json:"acknowledgement_ref"`
	Outcome               string `json:"outcome"`
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
	ExecutionCount int                      `json:"execution_count"`
	ArtifactCount  int                      `json:"artifact_count"`
	WorkItems      []v22WorkProjection      `json:"work_items"`
	Executions     []v22ExecutionProjection `json:"executions"`
	Attestations   []struct {
		Ref       string `json:"attestation_ref"`
		Verdict   string `json:"verdict"`
		Work      string `json:"work_item_ref"`
		Execution string `json:"execution_ref"`
		Change    string `json:"change_ref"`
		Tests     []struct {
			Ref    string `json:"required_test_ref"`
			Output string `json:"output_digest"`
			Exit   int    `json:"exit_code"`
		} `json:"tests"`
	} `json:"attestations"`
	Reviews []struct {
		Ref       string `json:"review_ref"`
		Work      string `json:"work_item_ref"`
		Change    string `json:"change_ref"`
		Subject   string `json:"subject_digest"`
		Role      string `json:"role"`
		Verdict   string `json:"verdict"`
		Execution string `json:"reviewer_execution_ref"`
		Attempt   uint64 `json:"reviewer_execution_attempt"`
	} `json:"reviews"`
	Controls []struct {
		Ref          string `json:"control_ref"`
		Operation    string `json:"operation"`
		Target       string `json:"target"`
		Status       string `json:"status"`
		Receipt      string `json:"receipt_ref"`
		Execution    string `json:"execution_ref"`
		GoalRevision uint64 `json:"goal_revision"`
		Plan         uint64 `json:"plan_generation"`
		App          uint64 `json:"app_spec_generation"`
		Attempt      uint64 `json:"execution_attempt"`
	} `json:"controls"`
	Integrations []struct {
		Ref      string `json:"integration_ref"`
		Change   string `json:"change_ref"`
		Status   string `json:"status"`
		Before   string `json:"target_before_oid"`
		After    string `json:"target_after_oid"`
		Tree     string `json:"tree_oid"`
		Conflict string `json:"conflict_digest"`
	} `json:"integration_receipts"`
	MailboxReceipts []v22MailboxReceiptProjection `json:"mailbox_receipts"`
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
	refs := map[string]string{"A": h.create(ctx, "A", v22PlanA()), "B": h.create(ctx, "B", v22PlanB()), "C": h.create(ctx, "C", v22PlanC()), "D": h.create(ctx, "D", v22PlanD())}
	running := h.waitRunning(ctx, refs["A"], refs["B"], refs["C"], refs["D"])
	bProcess := v22WaitProcess(t, h.root, running[refs["B"]].Ref)
	dProcess := v22WaitProcess(t, h.root, running[refs["D"]].Ref)
	progress := map[string]v22GoalProjection{}
	for _, id := range []string{"A", "C", "D"} {
		progress[id] = h.get(ctx, refs[id])
	}
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
	v22AssertStopped(t, b, running[refs["B"]], bProcess)
	for _, id := range []string{"A", "C"} {
		h.waitProgress(ctx, refs[id], progress[id])
	}
	if current := h.runningExecution(ctx, refs["D"]); current.Ref != running[refs["D"]].Ref {
		t.Fatalf("B stop disturbed D execution: got=%s want=%s", current.Ref, running[refs["D"]].Ref)
	}

	backup := h.backup(ctx)
	h.verifyRestoreCopy(ctx, backup, refs["D"], running[refs["D"]].Ref)
	if current := h.runningExecution(ctx, refs["D"]); current.Ref != running[refs["D"]].Ref {
		t.Fatalf("D execution changed before crash: got=%s want=%s", current.Ref, running[refs["D"]].Ref)
	}
	h.kill() // non-cooperative server death: SIGKILL, never Runtime.Shutdown.
	if alive, err := v22ProcessAlive(dProcess); err != nil || !alive {
		t.Fatalf("D exact process did not survive server SIGKILL: alive=%v err=%v", alive, err)
	}
	h.restart(ctx)
	h.waitAdopted(ctx, refs["D"], running[refs["D"]], dProcess)

	completed := map[string]v22GoalProjection{}
	for _, id := range []string{"A", "C", "D"} {
		completed[id] = h.waitTerminal(ctx, refs[id])
		if completed[id].Goal.State != "succeeded" {
			t.Fatalf("%s state=%s", id, completed[id].Goal.State)
		}
		v22NoContradiction(t, completed[id])
	}
	v22AssertAdoptedCompletion(t, completed["D"], running[refs["D"]].Ref)
	v22AssertFourGoals(t, completed, b)
}

func TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), v22RealDeadline)
	defer cancel()
	h := v22Start(t, ctx)
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
	t.Cleanup(h.close) // Registered before any owned process can be launched.
	build := exec.CommandContext(ctx, goTool, "build", "-mod=vendor", "-trimpath", "-buildvcs=false", "-o", h.binary, "./cmd/orquesta")
	build.Dir = evidenceRepositoryRoot(t)
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
func (h *v22Harness) verifyRestoreCopy(ctx context.Context, backup application.BackupRef, goalRaw, executionRaw string) {
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
	copyRepository, err := statesqlite.Open(ctx, statesqlite.Options{Path: restored, BusyTimeout: 5 * time.Second, MaxOpenConnections: 1})
	if err != nil {
		h.t.Fatalf("open isolated restored copy: %v", err)
	}
	defer copyRepository.Close()
	goalRef, err := goal.NewGoalRef(goalRaw)
	if err != nil {
		h.t.Fatal(err)
	}
	record, err := copyRepository.GetGoal(ctx, goalRef)
	if err != nil {
		h.t.Fatalf("read D from isolated restored copy: %v", err)
	}
	found := false
	for _, execution := range record.Executions {
		found = found || execution.Ref.String() == executionRaw
	}
	if !found || h.state == restored {
		h.t.Fatalf("isolated restore lost D execution or replaced original state: found=%v state=%q restored=%q",
			found, h.state, restored)
	}
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
func (h *v22Harness) waitRunning(ctx context.Context, refs ...string) map[string]v22ExecutionProjection {
	for {
		running := map[string]v22ExecutionProjection{}
		for _, ref := range refs {
			g := h.get(ctx, ref)
			switch g.Goal.State {
			case "succeeded", "failed", "cancelled", "stopped":
				h.t.Fatalf("Goal %s became terminal before exact running observation: %s", ref, g.Goal.State)
			}
			for _, execution := range g.Executions {
				if g.Goal.State == "running" && execution.State == "running" &&
					execution.Ref != "" && execution.Work != "" && execution.Attempt > 0 &&
					execution.Plan == g.Goal.Plan && execution.App == g.Goal.App {
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
	for _, execution := range d.Executions {
		if execution.Ref == exact && execution.Purpose == "work" && execution.State == "succeeded" {
			return
		}
	}
	t.Fatalf("restart did not complete the pre-crash D execution %s: %+v", exact, d.Executions)
}
func (h *v22Harness) runningExecution(ctx context.Context, ref string) v22ExecutionProjection {
	return h.waitRunning(ctx, ref)[ref]
}
func (h *v22Harness) waitAdopted(ctx context.Context, ref string, want v22ExecutionProjection, process v22ProcessRecord) {
	for {
		g := h.get(ctx, ref)
		for _, execution := range g.Executions {
			if execution.Ref == want.Ref && execution.State == "running" && execution.Replaces == "" {
				alive, err := v22ProcessAlive(process)
				if err != nil || !alive {
					h.t.Fatalf("restart did not adopt exact D process: alive=%v err=%v", alive, err)
				}
				return
			}
		}
		v22Wait(h.t, ctx, "D exact execution adoption")
	}
}
func (h *v22Harness) waitProgress(ctx context.Context, ref string, before v22GoalProjection) {
	for {
		after := h.get(ctx, ref)
		if after.Goal.State == "failed" || after.Goal.State == "cancelled" || after.Goal.State == "stopped" {
			h.t.Fatalf("independent Goal damaged by B stop: %+v", after.Goal)
		}
		if after.Goal.Revision > before.Goal.Revision || after.ArtifactCount > before.ArtifactCount ||
			v22ExecutionStates(after) != v22ExecutionStates(before) {
			return
		}
		v22Wait(h.t, ctx, "independent Goal progress")
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
		if err != nil {
			h.t.Fatal(err)
		}
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
	v22AssertMailboxClosure(t, done["A"])
	v22AssertProgrammingClosure(t, done["C"])
	d := done["D"]
	work := 0
	for _, execution := range d.Executions {
		if execution.Purpose == "work" {
			work++
			if execution.Replaces != "" {
				t.Fatalf("D duplicated/replaced work after restart: %+v", execution)
			}
		}
	}
	if work != 1 {
		t.Fatalf("D work execution count=%d want exactly one", work)
	}
}
func v22NoContradiction(t *testing.T, g v22GoalProjection) {
	t.Helper()
	if g.Goal.State != "succeeded" || g.ExecutionCount != len(g.Executions) || g.ArtifactCount == 0 {
		t.Fatalf("terminal projection contradiction: %+v", g)
	}
	seen := map[string]bool{}
	for _, e := range g.Executions {
		if e.Ref == "" || e.Work == "" || e.Attempt == 0 || e.Plan != g.Goal.Plan ||
			e.App != g.Goal.App || e.State != "succeeded" || e.Failure != "" || seen[e.Ref] {
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

func v22AssertStopped(t *testing.T, b v22GoalProjection, execution v22ExecutionProjection, process v22ProcessRecord) {
	t.Helper()
	v22AssertProcessGone(t, process)
	v22AssertStopEvidence(t, process)
	for _, current := range b.Executions {
		if current.Ref == execution.Ref && current.State == "running" {
			t.Fatalf("B exact execution remains running: %+v", current)
		}
	}
	for _, control := range b.Controls {
		if control.Operation == "stop" && control.Target == "goal" &&
			control.Status == "confirmed" && control.Receipt != "" &&
			control.Plan == b.Goal.Plan && control.App == b.Goal.App {
			return
		}
	}
	t.Fatalf("B lacks exact confirmed public control receipt: %+v", b.Controls)
}

func v22AssertMailboxClosure(t *testing.T, a v22GoalProjection) {
	t.Helper()
	var parent, child *v22WorkProjection
	for index := range a.WorkItems {
		item := &a.WorkItems[index]
		if item.Handoff {
			child = item
		} else {
			parent = item
		}
	}
	if parent == nil || child == nil || child.Parent != parent.Ref ||
		parent.Execution == "" || child.Execution == "" || parent.Execution == child.Execution ||
		len(parent.Artifacts) == 0 || len(child.Artifacts) == 0 {
		t.Fatalf("A lacks exact admitted/consumed/acknowledged handoff closure: parent=%+v child=%+v", parent, child)
	}
	if len(a.MailboxReceipts) != 1 {
		t.Fatalf("A mailbox receipts=%d want exactly one and zero orphans: %+v", len(a.MailboxReceipts), a.MailboxReceipts)
	}
	receipt := a.MailboxReceipts[0]
	if receipt.MessageRef == "" || receipt.State != "acknowledged" ||
		receipt.SourcePrincipalRef == "" || receipt.SourceExecutionRef != child.Execution ||
		receipt.RecipientPrincipalRef == "" || receipt.RecipientPrincipalRef == receipt.SourcePrincipalRef ||
		receipt.RecipientExecutionRef != parent.Execution || receipt.AdmissionRef == "" ||
		receipt.ConsumptionRef == "" || receipt.AcknowledgementRef == "" ||
		receipt.AdmissionRef == receipt.ConsumptionRef || receipt.AdmissionRef == receipt.AcknowledgementRef ||
		receipt.ConsumptionRef == receipt.AcknowledgementRef || receipt.Outcome != "acknowledged" {
		t.Fatalf("A public mailbox receipt is not exact admission/consume/ACK evidence: %+v", receipt)
	}
	for _, execution := range a.Executions {
		if execution.MailboxRetired {
			t.Fatalf("A contains retired/orphan recipient mailbox: %+v", execution)
		}
	}
}

func v22AssertProgrammingClosure(t *testing.T, c v22GoalProjection) {
	t.Helper()
	if len(c.Reviews) != 2 || len(c.Integrations) != 1 {
		t.Fatalf("C review/integration cardinality=%d/%d want 2/1", len(c.Reviews), len(c.Integrations))
	}
	first, second, integration := c.Reviews[0], c.Reviews[1], c.Integrations[0]
	roles := map[string]bool{first.Role: true, second.Role: true}
	if !roles["primary"] || !roles["adversarial"] || first.Verdict != "approve" ||
		second.Verdict != "approve" || first.Change == "" || first.Change != second.Change ||
		first.Subject == "" || first.Subject != second.Subject || first.Work != second.Work ||
		first.Execution == second.Execution || first.Attempt == 0 || second.Attempt == 0 {
		t.Fatalf("C reviews do not bind primary+adversarial to one exact subject: %+v %+v", first, second)
	}
	if integration.Change != first.Change || integration.Status != "integrated" ||
		integration.Ref == "" || integration.Before == "" || integration.After == "" ||
		integration.Tree == "" || integration.Conflict != "" {
		t.Fatalf("C integration is not causal to approved change: %+v", integration)
	}
	passed := false
	for _, attestation := range c.Attestations {
		if attestation.Work != first.Work || attestation.Change != first.Change ||
			attestation.Verdict != "passed" || attestation.Ref == "" || attestation.Execution == "" {
			continue
		}
		for _, test := range attestation.Tests {
			passed = passed || test.Ref == "required-test:v22-c" && test.Exit == 0 && test.Output != ""
		}
	}
	if !passed {
		t.Fatalf("C reviews/integration lack causal required-test PASS: %+v", c.Attestations)
	}
}

func v22ExecutionStates(g v22GoalProjection) string {
	var value strings.Builder
	for _, execution := range g.Executions {
		fmt.Fprintf(&value, "%s=%s;", execution.Ref, execution.State)
	}
	return value.String()
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
	template := "[server]\nlisten = \"127.0.0.1:%s\"\nshutdown_timeout = \"10s\"\n" +
		"[state.sqlite]\npath = %s\n[artifact.filesystem]\nroot = %s\n[credentials.local]\npath = %s\n" +
		"[runtime]\nprovider = \"codex\"\nmax_output_bytes = 1048576\n[runtime.codex]\ncommand = %s\ntimeout = \"10m\"\nmax_concurrent_executions = 4\nwork_root = %s\n" +
		"[workspace.local]\nroot = %s\n[repository.local]\nseed_path = %s\ntarget_ref = \"refs/heads/main\"\n" +
		"[test_attestor]\nprovider = \"bubblewrap\"\ntimeout = \"2m\"\n[test_attestor.bubblewrap]\ncommand = %s\n[test_attestor.go]\ntoolchain_root = %s\n[test_attestor.resources]\ncgroup_root = \"/sys/fs/cgroup\"\n" +
		"[identity]\nlocal_token_path = %s\n[project]\ndefault = \"project:v22-real\"\n" +
		"[scheduler]\npoll_interval = \"50ms\"\nobservation_interval = \"100ms\"\nexecution_timeout = \"11m\"\nattest_test_claim_lease = \"5m\"\n[config]\neffective_path = %s\n"
	return fmt.Sprintf(template, port, q(filepath.Join(root, "state", "orquesta.sqlite")),
		q(filepath.Join(root, "artifacts")), q(filepath.Join(root, "secrets", "credentials.json")),
		q(codex), q(filepath.Join(root, "work")), q(filepath.Join(root, "workspaces")),
		q(filepath.Join(root, "seed")), q(bwrap), q(toolchain),
		q(filepath.Join(root, "secrets", "local-owner.token")), q(filepath.Join(root, "effective.json")))
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
	run := func(args ...string) error {
		if out, err := exec.CommandContext(ctx, git, args...).CombinedOutput(); err != nil {
			return fmt.Errorf("git %v: %w: %s", args, err, out)
		}
		return nil
	}
	for _, args := range [][]string{{"init", "--initial-branch=main", seed}, {"-C", seed, "config", "user.email", "v22@example.invalid"}, {"-C", seed, "config", "user.name", "V22 E2E"}} {
		if err := run(args...); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("V22 real workspace\n"), 0o600); err != nil {
		return err
	}
	if err := run("-C", seed, "add", "README.md"); err != nil {
		return err
	}
	return run("-C", seed, "commit", "-m", "V22 seed")
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
