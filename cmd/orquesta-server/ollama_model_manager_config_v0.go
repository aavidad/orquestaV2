package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeollama "orquesta/modulos/orquesta-runtime-ollama"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	envOllamaModelManagerEnabledV0        = "ORQUESTA_OLLAMA_MODEL_MANAGER_ENABLED"
	envOllamaModelManagerBaseURLV0        = "ORQUESTA_OLLAMA_MODEL_MANAGER_BASE_URL"
	envOllamaModelManagerBearerTokenV0    = "ORQUESTA_OLLAMA_MODEL_MANAGER_BEARER_TOKEN"
	envOllamaModelManagerTimeoutSecondsV0 = "ORQUESTA_OLLAMA_MODEL_MANAGER_TIMEOUT_SECONDS"
)

func init() {
	serverEffectiveEnvRegistryV0[envOllamaModelManagerEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "runtime_models",
		Label:       "Gestor Ollama activo",
		Description: "Activa el puerto opt-in para listar, descargar, servir y parar modelos Ollama.",
	}
	serverEffectiveEnvRegistryV0[envOllamaModelManagerBaseURLV0] = serverEnvSettingMetadataV0{
		Scope:       "runtime_models",
		Label:       "Base URL Ollama",
		Description: "Endpoint HTTP de Ollama local, remoto o cloud.",
	}
	serverEffectiveEnvRegistryV0[envOllamaModelManagerTimeoutSecondsV0] = serverEnvSettingMetadataV0{
		Scope:       "runtime_models",
		Label:       "Timeout Ollama",
		Description: "Timeout HTTP para operaciones de gestion de modelos Ollama.",
	}
	serverEffectiveEnvRegistryV0[envOllamaModelManagerBearerTokenV0] = serverEnvSettingMetadataV0{
		Scope:       "runtime_models",
		Label:       "Bearer Ollama",
		Description: "Token bearer opcional para endpoints Ollama protegidos.",
	}
}

const (
	minOllamaModelManagerTimeoutSecondsV0 = 1
	maxOllamaModelManagerTimeoutSecondsV0 = 3600
)

type ollamaModelManagerConfigV0 struct {
	Enabled         bool
	BaseURL         string
	Timeout         time.Duration
	BearerTokenFile *string
}

func runtimeModelManagerFromConfigV0(
	serverConfig orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) (orquestaruntime.RuntimeModelManagerPortV0, error) {
	resolved, err := ollamaModelManagerConfigFromProjectConfigV0(projectConfig)
	if err != nil {
		return nil, err
	}
	if !resolved.Enabled {
		return nil, nil
	}
	manager := orquestaruntimeollama.NewOllamaModelManagerV0(resolved.BaseURL, &http.Client{Timeout: resolved.Timeout})
	token := strings.TrimSpace(os.Getenv(envOllamaModelManagerBearerTokenV0))
	if token == "" && resolved.BearerTokenFile != nil {
		token, err = readServerProjectSecretFileV0(serverConfig, *resolved.BearerTokenFile)
		if err != nil {
			return nil, fmt.Errorf("ollama_bearer_token_file_invalid: %w", err)
		}
	}
	manager.BearerToken = token
	return manager, nil
}

func ollamaModelManagerConfigFromProjectConfigV0(projectConfig serverProjectConfigFileV0) (ollamaModelManagerConfigV0, error) {
	timeout, err := ollamaModelManagerTimeoutFromProjectConfigV0(projectConfig.RuntimeModels.TimeoutSeconds)
	if err != nil {
		return ollamaModelManagerConfigV0{}, err
	}
	enabled, explicitlyConfigured, err := ollamaModelManagerEnabledFromProjectConfigV0(projectConfig.RuntimeModels.Enabled)
	if err != nil {
		return ollamaModelManagerConfigV0{}, err
	}
	baseURL := stringProjectConfigOrEnvOrDefaultV0(envOllamaModelManagerBaseURLV0, projectConfig.RuntimeModels.BaseURL, "")
	if !explicitlyConfigured {
		enabled = baseURL != ""
	}
	resolved := ollamaModelManagerConfigV0{
		Enabled:         enabled,
		Timeout:         timeout,
		BearerTokenFile: projectConfig.RuntimeModels.BearerTokenFile,
	}
	if !enabled {
		return resolved, nil
	}
	if baseURL == "" {
		baseURL = orquestaruntimeollama.DefaultOllamaBaseURLV0
	}
	baseURL, err = ollamaModelManagerBaseURLV0(baseURL)
	if err != nil {
		return ollamaModelManagerConfigV0{}, err
	}
	resolved.BaseURL = baseURL
	return resolved, nil
}

