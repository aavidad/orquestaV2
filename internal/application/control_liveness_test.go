package application

import (
	"context"
	"testing"

	"orquesta/internal/goal"
)

func TestControlResumeSchedulesWorkThatBecameReadyWhilePaused(t *testing.T) {
	system := newControlTestSystemWithPlan(t, completedObservation("dependency complete"), controlDependencyPlan())
	system.launch(t)
	if _, err := system.orchestrator.Control(context.Background(), system.access,
		system.request(t, "control:pause-dependency", ControlPause, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})); err != nil {
		t.Fatal(err)
	}
	system.observe(t)
	blocked := controlItemByObjective(t, system.record(t), "dependent")
	if controlExecutionForItem(system.record(t), blocked.Ref()) {
		t.Fatal("dependent work was scheduled while Goal was paused")
	}

	resume := system.request(t, "control:resume-dependency", ControlResume, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
	if result, err := system.orchestrator.Control(context.Background(), system.access, resume); err != nil || !result.Created {
		t.Fatalf("resume: result=%+v err=%v", result, err)
	}
	after := system.record(t)
	if !controlExecutionForItem(after, blocked.Ref()) || system.actionKindCount(ActionLaunchAgent) != 1 {
		t.Fatalf("resume did not schedule exactly one dependent execution: executions=%+v actions=%d",
			after.Executions, system.actionKindCount(ActionLaunchAgent))
	}
	effects := system.effects(t)
	if replay, err := system.orchestrator.Control(context.Background(), system.access, resume); err != nil || replay.Created {
		t.Fatalf("resume replay: result=%+v err=%v", replay, err)
	}
	system.assertEffects(t, effects)
}

func TestControlTerminalTransitionsReleaseWriteSetAndScheduleExactlyOnce(t *testing.T) {
	t.Run("queued cancel", func(t *testing.T) {
		system := newControlTestSystemWithPlan(t, nil, controlConflictingPlan())
		record := system.record(t)
		first := controlItemByObjective(t, record, "first writer")
		second := controlItemByObjective(t, record, "second writer")
		request := system.request(t, "control:cancel-queued-writer", ControlCancel, ControlTargetWorkItem, first.Ref(), goal.ExecutionRef{})
		if result, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil || result.Control.Status != ControlConfirmed {
			t.Fatalf("cancel queued: result=%+v err=%v", result, err)
		}
		after := system.record(t)
		if !controlExecutionForItem(after, second.Ref()) || system.actionKindCount(ActionLaunchAgent) != 1 {
			t.Fatalf("queued cancel did not release writer: executions=%+v actions=%d", after.Executions, system.actionKindCount(ActionLaunchAgent))
		}
	})

	t.Run("confirmed stop", func(t *testing.T) {
		system := newControlTestSystemWithPlan(t, nil, controlConflictingPlan())
		system.launch(t)
		running := system.record(t)
		first := controlItemByObjective(t, running, "first writer")
		second := controlItemByObjective(t, running, "second writer")
		execution := mustBoundExecution(t, running, first.Ref())
		request := system.request(t, "control:stop-running-writer", ControlStop, ControlTargetExecution, first.Ref(), execution.Ref)
		if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
			t.Fatal(err)
		}
		if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:stop-writer"); err != nil || result.Action != ActionStopAgent {
			t.Fatalf("process stop: result=%+v err=%v", result, err)
		}
		after := system.record(t)
		if !controlExecutionForItem(after, second.Ref()) || system.actionKindCount(ActionLaunchAgent) != 1 {
			t.Fatalf("stop did not release writer: executions=%+v actions=%d", after.Executions, system.actionKindCount(ActionLaunchAgent))
		}
	})

	t.Run("exhausted failure", func(t *testing.T) {
		system := newControlTestSystemWithPlan(t, &scriptedAgent{launchErr: definitelyUnappliedPermanentError{"permanent launch failure"}}, controlConflictingPlan())
		record := system.record(t)
		first := controlItemByObjective(t, record, "first writer")
		second := controlItemByObjective(t, record, "second writer")
		system.repository.mu.Lock()
		stored := system.repository.records[system.goalRef]
		stored.Executions[0].MaxExecutionAttempts = 1
		system.repository.records[system.goalRef] = stored
		system.repository.mu.Unlock()
		if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:exhaust-writer"); err != nil || !result.Processed {
			t.Fatalf("exhaust writer: result=%+v err=%v", result, err)
		}
		after := system.record(t)
		interrupted, _ := after.Goal.WorkItem(first.Ref())
		if interrupted.State() != goal.WorkItemStateInterrupted || !controlExecutionForItem(after, second.Ref()) ||
			system.actionKindCount(ActionLaunchAgent) != 1 {
			t.Fatalf("exhaustion did not release writer: item=%s executions=%+v actions=%d",
				interrupted.State(), after.Executions, system.actionKindCount(ActionLaunchAgent))
		}
	})
}

