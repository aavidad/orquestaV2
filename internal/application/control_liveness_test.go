package application

import (
	"context"
	"fmt"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestPendingStopQuarantinesUnknownAppliedAndYields(t *testing.T) {
	agent := &scriptedAgent{stopStatus: ports.AgentStopPending}
	system := newControlTestSystemWithPlan(t, agent, controlIndependentPlan())
	system.launch(t)
	running := system.record(t)
	runningItem := controlItemByObjective(t, running, "first independent")
	execution := mustBoundExecution(t, running, runningItem.Ref())
	requested, err := system.orchestrator.Control(context.Background(), system.access,
		system.request(t, "control:pending-stop-backoff", ControlStop,
			ControlTargetExecution, runningItem.Ref(), execution.Ref))
	if err != nil || !requested.Created {
		t.Fatalf("request pending stop: result=%+v err=%v", requested, err)
	}
	stopActionRef := "action:stop:" + requested.Control.Ref + ":" + execution.Ref.String()
	first, err := system.orchestrator.ProcessNext(context.Background(), "worker:pending-stop-first")
	if err == nil || err.Error() != effectUnknownAppliedCode || first.Action != ActionStopAgent {
		t.Fatalf("pending stop not quarantined: result=%+v err=%v", first, err)
	}
	prepared, err := system.orchestrator.ProcessNext(context.Background(), "worker:fresh-prepare-after-stop")
	if err != nil || !prepared.Processed || prepared.Action == ActionStopAgent {
		t.Fatalf("pending stop did not yield to other work: result=%+v err=%v", prepared, err)
	}
	closed := system.record(t)
	stopAttempts := 0
	for _, attempt := range closed.EffectAttempts {
		if attempt.ActionRef == stopActionRef {
			stopAttempts++
		}
	}
	control, _ := controlByRef(closed.Controls, requested.Control.Ref)
	if stopAttempts != 1 || control.Status != ControlRequested {
		t.Fatalf("pending quarantine facts attempts=%d control=%+v", stopAttempts, control)
	}
	if _, exists := system.effects(t).actions[stopActionRef]; exists {
		t.Fatalf("quarantined stop action remained schedulable: %s", stopActionRef)
	}
}

func TestPendingStopBackoffPreservesFirstUrgencyThenYieldsAndCaps(t *testing.T) {
	agent := &scriptedAgent{stopStatus: ports.AgentStopPending}
	system := newControlTestSystemWithPlan(t, agent, controlIndependentPlan())
	system.launch(t)
	running := system.record(t)
	runningItem := controlItemByObjective(t, running, "first independent")
	execution := mustBoundExecution(t, running, runningItem.Ref())
	requested, err := system.orchestrator.Control(context.Background(), system.access,
		system.request(t, "control:pending-stop-backoff", ControlStop,
			ControlTargetExecution, runningItem.Ref(), execution.Ref))
	if err != nil || !requested.Created {
		t.Fatalf("request pending stop: result=%+v err=%v", requested, err)
	}
	stopActionRef := "action:stop:" + requested.Control.Ref + ":" + execution.Ref.String()
	initial := system.effects(t).actions[stopActionRef]
	intent := initial.record.EffectIntent
	base, capDelay := intent.QuotaRetryDelay, 2*intent.QuotaRetryDelay
	system.orchestrator.executionTimeout = capDelay
	firstAt := system.clock.Now()
	first := system.claim(t, "pending-stop-first")
	if first.Action.Kind != ActionStopAgent {
		t.Fatalf("first pending stop lost urgent priority: claim=%+v", first)
	}
	if err := system.orchestrator.requeueStop(
		context.Background(), first, execution, "application.stop_waiting_launch_receipt",
	); err != nil {
		t.Fatalf("requeue first pre-effect stop: %v", err)
	}
	firstRetry := system.effects(t).actions[stopActionRef]
	if firstRetry.deliveryAttempt != 1 || firstRetry.fence != 1 ||
		firstRetry.record.EffectIntent != intent || !firstRetry.record.AvailableAt.Equal(firstAt.Add(base)) {
		t.Fatalf("first stop retry lost durable policy/fence: %+v", firstRetry)
	}
	prepared, err := system.orchestrator.ProcessNext(context.Background(), "worker:fresh-prepare-after-stop")
	if err != nil || prepared.Action != ActionPrepareWorkspace {
		t.Fatalf("pending stop did not yield to unrelated workspace preparation: result=%+v err=%v", prepared, err)
	}
	yielded, err := system.orchestrator.ProcessNext(context.Background(), "worker:fresh-launch-after-stop")
	if err != nil || yielded.Action != ActionLaunchAgent {
		t.Fatalf("prepared workspace did not launch: result=%+v err=%v", yielded, err)
	}
	for attempt := uint64(2); attempt <= 3; attempt++ {
		pending := system.effects(t).actions[stopActionRef]
		system.clock.Advance(pending.record.AvailableAt.Sub(system.clock.Now()))
		retriedAt := system.clock.Now()
		claim := system.claim(t, fmt.Sprintf("pending-stop-retry-%d", attempt))
		if claim.Action.Kind != ActionStopAgent {
			t.Fatalf("pending stop retry %d lost priority: claim=%+v", attempt, claim)
		}
		if err := system.orchestrator.requeueStop(
			context.Background(), claim, execution, "application.stop_waiting_launch_receipt",
		); err != nil {
			t.Fatalf("requeue pre-effect stop %d: %v", attempt, err)
		}
		retry := system.effects(t).actions[stopActionRef]
		if retry.deliveryAttempt != attempt || retry.fence != attempt || retry.record.EffectIntent != intent ||
			!retry.record.AvailableAt.Equal(retriedAt.Add(capDelay)) {
			t.Fatalf("bounded stop retry %d lost causal state: %+v", attempt, retry)
		}
	}
	closed := system.record(t)
	control, _ := controlByRef(closed.Controls, requested.Control.Ref)
	for _, attempt := range closed.EffectAttempts {
		if attempt.ActionRef == stopActionRef {
			t.Fatalf("pre-effect backoff created physical stop attempt: %+v", attempt)
		}
	}
	if system.stopCount() != 0 || control.Status != ControlRequested {
		t.Fatalf("pre-effect retry invoked adapter or closed control: stops=%d control=%+v", system.stopCount(), control)
	}
}

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
		if !controlExecutionForItem(after, second.Ref()) || system.actionKindCount(ActionPrepareWorkspace) != 1 {
			t.Fatalf("queued cancel did not release writer: executions=%+v actions=%d", after.Executions, system.actionKindCount(ActionPrepareWorkspace))
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
		if !controlExecutionForItem(after, second.Ref()) || system.actionKindCount(ActionPrepareWorkspace) != 1 {
			t.Fatalf("stop did not release writer: executions=%+v actions=%d", after.Executions, system.actionKindCount(ActionPrepareWorkspace))
		}
	})

	t.Run("exhausted failure", func(t *testing.T) {
		agent := &scriptedAgent{launchErr: definitelyUnappliedPermanentError{"permanent launch failure"}}
		system := newControlTestSystemWithPlan(t, agent, controlConflictingPlan())
		agent.launchErrorHook = func() { system.clock.Advance(time.Nanosecond) }
		record := system.record(t)
		first := controlItemByObjective(t, record, "first writer")
		second := controlItemByObjective(t, record, "second writer")
		system.repository.mu.Lock()
		stored := system.repository.records[system.goalRef]
		stored.Executions[0].MaxExecutionAttempts = 1
		system.repository.records[system.goalRef] = stored
		system.repository.mu.Unlock()
		if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:exhaust-prepare"); err != nil || result.Action != ActionPrepareWorkspace {
			t.Fatalf("prepare writer: result=%+v err=%v", result, err)
		}
		if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:exhaust-writer"); err != nil || !result.Processed || result.Action != ActionLaunchAgent {
			t.Fatalf("exhaust writer: result=%+v err=%v", result, err)
		}
		after := system.record(t)
		interrupted, _ := after.Goal.WorkItem(first.Ref())
		if interrupted.State() != goal.WorkItemStateInterrupted || !controlExecutionForItem(after, second.Ref()) ||
			system.actionKindCount(ActionPrepareWorkspace) != 1 {
			t.Fatalf("exhaustion did not release writer: item=%s executions=%+v actions=%d",
				interrupted.State(), after.Executions, system.actionKindCount(ActionPrepareWorkspace))
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
			{Key: "first", Objective: "first writer", Phase: "phase:control-writers", Role: "role:worker", WriteSet: []string{"internal/shared"}, RequiredTests: requiredTestSpecs("required-test:control-first"), OutputContract: goal.OutputContractEvidenceBundle},
			{Key: "second", Objective: "second writer", Phase: "phase:control-writers", Role: "role:worker", WriteSet: []string{"internal/shared/file.go"}, RequiredTests: requiredTestSpecs("required-test:control-second"), OutputContract: goal.OutputContractEvidenceBundle},
		},
	}
}

func controlIndependentPlan() *PlanSpec {
	return &PlanSpec{
		Phases: []PhaseSpec{{Ref: "phase-instance:control-independent", Key: "phase:control-independent", TemplateRef: "phase-template:control-independent"}},
		WorkItems: []WorkItemSpec{
			{Key: "first", Objective: "first independent", Phase: "phase:control-independent", Role: "role:worker", WriteSet: []string{"internal/first"}, RequiredTests: requiredTestSpecs("required-test:control-independent-first"), OutputContract: goal.OutputContractEvidenceBundle},
			{Key: "second", Objective: "second independent", Phase: "phase:control-independent", Role: "role:worker", WriteSet: []string{"internal/second"}, RequiredTests: requiredTestSpecs("required-test:control-independent-second"), OutputContract: goal.OutputContractEvidenceBundle},
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
