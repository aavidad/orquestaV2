package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/governance"
)

func TestBudgetContractUsesOneCanonicalEnvelopeAcrossLayers(t *testing.T) {
	base := time.Date(2026, 7, 18, 14, 0, 0, 0, time.UTC)
	policy := testBudgetPolicy(base)
	if err := ValidateBudgetPolicy(policy); err != nil {
		t.Fatalf("valid hierarchy: %v", err)
	}
	for name, mutate := range map[string]func(*BudgetPolicy){
		"demand_exceeds_goal": func(value *BudgetPolicy) {
			value.GoalEnvelopeTemplate.Limit.Tokens = value.DefaultWorkItemDemand.Tokens - 1
		},
		"goal_exceeds_project": func(value *BudgetPolicy) {
			value.ProjectEnvelopeTemplate.Limit.Tokens = value.GoalEnvelopeTemplate.Limit.Tokens - 1
		},
		"project_exceeds_deployment": func(value *BudgetPolicy) {
			value.DeploymentEnvelope.Limit.Tokens = value.ProjectEnvelopeTemplate.Limit.Tokens - 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := policy
			mutate(&invalid)
			if ValidateBudgetPolicy(invalid) == nil {
				t.Fatal("invalid hierarchy accepted")
			}
		})
	}

	clock := &mutableClock{now: base}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:canonical-budget", Statement: "one budget contract", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item := before.Goal.WorkItems()[0]
	intent := before.EffectIntents[0]
	historical, policyErr := historicalEffectPolicy(before)
	if item.BudgetDemand() != intent.Demand || policyErr != nil || historical.PolicyHash != policy.PolicyHash {
		t.Fatalf("plan/intent/envelope diverged: item=%+v intent=%+v", item.BudgetDemand(), intent.Demand)
	}
	duplicate := before
	for _, envelope := range before.BudgetEnvelopes {
		if envelope.Scope == governance.BudgetScopeGoal {
			duplicate.BudgetEnvelopes = append(duplicate.BudgetEnvelopes, envelope)
			break
		}
	}
	if _, err := historicalEffectPolicy(duplicate); err == nil {
		t.Fatal("duplicate Goal envelope accepted as canonical")
	}
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:canonical-budget"); err != nil {
		t.Fatal(err)
	}
	after, err := repository.GetGoal(context.Background(), before.Goal.Ref())
	if err != nil || len(after.BudgetReservations) != 1 || len(agent.launchRequests) != 1 {
		t.Fatalf("budget facts missing: reservations=%d launches=%d err=%v",
			len(after.BudgetReservations), len(agent.launchRequests), err)
	}
	reservation := after.BudgetReservations[0]
	request := agent.launchRequests[0]
	if request.BudgetDemand != intent.Demand || reservation.DemandRef != intent.Demand.Ref ||
		reservation.Resources != intent.Demand.Resources {
		t.Fatalf("intent/reservation/port diverged: intent=%+v reservation=%+v request=%+v",
			intent.Demand, reservation, request.BudgetDemand)
	}
}

func TestHistoricalGoalBudgetPolicySurvivesRuntimeRotation(t *testing.T) {
	base := time.Date(2026, 7, 18, 16, 30, 0, 0, time.UTC)
	oldPolicy := budgetPolicyVariant(testBudgetPolicy(base), "old", 3, 7*time.Second, 7*time.Minute, 100)
	newPolicy := budgetPolicyVariant(testBudgetPolicy(base), "new", 4, time.Second, time.Minute, 1_000)
	clock := &mutableClock{now: base}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	orchestrator.budgetPolicy = oldPolicy
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:policy-rotation", Statement: "keep historical budget policy", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	record, _ := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	item, execution, intent := record.Goal.WorkItems()[0], record.Executions[0], record.EffectIntents[0]
	existing := governance.BudgetReservation{
		Ref: "budget-reservation:rotation-existing", DemandRef: "budget-demand:rotation-existing",
		ActionRef: "action:rotation-existing", EffectIntentRef: "effect-intent:rotation-existing",
		ProjectRef: project.String(), GoalRef: record.Goal.Ref().String(), WorkItemRef: item.Ref().String(),
		ExecutionRef: execution.Ref.String(), PlanGeneration: uint64(execution.PlanGeneration),
		AppSpecGeneration: uint64(execution.AppSpecGeneration), WorkItemGeneration: uint64(item.Revision()),
		Fence: 1, SpecHash: record.Goal.SpecHash(), PolicyHash: oldPolicy.PolicyHash,
		Resources: governance.ResourceVector{Tokens: 1}, ReservedAt: base,
	}
	if err := governance.ValidateBudgetReservation(existing); err != nil {
		t.Fatal(err)
	}
	repository.mu.Lock()
	stored := repository.records[record.Goal.Ref()]
	stored.BudgetReservations = append(stored.BudgetReservations, existing)
	repository.records[record.Goal.Ref()] = stored
	repository.mu.Unlock()
	orchestrator.budgetPolicy = newPolicy
	result, err := orchestrator.ProcessNext(context.Background(), "worker:rotated-quota")
	if err != nil || result.Processed || agent.launches != 0 {
		t.Fatalf("new policy bypassed historical limit: result=%+v launches=%d err=%v", result, agent.launches, err)
	}
	repository.mu.Lock()
	action := repository.actions["action:launch:"+execution.Ref.String()].record
	repository.mu.Unlock()
	if !action.AvailableAt.Equal(base.Add(oldPolicy.QuotaRetryDelay)) || intent.PolicyHash != oldPolicy.PolicyHash ||
		intent.PolicyRevision != oldPolicy.GoalEnvelopeTemplate.Revision || intent.QuotaRetryDelay != oldPolicy.QuotaRetryDelay ||
		intent.ApprovalTTL != oldPolicy.EffectApprovalTTL {
		t.Fatalf("historical policy lost: intent=%+v available=%s", intent, action.AvailableAt)
	}
	settled, err := governance.Reconcile(existing, governance.ResourceUsage{
		Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	})
	if err != nil {
		t.Fatal(err)
	}
	settled.SettledAt = base.Add(time.Second)
	repository.mu.Lock()
	stored = repository.records[record.Goal.Ref()]
	stored.BudgetSettlements = append(stored.BudgetSettlements, settled)
	repository.records[record.Goal.Ref()] = stored
	repository.mu.Unlock()
	clock.Advance(oldPolicy.QuotaRetryDelay)
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:rotated-launch"); err != nil {
		t.Fatal(err)
	}
	closed, err := repository.GetGoal(context.Background(), record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	last := closed.BudgetReservations[len(closed.BudgetReservations)-1]
	if agent.launches != 1 || len(closed.BudgetReservations) != 2 || last.PolicyHash != oldPolicy.PolicyHash {
		t.Fatalf("historical reservation lost: launches=%d reservations=%d last=%+v",
			agent.launches, len(closed.BudgetReservations), last)
	}
}

