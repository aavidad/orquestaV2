package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func TestCodexLaunchDirectorWaveCommandV0DryRunConstruyePlanYPromptsPorAgente(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")

	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.MkdirAll(sourceCodeHome, 0o700); err != nil {
		t.Fatalf("crear source codex home: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--agents", "3",
		"--wave-ref", "director-wave-test",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(runtimeDir, "director-wave-test"),
		"--command", fakeCodex,
		"--source-code-home", sourceCodeHome,
		"--objective", "Implementar el puente Director Operativo a ola Codex.",
		"--write-set", "cmd/orquesta-server/codex_director_wave_command_v0.go,cmd/orquesta-server/codex_wave_command_v0.go,cmd/orquesta-server/codex_director_wave_command_v0_test.go",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-director-wave-test",
		"--worktree-ref", "worktree-director-wave-test",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if summary.SchemaVersion != codexDirectorWaveSummarySchemaVersionV0 {
		t.Fatalf("schema=%q", summary.SchemaVersion)
	}
	if len(summary.Issues) > 0 {
		t.Fatalf("issues=%+v", summary.Issues)
	}
	if summary.Plan.PlanRef == "" ||
		summary.Plan.MaxParallelAgents != 3 ||
		!summary.Launch.DryRun ||
		summary.Launch.AgentCount != 3 {
		t.Fatalf("summary inesperado: %+v", summary)
	}
	if summary.WaveWork.PlanRef != summary.Plan.PlanRef ||
		!summary.WaveWork.ReadyToLaunch {
		t.Fatalf("wave work inesperado: %+v", summary.WaveWork)
	}
	if len(summary.Launch.Agents) != 3 {
		t.Fatalf("agents=%d", len(summary.Launch.Agents))
	}

	firstPrompt := mustReadFileStringV0(t, summary.Launch.Agents[0].PromptPath)
	if !strings.Contains(firstPrompt, "Director Operativo Orquesta aprobo esta ola") ||
		!strings.Contains(firstPrompt, "plan_ref: "+summary.Plan.PlanRef) ||
		!strings.Contains(firstPrompt, "agent_index: 1 de 3") ||
		!strings.Contains(firstPrompt, "cmd/orquesta-server/codex_director_wave_command_v0.go") ||
		!strings.Contains(firstPrompt, "go test -count=1 ./cmd/orquesta-server") {
		t.Fatalf("prompt agente 1 incompleto:\n%s", firstPrompt)
	}

	secondPrompt := mustReadFileStringV0(t, summary.Launch.Agents[1].PromptPath)
	if !strings.Contains(secondPrompt, "agent_index: 2 de 3") ||
		!strings.Contains(secondPrompt, "cmd/orquesta-server/codex_wave_command_v0.go") {
		t.Fatalf("prompt agente 2 no contiene shard esperado:\n%s", secondPrompt)
	}
}

func TestCodexLaunchDirectorWaveCommandV0RechazaRequestSinTests(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")

	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.MkdirAll(sourceCodeHome, 0o700); err != nil {
		t.Fatalf("crear source codex home: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--agents", "2",
		"--wave-ref", "director-wave-blocked",
		"--strict-director-guards",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(root, "runtime", "director-wave-blocked"),
		"--command", fakeCodex,
		"--source-code-home", sourceCodeHome,
		"--objective", "Intentar lanzar sin tests requeridos.",
		"--write-set", "cmd/orquesta-server/codex_director_wave_command_v0.go",
		"--branch-ref", "branch-director-wave-blocked",
		"--worktree-ref", "worktree-director-wave-blocked",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v", err)
	}
	if !codexDirectorHasIssueV0(summary.Issues, "required_tests_missing") {
		t.Fatalf("missing required_tests issue: %+v", summary.Issues)
	}
	if summary.Launch.AgentCount != 0 || len(summary.Launch.Agents) != 0 {
		t.Fatalf("no debio lanzar: %+v", summary.Launch)
	}
}

func TestCodexLaunchDirectorWaveCommandV0ModoMinimoRellenaRails(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	fakeCodex := filepath.Join(root, "codex-fake")

	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--agents", "2",
		"--wave-ref", "director-wave-minimal",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(runtimeDir, "director-wave-minimal"),
		"--command", fakeCodex,
		"--objective", "Arrancar agentes con el minimo de rails y observar.",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if len(summary.Issues) > 0 ||
		summary.Request.BranchRef == "" ||
		len(summary.Plan.WriteSet) != 1 ||
		summary.Plan.WriteSet[0] != "workspace" ||
		len(summary.Plan.RequiredTests) != 1 ||
		summary.Plan.RequiredTests[0] != "operator-validation-required" {
		t.Fatalf("summary minimo inesperado: %+v", summary)
	}
	if !summary.Launch.DryRun || len(summary.Launch.Agents) != 2 {
		t.Fatalf("launch minimo inesperado: %+v", summary.Launch)
	}
}

func mustReadFileStringV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	return string(data)
}

func codexDirectorHasIssueV0(
	issues []orquestadirectoroperativo.OperationalDirectorIssueV0,
	code string,
) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
