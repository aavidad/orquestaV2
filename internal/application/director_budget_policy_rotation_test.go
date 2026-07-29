package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestDirectorReplanAfterBudgetPolicyRotationInheritsCausalDemandAndOutputLimit(t *testing.T) {
	ctx := context.Background()
	system := newDirectorBudgetRotationTestSystem(t)
	system.processCommit(t)
	system.process(t, ActionAttestTest)

	before := system.record(t)
	source := before.Goal.WorkItems()[0]
	sourceExecutionRef, bound := source.Execution()
	sourceExecution, executionFound := executionByRef(before.Executions, sourceExecutionRef)
	if !bound || !executionFound || sourceExecution.MaxOutputBytes != 1<<20 ||
		source.BudgetDemand().Resources.DiskBytes != sourceExecution.MaxOutputBytes {
		t.Fatalf("historical source demand/execution mismatch: item=%+v execution=%+v", source, sourceExecution)
	}
	historicalPolicy := system.orchestrator.budgetPolicy
	rotatedPolicy := budgetPolicyVariant(
		historicalPolicy, "director-output-64mib", 64, time.Second, time.Hour, 1_000_000,
	)
	rotatedPolicy.DefaultWorkItemDemand.DiskBytes = 64 << 20
	system.orchestrator.budgetPolicy = rotatedPolicy
	system.orchestrator.maxOutputBytes = 64 << 20

	leaseResult, err := system.orchestrator.ClaimDirector(ctx, system.access, ClaimDirectorRequest{
		RequestRef: "director-claim:rotated-budget-replan", GoalRef: before.Goal.Ref(),
	})
	appTestNoError(t, err)
	request := ProposeDirectorPlanRequest{
		RequestRef: "director-plan:rotated-budget-replan", GoalRef: before.Goal.Ref(),
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		LeaseToken: leaseResult.Lease.Token, LeaseFence: leaseResult.Lease.Fence,
		Cause: goal.ReplanCauseExecutionFailed, SourceWorkItemRef: source.Ref(),
		ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: sourceExecution.Ref,
		SourceExecutionAttempt: sourceExecution.AttemptNo, Reason: "repair under the source budget contract",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "work:rotated-budget-successor", Objective: "repair the failed candidate",
			Phase: source.Phase().String(), Role: source.Role().String(),
			WriteSet: []string{"internal/rotated-budget-successor"}, CouncilPolicy: council.PolicyAuto,
			RequiredTests:  requiredTestSpecs("required-test:rotated-budget-successor"),
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	result, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || !result.Created {
		t.Fatalf("rotated replan=%+v err=%v", result, err)
	}

	after := system.record(t)
	successor := after.Goal.WorkItems()[len(after.Goal.WorkItems())-1]
	successorExecution, found := executionForWorkItem(after.Executions, successor.Ref())
	if !found || successor.BudgetDemand().Resources != source.BudgetDemand().Resources ||
		successorExecution.MaxOutputBytes != sourceExecution.MaxOutputBytes ||
		successorExecution.MaxOutputBytes == system.orchestrator.maxOutputBytes {
		t.Fatalf("rotated successor leaked current defaults: item=%+v execution=%+v", successor, successorExecution)
	}
	intent, found := effectIntentForExecution(after.EffectIntents, successorExecution.Ref)
	if !found || intent.PolicyHash != historicalPolicy.PolicyHash {
		t.Fatalf("rotated successor intent lost historical policy: %+v", intent)
	}

	replay, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || replay.Created || replay.Decision != result.Decision {
		t.Fatalf("rotated replan replay=%+v err=%v", replay, err)
	}
}

func TestDelayedReplanSuccessorRecoversSourceOutputLimitAfterRuntimeRotation(t *testing.T) {
	ctx := context.Background()
	system := newDirectorBudgetRotationTestSystem(t)
	system.processCommit(t)
	system.process(t, ActionAttestTest)

	before := system.record(t)
	source := before.Goal.WorkItems()[0]
	sourceExecutionRef, bound := source.Execution()
	sourceExecution, executionFound := executionByRef(before.Executions, sourceExecutionRef)
	if !bound || !executionFound || sourceExecution.MaxOutputBytes != 1<<20 {
		t.Fatalf("historical source execution missing: item=%+v execution=%+v", source, sourceExecution)
	}
	historicalPolicy := system.orchestrator.budgetPolicy
	leaseResult, err := system.orchestrator.ClaimDirector(ctx, system.access, ClaimDirectorRequest{
		RequestRef: "director-claim:delayed-rotated-replan", GoalRef: before.Goal.Ref(),
	})
	appTestNoError(t, err)
	request := ProposeDirectorPlanRequest{
		RequestRef: "director-plan:delayed-rotated-replan", GoalRef: before.Goal.Ref(),
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		LeaseToken: leaseResult.Lease.Token, LeaseFence: leaseResult.Lease.Fence,
		Cause: goal.ReplanCauseExecutionFailed, SourceWorkItemRef: source.Ref(),
		ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: sourceExecution.Ref,
		SourceExecutionAttempt: sourceExecution.AttemptNo, Reason: "delay one successor behind causal work",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{
			{
				Key: "work:rotated-budget-gate", Objective: "complete the causal gate",
				Phase: source.Phase().String(), Role: source.Role().String(),
				OutputContract: goal.OutputContractEvidenceBundle,
			},
			{
				Key: "work:delayed-rotated-budget-successor", Objective: "repair after the causal gate",
				Phase: source.Phase().String(), Role: source.Role().String(),
				Dependencies: []string{"work:rotated-budget-gate"},
				WriteSet:     []string{"internal/delayed-rotated-budget-successor"}, CouncilPolicy: council.PolicyAuto,
				RequiredTests:  requiredTestSpecs("required-test:delayed-rotated-budget-successor"),
				OutputContract: goal.OutputContractEvidenceBundle,
			},
		}},
	}
	result, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || !result.Created {
		t.Fatalf("delayed replan=%+v err=%v", result, err)
	}
	proposed := system.record(t)
	var delayed goal.WorkItem
	for _, item := range proposed.Goal.WorkItems() {
		if item.Objective() == "repair after the causal gate" {
			delayed = item
		}
	}
	if delayed.Ref().String() == "" {
		t.Fatal("delayed successor missing")
	}
	if _, scheduled := executionForWorkItem(proposed.Executions, delayed.Ref()); scheduled {
		t.Fatal("dependency-blocked successor scheduled during propose")
	}

	rotatedPolicy := budgetPolicyVariant(
		historicalPolicy, "delayed-director-output-64mib", 65, time.Second, time.Hour, 1_000_000,
	)
	rotatedPolicy.DefaultWorkItemDemand.DiskBytes = 64 << 20
	system.orchestrator.budgetPolicy = rotatedPolicy
	system.orchestrator.maxOutputBytes = 64 << 20
	agent, ok := system.orchestrator.launcher.(*scriptedAgent)
	if !ok {
		t.Fatalf("launcher type=%T", system.orchestrator.launcher)
	}
	agent.mu.Lock()
	agent.observations = append(agent.observations, ports.AgentObservation{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("gate complete"),
	})
	agent.mu.Unlock()
	system.process(t, ActionLaunchAgent, ActionObserveAgent)

	after := system.record(t)
	delayedExecution, found := executionForWorkItem(after.Executions, delayed.Ref())
	if !found || delayedExecution.MaxOutputBytes != sourceExecution.MaxOutputBytes ||
		delayedExecution.MaxOutputBytes == system.orchestrator.maxOutputBytes {
		t.Fatalf("late scheduler lost source output limit: %+v", delayedExecution)
	}
	intent, found := effectIntentForExecution(after.EffectIntents, delayedExecution.Ref)
	if !found || intent.PolicyHash != historicalPolicy.PolicyHash {
		t.Fatalf("late scheduler lost source policy: %+v", intent)
	}
}

