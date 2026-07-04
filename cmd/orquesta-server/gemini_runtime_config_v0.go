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
	envGeminiEnabledV0        = "ORQUESTA_GEMINI_ENABLED"
	envGeminiProjectWorkDirV0 = "ORQUESTA_GEMINI_PROJECT_WORKDIR"
	envGeminiRuntimeWorkDirV0 = "ORQUESTA_GEMINI_RUNTIME_WORKDIR"
	envGeminiCommandV0        = "ORQUESTA_GEMINI_COMMAND"
	envGeminiHomeV0           = "ORQUESTA_GEMINI_HOME"
	envGeminiPathV0           = "ORQUESTA_GEMINI_PATH"
	envGeminiModelV0          = "ORQUESTA_GEMINI_MODEL"
	envGeminiApprovalModeV0   = "ORQUESTA_GEMINI_APPROVAL_MODE"
	envGeminiOutputFormatV0   = "ORQUESTA_GEMINI_OUTPUT_FORMAT"
	envGeminiExtraArgsV0      = "ORQUESTA_GEMINI_EXTRA_ARGS"

	geminiGoalBackendFileControlV0 = "gemini_file_control"
	geminiGoalBackendProcessV0     = "gemini_process"
)

func init() {
	serverEffectiveEnvRegistryV0[envGeminiEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "gemini_runtime",
		Label:       "Gemini activo",
		Description: "Activa el proveedor Gemini opt-in para trabajos visual_asset.",
	}
	serverEffectiveEnvRegistryV0[envGeminiModelV0] = serverEnvSettingMetadataV0{
		Scope:       "gemini_runtime",
		Label:       "Modelo Gemini",
		Description: "Modelo Gemini CLI usado por el adaptador visual.",
	}
	serverEffectiveEnvRegistryV0[envGeminiApprovalModeV0] = serverEnvSettingMetadataV0{
		Scope:       "gemini_runtime",
		Label:       "Approval Gemini",
		Description: "Modo de aprobacion Gemini CLI para trabajos visuales opt-in.",
	}
}

func geminiRuntimeConfigV0(
	serverConfig orquestaserver.ConfigV0,
) orquestaappcodexstack.GeminiRuntimeConfigV0 {
	if !boolEnvOrDefaultV0(envGeminiEnabledV0, false) {
		return orquestaappcodexstack.GeminiRuntimeConfigV0{}
	}
	projectConfig := projectConfigFromServerConfigBestEffortV0(serverConfig)
	return orquestaappcodexstack.GeminiRuntimeConfigV0{
		Enabled:        true,
		CommandPath:    geminiCommandPathV0(),
		ProjectWorkDir: absDirEnvOrDefaultV0(envGeminiProjectWorkDirV0, serverConfig.ProjectWorkDir),
		RuntimeWorkDir: absDirEnvOrDefaultV0(envGeminiRuntimeWorkDirV0, serverConfig.RuntimeWorkDir),
		HomeDir:        strings.TrimSpace(os.Getenv(envGeminiHomeV0)),
		PathEnv:        envOrDefaultV0(envGeminiPathV0, os.Getenv("PATH")),
		Model:          strings.TrimSpace(os.Getenv(envGeminiModelV0)),
		ApprovalMode:   envOrDefaultV0(envGeminiApprovalModeV0, "auto_edit"),
		OutputFormat:   envOrDefaultV0(envGeminiOutputFormatV0, "text"),
		PromptLocale:   goalBackendPromptLocaleFromProjectConfigFileV0(projectConfig),
		ExtraArgs:      strings.Fields(os.Getenv(envGeminiExtraArgsV0)),
	}
}

func geminiCommandPathV0() string {
	raw := strings.TrimSpace(os.Getenv(envGeminiCommandV0))
	if raw == "" {
		raw = "gemini"
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

func geminiGoalBackendFromEnvV0() string {
	return geminiGoalBackendFromValueV0(codexGoalBackendFromEnvV0())
}

func geminiGoalBackendFromValueV0(backend string) string {
	backend = strings.TrimSpace(backend)
	if backend == geminiGoalBackendFileControlV0 || backend == geminiGoalBackendProcessV0 {
		return backend
	}
	return ""
}

func geminiGoalBackendOperationalFromEnvV0() bool {
	switch geminiGoalBackendFromEnvV0() {
	case geminiGoalBackendFileControlV0, geminiGoalBackendProcessV0:
		return true
	default:
		return false
	}
}
