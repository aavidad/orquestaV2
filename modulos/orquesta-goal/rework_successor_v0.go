package orquestagoal

import (
	"strconv"
	"strings"
)

// ValidateGoalWorkReworkSuccessorV0 permits exactly one immutable-spec
// exception: a fresh next -rework-N attempt that carries required goal and
// closure context references to a parent whose closure requested rework.
// expectedParentClosureRef is optional for stores that only know the generic
// contract; callers that own the closure namespace pass its exact value.
func ValidateGoalWorkReworkSuccessorV0(parent, successor GoalWorkStateV0, expectedParentClosureRef string) []GoalWorkIssueV0 {
	parent = NormalizeGoalWorkStateV0(parent)
	successor = NormalizeGoalWorkStateV0(successor)
	var issues []GoalWorkIssueV0
	if _, err := NewGoalWorkStateV0(parent); err != nil {
		return []GoalWorkIssueV0{{Code: ErrGoalClosureInvalidV0, Field: "rework_successor.parent_state"}}
	}
	if parent.LastClosure == nil || !parent.LastClosure.NeedsRework {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "rework_successor.parent_closure"})
	}
	if successor.RunRef != parent.RunRef || successor.Spec.RunRef != parent.RunRef {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "rework_successor.run_ref"})
	}
	if successor.GoalRef != successor.Spec.GoalRef || successor.GoalRef != goalWorkNextReworkGoalRefV0(parent.GoalRef) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "rework_successor.goal_ref"})
	}
	if !goalWorkHasRequiredContextRefV0(successor.Spec.ContextRefs, "goal", parent.GoalRef) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "rework_successor.context_refs.goal"})
	}
	if !goalWorkHasRequiredClosureContextRefV0(successor.Spec.ContextRefs, expectedParentClosureRef) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "rework_successor.context_refs.closure"})
	}
	if successor.LastResult != nil || successor.LastClosure != nil {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalClosureInvalidV0, Field: "rework_successor.state"})
	}
	if successor.Spec.GoalRef != "" {
		issues = append(issues, ValidateGoalWorkSpecV0(successor.Spec)...)
	}
	return issues
}

func goalWorkNextReworkGoalRefV0(parentGoalRef string) string {
	base, index := goalWorkReworkGoalRefPartsV0(parentGoalRef)
	if base == "" {
		return ""
	}
	return base + "-rework-" + strconv.Itoa(index+1)
}

func goalWorkReworkGoalRefPartsV0(goalRef string) (string, int) {
	goalRef = strings.TrimSpace(goalRef)
	marker := strings.LastIndex(goalRef, "-rework-")
	if marker <= 0 {
		return goalRef, 0
	}
	index, err := strconv.Atoi(goalRef[marker+len("-rework-"):])
	if err != nil || index < 1 {
		return goalRef, 0
	}
	return goalRef[:marker], index
}

func goalWorkHasRequiredContextRefV0(contextRefs []GoalContextRefV0, kind, ref string) bool {
	for _, contextRef := range contextRefs {
		if contextRef.Required && strings.TrimSpace(contextRef.Kind) == kind && strings.TrimSpace(contextRef.Ref) == ref {
			return true
		}
	}
	return false
}

func goalWorkHasRequiredClosureContextRefV0(contextRefs []GoalContextRefV0, expectedRef string) bool {
	expectedRef = strings.TrimSpace(expectedRef)
	for _, contextRef := range contextRefs {
		if !contextRef.Required || strings.TrimSpace(contextRef.Kind) != "closure" {
			continue
		}
		ref := strings.TrimSpace(contextRef.Ref)
		if ref != "" && (expectedRef == "" || ref == expectedRef) {
			return true
		}
	}
	return false
}
