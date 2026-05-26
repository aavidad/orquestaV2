package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexLaunchDirectorWaveCommandV0EjecucionRealUsaGuardasEstrictasPorDefecto(t *testing.T) {
	root := t.TempDir()
	projectDir, fakeCodex := codexDirectorStrictGuardFixtureV0(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--agents", "1",
		"--wave-ref", "director-wave-strict-default",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(root, "runtime", "director-wave-strict-default"),
		"--command", fakeCodex,
		"--objective", "Intentar ejecucion real sin rails concretos.",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	for _, code := range []string{"branch_ref_missing", "worktree_ref_missing", "write_set_missing", "required_tests_missing"} {
		if !codexDirectorHasIssueV0(summary.Issues, code) {
			t.Fatalf("falta issue %s: %+v", code, summary.Issues)
		}
	}
	if len(summary.Launch.Agents) != 0 {
		t.Fatalf("no debe lanzar sin rails concretos: %+v", summary.Launch)
	}
}

func TestCodexLaunchDirectorWaveCommandV0WriteSetGlobalRequiereOptInAuditado(t *testing.T) {
	root := t.TempDir()
	projectDir, fakeCodex := codexDirectorStrictGuardFixtureV0(t, root)
	runtimeDir := filepath.Join(root, "runtime", "director-wave-global-blocked")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--agents", "1",
		"--wave-ref", "director-wave-global-blocked",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective", "Intentar ejecucion real con write_set global.",
		"--write-set", ".",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-director-wave-global-blocked",
		"--worktree-ref", "worktree-director-wave-global-blocked",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if !codexDirectorHasIssueV0(summary.Issues, "global_write_set_requires_opt_in") {
		t.Fatalf("falta issue global write-set: %+v", summary.Issues)
	}
	if len(summary.Launch.Agents) != 0 {
		t.Fatalf("no debe lanzar sin opt-in: %+v", summary.Launch)
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "codex_wave_registry_v0.json")); !os.IsNotExist(err) {
		t.Fatalf("no debe escribir registry tras bloqueo, err=%v", err)
	}
}

func TestCodexLaunchDirectorWaveCommandV0TestsPlaceholderRequierenOptInAuditado(t *testing.T) {
	root := t.TempDir()
	projectDir, fakeCodex := codexDirectorStrictGuardFixtureV0(t, root)
	runtimeDir := filepath.Join(root, "runtime", "director-wave-placeholder-blocked")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--agents", "1",
		"--wave-ref", "director-wave-placeholder-blocked",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective", "Intentar ejecucion real con tests placeholder.",
		"--write-set", "cmd/orquesta-server",
		"--required-tests", "operator-validation-required",
		"--branch-ref", "branch-director-wave-placeholder-blocked",
		"--worktree-ref", "worktree-director-wave-placeholder-blocked",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if !codexDirectorHasIssueV0(summary.Issues, "placeholder_tests_require_opt_in") {
		t.Fatalf("falta issue placeholder tests: %+v", summary.Issues)
	}
	if len(summary.Launch.Agents) != 0 {
		t.Fatalf("no debe lanzar sin opt-in: %+v", summary.Launch)
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "codex_wave_registry_v0.json")); !os.IsNotExist(err) {
		t.Fatalf("no debe escribir registry tras bloqueo, err=%v", err)
	}
}

func TestCodexLaunchDirectorWaveCommandV0OptInSinAuditoriaNoLanza(t *testing.T) {
	root := t.TempDir()
	projectDir, fakeCodex := codexDirectorStrictGuardFixtureV0(t, root)
	runtimeDir := filepath.Join(root, "runtime", "director-wave-opt-in-without-audit")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--agents", "1",
		"--allow-global-write-set",
		"--allow-placeholder-tests",
		"--wave-ref", "director-wave-opt-in-without-audit",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective", "Intentar opt-in sin razon ni evidencia.",
		"--write-set", ".",
		"--required-tests", "operator-validation-required",
		"--branch-ref", "branch-director-wave-opt-in-without-audit",
		"--worktree-ref", "worktree-director-wave-opt-in-without-audit",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if !codexDirectorHasIssueV0(summary.Issues, "guard_override_audit_missing") {
		t.Fatalf("falta issue auditoria: %+v", summary.Issues)
	}
	if len(summary.Launch.Agents) != 0 {
		t.Fatalf("no debe lanzar sin auditoria: %+v", summary.Launch)
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "codex_wave_registry_v0.json")); !os.IsNotExist(err) {
		t.Fatalf("no debe escribir registry tras bloqueo, err=%v", err)
	}
}

