package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type opesTopicRegistryConfigV0 struct {
	Enabled  bool
	ToolPath string
	AgentID  string
	Force    bool
}

func opesTopicRegistryConfigFromEnvV0() opesTopicRegistryConfigV0 {
	return opesTopicRegistryConfigFromProjectConfigFileV0(serverProjectConfigFileV0{})
}

func opesTopicRegistryConfigFromProjectConfigFileV0(config serverProjectConfigFileV0) opesTopicRegistryConfigV0 {
	enabled := boolProjectConfigOrEnvOrDefaultV0(envOPESTopicRegistryEnabledV0, config.OPESTopicRegistry.Enabled, false)
	toolPath := stringProjectConfigOrEnvOrDefaultV0(envOPESTopicRegistryToolPathV0, config.OPESTopicRegistry.ToolPath, "")
	if toolPath == "" {
		toolPath = discoveredOPESTopicRegistryToolPathV0(opesProjectWorkDirFromProjectConfigFileV0(config))
	}
	return opesTopicRegistryConfigV0{
		Enabled:  enabled || toolPath != "",
		ToolPath: toolPath,
		AgentID:  stringProjectConfigOrEnvOrDefaultV0(envOPESTopicRegistryAgentIDV0, config.OPESTopicRegistry.AgentID, ""),
		Force:    boolProjectConfigOrEnvOrDefaultV0(envOPESTopicRegistryForceV0, config.OPESTopicRegistry.Force, false),
	}
}

func discoveredOPESTopicRegistryToolPathV0(projectWorkDir string) string {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return ""
	}
	candidate := filepath.Join(projectWorkDir, "opes-salidas", "coordinacion_temarios", "tools", "registro_trabajo_temas.py")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return ""
	}
	return candidate
}

func opesTopicRegistryEffectiveConfigSettingsV0(
	serverConfig orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	config := opesTopicRegistryConfigFromProjectConfigFileV0(projectConfig)
	toolPathSource := configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envOPESTopicRegistryToolPathV0)
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESTopicRegistryEnabledV0,
			strconv.FormatBool(config.Enabled),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envOPESTopicRegistryEnabledV0),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envOPESTopicRegistryToolPathV0,
			sensitiveConfigValueFromSourceV0(config.ToolPath, "opes-topic-registry-tool-configured", toolPathSource),
			toolPathSource,
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESTopicRegistryAgentIDV0,
			config.AgentID,
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envOPESTopicRegistryAgentIDV0),
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envOPESTopicRegistryForceV0,
			strconv.FormatBool(config.Force),
			configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envOPESTopicRegistryForceV0),
		),
	}
}
