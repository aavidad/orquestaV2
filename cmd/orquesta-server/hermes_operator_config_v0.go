package main

import (
	"strconv"
	"time"

	operator "orquesta/modulos/orquesta-operator-mcp"
	operatorclient "orquesta/modulos/orquesta-operator-mcp-client"
	operatorhermes "orquesta/modulos/orquesta-operator-mcp-hermes"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	defaultHermesMCPPathV0          = "/mcp"
	defaultHermesTimeoutSecondsV0   = 30
	defaultHermesMaxRequestBytesV0  = 1 << 20
	defaultHermesMaxResponseBytesV0 = 1 << 20
)

type hermesOperatorEnvConfigV0 struct {
	Enabled          bool
	BaseURL          string
	MCPPath          string
	APIKey           string
	ToolNames        operatorclient.OperatorMCPClientToolNamesV0
	ConnectorRefs    operatorclient.OperatorMCPClientConnectorRefsV0
	TimeoutSeconds   int
	MaxRequestBytes  int
	MaxResponseBytes int
}

var newHermesOperatorMCPConnectorV0 = operatorhermes.NewHermesOperatorMCPConnectorV0

func hermesOperatorEnvConfigFromEnvV0() hermesOperatorEnvConfigV0 {
	return hermesOperatorEnvConfigV0{
		Enabled: boolEnvOrDefaultV0(envHermesEnabledV0, false),
		BaseURL: envOrDefaultV0(envHermesBaseURLV0, ""),
		MCPPath: envOrDefaultV0(envHermesMCPPathV0, defaultHermesMCPPathV0),
		APIKey:  envOrDefaultV0(envHermesAPIKeyV0, ""),
		ToolNames: operatorclient.OperatorMCPClientToolNamesV0{
			Status:        envOrDefaultV0(envHermesStatusToolV0, operator.OperatorMCPStatusToolNameV0),
			Burst:         envOrDefaultV0(envHermesBurstToolV0, operator.OperatorMCPBurstToolNameV0),
			Outbox:        envOrDefaultV0(envHermesOutboxToolV0, operator.OperatorMCPOutboxToolNameV0),
			DirectedQuery: envOrDefaultV0(envHermesQueryToolV0, operator.OperatorMCPDirectedQueryToolV0),
		},
		ConnectorRefs: operatorclient.OperatorMCPClientConnectorRefsV0{
			Status:        envOrDefaultV0(envHermesStatusConnectorRefV0, ""),
			Burst:         envOrDefaultV0(envHermesBurstConnectorRefV0, ""),
			Outbox:        envOrDefaultV0(envHermesOutboxConnectorRefV0, ""),
			DirectedQuery: envOrDefaultV0(envHermesQueryConnectorRefV0, ""),
		},
		TimeoutSeconds:   intEnvOrDefaultV0(envHermesTimeoutSecondsV0, defaultHermesTimeoutSecondsV0),
		MaxRequestBytes:  intEnvOrDefaultV0(envHermesMaxRequestBytesV0, defaultHermesMaxRequestBytesV0),
		MaxResponseBytes: intEnvOrDefaultV0(envHermesMaxResponseBytesV0, defaultHermesMaxResponseBytesV0),
	}
}

func hermesOperatorConnectorFromEnvV0() (operator.OperatorMCPConnectorV0, error) {
	config := hermesOperatorEnvConfigFromEnvV0()
	if !config.Enabled {
		return nil, nil
	}
	return newHermesOperatorMCPConnectorV0(operatorhermes.HermesOperatorMCPConfigV0{
		BaseURL:          config.BaseURL,
		MCPPath:          config.MCPPath,
		APIKey:           config.APIKey,
		ToolNames:        config.ToolNames,
		ConnectorRefs:    config.ConnectorRefs,
		Timeout:          time.Duration(config.TimeoutSeconds) * time.Second,
		MaxRequestBytes:  int64(config.MaxRequestBytes),
		MaxResponseBytes: int64(config.MaxResponseBytes),
	})
}

func hermesOperatorEffectiveConfigSettingsV0() []orquestaserver.ServerConfigSettingV0 {
	config := hermesOperatorEnvConfigFromEnvV0()
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryV0(envHermesEnabledV0, strconv.FormatBool(config.Enabled)),
		sensitiveServerConfigSettingFromRegistryV0(envHermesBaseURLV0, hermesConfigRefIfConfiguredV0(config.BaseURL, "hermes-base-url-configured")),
		serverConfigSettingFromRegistryV0(envHermesMCPPathV0, config.MCPPath),
		sensitiveServerConfigSettingFromRegistryV0(envHermesAPIKeyV0, hermesConfigRefIfConfiguredV0(config.APIKey, "hermes-api-key-configured")),
		serverConfigSettingFromRegistryV0(envHermesStatusToolV0, config.ToolNames.Status),
		serverConfigSettingFromRegistryV0(envHermesBurstToolV0, config.ToolNames.Burst),
		serverConfigSettingFromRegistryV0(envHermesOutboxToolV0, config.ToolNames.Outbox),
		serverConfigSettingFromRegistryV0(envHermesQueryToolV0, config.ToolNames.DirectedQuery),
		serverConfigSettingFromRegistryV0(envHermesStatusConnectorRefV0, config.ConnectorRefs.Status),
		serverConfigSettingFromRegistryV0(envHermesBurstConnectorRefV0, config.ConnectorRefs.Burst),
		serverConfigSettingFromRegistryV0(envHermesOutboxConnectorRefV0, config.ConnectorRefs.Outbox),
		serverConfigSettingFromRegistryV0(envHermesQueryConnectorRefV0, config.ConnectorRefs.DirectedQuery),
		serverConfigSettingFromRegistryV0(envHermesTimeoutSecondsV0, strconv.Itoa(config.TimeoutSeconds)),
		serverConfigSettingFromRegistryV0(envHermesMaxRequestBytesV0, strconv.Itoa(config.MaxRequestBytes)),
		serverConfigSettingFromRegistryV0(envHermesMaxResponseBytesV0, strconv.Itoa(config.MaxResponseBytes)),
	}
}

func hermesConfigRefIfConfiguredV0(value string, ref string) string {
	if value == "" {
		return ""
	}
	return ref
}

func sensitiveServerConfigSettingFromRegistryV0(key string, value string) orquestaserver.ServerConfigSettingV0 {
	setting := serverConfigSettingFromRegistryV0(key, value)
	setting.Sensitive = true
	return setting
}
