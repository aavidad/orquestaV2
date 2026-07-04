package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeclaude "orquesta/modulos/orquesta-runtime-claude"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestServerSupervisorWithCodexGoalBackendV0ExponePuertosSoloConBackendV0(t *testing.T) {
	base := serverStackSupervisorV0{}
	plain := serverSupervisorWithCodexGoalBackendV0(base, serverCodexGoalBackendV0{})
	if _, ok := plain.(orquestaserver.IdleSelfImprovementGoalLauncherPortV0); ok {
		t.Fatalf("supervisor sin backend no debe exponer launcher goal")
	}
	if _, ok := plain.(orquestaserver.IdleSelfImprovementGoalObserverPortV0); ok {
		t.Fatalf("supervisor sin backend no debe exponer observer goal")
	}

	protocol := &fakeCodexAppServerProtocolV0{}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	wrapped := serverSupervisorWithCodexGoalBackendV0(base, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	})
	if _, ok := wrapped.(orquestaserver.IdleSelfImprovementGoalLauncherPortV0); !ok {
		t.Fatalf("supervisor con backend debe exponer launcher goal")
	}
	if _, ok := wrapped.(orquestaserver.IdleSelfImprovementGoalObserverPortV0); !ok {
		t.Fatalf("supervisor con backend debe exponer observer goal")
	}
}

func TestServerGoalObservationFingerprintFromBackendV0EsOptInV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-fingerprint-wiring",
			Status:   "active",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	if serverGoalObservationFingerprintFromBackendV0(serverCodexGoalBackendV0{Observer: backend}, false) != nil {
		t.Fatalf("fingerprint no debe exponerse sin opt-in")
	}
	if serverGoalObservationFingerprintFromBackendV0(serverCodexGoalBackendV0{Observer: backend}, true) == nil {
		t.Fatalf("backend Codex app-server debe exponer fingerprint con opt-in")
	}
}

func TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	codeHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(codeHome, 0o700); err != nil {
		t.Fatalf("mkdir code home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codeHome, "auth.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	marker := filepath.Join(root, "codex-invoked")
	fakeCodex := filepath.Join(root, "codex")
	script := "#!/usr/bin/env bash\nprintf invoked > " + shellQuoteCodexAppServerWiringTestV0(marker) + "\nexit 42\n"
	if err := os.WriteFile(fakeCodex, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCodeHomeV0, codeHome)
	t.Setenv(envCodexCommandV0, fakeCodex)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0, "37")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backend, err := serverCodexGoalBackendFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvV0: %v", err)
	}
	if backend.Starter == nil || backend.Observer == nil || backend.ShutdownHook == nil {
		t.Fatalf("backend no cableado: %+v", backend)
	}
	starter, ok := backend.Starter.(serverCodexAppServerGoalBackendV0)
	if !ok || starter.HighTokenUsageThreshold != 37 {
		t.Fatalf("starter sin umbral alto configurable: %#v", backend.Starter)
	}
	observer, ok := backend.Observer.(serverCodexAppServerGoalBackendV0)
	if !ok || observer.HighTokenUsageThreshold != 37 {
		t.Fatalf("observer sin umbral alto configurable: %#v", backend.Observer)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("construir backend no debe invocar app-server")
	}
}

func TestServerGoalBackendFromEnvV0ClaudeFileControlExponePuertosNeutralesV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	t.Setenv(envCodexGoalBackendV0, claudeGoalBackendFileControlV0)

	config := orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:       stateDir,
	}
	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(config, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	if backend.Starter != nil || backend.Observer != nil {
		t.Fatalf("backend Claude goal no debe usar puertos Codex: %+v", backend)
	}
	if backend.GoalLauncher == nil || backend.GoalObserver == nil {
		t.Fatalf("backend Claude goal sin puertos neutrales: %+v", backend)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	observer := serverGoalWorkObserverFromBackendV0(backend)
	spec := orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       "goal-ref-server-claude-file-control-001",
		RequestRef:    "request-ref-server-claude-file-control-001",
		RunRef:        "run-ref-server-claude-file-control-001",
		Objective:     "Probar backend Claude file-control desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-claude-file-control",
			Command: "go test ./cmd/orquesta-server",
		}},
	}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!strings.HasPrefix(receipt.ExternalGoalRef, "claude-goal-") {
		t.Fatalf("receipt Claude inesperado: %+v", receipt)
	}
	runtimeDir := filepath.Join(filepath.Dir(stateDir), "claude-goal")
	if _, err := os.Stat(filepath.Join(runtimeDir, "claude_goal_prompt_goal-ref-server-claude-file-control-001.txt")); err != nil {
		t.Fatalf("prompt Claude no materializado: %v", err)
	}
	observed, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsServerGoalStringV0(observed.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceResultPendingV0) {
		t.Fatalf("observed Claude inesperado: %+v", observed)
	}
}

