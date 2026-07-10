package main

import (
	"os"
	"path/filepath"
	"strings"
)

const envServerAutonomyEnabledV0 = "ORQUESTA_SERVER_AUTONOMY_ENABLED"

func init() {
	serverEffectiveEnvRegistryV0[envServerAutonomyEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "server_autonomy",
		Label:       "Perfil autonomia",
		Description: "Activa defaults explicitos para ejecucion autonoma residente sin pisar overrides especificos.",
	}
	for key, metadata := range map[string]serverEnvSettingMetadataV0{
		envOPESProjectWorkDirV0: {
			Scope:       "domain_work",
			Label:       "Workdir OPES",
			Description: "Directorio OPES opt-in para guards de escritura local y contexto de composicion.",
		},
		envOPESTimeoutSecondsV0: {
			Scope:       "domain_work",
			Label:       "Timeout OPES segundos",
			Description: "Timeout HTTP en segundos para el conector REST OPES opt-in.",
		},
		envOPESDefaultMaxAttemptsV0: {
			Scope:       "domain_work",
			Label:       "Intentos REST OPES",
			Description: "Intentos maximos por defecto para el conector REST OPES opt-in.",
		},
	} {
		serverEffectiveEnvRegistryV0[key] = metadata
	}
}

func serverAutonomyEnabledFromEnvV0() bool {
	return boolEnvOrDefaultV0(envServerAutonomyEnabledV0, false)
}

func serverAutonomyEffectiveEnabledFromEnvV0() bool {
	return serverAutonomyEnabledFromEnvV0() || serverOPESAutomationContextFromEnvV0()
}

func serverResidentDirectorEnabledFromEnvV0() bool {
	if strings.TrimSpace(os.Getenv(envServerResidentDirectorEnabledV0)) != "" {
		return boolEnvOrDefaultV0(envServerResidentDirectorEnabledV0, false)
	}
	return serverAutonomyEffectiveEnabledFromEnvV0()
}

func serverOPESAutomationContextFromEnvV0() bool {
	return serverOPESAutomationContextFromSnapshotV0(serverOPESConfigSnapshotFromEnvV0())
}

func serverOPESAutomationContextFromSnapshotV0(opesConfig serverOPESConfigSnapshotV0) bool {
	if firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0) != "" {
		return true
	}
	if serverProjectWorkDirLooksLikeOPESV0(os.Getenv(envCodexProjectWorkDirV0)) {
		return true
	}
	if opesConfig.HasProjectWorkDir() {
		return true
	}
	for _, key := range []string{
		envOPESBridgeJobTypeV0,
		envOPESBridgeJobRefV0,
		envOPESBridgeJobTypeSequenceV0,
		envOPESTopicRegistryToolPathV0,
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	for _, key := range []string{
		envOPESBridgeEnabledV0,
		envOPESRegistryFinalPkgEnabledV0,
		envOPESTopicRegistryEnabledV0,
	} {
		if boolEnvOrDefaultV0(key, false) {
			return true
		}
	}
	return false
}

func serverProjectWorkDirLooksLikeOPESV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	clean := strings.ToLower(filepath.ToSlash(filepath.Clean(value)))
	return clean == "opes" ||
		strings.HasSuffix(clean, "/opes") ||
		strings.HasSuffix(clean, "/opes-salidas") ||
		strings.Contains(clean, "/opes/") ||
		strings.Contains(clean, "/opes-salidas/")
}
