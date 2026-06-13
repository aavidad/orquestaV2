package main

import (
	"os"
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
	return opesTopicRegistryConfigV0{
		Enabled:  enabled || toolPath != "",
		ToolPath: toolPath,
		AgentID:  strings.TrimSpace(os.Getenv(envOPESTopicRegistryAgentIDV0)),
		Force:    boolEnvOrDefaultV0(envOPESTopicRegistryForceV0, false),
	}
}

func opesTopicRegistryEffectiveConfigSettingsV0() []orquestaserver.ServerConfigSettingV0 {
	config := opesTopicRegistryConfigFromEnvV0()
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryV0(envOPESTopicRegistryEnabledV0, strconv.FormatBool(config.Enabled)),
		serverSensitiveConfigSettingFromRegistryV0(envOPESTopicRegistryToolPathV0, configuredEnvValueV0(envOPESTopicRegistryToolPathV0, "opes-topic-registry-tool-configured")),
		serverConfigSettingFromRegistryV0(envOPESTopicRegistryAgentIDV0, config.AgentID),
		serverConfigSettingFromRegistryV0(envOPESTopicRegistryForceV0, strconv.FormatBool(config.Force)),
	}
}
