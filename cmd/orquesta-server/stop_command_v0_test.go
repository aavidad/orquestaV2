package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestStopServerCommandV0ForceConDaemonMuertoLimpiaBackendGoalConfigurado(t *testing.T) {
	store, tmuxLog, socketPath := prepareStopServerCommandDeadDaemonWithFakeTmuxV0(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := stopServerCommandV0([]string{"--force", "--reason", "daemon muerto en statefile"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "stop forced stale") ||
		!strings.Contains(stdout.String(), "process-ref-stop-stale-001") {
		t.Fatalf("stdout inesperado: %s", stdout.String())
	}
	rawLog, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if !strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("tmux no recibio kill-session: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("session fake no eliminada err=%v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket fake no eliminado err=%v", err)
	}
	reconciled, err := store.LoadServerStateV0(context.Background())
	if err != nil {
		t.Fatalf("load reconciled: %v", err)
	}
	if reconciled.Status != "stale" ||
		reconciled.StartupReady ||
		reconciled.StartupStatus != orquestaserver.ServerProcessStaleReasonCodeV0 ||
		reconciled.LastError != "server_process_not_alive" {
		t.Fatalf("state no reconciliado: %+v", reconciled)
	}
}

func TestStopServerCommandV0ForceNoDeclaraLimpioSiCleanupR8FallaV0(t *testing.T) {
	_, tmuxLog, socketPath := prepareStopServerCommandDeadDaemonWithFakeTmuxV0(t)
	if err := os.WriteFile(tmuxLog+".kill-fail", []byte("fail"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := stopServerCommandV0([]string{"--force", "--reason", "cleanup R8 debe propagarse"}, &stdout, &stderr)
	if exitCode != 1 || !strings.Contains(stderr.String(), "reason_code=forced_cleanup_failed") {
		t.Fatalf("exit=%d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "stop forced stale") {
		t.Fatalf("stop publico falso limpio: %s", stdout.String())
	}
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("sesion debe persistir tras cleanup fallido: %v", err)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket debe persistir tras cleanup fallido: %v", err)
	}
}

func TestStopServerCommandV0SinForceConDaemonMuertoNoLimpiaBackendGoal(t *testing.T) {
	store, tmuxLog, socketPath := prepareStopServerCommandDeadDaemonWithFakeTmuxV0(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := stopServerCommandV0(nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "daemon_identity_unavailable") {
		t.Fatalf("stderr inesperado: %s", stderr.String())
	}
	rawLog, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("tmux no debia recibir kill-session sin force: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("session fake debe seguir viva err=%v", err)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket fake debe seguir vivo err=%v", err)
	}
	state, err := store.LoadServerStateV0(context.Background())
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Status != "running" || !state.StartupReady {
		t.Fatalf("state no debe reconciliarse sin force: %+v", state)
	}
}

func TestStopServerCommandV0ForceConDaemonIdentityMismatchNoLimpiaBackendGoal(t *testing.T) {
	store, tmuxLog, socketPath := prepareStopServerCommandDeadDaemonWithFakeTmuxV0(t)
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
			SchemaVersion:  orquestaserver.StateSchemaVersionV0,
			Status:         "running",
			Addr:           strings.TrimPrefix(r.Host, "http://"),
			StartedAt:      "2026-07-02T08:00:01Z",
			ProcessRef:     "process-ref-other-daemon",
			DaemonEpochRef: "daemon-epoch-ref-other-daemon",
		}))
	}))
	defer server.Close()
	state, err := store.LoadServerStateV0(context.Background())
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	state.Addr = strings.TrimPrefix(server.URL, "http://")
	if err := store.SaveServerStateV0(context.Background(), state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := stopServerCommandV0([]string{"--force", "--reason", "daemon muerto en statefile"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "daemon_identity_mismatch") {
		t.Fatalf("stderr inesperado: %s", stderr.String())
	}
	rawLog, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("tmux no debia recibir kill-session con identidad viva distinta: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("session fake debe seguir viva err=%v", err)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket fake debe seguir vivo err=%v", err)
	}
}

func prepareStopServerCommandDeadDaemonWithFakeTmuxV0(
	t *testing.T,
) (*orquestaserver.FileStateStoreV0, string, string) {
	t.Helper()
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(fakeCodexAppServerTmuxCommandForTestV0()), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	projectDir := filepath.Join(root, "project")
	stateDir := filepath.Join(root, "state")
	runtimeDir := filepath.Join(root, "runtime")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	pathEnv := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv("ORQUESTA_TEST_BINARY", os.Args[0])
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexPathV0, pathEnv)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "2000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		t.Fatalf("socket path: %v", err)
	}
	codeHomePath, err := codexAppServerTmuxCodeHomePathV0(config)
	if err != nil {
		t.Fatalf("code home path: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:        pathEnv,
		SocketPath:     socketPath,
		SessionName:    codexAppServerTmuxSessionNameV0(config),
		CodeHomeDir:    codeHomePath,
		RuntimeWorkDir: runtimeDir,
		Timeout:        2 * time.Second,
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(tmuxLog + ".kill-fail")
		_ = backend.ShutdownForcedStopV0(context.Background())
	})

	deadPID := 99999999
	if processAliveV0(deadPID) {
		t.Skip("pid de prueba vivo en este sistema")
	}
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), orquestaserver.StateV0{
		Status:         "running",
		PID:            deadPID,
		Addr:           "127.0.0.1:1",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		StartedAt:      "2026-07-02T08:00:00Z",
		ProcessRef:     "process-ref-stop-stale-001",
		DaemonEpochRef: "daemon-epoch-ref-stop-stale-001",
		StartupReady:   true,
		StartupStatus:  "startup_ready",
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	return store, tmuxLog, socketPath
}
