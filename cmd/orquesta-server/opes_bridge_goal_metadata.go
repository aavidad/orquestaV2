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
	result.RoutePolicy = firstNonEmptyEnvlessV0(metadata.RoutePolicy, result.RoutePolicy)
	result.DirectorExecutionMode = firstNonEmptyEnvlessV0(metadata.DirectorExecutionMode, result.DirectorExecutionMode)
	result.GoalRef = firstNonEmptyEnvlessV0(metadata.GoalRef, result.GoalRef)
	result.ExternalGoalRef = firstNonEmptyEnvlessV0(metadata.ExternalGoalRef, result.ExternalGoalRef)
	result.NextActions = compactStringsV0(append(result.NextActions, metadata.NextActions...))
	result.CurrentPhase = firstNonEmptyEnvlessV0(metadata.CurrentPhase, result.CurrentPhase)
	result.OperationalReason = firstNonEmptyEnvlessV0(metadata.OperationalReason, result.OperationalReason)
	result.AudioCounters = copyStringIntMapV0(firstNonEmptyStringIntMapV0(metadata.DomainCounters, result.AudioCounters))
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
		CurrentPhase:          strings.TrimSpace(result.CurrentPhase),
		OperationalReason:     strings.TrimSpace(result.OperationalReason),
		DomainCounters:        copyStringIntMapV0(result.AudioCounters),
	}
}

func firstNonEmptyStringIntMapV0(values ...map[string]int) map[string]int {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
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
