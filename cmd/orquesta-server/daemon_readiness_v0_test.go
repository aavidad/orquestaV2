package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerReadinessOKV0UsaReadinessNoHealthzV0(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		case orquestaserver.ServerReadinessEndpointV0:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	if serverReadinessOKV0(strings.TrimPrefix(server.URL, "http://")) {
		t.Fatalf("readiness false no debe pasar aunque healthz este vivo")
	}
}

func TestWaitForStateHealthyV0AceptaReadinessEstableV0(t *testing.T) {
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	config := daemonReadinessConfigForTestV0(t)
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), os.Getpid())
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	got, err := waitForStateHealthyV0(config, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("waitForStateHealthyV0: %v", err)
	}
	if got.PID != state.PID || got.ProcessRef != state.ProcessRef {
		t.Fatalf("state estable inesperado: %+v", got)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("readiness estable debe requerir al menos dos consultas, calls=%d", calls)
	}
}

func TestWaitForStateHealthyV0RechazaReadinessInestableTrasReadyV0(t *testing.T) {
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	config := daemonReadinessConfigForTestV0(t)
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), os.Getpid())
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	_, err := waitForStateHealthyV0(config, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "readiness_timeout") {
		t.Fatalf("readiness inestable debe expirar, err=%v", err)
	}
}

func TestWaitForStateHealthyV0RechazaCambioDeIdentidadTrasReadyV0(t *testing.T) {
	config := daemonReadinessConfigForTestV0(t)
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		call := atomic.AddInt32(&calls, 1)
		next := daemonReadinessStateForTestV0(r.Host, os.Getpid())
		next.Addr = r.Host
		next.ProcessRef = "process-ref-startup-test-changed-a"
		if call%2 == 0 {
			next.ProcessRef = "process-ref-startup-test-changed-b"
		}
		saveDaemonReadinessStateForTestV0(t, config, next)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), os.Getpid())
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	_, err := waitForStateHealthyV0(config, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "readiness_timeout") {
		t.Fatalf("cambio de identidad debe expirar, err=%v", err)
	}
}

func TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0(t *testing.T) {
	config := daemonReadinessConfigForTestV0(t)
	deadPID := daemonReadinessDeadPIDForTestV0(t)
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			next := daemonReadinessStateForTestV0(r.Host, deadPID)
			next.LastHeartbeatAt = "2026-07-01T00:00:00Z"
			saveDaemonReadinessStateForTestV0(t, config, next)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), os.Getpid())
	state.LastHeartbeatAt = "2026-07-01T00:00:00Z"
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	got, err := waitForStateHealthyV0(config, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "server_exited_after_readiness") {
		t.Fatalf("servidor muerto tras readiness debe reportar server_exited_after_readiness, err=%v state=%+v", err, got)
	}

	store, storeErr := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if storeErr != nil {
		t.Fatalf("NewFileStateStoreV0: %v", storeErr)
	}
	persisted, loadErr := store.LoadServerStateV0(context.Background())
	if loadErr != nil {
		t.Fatalf("LoadServerStateV0: %v", loadErr)
	}
	if persisted.Status != "stale" ||
		persisted.StartupReady ||
		persisted.StartupStatus != orquestaserver.ServerProcessStaleReasonCodeV0 ||
		persisted.LastHeartbeatAt != "2026-07-01T00:00:00Z" ||
		persisted.LastError != "server_process_not_alive" {
		t.Fatalf("statefile no reconciliado: %+v", persisted)
	}
	if len(persisted.RecentErrors) == 0 ||
		persisted.RecentErrors[0].Code != orquestaserver.ServerProcessStaleReasonCodeV0 {
		t.Fatalf("recent_errors sin server_process_stale: %+v", persisted.RecentErrors)
	}
	if got.Status != "stale" ||
		got.StartupStatus != orquestaserver.ServerProcessStaleReasonCodeV0 {
		t.Fatalf("proyeccion final no stale: %+v", got)
	}
}

func TestWaitForStateHealthyV0LimpiaGoalBackendConfiguradoSiMuereTrasReadinessV0(t *testing.T) {
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
	runtimeDir := filepath.Join(root, "runtime")
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	pathEnv := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexPathV0, pathEnv)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		t.Fatalf("socket path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.WriteFile(socketPath, []byte("stale socket"), 0o600); err != nil {
		t.Fatalf("write socket: %v", err)
	}
	if err := os.WriteFile(tmuxLog+".session", []byte("stale session"), 0o600); err != nil {
		t.Fatalf("write fake session: %v", err)
	}

	deadPID := daemonReadinessDeadPIDForTestV0(t)
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			next := daemonReadinessStateForTestV0(r.Host, deadPID)
			next.LastHeartbeatAt = "2026-07-01T00:00:00Z"
			saveDaemonReadinessStateForTestV0(t, config, next)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), os.Getpid())
	state.LastHeartbeatAt = "2026-07-01T00:00:00Z"
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	got, err := waitForStateHealthyV0(config, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "server_exited_after_readiness") {
		t.Fatalf("servidor muerto tras readiness debe reportar server_exited_after_readiness, err=%v state=%+v", err, got)
	}
	rawLog, readErr := os.ReadFile(tmuxLog)
	if readErr != nil {
		t.Fatalf("read tmux log: %v", readErr)
	}
	if !strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("tmux no recibio kill-session tras caida post-readiness: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("session fake no eliminada err=%v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket no eliminado err=%v", err)
	}
}

func daemonReadinessConfigForTestV0(t *testing.T) orquestaserver.ConfigV0 {
	t.Helper()
	return orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		StateDir:  t.TempDir(),
		StateFile: orquestaserver.DefaultStateFileV0,
	})
}

func daemonReadinessStateForTestV0(addr string, pid int) orquestaserver.StateV0 {
	return orquestaserver.StateV0{
		Status:         "running",
		PID:            pid,
		Addr:           addr,
		ProcessRef:     "process-ref-startup-test",
		DaemonEpochRef: "daemon-epoch-startup-test",
		StartedAt:      "2026-06-30T10:00:00Z",
		StartupStatus:  orquestaserver.StartupCheckStatusReadyV0,
		StartupReady:   true,
		RuntimeIdentity: orquestaserver.ServerRuntimeIdentityV0{
			BinarySHA256: "sha256-startup-test",
			BuildRef:     "build-ref-startup-test",
			CommitRef:    "commit-ref-startup-test",
			StartedAt:    "2026-06-30T10:00:00Z",
		},
	}
}

func daemonReadinessDeadPIDForTestV0(t *testing.T) int {
	t.Helper()
	for _, pid := range []int{99999999, 99999998, 99999997} {
		if !processAliveV0(pid) {
			return pid
		}
	}
	t.Skip("no hay PID muerto verificable en este sistema")
	return 0
}

func saveDaemonReadinessStateForTestV0(
	t *testing.T,
	config orquestaserver.ConfigV0,
	state orquestaserver.StateV0,
) {
	t.Helper()
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveServerStateV0: %v", err)
	}
}

func setDaemonReadinessTimingsForTestV0(poll time.Duration, stable time.Duration) func() {
	previousPoll := serverReadinessPollEveryV0
	previousStable := serverReadinessStableDelayV0
	serverReadinessPollEveryV0 = poll
	serverReadinessStableDelayV0 = stable
	return func() {
		serverReadinessPollEveryV0 = previousPoll
		serverReadinessStableDelayV0 = previousStable
	}
}
