package main

import (
	"os"
	"strings"
	"testing"
)

func TestCodexWaveConfigV0UsaSandboxWorkspaceWritePorDefecto(t *testing.T) {
	t.Setenv(envCodexWaveSandboxV0, "")

	config := mustCodexWaveConfigForTestV0(t, nil)

	if config.Sandbox != "workspace-write" {
		t.Fatalf("sandbox=%q want workspace-write", config.Sandbox)
	}
}

func TestCodexWaveConfigV0DangerFullAccessSoloOptInExplicito(t *testing.T) {
	config := mustCodexWaveConfigForTestV0(t, []string{"--sandbox", "danger-full-access"})

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestCodexWaveConfigV0ConservaReasoningHighYXHigh(t *testing.T) {
	for _, effort := range []string{"high", "xhigh"} {
		t.Run(effort, func(t *testing.T) {
			config := mustCodexWaveConfigForTestV0(t, []string{"--reasoning-effort", effort})

			if config.ReasoningEffort != effort {
				t.Fatalf("reasoning_effort=%q want %q", config.ReasoningEffort, effort)
			}
		})
	}
}

func mustCodexWaveConfigForTestV0(t *testing.T, extraArgs []string) codexWaveConfigV0 {
	t.Helper()
	args := []string{
		"--wave-ref", "wave-test",
		"--prompt", "probar config",
		"--project-dir", t.TempDir(),
		"--runtime-dir", t.TempDir(),
		"--command", codexWaveTestExecutablePathV0(t),
	}
	args = append(args, extraArgs...)
	var stderr strings.Builder
	config, err := codexWaveConfigFromArgsV0(args, &stderr)
	if err != nil {
		t.Fatalf("codexWaveConfigFromArgsV0: %v stderr=%s", err, stderr.String())
	}
	return config
}

func codexWaveTestExecutablePathV0(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return path
}
