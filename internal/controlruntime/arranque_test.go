package controlruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"orquesta/runtimeagente"
)

func TestArrancarPlanYEnviarInstruccionProceso(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script no disponible en el entorno")
	}

	dir := t.TempDir()
	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    "cat",
			WorkingDir: dir,
			Env:        map[string]string{},
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan: %v", err)
	}
	traceDir := filepath.Dir(arranque.LogPath)
	tracePrefix := filepath.Join(dir, ".orquesta-runtime", "codex1") + string(os.PathSeparator)
	if !strings.HasPrefix(traceDir, tracePrefix) {
		t.Fatalf("trace dir fuera del subdirectorio de trabajo: got=%s want_prefix=%s", traceDir, tracePrefix)
	}
	if filepath.Base(arranque.LogPath) != "pty.log" || filepath.Base(arranque.StdinPath) != "pty.stdin" || filepath.Base(arranque.StdinRawPath) != "pty.stdin.raw" {
		t.Fatalf("nombres de artefactos PTY inesperados: log=%s stdin=%s stdin_raw=%s", arranque.LogPath, arranque.StdinPath, arranque.StdinRawPath)
	}
	pid64 := int64(arranque.PID)

	t.Cleanup(func() {
		_, _, _ = DetenerProceso(ObjetivoProceso{
			PID:          &pid64,
			HandleKind:   arranque.HandleKind,
			HandleRef:    arranque.HandleRef,
			MetadataJSON: arranque.MetadataJSON,
		})
	})

	time.Sleep(150 * time.Millisecond)
	aplicado, pid, err := EnviarInstruccionProceso(ObjetivoProceso{
		PID:          &pid64,
		HandleKind:   arranque.HandleKind,
		HandleRef:    arranque.HandleRef,
		MetadataJSON: arranque.MetadataJSON,
	}, "hola runtime")
	if err != nil {
		t.Fatalf("enviar instruccion: %v", err)
	}
	if !aplicado || pid != arranque.PID {
		t.Fatalf("entrega directa inesperada: aplicado=%t pid=%d", aplicado, pid)
	}

	time.Sleep(250 * time.Millisecond)
	logData, err := os.ReadFile(arranque.LogPath)
	if err != nil {
		t.Fatalf("leer log: %v", err)
	}
	if !strings.Contains(string(logData), "hola runtime") {
		t.Fatalf("el log no contiene la instruccion: %s", string(logData))
	}
	rawData, err := os.ReadFile(arranque.StdinRawPath)
	if err != nil {
		t.Fatalf("leer stdin raw: %v", err)
	}
	if string(rawData) != "hola runtime\n" {
		t.Fatalf("stdin raw inesperado: %q", string(rawData))
	}

	manifestPath := filepath.Join(traceDir, "runtime.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("leer manifest runtime: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsear manifest runtime: %v", err)
	}
	if got, _ := manifest["agente"].(string); got != "Codex1" {
		t.Fatalf("agente inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["proyecto"].(string); got != "orquestador" {
		t.Fatalf("proyecto inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["log_path"].(string); got != arranque.LogPath {
		t.Fatalf("log_path inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["stdin_path"].(string); got != arranque.StdinPath {
		t.Fatalf("stdin_path inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["stdin_raw_path"].(string); got != arranque.StdinRawPath {
		t.Fatalf("stdin_raw_path inesperado en manifest: %+v", manifest)
	}
	if got, _ := manifest["can_send_input"].(bool); !got {
		t.Fatalf("el launcher PTY local deberia permitir input interactivo por defecto: %+v", manifest)
	}
	if got, _ := manifest["mailbox_delivery_mode"].(string); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("mailbox_delivery_mode inesperado en manifest: %+v", manifest)
	}
	workerManifestPath, _ := manifest["worker_manifest_path"].(string)
	workerStatusPath, _ := manifest["worker_status_path"].(string)
	workerHeartbeatPath, _ := manifest["worker_heartbeat_path"].(string)
	if workerManifestPath == "" || workerStatusPath == "" || workerHeartbeatPath == "" {
		t.Fatalf("faltan rutas de worker artifacts en runtime.json: %+v", manifest)
	}

	workerManifestData, err := os.ReadFile(workerManifestPath)
	if err != nil {
		t.Fatalf("leer worker manifest: %v", err)
	}
	var workerManifest map[string]any
	if err := json.Unmarshal(workerManifestData, &workerManifest); err != nil {
		t.Fatalf("parsear worker manifest: %v", err)
	}
	if got, _ := workerManifest["agent"].(string); got != "Codex1" {
		t.Fatalf("agent inesperado en worker manifest: %+v", workerManifest)
	}
	if got, _ := workerManifest["status_path"].(string); got != workerStatusPath {
		t.Fatalf("status_path inesperado en worker manifest: %+v", workerManifest)
	}
	if got, _ := workerManifest["heartbeat_path"].(string); got != workerHeartbeatPath {
		t.Fatalf("heartbeat_path inesperado en worker manifest: %+v", workerManifest)
	}

	workerStatusData, err := os.ReadFile(workerStatusPath)
	if err != nil {
		t.Fatalf("leer worker status: %v", err)
	}
	var workerStatus map[string]any
	if err := json.Unmarshal(workerStatusData, &workerStatus); err != nil {
		t.Fatalf("parsear worker status: %v", err)
	}
	state, _ := workerStatus["state"].(string)
	if state != workerStatusStarting && state != workerStatusRunning {
		t.Fatalf("state inesperado en worker status: %+v", workerStatus)
	}
	if alive, _ := workerStatus["alive"].(bool); !alive {
		t.Fatalf("worker status deberia marcar alive=true: %+v", workerStatus)
	}

	workerHeartbeatData, err := os.ReadFile(workerHeartbeatPath)
	if err != nil {
		t.Fatalf("leer worker heartbeat: %v", err)
	}
	var workerHeartbeat map[string]any
	if err := json.Unmarshal(workerHeartbeatData, &workerHeartbeat); err != nil {
		t.Fatalf("parsear worker heartbeat: %v", err)
	}
	if alive, _ := workerHeartbeat["alive"].(bool); !alive {
		t.Fatalf("worker heartbeat deberia marcar alive=true: %+v", workerHeartbeat)
	}
	if got, _ := workerHeartbeat["agent"].(string); got != "Codex1" {
		t.Fatalf("agent inesperado en worker heartbeat: %+v", workerHeartbeat)
	}

	var metaJSON map[string]any
	if err := json.Unmarshal([]byte(arranque.MetadataJSON), &metaJSON); err != nil {
		t.Fatalf("metadata arranque: %v", err)
	}
	if got, _ := metaJSON["can_send_input"].(bool); !got {
		t.Fatalf("metadata del arranque sin can_send_input=true: %+v", metaJSON)
	}
	if got, _ := metaJSON["mailbox_delivery_mode"].(string); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("metadata del arranque sin mailbox_delivery_mode=interactive: %+v", metaJSON)
	}
	if got, _ := metaJSON["worker_manifest_path"].(string); got != workerManifestPath {
		t.Fatalf("metadata del arranque sin worker_manifest_path: %+v", metaJSON)
	}
	if got, _ := metaJSON["worker_status_path"].(string); got != workerStatusPath {
		t.Fatalf("metadata del arranque sin worker_status_path: %+v", metaJSON)
	}
	if got, _ := metaJSON["worker_heartbeat_path"].(string); got != workerHeartbeatPath {
		t.Fatalf("metadata del arranque sin worker_heartbeat_path: %+v", metaJSON)
	}

	var capsJSON map[string]any
	if err := json.Unmarshal([]byte(arranque.CapabilitiesJSON), &capsJSON); err != nil {
		t.Fatalf("capabilities arranque: %v", err)
	}
	if got, _ := capsJSON["can_send_input"].(bool); !got {
		t.Fatalf("capabilities del arranque sin can_send_input=true: %+v", capsJSON)
	}
	if got, _ := capsJSON["mailbox_delivery_mode"].(string); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("capabilities del arranque sin mailbox_delivery_mode=interactive: %+v", capsJSON)
	}
}

