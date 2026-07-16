package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestDirectorAppendPreservesOlderObserveActionUntilGoalClosure(t *testing.T) {
	ctx := context.Background()
	system := newDirectorTestSystem(t)
	agent, ok := system.orchestrator.observer.(*scriptedAgent)
	if !ok {
		t.Fatal("test observer is not scriptedAgent")
	}
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{
		{Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("first generation result")},
		{Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("second generation result")},
	}
	agent.mu.Unlock()

	launched, err := system.orchestrator.ProcessNext(ctx, "worker:old-launch")
	if err != nil || !launched.Processed || launched.Action != ActionLaunchAgent {
		t.Fatalf("launch old generation: result=%+v err=%v", launched, err)
	}
	beforeAppend, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if beforeAppend.Goal.PlanGeneration() != 1 || len(beforeAppend.Executions) != 1 ||
		beforeAppend.Executions[0].PlanGeneration != 1 || beforeAppend.Executions[0].State != ExecutionRunning {
		t.Fatalf("old observe precondition: goal_generation=%d executions=%+v",
			beforeAppend.Goal.PlanGeneration(), beforeAppend.Executions)
	}
	oldItem := beforeAppend.Goal.WorkItems()[0]
	oldExecution := beforeAppend.Executions[0]
	lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:append-after-launch")
	proposal, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, ProposeDirectorPlanRequest{
		RequestRef: "director-plan:append-after-launch", GoalRef: beforeAppend.Goal.Ref(),
		ExpectedGoalRevision:   beforeAppend.Goal.Revision(),
		ExpectedPlanGeneration: beforeAppend.Goal.PlanGeneration(),
		LeaseToken:             lease.Token,
		LeaseFence:             lease.Fence,
		Reason:                 "append dependent work while old observation remains live",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "work:after-old-observation", Objective: "finish after old observation",
			Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
			Dependencies: []string{oldItem.Ref().String()}, OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	if err != nil || !proposal.Created || proposal.Decision.AppliedPlanGeneration != 2 {
		t.Fatalf("append plan: result=%+v err=%v", proposal, err)
	}
	afterAppend, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if afterAppend.Goal.PlanGeneration() != 2 || len(afterAppend.Executions) != 1 ||
		afterAppend.Executions[0].Ref != oldExecution.Ref || afterAppend.Executions[0].PlanGeneration != 1 {
		t.Fatalf("append rewrote old execution: goal_generation=%d executions=%+v",
			afterAppend.Goal.PlanGeneration(), afterAppend.Executions)
	}

	system.clock.Advance(time.Second)
	observed, err := system.orchestrator.ProcessNext(ctx, "worker:old-observe")
	if err != nil || !observed.Processed || observed.Action != ActionObserveAgent {
		t.Fatalf("observe old generation after append: result=%+v err=%v", observed, err)
	}
	afterOldObservation, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	oldItemAfter, found := afterOldObservation.Goal.WorkItem(oldItem.Ref())
	if !found || oldItemAfter.State() != goal.WorkItemStateSucceeded ||
		afterOldObservation.Goal.State() != goal.GoalStateRunning || len(afterOldObservation.Executions) != 2 ||
		afterOldObservation.Executions[0].State != ExecutionSucceeded ||
		afterOldObservation.Executions[1].PlanGeneration != 2 ||
		afterOldObservation.Executions[1].State != ExecutionQueued {
		t.Fatalf("old observation did not advance append-only Goal: goal=%+v executions=%+v",
			afterOldObservation.Goal.Snapshot(), afterOldObservation.Executions)
	}

	launched, err = system.orchestrator.ProcessNext(ctx, "worker:new-launch")
	if err != nil || !launched.Processed || launched.Action != ActionLaunchAgent {
		t.Fatalf("launch appended work: result=%+v err=%v", launched, err)
	}
	system.clock.Advance(time.Second)
	observed, err = system.orchestrator.ProcessNext(ctx, "worker:new-observe")
	if err != nil || !observed.Processed || observed.Action != ActionObserveAgent {
		t.Fatalf("observe appended work: result=%+v err=%v", observed, err)
	}
	closed, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if closed.Goal.State() != goal.GoalStateSucceeded || closed.Goal.PlanGeneration() != 2 ||
		len(closed.Executions) != 2 || len(closed.Artifacts) != 2 || len(closed.Attestations) != 2 {
		t.Fatalf("append-only Goal did not close: goal=%+v executions=%+v artifacts=%d attestations=%d",
			closed.Goal.Snapshot(), closed.Executions, len(closed.Artifacts), len(closed.Attestations))
	}
}

func TestClaimValidationFencesActionAndExecutionGenerationTogether(t *testing.T) {
	ctx := context.Background()
	system := newDirectorTestSystem(t)
	launched, err := system.orchestrator.ProcessNext(ctx, "worker:generation-validation")
	if err != nil || !launched.Processed || launched.Action != ActionLaunchAgent {
		t.Fatalf("launch: result=%+v err=%v", launched, err)
	}
	record, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	var observe ActionRecord
	system.repository.mu.Lock()
	for _, stored := range system.repository.actions {
		if stored.record.GoalRef == record.Goal.Ref() && stored.record.Kind == ActionObserveAgent {
			observe = stored.record
			break
		}
	}
	system.repository.mu.Unlock()
	if observe.Ref == "" {
		t.Fatal("observe action missing")
	}
	claim := ActionClaim{
		Action: observe, Token: "claim:generation-validation", WorkerRef: "worker:generation-validation",
		DeliveryAttempt: 1, Fence: 1, LeaseUntil: system.clock.Now().Add(time.Minute),
	}
	if err := validateClaimedRecord(claim, record, ActionObserveAgent); err != nil {
		t.Fatalf("valid generation pair rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*ActionClaim, *GoalRecord)
	}{
		{name: "zero action generation", mutate: func(claim *ActionClaim, _ *GoalRecord) {
			claim.Action.PlanGeneration = 0
		}},
		{name: "zero execution generation", mutate: func(_ *ActionClaim, record *GoalRecord) {
			record.Executions[0].PlanGeneration = 0
		}},
		{name: "action execution generation mismatch", mutate: func(_ *ActionClaim, record *GoalRecord) {
			record.Executions[0].PlanGeneration++
		}},
		{name: "future action and execution generation", mutate: func(claim *ActionClaim, record *GoalRecord) {
			claim.Action.PlanGeneration++
			record.Executions[0].PlanGeneration++
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidateClaim := claim
			candidateRecord := record
			candidateRecord.Executions = append([]ExecutionRecord(nil), record.Executions...)
			test.mutate(&candidateClaim, &candidateRecord)
			if err := validateClaimedRecord(candidateClaim, candidateRecord, ActionObserveAgent); err == nil {
				t.Fatal("invalid generation relation accepted")
			}
		})
	}
}
