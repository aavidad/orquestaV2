package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
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

type serverProjectConfigHermesOperatorV0 struct {
	Enabled            *bool   `json:"enabled,omitempty"`
	BaseURL            *string `json:"base_url,omitempty"`
	MCPPath            *string `json:"mcp_path,omitempty"`
	APIKeyFile         *string `json:"api_key_file,omitempty"`
	StatusTool         *string `json:"status_tool,omitempty"`
	BurstTool          *string `json:"burst_tool,omitempty"`
	OutboxTool         *string `json:"outbox_tool,omitempty"`
	QueryTool          *string `json:"query_tool,omitempty"`
	StatusConnectorRef *string `json:"status_connector_ref,omitempty"`
	BurstConnectorRef  *string `json:"burst_connector_ref,omitempty"`
	OutboxConnectorRef *string `json:"outbox_connector_ref,omitempty"`
	QueryConnectorRef  *string `json:"query_connector_ref,omitempty"`
	TimeoutSeconds     *int    `json:"timeout_seconds,omitempty"`
	MaxRequestBytes    *int    `json:"max_request_bytes,omitempty"`
	MaxResponseBytes   *int    `json:"max_response_bytes,omitempty"`
}

type hermesOperatorEnvConfigV0 struct {
	Enabled          bool
	BaseURL          string
	MCPPath          string
	APIKey           string
	APIKeyFile       string
	ToolNames        operatorclient.OperatorMCPClientToolNamesV0
	ConnectorRefs    operatorclient.OperatorMCPClientConnectorRefsV0
	TimeoutSeconds   int
	MaxRequestBytes  int
	MaxResponseBytes int
}

var newHermesOperatorMCPConnectorV0 = operatorhermes.NewHermesOperatorMCPConnectorV0

func hermesOperatorConfigFromProjectConfigV0(config serverProjectConfigFileV0) (hermesOperatorEnvConfigV0, error) {
	file := config.HermesOperator
	enabled, err := hermesOperatorEnabledV0(file.Enabled)
	if err != nil {
		return hermesOperatorEnvConfigV0{}, err
	}
	resolved := hermesOperatorEnvConfigV0{
		Enabled:    enabled,
		BaseURL:    stringProjectConfigOrEnvOrDefaultV0(envHermesBaseURLV0, file.BaseURL, ""),
		MCPPath:    stringProjectConfigOrEnvOrDefaultV0(envHermesMCPPathV0, file.MCPPath, defaultHermesMCPPathV0),
		APIKeyFile: stringProjectConfigFileOrDefaultV0(file.APIKeyFile, ""),
		ToolNames: operatorclient.OperatorMCPClientToolNamesV0{
			Status:        envOrFileOrDefaultHermesV0(envHermesStatusToolV0, file.StatusTool, operator.OperatorMCPStatusToolNameV0),
			Burst:         envOrFileOrDefaultHermesV0(envHermesBurstToolV0, file.BurstTool, operator.OperatorMCPBurstToolNameV0),
			Outbox:        envOrFileOrDefaultHermesV0(envHermesOutboxToolV0, file.OutboxTool, operator.OperatorMCPOutboxToolNameV0),
			DirectedQuery: envOrFileOrDefaultHermesV0(envHermesQueryToolV0, file.QueryTool, operator.OperatorMCPDirectedQueryToolV0),
		},
		ConnectorRefs: operatorclient.OperatorMCPClientConnectorRefsV0{
			Status:        envOrFileOrDefaultHermesV0(envHermesStatusConnectorRefV0, file.StatusConnectorRef, ""),
			Burst:         envOrFileOrDefaultHermesV0(envHermesBurstConnectorRefV0, file.BurstConnectorRef, ""),
			Outbox:        envOrFileOrDefaultHermesV0(envHermesOutboxConnectorRefV0, file.OutboxConnectorRef, ""),
			DirectedQuery: envOrFileOrDefaultHermesV0(envHermesQueryConnectorRefV0, file.QueryConnectorRef, ""),
		},
	}
	if resolved.TimeoutSeconds, err = hermesOperatorPositiveIntV0(envHermesTimeoutSecondsV0, file.TimeoutSeconds, defaultHermesTimeoutSecondsV0); err != nil {
		return hermesOperatorEnvConfigV0{}, err
	}
	if resolved.MaxRequestBytes, err = hermesOperatorPositiveIntV0(envHermesMaxRequestBytesV0, file.MaxRequestBytes, defaultHermesMaxRequestBytesV0); err != nil {
		return hermesOperatorEnvConfigV0{}, err
	}
	if resolved.MaxResponseBytes, err = hermesOperatorPositiveIntV0(envHermesMaxResponseBytesV0, file.MaxResponseBytes, defaultHermesMaxResponseBytesV0); err != nil {
		return hermesOperatorEnvConfigV0{}, err
	}
	if resolved.MCPPath, err = hermesOperatorMCPPathV0(resolved.MCPPath); err != nil {
		return hermesOperatorEnvConfigV0{}, err
	}
	if resolved.BaseURL != "" {
		if resolved.BaseURL, err = hermesOperatorBaseURLV0(resolved.BaseURL); err != nil {
			return hermesOperatorEnvConfigV0{}, err
		}
	}
	if resolved.Enabled && resolved.BaseURL == "" {
		return hermesOperatorEnvConfigV0{}, fmt.Errorf("hermes_operator_base_url_required")
	}
	if legacyKey := strings.TrimSpace(os.Getenv(envHermesAPIKeyV0)); legacyKey != "" {
		resolved.APIKey = legacyKey
	}
	return resolved, nil
}

