package main

const (
	envServerDaemonLogMaxBytesV0      = "ORQUESTA_SERVER_DAEMON_LOG_MAX_BYTES"
	envServerDaemonLogMaxRotatedV0    = "ORQUESTA_SERVER_DAEMON_LOG_MAX_ROTATED"
	envServerDaemonLogRetentionDaysV0 = "ORQUESTA_SERVER_DAEMON_LOG_RETENTION_DAYS"
	envServerDaemonLogRawEnabledV0    = "ORQUESTA_SERVER_DAEMON_LOG_RAW_ENABLED"
	envServerDaemonLogRawReasonV0     = "ORQUESTA_SERVER_DAEMON_LOG_RAW_REASON"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerDaemonLogMaxBytesV0] = serverEnvSettingMetadataV0{
		Scope:       "daemon_logs",
		Label:       "Bytes por log daemon",
		Description: "Tamano maximo por archivo operacional stdout stderr del daemon antes de rotar.",
	}
	serverEffectiveEnvRegistryV0[envServerDaemonLogMaxRotatedV0] = serverEnvSettingMetadataV0{
		Scope:       "daemon_logs",
		Label:       "Rotados daemon",
		Description: "Archivos rotados a conservar para stdout stderr del daemon residente.",
	}
	serverEffectiveEnvRegistryV0[envServerDaemonLogRetentionDaysV0] = serverEnvSettingMetadataV0{
		Scope:       "daemon_logs",
		Label:       "Retencion daemon",
		Description: "Dias declarados de retencion local para logs operacionales del daemon.",
	}
	serverEffectiveEnvRegistryV0[envServerDaemonLogRawEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "daemon_logs",
		Label:       "Raw daemon opt-in",
		Description: "Captura local de stdout stderr crudos del daemon; por defecto queda desactivada.",
	}
}
