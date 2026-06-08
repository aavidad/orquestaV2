package main

const (
	envServerResidentDirectorEnabledV0    = "ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED"
	envServerResidentDirectorMaxActionsV0 = "ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerResidentDirectorEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "resident_director",
		Label:       "Director residente",
		Description: "Activa el loop autonomo residente de briefing del Director.",
	}
	serverEffectiveEnvRegistryV0[envServerResidentDirectorMaxActionsV0] = serverEnvSettingMetadataV0{
		Scope:       "resident_director",
		Label:       "Acciones por tick",
		Description: "Acciones maximas que el Director residente puede ejecutar por pulso.",
	}
}
