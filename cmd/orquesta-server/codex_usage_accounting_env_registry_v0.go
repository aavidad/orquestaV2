package main

func init() {
	serverEffectiveEnvRegistryV0[envCodexUsageAccountingV0] = serverEnvSettingMetadataV0{
		Scope:       "codex_usage_accounting",
		Label:       "Usage accounting Codex",
		Description: "Modo opt-in para publicar métricas de uso Codex desde reportes redactados.",
	}
	serverEffectiveEnvRegistryV0[envCodexUsageLogMaxBytesV0] = serverEnvSettingMetadataV0{
		Scope:       "codex_usage_accounting",
		Label:       "Bytes usage Codex",
		Description: "Bytes máximos leídos del reporte redactado de usage accounting Codex.",
	}
}
