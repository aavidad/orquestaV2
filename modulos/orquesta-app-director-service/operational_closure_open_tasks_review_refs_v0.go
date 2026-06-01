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

func operationalDirectorPlanStateRefsWithoutV0(values []string, ignored string) []string {
	ignored = strings.TrimSpace(ignored)
	out := make([]string, 0, len(values))
	for _, value := range compactServiceRefsV0(values) {
		if strings.TrimSpace(value) == ignored {
			continue
		}
		out = append(out, value)
	}
	return out
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
