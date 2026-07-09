package main

import (
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	telegramOperatorBotLinkConfiguredRefV0 = "telegram-operator-bot-link-configured"
	telegramOperatorTokenConfiguredRefV0   = "telegram-operator-token-configured"
	telegramOperatorChatRefsConfiguredV0   = "telegram-operator-chat-refs-configured"

	telegramOperatorConfigKeyEnabledV0             = "telegram_operator.enabled"
	telegramOperatorConfigKeyBotLinkRefV0          = "telegram_operator.bot_link_ref"
	telegramOperatorConfigKeyTokenV0               = "telegram_operator.token"
	telegramOperatorConfigKeyAuthorizedChatRefsV0  = "telegram_operator.authorized_chat_refs"
	telegramOperatorConfigKeyRequireConfirmationV0 = "telegram_operator.require_confirmation"
	telegramOperatorConfigKeyNotificationTargetV0  = "telegram_operator.notification_target_ref"
)

type serverProjectConfigTelegramOperatorV0 struct {
	Enabled               *bool     `json:"enabled,omitempty"`
	BotLinkRef            *string   `json:"bot_link_ref,omitempty"`
	Token                 *string   `json:"token,omitempty"`
	AuthorizedChatRefs    *[]string `json:"authorized_chat_refs,omitempty"`
	RequireConfirmation   *bool     `json:"require_confirmation,omitempty"`
	NotificationTargetRef *string   `json:"notification_target_ref,omitempty"`
}

func telegramOperatorEnabledFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigFileOrDefaultV0(config.TelegramOperator.Enabled, false)
}

func telegramOperatorBotLinkRefFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigFileOrDefaultV0(config.TelegramOperator.BotLinkRef, "")
}

func telegramOperatorTokenFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigFileOrDefaultV0(config.TelegramOperator.Token, "")
}

func telegramOperatorAuthorizedChatRefsFromProjectConfigFileV0(config serverProjectConfigFileV0) []string {
	return stringSliceProjectConfigFileOrDefaultV0(config.TelegramOperator.AuthorizedChatRefs, nil)
}

func telegramOperatorRequireConfirmationFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigFileOrDefaultV0(config.TelegramOperator.RequireConfirmation, true)
}

func telegramOperatorNotificationTargetFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	target := stringProjectConfigFileOrDefaultV0(config.TelegramOperator.NotificationTargetRef, "")
	if strings.TrimSpace(target) != "" {
		return strings.TrimSpace(target)
	}
	chats := telegramOperatorAuthorizedChatRefsFromProjectConfigFileV0(config)
	if len(chats) > 0 {
		return strings.TrimSpace(chats[0])
	}
	return ""
}