func TestCodexLaunchDirectorWaveCommandV0OptInAuditadoMaterializaGuardasEnPrompt(t *testing.T) {
	root := t.TempDir()
	projectDir, fakeCodex := codexDirectorStrictGuardFixtureV0(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--strict-director-guards",
		"--allow-global-write-set",
		"--allow-placeholder-tests",
		"--guard-override-reason", "operador autoriza auditoria completa del repo",
		"--guard-override-evidence-ref", "evidence-ref-guard-override-001",
		"--agents", "1",
		"--wave-ref", "director-wave-opt-in",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(root, "runtime", "director-wave-opt-in"),
		"--command", fakeCodex,
		"--objective", "Dry-run auditado de opt-in global.",
		"--write-set", ".",
		"--required-tests", "operator-validation-required",
		"--branch-ref", "branch-director-wave-opt-in",
		"--worktree-ref", "worktree-director-wave-opt-in",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if !summary.GuardOptIn.AllowGlobalWriteSet ||
		!summary.GuardOptIn.AllowPlaceholderTests ||
		summary.GuardOptIn.Reason == "" ||
		len(summary.GuardOptIn.EvidenceRefs) != 1 {
		t.Fatalf("opt-in no auditado en summary: %+v", summary.GuardOptIn)
	}
	prompt := mustReadFileStringV0(t, summary.Launch.Agents[0].PromptPath)
	for _, want := range []string{
		"Write-set global autorizado",
		"Opt-in auditado de guardas",
		"No salgas del proyecto ni edites fuera del write-set global",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt no contiene %q:\n%s", want, prompt)
		}
	}
}

func TestCodexLaunchDirectorWaveCommandV0BloqueaLaunchRealNoGestionadoPorDefecto(t *testing.T) {
	root := t.TempDir()
	projectDir, fakeCodex := codexDirectorStrictGuardFixtureV0(t, root)
	runtimeDir := filepath.Join(root, "runtime", "director-wave-unmanaged-blocked")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--agents", "1",
		"--wave-ref", "director-wave-unmanaged-blocked",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective", "Intentar lanzamiento directo con rails validos pero fuera del servidor.",
		"--write-set", "cmd/orquesta-server",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-director-wave-unmanaged-blocked",
		"--worktree-ref", "worktree-director-wave-unmanaged-blocked",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if !codexDirectorHasIssueV0(summary.Issues, "unmanaged_launch_blocked") {
		t.Fatalf("falta issue unmanaged: %+v", summary.Issues)
	}
	if len(summary.Launch.Agents) != 0 {
		t.Fatalf("no debe lanzar agentes fuera de Orquesta: %+v", summary.Launch)
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "codex_wave_registry_v0.json")); !os.IsNotExist(err) {
		t.Fatalf("no debe escribir registry tras bloqueo, err=%v", err)
	}
}

func TestCodexLaunchDirectorWaveCommandV0PromptsDistinguenShardYWriteSetGlobal(t *testing.T) {
	root := t.TempDir()
	projectDir, fakeCodex := codexDirectorStrictGuardFixtureV0(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--strict-director-guards",
		"--allow-recursive-delegation",
		"--max-delegation-depth", "1",
		"--max-subagents-per-agent", "1",
		"--recursive-agent-budget", "2",
		"--agents", "1",
		"--wave-ref", "director-wave-recursive-strict",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(root, "runtime", "director-wave-recursive-strict"),
		"--command", fakeCodex,
		"--objective", "Verificar prompts estrictos con subarbol gobernado.",
		"--write-set", "cmd/orquesta-server,modulos/orquesta-app-codex-stack",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-director-wave-recursive-strict",
		"--worktree-ref", "worktree-director-wave-recursive-strict",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if len(summary.ChildLaunches) != 1 || len(summary.ChildLaunches[0].Launch.Agents) != 1 {
		t.Fatalf("child launches inesperados: %+v", summary.ChildLaunches)
	}
	for _, promptPath := range []string{
		summary.Launch.Agents[0].PromptPath,
		summary.ChildLaunches[0].Launch.Agents[0].PromptPath,
	} {
		prompt := mustReadFileStringV0(t, promptPath)
		for _, want := range []string{
			"alcance autorizado total es solo el write-set global",
			"No uses texto libre, criterios o hints para ampliar alcance",
			"No salgas del proyecto ni edites fuera del write-set global",
		} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("prompt %s no contiene %q:\n%s", promptPath, want, prompt)
			}
		}
	}
}

func codexDirectorStrictGuardFixtureV0(t *testing.T, root string) (string, string) {
	t.Helper()
	projectDir := filepath.Join(root, "project")
	fakeCodex := filepath.Join(root, "codex-fake")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	return projectDir, fakeCodex
}