func hermesOperatorEnabledV0(fileValue *bool) (bool, error) {
	if raw, ok := os.LookupEnv(envHermesEnabledV0); ok {
		value, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return false, fmt.Errorf("hermes_operator_enabled_invalid")
		}
		return value, nil
	}
	if fileValue != nil {
		return *fileValue, nil
	}
	return false, nil
}

func hermesOperatorPositiveIntV0(key string, fileValue *int, fallback int) (int, error) {
	if raw, ok := os.LookupEnv(key); ok {
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || value <= 0 {
			return 0, fmt.Errorf("hermes_operator_%s_invalid", strings.ToLower(strings.TrimPrefix(key, "ORQUESTA_HERMES_")))
		}
		return value, nil
	}
	if fileValue != nil {
		if *fileValue <= 0 {
			return 0, fmt.Errorf("hermes_operator_%s_invalid", strings.ToLower(strings.TrimPrefix(key, "ORQUESTA_HERMES_")))
		}
		return *fileValue, nil
	}
	return fallback, nil
}

func hermesOperatorBaseURLV0(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || strings.TrimSpace(parsed.Hostname()) == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("hermes_operator_base_url_invalid")
	}
	return parsed.String(), nil
}

func hermesOperatorMCPPathV0(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	parsed, err := url.ParseRequestURI(path)
	if err != nil || !strings.HasPrefix(path, "/") || parsed.Path != path || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("hermes_operator_mcp_path_invalid")
	}
	return path, nil
}

func envOrFileOrDefaultHermesV0(key string, fileValue *string, fallback string) string {
	return stringProjectConfigOrEnvOrDefaultV0(key, fileValue, fallback)
}

func hermesOperatorConnectorFromProjectConfigV0(serverConfig orquestaserver.ConfigV0, projectConfig serverProjectConfigFileV0) (operator.OperatorMCPConnectorV0, error) {
	config, err := hermesOperatorConfigFromProjectConfigV0(projectConfig)
	if err != nil || !config.Enabled {
		return nil, err
	}
	if config.APIKey == "" && config.APIKeyFile != "" {
		config.APIKey, err = readServerProjectSecretFileV0(serverConfig, config.APIKeyFile)
		if err != nil {
			return nil, fmt.Errorf("hermes_operator_api_key_file_invalid: %w", err)
		}
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

func hermesOperatorEffectiveConfigSettingsV0(serverConfig orquestaserver.ConfigV0, projectConfig serverProjectConfigFileV0) []orquestaserver.ServerConfigSettingV0 {
	config, err := hermesOperatorConfigFromProjectConfigV0(projectConfig)
	if err != nil {
		return nil
	}
	apiKeyRef := ""
	if config.APIKey != "" || config.APIKeyFile != "" {
		apiKeyRef = "hermes-api-key-configured"
		if strings.TrimSpace(os.Getenv(envHermesAPIKeyV0)) == "" {
			apiKeyRef = "hermes-api-key-file-configured"
		}
	}
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(envHermesEnabledV0, strconv.FormatBool(config.Enabled), configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesEnabledV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envHermesBaseURLV0, configuredRefValueV0(config.BaseURL, "hermes-base-url-configured"), configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesBaseURLV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesMCPPathV0, config.MCPPath, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesMCPPathV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envHermesAPIKeyV0, apiKeyRef, hermesOperatorAPIKeySourceV0(serverConfig, projectConfig)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesStatusToolV0, config.ToolNames.Status, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesStatusToolV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesBurstToolV0, config.ToolNames.Burst, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesBurstToolV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesOutboxToolV0, config.ToolNames.Outbox, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesOutboxToolV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesQueryToolV0, config.ToolNames.DirectedQuery, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesQueryToolV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesStatusConnectorRefV0, config.ConnectorRefs.Status, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesStatusConnectorRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesBurstConnectorRefV0, config.ConnectorRefs.Burst, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesBurstConnectorRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesOutboxConnectorRefV0, config.ConnectorRefs.Outbox, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesOutboxConnectorRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesQueryConnectorRefV0, config.ConnectorRefs.DirectedQuery, configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesQueryConnectorRefV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesTimeoutSecondsV0, strconv.Itoa(config.TimeoutSeconds), configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesTimeoutSecondsV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesMaxRequestBytesV0, strconv.Itoa(config.MaxRequestBytes), configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesMaxRequestBytesV0)),
		serverConfigSettingFromRegistryWithSourceV0(envHermesMaxResponseBytesV0, strconv.Itoa(config.MaxResponseBytes), configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesMaxResponseBytesV0)),
	}
}

