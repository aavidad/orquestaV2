package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestatelegram "orquesta/modulos/orquesta-operator-telegram"
)

func TestTelegramOperatorUpdateHTTPV0DespachaUpdateAutorizadoSinLLM(t *testing.T) {
	adapter, issues := orquestatelegram.NewAdapterV0(orquestatelegram.ConfigV0{
		Enabled:             true,
		BotLinkRef:          "inodo-bot-link-ref",
		TokenConfigured:     true,
		AuthorizedChatRefs:  []string{"telegram:39995054"},
		RequireConfirmation: true,
	}, fakeTelegramOperatorPortsV0{})
	if len(issues) > 0 {
		t.Fatalf("adapter issues: %+v", issues)
	}
	sender := &fakeTelegramOperatorHTTPSenderV0{}
	handler := newTelegramOperatorUpdateHTTPHandlerV0(telegramOperatorWiringResultV0{
		Enabled: true,
		Ready:   true,
		Adapter: adapter,
	}, sender)
	req := httptest.NewRequest(http.MethodPost, telegramOperatorUpdateHTTPPathV0, strings.NewReader(`{
		"update_id":123,
		"message":{"chat":{"id":39995054},"text":"/status run-ref-telegram"}
	}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response telegramOperatorHTTPResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v\n%s", err, rec.Body.String())
	}
	if response.Estado != "accepted" || response.Response.CommandKind != orquestatelegram.CommandStatusV0 {
		t.Fatalf("response inesperada: %+v", response)
	}
	if !response.Sent || sender.message.TargetRef != "telegram:39995054" ||
		!strings.Contains(sender.message.Text, "status ok") {
		t.Fatalf("send inesperado: sent=%v message=%+v", response.Sent, sender.message)
	}
	if response.Update.UpdateRef != "telegram-update-123" {
		t.Fatalf("update ref=%q", response.Update.UpdateRef)
	}
}

func TestTelegramOperatorUpdateHTTPV0BloqueoConfigVisible(t *testing.T) {
	handler := newTelegramOperatorUpdateHTTPHandlerV0(telegramOperatorWiringResultV0{
		Enabled:       true,
		Ready:         false,
		BlockedCode:   orquestatelegram.ErrTelegramLinkMissingV0,
		MissingFields: []string{"bot_link_ref", "token", "authorized_chat_refs"},
	}, nil)
	req := httptest.NewRequest(http.MethodPost, telegramOperatorUpdateHTTPPathV0, strings.NewReader(`{
		"update_id":124,
		"message":{"chat":{"id":39995054},"text":"/status"}
	}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response telegramOperatorHTTPResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v\n%s", err, rec.Body.String())
	}
	if response.Estado != "blocked" ||
		response.ErrorCode != orquestatelegram.ErrTelegramLinkMissingV0 ||
		strings.Join(response.Missing, ",") != "bot_link_ref,token,authorized_chat_refs" {
		t.Fatalf("response inesperada: %+v", response)
	}
}

type fakeTelegramOperatorHTTPSenderV0 struct {
	message hermesTelegramMessageV0
}

func (fake *fakeTelegramOperatorHTTPSenderV0) SendTelegramMessageV0(
	_ context.Context,
	message hermesTelegramMessageV0,
) (string, error) {
	fake.message = message
	return "evidence-ref-telegram-http-send", nil
}
