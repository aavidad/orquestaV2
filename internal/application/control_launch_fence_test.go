package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestPauseGatesRetryUntilRecordLaunchPrepared(t *testing.T) {
	system, itemRef, stoppedRef := stoppedControlSystem(t, 3)
	retry := system.request(
		t, "control:pause-fence-retry", ControlRetry, ControlTargetWorkItem, itemRef, stoppedRef,
	)
	if result, err := system.orchestrator.Control(context.Background(), system.access, retry); err != nil || !result.Created {
		t.Fatalf("create retry: result=%+v err=%v", result, err)
	}
	retried := system.record(t)
	replacement := mustBoundExecution(t, retried, itemRef)
	if replacement.State != ExecutionQueued || countExecutionEvents(system, replacement.Ref, "execution.dispatching") != 0 {
		t.Fatalf("retry crossed preparation before claim: execution=%+v events=%+v", replacement, system.repository.events)
	}
	pause := system.request(
		t, "control:pause-fence-before-retry-claim", ControlPause, ControlTargetGoal,
		goal.WorkItemRef{}, goal.ExecutionRef{},
	)
	if _, err := system.orchestrator.Control(context.Background(), system.access, pause); err != nil {
		t.Fatal(err)
	}
	launches := system.launchCount()
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:paused-retry"); err != nil || result.Processed {
		t.Fatalf("paused retry claimed: result=%+v err=%v", result, err)
	}
	if system.launchCount() != launches || mustBoundExecution(t, system.record(t), itemRef).State != ExecutionQueued {
		t.Fatal("paused retry escaped to provider")
	}

	resume := system.request(
		t, "control:resume-fence-retry", ControlResume, ControlTargetGoal,
		goal.WorkItemRef{}, goal.ExecutionRef{},
	)
	if _, err := system.orchestrator.Control(context.Background(), system.access, resume); err != nil {
		t.Fatal(err)
	}
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:resumed-retry"); err != nil ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("resumed retry launch: result=%+v err=%v", result, err)
	}
	if current := mustBoundExecution(t, system.record(t), itemRef); current.State != ExecutionRunning ||
		countExecutionEvents(system, replacement.Ref, "execution.dispatching") != 1 {
		t.Fatalf("retry preparation frontier missing: execution=%+v events=%+v", current, system.repository.events)
	}
}

func TestPauseGatesAutomaticReplacementThroughoutBackoff(t *testing.T) {
	agent := &scriptedAgent{launchErr: definitelyUnappliedPermanentError{"provider rejected first launch"}}
	system := newControlTestSystem(t, agent)
	agent.launchErrorHook = func() { system.clock.Advance(time.Nanosecond) }
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:replacement-failure"); err != nil ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("create automatic replacement: result=%+v err=%v", result, err)
	}
	replaced := system.record(t)
	item := replaced.Goal.WorkItems()[0]
	replacement := mustBoundExecution(t, replaced, item.Ref())
	if replacement.State != ExecutionQueued || replacement.AttemptNo != 2 ||
		countExecutionEvents(system, replacement.Ref, "execution.queued") != 1 ||
		countExecutionEvents(system, replacement.Ref, "execution.dispatching") != 0 {
		t.Fatalf("automatic replacement was pre-dispatched: %+v", replacement)
	}
	pause := system.request(
		t, "control:pause-replacement-backoff", ControlPause, ControlTargetWorkItem,
		item.Ref(), goal.ExecutionRef{},
	)
	if _, err := system.orchestrator.Control(context.Background(), system.access, pause); err != nil {
		t.Fatal(err)
	}
	system.clock.Advance(2 * time.Second)
	launches := system.launchCount()
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:paused-backoff"); err != nil || result.Processed {
		t.Fatalf("paused backoff replacement claimed: result=%+v err=%v", result, err)
	}
	if system.launchCount() != launches || mustBoundExecution(t, system.record(t), item.Ref()).State != ExecutionQueued {
		t.Fatal("paused backoff replacement escaped")
	}

	agent.mu.Lock()
	agent.launchErr = nil
	agent.mu.Unlock()
	resume := system.request(
		t, "control:resume-replacement-backoff", ControlResume, ControlTargetWorkItem,
		item.Ref(), goal.ExecutionRef{},
	)
	if _, err := system.orchestrator.Control(context.Background(), system.access, resume); err != nil {
		t.Fatal(err)
	}
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:resumed-backoff"); err != nil ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("resumed replacement launch: result=%+v err=%v", result, err)
	}
	if current := mustBoundExecution(t, system.record(t), item.Ref()); current.State != ExecutionRunning ||
		countExecutionEvents(system, replacement.Ref, "execution.dispatching") != 1 {
		t.Fatalf("replacement preparation frontier missing: execution=%+v events=%+v", current, system.repository.events)
	}
}

func countExecutionEvents(system *controlTestSystem, ref goal.ExecutionRef, kind string) int {
	system.repository.mu.Lock()
	defer system.repository.mu.Unlock()
	count := 0
	for _, event := range system.repository.events {
		if event.ExecutionRef == ref && event.Kind == kind {
			count++
		}
	}
	return count
}