func TestControlRetryRejectsEffectivePauseWithoutEffects(t *testing.T) {
	for _, target := range []ControlTarget{ControlTargetGoal, ControlTargetWorkItem} {
		t.Run(string(target), func(t *testing.T) {
			system, itemRef, executionRef := stoppedControlSystem(t, 3)
			pauseItem := goal.WorkItemRef{}
			if target == ControlTargetWorkItem {
				pauseItem = itemRef
			}
			pause := system.request(t, "control:pause-before-retry:"+string(target), ControlPause, target, pauseItem, goal.ExecutionRef{})
			if _, err := system.orchestrator.Control(context.Background(), system.access, pause); err != nil {
				t.Fatal(err)
			}
			before := system.effects(t)
			retry := system.request(t, "control:retry-paused:"+string(target), ControlRetry, ControlTargetWorkItem, itemRef, executionRef)
			if _, err := system.orchestrator.Control(context.Background(), system.access, retry); !IsStateError(err, StateConflict) {
				t.Fatalf("paused retry error=%v", err)
			}
			system.assertEffects(t, before)

			resume := system.request(t, "control:resume-before-retry:"+string(target), ControlResume, target, pauseItem, goal.ExecutionRef{})
			if _, err := system.orchestrator.Control(context.Background(), system.access, resume); err != nil {
				t.Fatal(err)
			}
			retry = system.request(t, "control:retry-after-resume:"+string(target), ControlRetry, ControlTargetWorkItem, itemRef, executionRef)
			if result, err := system.orchestrator.Control(context.Background(), system.access, retry); err != nil ||
				result.Control.Status != ControlConfirmed || system.actionKindCount(ActionLaunchAgent) != 1 {
				t.Fatalf("retry after resume: result=%+v actions=%d err=%v", result, system.actionKindCount(ActionLaunchAgent), err)
			}
		})
	}
}

func TestControlStopOwnerRejectsOverlapButAllowsIndependentExecutions(t *testing.T) {
	t.Run("stop owns execution", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		system.launch(t)
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		first := system.request(t, "control:owner-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
		if _, err := system.orchestrator.Control(context.Background(), system.access, first); err != nil {
			t.Fatal(err)
		}
		before := system.effects(t)
		for _, candidate := range []ControlRequest{
			system.request(t, "control:overlap-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref),
			system.request(t, "control:overlap-cancel", ControlCancel, ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{}),
		} {
			if _, err := system.orchestrator.Control(context.Background(), system.access, candidate); !IsStateError(err, StateConflict) {
				t.Fatalf("overlap %s error=%v", candidate.Operation, err)
			}
			system.assertEffects(t, before)
		}
	})

	t.Run("cancel owns execution", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		system.launch(t)
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		if _, err := system.orchestrator.Control(context.Background(), system.access,
			system.request(t, "control:owner-cancel", ControlCancel, ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})); err != nil {
			t.Fatal(err)
		}
		before := system.effects(t)
		if _, err := system.orchestrator.Control(context.Background(), system.access,
			system.request(t, "control:stop-after-cancel", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)); !IsStateError(err, StateConflict) {
			t.Fatalf("stop after cancel error=%v", err)
		}
		system.assertEffects(t, before)
	})

	t.Run("independent executions", func(t *testing.T) {
		system := newControlTestSystemWithPlan(t, nil, controlIndependentPlan())
		system.launch(t)
		system.launch(t)
		record := system.record(t)
		items := record.Goal.WorkItems()
		for index, item := range items {
			execution := mustBoundExecution(t, system.record(t), item.Ref())
			request := system.request(t, "control:independent-stop:"+string(rune('a'+index)), ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
			if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
				t.Fatalf("independent stop %d: %v", index, err)
			}
		}
		if system.actionKindCount(ActionStopAgent) != 2 {
			t.Fatalf("independent stop actions=%d", system.actionKindCount(ActionStopAgent))
		}
	})
}

func controlDependencyPlan() *PlanSpec {
	return &PlanSpec{
		Phases: []PhaseSpec{{Ref: "phase-instance:control-liveness", Key: "phase:control-liveness", TemplateRef: "phase-template:control-liveness"}},
		WorkItems: []WorkItemSpec{
			{Key: "first", Objective: "dependency", Phase: "phase:control-liveness", Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
			{Key: "second", Objective: "dependent", Phase: "phase:control-liveness", Role: "role:worker", Dependencies: []string{"first"}, OutputContract: goal.OutputContractEvidenceBundle},
		},
	}
}

func controlConflictingPlan() *PlanSpec {
	return &PlanSpec{
		Phases: []PhaseSpec{{Ref: "phase-instance:control-writers", Key: "phase:control-writers", TemplateRef: "phase-template:control-writers"}},
		WorkItems: []WorkItemSpec{
			{Key: "first", Objective: "first writer", Phase: "phase:control-writers", Role: "role:worker", WriteSet: []string{"internal/shared"}, OutputContract: goal.OutputContractEvidenceBundle},
			{Key: "second", Objective: "second writer", Phase: "phase:control-writers", Role: "role:worker", WriteSet: []string{"internal/shared/file.go"}, OutputContract: goal.OutputContractEvidenceBundle},
		},
	}
}

func controlIndependentPlan() *PlanSpec {
	return &PlanSpec{
		Phases: []PhaseSpec{{Ref: "phase-instance:control-independent", Key: "phase:control-independent", TemplateRef: "phase-template:control-independent"}},
		WorkItems: []WorkItemSpec{
			{Key: "first", Objective: "first independent", Phase: "phase:control-independent", Role: "role:worker", WriteSet: []string{"internal/first"}, OutputContract: goal.OutputContractEvidenceBundle},
			{Key: "second", Objective: "second independent", Phase: "phase:control-independent", Role: "role:worker", WriteSet: []string{"internal/second"}, OutputContract: goal.OutputContractEvidenceBundle},
		},
	}
}

func controlItemByObjective(t *testing.T, record GoalRecord, objective string) goal.WorkItem {
	t.Helper()
	for _, item := range record.Goal.WorkItems() {
		if item.Objective() == objective {
			return item
		}
	}
	t.Fatalf("WorkItem with objective %q missing", objective)
	return goal.WorkItem{}
}

func controlExecutionForItem(record GoalRecord, itemRef goal.WorkItemRef) bool {
	for _, execution := range record.Executions {
		if execution.WorkItemRef == itemRef {
			return true
		}
	}
	return false
}
