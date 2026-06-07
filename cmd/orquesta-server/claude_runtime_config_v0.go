package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	envClaudeEnabledV0        = "ORQUESTA_CLAUDE_ENABLED"
	envClaudeProjectWorkDirV0 = "ORQUESTA_CLAUDE_PROJECT_WORKDIR"
	envClaudeRuntimeWorkDirV0 = "ORQUESTA_CLAUDE_RUNTIME_WORKDIR"
	envClaudeCommandV0        = "ORQUESTA_CLAUDE_COMMAND"
	envClaudeHomeV0           = "ORQUESTA_CLAUDE_HOME"
	envClaudePathV0           = "ORQUESTA_CLAUDE_PATH"
	envClaudeModelV0          = "ORQUESTA_CLAUDE_MODEL"
	envClaudePermissionModeV0 = "ORQUESTA_CLAUDE_PERMISSION_MODE"
	envClaudeOutputFormatV0   = "ORQUESTA_CLAUDE_OUTPUT_FORMAT"
	envClaudeEffortV0         = "ORQUESTA_CLAUDE_EFFORT"
	envClaudeExtraArgsV0      = "ORQUESTA_CLAUDE_EXTRA_ARGS"
)

func init() {
	serverEffectiveEnvRegistryV0[envClaudeEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "claude_runtime",
		Label:       "Claude activo",
		Description: "Activa el proveedor Claude opt-in para revisiones OPES.",
	}
	serverEffectiveEnvRegistryV0[envClaudeModelV0] = serverEnvSettingMetadataV0{
		Scope:       "claude_runtime",
		Label:       "Modelo Claude",
		Description: "Modelo Claude CLI usado por el adaptador de revision.",
	}
	serverEffectiveEnvRegistryV0[envClaudePermissionModeV0] = serverEnvSettingMetadataV0{
		Scope:       "claude_runtime",
		Label:       "Permisos Claude",
		Description: "Modo de permisos Claude CLI para revisiones opt-in.",
	}
}

func claudeRuntimeConfigV0(
	serverConfig orquestaserver.ConfigV0,
) orquestaappcodexstack.ClaudeRuntimeConfigV0 {
	if !boolEnvOrDefaultV0(envClaudeEnabledV0, false) {
		return orquestaappcodexstack.ClaudeRuntimeConfigV0{}
	}
	return orquestaappcodexstack.ClaudeRuntimeConfigV0{
		Enabled:        true,
		CommandPath:    claudeCommandPathV0(),
		ProjectWorkDir: absDirEnvOrDefaultV0(envClaudeProjectWorkDirV0, serverConfig.ProjectWorkDir),
		RuntimeWorkDir: absDirEnvOrDefaultV0(envClaudeRuntimeWorkDirV0, serverConfig.RuntimeWorkDir),
		HomeDir:        strings.TrimSpace(os.Getenv(envClaudeHomeV0)),
		PathEnv:        envOrDefaultV0(envClaudePathV0, os.Getenv("PATH")),
		Model:          strings.TrimSpace(os.Getenv(envClaudeModelV0)),
		PermissionMode: envOrDefaultV0(envClaudePermissionModeV0, "bypassPermissions"),
		OutputFormat:   envOrDefaultV0(envClaudeOutputFormatV0, "text"),
		Effort:         strings.TrimSpace(os.Getenv(envClaudeEffortV0)),
		ExtraArgs:      strings.Fields(os.Getenv(envClaudeExtraArgsV0)),
	}
}

func claudeCommandPathV0() string {
	raw := strings.TrimSpace(os.Getenv(envClaudeCommandV0))
	if raw == "" {
		raw = "claude"
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		return raw
	}
	return path
}
