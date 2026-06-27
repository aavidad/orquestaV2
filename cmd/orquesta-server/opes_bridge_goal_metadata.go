package main

import "strings"

const (
	opesBridgeRoutePolicyGoalFirstV0           = "goal_first"
	opesBridgeDirectorExecutionModeGoalFirstV0 = "goal_first"
	opesBridgeNextActionObserveGoalV0          = "observe_goal"
)

func opesBridgeApplySubmitResultV0(
	result *opesDrainJobResultV0,
	submitResult opesExternalWorkRunSubmitResultV0,
) {
	if result == nil {
		return
	}
	result.RunRef = strings.TrimSpace(submitResult.RunRef)
	result.RoutePolicy = strings.TrimSpace(submitResult.RoutePolicy)
	result.DirectorExecutionMode = strings.TrimSpace(submitResult.DirectorExecutionMode)
	result.GoalRef = strings.TrimSpace(submitResult.GoalRef)
	result.ExternalGoalRef = strings.TrimSpace(submitResult.ExternalGoalRef)
	result.NextActions = compactStringsV0(submitResult.NextActions)
}

func opesBridgeApplyRunMetadataV0(
	result *opesDrainJobResultV0,
	metadata externalBridgeInputRunMetadataV0,
) {
	if result == nil {
		return
	}
	result.RoutePolicy = strings.TrimSpace(metadata.RoutePolicy)
	result.DirectorExecutionMode = strings.TrimSpace(metadata.DirectorExecutionMode)
	result.GoalRef = strings.TrimSpace(metadata.GoalRef)
	result.ExternalGoalRef = strings.TrimSpace(metadata.ExternalGoalRef)
	result.NextActions = compactStringsV0(metadata.NextActions)
}

func opesBridgeRunMetadataFromDrainResultV0(
	result opesDrainJobResultV0,
) externalBridgeInputRunMetadataV0 {
	return externalBridgeInputRunMetadataV0{
		RoutePolicy:           strings.TrimSpace(result.RoutePolicy),
		DirectorExecutionMode: strings.TrimSpace(result.DirectorExecutionMode),
		GoalRef:               strings.TrimSpace(result.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(result.ExternalGoalRef),
		NextActions:           compactStringsV0(result.NextActions),
	}
}

func opesBridgeResultUsesGoalFirstV0(
	result opesDrainJobResultV0,
) bool {
	if strings.TrimSpace(result.RoutePolicy) == opesBridgeRoutePolicyGoalFirstV0 ||
		strings.TrimSpace(result.DirectorExecutionMode) == opesBridgeDirectorExecutionModeGoalFirstV0 ||
		strings.TrimSpace(result.GoalRef) != "" ||
		strings.TrimSpace(result.ExternalGoalRef) != "" {
		return true
	}
	for _, action := range result.NextActions {
		if strings.TrimSpace(action) == opesBridgeNextActionObserveGoalV0 {
			return true
		}
	}
	return false
}

func opesBridgeGoalMetadataFromDirectorStatsV0(
	decoded opesBridgeDirectorStatsResponseV0,
) (externalBridgeInputRunMetadataV0, bool) {
	goal := decoded.Goal
	if strings.TrimSpace(goal.GoalRef) == "" &&
		strings.TrimSpace(goal.ExternalGoalRef) == "" &&
		strings.TrimSpace(goal.DirectorExecutionMode) != opesBridgeDirectorExecutionModeGoalFirstV0 {
		return externalBridgeInputRunMetadataV0{}, false
	}
	return externalBridgeInputRunMetadataV0{
		RoutePolicy:           opesBridgeRoutePolicyGoalFirstV0,
		DirectorExecutionMode: firstNonEmptyEnvlessV0(goal.DirectorExecutionMode, opesBridgeDirectorExecutionModeGoalFirstV0),
		GoalRef:               strings.TrimSpace(goal.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(goal.ExternalGoalRef),
		NextActions:           []string{opesBridgeNextActionObserveGoalV0},
	}, true
}
