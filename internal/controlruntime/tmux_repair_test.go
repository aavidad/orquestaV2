package controlruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"orquesta/runtimeagente"
)

func TestEnsureTMUXMonitorFromMetadataJSONReattachesDeadMonitor(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeLiveFakeTmuxScript(t, dir)
	fakeCodexProfile := writeFakeCodexProfileCommand(t, dir)

	arranque, err := ArrancarPlan(SolicitudArranque{
		Agente:   "CodexTMUXRepair",
		Proyecto: "orquestador",
		Plan: &runtimeagente.LaunchPlan{
			Comando:         fakeCodexProfile,
			Args:            []string{"CodexTMUXRepair"},
			WorkingDir:      dir,
			PerfilOperativo: "persistente",
			Env: map[string]string{
				"ORQUESTA_TERMINAL_BACKEND": "tmux",
				"ORQUESTA_TMUX_BIN":         fakeTmux,
			},
		},
	})
	if err != nil {
		t.Fatalf("arrancar plan tmux: %v", err)
	}

	meta := map[string]any{}
	if err := json.Unmarshal([]byte(arranque.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata tmux: %v", err)
	}
	heartbeatPath, _ := meta["worker_heartbeat_path"].(string)
	if strings.TrimSpace(heartbeatPath) == "" {
		t.Fatalf("metadata sin worker_heartbeat_path: %+v", meta)
	}
	manifestPath, _ := meta["worker_manifest_path"].(string)
	if strings.TrimSpace(manifestPath) == "" {
		t.Fatalf("metadata sin worker_manifest_path: %+v", meta)
	}
	monitorPID := int(meta["tmux_monitor_pid"].(float64))
	if monitorPID <= 0 {
		t.Fatalf("metadata sin tmux_monitor_pid valido: %+v", meta)
	}
	if got := strings.TrimSpace(stringValueFromMetadata(meta, "profile_name")); got != "CodexTMUXRepair" {
		t.Fatalf("metadata sin profile_name inicial: %+v", meta)
	}
	if got := filepath.Base(strings.TrimSpace(stringValueFromMetadata(meta, "profile_status_wrapper"))); got != "codex-perfil" {
		t.Fatalf("metadata sin profile_status_wrapper inicial: %+v", meta)
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read worker manifest: %v", err)
	}
	var manifest runtimeagente.WorkerManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("unmarshal worker manifest: %v", err)
	}
	if manifest.Profile != "CodexTMUXRepair" {
		t.Fatalf("worker manifest sin profile reparable: %+v", manifest)
	}
	if filepath.Base(strings.TrimSpace(manifest.ProfileStatusWrapper)) != "codex-perfil" {
		t.Fatalf("worker manifest sin wrapper reparable: %+v", manifest)
	}

	beforeInfo := waitFileStat(t, heartbeatPath)
	if err := syscall.Kill(monitorPID, syscall.SIGKILL); err != nil {
		t.Fatalf("kill monitor tmux: %v", err)
	}
	meta["tmux_monitor_pid"] = 99999999
	meta["supervisor_owner_pid"] = 1
	delete(meta, "perfil_operativo")
	delete(meta, "execution_profile")
	delete(meta, "profile_name")
	delete(meta, "profile_status_wrapper")
	delete(meta, "can_send_input")
	metaData, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("remarshal metadata tmux: %v", err)
	}

	repaired, restarted, err := EnsureTMUXMonitorFromMetadataJSON(string(metaData))
	if err != nil {
		t.Fatalf("EnsureTMUXMonitorFromMetadataJSON: %v", err)
	}
	if !restarted {
		t.Fatal("deberia reenganchar un monitor tmux muerto")
	}

	repairedMeta := map[string]any{}
	if err := json.Unmarshal([]byte(repaired), &repairedMeta); err != nil {
		t.Fatalf("metadata reparada: %v", err)
	}
	newMonitorPID := int(repairedMeta["tmux_monitor_pid"].(float64))
	if newMonitorPID <= 0 || newMonitorPID == monitorPID {
		t.Fatalf("tmux_monitor_pid no actualizado: before=%d after=%d meta=%+v", monitorPID, newMonitorPID, repairedMeta)
	}
	if got := strings.TrimSpace(stringValueFromMetadata(repairedMeta, "perfil_operativo")); got != "persistente" {
		t.Fatalf("perfil_operativo reparado = %q, want persistente; meta=%+v", got, repairedMeta)
	}
	if got := strings.TrimSpace(stringValueFromMetadata(repairedMeta, "execution_profile")); got != "persistente" {
		t.Fatalf("execution_profile reparado = %q, want persistente; meta=%+v", got, repairedMeta)
	}
	if got := strings.TrimSpace(stringValueFromMetadata(repairedMeta, "profile_name")); got != "CodexTMUXRepair" {
		t.Fatalf("profile_name reparado = %q; meta=%+v", got, repairedMeta)
	}
	if got := filepath.Base(strings.TrimSpace(stringValueFromMetadata(repairedMeta, "profile_status_wrapper"))); got != "codex-perfil" {
		t.Fatalf("profile_status_wrapper reparado = %q; meta=%+v", got, repairedMeta)
	}
	if _, ok := repairedMeta["can_send_input"]; !ok {
		t.Fatalf("can_send_input reparado ausente: %+v", repairedMeta)
	}

	waitFileModAfter(t, heartbeatPath, beforeInfo.ModTime())
	alive, err := procesoVivoPID(newMonitorPID)
	if err != nil {
		t.Fatalf("procesoVivoPID monitor nuevo: %v", err)
	}
	if !alive {
		t.Fatalf("monitor tmux reenganchado no sigue vivo: pid=%d", newMonitorPID)
	}
}

