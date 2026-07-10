package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCodexWaveStopCommandV0SolicitaParadaDesdeFicheroDedicado(t *testing.T) {
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
	fakeScript := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift || break
done
cat > "$HOME/prompt_seen.txt"
if [ -n "$out" ]; then
  printf 'started\n' > "$out"
fi
trap 'printf stopped > "$HOME/stopped.txt"; exit 0' INT TERM
/bin/sleep 30 &
printf '%s\n' "$!" > "$HOME/child.pid"
wait "$!"
`
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
		"--wave-ref", "wave-stop-test",
		"--allow-unmanaged-launch",
		"--unmanaged-launch-reason", "test de stop de bajo nivel con runtime falso",
		"--confirm-unmanaged-launch", "wave-stop-test",
		"--isolate-home=true",
		"--prompt", "quedate vivo hasta que te pare el operador",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("launch exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	summary := mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), runtimeDir)
	if len(summary.Agents) != 1 || summary.Agents[0].PID <= 0 {
		t.Fatalf("launch sin pid: %+v", summary)
	}
	t.Cleanup(func() {
		if processAliveV0(summary.Agents[0].PID) {
			_ = signalProcessGroupV0(summary.Agents[0].PID)
		}
	})
	waitForCodexWaveTestFileV0(t, summary.Agents[0].LastMessagePath)
	childPID := mustReadCodexWavePIDForTestV0(t, filepath.Join(summary.Agents[0].HomeDir, "child.pid"))

	stdout.Reset()
	stderr.Reset()
	exitCode = codexWaveStatusCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("running status exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	running := mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), summary.RuntimeWorkDir)
	if running.Agents[0].Status != "running" {
		t.Fatalf("status inicial=%q, want running: %+v", running.Agents[0].Status, running.Agents[0])
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = codexWaveTailCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
		"--reason", "test-running-tail",
		"--file", "last-message",
	}, &stdout, &stderr)
	if exitCode != 0 || !strings.Contains(stdout.String(), `"status":"running"`) {
		t.Fatalf("running tail exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = codexWaveStopCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
		"--confirm-stop", summary.WaveRef,
		"--reason", "test-stop",
		"--force",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("stop exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	waitForCodexWaveTestFileV0(t, filepath.Join(summary.Agents[0].HomeDir, "stopped.txt"))
	waitForCodexWaveProcessStoppedForTestV0(t, summary.Agents[0].PID)

	stdout.Reset()
	stderr.Reset()
	exitCode = codexWaveStatusCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("status exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	stopped := mustReadCodexWaveCommandSummaryForTest(t, stdout.Bytes(), summary.RuntimeWorkDir)
	if stopped.Agents[0].Status != "stopped" {
		t.Fatalf("status final=%q, want stopped: %+v", stopped.Agents[0].Status, stopped.Agents[0])
	}
	if stopped.Agents[0].StopRequestedAt == "" {
		t.Fatalf("stop_requested_at vacio: %+v", stopped.Agents[0])
	}
	waitForCodexWaveProcessStoppedForTestV0(t, childPID)
}

func mustReadCodexWavePIDForTestV0(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var lastData []byte
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			time.Sleep(10 * time.Millisecond)
			continue
		}
		lastData = data
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil && pid > 0 {
			return pid
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("pid invalido %q err=%v", string(lastData), lastErr)
	return 0
}
