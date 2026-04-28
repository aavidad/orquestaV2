package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/coordinacion"
	"orquesta/db"
)

func TestBuildSupervisorSubagentLaunchWorktreeMetadataExponeContratoCanonico(t *testing.T) {
	worktree := &coordinacion.Worktree{
		ID:      41,
		Path:    "/tmp/orquesta/wt-41",
		Branch:  "codex/openclaw-slice",
		BaseRef: "main",
	}

	got := buildSupervisorSubagentLaunchWorktreeMetadata(worktree)

	if got["worktree_id"] != int64(41) || got["ruta_worktree"] != worktree.Path {
		t.Fatalf("metadata canonica sin identidad de worktree: %+v", got)
	}
	if got["branch_worktree"] != worktree.Branch || got["base_ref_worktree"] != worktree.BaseRef {
		t.Fatalf("metadata canonica sin branch/base_ref: %+v", got)
	}
	if got["workspace_strategy"] != "git_worktree" || got["transport_preference"] != "tmux" {
		t.Fatalf("metadata canonica sin contrato workspace/tmux: %+v", got)
	}
	if got["resume_capability"] != "session_resume" || got["resume_strategy"] != "resumen_y_payload" || got["execution_profile"] != "worktree_tmux_resume" {
		t.Fatalf("metadata canonica sin contrato resume: %+v", got)
	}
}

func TestBuildSupervisorSubagentLaunchContractEnvExponeHintsCanonicos(t *testing.T) {
	env := buildSupervisorSubagentLaunchContractEnv(map[string]any{
		"worktree_id":          int64(77),
		"ruta_worktree":        "/tmp/orquesta/wt-77",
		"branch_worktree":      "codex/77",
		"base_ref_worktree":    "main",
		"workspace_strategy":   "git_worktree",
		"transport_preference": "tmux",
		"resume_capability":    "session_resume",
		"resume_strategy":      "resumen_y_payload",
		"execution_profile":    "worktree_tmux_resume",
	})
	joined := strings.Join(env, "\n")

	for _, fragment := range []string{
		"ORQUESTA_SUBAGENT_WORKTREE=/tmp/orquesta/wt-77",
		"ORQUESTA_SUBAGENT_PROJECT_PATH=/tmp/orquesta/wt-77",
		"ORQUESTA_SUBAGENT_BRANCH=codex/77",
		"ORQUESTA_SUBAGENT_BASE_REF=main",
		"ORQUESTA_SUBAGENT_WORKTREE_ID=77",
		"ORQUESTA_SUBAGENT_WORKSPACE_STRATEGY=git_worktree",
		"ORQUESTA_SUBAGENT_TRANSPORT_PREFERENCE=tmux",
		"ORQUESTA_SUBAGENT_RESUME_CAPABILITY=session_resume",
		"ORQUESTA_SUBAGENT_RESUME_STRATEGY=resumen_y_payload",
		"ORQUESTA_SUBAGENT_EXECUTION_PROFILE=worktree_tmux_resume",
	} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("env sin %q:\n%s", fragment, joined)
		}
	}
}

