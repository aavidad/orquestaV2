package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerReadinessHTTPResponseOKV0NuncaAcepta200DegradedOSHAComoReadinessV0(t *testing.T) {
	expected := daemonReadinessRuntimeIdentityForTestV0()
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{name: "ready exacto", status: http.StatusOK, body: daemonReadinessReadyBodyForTestV0(expected), want: true},
		{name: "identidad distinta", status: http.StatusOK, body: strings.Replace(daemonReadinessReadyBodyForTestV0(expected), strings.Repeat("a", 64), strings.Repeat("c", 64), 1)},
		{name: "degraded 200", status: http.StatusOK, body: `{"status":"degraded_identity","availability_status":"degraded_identity","startup_ready":false,"startup_status":"degraded_identity"}`},
		{name: "sha 200", status: http.StatusOK, body: `{"status":"running","binary_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`},
		{name: "startup status degradado 200", status: http.StatusOK, body: `{"status":"running","availability_status":"running","startup_ready":true,"startup_status":"startup_degraded"}`},
		{name: "ready exacto con HTTP 503", status: http.StatusServiceUnavailable, body: `{"status":"running","availability_status":"running","startup_ready":true,"startup_status":"startup_ready"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := &http.Response{
				StatusCode: test.status,
				Body:       io.NopCloser(strings.NewReader(test.body)),
			}
			if got := serverReadinessHTTPResponseOKV0(response, expected); got != test.want {
				t.Fatalf("readiness=%v want=%v body=%s", got, test.want, test.body)
			}
		})
	}
}

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

	if serverReadinessOKV0(strings.TrimPrefix(server.URL, "http://"), daemonReadinessRuntimeIdentityForTestV0()) {
		t.Fatalf("readiness false no debe pasar aunque healthz este vivo")
	}
}

func TestServerReadinessOKV0RechazaHTTP200DegradedAunqueIncluyaStartupReadyV0(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"schema_version":"orquesta_server_readiness.v0",
			"ready":false,
			"status":"running",
			"availability_status":"running",
			"availability_reason":"server_ready",
			"startup_ready":true,
			"startup_status":"startup_degraded_external_work_goal_backend_required",
			"diagnostics":[{"code":"external_work_goal_backend_required"}]
		}`))
	}))
	defer server.Close()

	if serverReadinessOKV0(strings.TrimPrefix(server.URL, "http://"), daemonReadinessRuntimeIdentityForTestV0()) {
		t.Fatalf("HTTP 200 degraded no debe pasar readiness")
	}
}

func TestServerReadinessOKV0AceptaSoloContratoStartupReadyExactoV0(t *testing.T) {
	expected := daemonReadinessRuntimeIdentityForTestV0()
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeDaemonReadinessReadyForTestV0(w, expected)
	}))
	defer server.Close()

	if !serverReadinessOKV0(strings.TrimPrefix(server.URL, "http://"), expected) {
		t.Fatalf("contrato startup_ready exacto rechazado")
	}
}

func TestWaitForStateHealthyV0AceptaReadinessEstableV0(t *testing.T) {
	expected := daemonReadinessRuntimeIdentityForTestV0()
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		writeDaemonReadinessReadyForTestV0(w, expected)
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
	expected := daemonReadinessRuntimeIdentityForTestV0()
	var calls int32
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			writeDaemonReadinessReadyForTestV0(w, expected)
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
	expected := daemonReadinessRuntimeIdentityForTestV0()
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
		writeDaemonReadinessReadyForTestV0(w, expected)
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

func TestWaitForStateHealthyV0RechazaStateYDaemonViejosAunqueCoincidanV0(t *testing.T) {
	config := daemonReadinessConfigForTestV0(t)
	oldIdentity := daemonReadinessRuntimeIdentityForTestV0()
	oldIdentity.BinarySHA256 = strings.Repeat("c", 64)
	oldIdentity.BuildRef = "build-ref-old-daemon"
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orquestaserver.ServerReadinessEndpointV0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeDaemonReadinessReadyForTestV0(w, oldIdentity)
	}))
	defer server.Close()
	state := daemonReadinessStateForTestV0(strings.TrimPrefix(server.URL, "http://"), os.Getpid())
	state.RuntimeIdentity = oldIdentity
	saveDaemonReadinessStateForTestV0(t, config, state)
	restore := setDaemonReadinessTimingsForTestV0(5*time.Millisecond, 5*time.Millisecond)
	defer restore()

	_, err := waitForStateHealthyV0(config, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "readiness_timeout") {
		t.Fatalf("state/daemon viejos no deben sustituir identidad esperada, err=%v", err)
	}
}

func TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0(t *testing.T) {
	expected := daemonReadinessRuntimeIdentityForTestV0()
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
		writeDaemonReadinessReadyForTestV0(w, expected)
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

func TestWaitForStateHealthyV0NoLimpiaBackendSinIdentidadDelDaemonV0(t *testing.T) {
	expected := daemonReadinessRuntimeIdentityForTestV0()
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
	t.Setenv("ORQUESTA_TEST_BINARY", os.Args[0])
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
	config.RuntimeIdentity = expected
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
		writeDaemonReadinessReadyForTestV0(w, expected)
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
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("read tmux log: %v", readErr)
	}
	if strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("wait sin identidad del daemon no debe limpiar tmux: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("session markerless debe preservarse err=%v", err)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket markerless debe preservarse err=%v", err)
	}
}

func daemonReadinessConfigForTestV0(t *testing.T) orquestaserver.ConfigV0 {
	t.Helper()
	return orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		StateDir:        t.TempDir(),
		StateFile:       orquestaserver.DefaultStateFileV0,
		RuntimeIdentity: daemonReadinessRuntimeIdentityForTestV0(),
	})
}

func daemonReadinessStateForTestV0(addr string, pid int) orquestaserver.StateV0 {
	return orquestaserver.StateV0{
		Status:          "running",
		PID:             pid,
		Addr:            addr,
		ProcessRef:      "process-ref-startup-test",
		DaemonEpochRef:  "daemon-epoch-startup-test",
		StartedAt:       "2026-06-30T10:00:00Z",
		StartupStatus:   orquestaserver.StartupCheckStatusReadyV0,
		StartupReady:    true,
		RuntimeIdentity: daemonReadinessRuntimeIdentityForTestV0(),
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

func daemonReadinessRuntimeIdentityForTestV0() orquestaserver.ServerRuntimeIdentityV0 {
	return orquestaserver.ServerRuntimeIdentityV0{
		SchemaVersion: orquestaserver.ServerRuntimeIdentitySchemaVersionV0,
		BinaryPath:    "/opaque/orquesta-server-test",
		BinaryPathRef: orquestaserver.ServerRuntimeBinaryPathRefV0,
		BinaryName:    "orquesta-server-test",
		BinarySHA256:  strings.Repeat("a", 64),
		BuildRef:      "build-ref-startup-test",
		CommitRef:     strings.Repeat("b", 40),
		StartedAt:     "2026-06-30T10:00:00Z",
	}
}

func daemonReadinessReadyBodyForTestV0(identity orquestaserver.ServerRuntimeIdentityV0) string {
	response := orquestaserver.ServerReadinessV0{
		SchemaVersion:      orquestaserver.ServerReadinessSchemaVersionV0,
		Ready:              true,
		Status:             "running",
		AvailabilityStatus: "running",
		LivenessStatus:     "ok",
		StartupReady:       true,
		StartupStatus:      orquestaserver.StartupCheckStatusReadyV0,
		RuntimeIdentity: orquestaserver.ServerPublicRuntimeIdentityV0{
			SchemaVersion: identity.SchemaVersion,
			BinaryPathRef: identity.BinaryPathRef,
			BinaryName:    identity.BinaryName,
			BinarySHA256:  identity.BinarySHA256,
			BuildRef:      identity.BuildRef,
			CommitRef:     identity.CommitRef,
			StartedAt:     identity.StartedAt,
		},
	}
	raw, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func writeDaemonReadinessReadyForTestV0(
	w http.ResponseWriter,
	identity orquestaserver.ServerRuntimeIdentityV0,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(daemonReadinessReadyBodyForTestV0(identity)))
}
