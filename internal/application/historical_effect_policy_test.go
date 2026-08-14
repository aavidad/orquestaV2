package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestHistoricalPolicySchedulesUnlockedDAGAfterRuntimeRotation(t *testing.T) {
	base := time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: base}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("root complete"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	oldPolicy := budgetPolicyVariant(testBudgetPolicy(base), "dag-old", 31, 7*time.Second, 7*time.Minute, 1_000_000)
	newPolicy := budgetPolicyVariant(testBudgetPolicy(base), "dag-new", 32, time.Second, time.Minute, 1_000_000)
	orchestrator.budgetPolicy = oldPolicy
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:historical-dag", Statement: "unlock dependent under historical policy", Confirm: true,
		Plan: controlDependencyPlan(),
	})
	if err != nil {
		t.Fatal(err)
	}
	orchestrator.budgetPolicy = newPolicy
	for _, want := range []ActionKind{ActionLaunchAgent, ActionObserveAgent} {
		result, processErr := orchestrator.ProcessNext(context.Background(), "worker:historical-dag")
		if processErr != nil || !result.Processed || result.Action != want {
			t.Fatalf("process %s: result=%+v err=%v", want, result, processErr)
		}
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || len(record.Executions) != 2 || len(record.EffectIntents) != 2 {
		t.Fatalf("dependent not scheduled: executions=%d intents=%d err=%v",
			len(record.Executions), len(record.EffectIntents), err)
	}
	assertIntentPolicy(t, record.EffectIntents[1], oldPolicy)
}

func TestDirectorExtensionKeepsHistoricalPolicyAfterRuntimeRotation(t *testing.T) {
	ctx := context.Background()
	system := newDirectorTestSystem(t)
	oldPolicy := system.orchestrator.budgetPolicy
	if result, err := system.orchestrator.ProcessNext(ctx, "worker:director-old-launch"); err != nil ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("initial launch: result=%+v err=%v", result, err)
	}
	current, err := system.repository.GetGoal(ctx, system.goal.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:historical-policy")
	system.orchestrator.budgetPolicy = budgetPolicyVariant(
		testBudgetPolicy(system.clock.Now()), "director-current-small", 62, time.Second, time.Minute, 100,
	)
	demand := governance.BudgetDemand{
		Ref: "budget-demand:director-historical", Resources: governance.ResourceVector{Tokens: 500, ProcessSlots: 1},
	}
	proposal, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, ProposeDirectorPlanRequest{
		RequestRef: "director-plan:historical-policy", GoalRef: current.Goal.Ref(),
		ExpectedGoalRevision: current.Goal.Revision(), ExpectedPlanGeneration: current.Goal.PlanGeneration(),
		LeaseToken: lease.Token, LeaseFence: lease.Fence, Reason: "extend under the Goal policy",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "work:historical-policy", Objective: "historical policy work",
			Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
			OutputContract: goal.OutputContractEvidenceBundle, BudgetDemand: demand,
		}}},
	})
	if err != nil || !proposal.Created {
		t.Fatalf("historical Director proposal=%+v err=%v", proposal, err)
	}
	record, _ := system.repository.GetGoal(ctx, current.Goal.Ref())
	var newItem goal.WorkItem
	for _, item := range record.Goal.WorkItems() {
		if item.Objective() == "historical policy work" {
			newItem = item
		}
	}
	intent := record.EffectIntents[len(record.EffectIntents)-1]
	if newItem.Ref().String() == "" || newItem.BudgetDemand() != demand || intent.Subject.WorkItemRef != newItem.Ref() {
		t.Fatalf("historical item/intent=%+v/%+v", newItem, intent)
	}
	assertIntentPolicy(t, intent, oldPolicy)
	if result, err := system.orchestrator.ProcessNext(ctx, "worker:director-historical-policy"); err != nil ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("historical Director launch: result=%+v err=%v", result, err)
	}
	record, _ = system.repository.GetGoal(ctx, current.Goal.Ref())
	reservation := record.BudgetReservations[len(record.BudgetReservations)-1]
	if reservation.PolicyHash != oldPolicy.PolicyHash || reservation.WorkItemRef != newItem.Ref().String() {
		t.Fatalf("historical reservation=%+v", reservation)
	}
}

