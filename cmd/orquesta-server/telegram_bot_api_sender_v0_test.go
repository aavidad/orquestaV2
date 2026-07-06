package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTelegramBotAPISenderV0EnviaMensajeSinExponerTokenEnReceipt(t *testing.T) {
	var gotPath string
	var gotBody string
	sender := telegramBotAPISenderV0{
		Token:   "123456:secret-token",
		BaseURL: "https://telegram.example.test",
		HTTPClient: telegramBotAPIHTTPDoerFuncV0(func(req *http.Request) (*http.Response, error) {
			gotPath = req.URL.EscapedPath()
			data, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			gotBody = string(data)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			}, nil
		}),
	}

	sendRef, err := sender.SendTelegramMessageV0(context.Background(), hermesTelegramMessageV0{
		TargetRef: "telegram:39995054",
		Text:      "estado ok",
		DedupeKey: "task:goal-ref-001:complete",
	})

	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotPath != "/bot123456:secret-token/sendMessage" {
		t.Fatalf("path=%q", gotPath)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatalf("decode payload: %v\n%s", err, gotBody)
	}
	if payload["chat_id"] != "39995054" || payload["text"] != "estado ok" {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if strings.Contains(sendRef, "secret-token") {
		t.Fatalf("receipt filtra token: %q", sendRef)
	}
}

func TestTelegramBotAPISenderV0OcultaTokenEnError(t *testing.T) {
	sender := telegramBotAPISenderV0{
		Token:   "123456:secret-token",
		BaseURL: "https://telegram.example.test",
		HTTPClient: telegramBotAPIHTTPDoerFuncV0(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("upstream includes secret-token")
		}),
	}

	_, err := sender.SendTelegramMessageV0(context.Background(), hermesTelegramMessageV0{
		TargetRef: "telegram:39995054",
		Text:      "estado ok",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error filtra token: %v", err)
	}
}

func TestTelegramBotAPISenderFromProjectConfigFileV0EsOptInPorToken(t *testing.T) {
	enabled := true
	config := serverProjectConfigFileV0{
		TelegramOperator: serverProjectConfigTelegramOperatorV0{
			Enabled: &enabled,
		},
	}
	if got := telegramBotAPISenderFromProjectConfigFileV0(config); got != nil {
		t.Fatalf("sender sin token debe ser nil: %#v", got)
	}
	token := "123456:secret-token"
	config.TelegramOperator.Token = &token
	if got := telegramBotAPISenderFromProjectConfigFileV0(config); got == nil {
		t.Fatal("sender con token debe existir")
	}
}

type telegramBotAPIHTTPDoerFuncV0 func(*http.Request) (*http.Response, error)

func (fn telegramBotAPIHTTPDoerFuncV0) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}