func TestLaunchClaudeSubagentExternalPersisteContratoCanonicoWorktreeTmuxResume(t *testing.T) {
	withTempOrquestaDB(t, func() {
		repoDir := t.TempDir()
		runGitCmdTest(t, repoDir, "init", "-b", "main")
		runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
		runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
		if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# repo\n"), 0o644); err != nil {
			t.Fatalf("write readme: %v", err)
		}
		runGitCmdTest(t, repoDir, "add", ".")
		runGitCmdTest(t, repoDir, "commit", "-m", "init")

		insertTestProyecto(t, "orquestador", "orquestador", repoDir)
		storeDir := t.TempDir()
		t.Setenv("CLAWD_AGENT_STORE", storeDir)
		launcherPath := filepath.Join(t.TempDir(), "launcher-contract.sh")
		script := `#!/usr/bin/env bash
set -euo pipefail
id="agent-test-contract-001"
manifest="${ORQUESTA_SUBAGENT_STORE}/${id}.json"
output="${ORQUESTA_SUBAGENT_STORE}/${id}.md"
mkdir -p "${ORQUESTA_SUBAGENT_STORE}"
printf 'worktree_id=%s\nworkspace_strategy=%s\ntransport_preference=%s\nresume_capability=%s\nresume_strategy=%s\nexecution_profile=%s\n' \
  "${ORQUESTA_SUBAGENT_WORKTREE_ID}" \
  "${ORQUESTA_SUBAGENT_WORKSPACE_STRATEGY}" \
  "${ORQUESTA_SUBAGENT_TRANSPORT_PREFERENCE}" \
  "${ORQUESTA_SUBAGENT_RESUME_CAPABILITY}" \
  "${ORQUESTA_SUBAGENT_RESUME_STRATEGY}" \
  "${ORQUESTA_SUBAGENT_EXECUTION_PROFILE}" > "${output}"
cat > "${manifest}" <<JSON
{"agentId":"${id}","name":"${ORQUESTA_SUBAGENT_NAME}","description":"${ORQUESTA_SUBAGENT_DESCRIPTION}","subagentType":"${ORQUESTA_SUBAGENT_TYPE}","model":"${ORQUESTA_SUBAGENT_MODEL}","status":"running","outputFile":"${output}","manifestFile":"${manifest}","createdAt":"2026-04-28T12:00:00Z","startedAt":"2026-04-28T12:00:00Z"}
JSON
printf 'ok\n'
`
		if err := os.WriteFile(launcherPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write launcher: %v", err)
		}
		t.Setenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER", launcherPath)

		resultado, err := launchClaudeSubagentExternal(supervisorSubagentLaunchRequest{
			Supervisor:   "OpenClaw",
			Proyecto:     "orquestador",
			Name:         "OpenClaw-Resume-Contract-1",
			Description:  "valida contrato canónico",
			Prompt:       "comprueba worktree tmux resume",
			SubagentType: "general-purpose",
			Model:        "claude-opus-4-6",
		})
		if err != nil {
			t.Fatalf("launchClaudeSubagentExternal: %v", err)
		}
		if resultado == nil || resultado.Worktree == nil {
			t.Fatalf("resultado sin worktree: %+v", resultado)
		}
		if resultado.Worktree["execution_profile"] != "worktree_tmux_resume" {
			t.Fatalf("resultado sin execution_profile canónico: %+v", resultado.Worktree)
		}

		rawOutput, err := os.ReadFile(filepath.Join(storeDir, "agent-test-contract-001.md"))
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		texto := string(rawOutput)
		for _, fragment := range []string{
			"workspace_strategy=git_worktree",
			"transport_preference=tmux",
			"resume_capability=session_resume",
			"resume_strategy=resumen_y_payload",
			"execution_profile=worktree_tmux_resume",
		} {
			if !strings.Contains(texto, fragment) {
				t.Fatalf("launcher sin %q:\n%s", fragment, texto)
			}
		}

		items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			Limit:        10,
		})
		if err != nil {
			t.Fatalf("listar subagentes: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("subagentes inesperados: %+v", items)
		}
		var metadata map[string]any
		if err := json.Unmarshal([]byte(items[0].MetadataJSON), &metadata); err != nil {
			t.Fatalf("unmarshal metadata: %v", err)
		}
		if metadata["worktree_id"] == nil || metadata["ruta_worktree"] == "" {
			t.Fatalf("metadata sin aliases canónicos de worktree: %+v", metadata)
		}
		if metadata["transport_preference"] != "tmux" || metadata["resume_strategy"] != "resumen_y_payload" {
			t.Fatalf("metadata sin contrato de resume/tmux: %+v", metadata)
		}
		preferencia := preferenciaEntregaGitSubagente(items[0])
		if preferencia.PreferenciaRutaWorktree == "" || preferencia.PreferenciaBranch == "" || preferencia.PreferenciaBaseRef == "" {
			t.Fatalf("preferencia git no derivada desde metadata canónica: %+v", preferencia)
		}
	})
}