func TestServerGoalBackendFromEnvV0ClaudeProcessLanzaYObservaResultadoV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	goalRef := "goal-ref-server-claude-process-001"
	t.Setenv(envCodexGoalBackendV0, claudeGoalBackendProcessV0)
	t.Setenv(envClaudeCommandV0, fakeClaudeGoalProcessCommandForTestV0(t, root, goalRef))

	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:       stateDir,
	}, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	observer := serverGoalWorkObserverFromBackendV0(backend)
	spec := orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-server-claude-process-001",
		RunRef:        "run-ref-server-claude-process-001",
		Objective:     "Probar backend Claude process desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-claude-process",
			Command: "fake claude process",
		}},
	}
	receipt, err := launcher.LaunchGoalWorkV0(context.Background(), spec)
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if !containsServerGoalStringV0(receipt.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceProcessLaunchedV0) {
		t.Fatalf("receipt sin evidencia proceso: %+v", receipt)
	}
	resultPath := filepath.Join(projectDir, "docs", orquestaruntimeclaude.ClaudeGoalResultFileNameV0)
	waitForServerGoalFileV0(t, resultPath)
	observed, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef:         goalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsServerGoalStringV0(observed.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceResultReadV0) {
		t.Fatalf("observed inesperado: %+v", observed)
	}
}

func TestServerGoalBackendFromEnvV0ClaudeProcessControlParaProcesoV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "control", "state")
	goalRef := "goal-ref-server-claude-process-control-001"
	startedPath := filepath.Join(root, "started.txt")
	t.Setenv(envCodexGoalBackendV0, claudeGoalBackendProcessV0)
	t.Setenv(envClaudeCommandV0, fakeClaudeGoalLongRunningCommandForTestV0(t, root, startedPath))

	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StateDir:       stateDir,
	}, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	control := serverGoalBackendControlFromBackendV0(backend)
	if control == nil {
		t.Fatalf("backend Claude process debe exponer control")
	}
	_, err = launcher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:       goalRef,
		RequestRef:    "request-ref-server-claude-process-control-001",
		RunRef:        "run-ref-server-claude-process-control-001",
		Objective:     "Probar control Claude process desde servidor.",
		DirectorKind:  orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path:    "docs",
			Purpose: "resultado durable",
		}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-server-claude-process-control",
			Command: "fake claude process control",
		}},
	})
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	waitForServerGoalFileV0(t, startedPath)

	result, err := control.ControlGoalBackendV0(context.Background(), orquestaappcodexstack.GoalBackendControlRequestV0{
		GoalRef:      goalRef,
		Action:       "stop",
		Reason:       "test_stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-server-claude-control"},
	})
	if err != nil {
		t.Fatalf("ControlGoalBackendV0: %v result=%+v", err, result)
	}
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!result.GoalStatusSet ||
		!result.BackendStopped ||
		!containsServerGoalStringV0(result.EvidenceRefs, orquestaruntimeclaude.ClaudeGoalEvidenceStopCompletedV0) {
		t.Fatalf("control result inesperado: %+v", result)
	}
}

func TestServerGoalBackendFromEnvV0RechazaBackendNoSoportadoV0(t *testing.T) {
	t.Setenv(envCodexGoalBackendV0, "claude_real_process")

	_, err := serverCodexGoalBackendFromEnvForWorkDirV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
		StateDir:       filepath.Join(t.TempDir(), "state"),
	}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_no_soportado") {
		t.Fatalf("backend no soportado no rechazado: %v", err)
	}
}

func TestServerGoalShutdownHooksFromBackendsV0DeduplicaMismoHookV0(t *testing.T) {
	hook := serverCodexAppServerTmuxBackendV0{
		SocketPath:  filepath.Join(t.TempDir(), "s.sock"),
		SessionName: "orquesta-goal-hook-1234567890",
	}
	hooks := serverGoalShutdownHooksFromBackendsV0(
		serverCodexGoalBackendV0{ShutdownHook: hook},
		serverCodexGoalBackendV0{ShutdownHook: hook},
	)
	if len(hooks) != 1 {
		t.Fatalf("hooks=%d", len(hooks))
	}
}