func hermesOperatorAPIKeySourceV0(serverConfig orquestaserver.ConfigV0, projectConfig serverProjectConfigFileV0) string {
	if strings.TrimSpace(os.Getenv(envHermesAPIKeyV0)) != "" {
		return "explicit"
	}
	if configStringPointerHasValueV0(projectConfig.HermesOperator.APIKeyFile) {
		return configSettingSourceConfigFileV0
	}
	return configSettingSourceFromConfigOrProjectConfigV0(serverConfig, envHermesAPIKeyV0)
}

func hermesOperatorProjectConfigHasValueForEnvKeyV0(config serverProjectConfigHermesOperatorV0, key string) bool {
	switch key {
	case envHermesEnabledV0:
		return config.Enabled != nil
	case envHermesBaseURLV0:
		return configStringPointerHasValueV0(config.BaseURL)
	case envHermesMCPPathV0:
		return configStringPointerHasValueV0(config.MCPPath)
	case envHermesAPIKeyV0:
		return configStringPointerHasValueV0(config.APIKeyFile)
	case envHermesStatusToolV0:
		return configStringPointerHasValueV0(config.StatusTool)
	case envHermesBurstToolV0:
		return configStringPointerHasValueV0(config.BurstTool)
	case envHermesOutboxToolV0:
		return configStringPointerHasValueV0(config.OutboxTool)
	case envHermesQueryToolV0:
		return configStringPointerHasValueV0(config.QueryTool)
	case envHermesStatusConnectorRefV0:
		return configStringPointerHasValueV0(config.StatusConnectorRef)
	case envHermesBurstConnectorRefV0:
		return configStringPointerHasValueV0(config.BurstConnectorRef)
	case envHermesOutboxConnectorRefV0:
		return configStringPointerHasValueV0(config.OutboxConnectorRef)
	case envHermesQueryConnectorRefV0:
		return configStringPointerHasValueV0(config.QueryConnectorRef)
	case envHermesTimeoutSecondsV0:
		return configIntPointerPositiveV0(config.TimeoutSeconds)
	case envHermesMaxRequestBytesV0:
		return configIntPointerPositiveV0(config.MaxRequestBytes)
	case envHermesMaxResponseBytesV0:
		return configIntPointerPositiveV0(config.MaxResponseBytes)
	default:
		return false
	}
}

func hermesOperatorEnvKeysV0() []string {
	return []string{envHermesEnabledV0, envHermesBaseURLV0, envHermesMCPPathV0, envHermesAPIKeyV0, envHermesStatusToolV0, envHermesBurstToolV0, envHermesOutboxToolV0, envHermesQueryToolV0, envHermesStatusConnectorRefV0, envHermesBurstConnectorRefV0, envHermesOutboxConnectorRefV0, envHermesQueryConnectorRefV0, envHermesTimeoutSecondsV0, envHermesMaxRequestBytesV0, envHermesMaxResponseBytesV0}
}

func hermesOperatorNonSecretEnvKeysV0() []string {
	keys := make([]string, 0, len(hermesOperatorEnvKeysV0())-1)
	for _, key := range hermesOperatorEnvKeysV0() {
		if key != envHermesAPIKeyV0 {
			keys = append(keys, key)
		}
	}
	return keys
}
