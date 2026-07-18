package application

import (
	"context"
	"testing"

	"orquesta/internal/goal"
)

func TestControlGoalCancelPersistsExactReceiptForEveryStoppedExecution(t *testing.T) {
	system := newControlTestSystemWithPlan(t, nil, controlIndependentPlan())
	system.launch(t)
	system.launch(t)
	running := system.record(t)
	if len(running.Executions) != 2 {
		t.Fatalf("live executions=%d", len(running.Executions))
	}
	request := system.request(t, "control:cancel-multiple-executions", ControlCancel,
		ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
	result, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || result.Control.Status != ControlRequested || system.actionKindCount(ActionStopAgent) != 2 {
		t.Fatalf("request multi cancel: result=%+v stop_actions=%d err=%v",
			result, system.actionKindCount(ActionStopAgent), err)
	}

	for index := 0; index < 2; index++ {
		processed, processErr := system.orchestrator.ProcessNext(context.Background(), "worker:multi-cancel")
		if processErr != nil || !processed.Processed || processed.Action != ActionStopAgent {
			t.Fatalf("stop %d: result=%+v err=%v", index+1, processed, processErr)
		}
		current := system.record(t)
		control := mustControlByRequest(t, current, request.RequestRef)
		if index == 0 && (control.Status != ControlRequested || current.Goal.IsTerminal()) {
			t.Fatalf("first receipt closed multi cancel: control=%s Goal=%s", control.Status, current.Goal.State())
		}
	}

	closed := system.record(t)
	control := mustControlByRequest(t, closed, request.RequestRef)
	if closed.Goal.State() != goal.GoalStateCanceled || control.Status != ControlConfirmed ||
		control.ReceiptRef != "receipt:"+control.Ref ||
		system.actionKindCount(ActionStopAgent) != 0 || system.stopCount() != 2 {
		t.Fatalf("multi cancel did not converge: Goal=%s control=%s actions=%d stops=%d",
			closed.Goal.State(), control.Status, system.actionKindCount(ActionStopAgent), system.stopCount())
	}
	effects := make(map[goal.ExecutionRef]EffectReceipt)
	for _, receipt := range closed.EffectReceipts {
		if receipt.Status == EffectStatusStopped {
			effects[receipt.Subject.ExecutionRef] = receipt
		}
	}
	if len(effects) != len(running.Executions) {
		t.Fatalf("stop effect receipts=%d want=%d: %+v", len(effects), len(running.Executions), effects)
	}
	for _, execution := range running.Executions {
		receipt, found := effects[execution.Ref]
		if !found || receipt.ActionRef != "action:stop:"+control.Ref+":"+execution.Ref.String() ||
			receipt.ExternalRef != "receipt:stop:"+execution.Ref.String() || receipt.ConfirmedAt.IsZero() {
			t.Fatalf("exact effect receipt for %s=%+v found=%v", execution.Ref, receipt, found)
		}
	}
}