func TestRuntimeArtifactsRunDirOrdenaPorAgenteYFecha(t *testing.T) {
	dir := t.TempDir()
	req := SolicitudArranque{
		Agente:   "Codex 2",
		Proyecto: "demo",
		Plan: &runtimeagente.LaunchPlan{
			WorkingDir: dir,
		},
	}
	t1 := time.Date(2026, 3, 29, 12, 30, 1, 123, time.UTC)
	t2 := t1.Add(time.Second)

	dir1 := runtimeArtifactsRunDir(req, t1)
	dir2 := runtimeArtifactsRunDir(req, t2)
	prefix := filepath.Join(dir, ".orquesta-runtime", "codex-2") + string(os.PathSeparator)
	if !strings.HasPrefix(dir1, prefix) || !strings.HasPrefix(dir2, prefix) {
		t.Fatalf("las trazas deberian agruparse por agente bajo el working dir: dir1=%s dir2=%s prefix=%s", dir1, dir2, prefix)
	}
	if !(dir1 < dir2) {
		t.Fatalf("las rutas deberian quedar ordenables cronologicamente: dir1=%s dir2=%s", dir1, dir2)
	}
}

func TestRenderedCommandLooksLikeTMUXPreferredCLIReconoceClaudeCode(t *testing.T) {
	if !RenderedCommandLooksLikeTMUXPreferredCLI("claude-code --print \"hola\"") {
		t.Fatalf("claude-code deberia preferir backend tmux")
	}
}

func TestArrancarPlanLocalRespetaOverrideCanSendInput(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script no disponible en el entorno")
	}

	dir := t.TempDir()
	canSendInput := false
	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:      "cat",
			WorkingDir:   dir,
			CanSendInput: &canSendInput,
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan: %v", err)
	}
	pid64 := int64(arranque.PID)
	t.Cleanup(func() {
		_, _, _ = DetenerProceso(ObjetivoProceso{
			PID:          &pid64,
			HandleKind:   arranque.HandleKind,
			HandleRef:    arranque.HandleRef,
			MetadataJSON: arranque.MetadataJSON,
		})
	})

	var manifest map[string]any
	data, err := os.ReadFile(filepath.Join(filepath.Dir(arranque.LogPath), "runtime.json"))
	if err != nil {
		t.Fatalf("leer manifest runtime: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsear manifest runtime: %v", err)
	}
	if got, _ := manifest["can_send_input"].(bool); got {
		t.Fatalf("el manifest deberia respetar el override can_send_input=false: %+v", manifest)
	}
	if got, _ := manifest["mailbox_delivery_mode"].(string); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("mailbox_delivery_mode inesperado en manifest: %+v", manifest)
	}

	var metaJSON map[string]any
	if err := json.Unmarshal([]byte(arranque.MetadataJSON), &metaJSON); err != nil {
		t.Fatalf("metadata arranque: %v", err)
	}
	if got, _ := metaJSON["can_send_input"].(bool); got {
		t.Fatalf("metadata del arranque deberia respetar can_send_input=false: %+v", metaJSON)
	}
	if got, _ := metaJSON["mailbox_delivery_mode"].(string); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("metadata del arranque sin mailbox_delivery_mode=bootstrap_only: %+v", metaJSON)
	}

	var capsJSON map[string]any
	if err := json.Unmarshal([]byte(arranque.CapabilitiesJSON), &capsJSON); err != nil {
		t.Fatalf("capabilities arranque: %v", err)
	}
	if got, _ := capsJSON["can_send_input"].(bool); got {
		t.Fatalf("capabilities del arranque deberian respetar can_send_input=false: %+v", capsJSON)
	}
	if got, _ := capsJSON["mailbox_delivery_mode"].(string); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("capabilities del arranque sin mailbox_delivery_mode=bootstrap_only: %+v", capsJSON)
	}
}

