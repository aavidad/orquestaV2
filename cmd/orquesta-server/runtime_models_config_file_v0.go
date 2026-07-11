package main

type serverProjectConfigRuntimeModelsV0 struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	TimeoutSeconds  *int    `json:"timeout_seconds,omitempty"`
	BearerTokenFile *string `json:"bearer_token_file,omitempty"`
}

func runtimeModelsProjectConfigHasValueForEnvKeyV0(config serverProjectConfigRuntimeModelsV0, key string) bool {
	switch key {
	case envOllamaModelManagerEnabledV0:
		return config.Enabled != nil
	case envOllamaModelManagerBaseURLV0:
		return configStringPointerHasValueV0(config.BaseURL)
	case envOllamaModelManagerTimeoutSecondsV0:
		return configIntPointerPositiveV0(config.TimeoutSeconds)
	case envOllamaModelManagerBearerTokenV0:
		return configStringPointerHasValueV0(config.BearerTokenFile)
	default:
		return false
	}
}
