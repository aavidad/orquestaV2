package orquestamcp

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func mcpAutoprogrammingOperatorWithGoalFirstActiveRunsV0(
	operator *MCPAutoprogrammingOperatorV0,
	states []orquestagoal.GoalWorkStateV0,
	markers []orquestagoal.GoalWorkRunMarkerV0,
) *MCPAutoprogrammingOperatorV0 {
	if operator == nil {
		return operator
	}
	seen := map[string]bool{}
	for _, active := range operator.ActiveRuns {
		runRef := strings.TrimSpace(active.RunRef)
		if runRef != "" {
			seen[runRef] = true
		}
	}
	for _, state := range states {
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" || seen[runRef] || !mcpAutoprogrammingGoalStateVisibleForStatusV0(state) {
			continue
		}
		seen[runRef] = true
		operator.ActiveRuns = append(operator.ActiveRuns, MCPAutoprogrammingActiveRunV0{
			RunRef: runRef,
			AppRef: firstNonEmptyMCPV0(
				strings.TrimSpace(state.Spec.ProjectRef),
				strings.TrimSpace(state.Spec.DomainRef),
			),
			Status: strings.TrimSpace(state.Status),
		})
	}
	for _, marker := range markers {
		runRef := strings.TrimSpace(marker.RunRef)
		if runRef == "" || seen[runRef] {
			continue
		}
		seen[runRef] = true
		appRef := ""
		if marker.Spec != nil {
			appRef = firstNonEmptyMCPV0(
				strings.TrimSpace(marker.Spec.ProjectRef),
				strings.TrimSpace(marker.Spec.DomainRef),
			)
		}
		operator.ActiveRuns = append(operator.ActiveRuns, MCPAutoprogrammingActiveRunV0{
			RunRef: runRef,
			AppRef: appRef,
			Status: strings.TrimSpace(marker.Status),
		})
	}
	return operator
}
