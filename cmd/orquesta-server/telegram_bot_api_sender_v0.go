package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const telegramBotAPIDefaultBaseURLV0 = "https://api.telegram.org"

type telegramBotAPIHTTPDoerV0 interface {
	Do(*http.Request) (*http.Response, error)
}

type telegramBotAPISenderV0 struct {
	Token       string
	HTTPClient  telegramBotAPIHTTPDoerV0
	BaseURL     string
	MaxBodyRead int64
}

func telegramBotAPISenderFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
) hermesTelegramSendPortV0 {
	token := strings.TrimSpace(telegramOperatorTokenFromProjectConfigFileV0(config))
	if token == "" {
		return nil
	}
	return telegramBotAPISenderV0{
		Token:       token,
		HTTPClient:  http.DefaultClient,
		BaseURL:     telegramBotAPIDefaultBaseURLV0,
		MaxBodyRead: 4096,
	}
}

func (sender telegramBotAPISenderV0) SendTelegramMessageV0(
	ctx context.Context,
	message hermesTelegramMessageV0,
) (string, error) {
	token := strings.TrimSpace(sender.Token)
	if token == "" {
		return "", errors.New("telegram_bot_api_token_missing")
	}
	chatID := telegramBotAPIChatIDFromTargetRefV0(message.TargetRef)
	if chatID == "" {
		return "", errors.New("telegram_bot_api_chat_ref_invalid")
	}
	text := strings.TrimSpace(message.Text)
	if text == "" {
		return "", errors.New("telegram_bot_api_text_missing")
	}
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", errors.New("telegram_bot_api_payload_invalid")
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(telegramBotAPIBaseURLV0(sender.BaseURL), "/")+"/bot"+token+"/sendMessage",
		bytes.NewReader(data),
	)
	if err != nil {
		return "", errors.New("telegram_bot_api_request_invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	client := sender.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("telegram_bot_api_send_failed")
	}
	defer resp.Body.Close()
	maxBodyRead := sender.MaxBodyRead
	if maxBodyRead <= 0 {
		maxBodyRead = 4096
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyRead))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", errors.New("telegram_bot_api_send_rejected")
	}
	return "evidence-ref-telegram-bot-api-send-" +
		telegramBotAPISafeEvidenceSuffixV0(firstNonEmptyOperatorNotificationV0(message.DedupeKey, message.TargetRef)), nil
}

func telegramBotAPIBaseURLV0(value string) string {
	if strings.TrimSpace(value) == "" {
		return telegramBotAPIDefaultBaseURLV0
	}
	return strings.TrimSpace(value)
}

func telegramBotAPIChatIDFromTargetRefV0(targetRef string) string {
	targetRef = strings.TrimSpace(targetRef)
	if targetRef == "" {
		return ""
	}
	if strings.HasPrefix(targetRef, "telegram:") {
		return strings.TrimSpace(strings.TrimPrefix(targetRef, "telegram:"))
	}
	return ""
}

func telegramBotAPISafeEvidenceSuffixV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			out.WriteByte('-')
		default:
			out.WriteByte('-')
		}
	}
	suffix := strings.Trim(out.String(), "-")
	for strings.Contains(suffix, "--") {
		suffix = strings.ReplaceAll(suffix, "--", "-")
	}
	if suffix == "" {
		return "message"
	}
	if len(suffix) > 80 {
		return suffix[:80]
	}
	return suffix
}
