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

func serverResidentDirectorEnabledFromEnvV0() bool {
	if strings.TrimSpace(os.Getenv(envServerResidentDirectorEnabledV0)) != "" {
		return boolEnvOrDefaultV0(envServerResidentDirectorEnabledV0, false)
	}
	return serverAutonomyEnabledFromEnvV0()
}