func TestHistoricalReplacementAndStopRetryKeepOriginalPolicy(t *testing.T) {
	t.Run("automatic replacement", func(t *testing.T) {
		base := time.Date(2026, 7, 18, 19, 0, 0, 0, time.UTC)
		clock := &mutableClock{now: base}
		repository := newMemoryRepository()
		agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
			Status: ports.AgentFailed, ErrorCode: "agent.historical-retry",
		}}}
		orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
		oldPolicy := budgetPolicyVariant(testBudgetPolicy(base), "retry-old", 41, 9*time.Second, 9*time.Minute, 1_000_000)
		newPolicy := budgetPolicyVariant(testBudgetPolicy(base), "retry-new", 42, time.Second, time.Minute, 1_000_000)
		orchestrator.budgetPolicy = oldPolicy
		actor, project := testScope(t)
		submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
			RequestRef: "request:historical-retry", Statement: "retry under original policy", Confirm: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = orchestrator.ProcessNext(context.Background(), "worker:historical-retry-launch"); err != nil {
			t.Fatal(err)
		}
		orchestrator.budgetPolicy = newPolicy
		if _, err = orchestrator.ProcessNext(context.Background(), "worker:historical-retry-observe"); err != nil {
			t.Fatal(err)
		}
		record, _ := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
		if len(record.EffectIntents) != 2 || len(record.Executions) != 2 {
			t.Fatalf("replacement facts=%d/%d", len(record.EffectIntents), len(record.Executions))
		}
		assertIntentPolicy(t, record.EffectIntents[1], oldPolicy)
	})

	t.Run("stop and explicit retry", func(t *testing.T) {
		system := newControlTestSystem(t, nil)
		oldPolicy := system.orchestrator.budgetPolicy
		system.launch(t)
		system.orchestrator.budgetPolicy = budgetPolicyVariant(
			testBudgetPolicy(system.clock.Now()), "control-new", 52, 2*time.Second, 2*time.Minute, 1_000_000,
		)
		record := system.record(t)
		item := record.Goal.WorkItems()[0]
		execution := mustBoundExecution(t, record, item.Ref())
		stop := system.request(t, "control:historical-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
		if _, err := system.orchestrator.Control(context.Background(), system.access, stop); err != nil {
			t.Fatal(err)
		}
		assertIntentPolicy(t, system.record(t).EffectIntents[1], oldPolicy)
		if _, err := system.orchestrator.ProcessNext(context.Background(), "worker:historical-stop"); err != nil {
			t.Fatal(err)
		}
		stopped := system.record(t)
		item, _ = stopped.Goal.WorkItem(item.Ref())
		retry := system.request(t, "control:historical-explicit-retry", ControlRetry, ControlTargetWorkItem, item.Ref(), execution.Ref)
		if _, err := system.orchestrator.Control(context.Background(), system.access, retry); err != nil {
			t.Fatal(err)
		}
		assertIntentPolicy(t, system.record(t).EffectIntents[2], oldPolicy)
	})
}

func TestHistoricalPolicyRejectsPartialOrCorruptStateWithoutRuntimeFallback(t *testing.T) {
	base := time.Date(2026, 7, 18, 20, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: base}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:historical-corruption", Statement: "reject incomplete policy", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	record := submitted.Record
	partial := record
	partial.BudgetEnvelopes = partial.BudgetEnvelopes[:2]
	corrupt := record
	corrupt.EffectIntents = append([]EffectIntent(nil), record.EffectIntents...)
	corrupt.EffectIntents[0].ApprovalTTL++
	for name, candidate := range map[string]GoalRecord{"partial": partial, "corrupt": corrupt} {
		t.Run(name, func(t *testing.T) {
			_, _, _, err := orchestrator.scheduleHistoricalReady(
				context.Background(), candidate, candidate.Goal, candidate.Executions, base,
			)
			if err == nil || err.Error() != "application.goal_effect_policy_invalid" {
				t.Fatalf("historical policy fallback: %v", err)
			}
		})
	}
}

