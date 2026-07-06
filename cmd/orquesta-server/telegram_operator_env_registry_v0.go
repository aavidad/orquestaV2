package main

func init() {
	serverEffectiveEnvRegistryV0[envTelegramOperatorEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Operador Telegram",
		Description: "Activa el adaptador operador Telegram opt-in reutilizando un bot existente.",
	}
	serverEffectiveEnvRegistryV0[envTelegramOperatorTokenV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Token Telegram",
		Description: "Presencia del token del bot existente; el valor real nunca se publica.",
	}
}
