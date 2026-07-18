package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestControlsForcedStopSupersedesOnlyExactPendingCooperativeStop(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	controller := &pendingThenForcedController{now: system.clock.Now}
	system.orchestrator.controller = controller
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())

	cooperative := system.request(
		t, "control:cooperative-owner", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
	)
	first, err := system.orchestrator.Control(context.Background(), system.access, cooperative)
	if err != nil || first.Control.Status != ControlRequested {
		t.Fatalf("cooperative request: result=%+v err=%v", first, err)
	}
	if result, processErr := system.orchestrator.ProcessNext(context.Background(), "worker:cooperative-pending"); processErr != nil || !result.Processed || result.Action != ActionStopAgent {
		t.Fatalf("cooperative pending: result=%+v err=%v", result, processErr)
	}
	for _, control := range []struct {
		ref       string
		operation ControlOperation
	}{
		{ref: "control:pause-between-stops", operation: ControlPause},
		{ref: "control:resume-between-stops", operation: ControlResume},
	} {
		request := system.request(
			t, control.ref, control.operation, ControlTargetGoal,
			goal.WorkItemRef{}, goal.ExecutionRef{},
		)
		if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
			t.Fatalf("%s: %v", control.ref, err)
		}
	}

	blocked := system.request(
		t, "control:second-cooperative", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
	)
	if _, err := system.orchestrator.Control(context.Background(), system.access, blocked); !IsStateError(err, StateConflict) {
		t.Fatalf("second cooperative owner error=%v", err)
	}

	forced := system.request(
		t, "control:forced-escalation", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
	)
	forced.Mode = ports.AgentStopForced
	escalated, err := system.orchestrator.Control(context.Background(), system.access, forced)
	if err != nil || !escalated.Created || escalated.Control.Status != ControlRequested ||
		escalated.Control.SupersedesControlRef != first.Control.Ref {
		t.Fatalf("forced escalation: result=%+v err=%v", escalated, err)
	}
	transferred := system.record(t)
	old := mustControlByRequest(t, transferred, cooperative.RequestRef)
	if old.Status != ControlSuperseded || old.SupersededByControlRef != escalated.Control.Ref ||
		old.SupersededAt != escalated.Control.RequestedAt || system.actionKindCount(ActionStopAgent) != 1 {
		t.Fatalf("atomic transfer old=%+v active_stops=%d", old, system.actionKindCount(ActionStopAgent))
	}
	approveForcedStop(t, system, "approval:forced-escalation")
	if result, processErr := system.orchestrator.ProcessNext(context.Background(), "worker:forced-escalation"); processErr != nil || !result.Processed || result.Action != ActionStopAgent {
		t.Fatalf("forced stop: result=%+v err=%v", result, processErr)
	}

	settled := system.record(t)
	old = mustControlByRequest(t, settled, cooperative.RequestRef)
	next := mustControlByRequest(t, settled, forced.RequestRef)
	current, _ := executionByRef(settled.Executions, execution.Ref)
	if old.Status != ControlSuperseded || next.Status != ControlConfirmed ||
		next.SupersedesControlRef != old.Ref || current.State != ExecutionStopped ||
		system.actionKindCount(ActionStopAgent) != 0 {
		t.Fatalf("settled lineage old=%+v next=%+v execution=%s actions=%d",
			old, next, current.State, system.actionKindCount(ActionStopAgent))
	}
	requests, physical := controller.snapshot()
	if len(requests) != 2 || requests[0].Mode != ports.AgentStopCooperative ||
		requests[1].Mode != ports.AgentStopForced || requests[0].IdempotencyKey == requests[1].IdempotencyKey ||
		physical != 1 {
		t.Fatalf("controller requests=%+v physical=%d", requests, physical)
	}

	oldReplay, err := system.orchestrator.Control(context.Background(), system.access, cooperative)
	if err != nil || oldReplay.Created || oldReplay.Control.Status != ControlSuperseded {
		t.Fatalf("old replay: result=%+v err=%v", oldReplay, err)
	}
	newReplay, err := system.orchestrator.Control(context.Background(), system.access, forced)
	if err != nil || newReplay.Created || newReplay.Control.Status != ControlConfirmed {
		t.Fatalf("new replay: result=%+v err=%v", newReplay, err)
	}
	afterRequests, afterPhysical := controller.snapshot()
	if len(afterRequests) != len(requests) || afterPhysical != physical {
		t.Fatalf("replay repeated effect requests=%d/%d physical=%d/%d",
			len(requests), len(afterRequests), physical, afterPhysical)
	}
}

