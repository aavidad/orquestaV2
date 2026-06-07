package orquestadirectorsupervisor

import (
	"fmt"
	"strings"
)

func normalizeDirectorSupervisorBriefingInputV0(
	input DirectorSupervisorBriefingInputV0,
) DirectorSupervisorBriefingInputV0 {
	input.ObjectiveRef = strings.TrimSpace(input.ObjectiveRef)
	input.ContextRefs = compactSupervisorStringsV0(input.ContextRefs)
	input.Decision.RunRef = strings.TrimSpace(input.Decision.RunRef)
	input.Decision.ReasonCode = strings.TrimSpace(input.Decision.ReasonCode)
	input.Decision.PendingOutboxRefs = compactSupervisorStringsV0(input.Decision.PendingOutboxRefs)
	input.Decision.BlockedRefs = compactSupervisorStringsV0(input.Decision.BlockedRefs)
	input.Decision.EvidenceRefs = compactSupervisorStringsV0(input.Decision.EvidenceRefs)
	return input
}

func supervisorActionKindV0(
	action DirectorSupervisorActionV0,
) (string, int, bool, bool) {
	switch action {
	case DirectorSupervisorActionContinueV0:
		return DirectorSupervisorActionKindRunStepV0, 10, true, false
	case DirectorSupervisorActionWaitOutboxV0:
		return DirectorSupervisorActionKindDispatchOutboxV0, 20, true, false
	case DirectorSupervisorActionWaitExternalV0:
		return DirectorSupervisorActionKindWaitSignalV0, 30, false, false
	case DirectorSupervisorActionNeedsDirectorV0:
		return DirectorSupervisorActionKindAskDirectorV0, 40, false, true
	case DirectorSupervisorActionBlockedV0:
		return DirectorSupervisorActionKindReviewBlockerV0, 50, false, true
	case DirectorSupervisorActionStopQuiescentV0:
		return DirectorSupervisorActionKindCloseOrIdleV0, 60, true, false
	case DirectorSupervisorActionStopMaxStepsV0:
		return DirectorSupervisorActionKindStopBudgetV0, 70, false, true
	case DirectorSupervisorActionStopErrorV0:
		return DirectorSupervisorActionKindInspectErrorV0, 80, false, true
	default:
		return "", 0, false, false
	}
}

func supervisorActionRefV0(
	decision DirectorSupervisorDecisionV0,
	kind string,
) string {
	return fmt.Sprintf(
		"director-action:%s:%03d:%s",
		decision.RunRef,
		decision.StepNumber,
		kind,
	)
}

func supervisorActionTargetRefsV0(decision DirectorSupervisorDecisionV0) []string {
	switch decision.Action {
	case DirectorSupervisorActionWaitOutboxV0:
		return compactSupervisorStringsV0(decision.PendingOutboxRefs)
	case DirectorSupervisorActionWaitExternalV0, DirectorSupervisorActionNeedsDirectorV0:
		return compactSupervisorStringsV0(supervisorWaitingReasonTargetRefsV0(decision))
	case DirectorSupervisorActionBlockedV0:
		return compactSupervisorStringsV0(decision.BlockedRefs)
	default:
		return compactSupervisorStringsV0([]string{decision.RunRef})
	}
}

func supervisorWaitingReasonTargetRefsV0(decision DirectorSupervisorDecisionV0) []string {
	reasons := supervisorWaitingReasonStringsV0(decision.WaitingReasons)
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, "waiting_reason:"+reason)
	}
	return out
}
