package main

func init() {
	serverEffectiveEnvRegistryV0[envTelegramOperatorEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Operador Telegram",
		Description: "Activa el adaptador operador Telegram opt-in reutilizando un bot existente.",
	}
	serverEffectiveEnvRegistryV0[envTelegramOperatorBotLinkRefV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Ref enlace bot",
		Description: "Ref opaca del enlace/configuracion existente del bot, por ejemplo Inodo Bot.",
	}
	serverEffectiveEnvRegistryV0[envTelegramOperatorTokenV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Token Telegram",
		Description: "Presencia del token del bot existente; el valor real nunca se publica.",
	}
	serverEffectiveEnvRegistryV0[envTelegramOperatorAuthorizedChatRefsV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Chats autorizados Telegram",
		Description: "Refs opacas de chats autorizados para operar Orquesta por Telegram.",
	}
	serverEffectiveEnvRegistryV0[envTelegramOperatorRequireConfirmationV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Confirmacion Telegram",
		Description: "Exige confirmacion explicita en comandos de detener/controlar.",
	}
	serverEffectiveEnvRegistryV0[envTelegramOperatorNotificationTargetV0] = serverEnvSettingMetadataV0{
		Scope:       "telegram_operator",
		Label:       "Target notificaciones Telegram",
		Description: "Ref opaca del chat destino para avisos terminales deduplicados.",
	}
}
