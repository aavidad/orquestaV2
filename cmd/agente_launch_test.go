package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveAgenteLaunchMode(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("tabs", false, "")
	cmd.Flags().Bool("ventanas", false, "")

	mode, err := resolveAgenteLaunchMode(cmd)
	if err != nil {
		t.Fatalf("resolveAgenteLaunchMode default: %v", err)
	}
	if mode != "windows" {
		t.Fatalf("mode default = %q, want windows", mode)
	}
}

func TestResolveAgenteLaunchModeConflicto(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("tabs", false, "")
	cmd.Flags().Bool("ventanas", false, "")
	cmd.Flags().Set("tabs", "true")
	cmd.Flags().Set("ventanas", "true")

	if _, err := resolveAgenteLaunchMode(cmd); err == nil {
		t.Fatalf("se esperaba error por conflicto tabs/ventanas")
	}
}

func TestBuildAgenteLanzarPlanSpecUsaScriptsDir(t *testing.T) {
	tmp := t.TempDir()
	scriptsDir := filepath.Join(tmp, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "launch_agentes.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	planPath := filepath.Join(tmp, "plan.plan")
	if err := os.WriteFile(planPath, []byte("Codex1|programador|/tmp/proj|Titulo|alta|mod|nota|codex-cli|codex\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	defer cambiarEnv(t, "ORQUESTA_SCRIPTS_DIR", scriptsDir)()
	spec, err := buildAgenteLanzarPlanSpec(planPath, "tmux", "", "", "sesion-demo", false, "tabs", true)
	if err != nil {
		t.Fatalf("buildAgenteLanzarPlanSpec: %v", err)
	}
	if spec.ScriptPath != scriptPath {
		t.Fatalf("scriptPath = %q, want %q", spec.ScriptPath, scriptPath)
	}
	if len(spec.Args) != 3 || spec.Args[0] != "--ejecutar" || spec.Args[1] != "--tabs" {
		t.Fatalf("args inesperados: %+v", spec.Args)
	}
	if spec.Env["ORQUESTA_TERMINAL_BACKEND"] != "tmux" {
		t.Fatalf("backend env inesperado: %+v", spec.Env)
	}
	if spec.Env["ORQUESTA_TMUX_SESSION_NAME"] != "sesion-demo" {
		t.Fatalf("tmux session env inesperado: %+v", spec.Env)
	}
	if spec.Env["ORQUESTA_TMUX_ATTACH"] != "0" {
		t.Fatalf("tmux attach env inesperado: %+v", spec.Env)
	}
}

func TestMergeAgenteLaunchEnvSobrescribe(t *testing.T) {
	out := mergeAgenteLaunchEnv([]string{"A=1", "B=2"}, map[string]string{"B": "3", "C": "4"})
	got := map[string]string{}
	for _, item := range out {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			t.Fatalf("env sin separador: %q", item)
		}
		got[key] = value
	}
	if got["A"] != "1" || got["B"] != "3" || got["C"] != "4" {
		t.Fatalf("env merge inesperado: %+v", got)
	}
}

func TestCommandNeedsDBAgenteLanzarPlanNo(t *testing.T) {
	if commandNeedsDB([]string{"agente", "lanzar-plan", "plan.plan"}) {
		t.Fatalf("agente lanzar-plan no deberia requerir DB")
	}
}

func TestAgenteLanzarPlanJSONSmoke(t *testing.T) {
	tmp := t.TempDir()
	scriptsDir := filepath.Join(tmp, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "launch_agentes.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	planPath := filepath.Join(tmp, "plan.plan")
	if err := os.WriteFile(planPath, []byte("Codex1|programador|/tmp/proj|Titulo|alta|mod|nota|codex-cli|codex\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	defer cambiarEnv(t, "ORQUESTA_SCRIPTS_DIR", scriptsDir)()

	out := capturarStdout(t, func() {
		if err := executeLocalArgs([]string{"agente", "lanzar-plan", planPath, "--backend", "tmux", "--tabs", "--json"}, os.Stdout, os.Stderr); err != nil {
			t.Fatalf("executeLocalArgs agente lanzar-plan: %v", err)
		}
	})

	var spec agenteLanzarPlanOutput
	if err := json.Unmarshal([]byte(out), &spec); err != nil {
		t.Fatalf("parse json: %v\n%s", err, out)
	}
	if spec.ScriptPath != scriptPath {
		t.Fatalf("scriptPath = %q, want %q", spec.ScriptPath, scriptPath)
	}
	if spec.Backend != "tmux" || spec.LaunchMode != "tabs" {
		t.Fatalf("spec inesperado: %+v", spec)
	}
}

func TestParseAgenteLanzarPlanEntries(t *testing.T) {
	tmp := t.TempDir()
	planPath := filepath.Join(tmp, "plan.plan")
	plan := strings.Join([]string{
		"# comentario",
		"",
		"Codex1|programador|orquestador|Titulo|alta|mod|nota|codex-cli|codex --x",
		"Codex2|programador|orquestador|Titulo 2|media|mod|nota||",
	}, "\n")
	if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	entries, err := parseAgenteLanzarPlanEntries(planPath)
	if err != nil {
		t.Fatalf("parseAgenteLanzarPlanEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	if entries[0].Agente != "Codex1" || entries[0].Proyecto != "orquestador" || entries[0].Conector != "codex-cli" {
		t.Fatalf("entry 0 inesperada: %+v", entries[0])
	}
	if entries[1].Conector != "codex-cli" {
		t.Fatalf("connector default inesperado: %+v", entries[1])
	}
}

func TestAgenteLanzarPlanServerFirstPreviewNoRequiereScripts(t *testing.T) {
	tmp := t.TempDir()
	planPath := filepath.Join(tmp, "plan.plan")
	if err := os.WriteFile(planPath, []byte("Codex1|programador|orquestador|Titulo|alta|mod|nota|codex-cli|codex\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := executeLocalArgs([]string{"agente", "lanzar-plan", planPath, "--server-first"}, os.Stdout, os.Stderr); err != nil {
			t.Fatalf("executeLocalArgs server-first preview: %v", err)
		}
	})
	if !strings.Contains(out, "Preview server-first") || !strings.Contains(out, "agente=Codex1") {
		t.Fatalf("salida preview inesperada: %s", out)
	}
}

func TestEnqueueAgenteLanzarPlanServerFirst(t *testing.T) {
	prev := createRuntimeOrderForLaunch
	defer func() { createRuntimeOrderForLaunch = prev }()
	type request struct {
		Agente   string
		Tipo     string
		Proyecto string
		Payload  string
	}
	var seen []request
	createRuntimeOrderForLaunch = func(agente, tipo, proyecto, payload string) (int64, bool, error) {
		seen = append(seen, request{
			Agente:   agente,
			Tipo:     tipo,
			Proyecto: proyecto,
			Payload:  payload,
		})
		return int64(len(seen)), true, nil
	}

	entries := []agenteLanzarPlanEntry{
		{Agente: "Codex1", Proyecto: "orquestador", Conector: "codex-cli"},
		{Agente: "Codex2", Proyecto: "orquestador", Conector: "codex-cli"},
	}
	ids, err := enqueueAgenteLanzarPlanServerFirst(entries)
	if err != nil {
		t.Fatalf("enqueueAgenteLanzarPlanServerFirst: %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids inesperados: %+v", ids)
	}
	if len(seen) != 2 {
		t.Fatalf("requests vistas inesperadas: %d", len(seen))
	}
	if got := seen[0].Agente; got != "Codex1" {
		t.Fatalf("agente request 0 inesperado: %v", got)
	}
	if got := seen[0].Tipo; got != "start" {
		t.Fatalf("tipo request 0 inesperado: %v", got)
	}
	if got := seen[0].Proyecto; got != "orquestador" {
		t.Fatalf("proyecto request 0 inesperado: %v", got)
	}
	if !strings.Contains(seen[0].Payload, `"conector":"codex-cli"`) {
		t.Fatalf("payload request 0 inesperado: %s", seen[0].Payload)
	}
}
