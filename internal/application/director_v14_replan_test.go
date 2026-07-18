package application

import (
	"context"
	"reflect"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestDirectorProposeReplanSplitStoppedAndFailedSources(t *testing.T) {
	for _, test := range []struct {
		name   string
		cause  goal.ReplanCause
		setup  func(*testing.T, *directorTestSystem) GoalRecord
		before ExecutionState
		after  ExecutionState
	}{
		{
			name: "split pending", cause: goal.ReplanCauseSplitPending, before: ExecutionQueued, after: ExecutionCanceled,
			setup: func(t *testing.T, system *directorTestSystem) GoalRecord {
				return directorCurrentRecord(t, system)
			},
		},
		{
			name: "execution stopped", cause: goal.ReplanCauseExecutionStopped, before: ExecutionStopped, after: ExecutionStopped,
			setup: func(t *testing.T, system *directorTestSystem) GoalRecord {
				if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v14-replan-launch"); err != nil ||
					!result.Processed || result.Action != ActionLaunchAgent {
					t.Fatalf("launch stopped source: result=%+v err=%v", result, err)
				}
				running := directorCurrentRecord(t, system)
				item := running.Goal.WorkItems()[0]
				execution := mustBoundExecution(t, running, item.Ref())
				request := directorControlRequest(running, "control:v14-replan-stop", ControlStop,
					ControlTargetExecution, item.Ref(), execution.Ref)
				request.Mode = ports.AgentStopCooperative
				if _, err := system.orchestrator.Control(context.Background(), system.ownerAccess, request); err != nil {
					t.Fatal(err)
				}
				if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v14-replan-stop"); err != nil ||
					!result.Processed || result.Action != ActionStopAgent {
					t.Fatalf("stop source: result=%+v err=%v", result, err)
				}
				return directorCurrentRecord(t, system)
			},
		},
		{
			name: "execution failed", cause: goal.ReplanCauseExecutionFailed, before: ExecutionFailed, after: ExecutionFailed,
			setup: func(t *testing.T, system *directorTestSystem) GoalRecord {
				system.repository.mu.Lock()
				stored := system.repository.records[system.goal.Goal.Ref()]
				stored.Executions[0].MaxExecutionAttempts = 1
				system.repository.records[system.goal.Goal.Ref()] = stored
				system.repository.mu.Unlock()
				agent, ok := system.orchestrator.launcher.(*scriptedAgent)
				if !ok {
					t.Fatal("Director test launcher is not scriptedAgent")
				}
				agent.mu.Lock()
				agent.launchErr = definitelyUnappliedPermanentError{"permanent launch failure"}
				agent.mu.Unlock()
				if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v14-replan-fail"); err != nil || !result.Processed {
					t.Fatalf("fail source: result=%+v err=%v", result, err)
				}
				return directorCurrentRecord(t, system)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			system := newDirectorTestSystem(t)
			lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:v14-replan:"+test.name)
			current := test.setup(t, system)
			source := current.Goal.WorkItems()[0]
			execution, found := executionByRef(current.Executions, func() goal.ExecutionRef {
				if ref, bound := source.Execution(); bound {
					return ref
				}
				return current.Executions[0].Ref
			}())
			if !found || execution.State != test.before {
				t.Fatalf("source execution=%+v found=%v want=%s", execution, found, test.before)
			}
			request := directorReplanRequest(current, lease, test.cause, source, execution,
				"director-plan:v14-replan:"+test.name)
			result, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.ownerAccess, request)
			if err != nil || !result.Created || result.Decision.Cause != test.cause {
				t.Fatalf("ProposeDirectorPlan: result=%+v err=%v", result, err)
			}
			after := directorCurrentRecord(t, system)
			updatedSource, _ := after.Goal.WorkItem(source.Ref())
			if updatedSource.State() != goal.WorkItemStateSuperseded || len(after.Goal.WorkItems()) != 2 {
				t.Fatalf("source/items after replan: source=%s items=%d", updatedSource.State(), len(after.Goal.WorkItems()))
			}
			successor := after.Goal.WorkItems()[1]
			if reworkOf, linked := successor.ReworkOf(); !linked || reworkOf != source.Ref() ||
				!controlExecutionForItem(after, successor.Ref()) {
				t.Fatalf("successor not causally scheduled: successor=%+v executions=%+v", successor, after.Executions)
			}
			retired, _ := executionByRef(after.Executions, execution.Ref)
			if retired.State != test.after {
				t.Fatalf("causal execution state=%s want=%s", retired.State, test.after)
			}
			stable := system.snapshot(t)
			replay, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.ownerAccess, request)
			if err != nil || replay.Created || !reflect.DeepEqual(replay.Decision, result.Decision) {
				t.Fatalf("replan replay: result=%+v err=%v", replay, err)
			}
			system.assertState(t, stable)
			if stable.actionCount != 1 {
				t.Fatalf("replan active action count=%d want=1", stable.actionCount)
			}
		})
	}
}

func directorCurrentRecord(t *testing.T, system *directorTestSystem) GoalRecord {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func directorReplanRequest(
	record GoalRecord,
	lease DirectorLeaseRecord,
	cause goal.ReplanCause,
	source goal.WorkItem,
	execution ExecutionRecord,
	requestRef string,
) ProposeDirectorPlanRequest {
	return ProposeDirectorPlanRequest{
		RequestRef: requestRef, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		LeaseToken: lease.Token, LeaseFence: lease.Fence, Cause: cause,
		SourceWorkItemRef: source.Ref(), ExpectedWorkItemRevision: source.Revision(),
		SourceExecutionRef: execution.Ref, SourceExecutionAttempt: execution.AttemptNo,
		Reason: "replace exact V14 source through Director application port",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "successor", Objective: "causal successor", Phase: source.Phase().String(),
			Role: source.Role().String(), OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
}

func directorControlRequest(
	record GoalRecord,
	requestRef string,
	operation ControlOperation,
	target ControlTarget,
	itemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
) ControlRequest {
	item, _ := record.Goal.WorkItem(itemRef)
	execution, _ := executionByRef(record.Executions, executionRef)
	return ControlRequest{
		RequestRef: requestRef, Operation: operation, Target: target, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		WorkItemRef: itemRef, ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: executionRef, ExpectedExecutionAttempt: execution.AttemptNo,
		Reason: "prepare exact V14 Director source",
	}
}
