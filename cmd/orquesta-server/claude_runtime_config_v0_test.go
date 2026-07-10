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
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	if err := os.WriteFile(configPath, []byte(`{"schema_version":"orquesta_config.v0","claude_runtime":{"enabled":true,"command_path":"`+filepath.Join(root, "claude")+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	config := claudeRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        projectDir,
		RuntimeWorkDir:        filepath.Join(root, "runtime"),
		ProjectConfigFilePath: configPath,
	})
	if config.PermissionMode != "bypassPermissions" {
		t.Fatalf("permission mode default=%q, want bypassPermissions", config.PermissionMode)
	}
}

func TestClaudeRuntimeConfigV0OptInDesdeConfigCanonica(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	fakeClaude := filepath.Join(root, "claude")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	configFile := `{"schema_version":"orquesta_config.v0","goal_backend":{"prompt_locale":"en-US"},"claude_runtime":{"enabled":true,"command_path":"` + fakeClaude + `","project_work_dir":"` + projectDir + `","runtime_work_dir":"` + runtimeDir + `","permission_mode":"dontAsk","output_format":"text","extra_args":["--bare"]},"claude_model_routing":{"aliases":{"claude-model-ref-haiku-v0":"haiku-4.5","claude-model-ref-sonnet-v0":"sonnet-5","claude-model-ref-fable-v0":"fable-5"}}}`
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
		config.PermissionMode != "dontAsk" ||
		config.OutputFormat != "text" ||
		config.ModelRouting.ModelAlias[claudeModelRefSonnetV0] != "sonnet-5" ||
		config.PromptLocale != "en-US" ||
		len(config.ExtraArgs) != 1 ||
		config.ExtraArgs[0] != "--bare" {
		t.Fatalf("config=%+v", config)
	}
}