func TestEnviarInstruccionProcesoRespetaCanSendInputFalse(t *testing.T) {
	dir := t.TempDir()
	pid := int64(os.Getpid())
	meta := `{"stdin_path":"` + filepath.Join(dir, "pty.stdin") + `","can_send_input":false}`
	aplicado, gotPID, err := EnviarInstruccionProceso(ObjetivoProceso{
		PID:          &pid,
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: meta,
	}, "hola")
	if err != nil {
		t.Fatalf("EnviarInstruccionProceso: %v", err)
	}
	if aplicado {
		t.Fatalf("no deberia intentar stdin cuando can_send_input=false")
	}
	if gotPID != int(pid) {
		t.Fatalf("pid inesperado: got=%d want=%d", gotPID, pid)
	}
}

func TestArrancarPlanReportaErrorTempranoDelBroker(t *testing.T) {
	dir := t.TempDir()
	req := SolicitudArranque{
		Agente:   "CodexFallo",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    filepath.Join(dir, "bin", "launcher-inexistente"),
			WorkingDir: dir,
			Env:        map[string]string{},
		},
	}

	_, err := ArrancarPlan(req)
	if err == nil {
		t.Fatalf("arrancar plan deberia fallar")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "no such file or directory") {
		t.Fatalf("error temprano inesperado: %v", err)
	}

	baseDir := runtimeArtifactsBaseDir(req)
	entries, readErr := os.ReadDir(baseDir)
	if readErr != nil {
		t.Fatalf("leer runtime dir: %v", readErr)
	}
	if len(entries) == 0 {
		t.Fatalf("sin runs creados en %s", baseDir)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	runDir := filepath.Join(baseDir, entries[len(entries)-1].Name())
	statusData, readErr := os.ReadFile(filepath.Join(runDir, "status.json"))
	if readErr != nil {
		t.Fatalf("leer status.json: %v", readErr)
	}
	var status map[string]any
	if err := json.Unmarshal(statusData, &status); err != nil {
		t.Fatalf("parsear status.json: %v", err)
	}
	if got, _ := status["state"].(string); got != workerStatusFailed {
		t.Fatalf("state inesperado en status.json: %+v", status)
	}
	if got, _ := status["alive"].(bool); got {
		t.Fatalf("status.json deberia marcar alive=false: %+v", status)
	}
	exitError, _ := status["exit_error"].(string)
	if !strings.Contains(strings.ToLower(exitError), "no such file or directory") {
		t.Fatalf("exit_error inesperado en status.json: %+v", status)
	}
}

