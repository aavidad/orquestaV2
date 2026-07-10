package main

import (
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	envClaudeEnabledV0        = "claude_runtime.enabled"
	envClaudeProjectWorkDirV0 = "claude_runtime.project_work_dir"
	envClaudeRuntimeWorkDirV0 = "claude_runtime.runtime_work_dir"
	envClaudeCommandV0        = "claude_runtime.command_path"
	envClaudeHomeV0           = "claude_runtime.home_dir"
	envClaudePathV0           = "claude_runtime.path"
	envClaudePermissionModeV0 = "claude_runtime.permission_mode"
	envClaudeOutputFormatV0   = "claude_runtime.output_format"
	envClaudeExtraArgsV0      = "claude_runtime.extra_args"

	claudeGoalBackendFileControlV0 = "claude_file_control"
	claudeGoalBackendProcessV0     = "claude_process"
)

type serverProjectConfigClaudeRuntimeV0 struct {
	Enabled        *bool    `json:"enabled,omitempty"`
	CommandPath    *string  `json:"command_path,omitempty"`
	ProjectWorkDir *string  `json:"project_work_dir,omitempty"`
	RuntimeWorkDir *string  `json:"runtime_work_dir,omitempty"`
	HomeDir        *string  `json:"home_dir,omitempty"`
	Path           *string  `json:"path,omitempty"`
	PermissionMode *string  `json:"permission_mode,omitempty"`
	OutputFormat   *string  `json:"output_format,omitempty"`
	ExtraArgs      []string `json:"extra_args,omitempty"`
}

func claudeRuntimeConfigV0(serverConfig orquestaserver.ConfigV0) orquestaappcodexstack.ClaudeRuntimeConfigV0 {
	projectConfig := projectConfigFromServerConfigBestEffortV0(serverConfig)
	runtime := projectConfig.ClaudeRuntime
	if runtime.Enabled == nil || !*runtime.Enabled {
		return orquestaappcodexstack.ClaudeRuntimeConfigV0{}
	}
	return orquestaappcodexstack.ClaudeRuntimeConfigV0{
		Enabled:        true,
		CommandPath:    claudeCommandPathV0(projectConfig),
		ProjectWorkDir: claudeConfiguredAbsDirV0(runtime.ProjectWorkDir, serverConfig.ProjectWorkDir),
		RuntimeWorkDir: claudeConfiguredAbsDirV0(runtime.RuntimeWorkDir, serverConfig.RuntimeWorkDir),
		HomeDir:        claudeConfiguredStringV0(runtime.HomeDir, ""),
		PathEnv:        claudeConfiguredStringV0(runtime.Path, ""),
		PermissionMode: claudeConfiguredStringV0(runtime.PermissionMode, "bypassPermissions"),
		OutputFormat:   claudeConfiguredStringV0(runtime.OutputFormat, "text"),
		ModelRouting:   claudeModelRoutingFromProjectConfigFileV0(projectConfig),
		PromptLocale:   goalBackendPromptLocaleFromProjectConfigFileV0(projectConfig),
		ExtraArgs:      compactStringsV0(runtime.ExtraArgs),
	}
}

func claudeCommandPathV0(project serverProjectConfigFileV0) string {
	path := claudeConfiguredStringV0(project.ClaudeRuntime.CommandPath, "")
	if path == "" || !filepath.IsAbs(path) {
		return ""
	}
	return filepath.Clean(path)
}

func claudeConfiguredAbsDirV0(value *string, fallback string) string {
	configured := claudeConfiguredStringV0(value, fallback)
	if configured == "" || !filepath.IsAbs(configured) {
		return configured
	}
	return filepath.Clean(configured)
}

func claudeConfiguredStringV0(value *string, fallback string) string {
	if value == nil {
		return strings.TrimSpace(fallback)
	}
	return strings.TrimSpace(*value)
}

func claudeGoalBackendFromEnvV0() string {
	return claudeGoalBackendFromValueV0(codexGoalBackendFromEnvV0())
}

func claudeGoalBackendFromValueV0(backend string) string {
	backend = strings.TrimSpace(backend)
	if backend == claudeGoalBackendFileControlV0 || backend == claudeGoalBackendProcessV0 {
		return backend
	}
	return ""
}

func claudeGoalBackendOperationalFromEnvV0() bool {
	switch claudeGoalBackendFromEnvV0() {
	case claudeGoalBackendFileControlV0, claudeGoalBackendProcessV0:
		return true
	default:
		return false
	}
}
