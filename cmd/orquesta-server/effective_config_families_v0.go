package main

import (
	"os"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func geminiRuntimeEffectiveConfigSettingsV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	runtime := projectConfig.GeminiRuntime
	effective := geminiRuntimeConfigV0(config)
	return []orquestaserver.ServerConfigSettingV0{
		effectiveConfigBoolSettingV0(envGeminiEnabledV0, effective.Enabled, effectiveConfigBoolSourceV0(runtime.Enabled, envGeminiEnabledV0, false), "gemini_runtime", "Gemini activo"),
		effectiveConfigSensitiveSettingV0(envGeminiCommandV0, effective.CommandPath, geminiRuntimeConfigSourceV0(runtime.CommandPath, envGeminiCommandV0, false), "gemini_runtime", "Comando Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiProjectWorkDirV0, effective.ProjectWorkDir, geminiRuntimeConfigSourceV0(runtime.ProjectWorkDir, envGeminiProjectWorkDirV0, false), "gemini_runtime", "Proyecto Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiRuntimeWorkDirV0, effective.RuntimeWorkDir, geminiRuntimeConfigSourceV0(runtime.RuntimeWorkDir, envGeminiRuntimeWorkDirV0, false), "gemini_runtime", "Runtime Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiHomeV0, effective.HomeDir, geminiRuntimeConfigSourceV0(runtime.HomeDir, envGeminiHomeV0, false), "gemini_runtime", "Home Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiPathV0, effective.PathEnv, geminiRuntimeConfigSourceV0(runtime.Path, envGeminiPathV0, false), "gemini_runtime", "PATH Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiModelV0, effective.Model, geminiRuntimeConfigSourceV0(runtime.Model, envGeminiModelV0, false), "gemini_runtime", "Modelo Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiApprovalModeV0, effective.ApprovalMode, geminiRuntimeConfigSourceV0(runtime.ApprovalMode, envGeminiApprovalModeV0, false), "gemini_runtime", "Approval Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiOutputFormatV0, effective.OutputFormat, geminiRuntimeConfigSourceV0(runtime.OutputFormat, envGeminiOutputFormatV0, false), "gemini_runtime", "Formato Gemini"),
		effectiveConfigSensitiveSettingV0(envGeminiExtraArgsV0, strings.Join(effective.ExtraArgs, " "), geminiRuntimeSliceConfigSourceV0(runtime.ExtraArgs, envGeminiExtraArgsV0), "gemini_runtime", "Argumentos Gemini"),
	}
}

func requiredTestRunnerEffectiveConfigSettingsV0(projectConfig serverProjectConfigFileV0) []orquestaserver.ServerConfigSettingV0 {
	runtime := projectConfig.RequiredTestRunner
	return []orquestaserver.ServerConfigSettingV0{
		effectiveConfigBoolSettingV0(envRequiredTestRunnerEnabledV0, requiredTestRunnerEnabledFromProjectConfigV0(projectConfig), effectiveConfigBoolSourceV0(runtime.Enabled, envRequiredTestRunnerEnabledV0, true), "required_test_runner", "Runner de tests requerido activo"),
		effectiveConfigSensitiveSettingV0(envRequiredTestGoCommandV0, stringProjectConfigOrEnvOrDefaultV0(envRequiredTestGoCommandV0, runtime.GoCommand, ""), effectiveConfigStringSourceV0(runtime.GoCommand, envRequiredTestGoCommandV0, false), "required_test_runner", "Comando Go permitido"),
		effectiveConfigSensitiveSettingV0(envRequiredTestAllowedCommandsV0, strings.Join(requiredTestAllowedCommandEntriesFromProjectConfigV0(runtime), ","), requiredTestRunnerMapConfigSourceV0(runtime.AllowedCommands, envRequiredTestAllowedCommandsV0), "required_test_runner", "Comandos permitidos"),
		effectiveConfigSensitiveSettingV0(envRequiredTestOutputDirV0, stringProjectConfigOrEnvOrDefaultV0(envRequiredTestOutputDirV0, runtime.OutputDir, ""), effectiveConfigStringSourceV0(runtime.OutputDir, envRequiredTestOutputDirV0, false), "required_test_runner", "Directorio de salida"),
		effectiveConfigSensitiveSettingV0(envRequiredTestEnvV0, strings.Join(requiredTestEnvEntriesFromProjectConfigV0(runtime), ","), requiredTestRunnerMapConfigSourceV0(runtime.Environment, envRequiredTestEnvV0), "required_test_runner", "Entorno permitido"),
		effectiveConfigSettingWithSourceV0(envRequiredTestMaxOutputBytesV0, strconv.Itoa(intProjectConfigOrEnvOrDefaultV0(envRequiredTestMaxOutputBytesV0, runtime.MaxOutputBytes, 1024*1024)), effectiveConfigIntSourceV0(runtime.MaxOutputBytes, envRequiredTestMaxOutputBytesV0), "required_test_runner", "Maximo de salida de tests"),
		effectiveConfigSettingWithSourceV0(envRequiredTestOutputMaxArtifactsV0, strconv.Itoa(intProjectConfigOrEnvOrDefaultV0(envRequiredTestOutputMaxArtifactsV0, runtime.MaxArtifacts, 200)), effectiveConfigIntSourceV0(runtime.MaxArtifacts, envRequiredTestOutputMaxArtifactsV0), "required_test_runner", "Maximo de artefactos de tests"),
	}
}

func autoprogrammingPromotionEffectiveConfigSettingsV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	promotion := projectConfig.Autoprogramming.Promotion
	effective := autoprogrammingPromotionConfigFromEnvV0(config)
	archiveDir := autoprogrammingPromotionArchiveDirFromProjectConfigV0(config)
	return []orquestaserver.ServerConfigSettingV0{
		effectiveConfigBoolSettingV0(envServerAutoprogrammingPromotionEnabledV0, effective.Enabled, effectiveConfigBoolSourceV0(promotion.Enabled, envServerAutoprogrammingPromotionEnabledV0, true), "autoprogramming.promotion", "Promocion de autoprogramacion activa"),
		effectiveConfigSensitiveSettingV0(envServerAutoprogrammingPromotionArchiveDirV0, archiveDir, effectiveConfigStringSourceV0(promotion.ArchiveDir, envServerAutoprogrammingPromotionArchiveDirV0, false), "autoprogramming.promotion", "Archivo de promocion"),
		effectiveConfigSensitiveSettingV0(envServerAutoprogrammingPromotionRepoRefV0, stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionRepoRefV0, promotion.RepoRef, defaultAutoprogrammingPromotionRepoRefV0), effectiveConfigStringSourceV0(promotion.RepoRef, envServerAutoprogrammingPromotionRepoRefV0, false), "autoprogramming.promotion", "Repositorio de promocion"),
		effectiveConfigSensitiveSettingV0(envServerAutoprogrammingPromotionAppRefV0, stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionAppRefV0, promotion.AppRef, defaultAutoprogrammingPromotionAppRefV0), effectiveConfigStringSourceV0(promotion.AppRef, envServerAutoprogrammingPromotionAppRefV0, false), "autoprogramming.promotion", "Aplicacion de promocion"),
		effectiveConfigSensitiveSettingV0(envServerAutoprogrammingPromotionCommitMessageV0, stringProjectConfigOrEnvOrDefaultV0(envServerAutoprogrammingPromotionCommitMessageV0, promotion.CommitMessage, defaultAutoprogrammingPromotionMessageV0), effectiveConfigStringSourceV0(promotion.CommitMessage, envServerAutoprogrammingPromotionCommitMessageV0, false), "autoprogramming.promotion", "Mensaje de commit de promocion"),
	}
}

