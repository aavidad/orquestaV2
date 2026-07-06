package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTelegramOperatorConfigV0LeeFicheroYRedactaEffectiveConfig(t *testing.T) {
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	raw := `{
		"schema_version":"orquesta_config.v0",
		"telegram_operator":{
			"enabled":true,
			"bot_link_ref":"inodo-bot-link-ref-real",
			"token":"123456:secret",
			"authorized_chat_refs":["chat-ref-alberto"],
			"notification_target_ref":"telegram:39995054",
			"require_confirmation":true
		}
	}`
	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 without project: %v", err)
	}
	config.ProjectWorkDir = projectDir
	config.ProjectConfigFilePath = configPath
	config.EffectiveConfig = serverEffectiveConfigFromEnvV0(config)
	settings := config.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, envTelegramOperatorEnabledV0); got != "true" {
		t.Fatalf("enabled=%q", got)
	}
	for _, key := range []string{
		telegramOperatorConfigKeyBotLinkRefV0,
		envTelegramOperatorTokenV0,
		telegramOperatorConfigKeyAuthorizedChatRefsV0,
		telegramOperatorConfigKeyNotificationTargetV0,
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if !setting.Sensitive || strings.Contains(setting.Value, "secret") || strings.Contains(setting.Value, "alberto") {
			t.Fatalf("setting sensible no redactado para %s: %+v", key, setting)
		}
	}
	if got := effectiveSettingForTestV0(settings, telegramOperatorConfigKeyRequireConfirmationV0); got.Source != configSettingSourceConfigFileV0 || got.Value != "true" {
		t.Fatalf("require_confirmation canonico inesperado: %+v", got)
	}
}

func TestTelegramOperatorWiringV0BloqueaSiFaltaEnlaceInodo(t *testing.T) {
	enabled := true
	config := serverProjectConfigFileV0{
		TelegramOperator: serverProjectConfigTelegramOperatorV0{
			Enabled: &enabled,
		},
	}
	result := telegramOperatorAdapterFromProjectConfigFileV0(config, fakeTelegramOperatorPortsV0{})
	if !result.Enabled || result.Ready || result.BlockedCode != "telegram_inodo_bot_link_missing" {
		t.Fatalf("wiring inesperado: %+v", result)
	}
	if strings.Join(result.MissingFields, ",") != "bot_link_ref,token,authorized_chat_refs" {
		t.Fatalf("missing fields inesperados: %+v", result.MissingFields)
	}
}

func TestTelegramOperatorWiringV0CreaAdaptadorConInodoConfigurado(t *testing.T) {
	enabled := true
	requireConfirmation := true
	botLink := "inodo-bot-link-ref"
	token := "token-real"
	chats := []string{"chat-ref-alberto"}
	config := serverProjectConfigFileV0{
		TelegramOperator: serverProjectConfigTelegramOperatorV0{
			Enabled:             &enabled,
			BotLinkRef:          &botLink,
			Token:               &token,
			AuthorizedChatRefs:  &chats,
			RequireConfirmation: &requireConfirmation,
		},
	}
	result := telegramOperatorAdapterFromProjectConfigFileV0(config, fakeTelegramOperatorPortsV0{})
	if !result.Enabled || !result.Ready {
		t.Fatalf("wiring no listo: %+v", result)
	}
}

type fakeTelegramOperatorPortsV0 struct{}

func (fakeTelegramOperatorPortsV0) QueryStatusV0(string) (string, []string, error) {
	return "status ok", nil, nil
}
func (fakeTelegramOperatorPortsV0) QueryQueueV0(string) (string, []string, error) {
	return "queue ok", nil, nil
}
func (fakeTelegramOperatorPortsV0) LaunchTaskV0(string, []string) (string, []string, error) {
	return "launch ok", nil, nil
}
func (fakeTelegramOperatorPortsV0) ObserveGoalV0(string) (string, []string, error) {
	return "goal ok", nil, nil
}
func (fakeTelegramOperatorPortsV0) SendDirectorMessageV0(string, string, []string) (string, []string, error) {
	return "director message ok", nil, nil
}
func (fakeTelegramOperatorPortsV0) StopV0(string, []string) (string, []string, error) {
	return "stop ok", nil, nil
}
func (fakeTelegramOperatorPortsV0) HandoffV0(string) (string, []string, error) {
	return "handoff ok", nil, nil
}
