package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestManualAgentOpsScriptsSyntaxAndNoServerModeV0(t *testing.T) {
	root := manualAgentOpsRepoRootV0(t)
	scripts := []string{
		"scripts/inicio_agente.sh",
		"scripts/cargar_agentes.sh",
		"scripts/terminator_agentes.sh",
		"scripts/agente_console.sh",
	}
	skipIfManualAgentOpsScriptsMissingV0(t, root, scripts)
	args := append([]string{"-n"}, scripts...)
	syntax := exec.Command("bash", args...)
	syntax.Dir = root
	if out, err := syntax.CombinedOutput(); err != nil {
		t.Fatalf("bash -n scripts manuales: %v %s", err, string(out))
	}

	fakeBin := t.TempDir()
	fakeCurl := filepath.Join(fakeBin, "curl")
	if err := os.WriteFile(fakeCurl, []byte("#!/usr/bin/env bash\nexit 7\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "scripts/inicio_agente.sh", "Codex1")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+fakeBin,
		"ORQUESTA_SERVER_URL=http://127.0.0.1:65534",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("inicio_agente sin servidor debe bloquear: %s", string(out))
	}
	text := string(out)
	if !strings.Contains(text, "manual_agent_ops_blocked") ||
		!strings.Contains(text, "servidor_no_disponible") {
		t.Fatalf("error publico recuperable inesperado: %s", text)
	}
}

func TestManualAgentOpsNoLegacyFleetSeedV0(t *testing.T) {
	root := manualAgentOpsRepoRootV0(t)
	skipIfManualAgentOpsScriptsMissingV0(t, root, []string{"scripts/cargar_agentes.sh"})
	plan := filepath.Join(t.TempDir(), "agentes.plan")
	content := "Codex1|programador|repo|tarea|alta|modulo|nota|codex|codex\n" +
		"Codex2|programador|repo|tarea|alta|modulo|nota|codex|codex\n"
	if err := os.WriteFile(plan, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "scripts/cargar_agentes.sh", "--plan", plan)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cargar_agentes dry-run: %v %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "manual_agent_ops_no_auto_fleet_seed") ||
		!strings.Contains(text, "manual_agent_ops_dry_run") {
		t.Fatalf("cargar_agentes debe ser dry-run sin --confirm: %s", text)
	}
}

func TestManualAgentOpsBlocksDirectRuntimeV0(t *testing.T) {
	root := manualAgentOpsRepoRootV0(t)
	skipIfManualAgentOpsScriptsMissingV0(t, root, []string{"scripts/agente_console.sh"})
	cmd := exec.Command("bash", "scripts/agente_console.sh", "Codex1", "--runtime-cmd", "codex")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("agente_console no debe aceptar runtime directo: %s", string(out))
	}
	text := string(out)
	if !strings.Contains(text, "manual_agent_ops_blocked") ||
		!strings.Contains(text, "runtime_directo_bloqueado") {
		t.Fatalf("error publico por runtime directo inesperado: %s", text)
	}
}

func manualAgentOpsRepoRootV0(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller no disponible")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func skipIfManualAgentOpsScriptsMissingV0(t *testing.T, root string, scripts []string) {
	t.Helper()
	for _, script := range scripts {
		if _, err := os.Stat(filepath.Join(root, script)); err != nil {
			t.Skipf("script manual fuera del write-set no disponible: %s", script)
		}
	}
}
