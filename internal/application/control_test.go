package application

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestControlsPauseBeforeAndAfterLaunchPrepared(t *testing.T) {
	t.Run("before launch prepared gates launch and resume reuses it", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		initial := system.record(t)
		item := initial.Goal.WorkItems()[0]
		request := system.request(t, "control:pause-before", ControlPause, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})

		result, err := system.orchestrator.Control(context.Background(), system.access, request)
		if err != nil || !result.Created || result.Control.Status != ControlConfirmed {
			t.Fatalf("pause before prepared: result=%+v err=%v", result, err)
		}
		paused := system.record(t)
		if !paused.Goal.Paused() || paused.Goal.ControlSequence() != initial.Goal.ControlSequence()+1 {
			t.Fatalf("Goal pause not durable: %+v", paused.Goal.Snapshot())
		}
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:paused")
		if err != nil || processed.Processed || system.launchCount() != 0 {
			t.Fatalf("paused launch escaped: result=%+v launches=%d err=%v", processed, system.launchCount(), err)
		}

		resume := system.request(t, "control:resume-before", ControlResume, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
		resumed, err := system.orchestrator.Control(context.Background(), system.access, resume)
		if err != nil || !resumed.Created || resumed.Control.Status != ControlConfirmed {
			t.Fatalf("resume: result=%+v err=%v", resumed, err)
		}
		processed, err = system.orchestrator.ProcessNext(context.Background(), "worker:resumed")
		if err != nil || !processed.Processed || processed.Action != ActionLaunchAgent || system.launchCount() != 1 {
			t.Fatalf("resumed action: result=%+v launches=%d err=%v", processed, system.launchCount(), err)
		}
		current := system.record(t)
		currentItem, _ := current.Goal.WorkItem(item.Ref())
		if current.Goal.Paused() || currentItem.State() != goal.WorkItemStateRunning {
			t.Fatalf("resume did not reuse pending launch: Goal paused=%v item=%s", current.Goal.Paused(), currentItem.State())
		}
	})

	t.Run("both effective scopes must be resumed", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		item := system.record(t).Goal.WorkItems()[0]
		for _, operation := range []struct {
			ref    string
			target ControlTarget
		}{
			{ref: "control:pause-goal-scope", target: ControlTargetGoal},
			{ref: "control:pause-item-scope", target: ControlTargetWorkItem},
		} {
			itemRef := goal.WorkItemRef{}
			if operation.target == ControlTargetWorkItem {
				itemRef = item.Ref()
			}
			request := system.request(t, operation.ref, ControlPause, operation.target, itemRef, goal.ExecutionRef{})
			if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
				t.Fatalf("%s: %v", operation.ref, err)
			}
		}
		resumeItem := system.request(t, "control:resume-item-scope", ControlResume, ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
		if _, err := system.orchestrator.Control(context.Background(), system.access, resumeItem); err != nil {
			t.Fatalf("resume item scope: %v", err)
		}
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:goal-still-paused")
		if err != nil || result.Processed || system.launchCount() != 0 {
			t.Fatalf("one resumed scope lifted effective gate: result=%+v launches=%d err=%v", result, system.launchCount(), err)
		}
		resumeGoal := system.request(t, "control:resume-goal-scope", ControlResume, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
		if _, err := system.orchestrator.Control(context.Background(), system.access, resumeGoal); err != nil {
			t.Fatalf("resume Goal scope: %v", err)
		}
		result, err = system.orchestrator.ProcessNext(context.Background(), "worker:both-resumed")
		if err != nil || !result.Processed || result.Action != ActionLaunchAgent || system.launchCount() != 1 {
			t.Fatalf("both scopes resumed: result=%+v launches=%d err=%v", result, system.launchCount(), err)
		}
	})

	t.Run("after launch prepared does not freeze inflight observation", func(t *testing.T) {
		entered := make(chan struct{}, 1)
		release := make(chan struct{})
		agent := &scriptedAgent{
			launchEntered: entered, launchRelease: release,
			observations: []ports.AgentObservation{{
				Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("finished while paused"),
			}},
		}
		system := newControlTestSystem(t, agent)
		done := make(chan error, 1)
		go func() {
			_, err := system.orchestrator.ProcessNext(context.Background(), "worker:prepared-pause")
			done <- err
		}()
		<-entered

		prepared := system.record(t)
		item := prepared.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, prepared, item.Ref())
		if execution.State != ExecutionDispatching || item.State() != goal.WorkItemStateRunning {
			close(release)
			<-done
			t.Fatalf("launch frontier not prepared: item=%s execution=%s", item.State(), execution.State)
		}
		request := system.request(t, "control:pause-after", ControlPause, ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
		result, err := system.orchestrator.Control(context.Background(), system.access, request)
		if err != nil {
			close(release)
			<-done
			t.Fatalf("pause after prepared: %v", err)
		}
		if result.Control.Status != ControlConfirmed {
			close(release)
			<-done
			t.Fatalf("pause result=%+v", result)
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatalf("prepared launch completion: %v", err)
		}
		if system.launchCount() != 1 {
			t.Fatalf("prepared launch count=%d", system.launchCount())
		}
		system.clock.Advance(time.Second)
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:observe-paused")
		if err != nil || !processed.Processed || processed.Action != ActionObserveAgent {
			t.Fatalf("paused inflight observation: result=%+v err=%v", processed, err)
		}
		closed := system.record(t)
		if closed.Goal.State() != goal.GoalStateSucceeded || system.observationCount() != 1 {
			t.Fatalf("inflight completion lost: state=%s observations=%d", closed.Goal.State(), system.observationCount())
		}
	})
}

