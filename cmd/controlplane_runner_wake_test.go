package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestProcesarRuntimeTranscriptBatchDespiertaRuntimeOrdersSiIngresaSalida(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Qwen1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: filepath.Join(tmp, "demo"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "demo"),
		Herramienta: "ollama-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "qwen-transcript.log")
	if err := os.WriteFile(logPath, []byte("```diff\n--- a.go\n+++ a.go\n+func ok() {}\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path": logPath,
		"driver":   "tmux_cli_session",
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	wakeCalls := 0
	previousWakeHook := wakeRuntimeOrdersAfterTranscript
	wakeRuntimeOrdersAfterTranscript = func() bool {
		wakeCalls++
		return true
	}
	defer func() {
		wakeRuntimeOrdersAfterTranscript = previousWakeHook
	}()

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n <= 0 {
		t.Fatalf("esperaba transcript ingerido, got=%d", n)
	}
	if wakeCalls != 1 {
		t.Fatalf("runtime_orders debería despertarse exactamente una vez, got=%d", wakeCalls)
	}
}
