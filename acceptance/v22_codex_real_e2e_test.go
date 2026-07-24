//go:build v22_real_e2e && linux

package acceptance_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"net"
	"net/http"
	statesqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Scenario mutations use official MCP; recovery and security probes are read-only.
const v22RealDeadline = 12 * time.Minute

type v22Output struct { Result struct { Data json.RawMessage `json:"data"`; Failure any `json:"failure"` } `json:"result"` }
type v22Object map[string]any
type v22GoalProjection = v22Object
type v22ExecutionProjection = v22Object

func (o v22Object) object(key string) v22Object { return v22Object(o[key].(map[string]any)) }
func (o v22Object) text(key string) string      { value, _ := o[key].(string); return value }
func (o v22Object) number(key string) uint64    { value, _ := o[key].(float64); return uint64(value) }
func (o v22Object) flag(key string) bool        { value, _ := o[key].(bool); return value }
func (o v22Object) strings(key string) []string {
	raw, _ := o[key].([]any); values := make([]string, len(raw))
	for index := range raw { values[index], _ = raw[index].(string) }
	return values
}
func (o v22Object) objects(key string) []v22Object {
	raw, _ := o[key].([]any); values := make([]v22Object, len(raw))
	for index := range raw { values[index] = v22Object(raw[index].(map[string]any)) }
	return values
}

type v22AdmissionIdentity struct { Message, Admission, SourcePrincipal, SourceExecution, RecipientPrincipal, RecipientExecution string }

func v22Must[T any](value T, err error) T { if err != nil { panic(err) }; return value }
func v22Decode[T any](path string) T {
	var value T; if err := json.Unmarshal(v22Must(os.ReadFile(path)), &value); err != nil { panic(err) }; return value
}

type v22Harness struct {
	t *testing.T; root, binary, config, state, endpoint string; server *exec.Cmd; log *os.File
	session *sdkmcp.ClientSession; mu sync.Mutex; request uint64
}

func TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), v22RealDeadline); defer cancel(); h := v22Start(t, ctx)
	refs := map[string]string{"A": h.create(ctx, "A", v22PlanA()), "B": h.create(ctx, "B", v22PlanB()), "C": h.create(ctx, "C", v22PlanC()), "D": h.create(ctx, "D", v22PlanD())}
	running := h.waitRunning(ctx, refs["A"], refs["B"], refs["C"], refs["D"])
	bProcess := v22WaitProcess(t, h.root, running[refs["B"]].text("execution_ref")); dProcess := v22WaitProcess(t, h.root, running[refs["D"]].text("execution_ref"))
	progress := map[string]v22GoalProjection{}
	for _, id := range []string{"A", "C", "D"} { progress[id] = h.get(ctx, refs[id]) }
	b := h.get(ctx, refs["B"]); bGoal := b.object("goal")
	h.call(ctx, "orquesta.goals.control", map[string]any{"operation": "stop", "target": "goal", "goal_ref": refs["B"], "expected_goal_revision": bGoal.number("revision"), "expected_plan_generation": bGoal.number("plan_generation"), "expected_app_spec_generation": bGoal.number("app_spec_generation"), "expected_spec_hash": bGoal.text("spec_hash"), "reason": "V22 public selective stop while A/C/D progress"})
	b = h.waitTerminal(ctx, refs["B"]); bState := b.object("goal").text("state")
	v22Require(t, bState == "stopped" || bState == "cancelled", "B=%s, want controlled stop", bState)
	v22AssertStopped(t, b, running[refs["B"]], bProcess)
	for _, id := range []string{"A", "C"} { h.waitProgress(ctx, refs[id], progress[id]) }
	exactD := running[refs["D"]].text("execution_ref"); current := h.runningExecution(ctx, refs["D"]).text("execution_ref")
	v22Require(t, current == exactD, "B stop disturbed D execution: got=%s want=%s", current, exactD)
	admission := h.assertMailboxArtifactIsolation(ctx, refs["A"]); backup := h.backup(ctx); h.verifyRestoreCopy(ctx, backup, refs["D"], exactD)
	current = h.runningExecution(ctx, refs["D"]).text("execution_ref")
	v22Require(t, current == exactD, "D execution changed before crash: got=%s want=%s", current, exactD)
	h.kill() // non-cooperative server death: SIGKILL, never Runtime.Shutdown.
	alive, aliveErr := v22ProcessAlive(dProcess)
	v22Require(t, aliveErr == nil && alive, "D exact process did not survive server SIGKILL: alive=%v err=%v", alive, aliveErr)
	h.restart(ctx); h.waitAdopted(ctx, refs["D"], running[refs["D"]], dProcess)
	recovered, recoveredOK := h.completedAdmission(ctx, refs["A"])
	v22Require(t, recoveredOK && recovered == admission, "restart changed completed service admission: got=%+v want=%+v found=%v", recovered, admission, recoveredOK)
	completed := map[string]v22GoalProjection{}
	for _, id := range []string{"A", "C", "D"} {
		completed[id] = h.waitTerminal(ctx, refs[id]); state := completed[id].object("goal").text("state")
		v22Require(t, state == "succeeded", "%s state=%s", id, state)
		v22NoContradiction(t, completed[id])
	}
	v22AssertAdoptedCompletion(t, completed["D"], exactD); v22AssertFourGoals(t, completed, b, admission)
}

