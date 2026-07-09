package orquestaoperatornotifications

import (
	"context"
	"errors"
	"testing"
)

func TestOperatorNotificationV0EnviaTerminalDeduplicado(t *testing.T) {
	sender := &fakeSenderV0{}
	service := ServiceV0{
		TargetRef: "telegram:39995054",
		Sender:    sender,
		Dedupe:    NewMemoryDedupeStoreV0(),
	}
	event := TerminalEventV0{
		Kind:         EventKindGoalV0,
		SubjectRef:   "goal-ref-1",
		Status:       StatusCompleteV0,
		Summary:      "cierre verificado",
		EvidenceRefs: []string{"evidence-ref-test"},
	}
	message, sent, err := NotifyTerminalEventV0(context.Background(), service, event)
	if err != nil || !sent || sender.calls != 1 {
		t.Fatalf("sent=%v err=%v calls=%d", sent, err, sender.calls)
	}
	if message.TargetRef != "telegram:39995054" || message.DedupeKey != "goal:goal-ref-1:complete" {
		t.Fatalf("message inesperado: %+v", message)
	}
	_, sent, err = NotifyTerminalEventV0(context.Background(), service, event)
	if err != nil || sent || sender.calls != 1 {
		t.Fatalf("dedupe roto sent=%v err=%v calls=%d", sent, err, sender.calls)
	}
}

func TestOperatorNotificationV0IgnoraNoTerminal(t *testing.T) {
	sender := &fakeSenderV0{}
	_, sent, err := NotifyTerminalEventV0(context.Background(), ServiceV0{Sender: sender}, TerminalEventV0{
		Kind:       EventKindRunV0,
		SubjectRef: "run-ref-1",
		Status:     "running",
	})
	if err != nil || sent || sender.calls != 0 {
		t.Fatalf("sent=%v err=%v calls=%d", sent, err, sender.calls)
	}
}

func TestOperatorNotificationV0NoDeduplicaSiSenderFallaYPermiteRetry(t *testing.T) {
	sender := &fakeSenderV0{failFirst: true}
	service := ServiceV0{
		TargetRef: "telegram:39995054",
		Sender:    sender,
		Dedupe:    NewMemoryDedupeStoreV0(),
	}
	event := TerminalEventV0{
		Kind:       EventKindProviderV0,
		SubjectRef: "codex_app_server_provider_unauthorized",
		RunRef:     "run-ref-provider-auth-001",
		Status:     StatusBlockedV0,
		Summary:    "provider_unavailable_paused",
	}
	_, sent, err := NotifyTerminalEventV0(context.Background(), service, event)
	if err == nil || sent || sender.calls != 1 {
		t.Fatalf("primer envio sent=%v err=%v calls=%d", sent, err, sender.calls)
	}
	message, sent, err := NotifyTerminalEventV0(context.Background(), service, event)
	if err != nil || !sent || sender.calls != 2 {
		t.Fatalf("retry sent=%v err=%v calls=%d", sent, err, sender.calls)
	}
	if message.DedupeKey != "provider:codex_app_server_provider_unauthorized:blocked" ||
		!containsNotificationEvidenceForTestV0(message.EvidenceRefs, "evidence-ref-fake-send") {
		t.Fatalf("message=%+v", message)
	}
}

type fakeSenderV0 struct {
	calls     int
	failFirst bool
}

func (fake *fakeSenderV0) SendOperatorNotificationV0(context.Context, MessageV0) (string, error) {
	fake.calls++
	if fake.failFirst && fake.calls == 1 {
		return "", errors.New("telegram temporalmente no disponible")
	}
	return "evidence-ref-fake-send", nil
}

func containsNotificationEvidenceForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
