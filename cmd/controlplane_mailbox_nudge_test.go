package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"orquesta/db"
)

func TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeConsumeNudgeConProgresoPosterior(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexNudge", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	runDir := filepath.Join(tmp, "tmux-mailbox-nudge-progress")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	now := time.Now().UTC()
	startedAt := now.Add(-30 * time.Minute)
	progressAt := now.Add(-1 * time.Minute)
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexNudge","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexnudge-progress","tmux_pane_id":"%41","started_at":"`+startedAt.Format(time.RFC3339Nano)+`","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","updated_at":"`+progressAt.Format(time.RFC3339Nano)+`","alive":true,"last_output_at":"`+progressAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+progressAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+progressAt.Format(time.RFC3339Nano)+`","started_at":"`+startedAt.Format(time.RFC3339Nano)+`","last_output_at":"`+progressAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+progressAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexnudge-progress",
		"tmux_pane_id":          "%41",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, '{}', CURRENT_TIMESTAMP)`,
		"CodexNudge", proyectoID, "tmux", "session", "orq-codexnudge-progress/%41", "activo", string(metaJSON))
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexNudge",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"sigue"}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	msgCreatedAt := now.Add(-5 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, msgCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at: %v", err)
	}
	msgs, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("CodexNudge"), Estado: strPtr("pendiente")})
	if err != nil || len(msgs) == 0 {
		t.Fatalf("listar mailbox: %v len=%d", err, len(msgs))
	}

	consumida, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msgs[0], handle, "session_resume", time.Now().UTC())
	if err != nil {
		t.Fatalf("consumir mailbox nudge por progreso: %v", err)
	}
	if !consumida {
		t.Fatalf("debería consumir el nudge si ya hubo progreso posterior")
	}

	estado := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("CodexNudge"), Estado: &estado})
	if err != nil {
		t.Fatalf("listar consumidos: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("la mailbox nudge debería quedar consumida, got=%+v", consumidos)
	}
}
