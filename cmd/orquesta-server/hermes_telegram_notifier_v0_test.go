package main

import (
	"context"
	"testing"

	notifications "orquesta/modulos/orquesta-operator-notifications"
)

func TestOperatorNotificationHermesTelegramNotifierV0UsaSendSinAuthCodex(t *testing.T) {
	sender := &fakeHermesTelegramSendV0{}
	notifier := hermesTelegramNotifierV0{Sender: sender}
	receipt, err := notifier.SendOperatorNotificationV0(context.Background(), notifications.MessageV0{
		TargetRef:    "telegram:39995054",
		Text:         "Orquesta | goal | goal-ref-1 | complete",
		DedupeKey:    "goal:goal-ref-1:complete",
		EvidenceRefs: []string{"evidence-ref-1"},
	})
	if err != nil || receipt != "evidence-ref-hermes-send" {
		t.Fatalf("receipt=%q err=%v", receipt, err)
	}
	if sender.message.TargetRef != "telegram:39995054" ||
		sender.message.DedupeKey != "goal:goal-ref-1:complete" {
		t.Fatalf("message inesperado: %+v", sender.message)
	}
}

type fakeHermesTelegramSendV0 struct {
	message hermesTelegramMessageV0
}

func (fake *fakeHermesTelegramSendV0) SendTelegramMessageV0(
	_ context.Context,
	message hermesTelegramMessageV0,
) (string, error) {
	fake.message = message
	return "evidence-ref-hermes-send", nil
}
