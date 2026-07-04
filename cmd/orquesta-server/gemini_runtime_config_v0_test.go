package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestGeminiRuntimeConfigV0DesactivadoPorDefecto(t *testing.T) {
	config := geminiRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: filepath.Join(t.TempDir(), "runtime"),
	})
	if config.Enabled {
		t.Fatalf("Gemini no debe activarse por defecto: %+v", config)
	}
}

func TestGeminiRuntimeConfigV0OptInDesdeEnv(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	fakeGemini := filepath.Join(root, "gemini")
	t.Setenv(envGeminiEnabledV0, "1")
	t.Setenv(envGeminiCommandV0, fakeGemini)
	t.Setenv(envGeminiProjectWorkDirV0, projectDir)
	t.Setenv(envGeminiRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envGeminiModelV0, "gemini-2.5-pro")
	t.Setenv(envGeminiApprovalModeV0, "auto_edit")
	t.Setenv(envGeminiOutputFormatV0, "text")
	t.Setenv(envGeminiExtraArgsV0, "--skip-trust")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	configFile := `{"schema_version":"orquesta_config.v0","goal_backend":{"prompt_locale":"en-US"}}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config := geminiRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        filepath.Join(root, "fallback-project"),
		RuntimeWorkDir:        filepath.Join(root, "fallback-runtime"),
		ProjectConfigFilePath: filepath.Join(projectDir, serverProjectConfigFileNameV0),
	})
	if !config.Enabled ||
		config.CommandPath != fakeGemini ||
		config.ProjectWorkDir != projectDir ||
		config.RuntimeWorkDir != runtimeDir ||
		config.Model != "gemini-2.5-pro" ||
		config.ApprovalMode != "auto_edit" ||
		config.OutputFormat != "text" ||
		config.PromptLocale != "en-US" ||
		len(config.ExtraArgs) != 1 ||
		config.ExtraArgs[0] != "--skip-trust" {
		t.Fatalf("config=%+v", config)
	}
}

func TestGeminiRuntimeConfigV0OptInDefaultUsaOutputText(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envGeminiEnabledV0, "1")
	t.Setenv(envGeminiCommandV0, filepath.Join(root, "gemini"))

	config := geminiRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	})
	if config.OutputFormat != "text" {
		t.Fatalf("output format default=%q, want text", config.OutputFormat)
	}
}
