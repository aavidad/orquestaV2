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
		"--reasoning-effort", "medium",
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
	firstWrapper := mustReadFileStringV0(t, summary.Launch.Agents[0].WrapperPath)
	if strings.Contains(firstWrapper, `model_reasoning_effort="medium"`) ||
		!strings.Contains(firstWrapper, `model_reasoning_effort="high"`) {
		t.Fatalf("wrapper agente 1 no eleva reasoning a high:\n%s", firstWrapper)
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
		summary.Plan.WriteSet[0] != "." ||
		len(summary.Plan.RequiredTests) != 1 ||
		summary.Plan.RequiredTests[0] != "operator-validation-required" {
		t.Fatalf("summary minimo inesperado: %+v", summary)
	}
	if !summary.Launch.DryRun || len(summary.Launch.Agents) != 2 {
		t.Fatalf("launch minimo inesperado: %+v", summary.Launch)
	}
}

func TestCodexLaunchDirectorWaveCommandV0RecursiveDryRunMaterializaHijosYContextoDominio(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime", "domain-a1-wave")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")
	stalePath := filepath.Join(runtimeDir, "stale.txt")
	domainContextPath := filepath.Join(root, "domain-context.md")

	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.MkdirAll(sourceCodeHome, 0o700); err != nil {
		t.Fatalf("crear source codex home: %v", err)
	}
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		t.Fatalf("crear runtime dir: %v", err)
	}
	if err := os.WriteFile(stalePath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("crear stale: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	if err := os.WriteFile(domainContextPath, []byte(strings.TrimSpace(`
- field: editorial_global_policy_v1
- tono adulto sin infantilizar; pedagogia clara y no infantil
- A1 entre 20.250 y 22.500 palabras cuando aplique
- field: html_publication_policy_v1
- patron web tipo Tema 11, primera lectura activa por defecto y banco afiliados externo
- 50 preguntas, 4 opciones y distractores plausibles
- field: html_topic_template_v1
- renderer obligatorio: connector-scripts/render_topic_html_v1.py
- validador obligatorio: connector-scripts/validate_topic_package_v1.py
`)), 0o600); err != nil {
		t.Fatalf("crear contexto dominio: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--purge-runtime",
		"--agents", "2",
		"--allow-recursive-delegation",
		"--max-delegation-depth", "1",
		"--max-subagents-per-agent", "6",
		"--wave-ref", "domain-a1-wave",
		"--project-ref", "temarios",
		"--domain-ref", "temarios-a1",
		"--domain-context-file", domainContextPath,
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--source-code-home", sourceCodeHome,
		"--objective", "Crear dos temas A1 distintos con calidad editorial.",
		"--write-set", "external/temarios/a1/tema_001,external/temarios/a1/tema_002",
		"--required-tests", "validar A1 >=20250 palabras,validar politica editorial del dominio",
		"--branch-ref", "branch-domain-a1-wave",
		"--worktree-ref", "worktree-domain-a1-wave",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("runtime no purgado, stat err=%v", err)
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if summary.Plan.MaxParallelAgents != 2 ||
		!summary.Plan.RecursiveDelegation ||
		summary.Plan.MaxSubagentsPerAgent != 6 ||
		len(summary.Launch.Agents) != 2 ||
		len(summary.ChildLaunches) != 2 {
		t.Fatalf("summary recursivo inesperado: %+v", summary)
	}
	for _, child := range summary.ChildLaunches {
		if child.Launch.AgentCount != 6 ||
			len(child.Launch.Agents) != 6 ||
			!child.Launch.DryRun ||
			child.ParentAgentRef == "" {
			t.Fatalf("child launch inesperado: %+v", child)
		}
	}

	parentPrompt := mustReadFileStringV0(t, summary.Launch.Agents[0].PromptPath)
	for _, want := range []string{
		"Contexto de dominio inyectado por adaptador",
		"source_ref: domain-context.md",
		"editorial_global_policy_v1",
		"html_publication_policy_v1",
		"html_topic_template_v1",
		"20.250",
		"patron web tipo Tema 11",
		"50 preguntas",
		"connector-scripts/render_topic_html_v1.py",
		"connector-scripts/validate_topic_package_v1.py",
		"max_subagents_per_agent: 6",
		"No invoques spawn_agent",
		"modifica solo lo necesario",
		"No uses generadores para redactar contenido doctrinal final",
		"external/temarios/a1/tema_001",
	} {
		if !strings.Contains(parentPrompt, want) {
			t.Fatalf("prompt padre no contiene %q:\n%s", want, parentPrompt)
		}
	}

	childPrompt := mustReadFileStringV0(t, summary.ChildLaunches[0].Launch.Agents[0].PromptPath)
	for _, want := range []string{
		"parent_agent_ref: " + summary.Launch.Agents[0].AgentRef,
		"child_agent_index: 1 de 6",
		"editorial_global_policy_v1",
		"html_publication_policy_v1",
		"html_topic_template_v1",
		"sin infantilizar",
		"primera lectura activa por defecto",
		"50 preguntas",
		"connector-scripts/render_topic_html_v1.py",
		"connector-scripts/validate_topic_package_v1.py",
		"modifica solo lo necesario",
		"No uses generadores para redactar contenido doctrinal final",
		"external/temarios/a1/tema_001",
	} {
		if !strings.Contains(childPrompt, want) {
			t.Fatalf("prompt hijo no contiene %q:\n%s", want, childPrompt)
		}
	}

	secondChildPrompt := mustReadFileStringV0(t, summary.ChildLaunches[1].Launch.Agents[5].PromptPath)
	if !strings.Contains(secondChildPrompt, "external/temarios/a1/tema_002") ||
		!strings.Contains(secondChildPrompt, "revision pedagogica") {
		t.Fatalf("prompt hijo tema 2 incompleto:\n%s", secondChildPrompt)
	}
}

func TestCodexLaunchDirectorWaveCommandV0RecursiveDryRunMaterializaNietosConLimitesYReview(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime", "recursive-grandchildren")
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
		"--agents", "1",
		"--allow-recursive-delegation",
		"--max-delegation-depth", "2",
		"--max-subagents-per-agent", "2",
		"--wave-ref", "recursive-grandchildren",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective", "Probar recursion Codex offline con padre, hijos y nietos.",
		"--write-set", "cmd/orquesta-server/codex_director_wave_command_v0.go,cmd/orquesta-server/codex_director_wave_command_v0_test.go",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-recursive-grandchildren",
		"--worktree-ref", "worktree-recursive-grandchildren",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if !summary.Plan.RecursiveDelegation ||
		summary.Plan.MaxDelegationDepth != 2 ||
		summary.Plan.MaxSubagentsPerAgent != 2 ||
		len(summary.Launch.Agents) != 1 ||
		len(summary.ChildLaunches) != 1 {
		t.Fatalf("summary recursivo inesperado: %+v", summary)
	}

	child := summary.ChildLaunches[0]
	if child.ParentAgentRef != summary.Launch.Agents[0].AgentRef ||
		child.ParentWaveRef != summary.Launch.WaveRef ||
		child.DelegationDepth != 1 ||
		child.MaxDelegationDepth != 2 ||
		child.MaxSubagentsPerAgent != 2 ||
		child.SubtreeAgentBudget != 6 ||
		!child.ReviewRequiredBeforeClose ||
		len(child.Launch.Agents) != 2 ||
		len(child.ChildLaunches) != 2 {
		t.Fatalf("child summary inesperado: %+v", child)
	}

	grandchild := child.ChildLaunches[0]
	if grandchild.ParentAgentRef != child.Launch.Agents[0].AgentRef ||
		grandchild.ParentWaveRef != child.Launch.WaveRef ||
		grandchild.DelegationDepth != 2 ||
		grandchild.SubtreeAgentBudget != 2 ||
		!grandchild.ReviewRequiredBeforeClose ||
		len(grandchild.Launch.Agents) != 2 ||
		len(grandchild.ChildLaunches) != 0 {
		t.Fatalf("grandchild summary inesperado: %+v", grandchild)
	}

	grandchildPrompt := mustReadFileStringV0(t, grandchild.Launch.Agents[0].PromptPath)
	for _, want := range []string{
		"parent_agent_ref: " + child.Launch.Agents[0].AgentRef,
		"delegation_depth: 2",
		"max_delegation_depth: 2",
		"max_subagents_per_agent: 2",
		"No lances mas subagentes desde este hijo",
		"No declares cierre de tu subarbol: el Director debe revisar entregas, tests y evidencias causales antes de cerrar.",
	} {
		if !strings.Contains(grandchildPrompt, want) {
			t.Fatalf("prompt nieto no contiene %q:\n%s", want, grandchildPrompt)
		}
	}
}

func TestCodexLaunchDirectorWaveCommandV0NoInyectaDominioPorProjectRef(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime", "opes-ref-sin-contexto")
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
		"--agents", "1",
		"--wave-ref", "opes-ref-sin-contexto",
		"--project-ref", "opes",
		"--domain-ref", "opes",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective", "Crear un trabajo OPES sin contexto de adaptador.",
		"--write-set", "external/opes/a1/tema_001",
		"--required-tests", "validacion externa",
		"--branch-ref", "branch-opes-ref-sin-contexto",
		"--worktree-ref", "worktree-opes-ref-sin-contexto",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexDirectorWaveSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if len(summary.Launch.Agents) != 1 {
		t.Fatalf("agents=%d", len(summary.Launch.Agents))
	}
	prompt := mustReadFileStringV0(t, summary.Launch.Agents[0].PromptPath)
	for _, forbidden := range []string{
		"Contexto de dominio inyectado por adaptador",
		"opes_global_editorial_policy_2026_05_18",
		"opes_html_publication_policy_2026_05_19",
		"opes_html_topic_template_v1",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt contiene contexto no inyectado %q:\n%s", forbidden, prompt)
		}
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