func TestControlsReplayAndSemanticConflict(t *testing.T) {
	system := newControlTestSystem(t, nil)
	request := system.request(t, "control:replay", ControlPause, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
	request.Reason = "  normalized operator reason  "
	first, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || !first.Created {
		t.Fatalf("first control: result=%+v err=%v", first, err)
	}
	afterFirst := system.effects(t)
	replay, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || replay.Created || !reflect.DeepEqual(replay.Control, first.Control) {
		t.Fatalf("replay: result=%+v want=%+v err=%v", replay, first, err)
	}
	if first.Control.Reason != "normalized operator reason" || replay.Control.Reason != first.Control.Reason {
		t.Fatalf("normalized reason missing from persisted/replayed control: first=%q replay=%q",
			first.Control.Reason, replay.Control.Reason)
	}
	system.assertEffects(t, afterFirst)

	conflict := request
	conflict.Reason = "different semantic intent"
	if _, err := system.orchestrator.Control(context.Background(), system.access, conflict); !IsStateError(err, StateConflict) {
		t.Fatalf("semantic replay conflict=%v", err)
	}
	system.assertEffects(t, afterFirst)
}

func TestControlsCancelBeforeAndAfterLaunchPrepared(t *testing.T) {
	t.Run("before prepared retires queued execution locally", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		initial := system.record(t)
		execution := onlyExecution(t, initial)
		request := system.request(t, "control:cancel-before", ControlCancel, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
		result, err := system.orchestrator.Control(context.Background(), system.access, request)
		if err != nil || result.Control.Status != ControlConfirmed {
			t.Fatalf("cancel before prepared: result=%+v err=%v", result, err)
		}
		closed := system.record(t)
		canceledExecution, _ := executionByRef(closed.Executions, execution.Ref)
		if closed.Goal.State() != goal.GoalStateCanceled || canceledExecution.State != ExecutionCanceled ||
			canceledExecution.FailureCode != "application.execution_canceled" ||
			system.launchCount() != 0 || system.stopCount() != 0 || system.actionCount() != 0 {
			t.Fatalf("local cancel leaked effect: Goal=%s execution=%s failure=%s launches=%d stops=%d actions=%d",
				closed.Goal.State(), canceledExecution.State, canceledExecution.FailureCode,
				system.launchCount(), system.stopCount(), system.actionCount())
		}
	})

	t.Run("after prepared preserves launch identity then stops exact execution", func(t *testing.T) {
		entered := make(chan struct{}, 1)
		release := make(chan struct{})
		system := newControlTestSystem(t, &scriptedAgent{launchEntered: entered, launchRelease: release})
		done := make(chan error, 1)
		go func() {
			_, err := system.orchestrator.ProcessNext(context.Background(), "worker:prepared-cancel")
			done <- err
		}()
		<-entered
		prepared := system.record(t)
		item := prepared.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, prepared, item.Ref())
		request := system.request(t, "control:cancel-after", ControlCancel, ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
		result, err := system.orchestrator.Control(context.Background(), system.access, request)
		if err != nil {
			close(release)
			<-done
			t.Fatalf("cancel after prepared: %v", err)
		}
		if result.Control.Status != ControlRequested {
			close(release)
			<-done
			t.Fatalf("cancel confirmed before exact stop: %+v", result.Control)
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatalf("launch receipt after cancel: %v", err)
		}
		launched := system.record(t)
		launchedExecution, _ := executionByRef(launched.Executions, execution.Ref)
		if launchedExecution.State != ExecutionRunning || launchedExecution.ExternalRef == "" {
			t.Fatalf("post-prepared launch identity lost: %+v", launchedExecution)
		}
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:cancel-stop")
		if err != nil || !processed.Processed || processed.Action != ActionStopAgent {
			t.Fatalf("cancel stop: result=%+v err=%v", processed, err)
		}
		closed := system.record(t)
		stopped, _ := executionByRef(closed.Executions, execution.Ref)
		control := mustControlByRequest(t, closed, request.RequestRef)
		item, _ = closed.Goal.WorkItem(item.Ref())
		if stopped.State != ExecutionStopped || item.State() != goal.WorkItemStateCanceled ||
			closed.Goal.State() != goal.GoalStateFailed || control.Status != ControlConfirmed || system.stopCount() != 1 {
			t.Fatalf("post-prepared cancel frontier: Goal=%s item=%s execution=%s control=%s stops=%d",
				closed.Goal.State(), item.State(), stopped.State, control.Status, system.stopCount())
		}
	})

	t.Run("after prepared rejection resolves cancel without inventing stop", func(t *testing.T) {
		entered := make(chan struct{}, 1)
		release := make(chan struct{})
		agent := &scriptedAgent{
			launchEntered: entered, launchRelease: release,
			launchErr: definitelyUnappliedPermanentError{"provider rejected launch"},
		}
		system := newControlTestSystem(t, agent)
		agent.launchErrorHook = func() { system.clock.Advance(time.Nanosecond) }
		done := make(chan error, 1)
		go func() {
			_, err := system.orchestrator.ProcessNext(context.Background(), "worker:prepared-rejection")
			done <- err
		}()
		<-entered

		prepared := system.record(t)
		item := prepared.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, prepared, item.Ref())
		request := system.request(t, "control:cancel-rejected-launch", ControlCancel, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
		result, err := system.orchestrator.Control(context.Background(), system.access, request)
		if err != nil || result.Control.Status != ControlRequested {
			close(release)
			<-done
			t.Fatalf("cancel prepared rejection: result=%+v err=%v", result, err)
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatalf("persist launch rejection after cancel: %v", err)
		}

		closed := system.record(t)
		failed, _ := executionByRef(closed.Executions, execution.Ref)
		item, _ = closed.Goal.WorkItem(item.Ref())
		control := mustControlByRequest(t, closed, request.RequestRef)
		if closed.Goal.State() != goal.GoalStateCanceled || item.State() != goal.WorkItemStateCanceled ||
			failed.State != ExecutionFailed || control.Status != ControlConfirmed ||
			system.stopCount() != 0 || system.actionCount() != 0 {
			t.Fatalf("rejected launch cancel: Goal=%s item=%s execution=%s control=%s stops=%d actions=%d",
				closed.Goal.State(), item.State(), failed.State, control.Status,
				system.stopCount(), system.actionCount())
		}
	})

	t.Run("overlapping work item then Goal cancel conflicts without duplicate stop", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		system.launch(t)
		running := system.record(t)
		item := running.Goal.WorkItems()[0]
		first := system.request(t, "control:cancel-item-first", ControlCancel, ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
		if result, err := system.orchestrator.Control(context.Background(), system.access, first); err != nil || result.Control.Status != ControlRequested {
			t.Fatalf("first scoped cancel: result=%+v err=%v", result, err)
		}
		before := system.effects(t)
		second := system.request(t, "control:cancel-goal-overlap", ControlCancel, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
		if _, err := system.orchestrator.Control(context.Background(), system.access, second); !IsStateError(err, StateConflict) {
			t.Fatalf("overlapping Goal cancel error=%v", err)
		}
		system.assertEffects(t, before)
		if system.actionKindCount(ActionStopAgent) != 1 {
			t.Fatalf("overlapping cancel stop actions=%d", system.actionKindCount(ActionStopAgent))
		}
	})
}

func TestControlsGoalAndWorkItemCancelCompletionCASBothOrders(t *testing.T) {
	for _, target := range []ControlTarget{ControlTargetGoal, ControlTargetWorkItem} {
		target := target
		t.Run(string(target)+"_completion_first", func(t *testing.T) {
			system := newControlTestSystem(t, completedObservation("completion first"))
			system.launch(t)
			system.observe(t)
			closed := system.record(t)
			item := closed.Goal.WorkItems()[0]
			itemRef := goal.WorkItemRef{}
			if target == ControlTargetWorkItem {
				itemRef = item.Ref()
			}
			request := system.request(t, "control:cancel-completion-first:"+string(target), ControlCancel, target, itemRef, goal.ExecutionRef{})
			before := system.effects(t)
			if _, err := system.orchestrator.Control(context.Background(), system.access, request); !IsStateError(err, StateConflict) {
				t.Fatalf("terminal cancel error=%v", err)
			}
			system.assertEffects(t, before)
			if closed.Goal.State() != goal.GoalStateSucceeded {
				t.Fatalf("completion-first Goal=%s", closed.Goal.State())
			}
		})

		t.Run(string(target)+"_cancel_first_with_leased_completion", func(t *testing.T) {
			system := newControlTestSystem(t, completedObservation("late completion evidence"))
			system.launch(t)
			system.clock.Advance(time.Second)
			claim := system.claim(t, "claim:late-completion:"+string(target))
			if claim.Action.Kind != ActionObserveAgent {
				t.Fatalf("claimed action=%s, want observe", claim.Action.Kind)
			}
			before := system.record(t)
			item := before.Goal.WorkItems()[0]
			itemRef := goal.WorkItemRef{}
			if target == ControlTargetWorkItem {
				itemRef = item.Ref()
			}
			request := system.request(t, "control:cancel-first:"+string(target), ControlCancel, target, itemRef, goal.ExecutionRef{})
			result, err := system.orchestrator.Control(context.Background(), system.access, request)
			if err != nil || result.Control.Status != ControlRequested {
				t.Fatalf("cancel first: result=%+v err=%v", result, err)
			}
			if err := system.orchestrator.processObservation(context.Background(), claim); err != nil {
				t.Fatalf("leased completion after cancel: %v", err)
			}
			closed := system.record(t)
			item, _ = closed.Goal.WorkItem(item.Ref())
			execution := mustBoundExecution(t, closed, item.Ref())
			control := mustControlByRequest(t, closed, request.RequestRef)
			wantGoal := goal.GoalStateCanceled
			if target == ControlTargetWorkItem {
				wantGoal = goal.GoalStateFailed
			}
			if closed.Goal.State() != wantGoal || item.State() != goal.WorkItemStateCanceled ||
				execution.State != ExecutionSucceeded || control.Status != ControlConfirmed || system.stopCount() != 0 {
				t.Fatalf("cancel-first CAS: target=%s Goal=%s item=%s execution=%s control=%s stops=%d",
					target, closed.Goal.State(), item.State(), execution.State, control.Status, system.stopCount())
			}
		})
	}
}

func TestControlsStopCompletionCASAndUnsupportedMode(t *testing.T) {
	t.Run("leased completion wins and stop settles as already completed", func(t *testing.T) {
		system := newControlTestSystem(t, completedObservation("completion beats stop"))
		system.launch(t)
		system.clock.Advance(time.Second)
		claim := system.claim(t, "claim:completion-before-stop")
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		request := system.request(t, "control:stop-completion-wins", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
		result, err := system.orchestrator.Control(context.Background(), system.access, request)
		if err != nil || result.Control.Status != ControlRequested {
			t.Fatalf("request stop: result=%+v err=%v", result, err)
		}
		if err := system.orchestrator.processObservation(context.Background(), claim); err != nil {
			t.Fatalf("leased completion: %v", err)
		}
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:settle-terminal-stop")
		if err != nil || !processed.Processed || processed.Action != ActionStopAgent {
			t.Fatalf("settle terminal stop: result=%+v err=%v", processed, err)
		}
		closed := system.record(t)
		finished, _ := executionByRef(closed.Executions, execution.Ref)
		control := mustControlByRequest(t, closed, request.RequestRef)
		if closed.Goal.State() != goal.GoalStateSucceeded || finished.State != ExecutionSucceeded ||
			control.Status != ControlConfirmed || system.stopCount() != 0 {
			t.Fatalf("completion/stop CAS: Goal=%s execution=%s control=%s physical_stops=%d",
				closed.Goal.State(), finished.State, control.Status, system.stopCount())
		}
	})

	t.Run("stop wins exact CAS", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		system.launch(t)
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		request := system.request(t, "control:stop-wins", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
		if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
			t.Fatalf("request stop: %v", err)
		}
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:stop-wins")
		if err != nil || !processed.Processed || processed.Action != ActionStopAgent {
			t.Fatalf("process stop: result=%+v err=%v", processed, err)
		}
		effects := system.effects(t)
		launchReceipt, found := acceptedLaunchReceiptForExecution(record, execution)
		if !found || len(effects.stopRequests) != 1 || effects.stopRequests[0].LaunchActionFence != launchReceipt.ActionFence {
			t.Fatalf("stop authority requests=%+v launch=%+v found=%v", effects.stopRequests, launchReceipt, found)
		}
		stopped := system.record(t)
		item, _ = stopped.Goal.WorkItem(item.Ref())
		execution, _ = executionByRef(stopped.Executions, execution.Ref)
		control := mustControlByRequest(t, stopped, request.RequestRef)
		if execution.State != ExecutionStopped || item.State() != goal.WorkItemStateInterrupted ||
			stopped.Goal.IsTerminal() || control.Status != ControlConfirmed || system.stopCount() != 1 {
			t.Fatalf("stop-first CAS: Goal=%s item=%s execution=%s control=%s stops=%d",
				stopped.Goal.State(), item.State(), execution.State, control.Status, system.stopCount())
		}
	})

	t.Run("unsupported forced mode has zero effects", func(t *testing.T) {
		capabilities := &ports.AgentControlCapabilities{CooperativeStop: true}
		system := newControlTestSystem(t, &scriptedAgent{controlCapabilities: capabilities})
		system.launch(t)
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		request := system.request(t, "control:stop-unsupported", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
		request.Mode = ports.AgentStopForced
		before := system.effects(t)
		_, err := system.orchestrator.Control(context.Background(), system.access, request)
		if err == nil || err.Error() != "application.control_stop_unsupported" {
			t.Fatalf("unsupported stop error=%v", err)
		}
		system.assertEffects(t, before)
	})
}

func TestControlsRetryCreatesFreshExecutionAndPreservesStoppedAttempt(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	stoppedRef := mustBoundExecution(t, record, item.Ref()).Ref
	stop := system.request(t, "control:retry-stop", ControlStop, ControlTargetExecution, item.Ref(), stoppedRef)
	if _, err := system.orchestrator.Control(context.Background(), system.access, stop); err != nil {
		t.Fatalf("request stop: %v", err)
	}
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:retry-stop"); err != nil || result.Action != ActionStopAgent {
		t.Fatalf("process stop: result=%+v err=%v", result, err)
	}
	stoppedRecord := system.record(t)
	stoppedAttempt, _ := executionByRef(stoppedRecord.Executions, stoppedRef)
	item, _ = stoppedRecord.Goal.WorkItem(item.Ref())
	retry := system.request(t, "control:retry-fresh", ControlRetry, ControlTargetWorkItem, item.Ref(), stoppedRef)
	result, err := system.orchestrator.Control(context.Background(), system.access, retry)
	if err != nil || !result.Created || result.Control.Status != ControlConfirmed {
		t.Fatalf("retry: result=%+v err=%v", result, err)
	}
	retried := system.record(t)
	if len(retried.Executions) != 2 {
		t.Fatalf("execution count=%d, want 2", len(retried.Executions))
	}
	preserved, _ := executionByRef(retried.Executions, stoppedRef)
	item, _ = retried.Goal.WorkItem(item.Ref())
	replacement := mustBoundExecution(t, retried, item.Ref())
	if !reflect.DeepEqual(preserved, stoppedAttempt) || replacement.Ref == stoppedRef ||
		replacement.AttemptNo != stoppedAttempt.AttemptNo+1 || replacement.ReplacesExecutionRef != stoppedRef ||
		replacement.IdempotencyKey == stoppedAttempt.IdempotencyKey || replacement.State != ExecutionQueued ||
		item.State() != goal.WorkItemStateRunning {
		t.Fatalf("retry history/replacement invalid: stopped=%+v replacement=%+v item=%s", preserved, replacement, item.State())
	}

	afterRetry := system.effects(t)
	replay, err := system.orchestrator.Control(context.Background(), system.access, retry)
	if err != nil || replay.Created || !reflect.DeepEqual(replay.Control, result.Control) {
		t.Fatalf("retry replay: result=%+v err=%v", replay, err)
	}
	system.assertEffects(t, afterRetry)
	processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:retry-launch")
	if err != nil || !processed.Processed || processed.Action != ActionLaunchAgent || system.launchCount() != 2 {
		t.Fatalf("replacement launch: result=%+v launches=%d err=%v", processed, system.launchCount(), err)
	}
}

func TestLegacyStoppedRetryRequiresReauthorizationWithoutNewEffect(t *testing.T) {
	system, itemRef, executionRef := stoppedControlSystem(t, 3)
	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	record.WorkItemAuthorities = nil
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()

	request := system.request(
		t, "control:legacy-retry-needs-reauthorization", ControlRetry,
		ControlTargetWorkItem, itemRef, executionRef,
	)
	before := system.effects(t)
	if _, err := system.orchestrator.Control(context.Background(), system.access, request); err == nil ||
		err.Error() != "governance.legacy_reauthorization_required" {
		t.Fatalf("legacy retry error=%v", err)
	}
	system.assertEffects(t, before)
}

func TestControlsRetryRejectsTerminalGoalRetiredMailboxAndAttemptLimit(t *testing.T) {
	t.Run("terminal Goal", func(t *testing.T) {
		system, itemRef, executionRef := stoppedControlSystem(t, 3)
		cancel := system.request(t, "control:terminal-cancel", ControlCancel, ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
		if _, err := system.orchestrator.Control(context.Background(), system.access, cancel); err != nil {
			t.Fatalf("cancel stopped Goal: %v", err)
		}
		retry := system.request(t, "control:terminal-retry", ControlRetry, ControlTargetWorkItem, itemRef, executionRef)
		before := system.effects(t)
		if _, err := system.orchestrator.Control(context.Background(), system.access, retry); !IsStateError(err, StateConflict) {
			t.Fatalf("terminal retry error=%v", err)
		}
		system.assertEffects(t, before)
	})

	t.Run("retired recipient mailbox marker", func(t *testing.T) {
		system, itemRef, executionRef := stoppedControlSystem(t, 3)
		system.repository.mu.Lock()
		record := system.repository.records[system.goalRef]
		execution, _ := executionByRef(record.Executions, executionRef)
		execution.RecipientMailboxRetired = true
		record.Executions = replaceExecution(record.Executions, execution)
		system.repository.records[system.goalRef] = record
		system.repository.mu.Unlock()
		retry := system.request(t, "control:retired-mailbox-retry", ControlRetry, ControlTargetWorkItem, itemRef, executionRef)
		before := system.effects(t)
		if _, err := system.orchestrator.Control(context.Background(), system.access, retry); !IsStateError(err, StateConflict) {
			t.Fatalf("retired mailbox retry error=%v", err)
		}
		system.assertEffects(t, before)
	})

	t.Run("attempt limit", func(t *testing.T) {
		system, itemRef, executionRef := stoppedControlSystem(t, 1)
		retry := system.request(t, "control:attempt-limit-retry", ControlRetry, ControlTargetWorkItem, itemRef, executionRef)
		before := system.effects(t)
		if _, err := system.orchestrator.Control(context.Background(), system.access, retry); !IsStateError(err, StateConflict) {
			t.Fatalf("attempt-limit retry error=%v", err)
		}
		system.assertEffects(t, before)
	})
}

func TestControlsExactFencesAndInvalidTargetPairsLeaveNoEffects(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	base := system.request(t, "control:invalid-base", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
	otherGoal, err := goal.NewGoalRef("goal:other")
	if err != nil {
		t.Fatal(err)
	}
	otherExecution, err := goal.NewExecutionRef("execution:other")
	if err != nil {
		t.Fatal(err)
	}
	zeroHash := strings.Repeat("0", 64)
	if zeroHash == base.ExpectedSpecHash {
		zeroHash = strings.Repeat("1", 64)
	}

	invalidFences := []struct {
		name   string
		mutate func(*ControlRequest)
	}{
		{name: "Goal ref", mutate: func(request *ControlRequest) { request.GoalRef = otherGoal }},
		{name: "Goal revision", mutate: func(request *ControlRequest) { request.ExpectedGoalRevision++ }},
		{name: "plan generation", mutate: func(request *ControlRequest) { request.ExpectedPlanGeneration++ }},
		{name: "AppSpec generation", mutate: func(request *ControlRequest) { request.ExpectedAppSpecGeneration++ }},
		{name: "spec hash", mutate: func(request *ControlRequest) { request.ExpectedSpecHash = zeroHash }},
		{name: "WorkItem revision", mutate: func(request *ControlRequest) { request.ExpectedWorkItemRevision++ }},
		{name: "Execution ref", mutate: func(request *ControlRequest) { request.ExecutionRef = otherExecution }},
		{name: "Execution attempt", mutate: func(request *ControlRequest) { request.ExpectedExecutionAttempt++ }},
	}
	for index, test := range invalidFences {
		t.Run("wrong_"+strings.ReplaceAll(test.name, " ", "_"), func(t *testing.T) {
			request := base
			request.RequestRef = "control:wrong-fence:" + string(rune('a'+index))
			test.mutate(&request)
			before := system.effects(t)
			if _, err := system.orchestrator.Control(context.Background(), system.access, request); err == nil {
				t.Fatalf("%s accepted", test.name)
			}
			system.assertEffects(t, before)
		})
	}

	itemControl := system.request(t, "control:invalid-item", ControlPause, ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
	retryControl := system.request(t, "control:invalid-retry", ControlRetry, ControlTargetWorkItem, item.Ref(), execution.Ref)
	invalidPairs := []ControlRequest{
		func() ControlRequest {
			request := itemControl
			request.RequestRef = "control:pause-execution"
			request.Target = ControlTargetExecution
			return request
		}(),
		func() ControlRequest {
			request := itemControl
			request.RequestRef = "control:resume-execution"
			request.Operation = ControlResume
			request.Target = ControlTargetExecution
			return request
		}(),
		func() ControlRequest {
			request := itemControl
			request.RequestRef = "control:cancel-execution"
			request.Operation = ControlCancel
			request.Target = ControlTargetExecution
			return request
		}(),
		func() ControlRequest {
			request := base
			request.RequestRef = "control:stop-work-item"
			request.Target = ControlTargetWorkItem
			return request
		}(),
		func() ControlRequest {
			request := retryControl
			request.RequestRef = "control:retry-execution"
			request.Target = ControlTargetExecution
			return request
		}(),
		func() ControlRequest {
			request := system.request(t, "control:generic-replan", ControlOperation("replan"), ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
			return request
		}(),
	}
	for _, request := range invalidPairs {
		request := request
		t.Run(request.RequestRef, func(t *testing.T) {
			before := system.effects(t)
			if _, err := system.orchestrator.Control(context.Background(), system.access, request); err == nil {
				t.Fatalf("invalid pair %s/%s accepted", request.Operation, request.Target)
			}
			system.assertEffects(t, before)
		})
	}
}

func TestControlsStopCrashReplayConvergesWithoutDuplicateEffect(t *testing.T) {
	t.Run("crash before physical stop", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		system.launch(t)
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		request := system.request(t, "control:crash-before-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
		if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
			t.Fatalf("request stop: %v", err)
		}
		claim := system.claim(t, "claim:crash-before-stop")
		if claim.Action.Kind != ActionStopAgent || system.stopCount() != 0 {
			t.Fatalf("pre-effect crash frontier: action=%s stops=%d", claim.Action.Kind, system.stopCount())
		}
		system.clock.Advance(2 * time.Minute)
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:recover-before-stop")
		if err != nil || !processed.Processed || processed.Action != ActionStopAgent || system.stopCount() != 1 {
			t.Fatalf("recover before effect: result=%+v stops=%d err=%v", processed, system.stopCount(), err)
		}
		stopped := system.record(t)
		if got, _ := executionByRef(stopped.Executions, execution.Ref); got.State != ExecutionStopped {
			t.Fatalf("recovered execution state=%s", got.State)
		}
	})

	t.Run("crash after physical stop before durable receipt", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		system.launch(t)
		controller := &idempotentStopController{now: system.clock.Now, receipts: make(map[string]ports.AgentStopReceipt)}
		system.orchestrator.controller = controller
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		request := system.request(t, "control:crash-after-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
		if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
			t.Fatalf("request stop: %v", err)
		}
		system.orchestrator.state = terminalEffectFaultState{StateRepository: system.repository, stop: true}

		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:crash-after-stop")
		if !processed.Processed || processed.Action != ActionStopAgent || err == nil || err.Error() != effectUnknownAppliedCode {
			t.Fatalf("crash after effect frontier: result=%+v err=%v", processed, err)
		}
		if calls, effects := controller.counts(); calls != 1 || effects != 1 {
			t.Fatalf("first controller calls/effects=%d/%d", calls, effects)
		}
		pending := system.record(t)
		if current, _ := executionByRef(pending.Executions, execution.Ref); current.State != ExecutionRunning {
			t.Fatalf("receipt failure partially mutated execution=%s", current.State)
		}

		system.clock.Advance(2 * time.Minute)
		processed, err = system.orchestrator.ProcessNext(context.Background(), "worker:recover-after-stop")
		if err != nil || (processed.Processed && processed.Action == ActionStopAgent) {
			t.Fatalf("unknown stop became schedulable: result=%+v err=%v", processed, err)
		}
		if calls, effects := controller.counts(); calls != 1 || effects != 1 {
			t.Fatalf("unknown stop was reinvoked calls/effects=%d/%d", calls, effects)
		}
		closed := system.record(t)
		current, _ := executionByRef(closed.Executions, execution.Ref)
		if current.State != ExecutionRunning || len(closed.EffectReceipts) != 1 {
			t.Fatalf("unknown stop mutated terminal frontier: execution=%s receipts=%+v",
				current.State, closed.EffectReceipts)
		}
	})
}

type controlTestSystem struct {
	clock        *mutableClock
	repository   *memoryRepository
	accessStore  *memoryAccessRepository
	agent        *scriptedAgent
	orchestrator *Orchestrator
	access       Access
	goalRef      goal.GoalRef
}

func newControlTestSystem(t *testing.T, agent *scriptedAgent) *controlTestSystem {
	return newControlTestSystemWithPlan(t, agent, nil)
}

func newControlTestSystemWithPlan(t *testing.T, agent *scriptedAgent, plan *PlanSpec) *controlTestSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)}
	if agent == nil {
		agent = &scriptedAgent{}
	}
	agent.now = clock.Now
	repository := newMemoryRepository()
	accessStore := newMemoryAccessRepository()
	accessStore.defaultRole = ""
	orchestrator, _ := newTestOrchestratorWithAccess(t, repository, accessStore, clock, agent)
	project, err := goal.NewProjectRef("project:controls")
	if err != nil {
		t.Fatal(err)
	}
	owner := testPrincipal(t, "principal:controls-owner", "actor:controls-owner", identity.PrincipalKindHuman)
	seedDirectorMembership(t, accessStore, owner, project, identity.RoleProjectOwner, clock.Now())
	access := mustDirectorAccess(t, owner, project)
	submitted, err := orchestrator.Submit(context.Background(), access, SubmitRequest{
		RequestRef: "request:controls-goal", Statement: "exercise exact durable controls", Confirm: true,
		Plan: plan,
	})
	if err != nil {
		t.Fatalf("submit control Goal: %v", err)
	}
	return &controlTestSystem{
		clock: clock, repository: repository, accessStore: accessStore, agent: agent,
		orchestrator: orchestrator, access: access, goalRef: submitted.Record.Goal.Ref(),
	}
}

func completedObservation(content string) *scriptedAgent {
	return &scriptedAgent{observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte(content),
	}}}
}

