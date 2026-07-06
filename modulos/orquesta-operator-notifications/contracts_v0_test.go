package orquestaoperatornotifications

import (
	"context"
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

type fakeSenderV0 struct {
	calls int
}

func (fake *fakeSenderV0) SendOperatorNotificationV0(context.Context, MessageV0) (string, error) {
	fake.calls++
	return "evidence-ref-fake-send", nil
}