func TestArrancarPlanTMUXEscribeArtefactosWorker(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)

	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "CodexTMUX",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:         "/bin/echo",
			Args:            []string{"hola"},
			WorkingDir:      dir,
			PerfilOperativo: "qa-heavy",
			Env: map[string]string{
				"ORQUESTA_TERMINAL_BACKEND": "tmux",
				"ORQUESTA_TMUX_BIN":         fakeTmux,
			},
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan tmux: %v", err)
	}
	if filepath.Base(arranque.LogPath) != "tmux.log" {
		t.Fatalf("log path inesperado para tmux: %s", arranque.LogPath)
	}
	traceDir := filepath.Dir(arranque.LogPath)
	manifestPath := filepath.Join(traceDir, "runtime.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("leer runtime manifest tmux: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsear runtime manifest tmux: %v", err)
	}
	if got, _ := manifest["driver"].(string); got != "tmux_cli_session" {
		t.Fatalf("driver inesperado en runtime manifest: %+v", manifest)
	}
	if got, _ := manifest["transport"].(string); got != "tmux" {
		t.Fatalf("transport inesperado en runtime manifest: %+v", manifest)
	}
	if got, _ := manifest["perfil_operativo"].(string); got != "qa-heavy" {
		t.Fatalf("perfil_operativo inesperado en runtime manifest: %+v", manifest)
	}
	if got := strings.TrimSpace(arranque.HandleKind); got != "session" {
		t.Fatalf("handle kind inesperado para tmux: %q", got)
	}
	if got, _ := manifest["tmux_pane_id"].(string); got != "%1" {
		t.Fatalf("tmux_pane_id inesperado en runtime manifest: %+v", manifest)
	}
	if sessionName, _ := manifest["tmux_session"].(string); strings.TrimSpace(sessionName) == "" {
		t.Fatalf("tmux_session inesperado en runtime manifest: %+v", manifest)
	} else if want := sessionName + "/%1"; strings.TrimSpace(arranque.HandleRef) != want {
		t.Fatalf("handle ref inesperado para tmux: got=%q want=%q", arranque.HandleRef, want)
	}
	workerManifestPath, _ := manifest["worker_manifest_path"].(string)
	workerStatusPath, _ := manifest["worker_status_path"].(string)
	workerHeartbeatPath, _ := manifest["worker_heartbeat_path"].(string)
	if workerManifestPath == "" || workerStatusPath == "" || workerHeartbeatPath == "" {
		t.Fatalf("faltan worker artifacts en runtime manifest tmux: %+v", manifest)
	}

	time.Sleep(250 * time.Millisecond)

	workerManifestData, err := os.ReadFile(workerManifestPath)
	if err != nil {
		t.Fatalf("leer worker manifest tmux: %v", err)
	}
	var workerManifest map[string]any
	if err := json.Unmarshal(workerManifestData, &workerManifest); err != nil {
		t.Fatalf("parsear worker manifest tmux: %v", err)
	}
	if got, _ := workerManifest["driver"].(string); got != "tmux_cli_session" {
		t.Fatalf("driver inesperado en worker manifest tmux: %+v", workerManifest)
	}
	if got, _ := workerManifest["transport"].(string); got != "tmux" {
		t.Fatalf("transport inesperado en worker manifest tmux: %+v", workerManifest)
	}
	if got, _ := workerManifest["execution_profile"].(string); got != "qa-heavy" {
		t.Fatalf("execution_profile inesperado en worker manifest tmux: %+v", workerManifest)
	}

	workerStatusData, err := os.ReadFile(workerStatusPath)
	if err != nil {
		t.Fatalf("leer worker status tmux: %v", err)
	}
	var workerStatus map[string]any
	if err := json.Unmarshal(workerStatusData, &workerStatus); err != nil {
		t.Fatalf("parsear worker status tmux: %v", err)
	}
	state, _ := workerStatus["state"].(string)
	if state != workerStatusStarting && state != workerStatusReady && state != workerStatusRunning && state != workerStatusStopped {
		t.Fatalf("state inesperado en worker status tmux: %+v", workerStatus)
	}
	if state == workerStatusReady {
		if got, _ := workerStatus["ready_at"].(string); strings.TrimSpace(got) == "" {
			t.Fatalf("worker ready sin ready_at: %+v", workerStatus)
		}
	}

	workerHeartbeatData, err := os.ReadFile(workerHeartbeatPath)
	if err != nil {
		t.Fatalf("leer worker heartbeat tmux: %v", err)
	}
	var workerHeartbeat map[string]any
	if err := json.Unmarshal(workerHeartbeatData, &workerHeartbeat); err != nil {
		t.Fatalf("parsear worker heartbeat tmux: %v", err)
	}
	if got, _ := workerHeartbeat["agent"].(string); got != "CodexTMUX" {
		t.Fatalf("agent inesperado en worker heartbeat tmux: %+v", workerHeartbeat)
	}
	if state == workerStatusReady {
		if got, _ := workerHeartbeat["ready_at"].(string); strings.TrimSpace(got) == "" {
			t.Fatalf("worker heartbeat ready sin ready_at: %+v", workerHeartbeat)
		}
	}

	var metaJSON map[string]any
	if err := json.Unmarshal([]byte(arranque.MetadataJSON), &metaJSON); err != nil {
		t.Fatalf("metadata arranque tmux: %v", err)
	}
	if got, _ := metaJSON["driver"].(string); got != "tmux_cli_session" {
		t.Fatalf("metadata sin driver tmux_cli_session: %+v", metaJSON)
	}
	if got, _ := metaJSON["perfil_operativo"].(string); got != "qa-heavy" {
		t.Fatalf("metadata sin perfil_operativo canonico: %+v", metaJSON)
	}
	if got, _ := metaJSON["tmux_pane_id"].(string); got != "%1" {
		t.Fatalf("metadata sin tmux_pane_id: %+v", metaJSON)
	}

	invocations, err := os.ReadFile(filepath.Join(dir, "fake-tmux-state", "invocations.log"))
	if err != nil {
		t.Fatalf("leer invocations fake tmux: %v", err)
	}
	logText := string(invocations)
	if !strings.Contains(logText, "new-session") || !strings.Contains(logText, "pipe-pane") {
		t.Fatalf("fake tmux no recibio comandos esperados: %s", logText)
	}
	logData, err := os.ReadFile(arranque.LogPath)
	if err != nil {
		t.Fatalf("leer tmux.log: %v", err)
	}
	if !strings.Contains(string(logData), "> ready") {
		t.Fatalf("tmux.log deberia sembrarse con el pane inicial: %q", string(logData))
	}
}

func TestArrancarPlanTMUXTimeoutArranque(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := filepath.Join(dir, "fake-tmux-slow")
	script := `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  new-session)
    sleep 1
    printf '%s\n' "orq-timeout|worker|%1|12345"
    ;;
  *)
    exit 0
    ;;
esac
`
	if err := os.WriteFile(fakeTmux, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux slow: %v", err)
	}
	t.Setenv("ORQUESTA_TMUX_START_TIMEOUT_MS", "25")

	_, err := ArrancarPlan(SolicitudArranque{
		Agente:   "CodexTMUXTimeout",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    "/bin/echo",
			Args:       []string{"hola"},
			WorkingDir: dir,
			Env: map[string]string{
				"ORQUESTA_TERMINAL_BACKEND": "tmux",
				"ORQUESTA_TMUX_BIN":         fakeTmux,
			},
		},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout arrancando tmux") {
		t.Fatalf("esperaba timeout de tmux, got=%v", err)
	}
}

func TestArrancarPlanPrefiereTMUXPorDefectoParaCodexCLI(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)

	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "CodexTMUXAuto",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:         "codex-perfil",
			Args:            []string{"CodexTMUXAuto", "--version"},
			WorkingDir:      dir,
			PerfilOperativo: "persistente",
			Env: map[string]string{
				"ORQUESTA_TMUX_BIN": fakeTmux,
			},
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan tmux auto: %v", err)
	}
	if filepath.Base(arranque.LogPath) != "tmux.log" {
		t.Fatalf("deberia preferir tmux por defecto para codex-cli: %s", arranque.LogPath)
	}
	var metaJSON map[string]any
	if err := json.Unmarshal([]byte(arranque.MetadataJSON), &metaJSON); err != nil {
		t.Fatalf("metadata arranque: %v", err)
	}
	if got, _ := metaJSON["driver"].(string); got != "tmux_cli_session" {
		t.Fatalf("metadata sin driver tmux_cli_session: %+v", metaJSON)
	}
	if got, _ := metaJSON["profile_name"].(string); got != "CodexTMUXAuto" {
		t.Fatalf("metadata sin profile_name codex: %+v", metaJSON)
	}
	if got, _ := metaJSON["profile_status_wrapper"].(string); filepath.Base(got) != "codex-perfil" {
		t.Fatalf("metadata sin profile_status_wrapper canonico: %+v", metaJSON)
	}
	if got, _ := metaJSON["perfil_operativo"].(string); got != "persistente" {
		t.Fatalf("metadata sin perfil_operativo persistente: %+v", metaJSON)
	}

	traceDir := filepath.Dir(arranque.LogPath)
	workerManifestPath := filepath.Join(traceDir, "manifest.json")
	data, err := os.ReadFile(workerManifestPath)
	if err != nil {
		t.Fatalf("leer worker manifest tmux auto: %v", err)
	}
	var workerManifest map[string]any
	if err := json.Unmarshal(data, &workerManifest); err != nil {
		t.Fatalf("parsear worker manifest tmux auto: %v", err)
	}
	if got, _ := workerManifest["profile"].(string); got != "CodexTMUXAuto" {
		t.Fatalf("worker manifest sin profile codex: %+v", workerManifest)
	}
	if got, _ := workerManifest["execution_profile"].(string); got != "persistente" {
		t.Fatalf("worker manifest sin execution_profile persistente: %+v", workerManifest)
	}
	if got, _ := workerManifest["profile_status_wrapper"].(string); filepath.Base(got) != "codex-perfil" {
		t.Fatalf("worker manifest sin profile_status_wrapper canonico: %+v", workerManifest)
	}
}