func (system *controlTestSystem) record(t *testing.T) GoalRecord {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), system.goalRef)
	if err != nil {
		t.Fatalf("get control Goal: %v", err)
	}
	return record
}

func (system *controlTestSystem) request(
	t *testing.T,
	requestRef string,
	operation ControlOperation,
	target ControlTarget,
	itemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
) ControlRequest {
	t.Helper()
	record := system.record(t)
	request := ControlRequest{
		RequestRef: requestRef, Operation: operation, Target: target, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		Reason: "test exact control " + requestRef,
	}
	if itemRef.String() != "" {
		item, found := record.Goal.WorkItem(itemRef)
		if !found {
			t.Fatalf("control WorkItem %s missing", itemRef.String())
		}
		request.WorkItemRef = itemRef
		request.ExpectedWorkItemRevision = item.Revision()
	}
	if executionRef.String() != "" {
		execution, found := executionByRef(record.Executions, executionRef)
		if !found {
			t.Fatalf("control Execution %s missing", executionRef.String())
		}
		request.ExecutionRef = executionRef
		request.ExpectedExecutionAttempt = execution.AttemptNo
	}
	if operation == ControlStop {
		request.Mode = ports.AgentStopCooperative
	}
	return request
}

func (system *controlTestSystem) launch(t *testing.T) {
	t.Helper()
	for attempts := 0; attempts < 8; attempts++ {
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:control-launch")
		if err != nil {
			t.Fatalf("launch control fixture: result=%+v err=%v", result, err)
		}
		if result.Processed && result.Action == ActionLaunchAgent {
			return
		}
	}
	t.Fatal("launch control fixture did not drain workspace preparation")
}

