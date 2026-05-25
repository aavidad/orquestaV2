package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexLaunchWaveCommandV0LanzaOlaConCodexFalso(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "auth.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("crear auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "config.toml"), []byte("model = \"test\"\n"), 0o600); err != nil {
		t.Fatalf("crear config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceCodeHome, "skills", "local"), 0o700); err != nil {
		t.Fatalf("crear skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "skills", "local", "SKILL.md"), []byte("# skill\n"), 0o600); err != nil {
		t.Fatalf("crear skill: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceCodeHome, "rules"), 0o700); err != nil {
		t.Fatalf("crear rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "rules", "default.rules"), []byte("rule\n"), 0o600); err != nil {
		t.Fatalf("crear rule: %v", err)
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
input=$(cat)
printf '%s\n' "$input" > "$HOME/prompt_seen.txt"
printf '%s\n' "$CODEX_HOME" > "$HOME/code_home_seen.txt"
if [ -n "$out" ]; then
  printf 'fake final\n' > "$out"
fi
printf 'fake stdout\n'
printf 'access_token=valor-real\n'
printf '/home/alberto/privado\n'
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
		"--agents", "2",
		"--wave-ref", "wave-test",
		"--isolate-home=true",
		"--prompt", "divide este trabajo en dos partes y ejecuta una parte concreta",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}

	var summary codexWaveLaunchSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v\n%s", err, stdout.String())
	}
	if summary.SchemaVersion != codexWaveSummarySchemaVersionV0 {
		t.Fatalf("schema=%q", summary.SchemaVersion)
	}
	if summary.WaveRef != "wave-test" || summary.AgentCount != 2 {
		t.Fatalf("summary inesperado: %+v", summary)
	}
	if summary.RegistryPath == "" {
		t.Fatalf("registry_path vacio: %+v", summary)
	}
	if _, err := os.Stat(summary.RegistryPath); err != nil {
		t.Fatalf("registry no escrito: %v", err)
	}
	if summary.Sandbox != "workspace-write" || summary.ApprovalPolicy != "never" {
		t.Fatalf("permisos inesperados sandbox=%q approval=%q", summary.Sandbox, summary.ApprovalPolicy)
	}
	if len(summary.Agents) != 2 {
		t.Fatalf("agents=%d", len(summary.Agents))
	}
	for i, agent := range summary.Agents {
		if agent.AgentRef == "" || agent.ProcessRef == "" || agent.WrapperPath == "" {
			t.Fatalf("agent incompleto: %+v", agent)
		}
		promptData, err := os.ReadFile(agent.PromptPath)
		if err != nil {
			t.Fatalf("leer prompt agente %d: %v", i, err)
		}
		promptText := string(promptData)
		if !strings.Contains(promptText, "agente: "+string(rune('1'+i))+" de 2") ||
			!strings.Contains(promptText, "divide este trabajo") {
			t.Fatalf("prompt agente %d no contiene identidad/instrucciones:\n%s", i, promptText)
		}
		wrapperData, err := os.ReadFile(agent.WrapperPath)
		if err != nil {
			t.Fatalf("leer wrapper agente %d: %v", i, err)
		}
		wrapperText := string(wrapperData)
		if !strings.Contains(wrapperText, "--sandbox 'workspace-write'") ||
			!strings.Contains(wrapperText, "--ask-for-approval 'never'") {
			t.Fatalf("wrapper no transporta permisos esperados:\n%s", wrapperText)
		}
		if _, err := os.Stat(filepath.Join(agent.CodeHomeDir, "auth.json")); err != nil {
			t.Fatalf("auth copiado agente %d: %v", i, err)
		}
		if _, err := os.Stat(filepath.Join(agent.CodeHomeDir, "skills", "local", "SKILL.md")); err != nil {
			t.Fatalf("skill copiada agente %d: %v", i, err)
		}
		if _, err := os.Stat(filepath.Join(agent.CodeHomeDir, "rules", "default.rules")); err != nil {
			t.Fatalf("rule copiada agente %d: %v", i, err)
		}
		waitForCodexWaveTestFileV0(t, agent.LastMessagePath)
		stdoutData, err := os.ReadFile(agent.StdoutPath)
		if err != nil {
			t.Fatalf("leer stdout agente %d: %v", i, err)
		}
		if !strings.Contains(string(stdoutData), "fake stdout") {
			t.Fatalf("stdout agente %d inesperado: %s", i, string(stdoutData))
		}
		seenPrompt, err := os.ReadFile(filepath.Join(agent.HomeDir, "prompt_seen.txt"))
		if err != nil {
			t.Fatalf("leer prompt visto agente %d: %v", i, err)
		}
		if !strings.Contains(string(seenPrompt), "divide este trabajo") {
			t.Fatalf("prompt visto agente %d inesperado: %s", i, string(seenPrompt))
		}
	}

	var statusOut bytes.Buffer
	stderr.Reset()
	exitCode = codexWaveStatusCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
	}, &statusOut, &stderr)
	if exitCode != 0 {
		t.Fatalf("status exit=%d stderr=%s", exitCode, stderr.String())
	}
	var statusSummary codexWaveLaunchSummaryV0
	if err := json.Unmarshal(statusOut.Bytes(), &statusSummary); err != nil {
		t.Fatalf("status json invalido: %v", err)
	}
	for _, agent := range statusSummary.Agents {
		if agent.Status != "stopped" {
			t.Fatalf("status agente=%s, want stopped: %+v", agent.Status, agent)
		}
		if agent.StdoutBytes == 0 || agent.LastMessageBytes == 0 {
			t.Fatalf("sizes no refrescados: %+v", agent)
		}
	}

	var tailOut bytes.Buffer
	stderr.Reset()
	exitCode = codexWaveTailCommandV0([]string{
		"--runtime-dir", summary.RuntimeWorkDir,
		"--reason", "diagnostico-test",
		"--file", "stdout",
		"--mode", "fragment",
		"--lines", "1",
	}, &tailOut, &stderr)
	if exitCode != 0 {
		t.Fatalf("tail exit=%d stderr=%s", exitCode, stderr.String())
	}
	var tailReport codexWaveTailReportV0
	if err := json.Unmarshal(tailOut.Bytes(), &tailReport); err != nil {
		t.Fatalf("tail json invalido: %v\n%s", err, tailOut.String())
	}
	if tailReport.Mode != "fragment" || len(tailReport.Agents) != 2 {
		t.Fatalf("tail report inesperado: %+v", tailReport)
	}
	if tailReport.Agents[0].AgentRef != "wave-test-agent-01" {
		t.Fatalf("agent_ref no enlazado: %+v", tailReport.Agents[0])
	}
	encoded := tailOut.String()
	if strings.Contains(encoded, "valor-real") || strings.Contains(encoded, "/home/alberto") {
		t.Fatalf("tail sin redactar: %s", encoded)
	}
}

