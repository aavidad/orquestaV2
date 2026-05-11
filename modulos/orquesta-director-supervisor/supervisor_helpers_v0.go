package orquestadirectorsupervisor

import (
	"strings"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func normalizeDirectorSupervisorInputV0(
	input DirectorSupervisorDecisionInputV0,
) DirectorSupervisorDecisionInputV0 {
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.LastErrorCode = strings.TrimSpace(input.LastErrorCode)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.EvidenceRefs = compactSupervisorStringsV0(input.EvidenceRefs)
	return input
}

func newDirectorSupervisorDecisionV0(
	input DirectorSupervisorDecisionInputV0,
) DirectorSupervisorDecisionV0 {
	last := input.LastStepResult
	return DirectorSupervisorDecisionV0{
		RunRef:            input.RunRef,
		StepNumber:        input.StepNumber,
		MaxSteps:          input.MaxSteps,
		PendingOutboxRefs: supervisorPendingOutboxRefsV0(last),
		WaitingReasons:    append(last.WaitingReasons[:0:0], last.WaitingReasons...),
		BlockedRefs:       compactSupervisorStringsV0(last.BlockedRefs),
		EvidenceRefs:      compactSupervisorStringsV0(input.EvidenceRefs),
	}
}

func supervisorDecisionWithActionV0(
	decision DirectorSupervisorDecisionV0,
	action DirectorSupervisorActionV0,
	reason string,
	shouldContinue bool,
) DirectorSupervisorDecisionV0 {
	decision.Action = action
	decision.ReasonCode = reason
	decision.ShouldContinue = shouldContinue
	decision.AutonomousRecommendation = supervisorAutonomousRecommendationForActionV0(action)
	return decision
}

func supervisorWaitOutboxDecisionV0(
	decision DirectorSupervisorDecisionV0,
) DirectorSupervisorDecisionV0 {
	return supervisorDecisionWithActionV0(
		decision,
		DirectorSupervisorActionWaitOutboxV0,
		DirectorSupervisorReasonOutboxPendingV0,
		false,
	)
}

func supervisorHasWaitingReasonV0(
	input DirectorSupervisorDecisionInputV0,
	reason orquestadirectorscheduler.SchedulerWaitingReasonV0,
) bool {
	for _, waitingReason := range input.LastStepResult.WaitingReasons {
		if waitingReason == reason {
			return true
		}
	}
	return false
}

func supervisorAutonomousRecommendationForActionV0(
	action DirectorSupervisorActionV0,
) DirectorSupervisorAutonomousRecommendationV0 {
	switch action {
	case DirectorSupervisorActionContinueV0:
		return DirectorSupervisorAutonomousContinueV0
	case DirectorSupervisorActionWaitOutboxV0, DirectorSupervisorActionWaitExternalV0:
		return DirectorSupervisorAutonomousWaitV0
	case DirectorSupervisorActionNeedsDirectorV0:
		return DirectorSupervisorAutonomousNeedsDirectorV0
	default:
		return DirectorSupervisorAutonomousStopV0
	}
}

func isKnownSupervisorAutonomousRecommendationV0(
	recommendation DirectorSupervisorAutonomousRecommendationV0,
) bool {
	switch recommendation {
	case DirectorSupervisorAutonomousContinueV0,
		DirectorSupervisorAutonomousWaitV0,
		DirectorSupervisorAutonomousNeedsDirectorV0,
		DirectorSupervisorAutonomousStopV0:
		return true
	default:
		return false
	}
}

func supervisorPendingOutboxRefsV0(
	result orquestadirectorcycle.DirectorCycleStepResultV0,
) []string {
	values := append(result.PendingOutboxBeforeRefs[:0:0], result.PendingOutboxBeforeRefs...)
	values = append(values, result.PendingOutboxAfterRefs...)
	return compactSupervisorStringsV0(values)
}

func compactSupervisorStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
