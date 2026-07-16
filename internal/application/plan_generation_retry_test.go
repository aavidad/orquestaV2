package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestDirectorAppendPreservesBirthGenerationAcrossExecutionRetry(t *testing.T) {
	ctx := context.Background()
	system := newDirectorTestSystem(t)
	agent := system.orchestrator.observer.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{
		{Status: ports.AgentFailed, ErrorCode: "agent.retry_after_plan_append"},
		{Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("retried generation-one result")},
		{Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("generation-two result")},
	}
	agent.mu.Unlock()

	if result, err := system.orchestrator.ProcessNext(ctx, "worker:birth-launch"); err != nil ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("initial launch: result=%+v err=%v", result, err)
	}
	before, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	oldItem := before.Goal.WorkItems()[0]
	lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:birth-retry")
	proposal, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, ProposeDirectorPlanRequest{
		RequestRef: "director-plan:birth-retry", GoalRef: before.Goal.Ref(),
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		LeaseToken: lease.Token, LeaseFence: lease.Fence,
		Reason: "append dependent work before the original execution retries",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "work:after-retried-birth", Objective: "run after retried generation-one work",
			Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
			Dependencies: []string{oldItem.Ref().String()}, OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	if err != nil || !proposal.Created || proposal.Decision.AppliedPlanGeneration != 2 {
		t.Fatalf("append plan: result=%+v err=%v", proposal, err)
	}

	if result, err := system.orchestrator.ProcessNext(ctx, "worker:birth-failed-observe"); err != nil ||
		!result.Processed || result.Action != ActionObserveAgent {
		t.Fatalf("failed old observation: result=%+v err=%v", result, err)
	}
	retried, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if retried.Goal.PlanGeneration() != 2 || len(retried.Executions) != 2 ||
		retried.Executions[0].State != ExecutionFailed || retried.Executions[0].PlanGeneration != 1 ||
		retried.Executions[1].State != ExecutionDispatching || retried.Executions[1].PlanGeneration != 1 {
		t.Fatalf("replacement changed birth generation: goal=%d executions=%+v",
			retried.Goal.PlanGeneration(), retried.Executions)
	}
	system.repository.mu.Lock()
	var replacementAction ActionRecord
	for _, stored := range system.repository.actions {
		if stored.record.ExecutionRef == retried.Executions[1].Ref {
			replacementAction = stored.record
		}
	}
	system.repository.mu.Unlock()
	if replacementAction.Kind != ActionLaunchAgent || replacementAction.PlanGeneration != 1 {
		t.Fatalf("replacement action lost birth generation: %+v", replacementAction)
	}

	system.clock.Advance(time.Second)
	for index, want := range []ActionKind{
		ActionLaunchAgent, ActionObserveAgent, ActionLaunchAgent, ActionObserveAgent,
	} {
		result, processErr := system.orchestrator.ProcessNext(ctx, "worker:birth-retry")
		if processErr != nil || !result.Processed || result.Action != want {
			t.Fatalf("post-retry step %d want=%s result=%+v err=%v", index+1, want, result, processErr)
		}
	}
	closed, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if closed.Goal.State() != goal.GoalStateSucceeded || closed.Goal.PlanGeneration() != 2 ||
		len(closed.Executions) != 3 || closed.Executions[1].PlanGeneration != 1 ||
		closed.Executions[2].PlanGeneration != 2 {
		t.Fatalf("birth-generation retry did not close: goal=%+v executions=%+v",
			closed.Goal.Snapshot(), closed.Executions)
	}
}
