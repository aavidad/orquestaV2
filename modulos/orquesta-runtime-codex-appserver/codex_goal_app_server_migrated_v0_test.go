package orquestaruntimecodexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestServerCodexAppServerGoalBackendV0LanzaThreadGoalYTurnMigradoV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-001", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		Model:           "gpt-test",
		ReasoningEffort: "medium",
		Sandbox:         "workspace-write",
		ApprovalPolicy:  "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-001",
		Objective: "hacer una migracion acotada",
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.ExternalGoalRef != "thread-ref-goal-001" ||
		!containsStringMigratedTestV0(protocol.calls, "thread/start") ||
		!containsStringMigratedTestV0(protocol.calls, "thread/goal/set") ||
		!containsStringMigratedTestV0(protocol.calls, "turn/start") {
		t.Fatalf("receipt=%+v calls=%+v", receipt, protocol.calls)
	}
	if protocol.turnParams.Model != "gpt-test" ||
		protocol.turnParams.Effort != "medium" ||
		protocol.turnParams.ApprovalPolicy != "never" {
		t.Fatalf("turn params=%+v", protocol.turnParams)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaResultadoMarcadoMigradoV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-002",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-002",
			Status: "closed",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-goal-002",
				Items: []serverCodexAppServerReadItemV0{{
					Type:  "agentMessage",
					Phase: "final_answer",
					Text:  orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {"schema_version":"orquesta_goal_work_result.v0","status":"complete","summary":"ok","goal_ref":"goal-ref-002","external_goal_ref":"thread-ref-goal-002"}`,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-002",
		ExternalGoalRef: "thread-ref-goal-002",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusCompleteV0 || receipt.Summary != "ok" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0UmbralUsoAltoConfigurableV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID:        "thread-ref-goal-high-usage-001",
			Status:          "active",
			TokensUsed:      42,
			TimeUsedSeconds: 7,
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-high-usage-001",
			Status: "running",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:                protocol,
		HighTokenUsageThreshold: 10,
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-high-usage-001",
		ExternalGoalRef: "thread-ref-goal-high-usage-001",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!strings.Contains(receipt.Summary, "codex_app_server_goal_status_active_high_token_usage") ||
		!strings.Contains(receipt.Summary, "tokens_used=42") ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-high-token-usage") {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0UmbralUsoAltoDefaultConserva100kV0(t *testing.T) {
	backend := serverCodexAppServerGoalBackendV0{}

	if got := backend.codexAppServerGoalHighTokenUsageThresholdV0(); got != 100000 {
		t.Fatalf("threshold=%d", got)
	}
}

func TestCodexAppServerIssueCodeFromLogFileV0ClasificaResetStdioMigradoV0(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "orquesta-goal.log")
	raw := strings.Repeat("x", codexAppServerDiagnosticLogMaxBytesV0+64) +
		"\nvoid node::ResetStdio() at ../src/node.cc:751\n"
	if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	if got := codexAppServerIssueCodeFromLogFileV0(logPath); got != "codex_app_server_wrapper_stdio_failed" {
		t.Fatalf("code=%q", got)
	}
}

func TestPrepareCodexGoalWriteSetV0NoFallaConFicheroExistenteMigradoV0(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "modulos", "orquesta-server")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(dir, "config_v0.go")
	if err := os.WriteFile(existing, []byte("package orquestaserver\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	backend := serverCodexAppServerGoalBackendV0{CWD: root}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		WriteSet: []orquestagoal.GoalWriteScopeV0{
			{Path: "modulos/orquesta-server"},
			{Path: "modulos/orquesta-server/config_v0.go"},
		},
	}
	if err := backend.prepareCodexGoalWriteSetV0(packet); err != nil {
		t.Fatalf("prepare fallo con fichero existente en write-set: %v", err)
	}
	info, err := os.Stat(existing)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("el fichero existente del write-set se convirtio en directorio")
	}
}

func TestServerCodexAppServerGoalBackendV0LaunchPrepareFallidoPublicaDetailRelativoV0(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "modulos")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-prepare-detail-001"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-prepare-detail-001",
		Objective: "preparar write set",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path: "modulos/orquesta-goal",
		}},
	})

	if err == nil {
		t.Fatal("StartCodexGoalV0 debe fallar al preparar write_set bloqueado por fichero intermedio")
	}
	if len(protocol.calls) != 0 {
		t.Fatalf("no debe llamar al backend tras prepare fallido: calls=%v", protocol.calls)
	}
	if receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		!strings.HasPrefix(receipt.IssueCode, "codex_app_server_write_set_prepare_failed: ") ||
		!strings.Contains(receipt.IssueCode, "write_set_prepare_failed") ||
		!strings.Contains(receipt.IssueCode, "modulos/orquesta-goal") ||
		strings.Contains(receipt.IssueCode, root) {
		t.Fatalf("receipt=%+v root=%q", receipt, root)
	}
	normalized := orquestagoal.NormalizeGoalLaunchReceiptV0(orquestagoal.GoalLaunchReceiptV0{
		Status:  receipt.Status,
		GoalRef: receipt.GoalRef,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: receipt.IssueCode,
		}},
	})
	if len(normalized.Issues) != 1 ||
		normalized.Issues[0].Code != "codex_app_server_write_set_prepare_failed" ||
		!strings.Contains(normalized.Issues[0].Detail, "modulos/orquesta-goal") ||
		strings.Contains(normalized.Issues[0].Detail, root) {
		t.Fatalf("normalized=%+v root=%q", normalized.Issues, root)
	}
}

func TestServerCodexAppServerGoalBackendV0RuntimeWriteSetGuardBloqueaCambioFueraDeScope(t *testing.T) {
	root := t.TempDir()
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-runtime-write-set-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-runtime-write-set-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-runtime-write-set-001", Status: "complete"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
		Sandbox:  "workspace-write",
		Runtime:  &serverCodexAppServerGoalRuntimeV0{},
	}
	packet := codexAppServerRuntimeWriteSetGuardPacketForTestV0(
		"goal-ref-runtime-write-set-001",
		"docs",
	)

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil || receipt.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("StartCodexGoalV0 receipt=%+v err=%v", receipt, err)
	}
	writeCodexAppServerGoalResultForTestV0(t, root, "docs", packet.GoalRef, receipt.ExternalGoalRef)
	writeCodexAppServerTestFileV0(t, root, "docs/ok.md", "ok\n")
	writeCodexAppServerTestFileV0(t, root, "fuera.md", "fuera\n")
	protocol.observedGoal = &serverCodexAppServerThreadGoalV0{
		ThreadID: receipt.ExternalGoalRef,
		Status:   "complete",
	}

	observed, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusBlockedV0 ||
		observed.IssueCode != codexAppServerRuntimeWriteSetViolationV0 ||
		len(observed.DomainReceiptRefs) != 0 ||
		!containsStringMigratedTestV0(observed.EvidenceRefs, codexAppServerRuntimeWriteSetViolationEvidenceV0) {
		t.Fatalf("observed=%+v", observed)
	}
	if !containsStringPrefixMigratedTestV0(observed.EvidenceRefs, "evidence-ref-codex-app-server-runtime-write-set-outside-fuera-md") {
		t.Fatalf("evidence_refs=%+v", observed.EvidenceRefs)
	}
}

func TestServerCodexAppServerGoalBackendV0RuntimeWriteSetGuardPermiteCambioDentroDeScope(t *testing.T) {
	root := t.TempDir()
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-runtime-write-set-002"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-runtime-write-set-002", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-runtime-write-set-002", Status: "complete"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
		Sandbox:  "workspace-write",
		Runtime:  &serverCodexAppServerGoalRuntimeV0{},
	}
	packet := codexAppServerRuntimeWriteSetGuardPacketForTestV0(
		"goal-ref-runtime-write-set-002",
		"docs",
	)

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil || receipt.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("StartCodexGoalV0 receipt=%+v err=%v", receipt, err)
	}
	writeCodexAppServerGoalResultForTestV0(t, root, "docs", packet.GoalRef, receipt.ExternalGoalRef)
	writeCodexAppServerTestFileV0(t, root, "docs/ok.md", "ok\n")
	protocol.observedGoal = &serverCodexAppServerThreadGoalV0{
		ThreadID: receipt.ExternalGoalRef,
		Status:   "complete",
	}

	observed, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsStringMigratedTestV0(observed.DomainReceiptRefs, "domain-receipt-ref-runtime-write-set") ||
		!containsStringMigratedTestV0(observed.EvidenceRefs, codexAppServerRuntimeWriteSetGuardEvidenceV0) {
		t.Fatalf("observed=%+v", observed)
	}
}

func TestCodexAppServerWriteSetLooksLikeFileV0TrataExtensionesComoFicheroMigradoV0(t *testing.T) {
	cases := map[string]bool{
		"docs/informe.md":              true,
		"scripts/foo.sh":               true,
		"modulos/x/config_v0.go":       true,
		"docs/resultado.json":          true,
		"modulos/orquesta-server":      false,
		"generated-apps/demo":          false,
		"docs/paquete/":                false,
		"modulos/orquesta-estado-vivo": false,
	}
	for path, want := range cases {
		if got := codexAppServerWriteSetLooksLikeFileV0(path); got != want {
			t.Fatalf("codexAppServerWriteSetLooksLikeFileV0(%q) = %v, esperado %v", path, got, want)
		}
	}
}

func TestCodexAppServerTmuxBackendV0EnsureShutdownCleanupMigradoV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(fakeCodexAppServerTmuxCommandMigratedTestV0()), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	runtimeDir := filepath.Join(root, "runtime")
	socketPath := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "goal.sock")
	backend := serverCodexAppServerTmuxBackendV0{
		CommandPath:    filepath.Join(root, "codex"),
		PathEnv:        binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:     socketPath,
		SessionName:    "orquesta-goal-migrated-1234567890",
		RuntimeWorkDir: runtimeDir,
		ProjectWorkDir: filepath.Join(root, "project"),
		Timeout:        time.Second,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)

	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(socketPath), codexAppServerTmuxMarkerFileV0)); err != nil {
		t.Fatalf("owner marker ausente: %v", err)
	}
	active, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(active.ActiveWorks) != 1 ||
		!containsStringMigratedTestV0(active.EvidenceRefs, "evidence-ref-codex-app-server-tmux-session") {
		t.Fatalf("active=%+v", active)
	}
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("ShutdownV0: %v", err)
	}
	active, err = backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0 after shutdown: %v", err)
	}
	if len(active.ActiveWorks) != 0 {
		t.Fatalf("active after shutdown=%+v", active)
	}
	_, err = backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
	})
	if err != nil {
		t.Fatalf("CleanupActiveShutdownWorkV0: %v", err)
	}
}

type fakeCodexAppServerProbeV0 struct{}

func (fakeCodexAppServerProbeV0) ProbeV0(context.Context) error {
	return nil
}

type fakeCodexAppServerProtocolV0 struct {
	calls       []string
	startParams serverCodexAppServerThreadStartParamsV0
	setParams   serverCodexAppServerThreadGoalSetParamsV0
	turnParams  serverCodexAppServerTurnStartParamsV0
	getThreadID string

	thread           serverCodexAppServerThreadV0
	goal             serverCodexAppServerThreadGoalV0
	turn             serverCodexAppServerTurnV0
	setGoalErr       error
	getGoalErr       error
	observedGoal     *serverCodexAppServerThreadGoalV0
	readThread       serverCodexAppServerThreadReadV0
	readThreadErr    error
	readThreadID     string
	readIncludeTurns bool
}

func (fake *fakeCodexAppServerProtocolV0) StartThreadV0(
	_ context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	fake.calls = append(fake.calls, "thread/start")
	fake.startParams = params
	return fake.thread, nil
}

func (fake *fakeCodexAppServerProtocolV0) SetGoalV0(
	_ context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	fake.calls = append(fake.calls, "thread/goal/set")
	fake.setParams = params
	if fake.setGoalErr != nil {
		return serverCodexAppServerThreadGoalV0{}, fake.setGoalErr
	}
	return fake.goal, nil
}

func (fake *fakeCodexAppServerProtocolV0) StartTurnV0(
	_ context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	fake.calls = append(fake.calls, "turn/start")
	fake.turnParams = params
	return fake.turn, nil
}

func (fake *fakeCodexAppServerProtocolV0) GetGoalV0(
	_ context.Context,
	threadID string,
) (*serverCodexAppServerThreadGoalV0, error) {
	fake.calls = append(fake.calls, "thread/goal/get")
	fake.getThreadID = threadID
	if fake.getGoalErr != nil {
		return nil, fake.getGoalErr
	}
	return fake.observedGoal, nil
}

func (fake *fakeCodexAppServerProtocolV0) ReadThreadV0(
	_ context.Context,
	threadID string,
	includeTurns bool,
) (serverCodexAppServerThreadReadV0, error) {
	fake.calls = append(fake.calls, "thread/read")
	fake.readThreadID = threadID
	fake.readIncludeTurns = includeTurns
	if fake.readThreadErr != nil {
		return serverCodexAppServerThreadReadV0{}, fake.readThreadErr
	}
	return fake.readThread, nil
}

func fakeCodexAppServerTmuxCommandMigratedTestV0() string {
	return `#!/bin/sh
set -eu
if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
  printf '%s\n' "$*" >> "$ORQUESTA_TEST_TMUX_LOG"
fi
case "${1:-}" in
  ` + "has" + `-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ] && [ -e "${ORQUESTA_TEST_TMUX_LOG}.session" ]; then
      exit 0
    fi
    exit 1
    ;;
  kill-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
      rm -f "${ORQUESTA_TEST_TMUX_LOG}.session"
    fi
    exit 0
    ;;
  display-message)
    exit 0
    ;;
  new-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
      : > "${ORQUESTA_TEST_TMUX_LOG}.session"
    fi
    sock=""
    for arg in "$@"; do
      case "$arg" in
        unix://*)
          sock="${arg#unix://}"
          ;;
        *unix://*)
          sock="${arg#*unix://}"
          sock="${sock%%\'*}"
          sock="${sock%%\"*}"
          sock="${sock%% *}"
          ;;
      esac
    done
    if [ -z "$sock" ]; then
      echo "socket missing" >&2
      exit 2
    fi
    mkdir -p "$(dirname "$sock")"
    : > "$sock"
    exit 0
    ;;
esac
echo "tmux args inesperados: $*" >&2
exit 2
`
}

func containsStringMigratedTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsStringPrefixMigratedTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func codexAppServerRuntimeWriteSetGuardPacketForTestV0(
	goalRef string,
	writeSet string,
) orquestaruntimecodexgoal.CodexGoalStartPacketV0 {
	scopes := []orquestagoal.GoalWriteScopeV0{{Path: writeSet}}
	return orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   goalRef,
		Objective: "probar guard runtime de write-set",
		WriteSet:  scopes,
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			WriteSetEnforcement: orquestaruntimecodexgoal.CodexGoalWriteSetEnforcementV0,
			MinimumSandbox:      orquestaruntimecodexgoal.CodexGoalMinimumSandboxV0,
			AllowedWriteSet:     scopes,
		},
	}
}

func writeCodexAppServerGoalResultForTestV0(
	t *testing.T,
	root string,
	dir string,
	goalRef string,
	externalGoalRef string,
) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"schema_version":      orquestaruntimecodexgoal.CodexGoalResultSchemaV0,
		"status":              orquestagoal.GoalStatusCompleteV0,
		"goal_ref":            goalRef,
		"external_goal_ref":   externalGoalRef,
		"summary":             "resultado terminal",
		"artifact_paths":      []string{filepath.ToSlash(filepath.Join(dir, "ok.md"))},
		"domain_receipt_refs": []string{"domain-receipt-ref-runtime-write-set"},
		"evidence_refs":       []string{"evidence-ref-runtime-write-set-result"},
	})
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	writeCodexAppServerTestFileV0(
		t,
		root,
		filepath.ToSlash(filepath.Join(dir, orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(goalRef))),
		string(payload),
	)
}

func writeCodexAppServerTestFileV0(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

var _ = errors.New
