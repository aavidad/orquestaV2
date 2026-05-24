package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestRunStatusCommandV0ConsultaStatsDelRunPersistido(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))

	var received orquestamcp.MCPDirectorStatsToolInputV0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != orquestamcp.MCPDirectorStatsHTTPPathV0 {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-Correlation-ID"); got != "run-cli-status-001" {
			t.Fatalf("correlation header = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPDirectorStatsToolResultV0{
			Estado: "ok",
			RunRef: received.RunRef,
		})
	}))
	defer server.Close()
	saveRunStatusServerStateV0(t, stateDir, strings.TrimPrefix(server.URL, "http://"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runStatusCommandV0([]string{"--run-ref", "run-cli-status-001", "--progress"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", exitCode, stderr.String())
	}
	if received.RunRef != "run-cli-status-001" ||
		!received.IncludeProcessRefs ||
		!received.IncludeAgentProgress ||
		!received.IncludeAgentUsage {
		t.Fatalf("input inesperado: %+v", received)
	}
	if !strings.Contains(stdout.String(), `"run_ref":"run-cli-status-001"`) {
		t.Fatalf("stdout no contiene run_ref: %s", stdout.String())
	}
}

func TestRunStatusCommandV0RequiereRunRef(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runStatusCommandV0(nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit=%d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "run_ref requerido") {
		t.Fatalf("stderr inesperado: %s", stderr.String())
	}
}

func saveRunStatusServerStateV0(t *testing.T, stateDir string, addr string) {
	t.Helper()
	store, err := orquestaserver.NewFileStateStoreV0(
		filepath.Join(stateDir, orquestaserver.DefaultStateFileV0),
	)
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), orquestaserver.StateV0{
		Status: "running",
		PID:    1234,
		Addr:   addr,
	}); err != nil {
		t.Fatalf("SaveServerStateV0: %v", err)
	}
}