func TestLocalTerminalBackendNoFuerzaTMUXParaProcesoGenerico(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)

	backend := localTerminalBackend(SolicitudArranque{
		Agente:   "WorkerX",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    "/bin/echo",
			Args:       []string{"hola"},
			WorkingDir: dir,
			Env: map[string]string{
				"ORQUESTA_TMUX_BIN": fakeTmux,
			},
		},
	})
	if backend != "" {
		t.Fatalf("un proceso generico no deberia forzar tmux: %q", backend)
	}
}

func TestArrancarPlanCodexCLIRequiereTMUXPorDefecto(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", "")
	t.Setenv("ORQUESTA_TMUX_BIN", "")

	_, err := ArrancarPlan(SolicitudArranque{
		Agente:   "CodexSinTMUX",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    "codex-perfil",
			Args:       []string{"CodexSinTMUX", "--version"},
			WorkingDir: dir,
		},
	})
	if err == nil {
		t.Fatal("deberia exigir tmux para codex-cli por defecto")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "tmux") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestArrancarPlanCodexCLIPermitePTYPorOverrideExplicito(t *testing.T) {
	dir := t.TempDir()
	fakeCodex := filepath.Join(dir, "codex-perfil")
	if err := os.WriteFile(fakeCodex, []byte("#!/usr/bin/env bash\nprintf 'hola codex pty\\n'\nsleep 1\n"), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}

	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "CodexPTY",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    fakeCodex,
			Args:       []string{"CodexPTY"},
			WorkingDir: dir,
			Env: map[string]string{
				"ORQUESTA_TERMINAL_BACKEND": "pty",
			},
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan codex pty: %v", err)
	}
	if filepath.Base(arranque.LogPath) != "pty.log" {
		t.Fatalf("deberia usar pty por override explicito: %s", arranque.LogPath)
	}
}

func TestArrancarPlanPrefiereTMUXPorDefectoParaOllamaCLI(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)

	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "Ollama1",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:    "ollama",
			Args:       []string{"run", "qwen2.5-coder:7b"},
			WorkingDir: dir,
			Env: map[string]string{
				"ORQUESTA_TMUX_BIN": fakeTmux,
			},
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan ollama tmux: %v", err)
	}
	if filepath.Base(arranque.LogPath) != "tmux.log" {
		t.Fatalf("deberia preferir tmux por defecto para ollama-cli: %s", arranque.LogPath)
	}
}

func TestEnviarInstruccionProcesoTMUXUsaSendKeys(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)
	pid := int64(os.Getpid())
	meta := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_pane_id":"%%42","can_send_input":true}`, fakeTmux)

	aplicado, gotPID, err := EnviarInstruccionProceso(ObjetivoProceso{
		PID:          &pid,
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: meta,
	}, "hola runtime tmux")
	if err != nil {
		t.Fatalf("EnviarInstruccionProceso tmux: %v", err)
	}
	if !aplicado {
		t.Fatal("deberia entregar la instruccion por tmux send-keys")
	}
	if gotPID != int(pid) {
		t.Fatalf("pid inesperado: got=%d want=%d", gotPID, pid)
	}
	invocations, err := os.ReadFile(filepath.Join(dir, "fake-tmux-state", "invocations.log"))
	if err != nil {
		t.Fatalf("leer invocations fake tmux: %v", err)
	}
	logText := string(invocations)
	if !strings.Contains(logText, "send-keys -t %42 -l hola runtime tmux") {
		t.Fatalf("faltan send-keys literales en fake tmux: %s", logText)
	}
	if !strings.Contains(logText, "send-keys -t %42 Enter") {
		t.Fatalf("falta send-keys Enter en fake tmux: %s", logText)
	}
}

func TestEnviarInstruccionProcesoTMUXUsaSendKeysSinPID(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)
	meta := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_pane_id":"%%43","can_send_input":true}`, fakeTmux)

	aplicado, gotPID, err := EnviarInstruccionProceso(ObjetivoProceso{
		HandleKind:   "session",
		HandleRef:    "orq-codex7/%43",
		MetadataJSON: meta,
	}, "hola runtime tmux sin pid")
	if err != nil {
		t.Fatalf("EnviarInstruccionProceso tmux sin pid: %v", err)
	}
	if !aplicado {
		t.Fatal("deberia entregar la instruccion por tmux send-keys sin requerir PID")
	}
	if gotPID != 0 {
		t.Fatalf("pid inesperado sin PID origen: got=%d", gotPID)
	}
	invocations, err := os.ReadFile(filepath.Join(dir, "fake-tmux-state", "invocations.log"))
	if err != nil {
		t.Fatalf("leer invocations fake tmux: %v", err)
	}
	logText := string(invocations)
	if !strings.Contains(logText, "send-keys -t %43 -l hola runtime tmux sin pid") {
		t.Fatalf("tmux no recibió el texto esperado sin PID: %s", logText)
	}
}

