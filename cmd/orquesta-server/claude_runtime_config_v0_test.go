package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestClaudeRuntimeConfigV0DesactivadoPorDefecto(t *testing.T) {
	config := claudeRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: filepath.Join(t.TempDir(), "runtime"),
	})
	if config.Enabled {
		t.Fatalf("Claude no debe activarse por defecto: %+v", config)
	}
}

func TestClaudeRuntimeConfigV0OptInDefaultPermitePersistirACK(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envClaudeEnabledV0, "1")
	t.Setenv(envClaudeCommandV0, filepath.Join(root, "claude"))

	config := claudeRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	})
	if config.PermissionMode != "bypassPermissions" {
		t.Fatalf("permission mode default=%q, want bypassPermissions", config.PermissionMode)
	}
}

func TestClaudeRuntimeConfigV0OptInDesdeEnv(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	fakeClaude := filepath.Join(root, "claude")
	t.Setenv(envClaudeEnabledV0, "1")
	t.Setenv(envClaudeCommandV0, fakeClaude)
	t.Setenv(envClaudeProjectWorkDirV0, projectDir)
	t.Setenv(envClaudeRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envClaudeModelV0, "sonnet")
	t.Setenv(envClaudePermissionModeV0, "dontAsk")
	t.Setenv(envClaudeOutputFormatV0, "text")
	t.Setenv(envClaudeEffortV0, "medium")
	t.Setenv(envClaudeExtraArgsV0, "--bare")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	configFile := `{"schema_version":"orquesta_config.v0","goal_backend":{"prompt_locale":"en-US"}}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config := claudeRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        filepath.Join(root, "fallback-project"),
		RuntimeWorkDir:        filepath.Join(root, "fallback-runtime"),
		ProjectConfigFilePath: filepath.Join(projectDir, serverProjectConfigFileNameV0),
	})
	if !config.Enabled ||
		config.CommandPath != fakeClaude ||
		config.ProjectWorkDir != projectDir ||
		config.RuntimeWorkDir != runtimeDir ||
		config.Model != "sonnet" ||
		config.PermissionMode != "dontAsk" ||
		config.OutputFormat != "text" ||
		config.Effort != "medium" ||
		config.PromptLocale != "en-US" ||
		len(config.ExtraArgs) != 1 ||
		config.ExtraArgs[0] != "--bare" {
		t.Fatalf("config=%+v", config)
	}
}
