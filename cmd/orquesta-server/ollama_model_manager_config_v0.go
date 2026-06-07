package main

import (
	"net/http"
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

func runtimeModelManagerFromEnvV0() orquestaruntime.RuntimeModelManagerPortV0 {
	baseURL := strings.TrimSpace(os.Getenv(envOllamaModelManagerBaseURLV0))
	if !boolEnvOrDefaultV0(envOllamaModelManagerEnabledV0, false) && baseURL == "" {
		return nil
	}
	if baseURL == "" {
		baseURL = orquestaruntimeollama.DefaultOllamaBaseURLV0
	}
	timeout := time.Duration(intEnvOrDefaultV0(envOllamaModelManagerTimeoutSecondsV0, 120)) * time.Second
	manager := orquestaruntimeollama.NewOllamaModelManagerV0(baseURL, &http.Client{Timeout: timeout})
	manager.BearerToken = strings.TrimSpace(os.Getenv(envOllamaModelManagerBearerTokenV0))
	return manager
}

func ollamaModelManagerEffectiveConfigSettingsV0() []orquestaserver.ServerConfigSettingV0 {
	baseURL := strings.TrimSpace(os.Getenv(envOllamaModelManagerBaseURLV0))
	enabled := boolEnvOrDefaultV0(envOllamaModelManagerEnabledV0, false) || baseURL != ""
	token := strings.TrimSpace(os.Getenv(envOllamaModelManagerBearerTokenV0))
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryV0(envOllamaModelManagerEnabledV0, strconv.FormatBool(enabled)),
		sensitiveServerConfigSettingFromRegistryV0(envOllamaModelManagerBaseURLV0, hermesConfigRefIfConfiguredV0(baseURL, "ollama-base-url-configured")),
		sensitiveServerConfigSettingFromRegistryV0(envOllamaModelManagerBearerTokenV0, hermesConfigRefIfConfiguredV0(token, "ollama-bearer-token-configured")),
		serverConfigSettingFromRegistryV0(envOllamaModelManagerTimeoutSecondsV0, strconv.Itoa(intEnvOrDefaultV0(envOllamaModelManagerTimeoutSecondsV0, 120))),
	}
}