func telegramOperatorEffectiveConfigSettingsV0(
	config orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) []orquestaserver.ServerConfigSettingV0 {
	_ = config
	enabledSource := configFileSettingSourceFromBoolPointerV0(projectConfig.TelegramOperator.Enabled)
	tokenSource := configFileSettingSourceFromStringPointerV0(projectConfig.TelegramOperator.Token)
	botLinkSource := configFileSettingSourceFromStringPointerV0(projectConfig.TelegramOperator.BotLinkRef)
	chatsSource := configFileSettingSourceFromStringSlicePointerV0(projectConfig.TelegramOperator.AuthorizedChatRefs)
	confirmationSource := configFileSettingSourceFromBoolPointerV0(projectConfig.TelegramOperator.RequireConfirmation)
	targetSource := telegramOperatorNotificationTargetSourceV0(projectConfig)
	return []orquestaserver.ServerConfigSettingV0{
		telegramOperatorConfigFileSettingV0(
			telegramOperatorConfigKeyEnabledV0,
			strconv.FormatBool(telegramOperatorEnabledFromProjectConfigFileV0(projectConfig)),
			enabledSource,
			"Operador Telegram",
			"Activa el adaptador operador Telegram opt-in reutilizando un bot existente.",
			false,
		),
		telegramOperatorConfigFileSettingV0(
			telegramOperatorConfigKeyBotLinkRefV0,
			sensitiveConfigValueFromSourceV0(
				telegramOperatorBotLinkRefFromProjectConfigFileV0(projectConfig),
				telegramOperatorBotLinkConfiguredRefV0,
				botLinkSource,
			),
			botLinkSource,
			"Ref enlace bot",
			"Ref opaca del enlace/configuracion existente del bot; se configura por fichero canonico.",
			true,
		),
		telegramOperatorConfigFileSettingV0(
			telegramOperatorConfigKeyTokenV0,
			sensitiveConfigValueFromSourceV0(
				telegramOperatorTokenFromProjectConfigFileV0(projectConfig),
				telegramOperatorTokenConfiguredRefV0,
				tokenSource,
			),
			tokenSource,
			"Token Telegram",
			"Presencia del token del bot existente; el valor real nunca se publica.",
			true,
		),
		telegramOperatorConfigFileSettingV0(
			telegramOperatorConfigKeyAuthorizedChatRefsV0,
			sensitiveConfigValueFromSourceV0(
				strings.Join(telegramOperatorAuthorizedChatRefsFromProjectConfigFileV0(projectConfig), ","),
				telegramOperatorChatRefsConfiguredV0,
				chatsSource,
			),
			chatsSource,
			"Chats autorizados Telegram",
			"Refs opacas de chats autorizados para operar Orquesta; se configuran por fichero canonico.",
			true,
		),
		telegramOperatorConfigFileSettingV0(
			telegramOperatorConfigKeyRequireConfirmationV0,
			strconv.FormatBool(telegramOperatorRequireConfirmationFromProjectConfigFileV0(projectConfig)),
			confirmationSource,
			"Confirmacion Telegram",
			"Exige confirmacion explicita en comandos de detener/controlar.",
			false,
		),
		telegramOperatorConfigFileSettingV0(
			telegramOperatorConfigKeyNotificationTargetV0,
			sensitiveConfigValueFromSourceV0(
				telegramOperatorNotificationTargetFromProjectConfigFileV0(projectConfig),
				"telegram-operator-notification-target-configured",
				targetSource,
			),
			targetSource,
			"Target notificaciones Telegram",
			"Ref opaca del chat destino para avisos terminales deduplicados.",
			true,
		),
	}
}

func telegramOperatorConfigFileSettingV0(
	key string,
	value string,
	source string,
	label string,
	description string,
	sensitive bool,
) orquestaserver.ServerConfigSettingV0 {
	setting := orquestaserver.ServerConfigSettingV0{
		Key:             key,
		Value:           value,
		Source:          strings.TrimSpace(source),
		Scope:           "telegram_operator",
		Label:           label,
		Description:     description,
		RestartBehavior: "restart_required",
		Editable:        true,
		Canonical:       true,
		Sensitive:       sensitive,
	}
	if setting.Source == "" {
		setting.Source = "defaulted"
	}
	return setting
}

func stringProjectConfigFileOrDefaultV0(fileValue *string, fallback string) string {
	if fileValue != nil {
		if value := strings.TrimSpace(*fileValue); value != "" {
			return value
		}
	}
	return fallback
}

func stringSliceProjectConfigFileOrDefaultV0(fileValue *[]string, fallback []string) []string {
	if fileValue == nil {
		return fallback
	}
	out := make([]string, 0, len(*fileValue))
	for _, raw := range *fileValue {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func boolProjectConfigFileOrDefaultV0(fileValue *bool, fallback bool) bool {
	if fileValue != nil {
		return *fileValue
	}
	return fallback
}

func configFileSettingSourceFromStringPointerV0(value *string) string {
	if configStringPointerHasValueV0(value) {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}

func configFileSettingSourceFromStringSlicePointerV0(value *[]string) string {
	if configStringSlicePointerHasValueV0(value) {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}

func configFileSettingSourceFromBoolPointerV0(value *bool) string {
	if value != nil {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}

func telegramOperatorNotificationTargetSourceV0(config serverProjectConfigFileV0) string {
	if configStringPointerHasValueV0(config.TelegramOperator.NotificationTargetRef) {
		return configSettingSourceConfigFileV0
	}
	if configStringSlicePointerHasValueV0(config.TelegramOperator.AuthorizedChatRefs) {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}
