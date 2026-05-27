package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexWavePromptFileInputBoundsV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	fakeCodex := filepath.Join(root, "codex-fake")
	largePrompt := filepath.Join(root, "prompt.md")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	if err := os.WriteFile(largePrompt, bytes.Repeat([]byte("x"), int(codexWaveOperatorInputMaxBytesV0)+1), 0o600); err != nil {
		t.Fatalf("crear prompt grande: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchWaveCommandV0([]string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "wave-prompt-bounds",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--prompt-file", largePrompt,
	}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "prompt_file_too_large") || strings.Contains(stderr.String(), largePrompt) {
		t.Fatalf("error no redactado o codigo ausente: %s", stderr.String())
	}
}

func TestCodexWavePromptFileRejectsControlInputV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime", "codex-waves", "wave-control")
	fakeCodex := filepath.Join(root, "codex-fake")
	controlPrompt := filepath.Join(runtimeDir, "agent_prompt.txt")
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		t.Fatalf("crear runtime dir: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	if err := os.WriteFile(controlPrompt, []byte("no usar prompt previo"), 0o600); err != nil {
		t.Fatalf("crear prompt control: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchWaveCommandV0([]string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "wave-control",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--prompt-file", controlPrompt,
	}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "prompt_file_invalid") || strings.Contains(stderr.String(), controlPrompt) {
		t.Fatalf("control input no bloqueado/redactado: %s", stderr.String())
	}
}

func TestCodexDirectorWaveOperatorInputsPublicSummaryV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime", "director-wave-inputs")
	fakeCodex := filepath.Join(root, "codex-fake")
	objectiveFile := filepath.Join(root, "objective.md")
	domainContextFile := filepath.Join(root, "domain-context.md")
	objectiveText := "objetivo privado de operador para crear una ola acotada"
	domainText := "politica editorial externa sin publicar ruta local"
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	if err := os.WriteFile(objectiveFile, []byte(objectiveText), 0o600); err != nil {
		t.Fatalf("crear objective: %v", err)
	}
	if err := os.WriteFile(domainContextFile, []byte(domainText), 0o600); err != nil {
		t.Fatalf("crear contexto: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "director-wave-inputs",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective-file", objectiveFile,
		"--domain-context-file", domainContextFile,
		"--write-set", "cmd/orquesta-server",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-director-wave-inputs",
		"--worktree-ref", "worktree-director-wave-inputs",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	var public codexDirectorWavePublicSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &public); err != nil {
		t.Fatalf("summary publico invalido: %v\n%s", err, stdout.String())
	}
	encoded := stdout.String()
	for _, forbidden := range []string{objectiveFile, domainContextFile, objectiveText, domainText} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("summary publico filtra input local %q: %s", forbidden, encoded)
		}
	}
	if len(public.OperatorInputs) != 2 ||
		!codexWaveOperatorInputPublicHasKindV0(public.OperatorInputs, "objective") ||
		!codexWaveOperatorInputPublicHasKindV0(public.OperatorInputs, "domain_context") {
		t.Fatalf("operator inputs ausentes: %+v", public.OperatorInputs)
	}
	if !strings.Contains(encoded, "operator_local_file") || !strings.Contains(encoded, "sha256:") {
		t.Fatalf("summary no publica categorias/refs compactas: %s", encoded)
	}
}

func TestCodexDirectorWaveDomainContextRejectsLogInputV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime", "director-wave-log")
	fakeCodex := filepath.Join(root, "codex-fake")
	logFile := filepath.Join(root, "codex_stdout.log")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project dir: %v", err)
	}
	if err := os.WriteFile(fakeCodex, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	if err := os.WriteFile(logFile, []byte("stdout crudo"), 0o600); err != nil {
		t.Fatalf("crear log: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchDirectorWaveCommandV0([]string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "director-wave-log",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--objective", "validar contexto",
		"--domain-context-file", logFile,
	}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "domain_context_file_unreadable") || strings.Contains(stderr.String(), logFile) {
		t.Fatalf("error dominio no redactado/codigo ausente: %s", stderr.String())
	}
}

func codexWaveOperatorInputPublicHasKindV0(items []codexWaveOperatorInputReceiptV0, kind string) bool {
	for _, item := range items {
		if item.Kind == kind && item.SourceRef != "" && item.ContentRef != "" {
			return true
		}
	}
	return false
}
