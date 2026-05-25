package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
while :; do /bin/sleep 1; done
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
		"--isolate-home=true",
		"--prompt", "quedate vivo hasta que te pare el operador",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("launch exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	var summary codexWaveLaunchSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("launch json invalido: %v", err)
	}
	if len(summary.Agents) != 1 || summary.Agents[0].PID <= 0 {
		t.Fatalf("launch sin pid: %+v", summary)
	}
	waitForCodexWaveTestFileV0(t, summary.Agents[0].LastMessagePath)

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

	stdout.Reset()
	stderr.Reset()
	exitCode = codexWaveStatusCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("status exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	var stopped codexWaveLaunchSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &stopped); err != nil {
		t.Fatalf("status json invalido: %v", err)
	}
	if stopped.Agents[0].Status != "stopped" {
		t.Fatalf("status final=%q, want stopped: %+v", stopped.Agents[0].Status, stopped.Agents[0])
	}
	if stopped.Agents[0].StopRequestedAt == "" {
		t.Fatalf("stop_requested_at vacio: %+v", stopped.Agents[0])
	}
}