func (system *controlTestSystem) observe(t *testing.T) {
	t.Helper()
	system.clock.Advance(time.Second)
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:control-observe")
	if err != nil || !result.Processed || result.Action != ActionObserveAgent {
		t.Fatalf("observe control fixture: result=%+v err=%v", result, err)
	}
}

func (system *controlTestSystem) claim(t *testing.T, token string) ActionClaim {
	t.Helper()
	claim, found, err := system.repository.ClaimNextAction(context.Background(), ClaimRequest{
		WorkerRef: "worker:" + token, Token: token, LeaseDuration: time.Minute,
		Capabilities: testAgentCapabilities(),
	})
	if err != nil || !found {
		t.Fatalf("claim %s: found=%v err=%v", token, found, err)
	}
	return claim
}

func (system *controlTestSystem) launchCount() int {
	system.agent.mu.Lock()
	defer system.agent.mu.Unlock()
	return system.agent.launches
}

func (system *controlTestSystem) observationCount() int {
	system.agent.mu.Lock()
	defer system.agent.mu.Unlock()
	return system.agent.observationCalls
}

func (system *controlTestSystem) stopCount() int {
	system.agent.mu.Lock()
	defer system.agent.mu.Unlock()
	return system.agent.stopCalls
}

func (system *controlTestSystem) actionCount() int {
	system.repository.mu.Lock()
	defer system.repository.mu.Unlock()
	return len(system.repository.actions)
}