func effectiveConfigBoolSettingV0(key string, value bool, source string, scope string, label string) orquestaserver.ServerConfigSettingV0 {
	return effectiveConfigSettingWithSourceV0(key, strconv.FormatBool(value), source, scope, label)
}

func effectiveConfigSensitiveSettingV0(key string, value string, source string, scope string, label string) orquestaserver.ServerConfigSettingV0 {
	setting := effectiveConfigSettingWithSourceV0(key, configuredRefValueV0(value, strings.ToLower(strings.ReplaceAll(scope, ".", "-"))+"-configured"), source, scope, label)
	setting.Sensitive = true
	return setting
}

func effectiveConfigSettingWithSourceV0(key string, value string, source string, scope string, label string) orquestaserver.ServerConfigSettingV0 {
	return serverConfigSettingWithSourceV0(key, value, source, scope, label, "Valor efectivo de configuracion; los valores sensibles se publican solo como presencia configurada.")
}

func geminiRuntimeConfigSourceV0(value *string, key string, envPresence bool) string {
	return effectiveConfigStringSourceV0(value, key, envPresence)
}

func geminiRuntimeSliceConfigSourceV0(values []string, key string) string {
	return effectiveConfigPointerSourceV0(len(compactStringsV0(values)) > 0, key, false)
}

func effectiveConfigStringSourceV0(value *string, key string, envPresence bool) string {
	return effectiveConfigPointerSourceV0(value != nil && strings.TrimSpace(*value) != "", key, envPresence)
}

func requiredTestRunnerMapConfigSourceV0(values map[string]string, key string) string {
	return effectiveConfigPointerSourceV0(len(values) > 0, key, false)
}

func effectiveConfigBoolSourceV0(value *bool, key string, envPresence bool) string {
	return effectiveConfigPointerSourceV0(value != nil, key, envPresence)
}

func effectiveConfigIntSourceV0(value *int, key string) string {
	return effectiveConfigPointerSourceV0(value != nil && *value > 0, key, false)
}

func effectiveConfigPointerSourceV0(fileConfigured bool, key string, envPresence bool) string {
	if envPresence {
		if _, ok := os.LookupEnv(key); ok {
			return "explicit"
		}
	} else if strings.TrimSpace(os.Getenv(key)) != "" {
		return "explicit"
	}
	if fileConfigured {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}
