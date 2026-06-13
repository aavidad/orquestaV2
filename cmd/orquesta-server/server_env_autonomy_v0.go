package main

import (
	"os"
	"strings"
)

const envServerAutonomyEnabledV0 = "ORQUESTA_SERVER_AUTONOMY_ENABLED"

func init() {
	serverEffectiveEnvRegistryV0[envServerAutonomyEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "server_autonomy",
		Label:       "Perfil autonomia",
		Description: "Activa defaults explicitos para ejecucion autonoma residente sin pisar overrides especificos.",
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

func serverResidentDirectorDisabledExplicitlyFromEnvV0() bool {
	return strings.TrimSpace(os.Getenv(envServerResidentDirectorEnabledV0)) != "" &&
		!boolEnvOrDefaultV0(envServerResidentDirectorEnabledV0, false)
}

func serverOPESAutomationContextFromEnvV0() bool {
	if firstNonEmptyEnvV0(envOPESBaseURLV0, envOPESBaseURLLegacyV0) != "" {
		return true
	}
	for _, key := range []string{
		envOPESProjectWorkDirV0,
		envOPESBridgeEnabledV0,
		envOPESBridgeJobTypeV0,
		envOPESBridgeJobRefV0,
		envOPESBridgeJobTypeSequenceV0,
		envOPESRegistryFinalPkgEnabledV0,
		envOPESTopicRegistryEnabledV0,
		envOPESTopicRegistryToolPathV0,
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}