func ollamaModelManagerEnabledFromProjectConfigV0(fileValue *bool) (bool, bool, error) {
	if raw, ok := os.LookupEnv(envOllamaModelManagerEnabledV0); ok {
		value, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return false, true, fmt.Errorf("ollama_model_manager_enabled_invalid")
		}
		return value, true, nil
	}
	if fileValue != nil {
		return *fileValue, true, nil
	}
	return false, false, nil
}

func ollamaModelManagerTimeoutFromProjectConfigV0(fileValue *int) (time.Duration, error) {
	seconds := int64(120)
	if raw, ok := os.LookupEnv(envOllamaModelManagerTimeoutSecondsV0); ok {
		parsed, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("ollama_model_manager_timeout_invalid")
		}
		seconds = parsed
	} else if fileValue != nil {
		seconds = int64(*fileValue)
	}
	if seconds < minOllamaModelManagerTimeoutSecondsV0 || seconds > maxOllamaModelManagerTimeoutSecondsV0 {
		return 0, fmt.Errorf("ollama_model_manager_timeout_invalid")
	}
	return time.Duration(seconds) * time.Second, nil
}

func ollamaModelManagerBaseURLV0(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		strings.TrimSpace(parsed.Hostname()) == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("ollama_model_manager_base_url_invalid")
	}
	return parsed.String(), nil
}

func ollamaModelManagerEffectiveConfigSettingsV0(config orquestaserver.ConfigV0, projectConfig serverProjectConfigFileV0) []orquestaserver.ServerConfigSettingV0 {
	resolved, err := ollamaModelManagerConfigFromProjectConfigV0(projectConfig)
	if err != nil {
		return nil
	}
	tokenPresent := strings.TrimSpace(os.Getenv(envOllamaModelManagerBearerTokenV0)) != "" || configStringPointerHasValueV0(projectConfig.RuntimeModels.BearerTokenFile)
	tokenRef := ""
	if tokenPresent {
		tokenRef = "ollama-bearer-token-configured"
		if strings.TrimSpace(os.Getenv(envOllamaModelManagerBearerTokenV0)) == "" {
			tokenRef = "ollama-bearer-token-file-configured"
		}
	}
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(envOllamaModelManagerEnabledV0, strconv.FormatBool(resolved.Enabled), configSettingSourceFromConfigOrProjectConfigV0(config, envOllamaModelManagerEnabledV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envOllamaModelManagerBaseURLV0, configuredRefValueV0(resolved.BaseURL, "ollama-base-url-configured"), configSettingSourceFromConfigOrProjectConfigV0(config, envOllamaModelManagerBaseURLV0)),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(envOllamaModelManagerBearerTokenV0, tokenRef, configSettingSourceFromConfigOrProjectConfigV0(config, envOllamaModelManagerBearerTokenV0)),
		serverConfigSettingFromRegistryWithSourceV0(envOllamaModelManagerTimeoutSecondsV0, strconv.Itoa(int(resolved.Timeout/time.Second)), configSettingSourceFromConfigOrProjectConfigV0(config, envOllamaModelManagerTimeoutSecondsV0)),
	}
}

func ollamaModelManagerEnvKeysV0() []string {
	return []string{envOllamaModelManagerEnabledV0, envOllamaModelManagerBaseURLV0, envOllamaModelManagerBearerTokenV0, envOllamaModelManagerTimeoutSecondsV0}
}
