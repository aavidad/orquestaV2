package main

const (
	envServerEscalationDirectorEnabledV0        = "ORQUESTA_SERVER_ESCALATION_DIRECTOR_ENABLED"
	envServerEscalationDirectorCommandV0        = "ORQUESTA_SERVER_ESCALATION_DIRECTOR_COMMAND"
	envServerEscalationDirectorTimeoutSecondsV0 = "ORQUESTA_SERVER_ESCALATION_DIRECTOR_TIMEOUT_SECONDS"
	envServerEscalationDirectorMaxPerDayV0      = "ORQUESTA_SERVER_ESCALATION_DIRECTOR_MAX_PER_DAY"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerEscalationDirectorEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "escalation_director",
		Label:       "Director de escalada",
		Description: "Activa el director externo de escalada ante anomalias sin accion automatica.",
	}
	serverEffectiveEnvRegistryV0[envServerEscalationDirectorCommandV0] = serverEnvSettingMetadataV0{
		Scope:       "escalation_director",
		Label:       "Comando de escalada",
		Description: "Comando argv CSV para invocar el director externo de escalada.",
	}
	serverEffectiveEnvRegistryV0[envServerEscalationDirectorTimeoutSecondsV0] = serverEnvSettingMetadataV0{
		Scope:       "escalation_director",
		Label:       "Timeout de escalada",
		Description: "Timeout por invocacion del director externo de escalada.",
	}
	serverEffectiveEnvRegistryV0[envServerEscalationDirectorMaxPerDayV0] = serverEnvSettingMetadataV0{
		Scope:       "escalation_director",
		Label:       "Presupuesto diario de escalada",
		Description: "Maximo de invocaciones diarias del director externo de escalada.",
	}
}
