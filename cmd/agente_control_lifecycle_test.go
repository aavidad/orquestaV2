package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestResolverHandleControlAgenteOmiteLegacyTMUXPreferred(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("CodexLegacy", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	insertRuntimeHandleControlLifecycleTest(t, "CodexLegacy", `{"driver":"process_pty_cli","rendered_command":"codex-perfil CodexLegacy --model gpt-5.4"}`)

	handle, err := resolverHandleControlAgente("CodexLegacy", nil)
	if err != nil {
		t.Fatalf("resolverHandleControlAgente: %v", err)
	}
	if handle != nil {
		t.Fatalf("deberia omitir el handle legacy tmux-preferred, obtuvo %+v", handle)
	}
}

func TestResolverHandleControlAgenteOmiteWorkerTMUXStale(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("CodexStale", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	metaJSON := mustStructuredWorkerMetadataJSONControlLifecycleTest(t, time.Now().UTC().Add(-10*time.Minute), "running", true)
	insertRuntimeHandleControlLifecycleTest(t, "CodexStale", metaJSON)

	handle, err := resolverHandleControlAgente("CodexStale", nil)
	if err != nil {
		t.Fatalf("resolverHandleControlAgente: %v", err)
	}
	if handle != nil {
		t.Fatalf("deberia omitir el worker tmux stale, obtuvo %+v", handle)
	}
}

func TestResolverHandleControlAgenteAceptaWorkerTMUXFresh(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("CodexFresh", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	metaJSON := mustStructuredWorkerMetadataJSONControlLifecycleTest(t, time.Now().UTC(), "running", true)
	insertRuntimeHandleControlLifecycleTest(t, "CodexFresh", metaJSON)

	handle, err := resolverHandleControlAgente("CodexFresh", nil)
	if err != nil {
		t.Fatalf("resolverHandleControlAgente: %v", err)
	}
	if handle == nil {
		t.Fatal("deberia aceptar el worker tmux fresco")
	}
}

func insertRuntimeHandleControlLifecycleTest(t *testing.T, agente string, metadataJSON string) int64 {
	t.Helper()
	var sesionID int64
	if err := db.DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)
		RETURNING id`,
		agente, 1, "activa", "codex-cli", "localhost",
	).Scan(&sesionID); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	transporte := "cli"
	handleKind := "process"
	handleRef := strconv.Itoa(os.Getpid())
	if strings.Contains(metadataJSON, `"driver":"tmux_cli_session"`) {
		transporte = "tmux"
		handleKind = "session"
		handleRef = "orq-test/%1"
	}
	res, err := db.DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,'activo','{}',?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		agente, sesionID, transporte, handleKind, handleRef, metadataJSON,
	)
	if err != nil {
		t.Fatalf("insert runtime handle: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	return id
}

func mustStructuredWorkerMetadataJSONControlLifecycleTest(t *testing.T, heartbeatAt time.Time, state string, alive bool) string {
	t.Helper()
	tmp := t.TempDir()
	heartbeatAt = heartbeatAt.UTC()
	workingDir := filepath.Join(tmp, "repo")
	runDir := filepath.Join(workingDir, ".orquesta-runtime", "codexfresh", "20260407-010203-000000001")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestPath := filepath.Join(tmp, "manifest.json")
	runtimeManifestPath := filepath.Join(runDir, "runtime.json")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	statusRaw, err := json.Marshal(map[string]any{
		"state":      state,
		"updated_at": heartbeatAt.Format(time.RFC3339Nano),
		"alive":      alive,
		"child_pid":  os.Getpid(),
	})
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	heartbeatRaw, err := json.Marshal(map[string]any{
		"alive":        alive,
		"heartbeat_at": heartbeatAt.Format(time.RFC3339Nano),
		"child_pid":    os.Getpid(),
	})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	manifestRaw, err := json.Marshal(map[string]any{
		"version":        1,
		"agent":          "CodexFresh",
		"driver":         "tmux_cli_session",
		"transport":      "tmux",
		"tmux_session":   "orq-codexfresh-1",
		"tmux_pane_id":   "%1",
		"status_path":    statusPath,
		"heartbeat_path": heartbeatPath,
		"started_at":     heartbeatAt.Add(-time.Minute).Format(time.RFC3339Nano),
		"working_dir":    workingDir,
		"child_pid":      os.Getpid(),
		"runtime_manifest_path": runtimeManifestPath,
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, statusRaw, 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, heartbeatRaw, 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestRaw, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	runtimeManifestRaw, err := json.Marshal(map[string]any{
		"agente":           "CodexFresh",
		"proyecto":         "orquestador",
		"working_dir":      workingDir,
		"driver":           "tmux_cli_session",
		"transport":        "tmux",
		"pid":              os.Getpid(),
		"supervisor_ref":   runDir,
		"tmux_session":     "orq-codexfresh-1",
		"tmux_pane_id":     "%1",
		"status_path":      statusPath,
		"heartbeat_path":   heartbeatPath,
		"mailbox_delivery_mode": "session_resume",
		"created_at":       heartbeatAt.Add(-time.Minute).Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("marshal runtime manifest: %v", err)
	}
	if err := os.WriteFile(runtimeManifestPath, append(runtimeManifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write runtime manifest: %v", err)
	}
	metaRaw, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"working_dir":           workingDir,
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	return string(metaRaw)
}
