package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestMalformedLaunchReceiptReconcilesWithSameEffectKey(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 18, 16, 30, 0, 0, time.UTC)}
	physical, returnedMalformed := 0, false
	var durable ports.AgentLaunchReceipt
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("reconciled receipt"),
	}}}
	agent.launchOverride = func(request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
		if durable.ReceiptRef == "" {
			physical++
			durable = launchReceiptForRequest(request, clock.Now())
		}
		if !returnedMalformed {
			returnedMalformed = true
			malformed := durable
			malformed.SpecHash = ""
			return malformed, nil
		}
		return durable, nil
	}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:malformed-effect-receipt", Statement: "reconcile malformed receipt", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:malformed"); err == nil || err.Error() != effectUnknownAppliedCode {
		t.Fatalf("malformed receipt error=%v want=%s", err, effectUnknownAppliedCode)
	}
	intermediate, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || intermediate.Goal.State() != goal.GoalStateRunning || physical != 1 ||
		len(intermediate.EffectAttempts) != 1 || len(intermediate.EffectReceipts) != 0 ||
		len(intermediate.BudgetReservations) != 1 || len(intermediate.BudgetSettlements) != 0 {
		t.Fatalf("malformed receipt became terminal: record=%+v physical=%d err=%v", intermediate, physical, err)
	}
	clock.Advance(time.Second)
	if result, err := orchestrator.ProcessNext(context.Background(), "worker:receipt-reconcile"); err != nil || result.Processed {
		t.Fatalf("malformed receipt replayed: result=%+v err=%v", result, err)
	}
	closed, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	agent.mu.Lock()
	requests := append([]ports.AgentLaunchRequest(nil), agent.launchRequests...)
	agent.mu.Unlock()
	if err != nil || physical != 1 || len(requests) != 1 ||
		len(closed.EffectAttempts) != 1 || len(closed.EffectReceipts) != 0 ||
		len(closed.BudgetReservations) != 1 || len(closed.BudgetSettlements) != 0 {
		t.Fatalf("receipt reconciliation diverged: physical=%d requests=%+v record=%+v err=%v",
			physical, requests, closed, err)
	}
}

func TestMalformedStopReceiptReconcilesWithSameEffectKey(t *testing.T) {
	physical, returnedMalformed := 0, false
	var durable ports.AgentStopReceipt
	agent := &scriptedAgent{}
	agent.stopOverride = func(request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
		if durable.ReceiptRef == "" {
			physical++
			durable = ports.AgentStopReceipt{
				ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
				PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
				ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
				ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
				ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
				Status: ports.AgentStopped, ReceiptRef: "receipt:stop:reconciled", ConfirmedAt: agent.now().UTC(),
			}
		}
		if !returnedMalformed {
			returnedMalformed = true
			malformed := durable
			malformed.SpecHash = ""
			return malformed, nil
		}
		return durable, nil
	}
	system := newControlTestSystem(t, agent)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	control, err := system.orchestrator.Control(context.Background(), system.access, system.request(
		t, "control:malformed-stop-receipt", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
	))
	if err != nil || !control.Created {
		t.Fatalf("create stop: result=%+v err=%v", control, err)
	}
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:malformed-stop"); err == nil || err.Error() != effectUnknownAppliedCode || !result.Processed || result.Action != ActionStopAgent {
		t.Fatalf("malformed stop call: result=%+v err=%v", result, err)
	}
	intermediate := system.record(t)
	if physical != 1 || intermediate.Goal.IsTerminal() || len(intermediate.EffectReceipts) != 1 {
		t.Fatalf("malformed stop receipt became terminal: physical=%d record=%+v", physical, intermediate)
	}
	system.clock.Advance(time.Second)
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:stop-reconcile"); err != nil ||
		(result.Processed && result.Action == ActionStopAgent) {
		t.Fatalf("quarantined stop replayed: result=%+v err=%v", result, err)
	}
	closed := system.record(t)
	agent.mu.Lock()
	requests := append([]ports.AgentStopRequest(nil), agent.stopRequests...)
	agent.mu.Unlock()
	if physical != 1 || len(requests) != 1 || len(closed.EffectReceipts) != 1 ||
		onlyExecution(t, closed).State != ExecutionRunning {
		t.Fatalf("stop reconciliation diverged: physical=%d requests=%+v record=%+v", physical, requests, closed)
	}
}