func TestEnviarInstruccionTMUXVerificadaConfirmaTareaActiva(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxDispatchScript(t, dir, "active_after_enter")

	result, err := EnviarInstruccionTMUXVerificada(ObjetivoProceso{
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_pane_id":"%%77"}`, fakeTmux),
	}, "haz refresh corto")
	if err != nil {
		t.Fatalf("EnviarInstruccionTMUXVerificada: %v", err)
	}
	if !result.Attempted || !result.Delivered || !result.ActiveTask {
		t.Fatalf("resultado tmux verificado inesperado: %+v", result)
	}
	if result.Reason != "active_task" {
		t.Fatalf("reason inesperado: %+v", result)
	}
}

func TestEnviarInstruccionTMUXVerificadaAceptaCambioDeModeloPorRateLimit(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxDispatchScript(t, dir, "rate_limit_switch")

	result, err := EnviarInstruccionTMUXVerificada(ObjetivoProceso{
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_pane_id":"%%77"}`, fakeTmux),
	}, "haz refresh corto")
	if err != nil {
		t.Fatalf("EnviarInstruccionTMUXVerificada: %v", err)
	}
	if !result.Attempted || !result.Delivered {
		t.Fatalf("deberia desbloquear el cambio de modelo y entregar la instruccion: %+v", result)
	}
	invocations, err := os.ReadFile(filepath.Join(dir, "fake-tmux-dispatch-state", "invocations.log"))
	if err != nil {
		t.Fatalf("leer invocations fake tmux: %v", err)
	}
	logText := string(invocations)
	if !strings.Contains(logText, "send-keys -t %77 1") || !strings.Contains(logText, "send-keys -t %77 C-m") {
		t.Fatalf("deberia aceptar el menu de cambio de modelo antes del dispatch: %s", logText)
	}
}

func TestEnviarInstruccionTMUXVerificadaDejaPendienteSiNoConfirmaConsumo(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxDispatchScript(t, dir, "draft_persists")

	result, err := EnviarInstruccionTMUXVerificada(ObjetivoProceso{
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_pane_id":"%%77"}`, fakeTmux),
	}, "haz refresh corto")
	if err != nil {
		t.Fatalf("EnviarInstruccionTMUXVerificada: %v", err)
	}
	if !result.Attempted {
		t.Fatalf("deberia intentar el dispatch tmux: %+v", result)
	}
	if result.Delivered {
		t.Fatalf("no deberia marcar entregado sin confirmacion: %+v", result)
	}
	if result.Reason != "unconfirmed" {
		t.Fatalf("reason inesperado: %+v", result)
	}
}

func TestEnviarInstruccionTMUXVerificadaDejaPendienteSiPaneSigueIgual(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxDispatchScript(t, dir, "ready_unchanged")

	result, err := EnviarInstruccionTMUXVerificada(ObjetivoProceso{
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_pane_id":"%%77"}`, fakeTmux),
	}, "haz refresh corto")
	if err != nil {
		t.Fatalf("EnviarInstruccionTMUXVerificada: %v", err)
	}
	if !result.Attempted {
		t.Fatalf("deberia intentar el dispatch tmux: %+v", result)
	}
	if result.Delivered {
		t.Fatalf("no deberia marcar entregado si el pane sigue igual: %+v", result)
	}
	if result.Reason != "unconfirmed" {
		t.Fatalf("reason inesperado: %+v", result)
	}
}

func TestEnviarInstruccionTMUXVerificadaAceptaEditsGeminiYActivaTrabajo(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxDispatchScript(t, dir, "gemini_accept_edits")

	result, err := EnviarInstruccionTMUXVerificada(ObjetivoProceso{
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_pane_id":"%%77"}`, fakeTmux),
	}, "haz refresh corto")
	if err != nil {
		t.Fatalf("EnviarInstruccionTMUXVerificada: %v", err)
	}
	if !result.Attempted || !result.Delivered || !result.ActiveTask {
		t.Fatalf("resultado tmux verificado inesperado: %+v", result)
	}
	if result.Reason != "active_task" {
		t.Fatalf("reason inesperado: %+v", result)
	}
}

func TestDetenerProcesoTMUXUsaKillSession(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)
	pid := int64(os.Getpid())
	meta := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-codex7-111552"}`, fakeTmux)

	aplicado, gotPID, err := DetenerProceso(ObjetivoProceso{
		PID:          &pid,
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: meta,
	})
	if err != nil {
		t.Fatalf("DetenerProceso tmux: %v", err)
	}
	if !aplicado {
		t.Fatal("deberia detener la sesion tmux")
	}
	if gotPID != int(pid) {
		t.Fatalf("pid inesperado: got=%d want=%d", gotPID, pid)
	}
	invocations, err := os.ReadFile(filepath.Join(dir, "fake-tmux-state", "invocations.log"))
	if err != nil {
		t.Fatalf("leer invocations fake tmux: %v", err)
	}
	if !strings.Contains(string(invocations), "kill-session -t orq-codex7-111552") {
		t.Fatalf("faltan kill-session en fake tmux: %s", string(invocations))
	}
}

func TestDetenerProcesoTMUXIgnoraSesionYaAusente(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "fake-tmux-missing")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
state_dir=%q
mkdir -p "$state_dir"
printf '%%s\n' "$*" >> "$state_dir/invocations.log"
if [[ "${1:-}" == "kill-session" ]]; then
  echo "can't find session: orq-codex7-111552" >&2
  exit 1
fi
exit 0
`, filepath.Join(dir, "fake-tmux-state"))
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux missing script: %v", err)
	}

	pid := int64(os.Getpid())
	meta := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-codex7-111552"}`, scriptPath)
	aplicado, gotPID, err := DetenerProceso(ObjetivoProceso{
		PID:          &pid,
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: meta,
	})
	if err != nil {
		t.Fatalf("DetenerProceso tmux missing session: %v", err)
	}
	if !aplicado {
		t.Fatal("deberia tratar la sesion tmux ausente como stop aplicado")
	}
	if gotPID != int(pid) {
		t.Fatalf("pid inesperado: got=%d want=%d", gotPID, pid)
	}
}

