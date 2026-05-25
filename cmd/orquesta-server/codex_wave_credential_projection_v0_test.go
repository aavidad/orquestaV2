package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexWaveCredentialProjectionV0DeclaraCategoriasSinValores(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")

	mustMkdirV0(t, projectDir)
	mustMkdirV0(t, filepath.Join(sourceCodeHome, "skills", "local"))
	mustMkdirV0(t, filepath.Join(sourceCodeHome, "memories"))
	mustWriteV0(t, fakeCodex, "#!/bin/sh\nexit 0\n", 0o700)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "auth.json"), `{"token":"secret-value"}`, 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "config.toml"), "model = \"x\"\n", 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "skills", "local", "SKILL.md"), "# skill\n", 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "memories", "notes.md"), "private memory\n", 0o600)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchWaveCommandV0([]string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "wave-projection",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--source-code-home", sourceCodeHome,
		"--isolate-home=true",
		"--strict-credential-projection=true",
		"--prompt", "materializa proyeccion",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexWaveLaunchSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v", err)
	}
	receipt := summary.Agents[0].CredentialProjection
	if receipt == nil {
		t.Fatalf("receipt ausente: %+v", summary.Agents[0])
	}
	for _, category := range []string{"auth", "config", "skills"} {
		if !hasStringV0(receipt.Categories, category) {
			t.Fatalf("categoria %s ausente: %+v", category, receipt)
		}
	}
	if !hasStringV0(receipt.OmittedCategories, "memories") {
		t.Fatalf("memories debe omitirse sin opt-in: %+v", receipt)
	}
	rawReceipt, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal receipt: %v", err)
	}
	for _, forbidden := range []string{sourceCodeHome, "secret-value", "private memory"} {
		if strings.Contains(string(rawReceipt), forbidden) {
			t.Fatalf("receipt filtra valor prohibido %q: %s", forbidden, string(rawReceipt))
		}
	}
	if _, err := os.Stat(filepath.Join(summary.Agents[0].CodeHomeDir, "memories", "notes.md")); !os.IsNotExist(err) {
		t.Fatalf("memories no debe proyectarse por defecto: %v", err)
	}
}

func TestCodexWaveCredentialProjectionV0StrictBloqueaAuthConfigFaltante(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")
	mustMkdirV0(t, projectDir)
	mustMkdirV0(t, sourceCodeHome)
	mustWriteV0(t, fakeCodex, "#!/bin/sh\nexit 0\n", 0o700)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchWaveCommandV0([]string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "wave-projection-missing",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(root, "runtime"),
		"--command", fakeCodex,
		"--source-code-home", sourceCodeHome,
		"--isolate-home=true",
		"--strict-credential-projection=true",
		"--prompt", "materializa proyeccion",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d want 1 stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	var summary codexWaveLaunchSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v", err)
	}
	if len(summary.Errors) != 1 || !strings.Contains(summary.Errors[0].Message, "credential_projection_missing_required") {
		t.Fatalf("error publico inesperado: %+v", summary.Errors)
	}
}

func mustMkdirV0(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteV0(t *testing.T, path string, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func hasStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