func TestV22RealCodexNoTerminalContradictionAndNoOwnedProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), v22RealDeadline); defer cancel(); h := v22Start(t, ctx)
	v22NoContradiction(t, h.waitTerminal(ctx, h.create(ctx, "census", v22PlanC())))
	h.stop(); h.census()
}

func v22Start(t *testing.T, ctx context.Context) *v22Harness {
	t.Helper(); root := t.TempDir(); codex := v22Must(exec.LookPath("codex")); git := v22Must(exec.LookPath("git")); goTool := v22Must(exec.LookPath("go"))
	bwrap, toolchain := v22AttestorPrerequisites(t, ctx, goTool)
	v22Seed(t, ctx, root, git)
	h := &v22Harness{t: t, root: root, binary: filepath.Join(root, "orquesta"), config: filepath.Join(root, "orquesta.toml"), state: filepath.Join(root, "state", "orquesta.sqlite"), endpoint: "http://127.0.0.1:" + v22Port(t) + "/mcp"}
	v22Require(t, os.WriteFile(h.config, []byte(v22Config(root, strings.TrimSuffix(h.endpoint, "/mcp")[len("http://127.0.0.1:"):], codex, bwrap, toolchain)), 0o600) == nil, "write V22 config")
	t.Cleanup(h.close) // Registered before any owned process can be launched.
	build := exec.CommandContext(ctx, goTool, "build", "-mod=vendor", "-trimpath", "-buildvcs=false", "-o", h.binary, "./cmd/orquesta"); build.Dir = evidenceRepositoryRoot(t)
	out, err := build.CombinedOutput()
	v22Require(t, err == nil, "build external cmd/orquesta: %v\n%s", err, out)
	h.launch(ctx); return h
}
func (h *v22Harness) launch(ctx context.Context) {
	h.t.Helper()
	log := v22Must(os.OpenFile(filepath.Join(h.root, fmt.Sprintf("server-%d.log", time.Now().UnixNano())), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600))
	cmd := exec.Command(h.binary, "serve", "--config", h.config); cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}; cmd.Stdout, cmd.Stderr = log, log
	v22Require(h.t, cmd.Start() == nil, "start external cmd/orquesta")
	h.server, h.log = cmd, log
	for until := time.Now().Add(30 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		if h.connect(ctx) == nil { return }
	}
	out, _ := os.ReadFile(log.Name()); h.t.Fatalf("external MCP server unavailable: %s", out)
}
func (h *v22Harness) connect(ctx context.Context) error {
	if h.session != nil { _ = h.session.Close(); h.session = nil }
	token, err := os.ReadFile(filepath.Join(h.root, "secrets", "local-owner.token"))
	if err != nil { return err }
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "orquesta-v22-real-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{Endpoint: h.endpoint, HTTPClient: &http.Client{Transport: v22Bearer{strings.TrimSpace(string(token))}}}, nil)
	if err == nil { h.session = session }
	return err
}
func (h *v22Harness) restart(ctx context.Context) { h.releaseLog(); h.server = nil; h.launch(ctx) }
func (h *v22Harness) close()                      { h.stop(); h.census() }
func (h *v22Harness) releaseLog() {
	if h.log != nil { _ = h.log.Close(); h.log = nil }
}
func (h *v22Harness) disconnect() {
	if h.session != nil { _ = h.session.Close(); h.session = nil }
}
func (h *v22Harness) stop() { h.finish(os.Interrupt, false) }
func (h *v22Harness) kill() { h.finish(os.Kill, true) }
func (h *v22Harness) finish(signal os.Signal, requireFailure bool) {
	h.disconnect()
	if h.server == nil || h.server.Process == nil {
		v22Require(h.t, !requireFailure, "no server to SIGKILL"); h.releaseLog(); return
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
	v22Require(h.t, !requireFailure || err != nil, "SIGKILL unexpectedly clean"); h.server = nil; h.releaseLog()
}

func (h *v22Harness) backup(ctx context.Context) application.BackupRef {
	var ref application.BackupRef
	h.recovery(ctx, func(recovery *statesqlite.Recovery) {
		receipt := v22Must(recovery.CreateBackup(ctx)); v22Must(recovery.VerifyBackup(ctx, receipt.Ref)); ref = receipt.Ref
	})
	return ref
}
func (h *v22Harness) verifyRestoreCopy(ctx context.Context, backup application.BackupRef, goalRaw, executionRaw string) {
	var restored string
	h.recovery(ctx, func(recovery *statesqlite.Recovery) {
		target := v22Must(application.NewRecoveryTargetRef("recovery-target:v22-real")); v22Must(recovery.RestoreBackup(ctx, backup, target)); restored = v22Must(recovery.TargetPath(target))
	})
	copyRepository := v22Must(sql.Open("sqlite", restored)); defer copyRepository.Close()
	var found int
	err := copyRepository.QueryRowContext(ctx, `SELECT COUNT(*) FROM executions WHERE goal_ref=? AND ref=?`, goalRaw, executionRaw).Scan(&found)
	v22Require(h.t, err == nil && found == 1 && h.state != restored, "isolated restore D execution=%d state=%q restored=%q err=%v", found, h.state, restored, err)
}
func (h *v22Harness) recovery(ctx context.Context, run func(*statesqlite.Recovery)) {
	repository := v22Must(statesqlite.Open(ctx, statesqlite.Options{Path: h.state, BusyTimeout: 5 * time.Second, MaxOpenConnections: 8})); defer repository.Close()
	recovery := v22Must(statesqlite.NewRecovery(statesqlite.RecoveryOptions{Repository: repository, BackupRoot: filepath.Join(h.root, "backup"), RestoreRoot: filepath.Join(h.root, "restore")})); defer recovery.Close(); run(recovery)
}

func (h *v22Harness) create(ctx context.Context, id string, plan map[string]any) string {
	r := h.call(ctx, "orquesta.goals.create", map[string]any{"statement": "V22 " + id + " real Codex acceptance", "confirm": true, "plan": plan})
	var data v22Object; v22Data(h.t, r, &data); ref := data.object("goal").text("goal_ref")
	v22Require(h.t, ref != "", "create %s omitted goal_ref", id); return ref
}
func (h *v22Harness) call(ctx context.Context, tool string, payload map[string]any) v22Output {
	h.mu.Lock(); h.request++; ref := fmt.Sprintf("request:v22-real:%d", h.request); h.mu.Unlock()
	result, err := h.session.CallTool(ctx, &sdkmcp.CallToolParams{Name: tool, Arguments: map[string]any{"version": "1", "project_ref": "project:v22-real", "request_ref": ref, "payload": payload}})
	v22Require(h.t, err == nil, "MCP %s: %v", tool, err)
	raw := v22Must(json.Marshal(result.StructuredContent)); var out v22Output
	v22Require(h.t, json.Unmarshal(raw, &out) == nil, "MCP %s decode", tool)
	v22Require(h.t, !result.IsError && out.Result.Failure == nil, "MCP %s rejected public operation: %+v", tool, out.Result.Failure)
	return out
}
func (h *v22Harness) get(ctx context.Context, ref string) v22GoalProjection {
	r := h.call(ctx, "orquesta.goals.get", map[string]any{"goal_ref": ref}); var g v22GoalProjection; v22Data(h.t, r, &g); return g
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
					running[ref] = execution; break
				}
			}
		}
		if len(running) == len(refs) { return running }
		v22Wait(h.t, ctx, "parallel Goals")
	}
}
func v22AssertAdoptedCompletion(t *testing.T, d v22GoalProjection, exact string) {
	t.Helper()
	for _, execution := range d.objects("executions") {
		if execution.text("execution_ref") == exact && execution.text("purpose") == "work" && execution.text("state") == "succeeded" { return }
	}
	t.Fatalf("restart did not complete the pre-crash D execution %s: %+v", exact, d.objects("executions"))
}
func (h *v22Harness) runningExecution(ctx context.Context, ref string) v22ExecutionProjection { return h.waitRunning(ctx, ref)[ref] }
func (h *v22Harness) waitAdopted(ctx context.Context, ref string, want v22ExecutionProjection, process v22ProcessRecord) {
	for {
		g := h.get(ctx, ref)
		for _, execution := range g.objects("executions") {
			if execution.text("execution_ref") == want.text("execution_ref") && execution.text("state") == "running" && execution.text("replaces_execution_ref") == "" {
				alive, err := v22ProcessAlive(process); v22Require(h.t, err == nil && alive, "restart did not adopt exact D process: alive=%v err=%v", alive, err)
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
		v22Require(h.t, state != "failed" && state != "cancelled" && state != "stopped", "independent Goal damaged by B stop: %+v", after.object("goal"))
		if after.object("goal").number("revision") > before.object("goal").number("revision") || after.number("artifact_count") > before.number("artifact_count") || v22ExecutionStates(after) != v22ExecutionStates(before) { return }
		v22Wait(h.t, ctx, "independent Goal progress")
	}
}
func (h *v22Harness) waitTerminal(ctx context.Context, ref string) v22GoalProjection {
	for {
		g := h.get(ctx, ref)
		if v22Terminal(g.object("goal").text("state")) { return g }
		v22Wait(h.t, ctx, "terminal Goal")
	}
}
func v22Terminal(state string) bool {
	return state == "succeeded" || state == "failed" || state == "cancelled" || state == "stopped"
}
func (h *v22Harness) assertMailboxArtifactIsolation(ctx context.Context, goalRaw string) v22AdmissionIdentity {
	for {
		admission, ok := h.completedAdmission(ctx, goalRaw); items := map[string]v22Object{}
		for _, item := range h.get(ctx, goalRaw).objects("work_items") { items[item.text("execution_ref")] = item }
		source, recipient := items[admission.SourceExecution], items[admission.RecipientExecution]
		var sibling string
		for execution, item := range items {
			if execution != admission.SourceExecution && execution != admission.RecipientExecution && len(item.strings("artifact_refs")) == 1 { sibling = item.strings("artifact_refs")[0] }
		}
		if ok && len(source.strings("artifact_refs")) == 1 && recipient.text("state") == "running" && sibling != "" { return h.probeMailboxArtifactIsolation(ctx, goalRaw, admission, source.strings("artifact_refs")[0], sibling) }
		v22Wait(h.t, ctx, "A completed admission, delivered/sibling artifacts and live recipient")
	}
}
func (h *v22Harness) probeMailboxArtifactIsolation(ctx context.Context, goalRaw string, admission v22AdmissionIdentity, delivered, sibling string) v22AdmissionIdentity {
	repository := v22Must(statesqlite.Open(ctx, statesqlite.Options{Path: h.state, BusyTimeout: 5 * time.Second, MaxOpenConnections: 1})); defer repository.Close()
	recipientRef := v22Must(goal.NewExecutionRef(admission.RecipientExecution))
	authority, err := repository.ExecutionSessionAuthority(ctx, recipientRef, "execution_token")
	v22Require(h.t, err == nil && authority.ServicePrincipal.Ref.String() == admission.RecipientPrincipal, "recipient authority drift: %+v admission=%+v err=%v", authority, admission, err)
	record := v22Must(repository.GetGoal(ctx, mustGoalRef(goalRaw)))
	sourceExact := false
	for _, execution := range record.Executions {
		if execution.Ref.String() == admission.SourceExecution {
			source := v22Must(application.DeriveExecutionSessionAuthority(application.ExecutionSessionRequest(record.Goal, execution), "execution_token")); sourceExact = source.ServicePrincipal.Ref.String() == admission.SourcePrincipal
		}
	}
	v22Require(h.t, sourceExact, "source authority drift: admission=%+v", admission)
	authorize := func(ref, artifact string) (identity.AuthorizationReceipt, error) {
		request := v22Must(identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{RequestRef: "authorization:v22-real:" + ref, Principal: authority.ServicePrincipal, ProjectRef: record.Goal.Project(), Permission: identity.PermissionArtifactsRead, ResourceRef: artifact, RequestedAt: time.Now().UTC()}))
		return repository.Authorize(ctx, request)
	}
	receipt, deliveredErr := authorize("delivered", delivered); _, siblingErr := authorize("sibling", sibling)
	v22Require(h.t, delivered != sibling && deliveredErr == nil && receipt.Decision().Role() == identity.RoleExecutionService && errors.Is(siblingErr, application.ErrForbidden), "artifact delivery fence delivered=%s sibling=%s receipt=%+v errors=%v/%v", delivered, sibling, receipt, deliveredErr, siblingErr)
	return admission
}
func (h *v22Harness) completedAdmission(ctx context.Context, goalRef string) (v22AdmissionIdentity, bool) {
	database := v22Must(sql.Open("sqlite", h.state)); defer database.Close()
	var found v22AdmissionIdentity
	err := database.QueryRowContext(ctx, `SELECT envelope.ref,admission.ref,envelope.source_principal_ref,envelope.source_execution_ref,envelope.recipient_principal_ref,envelope.recipient_execution_ref FROM mailbox_envelopes envelope JOIN mailbox_admission_receipts admission ON admission.mailbox_message_ref=envelope.ref JOIN outbox action ON action.kind='admit_mailbox' AND action.goal_ref=envelope.goal_ref AND action.work_item_ref=envelope.child_work_item_ref AND action.execution_ref=envelope.source_execution_ref WHERE envelope.goal_ref=? AND action.completed_at IS NOT NULL`, goalRef).Scan(&found.Message, &found.Admission, &found.SourcePrincipal, &found.SourceExecution, &found.RecipientPrincipal, &found.RecipientExecution)
	if errors.Is(err, sql.ErrNoRows) { return found, false }
	v22Require(h.t, err == nil, "read completed admission: %v", err); return found, true
}
func mustGoalRef(raw string) goal.GoalRef { return v22Must(goal.NewGoalRef(raw)) }
func (h *v22Harness) census() {
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		if len(h.liveProcesses()) == 0 { return }
		time.Sleep(100 * time.Millisecond)
	}
	if records := h.liveProcesses(); len(records) > 0 {
		for _, record := range records { v22KillExact(record) }
		h.t.Errorf("owned exact processes survived cleanup and were killed: %+v", records)
	}
}
func (h *v22Harness) liveProcesses() []v22ProcessRecord {
	records, live := v22ProcessRecords(h.t, h.root), []v22ProcessRecord{}
	for _, record := range records {
		alive, err := v22ProcessGroupAlive(record)
		v22Require(h.t, err == nil, "process group liveness: %v", err)
		if alive { live = append(live, record) }
	}
	return live
}