func (system *controlTestSystem) actionKindCount(kind ActionKind) int {
	system.repository.mu.Lock()
	defer system.repository.mu.Unlock()
	count := 0
	for _, action := range system.repository.actions {
		if action.record.Kind == kind {
			count++
		}
	}
	return count
}

type controlEffectSnapshot struct {
	record          GoalRecord
	actions         map[string]memoryAction
	events          []EventRecord
	controlRequests map[string]ControlReplayRequest
	controlRefs     map[string]string
	launches        int
	observations    int
	stops           int
	stopRequests    []ports.AgentStopRequest
}

func (system *controlTestSystem) effects(t *testing.T) controlEffectSnapshot {
	t.Helper()
	system.repository.mu.Lock()
	record := cloneGoalRecord(system.repository.records[system.goalRef])
	actions := make(map[string]memoryAction, len(system.repository.actions))
	for ref, action := range system.repository.actions {
		actions[ref] = action
	}
	events := append([]EventRecord(nil), system.repository.events...)
	controlRequests := make(map[string]ControlReplayRequest, len(system.repository.controlRequests))
	for key, request := range system.repository.controlRequests {
		controlRequests[key] = request
	}
	controlRefs := make(map[string]string, len(system.repository.controlRefs))
	for key, ref := range system.repository.controlRefs {
		controlRefs[key] = ref
	}
	system.repository.mu.Unlock()

	system.agent.mu.Lock()
	snapshot := controlEffectSnapshot{
		record: record, actions: actions, events: events,
		controlRequests: controlRequests, controlRefs: controlRefs,
		launches: system.agent.launches, observations: system.agent.observationCalls,
		stops: system.agent.stopCalls, stopRequests: append([]ports.AgentStopRequest(nil), system.agent.stopRequests...),
	}
	system.agent.mu.Unlock()
	return snapshot
}

