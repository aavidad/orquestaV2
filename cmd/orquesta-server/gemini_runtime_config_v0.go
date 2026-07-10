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
	for key, metadata := range map[string]serverEnvSettingMetadataV0{
		envGeminiEnabledV0: {
			Scope: "gemini_runtime", Label: "Gemini activo",
			Description: "Activa el proveedor Gemini opt-in para trabajos visual_asset.",
		},
		envGeminiCommandV0: {
			Scope: "gemini_runtime", Label: "Comando Gemini",
			Description: "Comando o ruta del ejecutable Gemini para el adaptador opt-in.",
		},
		envGeminiProjectWorkDirV0: {
			Scope: "gemini_runtime", Label: "Proyecto Gemini",
			Description: "Directorio del proyecto que Gemini usa como contexto de trabajo.",
		},
		envGeminiRuntimeWorkDirV0: {
			Scope: "gemini_runtime", Label: "Runtime Gemini",
			Description: "Directorio aislado de recibos y estado del runtime Gemini.",
		},
		envGeminiHomeV0: {
			Scope: "gemini_runtime", Label: "Home Gemini",
			Description: "HOME opt-in proyectado al proceso Gemini.",
		},
		envGeminiPathV0: {
			Scope: "gemini_runtime", Label: "PATH Gemini",
			Description: "PATH opt-in proyectado al proceso Gemini.",
		},
		envGeminiModelV0: {
			Scope: "gemini_runtime", Label: "Modelo Gemini",
			Description: "Modelo Gemini CLI usado por el adaptador visual.",
		},
		envGeminiApprovalModeV0: {
			Scope: "gemini_runtime", Label: "Approval Gemini",
			Description: "Modo de aprobacion Gemini CLI para trabajos visuales opt-in.",
		},
		envGeminiOutputFormatV0: {
			Scope: "gemini_runtime", Label: "Formato Gemini",
			Description: "Formato de salida solicitado al proceso Gemini.",
		},
		envGeminiExtraArgsV0: {
			Scope: "gemini_runtime", Label: "Argumentos Gemini",
			Description: "Argumentos extra auditables para Gemini; no sustituyen contratos de goal.",
		},
	} {
		serverEffectiveEnvRegistryV0[key] = metadata
	}
}

func geminiRuntimeConfigV0(
	serverConfig orquestaserver.ConfigV0,
) orquestaappcodexstack.GeminiRuntimeConfigV0 {
	projectConfig := projectConfigFromServerConfigBestEffortV0(serverConfig)
	if strings.TrimSpace(os.Getenv(envGeminiEnabledV0)) != "" {
		if !boolEnvOrDefaultV0(envGeminiEnabledV0, false) {
			return orquestaappcodexstack.GeminiRuntimeConfigV0{}
		}
	} else if projectConfig.GeminiRuntime.Enabled == nil || !*projectConfig.GeminiRuntime.Enabled {
		return orquestaappcodexstack.GeminiRuntimeConfigV0{}
	}
	return geminiRuntimeConfigFromProjectConfigV0(
		projectConfig,
		absDirProjectConfigOrEnvOrDefaultV0(
			envGeminiProjectWorkDirV0,
			projectConfig.GeminiRuntime.ProjectWorkDir,
			serverConfig.ProjectWorkDir,
		),
		absDirProjectConfigOrEnvOrDefaultV0(
			envGeminiRuntimeWorkDirV0,
			projectConfig.GeminiRuntime.RuntimeWorkDir,
			serverConfig.RuntimeWorkDir,
		),
	)
}

func geminiRuntimeConfigFromProjectConfigV0(
	projectConfig serverProjectConfigFileV0,
	projectWorkDir string,
	runtimeWorkDir string,
) orquestaappcodexstack.GeminiRuntimeConfigV0 {
	runtime := projectConfig.GeminiRuntime
	return orquestaappcodexstack.GeminiRuntimeConfigV0{
		Enabled:        true,
		CommandPath:    geminiCommandPathV0(projectConfig),
		ProjectWorkDir: projectWorkDir,
		RuntimeWorkDir: runtimeWorkDir,
		HomeDir:        stringProjectConfigOrEnvOrDefaultV0(envGeminiHomeV0, runtime.HomeDir, ""),
		PathEnv:        stringProjectConfigOrEnvOrDefaultV0(envGeminiPathV0, runtime.Path, os.Getenv("PATH")),
		Model:          stringProjectConfigOrEnvOrDefaultV0(envGeminiModelV0, runtime.Model, ""),
		ApprovalMode:   stringProjectConfigOrEnvOrDefaultV0(envGeminiApprovalModeV0, runtime.ApprovalMode, "auto_edit"),
		OutputFormat:   stringProjectConfigOrEnvOrDefaultV0(envGeminiOutputFormatV0, runtime.OutputFormat, "text"),
		PromptLocale:   goalBackendPromptLocaleFromProjectConfigFileV0(projectConfig),
		ExtraArgs:      geminiExtraArgsFromProjectConfigV0(runtime),
	}
}

func geminiCommandPathV0(projectConfig serverProjectConfigFileV0) string {
	raw := stringProjectConfigOrEnvOrDefaultV0(
		envGeminiCommandV0,
		projectConfig.GeminiRuntime.CommandPath,
		"gemini",
	)
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		return raw
	}
	return path
}

func geminiExtraArgsFromProjectConfigV0(runtime serverProjectConfigGeminiRuntimeV0) []string {
	if raw := strings.TrimSpace(os.Getenv(envGeminiExtraArgsV0)); raw != "" {
		return strings.Fields(raw)
	}
	return compactStringsV0(runtime.ExtraArgs)
}

func geminiGoalBackendFromValueV0(backend string) string {
	backend = strings.TrimSpace(backend)
	if backend == geminiGoalBackendFileControlV0 || backend == geminiGoalBackendProcessV0 {
		return backend
	}
	return ""
}
