package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeHandleHotEnsureLoadedNoCierraDuplicadosActivos(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexHot", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	pid := int64(os.Getpid())
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexHot",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "codex-perfil CodexHot",
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexHot", proyectoID, runtimeLegacyID, "cli", "process", "legacy-hot", string(legacyMetaJSON), time.Now().UTC().Add(-5*time.Second)); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	runDir := filepath.Join(tmp, "worker-hot")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexHot","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexhot-1","tmux_pane_id":"%7","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+jsonNumber(int64(os.Getpid()))+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+jsonNumber(int64(os.Getpid()))+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeWorkerID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexHot",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime worker: %v", err)
	}
	workerMetaJSON, _ := json.Marshal(map[string]any{
		"driver":               "tmux_cli_session",
		"worker_manifest_path": manifestPath,
		"tmux_session":         "orq-codexhot-1",
		"tmux_pane_id":         "%7",
		"status_path":          statusPath,
		"heartbeat_path":       heartbeatPath,
		"transport":            "tmux",
		"working_dir":          filepath.Join(tmp, "orquestador"),
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexHot", proyectoID, runtimeWorkerID, "tmux", "session", "orq-codexhot-1/%7", string(workerMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle worker: %v", err)
	}

	runtimeHandleHotReset()
	if err := runtimeHandleHotEnsureLoaded(); err != nil {
		t.Fatalf("runtimeHandleHotEnsureLoaded: %v", err)
	}

	rows, err := DB.Query(`SELECT estado FROM runtime_handles WHERE agente=? AND proyecto_id=? ORDER BY id`, "CodexHot", proyectoID)
	if err != nil {
		t.Fatalf("query runtime_handles: %v", err)
	}
	defer rows.Close()

	var estados []string
	for rows.Next() {
		var estado string
		if err := rows.Scan(&estado); err != nil {
			t.Fatalf("scan estado: %v", err)
		}
		estados = append(estados, estado)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	if len(estados) != 2 {
		t.Fatalf("deberia mantener dos handles, got=%d", len(estados))
	}
	for _, estado := range estados {
		if estado != "activo" {
			t.Fatalf("cargar hot cache no debe mutar estado, got=%v", estados)
		}
	}
}
