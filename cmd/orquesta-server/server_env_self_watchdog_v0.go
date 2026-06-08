package main

const (
	envServerSelfWatchdogDisabledV0          = "ORQUESTA_SERVER_SELF_WATCHDOG_DISABLED"
	envServerSelfWatchdogCPUHighPercentV0    = "ORQUESTA_SERVER_SELF_WATCHDOG_CPU_HIGH_PERCENT"
	envServerSelfWatchdogSustainedSecondsV0  = "ORQUESTA_SERVER_SELF_WATCHDOG_SUSTAINED_SECONDS"
	envServerSelfWatchdogNoProgressSecondsV0 = "ORQUESTA_SERVER_SELF_WATCHDOG_NO_PROGRESS_SECONDS"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerSelfWatchdogDisabledV0] = serverEnvSettingMetadataV0{
		Scope:       "server_self_watchdog",
		Label:       "Watchdog CPU desactivado",
		Description: "Desactiva la parada cooperativa por CPU sostenida sin causa ni progreso.",
	}
	serverEffectiveEnvRegistryV0[envServerSelfWatchdogCPUHighPercentV0] = serverEnvSettingMetadataV0{
		Scope:       "server_self_watchdog",
		Label:       "CPU alta %",
		Description: "Uso de CPU del proceso a partir del cual se observa causa operativa.",
	}
	serverEffectiveEnvRegistryV0[envServerSelfWatchdogSustainedSecondsV0] = serverEnvSettingMetadataV0{
		Scope:       "server_self_watchdog",
		Label:       "CPU sostenida segundos",
		Description: "Ventana minima de CPU alta antes de pedir parada cooperativa.",
	}
	serverEffectiveEnvRegistryV0[envServerSelfWatchdogNoProgressSecondsV0] = serverEnvSettingMetadataV0{
		Scope:       "server_self_watchdog",
		Label:       "Sin progreso segundos",
		Description: "Ventana sin progreso observable requerida antes de parar por CPU alta.",
	}
}