func TestDetenerProcesoTMUXBuscaTmuxEnPATHSiFaltaEnMetadata(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeFakeTmuxScript(t, dir)
	pathTmux := filepath.Join(dir, "tmux")
	data, err := os.ReadFile(fakeTmux)
	if err != nil {
		t.Fatalf("leer fake tmux: %v", err)
	}
	if err := os.WriteFile(pathTmux, data, 0o755); err != nil {
		t.Fatalf("write tmux path helper: %v", err)
	}
	pid := int64(os.Getpid())
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)

	aplicado, gotPID, err := DetenerProceso(ObjetivoProceso{
		PID:          &pid,
		HandleKind:   "process",
		HandleRef:    "0",
		MetadataJSON: `{"driver":"tmux_cli_session","tmux_session":"orq-codex7-path"}`,
	})
	if err != nil {
		t.Fatalf("DetenerProceso tmux sin command: %v", err)
	}
	if !aplicado {
		t.Fatal("deberia detener la sesion tmux via PATH")
	}
	if gotPID != int(pid) {
		t.Fatalf("pid inesperado: got=%d want=%d", gotPID, pid)
	}
	invocations, err := os.ReadFile(filepath.Join(dir, "fake-tmux-state", "invocations.log"))
	if err != nil {
		t.Fatalf("leer invocations fake tmux: %v", err)
	}
	if !strings.Contains(string(invocations), "kill-session -t orq-codex7-path") {
		t.Fatalf("faltan kill-session via PATH en fake tmux: %s", string(invocations))
	}
}

func TestLatestTMUXWorktreeProgressMomentIgnoraCambiosPreviosAlArranque(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no disponible en el entorno")
	}

	dir := t.TempDir()
	cmd := exec.Command("git", "-C", dir, "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, string(out))
	}
	oldFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(oldFile, []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	startedAt := time.Now().UTC()
	oldMoment := startedAt.Add(-time.Hour)
	if err := os.Chtimes(oldFile, oldMoment, oldMoment); err != nil {
		t.Fatalf("chtimes old file: %v", err)
	}

	progressAt, err := latestTMUXWorktreeProgressMoment(dir, startedAt, time.Now().UTC())
	if err != nil {
		t.Fatalf("latestTMUXWorktreeProgressMoment: %v", err)
	}
	if !progressAt.IsZero() {
		t.Fatalf("no deberia contar dirty viejo como progreso del run actual: %s", progressAt)
	}
}

func TestLatestTMUXWorktreeProgressMomentCuentaCambiosNuevosTrasArranque(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no disponible en el entorno")
	}

	dir := t.TempDir()
	cmd := exec.Command("git", "-C", dir, "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, string(out))
	}
	startedAt := time.Now().UTC()
	time.Sleep(20 * time.Millisecond)
	newFile := filepath.Join(dir, "worker.go")
	if err := os.WriteFile(newFile, []byte("package worker\n"), 0o600); err != nil {
		t.Fatalf("write new file: %v", err)
	}
	info, err := os.Stat(newFile)
	if err != nil {
		t.Fatalf("stat new file: %v", err)
	}

	progressAt, err := latestTMUXWorktreeProgressMoment(dir, startedAt, time.Now().UTC())
	if err != nil {
		t.Fatalf("latestTMUXWorktreeProgressMoment: %v", err)
	}
	if progressAt.IsZero() {
		t.Fatal("deberia detectar progreso nuevo tras el arranque")
	}
	if progressAt.Before(startedAt) {
		t.Fatalf("progreso no deberia caer antes del arranque: %s < %s", progressAt, startedAt)
	}
	if progressAt.Before(info.ModTime().UTC()) {
		t.Fatalf("progreso deberia reflejar el mtime real del fichero: got=%s want>=%s", progressAt, info.ModTime().UTC())
	}
}

func TestLatestTMUXWorktreeProgressMomentIgnoraRepoRaizDelSistema(t *testing.T) {
	dir := t.TempDir()
	gitPath := filepath.Join(dir, "git")
	statePath := filepath.Join(dir, "git-state")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
printf '%%s\n' "$*" >> %q
if [[ "${*: -2}" == "rev-parse --show-toplevel" ]]; then
  printf '/\n'
  exit 0
fi
printf 'git status no deberia ejecutarse\n' >&2
exit 91
`, statePath)
	if err := os.WriteFile(gitPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	prev := tmuxMonitorGitCommand
	tmuxMonitorGitCommand = gitPath
	t.Cleanup(func() { tmuxMonitorGitCommand = prev })

	progressAt, err := latestTMUXWorktreeProgressMoment("/tmp/orquesta-sin-repo", time.Now().UTC(), time.Now().UTC())
	if err != nil {
		t.Fatalf("latestTMUXWorktreeProgressMoment: %v", err)
	}
	if !progressAt.IsZero() {
		t.Fatalf("no deberia reportar progreso para repo raiz del sistema: %s", progressAt)
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("leer estado fake git: %v", err)
	}
	log := string(data)
	if strings.Contains(log, "status --porcelain --untracked-files=all") {
		t.Fatalf("no deberia invocar git status cuando rev-parse devuelve '/': %s", log)
	}
	if !strings.Contains(log, "rev-parse --show-toplevel") {
		t.Fatalf("faltaba invocacion a rev-parse: %s", log)
	}
}

func writeFakeTmuxScript(t *testing.T, dir string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
state_dir=%q
mkdir -p "$state_dir"
printf '%%s\n' "$*" >> "$state_dir/invocations.log"
cmd="${1:-}"
shift || true
case "$cmd" in
  new-session)
    session=""
    window=""
    cwd=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        -s) session="$2"; shift 2 ;;
        -n) window="$2"; shift 2 ;;
        -c) cwd="$2"; shift 2 ;;
        -d|-P) shift ;;
        -F) shift 2 ;;
        *) break ;;
      esac
    done
    printf '%%s' "$cwd" > "$state_dir/cwd"
    printf '%%s' "$session" > "$state_dir/session"
    printf '%%s' "$window" > "$state_dir/window"
    printf '%%s\n' "${session}|${window}|%%1|12345"
    ;;
  pipe-pane)
    exit 0
    ;;
  send-keys)
    exit 0
    ;;
  display-message)
    cwd="$(cat "$state_dir/cwd" 2>/dev/null || pwd)"
    session="$(cat "$state_dir/session" 2>/dev/null || printf 'orq-test')"
    window="$(cat "$state_dir/window" 2>/dev/null || printf 'worker')"
    printf '%%s\n' "${session}|${window}|%%1|12345|0|bash|${cwd}"
    ;;
  capture-pane)
    printf '%%s\n' '> ready'
    ;;
  kill-session)
    exit 0
    ;;
  *)
    echo "unsupported fake tmux command: $cmd" >&2
    exit 1
    ;;
esac
`, filepath.Join(dir, "fake-tmux-state"))
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	return scriptPath
}

