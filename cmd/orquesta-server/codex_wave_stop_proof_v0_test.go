package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexWaveStopCommandV0BloqueaRegistrySinProof(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeBase := filepath.Join(root, "runtime")
	waveRef := "wave-stop-untrusted"
	runtimeDir := filepath.Join(runtimeBase, "codex-waves", waveRef)
	agentDir := filepath.Join(runtimeDir, "agent-01")
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeBase)
	if err := os.MkdirAll(agentDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	summary := codexWaveLaunchSummaryV0{
		SchemaVersion:  codexWaveSummarySchemaVersionV0,
		WaveRef:        waveRef,
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		RegistryPath:   codexWaveRegistryPathV0(runtimeDir),
		Agents: []codexWaveAgentSummaryV0{{
			AgentRef:       waveRef + "-agent-01",
			RuntimeWorkDir: agentDir,
			ProcessRef:     waveRef + "-process-01",
			SessionRef:     waveRef + "-session-01",
			LaunchRef:      waveRef + "-launch-01",
			PID:            cmd.Process.Pid,
			StartedAt:      "2026-05-25T00:00:00Z",
			Status:         "running",
		}},
	}
	if err := codexWaveSaveRegistryV0(summary); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexWaveStopCommandV0([]string{
		"--runtime-dir", runtimeDir,
		"--confirm-stop", waveRef,
		"--reason", "test-untrusted",
		"--force",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("stop untrusted exit=0 stdout=%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "process_proof") {
		t.Fatalf("stdout filtro detalle interno: %s", stdout.String())
	}
	stopped := mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), runtimeDir)
	if len(stopped.Errors) != 1 || stopped.Errors[0].Code != "blocked_registry_untrusted" {
		t.Fatalf("error publico inesperado: %+v", stopped.Errors)
	}
	if !processAliveV0(cmd.Process.Pid) {
		t.Fatalf("proceso fue senalado pese a registry sin proof")
	}
}

func TestCodexWaveStopCommandV0CooperativoAntesDeForceV0(t *testing.T) {
	summary := launchCodexWaveStopTestAgentV0(t, "wave-stop-coop")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexWaveStopCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
		"--confirm-stop", summary.WaveRef,
		"--reason", "test-coop",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("coop stop exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	requestPath := filepath.Join(summary.Agents[0].RuntimeWorkDir, "orquesta_shutdown_request.json")
	if _, err := os.Stat(requestPath); err != nil {
		t.Fatalf("shutdown request ausente: %v", err)
	}
	if !processAliveV0(summary.Agents[0].PID) {
		t.Fatalf("coop stop no debe senalar proceso")
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = codexWaveStopCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
		"--confirm-stop", summary.WaveRef,
		"--reason", "test-force",
		"--force",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("force stop exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	waitForCodexWaveProcessStoppedForTestV0(t, summary.Agents[0].PID)
}

func waitForCodexWaveProcessStoppedForTestV0(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAliveV0(pid) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("proceso %d sigue vivo tras force stop", pid)
}

func launchCodexWaveStopTestAgentV0(t *testing.T, waveRef string) codexWaveLaunchSummaryV0 {
	t.Helper()
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	fakeCodex := filepath.Join(root, "codex-fake")
	for _, dir := range []string{projectDir, sourceCodeHome} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("crear dir: %v", err)
		}
	}
	fakeScript := "#!/bin/sh\ntrap 'exit 0' INT TERM\nwhile :; do /bin/sleep 1; done\n"
	if err := os.WriteFile(fakeCodex, []byte(fakeScript), 0o700); err != nil {
		t.Fatalf("crear codex falso: %v", err)
	}
	t.Setenv("ORQUESTA_CODEX_COMMAND", fakeCodex)
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir)
	t.Setenv("ORQUESTA_CODEX_WAVE_SOURCE_CODEX_HOME", sourceCodeHome)
	t.Setenv("ORQUESTA_CODEX_WAVE_PATH", os.Getenv("PATH"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := codexLaunchWaveCommandV0([]string{
		"--agents", "1",
		"--wave-ref", waveRef,
		"--allow-unmanaged-launch",
		"--unmanaged-launch-reason", "test de stop/proof de bajo nivel con runtime falso",
		"--confirm-unmanaged-launch", waveRef,
		"--isolate-home=true",
		"--prompt", "test stop",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("launch exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	return mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), runtimeDir)
}