type v22Bearer struct{ token string }

func (b v22Bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context()); c.Header = r.Header.Clone(); c.Header.Set("Authorization", "Bearer "+b.token)
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
	case <-ctx.Done(): t.Fatalf("wait %s: %v", what, ctx.Err())
	case <-time.After(300 * time.Millisecond):
	}
}

func v22AssertFourGoals(t *testing.T, done map[string]v22GoalProjection, b v22GoalProjection, admission v22AdmissionIdentity) {
	t.Helper()
	state := b.object("goal").text("state")
	v22Require(t, state == "stopped" || state == "cancelled", "B=%s", state)
	v22AssertMailboxClosure(t, done["A"], admission)
	v22AssertProgrammingClosure(t, done["C"])
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

func v22AssertStopped(t *testing.T, b v22GoalProjection, execution v22ExecutionProjection, process v22ProcessRecord) {
	t.Helper()
	v22AssertProcessGone(t, process)
	v22AssertStopEvidence(t, process)
	for _, current := range b.objects("executions") {
		v22Require(t, current.text("execution_ref") != execution.text("execution_ref") || current.text("state") != "running", "B exact execution remains running: %+v", current)
	}
	goalView := b.object("goal")
	for _, control := range b.objects("controls") {
		if control.text("operation") == "stop" && control.text("target") == "goal" && control.text("status") == "confirmed" && control.text("receipt_ref") != "" && control.number("plan_generation") == goalView.number("plan_generation") && control.number("app_spec_generation") == goalView.number("app_spec_generation") {
			return
		}
	}
	t.Fatalf("B lacks exact confirmed public control receipt: %+v", b.objects("controls"))
}

func v22AssertMailboxClosure(t *testing.T, a v22GoalProjection, want v22AdmissionIdentity) {
	t.Helper()
	var parent, child v22Object
	items := map[string]v22Object{}
	for _, item := range a.objects("work_items") {
		items[item.text("work_item_ref")] = item; if item.flag("handoff_required") { child = item }
	}
	if child != nil { parent = items[child.text("parent_work_item_ref")] }
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

func v22AssertProgrammingClosure(t *testing.T, c v22GoalProjection) {
	t.Helper()
	reviews, integrations := c.objects("reviews"), c.objects("integration_receipts")
	v22Require(t, len(reviews) == 2 && len(integrations) == 1, "C review/integration cardinality=%d/%d want 2/1", len(reviews), len(integrations))
	first, second, integration := reviews[0], reviews[1], integrations[0]
	roles := map[string]bool{first.text("role"): true, second.text("role"): true}
	v22Require(t, roles["primary"] && roles["adversarial"] && first.text("verdict") == "approve" && second.text("verdict") == "approve" && first.text("change_ref") != "" && first.text("change_ref") == second.text("change_ref") && first.text("subject_digest") != "" && first.text("subject_digest") == second.text("subject_digest") && first.text("work_item_ref") == second.text("work_item_ref") && first.text("reviewer_execution_ref") != second.text("reviewer_execution_ref") && first.number("reviewer_execution_attempt") > 0 && second.number("reviewer_execution_attempt") > 0, "C reviews do not bind primary+adversarial to one exact subject: %+v %+v", first, second)
	v22Require(t, integration.text("change_ref") == first.text("change_ref") && integration.text("status") == "integrated" && integration.text("integration_ref") != "" && integration.text("target_before_oid") != "" && integration.text("target_after_oid") != "" && integration.text("tree_oid") != "" && integration.text("conflict_digest") == "", "C integration is not causal to approved change: %+v", integration)
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
	parent := v22Item("parent", "Poll the execution-bound mailbox, deliver/consume/ACK its child, read only delivered artifact refs, then remain live for 20 seconds before producing the parent artifact.", "phase:a", []string{"docs/v22-parent.md"}, false, nil, "artifact")
	parent["dependencies"] = []string{"sibling"}
	return v22Plan("a",
		v22Item("sibling", "Produce a distinct independent artifact containing V22-SIBLING-NOT-DELIVERED.", "phase:a", []string{"docs/v22-sibling.md"}, false, nil, "artifact"),
		parent,
		v22Item("child", "Produce a distinct child artifact containing V22-CHILD-DELIVERED and let post-artifact delivery admit it to parent.", "phase:a", []string{"docs/v22-child.md"}, true, "parent", "artifact"),
	)
}
func v22PlanB() map[string]any { return v22Plan("b", v22Item("stop", "Remain working until public selective stop; do not affect another Goal.", "phase:b", []string{"docs/v22-stop.md"}, false, nil, "artifact")) }
func v22PlanC() map[string]any { return v22Plan("c", v22Item("program", "Use isolated Git workspace: add a minimal go.mod and v22_marker.txt, run the required Go test, obtain review, then integrate.", "phase:c", []string{"go.mod", "v22_marker.txt"}, false, nil, "evidence_bundle")) }
func v22PlanD() map[string]any { return v22Plan("d", v22Item("recover", "Remain live across crash, resume exactly once after restart, and produce recovery artifact.", "phase:d", []string{"docs/v22-recover.md"}, false, nil, "artifact")) }
func v22Plan(key string, items ...map[string]any) map[string]any { return map[string]any{"phases": []any{v22Phase(key)}, "work_items": items} }
func v22Phase(key string) map[string]any { return map[string]any{"ref": "phase-instance:" + key, "key": "phase:" + key, "template_ref": "phase-template:program"} }
func v22Item(key, objective, phase string, write []string, handoff bool, parent any, output string) map[string]any {
	item := map[string]any{"key": key, "objective": objective, "phase": phase, "role": "role:codex", "dependencies": []string{}, "write_set": write, "handoff_required": handoff, "output_contract": output, "council_policy": "required"}
	if parent != nil { item["parent"] = parent }
	if key == "program" { item["required_tests"] = []any{map[string]any{"ref": "required-test:v22-c", "tool_ref": "tool:go", "arguments": []string{"test", "./..."}, "working_directory": "."}} }
	return item
}

func v22Config(root, port, codex, bwrap, toolchain string) string {
	q := func(v string) string { b, _ := json.Marshal(v); return string(b) }
	template := "[server]\nlisten = \"127.0.0.1:%s\"\nshutdown_timeout = \"10s\"\n[state.sqlite]\npath = %s\n[artifact.filesystem]\nroot = %s\n[credentials.local]\npath = %s\n[runtime]\nprovider = \"codex\"\nmax_output_bytes = 1048576\n[runtime.codex]\ncommand = %s\ntimeout = \"10m\"\nmax_concurrent_executions = 4\nwork_root = %s\n[workspace.local]\nroot = %s\n[repository.local]\nseed_path = %s\ntarget_ref = \"refs/heads/main\"\n[test_attestor]\nprovider = \"bubblewrap\"\ntimeout = \"2m\"\n[test_attestor.bubblewrap]\ncommand = %s\n[test_attestor.go]\ntoolchain_root = %s\n[test_attestor.resources]\ncgroup_root = \"/sys/fs/cgroup\"\n[identity]\nlocal_token_path = %s\n[project]\ndefault = \"project:v22-real\"\n[scheduler]\npoll_interval = \"50ms\"\nobservation_interval = \"100ms\"\nexecution_timeout = \"11m\"\nattest_test_claim_lease = \"5m\"\n[config]\neffective_path = %s\n"
	return fmt.Sprintf(template, port, q(filepath.Join(root, "state", "orquesta.sqlite")), q(filepath.Join(root, "artifacts")), q(filepath.Join(root, "secrets", "credentials.json")), q(codex), q(filepath.Join(root, "work")), q(filepath.Join(root, "workspaces")), q(filepath.Join(root, "seed")), q(bwrap), q(toolchain), q(filepath.Join(root, "secrets", "local-owner.token")), q(filepath.Join(root, "effective.json")))
}
func v22AttestorPrerequisites(t *testing.T, ctx context.Context, goTool string) (string, string) {
	t.Helper(); bwrap, err := exec.LookPath("bwrap")
	v22Require(t, err == nil, "V22 requires real bubblewrap test attestor: %v", err)
	output, err := exec.CommandContext(ctx, goTool, "env", "GOROOT").Output()
	v22Require(t, err == nil, "V22 resolve Go toolchain root: %v", err)
	toolchain := strings.TrimSpace(string(output))
	v22Require(t, filepath.IsAbs(bwrap) && filepath.IsAbs(toolchain), "V22 invalid bwrap/GOROOT: %q %q", bwrap, toolchain)
	return bwrap, toolchain
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
	v22Require(t, os.WriteFile(filepath.Join(seed, "README.md"), []byte("V22 real workspace\n"), 0o600) == nil, "write V22 seed")
	run("-C", seed, "add", "README.md")
	run("-C", seed, "commit", "-m", "V22 seed")
}
func v22Port(t *testing.T) string {
	t.Helper(); l, err := net.Listen("tcp", "127.0.0.1:0")
	v22Require(t, err == nil, "listen V22 port: %v", err)
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}
