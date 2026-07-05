package main

import (
	"context"
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
