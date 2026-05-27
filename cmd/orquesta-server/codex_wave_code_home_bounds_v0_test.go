package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexWaveCodeHomeProjectionV0AplicaSymlinkHardlinkModosYReceipt(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")

	mustMkdirV0(t, projectDir)
	mustMkdirV0(t, filepath.Join(sourceCodeHome, "skills", "local"))
	mustWriteV0(t, fakeCodex, "#!/bin/sh\nexit 0\n", 0o700)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "auth.json"), `{"ok":true}`, 0o700)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "config.toml"), "model='test'\n", 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "skills", "local", "SKILL.md"), "# skill\n", 0o755)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "skills", "local", "too-large.md"), "012345678901234567890123456789", 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "skills", "local", "hardlink-a.md"), "hardlink\n", 0o600)
	if err := os.Link(
		filepath.Join(sourceCodeHome, "skills", "local", "hardlink-a.md"),
		filepath.Join(sourceCodeHome, "skills", "local", "hardlink-b.md"),
	); err != nil {
		t.Fatalf("crear hardlink: %v", err)
	}
	if err := os.Symlink("../../auth.json", filepath.Join(sourceCodeHome, "skills", "local", "auth-link.json")); err != nil {
		t.Fatalf("crear symlink: %v", err)
	}

	summary := runCodexWaveProjectionForTestV0(t, projectDir, runtimeDir, sourceCodeHome, fakeCodex,
		"--projection-max-files", "20",
		"--projection-max-file-bytes", "20",
		"--projection-max-total-bytes", "200",
	)
	receipt := summary.Agents[0].CredentialProjection
	if receipt == nil {
		t.Fatalf("receipt ausente")
	}
	if receipt.FilesCopied != 3 || receipt.BytesCopied == 0 {
		t.Fatalf("contadores inesperados: %+v", receipt)
	}
	for _, want := range []string{"auth", "config", "skills"} {
		if !hasStringV0(receipt.Categories, want) {
			t.Fatalf("categoria copiada ausente %s: %+v", want, receipt)
		}
	}
	for _, want := range []string{"symlink", "hardlink", "max_file_bytes_exceeded"} {
		if !hasProjectionOmissionReasonForTestV0(receipt.Omissions, want) {
			t.Fatalf("omission %s ausente: %+v", want, receipt.Omissions)
		}
	}
	assertProjectedModeV0(t, filepath.Join(summary.Agents[0].CodeHomeDir, "auth.json"), 0o600)
	assertProjectedModeV0(t, filepath.Join(summary.Agents[0].CodeHomeDir, "skills", "local", "SKILL.md"), 0o600)
	if _, err := os.Lstat(filepath.Join(summary.Agents[0].CodeHomeDir, "skills", "local", "auth-link.json")); !os.IsNotExist(err) {
		t.Fatalf("symlink no debe copiarse: %v", err)
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal receipt: %v", err)
	}
	if bytes.Contains(raw, []byte(sourceCodeHome)) || bytes.Contains(raw, []byte("too-large")) {
		t.Fatalf("receipt filtra detalle local: %s", string(raw))
	}
}

func TestCodexWaveCodeHomeProjectionV0RespetaPresupuestoTotalYFicheros(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")

	mustMkdirV0(t, projectDir)
	mustMkdirV0(t, filepath.Join(sourceCodeHome, "skills", "local"))
	mustWriteV0(t, fakeCodex, "#!/bin/sh\nexit 0\n", 0o700)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "auth.json"), `{"ok":true}`, 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "config.toml"), "model='test'\n", 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "skills", "local", "extra.md"), "extra\n", 0o600)

	summary := runCodexWaveProjectionForTestV0(t, projectDir, runtimeDir, sourceCodeHome, fakeCodex,
		"--projection-max-files", "2",
		"--projection-max-file-bytes", "100",
		"--projection-max-total-bytes", "100",
	)
	receipt := summary.Agents[0].CredentialProjection
	if receipt == nil || receipt.FilesCopied != 2 {
		t.Fatalf("max_files no aplicado: %+v", receipt)
	}
	if !hasProjectionOmissionReasonForTestV0(receipt.Omissions, "max_files_exceeded") {
		t.Fatalf("omission max_files ausente: %+v", receipt)
	}
}

func runCodexWaveProjectionForTestV0(
	t *testing.T,
	projectDir string,
	runtimeDir string,
	sourceCodeHome string,
	fakeCodex string,
	extraArgs ...string,
) codexWaveLaunchSummaryV0 {
	t.Helper()
	args := []string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "wave-projection-bounds",
		"--project-dir", projectDir,
		"--runtime-dir", runtimeDir,
		"--command", fakeCodex,
		"--source-code-home", sourceCodeHome,
		"--isolate-home=true",
		"--strict-credential-projection=true",
		"--prompt", "materializa proyeccion acotada",
	}
	args = append(args, extraArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchWaveCommandV0(args, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	return mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), runtimeDir)
}

func hasProjectionOmissionReasonForTestV0(omissions []codexWaveCredentialProjectionOmissionV0, reason string) bool {
	for _, omission := range omissions {
		if omission.Reason == reason && omission.Count > 0 {
			return true
		}
	}
	return false
}

func assertProjectedModeV0(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("stat proyectado %s: %v", filepath.Base(path), err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("modo %s=%#o want %#o", filepath.Base(path), got, want)
	}
}
