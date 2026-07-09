package main

import (
	"context"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	notifications "orquesta/modulos/orquesta-operator-notifications"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestOperatorNotificationServerV0NotificaGoalYRunTerminalDeduplicado(t *testing.T) {
	sender := &fakeOperatorNotificationSenderV0{}
	notifier := newOperatorTaskTerminalNotifierV0("telegram:39995054", sender)
	_, sent, err := notifier.NotifyGoalStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:     "run-ref-1",
		GoalRef:    "goal-ref-1",
		Status:     orquestagoal.GoalStatusCompleteV0,
		LastResult: &orquestagoal.GoalWorkResultV0{Summary: "ok"},
	})
	if err != nil || !sent {
		t.Fatalf("goal sent=%v err=%v", sent, err)
	}
	_, sent, err = notifier.NotifyGoalStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:  "run-ref-1",
		GoalRef: "goal-ref-1",
		Status:  orquestagoal.GoalStatusCompleteV0,
	})
	if err != nil || sent {
		t.Fatalf("goal dedupe sent=%v err=%v", sent, err)
	}
	_, sent, err = notifier.NotifyRunControlStateV0(context.Background(), orquestaruncontrol.RunControlStateV0{
		RunRef: "run-ref-1",
		Status: orquestaruncontrol.RunControlStatusStoppedV0,
	})
	if err != nil || !sent || sender.calls != 2 {
		t.Fatalf("run sent=%v err=%v calls=%d", sent, err, sender.calls)
	}
}

func TestOperatorNotificationServerV0NotificaProviderUnauthorizedUnaVezV0(t *testing.T) {
	sender := &fakeOperatorNotificationSenderV0{}
	notifier := newOperatorTaskTerminalNotifierV0("telegram:39995054", sender)
	message, sent, err := notifier.NotifyProviderIssueV0(
		context.Background(),
		"codex_app_server_provider_unauthorized",
		[]string{"run-ref-provider-unauthorized-001"},
		[]string{"evidence-ref-codex-app-server-provider-unauthorized"},
	)
	if err != nil || !sent {
		t.Fatalf("provider sent=%v err=%v", sent, err)
	}
	if message.DedupeKey != "provider:codex_app_server_provider_unauthorized:run-ref-provider-unauthorized-001" ||
		!containsOperatorNotificationTestV0(message.Text, "hermes auth") ||
		!containsOperatorNotificationTestV0(message.Text, "blocked") {
		t.Fatalf("mensaje provider inesperado: %+v", message)
	}
	_, sent, err = notifier.NotifyProviderIssueV0(
		context.Background(),
		"codex_app_server_provider_unauthorized",
		[]string{"run-ref-provider-unauthorized-001"},
		[]string{"evidence-ref-codex-app-server-provider-unauthorized"},
	)
	if err != nil || sent || sender.calls != 1 {
		t.Fatalf("dedupe provider sent=%v err=%v calls=%d", sent, err, sender.calls)
	}
}

type fakeOperatorNotificationSenderV0 struct {
	calls int
}

func (fake *fakeOperatorNotificationSenderV0) SendOperatorNotificationV0(
	_ context.Context,
	message notifications.MessageV0,
) (string, error) {
	fake.calls++
	if message.TargetRef != "telegram:39995054" || message.Text == "" {
		return "", nil
	}
	return "evidence-ref-server-notification", nil
}

func containsOperatorNotificationTestV0(value string, target string) bool {
	return strings.Contains(value, target)
}
