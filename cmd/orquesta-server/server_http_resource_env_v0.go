package main

const (
	envServerReadHeaderTimeoutMSV0 = "ORQUESTA_SERVER_READ_HEADER_TIMEOUT_MS"
	envServerReadTimeoutMSV0       = "ORQUESTA_SERVER_READ_TIMEOUT_MS"
	envServerWriteTimeoutMSV0      = "ORQUESTA_SERVER_WRITE_TIMEOUT_MS"
	envServerIdleTimeoutMSV0       = "ORQUESTA_SERVER_IDLE_TIMEOUT_MS"
	envServerMaxHeaderBytesV0      = "ORQUESTA_SERVER_MAX_HEADER_BYTES"
	envServerControlBodyMaxBytesV0 = "ORQUESTA_SERVER_CONTROL_BODY_MAX_BYTES"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerReadHeaderTimeoutMSV0] = serverEnvSettingMetadataV0{
		Scope:       "server_http",
		Label:       "Timeout cabeceras HTTP ms",
		Description: "Tiempo maximo para leer cabeceras del servidor residente.",
	}
	serverEffectiveEnvRegistryV0[envServerReadTimeoutMSV0] = serverEnvSettingMetadataV0{
		Scope:       "server_http",
		Label:       "Timeout lectura HTTP ms",
		Description: "Tiempo maximo para leer request completo del servidor residente.",
	}
	serverEffectiveEnvRegistryV0[envServerWriteTimeoutMSV0] = serverEnvSettingMetadataV0{
		Scope:       "server_http",
		Label:       "Timeout escritura HTTP ms",
		Description: "Tiempo maximo para escribir respuesta del servidor residente.",
	}
	serverEffectiveEnvRegistryV0[envServerIdleTimeoutMSV0] = serverEnvSettingMetadataV0{
		Scope:       "server_http",
		Label:       "Timeout idle HTTP ms",
		Description: "Tiempo maximo de conexion idle en loopback/residente.",
	}
	serverEffectiveEnvRegistryV0[envServerMaxHeaderBytesV0] = serverEnvSettingMetadataV0{
		Scope:       "server_http",
		Label:       "Bytes cabecera HTTP",
		Description: "Tamano maximo de cabeceras HTTP del servidor residente.",
	}
	serverEffectiveEnvRegistryV0[envServerControlBodyMaxBytesV0] = serverEnvSettingMetadataV0{
		Scope:       "server_http",
		Label:       "Bytes body control",
		Description: "Limite de body JSON para rutas de control plane residentes.",
	}
}
