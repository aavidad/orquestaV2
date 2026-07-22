package goal_test

import (
	"reflect"
	"testing"

	domain "orquesta/internal/goal"
	"orquesta/internal/governance"
)

func TestWorkItemGovernanceMetadataRoundTripsAndRemainsImmutable(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:governance")
	currency, err := governance.NewCurrency("usd")
	if err != nil {
		t.Fatal(err)
	}
	demand := governance.BudgetDemand{
		Ref: "budget-demand:explicit",
		Resources: governance.ResourceVector{
			Tokens: 200000, MoneyMicros: 1000000, Currency: currency,
			ActiveTimeNS: 60_000_000_000, ProcessSlots: 1, DiskBytes: 1048576,
		},
	}
	item, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:governance", domain.NewWorkItemRef), Phase: phase.Key(),
		BudgetDemand: demand, SecurityCriticality: governance.SecurityCriticalityCritical,
		ReasoningEffort: governance.ReasoningEffortXHigh,
	})
	if err != nil {
		t.Fatalf("NewWorkItem(governance) error = %v", err)
	}
	plan := mustPlan(t, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{item},
	})
	planned, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := planned.Snapshot()
	if snapshot.SchemaVersion != domain.GoalSnapshotSchemaVersion ||
		snapshot.WorkItems[0].BudgetDemand != demand ||
		snapshot.WorkItems[0].SecurityCriticality != governance.SecurityCriticalityCritical ||
		snapshot.WorkItems[0].ReasoningEffort != governance.ReasoningEffortXHigh {
		t.Fatalf("governance snapshot = %+v", snapshot.WorkItems[0])
	}
	restored, err := domain.RestoreGoal(snapshot)
	if err != nil {
		t.Fatalf("RestoreGoal(governance) error = %v", err)
	}
	if got := restored.Snapshot(); !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("governance roundtrip differs:\n got: %#v\nwant: %#v", got, snapshot)
	}
	restoredItem, _ := restored.WorkItem(item.Ref())
	if restoredItem.BudgetDemand() != demand ||
		restoredItem.SecurityCriticality() != governance.SecurityCriticalityCritical ||
		restoredItem.ReasoningEffort() != governance.ReasoningEffortXHigh {
		t.Fatalf("restored governance = %+v/%q/%q", restoredItem.BudgetDemand(), restoredItem.SecurityCriticality(), restoredItem.ReasoningEffort())
	}
}

func TestSnapshotV5UpgradesOnlyMissingGovernanceMetadata(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:v5-governance")
	item := fixture.item(t, mustRef(t, "work-item:v5-governance", domain.NewWorkItemRef), phase.Key(), nil, nil)
	plan := mustPlan(t, domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{item},
	})
	planned, err := fixture.goal.ApplyPlan(fixture.goal.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	legacy := planned.Snapshot()
	legacy.SchemaVersion = domain.GoalSnapshotSchemaVersion - 2
	legacy.WorkItems[0].BudgetDemand = governance.BudgetDemand{}
	legacy.WorkItems[0].SecurityCriticality = ""
	legacy.WorkItems[0].ReasoningEffort = ""

	restored, err := domain.RestoreGoal(legacy)
	if err != nil {
		t.Fatalf("RestoreGoal(V5) error = %v", err)
	}
	upgraded := restored.Snapshot()
	if upgraded.SchemaVersion != domain.GoalSnapshotSchemaVersion ||
		upgraded.WorkItems[0].BudgetDemand.Ref != "budget-demand:"+item.Ref().String() ||
		upgraded.WorkItems[0].SecurityCriticality != governance.SecurityCriticalityNormal ||
		upgraded.WorkItems[0].ReasoningEffort != governance.ReasoningEffortMedium {
		t.Fatalf("V5 upgrade = %+v", upgraded.WorkItems[0])
	}

	for _, mutate := range []func(*domain.WorkItemSnapshot){
		func(value *domain.WorkItemSnapshot) { value.BudgetDemand = governance.BudgetDemand{} },
		func(value *domain.WorkItemSnapshot) { value.SecurityCriticality = "" },
		func(value *domain.WorkItemSnapshot) { value.ReasoningEffort = "" },
	} {
		invalid := planned.Snapshot()
		mutate(&invalid.WorkItems[0])
		if _, restoreErr := domain.RestoreGoal(invalid); domain.ErrorCodeOf(restoreErr) != domain.ErrorSnapshotInvalid {
			t.Fatalf("current snapshot without governance error = %v", restoreErr)
		}
	}
}

func TestPlanRejectsDuplicateBudgetDemandRefs(t *testing.T) {
	fixture := newPlanFixture(t)
	phase := mustPhase(t, "phase:demand-refs")
	demand := governance.BudgetDemand{Ref: "budget-demand:shared"}
	first, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:demand-a", domain.NewWorkItemRef), Phase: phase.Key(), BudgetDemand: demand,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fixture.newItem(domain.NewWorkItemInput{
		Ref: mustRef(t, "work-item:demand-b", domain.NewWorkItemRef), Phase: phase.Key(), BudgetDemand: demand,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = domain.NewPlan(domain.PlanInput{
		Generation: 1, Phases: []domain.PhaseInstance{phase}, WorkItems: []domain.WorkItem{first, second},
	})
	requireCode(t, err, domain.ErrorInvalidPlan)
}
