package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"orquesta/coordinacion"
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
	mailboxWakeCalls := 0
	previousWakeHook := wakeRuntimeOrdersAfterTranscript
	previousMailboxWakeHook := wakeRuntimeMailboxAfterTranscript
	wakeRuntimeOrdersAfterTranscript = func() bool {
		wakeCalls++
		return true
	}
	wakeRuntimeMailboxAfterTranscript = func() bool {
		mailboxWakeCalls++
		return true
	}
	defer func() {
		wakeRuntimeOrdersAfterTranscript = previousWakeHook
		wakeRuntimeMailboxAfterTranscript = previousMailboxWakeHook
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
	if mailboxWakeCalls != 0 {
		t.Fatalf("runtime_mailbox no deberia despertarse con una mera ingesta de transcript, got=%d", mailboxWakeCalls)
	}
}

func TestProcesarRuntimeTranscriptBatchDespiertaRuntimeMailboxTrasEntregaGitPremiumBootstrap(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repoDir := filepath.Join(tmp, "repo-git-transcript-premium-bootstrap-wake")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdTest(t, repoDir, "init", "-b", "main")
	runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerBootstrapWake() string { return \"old\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "init")

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Codex1",
		Name:       "wt-codex1-git-transcript-premium-bootstrap-wake",
		Branch:     "orq/orquestador/codex1-git-transcript-premium-bootstrap-wake",
		BaseRef:    "main",
		Reason:     "test_runtime_transcript_git_premium_bootstrap_wake",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerBootstrapWake() string { return \"new\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker in worktree: %v", err)
	}
	msgID, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &projectID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":604,"write_set":["cmd/controlplane_support.go"],"worktree_id":` + strconv.FormatInt(worktree.ID, 10) + `,"ruta_worktree":"` + worktree.Path + `","branch_worktree":"` + worktree.Branch + `","base_ref_worktree":"main"}`,
	})
	if err != nil {
		t.Fatalf("create runtime mailbox premium bootstrap: %v", err)
	}
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &projectID,
		Tipo:          "start",
		Estado:        "completada",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"delivered","mailbox_ids":[%d],"sesion_id":12}`, msgID),
	})
	if err != nil {
		t.Fatalf("create runtime order premium bootstrap: %v", err)
	}

	wakeCalls := 0
	mailboxWakeCalls := 0
	previousWakeHook := wakeRuntimeOrdersAfterTranscript
	previousMailboxWakeHook := wakeRuntimeMailboxAfterTranscript
	wakeRuntimeOrdersAfterTranscript = func() bool {
		wakeCalls++
		return true
	}
	wakeRuntimeMailboxAfterTranscript = func() bool {
		mailboxWakeCalls++
		return true
	}
	defer func() {
		wakeRuntimeOrdersAfterTranscript = previousWakeHook
		wakeRuntimeMailboxAfterTranscript = previousMailboxWakeHook
	}()

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch premium bootstrap wake: %v", err)
	}
	if n < 1 {
		t.Fatalf("se esperaba al menos una entrega premium bootstrap registrada, got=%d", n)
	}
	if wakeCalls < 1 {
		t.Fatalf("runtime_orders deberia despertarse tras registrar entrega premium bootstrap, got=%d", wakeCalls)
	}
	if mailboxWakeCalls < 1 {
		t.Fatalf("runtime_mailbox deberia despertarse tras registrar entrega premium bootstrap, got=%d", mailboxWakeCalls)
	}
	order, err := runtimesService.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order premium bootstrap: %+v err=%v", order, err)
	}
	if order.Estado != "completada" {
		t.Fatalf("runtime order premium bootstrap inesperada: %+v", order)
	}
}
