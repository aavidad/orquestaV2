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

func TestGeminiRuntimeConfigV0OptInDesdeConfigFileV0(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	configPath := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"gemini_runtime":{
			"enabled":true,
			"command_path":"/opt/gemini/bin/gemini",
			"project_work_dir":"` + projectDir + `",
			"runtime_work_dir":"` + runtimeDir + `",
			"home_dir":"/tmp/gemini-home",
			"path":"/opt/gemini/bin",
			"model":"gemini-config-model",
			"approval_mode":"auto_edit",
			"output_format":"json",
			"extra_args":["--config-arg"]
		}
	}`
	if err := os.WriteFile(configPath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config := geminiRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        filepath.Join(root, "fallback-project"),
		RuntimeWorkDir:        filepath.Join(root, "fallback-runtime"),
		ProjectConfigFilePath: configPath,
	})
	if !config.Enabled ||
		config.CommandPath != "/opt/gemini/bin/gemini" ||
		config.ProjectWorkDir != projectDir ||
		config.RuntimeWorkDir != runtimeDir ||
		config.HomeDir != "/tmp/gemini-home" ||
		config.PathEnv != "/opt/gemini/bin" ||
		config.Model != "gemini-config-model" ||
		config.ApprovalMode != "auto_edit" ||
		config.OutputFormat != "json" ||
		len(config.ExtraArgs) != 1 ||
		config.ExtraArgs[0] != "--config-arg" {
		t.Fatalf("config=%+v", config)
	}
	if got := geminiGoalRuntimeWorkDirFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        projectDir,
		RuntimeWorkDir:        filepath.Join(root, "fallback-runtime"),
		ProjectConfigFilePath: configPath,
	}); got != runtimeDir {
		t.Fatalf("runtime goal=%q, want %q", got, runtimeDir)
	}
}

func TestGeminiRuntimeConfigV0EnvPrevaleceSobreConfigFileV0(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, serverProjectConfigFileNameV0)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"gemini_runtime":{"enabled":true,"model":"gemini-config-model","extra_args":["--config-arg"]}
	}`
	if err := os.WriteFile(configPath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(envGeminiModelV0, "gemini-env-model")
	t.Setenv(envGeminiExtraArgsV0, "--env-arg --second")

	config := geminiRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        root,
		RuntimeWorkDir:        filepath.Join(root, "runtime"),
		ProjectConfigFilePath: configPath,
	})
	if config.Model != "gemini-env-model" ||
		len(config.ExtraArgs) != 2 ||
		config.ExtraArgs[0] != "--env-arg" ||
		config.ExtraArgs[1] != "--second" {
		t.Fatalf("config=%+v", config)
	}

	t.Setenv(envGeminiEnabledV0, "false")
	if config := geminiRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        root,
		RuntimeWorkDir:        filepath.Join(root, "runtime"),
		ProjectConfigFilePath: configPath,
	}); config.Enabled {
		t.Fatalf("enabled por config pese a override explicito false: %+v", config)
	}
}
