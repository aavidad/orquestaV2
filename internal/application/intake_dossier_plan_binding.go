package application

import (
	"reflect"
	"slices"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

// intakeDossierPlanMatchesInitialGoal compares the complete request-local plan
// with its newly compiled Goal representation. Generated Goal/WorkItem refs are
// compared by stable item position and relationship, never trusted as input.
func intakeDossierPlanMatchesInitialGoal(
	spec PlanSpec,
	aggregate goal.Goal,
) bool {
	return intakeDossierPlanMatchesGoal(spec, aggregate, true)
}

// intakeDossierPlanMatchesLiveGoal preserves the immutable initial plan
// binding while allowing normal Goal/WorkItem progress and append-only replans.
func intakeDossierPlanMatchesLiveGoal(
	spec PlanSpec,
	aggregate goal.Goal,
) bool {
	return intakeDossierPlanMatchesGoal(spec, aggregate, false)
}

func intakeDossierPlanMatchesGoal(
	spec PlanSpec,
	aggregate goal.Goal,
	initial bool,
) bool {
	snapshot := aggregate.Snapshot()
	if snapshot.PlanGeneration < 1 ||
		len(snapshot.Phases) < len(spec.Phases) ||
		len(snapshot.WorkItems) < len(spec.WorkItems) ||
		(initial &&
			(snapshot.PlanGeneration != 1 ||
				len(snapshot.Phases) != len(spec.Phases) ||
				len(snapshot.WorkItems) != len(spec.WorkItems))) {
		return false
	}
	for index, expected := range spec.Phases {
		actual := snapshot.Phases[index]
		if actual.Ref != expected.Ref || actual.Key != expected.Key ||
			actual.TemplateRef != expected.TemplateRef ||
			!slices.Equal(actual.InputRefs, expected.InputRefs) ||
			!slices.Equal(actual.CriterionRefs, expected.CriterionRefs) {
			return false
		}
	}
	itemPositions := make(map[string]int, len(spec.WorkItems))
	for index, item := range spec.WorkItems {
		if _, duplicate := itemPositions[item.Key]; duplicate {
			return false
		}
		itemPositions[item.Key] = index
	}
	for index, expected := range spec.WorkItems {
		actual := snapshot.WorkItems[index]
		if actual.GoalRef != snapshot.Ref ||
			actual.ActorRef != snapshot.ActorRef ||
			actual.ProjectRef != snapshot.ProjectRef ||
			actual.Objective != expected.Objective ||
			actual.PhaseKey != expected.Phase ||
			actual.RoleKey != expected.Role ||
			actual.HandoffRequired == nil ||
			*actual.HandoffRequired != expected.HandoffRequired ||
			!slices.Equal(actual.WriteSet, expected.WriteSet) ||
			actual.CouncilPolicy != expected.CouncilPolicy ||
			actual.OutputContract != expected.OutputContract ||
			!workItemRequiredTestsMatch(
				actual.RequiredTests, expected.RequiredTests,
			) ||
			!slices.Equal(actual.SkillRefs, expected.SkillRefs) ||
			!slices.Equal(actual.ToolRefs, expected.ToolRefs) ||
			!slices.Equal(actual.CapabilityRefs, expected.CapabilityRefs) ||
			!workItemBudgetDemandMatches(
				actual.Ref, actual.BudgetDemand, expected.BudgetDemand,
			) ||
			actual.SecurityCriticality !=
				effectiveSecurityCriticality(expected.SecurityCriticality) ||
			actual.ReasoningEffort !=
				effectiveReasoningEffort(expected.ReasoningEffort) ||
			actual.CreatedAt != snapshot.CreatedAt ||
			(initial && !initialIntakeDossierWorkItemState(actual)) {
			return false
		}
		parentRef := ""
		if expected.Parent != "" {
			position, found := itemPositions[expected.Parent]
			if !found {
				return false
			}
			parentRef = snapshot.WorkItems[position].Ref
		}
		if actual.ParentRef != parentRef {
			return false
		}
		dependencies := make([]string, len(expected.Dependencies))
		for dependencyIndex, key := range expected.Dependencies {
			position, found := itemPositions[key]
			if !found {
				return false
			}
			dependencies[dependencyIndex] = snapshot.WorkItems[position].Ref
		}
		if !slices.Equal(actual.DependencyRefs, dependencies) {
			return false
		}
	}
	return true
}

func initialIntakeDossierWorkItemState(actual goal.WorkItemSnapshot) bool {
	return actual.State == goal.WorkItemStatePending &&
		actual.Revision == 1 &&
		actual.SkipReason == "" &&
		actual.InterruptCause == "" &&
		actual.ReworkOf == "" &&
		!actual.Paused &&
		!actual.CancelRequested &&
		actual.ControlSequence == 0 &&
		actual.StartedAt.IsZero() &&
		actual.InterruptedAt.IsZero() &&
		actual.FinishedAt.IsZero() &&
		actual.ExecutionRef == "" &&
		len(actual.ArtifactRefs) == 0 &&
		len(actual.AttestationRefs) == 0
}

func workItemRequiredTestsMatch(
	actual []goal.RequiredTestSpecSnapshot,
	expected []RequiredTestSpec,
) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index, want := range expected {
		got := actual[index]
		if got.Ref != want.Ref || got.ToolRef != want.ToolRef ||
			got.WorkingDirectory != want.WorkingDirectory ||
			!slices.Equal(got.Arguments, want.Arguments) {
			return false
		}
	}
	return true
}

func workItemBudgetDemandMatches(
	actualItemRef string,
	actual governance.BudgetDemand,
	expected governance.BudgetDemand,
) bool {
	if expected.Ref != "" {
		return reflect.DeepEqual(actual, expected)
	}
	if actual.Ref != "budget-demand:"+actualItemRef {
		return false
	}
	return expected.Resources == (governance.ResourceVector{}) ||
		actual.Resources == expected.Resources
}

func effectiveSecurityCriticality(
	value governance.SecurityCriticality,
) governance.SecurityCriticality {
	if value == "" {
		return governance.SecurityCriticalityNormal
	}
	return value
}

func effectiveReasoningEffort(
	value governance.ReasoningEffort,
) governance.ReasoningEffort {
	if value == "" {
		return governance.ReasoningEffortMedium
	}
	return value
}
