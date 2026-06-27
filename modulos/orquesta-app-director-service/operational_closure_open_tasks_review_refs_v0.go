package orquestaappdirectorservice

import (
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func operationalDirectorPlanStateHasReasonOrBlockerV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	reason string,
) bool {
	reason = strings.TrimSpace(reason)
	return strings.TrimSpace(state.ClosureReason) == reason ||
		strings.TrimSpace(step.Reason) == reason ||
		startAppDirectorStringInSetV0(state.BlockerRefs, reason) ||
		startAppDirectorStringInSetV0(step.BlockerRefs, reason)
}

func operationalDirectorPlanStateSameRefsV0(left []string, right []string) bool {
	left = compactServiceRefsV0(left)
	right = compactServiceRefsV0(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