func writeFakeCodexProfileCommand(t *testing.T, dir string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir fake codex-perfil: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake codex-perfil: %v", err)
	}
	return scriptPath
}

func TestCleanupTMUXSessionOnOwnerExitKillsSesionOllama(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeLiveFakeTmuxScript(t, dir)
	if err := cleanupTMUXSessionOnOwnerExit(embeddedTmuxMonitorSpec{
		TmuxCommand: fakeTmux,
		SessionName: "orq-ollama1-local",
		Command:     "ollama run qwen2.5-coder:14b",
	}); err != nil {
		t.Fatalf("cleanupTMUXSessionOnOwnerExit: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "fake-tmux-state-live", "invocations.log"))
	if err != nil {
		t.Fatalf("read invocations: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-ollama1-local") {
		t.Fatalf("faltaba kill-session para cleanup ollama: %s", string(data))
	}
}

func TestCleanupTMUXSessionOnOwnerExitNoMataSesionNoOllama(t *testing.T) {
	dir := t.TempDir()
	fakeTmux := writeLiveFakeTmuxScript(t, dir)
	if err := cleanupTMUXSessionOnOwnerExit(embeddedTmuxMonitorSpec{
		TmuxCommand: fakeTmux,
		SessionName: "orq-codex1-local",
		Command:     "codex resume 123",
	}); err != nil {
		t.Fatalf("cleanupTMUXSessionOnOwnerExit codex: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "fake-tmux-state-live", "invocations.log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read invocations: %v", err)
	}
	if strings.Contains(string(data), "kill-session -t orq-codex1-local") {
		t.Fatalf("no deberia matar sesion no-ollama: %s", string(data))
	}
}

func waitFileStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err == nil {
			return info
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("archivo no disponible a tiempo: %s", path)
	return nil
}

func waitFileModAfter(t *testing.T, path string, after time.Time) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		info, err := os.Stat(path)
		if err == nil && info.ModTime().After(after) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("heartbeat no se actualizo a tiempo: %s", path)
}

func writeLiveFakeTmuxScript(t *testing.T, dir string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux-live")
	stateDir := filepath.Join(dir, "fake-tmux-state-live")
	script := strings.Join([]string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"state_dir=" + strconv.Quote(stateDir),
		"mkdir -p \"$state_dir\"",
		"printf '%s\\n' \"$*\" >> \"$state_dir/invocations.log\"",
		"cmd=\"${1:-}\"",
		"shift || true",
		"case \"$cmd\" in",
		"  new-session)",
		"    session=\"\"",
		"    window=\"\"",
		"    cwd=\"\"",
		"    while [[ $# -gt 0 ]]; do",
		"      case \"$1\" in",
		"        -s) session=\"$2\"; shift 2 ;;",
		"        -n) window=\"$2\"; shift 2 ;;",
		"        -c) cwd=\"$2\"; shift 2 ;;",
		"        -d|-P) shift ;;",
		"        -F) shift 2 ;;",
		"        *) break ;;",
		"      esac",
		"    done",
		"    printf '%s' \"$cwd\" > \"$state_dir/cwd\"",
		"    printf '%s' \"$session\" > \"$state_dir/session\"",
		"    printf '%s' \"$window\" > \"$state_dir/window\"",
		"    printf '%s\\n' \"${session}|${window}|%1|12345\"",
		"    ;;",
		"  pipe-pane)",
		"    exit 0",
		"    ;;",
		"  send-keys)",
		"    exit 0",
		"    ;;",
		"  capture-pane)",
		"    capture_file=\"$state_dir/capture-pane.txt\"",
		"    if [[ -f \"$capture_file\" ]]; then",
		"      cat \"$capture_file\"",
		"    else",
		"      printf '%s\\n' \"> ready\"",
		"    fi",
		"    ;;",
		"  has-session)",
		"    exit 0",
		"    ;;",
		"  display-message)",
		"    cwd=\"$(cat \"$state_dir/cwd\" 2>/dev/null || pwd)\"",
		"    session=\"$(cat \"$state_dir/session\" 2>/dev/null || printf 'orq-test')\"",
		"    window=\"$(cat \"$state_dir/window\" 2>/dev/null || printf 'worker')\"",
		"    printf '%s\\n' \"${session}|${window}|%1|12345|0|codex|${cwd}\"",
		"    ;;",
		"  kill-session)",
		"    exit 0",
		"    ;;",
		"  *)",
		"    echo \"unsupported fake tmux command: $cmd\" >&2",
		"    exit 1",
		"    ;;",
		"esac",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux live: %v", err)
	}
	return scriptPath
}
