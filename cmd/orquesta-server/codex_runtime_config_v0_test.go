package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestMain(m *testing.M) {
	cleanup := configureServerPackageTestEnvV0()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func configureServerPackageTestEnvV0() func() {
	tmpParent := "."
	for _, candidate := range []string{"/workspace/runtime", "/var/tmp"} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			tmpParent = candidate
			break
		}
	}
	tmpDir, err := os.MkdirTemp(tmpParent, "orquesta-server-test-tmp-")
	if err == nil {
		if abs, absErr := filepath.Abs(tmpDir); absErr == nil {
			_ = os.Setenv("TMPDIR", abs)
			_ = os.Setenv("GOTMPDIR", abs)
			_ = os.Setenv("GOCACHE", filepath.Join(abs, "go-cache"))
			_ = os.Setenv("GOPATH", filepath.Join(abs, "go"))
		}
	}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "ORQUESTA_") || strings.HasPrefix(key, "OPES_") {
			_ = os.Unsetenv(key)
		}
	}
	_ = os.Setenv(envCodexSandboxV0, "danger-full-access")
	return func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	}
}

func TestCodexRuntimeConfigV0UsaSandboxWorkspaceWritePorDefecto(t *testing.T) {
	t.Setenv(envCodexSandboxV0, "")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "workspace-write" {
		t.Fatalf("sandbox=%q want workspace-write", config.Sandbox)
	}
}

func TestCodexRuntimeConfigV0DangerFullAccessSoloOptInExplicito(t *testing.T) {
	t.Setenv(envCodexSandboxV0, "danger-full-access")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestCodexRuntimeConfigV0ConservaReasoningHighYXHigh(t *testing.T) {
	for _, effort := range []string{"high", "xhigh"} {
		t.Run(effort, func(t *testing.T) {
			t.Setenv(envCodexReasoningEffortV0, effort)

			config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
				ProjectWorkDir: t.TempDir(),
				RuntimeWorkDir: t.TempDir(),
			}, nil)

			if config.ReasoningEffort != effort {
				t.Fatalf("reasoning_effort=%q want %q", config.ReasoningEffort, effort)
			}
		})
	}
}

func TestCodexStackCapacityConfigV0ConservaReasoningHighYXHigh(t *testing.T) {
	for _, effort := range []orquestacoreworkflow.OrchestrationCapacityRecommendationV0{
		orquestacoreworkflow.OrchestrationCapacityHighV0,
		orquestacoreworkflow.OrchestrationCapacityXHighV0,
	} {
		t.Run(string(effort), func(t *testing.T) {
			t.Setenv(envCapacityReasoningEffortV0, string(effort))

			config := codexStackCapacityEnvConfigFromEnvV0()

			if config.ReasoningEffort != effort {
				t.Fatalf("capacity_reasoning=%q want %q", config.ReasoningEffort, effort)
			}
		})
	}
}
