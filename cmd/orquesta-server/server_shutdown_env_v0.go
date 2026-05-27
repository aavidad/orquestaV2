package main

const envServerShutdownGraceMSV0 = "ORQUESTA_SERVER_SHUTDOWN_GRACE_MS"

func init() {
	serverEffectiveEnvRegistryV0[envServerShutdownGraceMSV0] = serverEnvSettingMetadataV0{
		Scope:       "server_lifecycle",
		Label:       "Grace apagado ms",
		Description: "Deadline de composicion para drenar HTTP y goroutines internas antes de publicar stopped.",
	}
}
