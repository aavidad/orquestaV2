package main

import (
	orquestatelegram "orquesta/modulos/orquesta-operator-telegram"
)

func telegramOperatorConfigFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
) orquestatelegram.ConfigV0 {
	token := telegramOperatorTokenFromProjectConfigFileV0(config)
	return orquestatelegram.ConfigV0{
		Enabled:             telegramOperatorEnabledFromProjectConfigFileV0(config),
		BotLinkRef:          telegramOperatorBotLinkRefFromProjectConfigFileV0(config),
		TokenConfigured:     token != "",
		AuthorizedChatRefs:  telegramOperatorAuthorizedChatRefsFromProjectConfigFileV0(config),
		RequireConfirmation: telegramOperatorRequireConfirmationFromProjectConfigFileV0(config),
	}
}
