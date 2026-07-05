package main

import (
	"context"
	"strings"

	notifications "orquesta/modulos/orquesta-operator-notifications"
)

type hermesTelegramSendPortV0 interface {
	SendTelegramMessageV0(context.Context, hermesTelegramMessageV0) (string, error)
}

type hermesTelegramMessageV0 struct {
	TargetRef    string   `json:"target_ref"`
	Text         string   `json:"text"`
	DedupeKey    string   `json:"dedupe_key"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type hermesTelegramNotifierV0 struct {
	Sender hermesTelegramSendPortV0
}

func (notifier hermesTelegramNotifierV0) SendOperatorNotificationV0(
	ctx context.Context,
	message notifications.MessageV0,
) (string, error) {
	if notifier.Sender == nil {
		return "", nil
	}
	return notifier.Sender.SendTelegramMessageV0(ctx, hermesTelegramMessageV0{
		TargetRef:    strings.TrimSpace(message.TargetRef),
		Text:         strings.TrimSpace(message.Text),
		DedupeKey:    strings.TrimSpace(message.DedupeKey),
		EvidenceRefs: message.EvidenceRefs,
	})
}