func TestServerGoalWorkPortsFromBackendV0PropaganActiveShutdownWorkV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-wiring-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-wiring-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-wiring-001", Status: "inProgress"},
	}
	hook := serverCodexAppServerTmuxBackendV0{
		SocketPath:  filepath.Join(t.TempDir(), "goal.sock"),
		SessionName: "orquesta-goal-active-1234567890",
		Timeout:     10 * time.Millisecond,
	}
	backend := serverCodexGoalBackendV0{
		Starter:      serverCodexAppServerGoalBackendV0{Protocol: protocol},
		Observer:     serverCodexAppServerGoalBackendV0{Protocol: protocol},
		ShutdownHook: hook,
	}
	launcher := serverGoalWorkLauncherFromBackendV0(backend)
	if _, ok := launcher.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0); !ok {
		t.Fatalf("launcher debe propagar active shutdown work")
	}
	observer := serverGoalWorkObserverFromBackendV0(backend)
	if _, ok := observer.(orquestaservershutdown.ActiveShutdownWorkCleanerPortV0); !ok {
		t.Fatalf("observer debe propagar cleanup active shutdown work")
	}
}

func TestServerEffectiveConfigV0ExponeGoalFirstYBackendV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "true")
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalTimeoutMSV0, "12000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, envServerIdleSelfImprovementGoalFirstV0); got != "true" {
		t.Fatalf("goal-first setting=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalBackendV0); got != codexGoalBackendAppServerTmuxV0 {
		t.Fatalf("goal backend=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalTimeoutMSV0); got != "12000" {
		t.Fatalf("goal timeout=%q", got)
	}
}

func shellQuoteCodexAppServerWiringTestV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func containsServerGoalStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func fakeClaudeGoalProcessCommandForTestV0(t *testing.T, root string, goalRef string) string {
	t.Helper()
	path := filepath.Join(root, "fake-claude-goal-process")
	script := "#!/bin/sh\nset -eu\nprompt=$(cat)\ncase \"$prompt\" in\n  *" + goalRef + "*) ;;\n  *) exit 7 ;;\nesac\nmkdir -p docs\ncat > docs/" + orquestaruntimeclaude.ClaudeGoalResultFileNameV0 + " <<'JSON'\n{\"schema_version\":\"orquesta_goal_result.v0\",\"status\":\"complete\",\"goal_ref\":\"" + goalRef + "\",\"summary\":\"server claude process complete\",\"checklist\":{\"expected_refs\":[\"server_claude_process\"],\"completed_refs\":[\"server_claude_process\"]},\"required_test_results\":[{\"test_ref\":\"required-test-ref-server-claude-process\",\"status\":\"passed\",\"evidence_refs\":[\"evidence-ref-server-claude-process-test\"]}],\"evidence_refs\":[\"evidence-ref-server-claude-process-result\"]}\nJSON\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake claude process: %v", err)
	}
	return path
}

func fakeClaudeGoalLongRunningCommandForTestV0(t *testing.T, root string, startedPath string) string {
	t.Helper()
	path := filepath.Join(root, "fake-claude-goal-long-running")
	script := "#!/bin/sh\nset -eu\ncat >/dev/null\nprintf started > " + shellQuoteCodexAppServerWiringTestV0(startedPath) + "\nsleep 30\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake claude long running: %v", err)
	}
	return path
}

func waitForServerGoalFileV0(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout esperando fichero %s", path)
}

type fakeCodexAppServerProbeV0 struct{}

func (fakeCodexAppServerProbeV0) ProbeV0(context.Context) error {
	return nil
}

func fakeCodexAppServerTmuxCommandForTestV0() string {
	return `#!/bin/sh
set -eu
if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
  printf '%s\n' "$*" >> "$ORQUESTA_TEST_TMUX_LOG"
fi
case "${1:-}" in
  has-session)
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

type fakeServerGoalActiveShutdownHookV0 struct {
	result orquestaservershutdown.ActiveShutdownWorkResultV0
}

func (hook fakeServerGoalActiveShutdownHookV0) ShutdownV0(context.Context) error {
	return nil
}

func (hook fakeServerGoalActiveShutdownHookV0) ReadActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	return hook.result, nil
}

func (hook fakeServerGoalActiveShutdownHookV0) CleanupActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
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

var _ = orquestagoal.GoalStatusRunningV0
var _ = orquestaruntimecodexgoal.CodexGoalResultMarkerV0