func TestRetryBudgetReleasesSettledUnknownProcessSlot(t *testing.T) {
	base := time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: base}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, &scriptedAgent{now: clock.Now})
	orchestrator.budgetPolicy.GoalEnvelopeTemplate.Limit.ProcessSlots = 1
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:retry-settled-slot", Statement: "reuse settled process slot", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	record := submitted.Record
	item, execution := record.Goal.WorkItems()[0], record.Executions[0]
	reservation := func(marker string, resources governance.ResourceVector, fence uint64) governance.BudgetReservation {
		return governance.BudgetReservation{
			Ref: "budget-reservation:" + marker, DemandRef: "budget-demand:" + marker,
			ActionRef: "action:" + marker, EffectIntentRef: "effect-intent:" + marker,
			ProjectRef: project.String(), GoalRef: record.Goal.Ref().String(), WorkItemRef: item.Ref().String(),
			ExecutionRef: execution.Ref.String(), PlanGeneration: uint64(execution.PlanGeneration),
			AppSpecGeneration: uint64(execution.AppSpecGeneration), WorkItemGeneration: uint64(item.Revision()),
			Fence: fence, SpecHash: record.Goal.SpecHash(), PolicyHash: orchestrator.budgetPolicy.PolicyHash,
			Resources: resources, ReservedAt: base,
		}
	}
	prior := reservation("prior-slot", governance.ResourceVector{ProcessSlots: 1}, 1)
	current := reservation("current-release", item.BudgetDemand().Resources, 2)
	for _, value := range []governance.BudgetReservation{prior, current} {
		if err := governance.ValidateBudgetReservation(value); err != nil {
			t.Fatal(err)
		}
	}
	priorSettlement, err := governance.Reconcile(prior, governance.ResourceUsage{Quality: governance.UsageQualityUnknown})
	if err != nil {
		t.Fatal(err)
	}
	zero := governance.ResourceVector{Currency: current.Resources.Currency}
	currentSettlement, err := governance.Reconcile(current, governance.ResourceUsage{
		Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	})
	if err != nil {
		t.Fatal(err)
	}
	record.BudgetReservations = append(record.BudgetReservations, prior, current)
	record.BudgetSettlements = append(record.BudgetSettlements, priorSettlement)
	fits, err := retryFitsIrreversibleGoalBudget(record, &currentSettlement, item.BudgetDemand())
	if err != nil || !fits {
		t.Fatalf("settled process slot remained cumulative: fits=%v err=%v", fits, err)
	}
}

func budgetPolicyVariant(
	policy BudgetPolicy,
	marker string,
	revision uint64,
	retry time.Duration,
	approvalTTL time.Duration,
	tokens int64,
) BudgetPolicy {
	policy.PolicyHash = effectAdmissionFingerprint("policy-rotation-" + marker)
	policy.QuotaRetryDelay = retry
	policy.EffectApprovalTTL = approvalTTL
	policy.DefaultWorkItemDemand.Tokens = 100
	for _, envelope := range []*governance.BudgetEnvelope{
		&policy.DeploymentEnvelope, &policy.ProjectEnvelopeTemplate, &policy.GoalEnvelopeTemplate,
	} {
		envelope.PolicyHash = policy.PolicyHash
		envelope.Revision = revision
		envelope.Limit.Tokens = tokens
	}
	return policy
}
