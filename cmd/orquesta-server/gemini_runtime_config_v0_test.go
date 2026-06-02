package main

import (
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

	config := geminiRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: filepath.Join(root, "fallback-project"),
		RuntimeWorkDir: filepath.Join(root, "fallback-runtime"),
	})
	if !config.Enabled ||
		config.CommandPath != fakeGemini ||
		config.ProjectWorkDir != projectDir ||
		config.RuntimeWorkDir != runtimeDir ||
		config.Model != "gemini-2.5-pro" ||
		config.ApprovalMode != "auto_edit" ||
		config.OutputFormat != "text" ||
		len(config.ExtraArgs) != 1 ||
		config.ExtraArgs[0] != "--skip-trust" {
		t.Fatalf("config=%+v", config)
	}
}
