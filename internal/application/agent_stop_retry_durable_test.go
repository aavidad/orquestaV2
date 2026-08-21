package application

import (
	"context"
	"testing"
)

func TestDefinitelyNotAppliedStopBuildsNeutralRequeueWithoutSettlement(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	created, err := system.orchestrator.Control(context.Background(), system.access,
		system.request(t, "control:durable-stop-retry", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref))
	if err != nil || !created.Created {
		t.Fatalf("control: result=%+v err=%v", created, err)
	}
	claim := system.claim(t, "worker:durable-stop-retry")
	attempt, err := system.orchestrator.beginNewEffectAttempt(context.Background(), claim, system.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	capture := &stopOutcomeCapture{StateRepository: system.repository}
	system.orchestrator.state = capture
	if err := system.orchestrator.requeueDefinitelyUnappliedStop(
		context.Background(), claim, execution, attempt, "agent.stop_definitely_not_applied",
	); err != nil {
		t.Fatal(err)
	}
	state := capture.state
	if state.EffectAttemptOutcome == nil || state.BudgetSettlement != nil || state.ClearEffectBinding ||
		ValidateEffectAttemptOutcome(attempt, *state.EffectAttemptOutcome) != nil {
		t.Fatalf("non-application requeue=%+v", state)
	}
	if err := system.repository.RequeueAction(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	durable := system.record(t)
	if len(durable.EffectAttemptOutcomes) != 1 ||
		durable.EffectAttemptOutcomes[0] != *state.EffectAttemptOutcome {
		t.Fatalf("durable outcomes=%+v", durable.EffectAttemptOutcomes)
	}
	replayed := durable
	if !effectAttemptDefinitelyUnapplied(replayed, attempt) {
		t.Fatal("exact durable outcome did not unblock its attempt")
	}
	receipt := EffectReceipt{AttemptRef: attempt.Ref, ActionRef: attempt.ActionRef,
		IntentRef: attempt.IntentRef, ActionFence: attempt.ActionFence}
	replayed.EffectReceipts = append(replayed.EffectReceipts, receipt)
	if effectAttemptDefinitelyUnapplied(replayed, attempt) {
		t.Fatal("outcome coexisting with a receipt unblocked an attempt")
	}
	replayed.EffectReceipts = nil
	forged := *state.EffectAttemptOutcome
	forged.ActionFence++
	replayed.EffectAttemptOutcomes[0] = forged
	if effectAttemptDefinitelyUnapplied(replayed, attempt) {
		t.Fatal("crossed durable outcome unblocked an attempt")
	}
	replayed.EffectAttemptOutcomes = []EffectAttemptOutcome{*state.EffectAttemptOutcome, *state.EffectAttemptOutcome}
	if effectAttemptDefinitelyUnapplied(replayed, attempt) {
		t.Fatal("duplicate durable outcomes unblocked an attempt")
	}
	future := *state.EffectAttemptOutcome
	future.ObservedAt = attempt.ClaimLeaseUntil.Add(1)
	replayed.EffectAttemptOutcomes = []EffectAttemptOutcome{future}
	if effectAttemptDefinitelyUnapplied(replayed, attempt) {
		t.Fatal("future durable outcome unblocked an attempt")
	}
}

type stopOutcomeCapture struct {
	StateRepository
	state ActionRequeuedState
}

func (capture *stopOutcomeCapture) RequeueAction(_ context.Context, state ActionRequeuedState) error {
	capture.state = state
	return nil
}
