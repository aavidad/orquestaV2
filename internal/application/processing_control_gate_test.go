package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestClaimedRetryRevalidatesPauseBeforeLaunchPreparation(t *testing.T) {
	system, itemRef, stoppedRef := stoppedControlSystem(t, 3)
	retry := system.request(
		t, "control:retry-before-claimed-pause", ControlRetry, ControlTargetWorkItem, itemRef, stoppedRef,
	)
	if result, err := system.orchestrator.Control(context.Background(), system.access, retry); err != nil || !result.Created {
		t.Fatalf("create retry: result=%+v err=%v", result, err)
	}
	replacement := mustBoundExecution(t, system.record(t), itemRef)

	assertClaimedLaunchRevalidatesPause(t, system, ControlTargetGoal, itemRef, replacement.Ref)
}

func TestClaimedAutomaticReplacementRevalidatesPauseBeforeLaunchPreparation(t *testing.T) {
	agent := &scriptedAgent{launchErr: definitelyUnappliedPermanentError{"provider rejected first launch"}}
	system := newControlTestSystem(t, agent)
	agent.launchErrorHook = func() { system.clock.Advance(time.Nanosecond) }
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:create-replacement"); err != nil ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("create replacement: result=%+v err=%v", result, err)
	}
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	replacement := mustBoundExecution(t, record, item.Ref())
	if replacement.State != ExecutionQueued || replacement.AttemptNo != 2 {
		t.Fatalf("replacement=%+v", replacement)
	}
	system.clock.Advance(time.Second)
	agent.mu.Lock()
	agent.launchErr = nil
	agent.mu.Unlock()

	assertClaimedLaunchRevalidatesPause(t, system, ControlTargetWorkItem, item.Ref(), replacement.Ref)
}

func assertClaimedLaunchRevalidatesPause(
	t *testing.T,
	system *controlTestSystem,
	pauseTarget ControlTarget,
	itemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
) {
	t.Helper()
	gate := &holdFirstLaunchClaimState{
		StateRepository: system.orchestrator.state,
		claimed:         make(chan struct{}),
		release:         make(chan struct{}),
	}
	system.orchestrator.state = gate
	result := make(chan processCallResult, 1)
	go func() {
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:held-launch")
		result <- processCallResult{result: processed, err: err}
	}()

	select {
	case <-gate.claimed:
	case <-time.After(2 * time.Second):
		t.Fatal("launch action was not claimed")
	}
	pauseItem := goal.WorkItemRef{}
	if pauseTarget == ControlTargetWorkItem {
		pauseItem = itemRef
	}
	pause := system.request(
		t, "control:pause-after-claim:"+executionRef.String(), ControlPause, pauseTarget,
		pauseItem, goal.ExecutionRef{},
	)
	if _, err := system.orchestrator.Control(context.Background(), system.access, pause); err != nil {
		t.Fatalf("pause claimed launch: %v", err)
	}
	launches := system.launchCount()
	close(gate.release)
	select {
	case call := <-result:
		if call.err != nil || !call.result.Processed || call.result.Action != ActionLaunchAgent {
			t.Fatalf("process held launch: result=%+v err=%v", call.result, call.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("held launch did not finish")
	}

	current := mustBoundExecution(t, system.record(t), itemRef)
	if system.launchCount() != launches || current.Ref != executionRef || current.State != ExecutionQueued ||
		countExecutionEvents(system, executionRef, "execution.dispatching") != 0 {
		t.Fatalf("claimed launch crossed pause: launches=%d/%d execution=%+v events=%+v",
			system.launchCount(), launches, current, system.repository.events)
	}
	system.repository.mu.Lock()
	pending, found := system.repository.actions["action:launch:"+executionRef.String()]
	system.repository.mu.Unlock()
	if !found || pending.token != "" || pending.workerRef != "" || pending.lease != (time.Time{}) {
		t.Fatalf("launch action not reusable after pause: found=%v action=%+v", found, pending)
	}
	assertActionReservationReleased(t, system.record(t), pending.record.Ref)

	resume := system.request(
		t, "control:resume-after-claim:"+executionRef.String(), ControlResume, pauseTarget,
		pauseItem, goal.ExecutionRef{},
	)
	if _, err := system.orchestrator.Control(context.Background(), system.access, resume); err != nil {
		t.Fatalf("resume claimed launch: %v", err)
	}
	system.clock.Advance(2 * time.Second)
	processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:resumed-held-launch")
	if err != nil || !processed.Processed || processed.Action != ActionLaunchAgent {
		t.Fatalf("resumed launch: result=%+v err=%v", processed, err)
	}
	current = mustBoundExecution(t, system.record(t), itemRef)
	if system.launchCount() != launches+1 || current.Ref != executionRef || current.State != ExecutionRunning ||
		countExecutionEvents(system, executionRef, "execution.dispatching") != 1 {
		t.Fatalf("resumed launch not reused once: launches=%d/%d execution=%+v events=%+v",
			system.launchCount(), launches+1, current, system.repository.events)
	}
}

type processCallResult struct {
	result ProcessResult
	err    error
}

type holdFirstLaunchClaimState struct {
	StateRepository
	once    sync.Once
	claimed chan struct{}
	release chan struct{}
}

func (state *holdFirstLaunchClaimState) ClaimNextAction(
	ctx context.Context,
	request ClaimRequest,
) (ActionClaim, bool, error) {
	claim, found, err := state.StateRepository.ClaimNextAction(ctx, request)
	if err != nil || !found || claim.Action.Kind != ActionLaunchAgent {
		return claim, found, err
	}
	state.once.Do(func() {
		close(state.claimed)
		select {
		case <-state.release:
		case <-ctx.Done():
		}
	})
	return claim, found, err
}
