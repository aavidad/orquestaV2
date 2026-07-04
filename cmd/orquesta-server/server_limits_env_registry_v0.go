package main

func init() {
	serverEffectiveEnvRegistryV0[envServerDrainMaxCommandsV0] = serverEnvSettingMetadataV0{
		Scope:       "server_supervisor",
		Label:       "Comandos por drain",
		Description: "Comandos que el supervisor residente puede procesar por ciclo de drain.",
	}
	serverEffectiveEnvRegistryV0[envCodexMaxExpectedSecondsV0] = serverEnvSettingMetadataV0{
		Scope:       "codex_runtime",
		Label:       "Duracion esperada Codex",
		Description: "Duracion esperada maxima en segundos antes de diagnosticar actividad Codex anomala.",
	}
}
