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
	enabled := boolEnvOrDefaultV0(envOPESTopicRegistryEnabledV0, false)
	toolPath := strings.TrimSpace(os.Getenv(envOPESTopicRegistryToolPathV0))
	if toolPath == "" {
		toolPath = discoveredOPESTopicRegistryToolPathV0(enabled)
	}
	return opesTopicRegistryConfigV0{
		Enabled:  enabled || toolPath != "",
		ToolPath: toolPath,
		AgentID:  strings.TrimSpace(os.Getenv(envOPESTopicRegistryAgentIDV0)),
		Force:    boolEnvOrDefaultV0(envOPESTopicRegistryForceV0, false),
	}
}

func discoveredOPESTopicRegistryToolPathV0(enabled bool) string {
	projectWorkDir := strings.TrimSpace(os.Getenv(envOPESProjectWorkDirV0))
	if projectWorkDir == "" {
		if !enabled {
			return ""
		}
		projectWorkDir = defaultOPESProjectWorkDirV0
	}
	candidate := filepath.Join(projectWorkDir, "opes-salidas", "coordinacion_temarios", "tools", "registro_trabajo_temas.py")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return ""
	}
	return candidate
}

func opesTopicRegistryEffectiveConfigSettingsV0() []orquestaserver.ServerConfigSettingV0 {
	config := opesTopicRegistryConfigFromEnvV0()
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryV0(envOPESTopicRegistryEnabledV0, strconv.FormatBool(config.Enabled)),
		serverSensitiveConfigSettingFromRegistryV0(envOPESTopicRegistryToolPathV0, configuredRefValueV0(config.ToolPath, "opes-topic-registry-tool-configured")),
		serverConfigSettingFromRegistryV0(envOPESTopicRegistryAgentIDV0, config.AgentID),
		serverConfigSettingFromRegistryV0(envOPESTopicRegistryForceV0, strconv.FormatBool(config.Force)),
	}
}