func TestCodexLaunchWaveCommandV0DryRunNoExigeCodexReal(t *testing.T) {
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
	exitCode := codexLaunchWaveCommandV0([]string{
		"--dry-run",
		"--agents", "1",
		"--wave-ref", "wave-dry-run",
		"--project-dir", projectDir,
		"--runtime-dir", filepath.Join(root, "runtime", "wave-dry-run"),
		"--command", fakeCodex,
		"--source-code-home", sourceCodeHome,
		"--reasoning-effort", "medium",
		"--prompt", "solo materializa",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	var summary codexWaveLaunchSummaryV0
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("json invalido: %v", err)
	}
	if !summary.DryRun || len(summary.Agents) != 1 || summary.Agents[0].Status != "dry_run" {
		t.Fatalf("dry-run inesperado: %+v", summary)
	}
	if _, err := os.Stat(summary.Agents[0].WrapperPath); err != nil {
		t.Fatalf("wrapper no materializado: %v", err)
	}
	wrapperData, err := os.ReadFile(summary.Agents[0].WrapperPath)
	if err != nil {
		t.Fatalf("leer wrapper dry-run: %v", err)
	}
	if !strings.Contains(string(wrapperData), `model_reasoning_effort="medium"`) {
		t.Fatalf("wrapper no conserva reasoning medium:\n%s", string(wrapperData))
	}
}

func waitForCodexWaveTestFileV0(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("archivo no aparecio: %s", path)
}