func TestLegacyGovernanceParkingDoesNotInventStopAuthority(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	record.BudgetEnvelopes, record.WorkItemAuthorities, record.EffectIntents = nil, nil, nil
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()

	parkedExecutions, parkedActions, _, err := system.orchestrator.scheduleHistoricalReady(
		context.Background(), record, record.Goal, nil, system.clock.Now(),
	)
	if err != nil || len(parkedExecutions) != 0 || len(parkedActions) != 0 {
		t.Fatalf("legacy work was synthesized: executions=%d actions=%d err=%v",
			len(parkedExecutions), len(parkedActions), err)
	}
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	stop := system.request(t, "control:legacy-authorized-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref)
	if _, err = system.orchestrator.Control(context.Background(), system.access, stop); err != nil {
		t.Fatal(err)
	}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:legacy-authorized-stop")
	if err == nil || err.Error() != "application.agent_stop_authority_invalid" ||
		!result.Processed || result.Action != ActionStopAgent || system.stopCount() != 0 {
		t.Fatalf("legacy stop authority: result=%+v calls=%d err=%v", result, system.stopCount(), err)
	}
}

func TestLegacyGovernanceDirectorExtensionRequiresReauthorizationWithoutWrite(t *testing.T) {
	system := newDirectorTestSystem(t)
	lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:legacy-governance")
	system.repository.mu.Lock()
	record := system.repository.records[system.goal.Goal.Ref()]
	record.BudgetEnvelopes, record.WorkItemAuthorities, record.EffectIntents = nil, nil, nil
	system.repository.records[system.goal.Goal.Ref()] = record
	system.repository.mu.Unlock()
	before := system.snapshot(t)
	_, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.ownerAccess, ProposeDirectorPlanRequest{
		RequestRef: "director-plan:legacy-governance", GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		LeaseToken: lease.Token, LeaseFence: lease.Fence, Reason: "legacy requires governed adoption",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "work:legacy-director", Objective: "must not persist",
			Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	if err == nil || err.Error() != "governance.legacy_reauthorization_required" {
		t.Fatalf("legacy Director error=%v", err)
	}
	after := system.snapshot(t)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("legacy Director wrote state: before=%+v after=%+v", before, after)
	}
}

func TestWorkItemAuthorityRejectsCrossedSourcePermissionBeforeScheduling(t *testing.T) {
	base := time.Date(2026, 7, 18, 21, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: base}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:crossed-authority", Statement: "reject crossed authority", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	authority := submitted.Record.WorkItemAuthorities[0]
	authority.Source = EffectApprovalSourceDirectorDecision
	_, actions, _, err := orchestrator.scheduleReady(
		context.Background(), submitted.Record.Goal, nil, []WorkItemAuthority{authority},
		orchestrator.budgetPolicy.effectPolicy(), base,
	)
	if err == nil || len(actions) != 0 {
		t.Fatalf("crossed authority scheduled: actions=%d err=%v", len(actions), err)
	}
}

func assertIntentPolicy(t *testing.T, intent EffectIntent, policy BudgetPolicy) {
	t.Helper()
	wantFingerprint := effectAdmissionFingerprint(intent.ActionRef, intent.Authority.Ref(), policy.PolicyHash)
	if intent.PolicyHash != policy.PolicyHash || intent.PolicyRevision != policy.GoalEnvelopeTemplate.Revision ||
		intent.QuotaRetryDelay != policy.QuotaRetryDelay || intent.ApprovalTTL != policy.EffectApprovalTTL ||
		intent.RequestFingerprint != wantFingerprint {
		t.Fatalf("intent policy=%s/%d/%s/%s want=%s/%d/%s/%s", intent.PolicyHash, intent.PolicyRevision,
			intent.QuotaRetryDelay, intent.ApprovalTTL, policy.PolicyHash, policy.GoalEnvelopeTemplate.Revision,
			policy.QuotaRetryDelay, policy.EffectApprovalTTL)
	}
}
