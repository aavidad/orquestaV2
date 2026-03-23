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
