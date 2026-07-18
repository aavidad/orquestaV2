package application

import (
	"context"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestTamperedStopTargetNeverInvokesAdapter(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	requested, err := system.orchestrator.Control(context.Background(), system.access, system.request(
		t, "control:tampered-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
	))
	if err != nil || !requested.Created {
		t.Fatalf("request stop: result=%+v err=%v", requested, err)
	}

	system.repository.mu.Lock()
	stored := system.repository.records[record.Goal.Ref()]
	control, found := controlByRef(stored.Controls, requested.Control.Ref)
	if !found {
		system.repository.mu.Unlock()
		t.Fatal("stop control missing")
	}
	control.Target, control.Mode = ControlTargetGoal, ports.AgentStopForced
	stored.Controls = replaceControl(stored.Controls, control)
	system.repository.records[record.Goal.Ref()] = stored
	system.repository.mu.Unlock()

	processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:tampered-stop")
	if !processed.Processed || processed.Action != ActionStopAgent || err == nil ||
		err.Error() != "application.effect_target_mismatch" || system.stopCount() != 0 {
		t.Fatalf("tampered target crossed adapter: result=%+v stops=%d err=%v", processed, system.stopCount(), err)
	}
}

func TestLaunchTargetDigestBindsExactExecution(t *testing.T) {
	actor, project := testScope(t)
	goalRef, _ := goal.NewGoalRef("goal:target-digest")
	itemRef, _ := goal.NewWorkItemRef("work-item:target-digest")
	executionRef, _ := goal.NewExecutionRef("execution:target-digest")
	request := ports.AgentLaunchRequest{
		ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: itemRef,
		PlanGeneration: 1, AppSpecGeneration: 1, ExecutionAttempt: 1,
		SpecHash: "sha256:" + effectAdmissionFingerprint("target-spec"), ActorRef: actor,
		ProjectRef: project, IdempotencyKey: "launch:target-digest",
	}
	want := launchTargetDigest(request)
	request.ExecutionAttempt++
	if got := launchTargetDigest(request); got == want {
		t.Fatal("execution attempt retained launch target digest")
	}
}