func (system *controlTestSystem) assertEffects(t *testing.T, want controlEffectSnapshot) {
	t.Helper()
	if got := system.effects(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("partial control effect:\ngot=%+v\nwant=%+v", got, want)
	}
}

func mustBoundExecution(t *testing.T, record GoalRecord, itemRef goal.WorkItemRef) ExecutionRecord {
	t.Helper()
	item, found := record.Goal.WorkItem(itemRef)
	if !found {
		t.Fatalf("WorkItem %s missing", itemRef.String())
	}
	executionRef, bound := item.Execution()
	if !bound {
		t.Fatalf("WorkItem %s has no bound execution", itemRef.String())
	}
	execution, found := executionByRef(record.Executions, executionRef)
	if !found {
		t.Fatalf("bound Execution %s missing", executionRef.String())
	}
	return execution
}

func mustControlByRequest(t *testing.T, record GoalRecord, requestRef string) ControlRecord {
	t.Helper()
	for _, control := range record.Controls {
		if control.RequestRef == requestRef {
			return control
		}
	}
	t.Fatalf("control request %s missing", requestRef)
	return ControlRecord{}
}

func stoppedControlSystem(t *testing.T, maxAttempts uint64) (*controlTestSystem, goal.WorkItemRef, goal.ExecutionRef) {
	t.Helper()
	system := newControlTestSystem(t, nil)
	if maxAttempts != 3 {
		system.repository.mu.Lock()
		record := system.repository.records[system.goalRef]
		record.Executions[0].MaxExecutionAttempts = maxAttempts
		system.repository.records[system.goalRef] = record
		system.repository.mu.Unlock()
	}
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	request := system.request(t, "control:stop-for-retry:"+execution.Ref.String(), ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
	if _, err := system.orchestrator.Control(context.Background(), system.access, request); err != nil {
		t.Fatalf("request stop for retry: %v", err)
	}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:stop-for-retry")
	if err != nil || !result.Processed || result.Action != ActionStopAgent {
		t.Fatalf("stop for retry: result=%+v err=%v", result, err)
	}
	return system, item.Ref(), execution.Ref
}

type idempotentStopController struct {
	mu       sync.Mutex
	now      func() time.Time
	receipts map[string]ports.AgentStopReceipt
	requests []ports.AgentStopRequest
	effects  int
}

func (controller *idempotentStopController) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (controller *idempotentStopController) Stop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.requests = append(controller.requests, request)
	if receipt, found := controller.receipts[request.IdempotencyKey]; found {
		return receipt, nil
	}
	controller.effects++
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: ports.AgentStopped, ReceiptRef: "receipt:idempotent:" + request.ExecutionRef.String(),
		ConfirmedAt: controller.now().UTC(),
	}
	controller.receipts[request.IdempotencyKey] = receipt
	return receipt, nil
}

func (controller *idempotentStopController) counts() (int, int) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return len(controller.requests), controller.effects
}

func (controller *idempotentStopController) snapshot() struct {
	Requests []ports.AgentStopRequest
	Effects  int
} {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return struct {
		Requests []ports.AgentStopRequest
		Effects  int
	}{Requests: append([]ports.AgentStopRequest(nil), controller.requests...), Effects: controller.effects}
}