func writeFakeTmuxDispatchScript(t *testing.T, dir, mode string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux-dispatch")
	stateDir := filepath.Join(dir, "fake-tmux-dispatch-state")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
state_dir=%q
mode=%q
mkdir -p "$state_dir"
printf '%%s\n' "$*" >> "$state_dir/invocations.log"
cmd="${1:-}"
shift || true
pane_state_file="$state_dir/pane_state"
prompt_file="$state_dir/prompt"
if [[ ! -f "$pane_state_file" ]]; then
  if [[ "$mode" == "rate_limit_switch" ]]; then
    printf 'rate_limit_switch\n' > "$pane_state_file"
  else
    printf 'ready\n' > "$pane_state_file"
  fi
fi
case "$cmd" in
  display-message)
    printf 'orq-test|worker|%%77|12345|0|bash|%s\n' "$state_dir"
    ;;
  capture-pane)
    pane_state="$(cat "$pane_state_file" 2>/dev/null || printf 'ready')"
    prompt="$(cat "$prompt_file" 2>/dev/null || true)"
    case "$pane_state" in
      ready)
        printf '%%s\n' '> ready'
        ;;
      rate_limit_switch)
        printf '%%s\n' 'Approaching rate limits'
        printf '%%s\n' 'Switch to gpt-5.4-mini for lower credit usage?'
        printf '%%s\n' '› 1. Switch to gpt-5.4-mini'
        printf '%%s\n' '  2. Keep current model'
        ;;
      draft)
        printf '%%s\n' "> $prompt"
        ;;
      gemini_draft)
        printf '%%s\n' "> $prompt"
        printf '%%s\n' 'Shift+Tab to accept edits'
        ;;
      active)
        printf '%%s\n' 'Esc to interrupt'
        ;;
      *)
        printf '%%s\n' '> ready'
        ;;
    esac
    ;;
  send-keys)
    if [[ "${1:-}" == "-t" ]]; then
      shift 2
    fi
    if [[ "${1:-}" == "-l" ]]; then
      shift
      if [[ "$mode" == "ready_unchanged" ]]; then
        printf 'ready\n' > "$pane_state_file"
        : > "$prompt_file"
        exit 0
      fi
      if [[ "$mode" == "rate_limit_switch" ]]; then
        pane_state="$(cat "$pane_state_file" 2>/dev/null || printf 'ready')"
        printf '%%s' "$*" > "$prompt_file"
        if [[ "$pane_state" == "ready" ]]; then
          printf 'draft\n' > "$pane_state_file"
        fi
        exit 0
      fi
      printf '%%s' "$*" > "$prompt_file"
      if [[ "$mode" == "gemini_accept_edits" ]]; then
        printf 'gemini_draft\n' > "$pane_state_file"
      else
        printf 'draft\n' > "$pane_state_file"
      fi
      exit 0
    fi
    key="${1:-}"
    if [[ "$key" == "Enter" || "$key" == "C-m" ]]; then
      if [[ "$mode" == "active_after_enter" ]]; then
        printf 'active\n' > "$pane_state_file"
      elif [[ "$mode" == "rate_limit_switch" ]]; then
        pane_state="$(cat "$pane_state_file" 2>/dev/null || printf 'ready')"
        if [[ "$pane_state" == "rate_limit_switch" ]]; then
          printf 'ready\n' > "$pane_state_file"
        elif [[ "$pane_state" == "draft" ]]; then
          printf 'active\n' > "$pane_state_file"
        else
          printf 'draft\n' > "$pane_state_file"
        fi
      elif [[ "$mode" == "gemini_accept_edits" ]]; then
        pane_state="$(cat "$pane_state_file" 2>/dev/null || printf 'ready')"
        if [[ "$pane_state" == "gemini_draft" ]]; then
          printf 'gemini_draft\n' > "$pane_state_file"
        else
          printf 'active\n' > "$pane_state_file"
        fi
      elif [[ "$mode" == "ready_unchanged" ]]; then
        printf 'ready\n' > "$pane_state_file"
      else
        printf 'draft\n' > "$pane_state_file"
      fi
      exit 0
    fi
    if [[ "$mode" == "rate_limit_switch" && "$key" == "1" ]]; then
      printf 'rate_limit_switch\n' > "$pane_state_file"
      exit 0
    fi
    if [[ "$key" == "BTab" && "$mode" == "gemini_accept_edits" ]]; then
      printf 'ready\n' > "$pane_state_file"
      exit 0
    fi
    exit 0
    ;;
  *)
    echo "unsupported fake tmux command: $cmd" >&2
    exit 1
    ;;
esac
`, stateDir, mode, "%s")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux dispatch: %v", err)
	}
	return scriptPath
}
