package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSmokeCommonShutdownCleanupMataAppServerPropioSinBaseURLV0(t *testing.T) {
	if _, err := os.Stat("/proc/self/cwd"); err != nil {
		t.Skip("procfs no disponible")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash no disponible")
	}
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	runtimeDir := t.TempDir()
	goalDir := filepath.Join(runtimeDir, "goal-srv", "owned-cwd")
	if err := os.MkdirAll(goalDir, 0o700); err != nil {
		t.Fatalf("mkdir goal dir: %v", err)
	}
	process := exec.Command("bash", "-c", "trap '' TERM; exec -a 'codex app-server' sleep 30")
	process.Dir = goalDir
	if err := process.Start(); err != nil {
		t.Fatalf("start fake app-server: %v", err)
	}
	processDone := make(chan error, 1)
	go func() {
		processDone <- process.Wait()
	}()
	defer func() {
		_ = process.Process.Kill()
		select {
		case <-processDone:
		case <-time.After(time.Second):
		}
	}()

	cleanup := exec.Command(
		"bash",
		"-c",
		`source scripts/lib/smoke_common.sh; smoke_shutdown_orquesta_server "" "" 1 1 "$ORQUESTA_TEST_RUNTIME_DIR"`,
	)
	cleanup.Dir = root
	cleanup.Env = append(os.Environ(), "ORQUESTA_TEST_RUNTIME_DIR="+runtimeDir)
	if output, err := cleanup.CombinedOutput(); err != nil {
		t.Fatalf("cleanup failed: %v output=%s", err, string(output))
	}
	select {
	case <-processDone:
	case <-time.After(time.Second):
		t.Fatalf("fake app-server propio sigue vivo tras cleanup sin base_url")
	}
}

func TestSmokeCommonReadinessStateFileRechazaPIDMuertoV0(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash no disponible")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq no disponible")
	}
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	server := newLocalHTTPTestServerOrSkipV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/server/readiness" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"ready":true}`))
	}))
	defer server.Close()
	addr := strings.TrimPrefix(server.URL, "http://")
	stateFile := filepath.Join(t.TempDir(), "orquesta_server_state_v0.json")
	if err := os.WriteFile(
		stateFile,
		[]byte(fmt.Sprintf(`{"addr":%q,"pid":999999999}`, addr)),
		0o600,
	); err != nil {
		t.Fatalf("write state: %v", err)
	}

	check := exec.Command(
		"bash",
		"-c",
		`source scripts/lib/smoke_common.sh; if smoke_wait_orquesta_readiness_from_state_file "$ORQUESTA_TEST_STATE_FILE" 1 0 base_url; then echo "ready:$base_url"; exit 0; fi; echo "pid_dead"; exit 7`,
	)
	check.Dir = root
	check.Env = append(os.Environ(), "ORQUESTA_TEST_STATE_FILE="+stateFile)
	output, err := check.CombinedOutput()
	if err == nil {
		t.Fatalf("readiness acepto PID muerto: output=%s", string(output))
	}
	if !strings.Contains(string(output), "pid_dead") {
		t.Fatalf("salida inesperada: err=%v output=%s", err, string(output))
	}
}

func TestSmokeCommonShutdownCleanupEnviaContratoDirectorV0(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash no disponible")
	}
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep no disponible")
	}
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	process := exec.Command("sleep", "30")
	if err := process.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	processDone := make(chan error, 1)
	go func() {
		processDone <- process.Wait()
	}()
	defer func() {
		_ = process.Process.Kill()
		select {
		case <-processDone:
		case <-time.After(time.Second):
		}
	}()

	received := make(chan map[string]any, 1)
	server := newLocalHTTPTestServerOrSkipV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/server/shutdown" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		select {
		case received <- payload:
		default:
		}
		_ = process.Process.Signal(os.Interrupt)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready","shutdown_ready":true}`))
	}))
	defer server.Close()

	cleanup := exec.Command(
		"bash",
		"-c",
		`source scripts/lib/smoke_common.sh; smoke_shutdown_orquesta_server "$SMOKE_TEST_SERVER_PID" "$SMOKE_TEST_BASE_URL" 2 20 ""`,
	)
	cleanup.Dir = root
	cleanup.Env = append(
		os.Environ(),
		fmt.Sprintf("SMOKE_TEST_SERVER_PID=%d", process.Process.Pid),
		"SMOKE_TEST_BASE_URL="+server.URL,
	)
	if output, err := cleanup.CombinedOutput(); err != nil {
		t.Fatalf("cleanup failed: %v output=%s", err, string(output))
	}
	var payload map[string]any
	select {
	case payload = <-received:
	case <-time.After(time.Second):
		t.Fatalf("shutdown HTTP no recibido")
	}
	if got := payload["idempotency_key"]; got != "idem-smoke-shutdown-cleanup" {
		t.Fatalf("idempotency_key inesperada: %v", got)
	}
	if got := payload["requested_by"]; got != "orquesta-director" {
		t.Fatalf("requested_by inesperado: %v", got)
	}
	if got := payload["cleanup_goal_backends"]; got != true {
		t.Fatalf("cleanup_goal_backends inesperado: %v", got)
	}
}

func newLocalHTTPTestServerOrSkipV0(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("socket local no disponible en sandbox: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}

func TestSmokeCommonReadinessAceptaStartupReadyDegradadoV0(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash no disponible")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq no disponible")
	}
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	server := newLocalHTTPTestServerOrSkipV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/server/readiness" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
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

	check := exec.Command(
		"bash",
		"-c",
		`source scripts/lib/smoke_common.sh; smoke_orquesta_readiness_ok "$TEST_BASE_URL"`,
	)
	check.Dir = root
	check.Env = append(os.Environ(), "TEST_BASE_URL="+server.URL)
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("readiness degradada con startup_ready no aceptada: %v output=%s", err, string(output))
	}
}
