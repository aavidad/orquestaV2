package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestOpenDaemonOutputFilesV0DefaultEsResumenRedactadoV0(t *testing.T) {
	stateDir := t.TempDir()
	stdout, stderr, err := openDaemonOutputFilesV0(orquestaserver.ConfigV0{
		StateDir: stateDir,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, _ = stdout.WriteString("HOME=/home/alberto token=abc\n")
	_, _ = stderr.WriteString("prompt crudo\n")
	_ = stdout.Close()
	_ = stderr.Close()

	body := mustReadDaemonLogForTestV0(t, filepath.Join(stateDir, "stdout.log"))
	if !strings.Contains(body, daemonLogSummarySchemaVersionV0) ||
		!strings.Contains(body, orquestaserver.DaemonLogModeSummaryRedactedV0) {
		t.Fatalf("summary=%s", body)
	}
	for _, forbidden := range []string{"/home/alberto", "token=abc", "prompt crudo"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("summary filtra %q: %s", forbidden, body)
		}
	}
}

func TestOpenDaemonOutputFilesV0RawLocalOptInV0(t *testing.T) {
	stateDir := t.TempDir()
	stdout, stderr, err := openDaemonOutputFilesV0(orquestaserver.ConfigV0{
		StateDir: stateDir,
		DaemonLogPolicy: orquestaserver.DaemonLogPolicyV0{
			LocalRawEnabled: true,
			LocalRawReason:  "local test",
		},
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, _ = stdout.WriteString("raw-provider-output\n")
	_, _ = stderr.WriteString("raw-provider-error\n")
	_ = stdout.Close()
	_ = stderr.Close()

	stdoutBody := mustReadDaemonLogForTestV0(t, filepath.Join(stateDir, "stdout.log"))
	stderrBody := mustReadDaemonLogForTestV0(t, filepath.Join(stateDir, "stderr.log"))
	if !strings.Contains(stdoutBody, "raw-provider-output") ||
		!strings.Contains(stderrBody, "raw-provider-error") {
		t.Fatalf("stdout=%s stderr=%s", stdoutBody, stderrBody)
	}
}

func TestWriteDaemonLogSummaryV0RotaArchivoGrandeV0(t *testing.T) {
	stateDir := t.TempDir()
	path := filepath.Join(stateDir, "stdout.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 64)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	policy := orquestaserver.NormalizeDaemonLogPolicyV0(orquestaserver.DaemonLogPolicyV0{
		MaxBytes:        32,
		MaxRotatedFiles: 1,
	})

	if err := writeDaemonLogSummaryV0(stateDir, "stdout.log", policy); err != nil {
		t.Fatalf("summary: %v", err)
	}

	body := mustReadDaemonLogForTestV0(t, path)
	matches, _ := filepath.Glob(path + ".*")
	if !strings.Contains(body, daemonLogSummarySchemaVersionV0) || len(matches) != 1 {
		t.Fatalf("body=%s rotated=%v", body, matches)
	}
}

func mustReadDaemonLogForTestV0(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