func TestControlsCancelPrefersCooperativeStopCapability(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	request := system.request(
		t, "control:cancel-forced-first", ControlCancel, ControlTargetWorkItem,
		item.Ref(), goal.ExecutionRef{},
	)
	result, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || result.Control.Mode != ports.AgentStopCooperative {
		t.Fatalf("cancel mode=%q err=%v", result.Control.Mode, err)
	}
}

func TestControlsForcedEscalationSettlesWhenCooperativeAlreadyStoppedTarget(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	controller := &pendingThenForcedController{
		now: system.clock.Now, forcedStatus: ports.AgentStopAlreadyStopped,
	}
	system.orchestrator.controller = controller
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())

	cooperative := system.request(
		t, "control:cooperative-already-stopped", ControlStop, ControlTargetExecution,
		item.Ref(), execution.Ref,
	)
	first, err := system.orchestrator.Control(context.Background(), system.access, cooperative)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:cooperative-already-stopped"); err != nil {
		t.Fatal(err)
	}
	forced := system.request(
		t, "control:forced-already-stopped", ControlStop, ControlTargetExecution,
		item.Ref(), execution.Ref,
	)
	forced.Mode = ports.AgentStopForced
	next, err := system.orchestrator.Control(context.Background(), system.access, forced)
	if err != nil {
		t.Fatal(err)
	}
	approveForcedStop(t, system, "approval:forced-already-stopped")
	if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:forced-already-stopped"); err != nil {
		t.Fatal(err)
	}

	settled := system.record(t)
	old := mustControlByRequest(t, settled, cooperative.RequestRef)
	escalated := mustControlByRequest(t, settled, forced.RequestRef)
	current, _ := executionByRef(settled.Executions, execution.Ref)
	requests, physical := controller.snapshot()
	if old.Ref != first.Control.Ref || old.Status != ControlSuperseded ||
		escalated.Ref != next.Control.Ref || escalated.Status != ControlConfirmed ||
		current.State != ExecutionStopped || len(requests) != 2 || physical != 0 {
		t.Fatalf("already-stopped lineage old=%+v next=%+v execution=%s requests=%d physical=%d",
			old, escalated, current.State, len(requests), physical)
	}
}

func approveForcedStop(t *testing.T, system *controlTestSystem, requestRef string) {
	t.Helper()
	record := system.record(t)
	var intent EffectIntent
	for _, candidate := range record.EffectIntents {
		if candidate.Kind == EffectKindAgentStop && candidate.SecurityCriticality == governance.SecurityCriticalitySensitive {
			intent = candidate
		}
	}
	if intent.Ref == "" {
		t.Fatal("forced stop intent missing")
	}
	result, err := system.orchestrator.DecideEffect(context.Background(), system.access, DecideEffectRequest{
		RequestRef: requestRef, GoalRef: record.Goal.Ref(), IntentRef: intent.Ref,
		ExpectedIntentDigest: intent.Digest, Decision: EffectApproved, Reason: "owner forced stop approval",
	})
	if err != nil || !result.Created {
		t.Fatalf("forced stop approval=%+v err=%v", result, err)
	}
}

type pendingThenForcedController struct {
	mu           sync.Mutex
	now          func() time.Time
	requests     []ports.AgentStopRequest
	physical     int
	forcedStatus ports.AgentStopStatus
}

func (*pendingThenForcedController) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (controller *pendingThenForcedController) Stop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.requests = append(controller.requests, request)
	status := ports.AgentStopPending
	if request.Mode == ports.AgentStopForced {
		status = controller.forcedStatus
		if status == "" {
			status = ports.AgentStopped
		}
		if status == ports.AgentStopped {
			controller.physical++
		}
	}
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: status,
	}
	if status == ports.AgentStopped || status == ports.AgentStopAlreadyStopped {
		receipt.ReceiptRef = "receipt:forced:" + request.ExecutionRef.String()
		receipt.ConfirmedAt = controller.now().UTC()
	}
	return receipt, nil
}

func (controller *pendingThenForcedController) snapshot() ([]ports.AgentStopRequest, int) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return append([]ports.AgentStopRequest(nil), controller.requests...), controller.physical
}
