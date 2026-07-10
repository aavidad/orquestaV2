package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodexWaveProgressBudgetV0SolicitaReworkCuandoSoloHayDiagnostico(t *testing.T) {
	root := t.TempDir()
	summary := launchCodexWaveProgressBudgetFixtureV0(t, root, "wave-no-progress")
	agent := summary.Agents[0]
	waitForCodexWaveDiagnosticBytesV0(t, agent.StderrPath)
	time.Sleep(1100 * time.Millisecond)

	updated := codexWaveProgressBudgetStatusForTestV0(t, summary.RuntimeWorkDir)
	got := updated.Agents[0]
	if got.Status != "stop_requested" || got.ReworkRef == "" || got.ProgressReceipt == nil ||
		got.ProgressReceipt.Reason != codexWaveNoProgressReasonV0 {
		t.Fatalf("presupuesto no solicito rework durable: %+v", got)
	}
	requestPath := filepath.Join(got.RuntimeWorkDir, "orquesta_shutdown_request.json")
	if _, err := os.Stat(requestPath); err != nil {
		t.Fatalf("solicitud cooperativa ausente: %v", err)
	}
	codexWaveForceStopForProgressBudgetTestV0(t, updated)
}

func TestCodexWaveProgressBudgetV0ConservaOlaConMensajeFinal(t *testing.T) {
	root := t.TempDir()
	summary := launchCodexWaveProgressBudgetFixtureV0(t, root, "wave-with-progress")
	agent := summary.Agents[0]
	waitForCodexWaveDiagnosticBytesV0(t, agent.StderrPath)
	if err := os.WriteFile(agent.LastMessagePath, []byte("avance durable\n"), 0o600); err != nil {
		t.Fatalf("escribir mensaje final: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)

	updated := codexWaveProgressBudgetStatusForTestV0(t, summary.RuntimeWorkDir)
	got := updated.Agents[0]
	if got.Status != "running" || got.ProgressReceipt != nil || got.ReworkRef != "" {
		t.Fatalf("ola con progreso fue cortada: %+v", got)
	}
	codexWaveForceStopForProgressBudgetTestV0(t, updated)
}

func launchCodexWaveProgressBudgetFixtureV0(t *testing.T, root string, waveRef string) codexWaveLaunchSummaryV0 {
	t.Helper()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(projectDir, "runtime")
	fakeCodex := filepath.Join(root, "codex-fake")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	// The fake deliberately ignores the cooperative file: the test exercises
	// receipt/rework first and the normal force-stop path then owns cleanup.
	fakeScript := "#!/bin/sh\ntrap 'exit 0' INT TERM\nwhile :; do printf 'diagnostic\\n' >&2; sleep 0.01; done\n"
	if err := os.WriteFile(fakeCodex, []byte(fakeScript), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	t.Setenv("ORQUESTA_CODEX_COMMAND", fakeCodex)
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_WAVE_PATH", os.Getenv("PATH"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchWaveCommandV0([]string{
		"--agents", "1",
		"--wave-ref", waveRef,
		"--allow-unmanaged-launch",
		"--unmanaged-launch-reason", "focal progress budget test",
		"--confirm-unmanaged-launch", waveRef,
		"--no-progress-budget-seconds", "1",
		"--diagnostic-budget-bytes", "1",
		"--progress-write-set", "result.txt",
		"--prompt", "run controlled diagnostic fixture",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("launch exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	return mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), runtimeDir)
}

func codexWaveProgressBudgetStatusForTestV0(t *testing.T, runtimeDir string) codexWaveLaunchSummaryV0 {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := codexWaveStatusCommandV0([]string{"--runtime-dir", runtimeDir}, &stdout, &stderr); exitCode != 0 {
		t.Fatalf("status exit=%d stderr=%s", exitCode, stderr.String())
	}
	return mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), runtimeDir)
}

func codexWaveForceStopForProgressBudgetTestV0(t *testing.T, summary codexWaveLaunchSummaryV0) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := codexWaveStopCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
		"--confirm-stop", summary.WaveRef,
		"--force",
		"--reason", "test_cleanup",
	}, &stdout, &stderr); exitCode != 0 {
		t.Fatalf("force stop exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if !waitUntilProcessGroupDownV0(summary.Agents[0].PID, 2*time.Second) {
		t.Fatalf("proceso de fixture sigue vivo")
	}
}

func waitForCodexWaveDiagnosticBytesV0(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("diagnostico no producido: %s", path)
}
