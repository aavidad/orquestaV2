package main

import (
	"os"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	telegramOperatorBotLinkConfiguredRefV0 = "telegram-operator-bot-link-configured"
	telegramOperatorTokenConfiguredRefV0   = "telegram-operator-token-configured"
	telegramOperatorChatRefsConfiguredV0   = "telegram-operator-chat-refs-configured"

	envTelegramOperatorEnabledV0             = "ORQUESTA_TELEGRAM_OPERATOR_ENABLED"
	envTelegramOperatorBotLinkRefV0          = "ORQUESTA_TELEGRAM_OPERATOR_BOT_LINK_REF"
	envTelegramOperatorTokenV0               = "ORQUESTA_TELEGRAM_OPERATOR_TOKEN"
	envTelegramOperatorAuthorizedChatRefsV0  = "ORQUESTA_TELEGRAM_OPERATOR_AUTHORIZED_CHAT_REFS"
	envTelegramOperatorRequireConfirmationV0 = "ORQUESTA_TELEGRAM_OPERATOR_REQUIRE_CONFIRMATION"
	envTelegramOperatorNotificationTargetV0  = "ORQUESTA_TELEGRAM_OPERATOR_NOTIFICATION_TARGET_REF"
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
	return boolProjectConfigOrEnvOrDefaultV0(
		envTelegramOperatorEnabledV0,
		config.TelegramOperator.Enabled,
		false,
	)
}

func telegramOperatorBotLinkRefFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envTelegramOperatorBotLinkRefV0,
		config.TelegramOperator.BotLinkRef,
		"",
	)
}

func telegramOperatorTokenFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	return stringProjectConfigOrEnvOrDefaultV0(
		envTelegramOperatorTokenV0,
		config.TelegramOperator.Token,
		"",
	)
}

func telegramOperatorAuthorizedChatRefsFromProjectConfigFileV0(config serverProjectConfigFileV0) []string {
	return stringSliceProjectConfigOrEnvOrDefaultV0(
		envTelegramOperatorAuthorizedChatRefsV0,
		config.TelegramOperator.AuthorizedChatRefs,
		nil,
	)
}

func telegramOperatorRequireConfirmationFromProjectConfigFileV0(config serverProjectConfigFileV0) bool {
	return boolProjectConfigOrEnvOrDefaultV0(
		envTelegramOperatorRequireConfirmationV0,
		config.TelegramOperator.RequireConfirmation,
		true,
	)
}

func telegramOperatorNotificationTargetFromProjectConfigFileV0(config serverProjectConfigFileV0) string {
	target := stringProjectConfigOrEnvOrDefaultV0(
		envTelegramOperatorNotificationTargetV0,
		config.TelegramOperator.NotificationTargetRef,
		"",
	)
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
	botLinkSource := telegramOperatorConfigSettingSourceV0(envTelegramOperatorBotLinkRefV0, projectConfig)
	tokenSource := telegramOperatorConfigSettingSourceV0(envTelegramOperatorTokenV0, projectConfig)
	chatsSource := telegramOperatorConfigSettingSourceV0(envTelegramOperatorAuthorizedChatRefsV0, projectConfig)
	targetSource := telegramOperatorConfigSettingSourceV0(envTelegramOperatorNotificationTargetV0, projectConfig)
	return []orquestaserver.ServerConfigSettingV0{
		serverConfigSettingFromRegistryWithSourceV0(
			envTelegramOperatorEnabledV0,
			strconv.FormatBool(telegramOperatorEnabledFromProjectConfigFileV0(projectConfig)),
			telegramOperatorConfigSettingSourceV0(envTelegramOperatorEnabledV0, projectConfig),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envTelegramOperatorBotLinkRefV0,
			sensitiveConfigValueFromSourceV0(
				telegramOperatorBotLinkRefFromProjectConfigFileV0(projectConfig),
				telegramOperatorBotLinkConfiguredRefV0,
				botLinkSource,
			),
			botLinkSource,
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envTelegramOperatorTokenV0,
			sensitiveConfigValueFromSourceV0(
				telegramOperatorTokenFromProjectConfigFileV0(projectConfig),
				telegramOperatorTokenConfiguredRefV0,
				tokenSource,
			),
			tokenSource,
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envTelegramOperatorAuthorizedChatRefsV0,
			sensitiveConfigValueFromSourceV0(
				strings.Join(telegramOperatorAuthorizedChatRefsFromProjectConfigFileV0(projectConfig), ","),
				telegramOperatorChatRefsConfiguredV0,
				chatsSource,
			),
			chatsSource,
		),
		serverConfigSettingFromRegistryWithSourceV0(
			envTelegramOperatorRequireConfirmationV0,
			strconv.FormatBool(telegramOperatorRequireConfirmationFromProjectConfigFileV0(projectConfig)),
			telegramOperatorConfigSettingSourceV0(envTelegramOperatorRequireConfirmationV0, projectConfig),
		),
		serverSensitiveConfigSettingFromRegistryWithSourceV0(
			envTelegramOperatorNotificationTargetV0,
			sensitiveConfigValueFromSourceV0(
				telegramOperatorNotificationTargetFromProjectConfigFileV0(projectConfig),
				"telegram-operator-notification-target-configured",
				targetSource,
			),
			targetSource,
		),
	}
}

func telegramOperatorConfigSettingSourceV0(key string, config serverProjectConfigFileV0) string {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return "explicit"
	}
	switch key {
	case envTelegramOperatorEnabledV0:
		if config.TelegramOperator.Enabled != nil {
			return configSettingSourceConfigFileV0
		}
	case envTelegramOperatorBotLinkRefV0:
		if configStringPointerHasValueV0(config.TelegramOperator.BotLinkRef) {
			return configSettingSourceConfigFileV0
		}
	case envTelegramOperatorTokenV0:
		if configStringPointerHasValueV0(config.TelegramOperator.Token) {
			return configSettingSourceConfigFileV0
		}
	case envTelegramOperatorAuthorizedChatRefsV0:
		if configStringSlicePointerHasValueV0(config.TelegramOperator.AuthorizedChatRefs) {
			return configSettingSourceConfigFileV0
		}
	case envTelegramOperatorRequireConfirmationV0:
		if config.TelegramOperator.RequireConfirmation != nil {
			return configSettingSourceConfigFileV0
		}
	case envTelegramOperatorNotificationTargetV0:
		if configStringPointerHasValueV0(config.TelegramOperator.NotificationTargetRef) {
			return configSettingSourceConfigFileV0
		}
	}
	return "defaulted"
}