func newDirectorBudgetRotationTestSystem(t *testing.T) *testAttestationSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 29, 2, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	accessStore := newMemoryAccessRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("candidate output"),
	}}}
	orchestrator, _ := newTestOrchestratorWithAccess(t, repository, accessStore, clock, agent)
	historicalPolicy := orchestrator.budgetPolicy
	historicalPolicy.DefaultWorkItemDemand.DiskBytes = 1 << 20
	orchestrator.budgetPolicy = historicalPolicy
	control := &scriptedVersionControl{}
	attestor := &scriptedTestAttestor{verdict: ports.TestAttestationFailed}
	orchestrator.versionControl, orchestrator.testAttestor = control, attestor
	project, err := goal.NewProjectRef("project:director-budget-rotation")
	appTestNoError(t, err)
	owner := testPrincipal(
		t, "principal:director-budget-rotation", "actor:director-budget-rotation", identity.PrincipalKindHuman,
	)
	seedDirectorMembership(t, accessStore, owner, project, identity.RoleProjectOwner, clock.Now())
	access := mustDirectorAccess(t, owner, project)
	submitted, err := orchestrator.Submit(context.Background(), access, SubmitRequest{
		RequestRef: "request:director-budget-rotation", Statement: "produce and attest a rotatable candidate",
		Confirm: true, Plan: workspaceWritePlan(),
	})
	appTestNoError(t, err)
	return &testAttestationSystem{
		orchestrator: orchestrator, repository: repository, control: control,
		attestor: attestor, access: access, goalRef: submitted.Record.Goal.Ref(),
	}
}

func executionForWorkItem(records []ExecutionRecord, ref goal.WorkItemRef) (ExecutionRecord, bool) {
	for _, record := range records {
		if record.WorkItemRef == ref {
			return record, true
		}
	}
	return ExecutionRecord{}, false
}

func effectIntentForExecution(records []EffectIntent, ref goal.ExecutionRef) (EffectIntent, bool) {
	for _, record := range records {
		if record.Subject.ExecutionRef == ref {
			return record, true
		}
	}
	return EffectIntent{}, false
}
