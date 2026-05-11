package orquestacionnucleoapp

import (
	"strings"

	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const leaseActionProjectionAgentSeparatorV0 = "#agent:"

func normalizeAgentLeaseAssessmentV0(
	assessment orquestacoreleases.AgentTimeoutAssessmentV0,
) orquestacoreleases.AgentTimeoutAssessmentV0 {
	assessment.AssessmentRef = strings.TrimSpace(assessment.AssessmentRef)
	assessment.RunRef = strings.TrimSpace(assessment.RunRef)
	assessment.AgentRequestID = strings.TrimSpace(assessment.AgentRequestID)
	assessment.LeaseRef = strings.TrimSpace(assessment.LeaseRef)
	assessment.NowObservedAt = strings.TrimSpace(assessment.NowObservedAt)
	assessment.Decision = orquestacoreleases.AgentTimeoutDecisionV0(
		strings.TrimSpace(string(assessment.Decision)),
	)
	assessment.ReasonCode = strings.TrimSpace(assessment.ReasonCode)
	assessment.EvidenceRefs = compactStringsV0(assessment.EvidenceRefs)
	return assessment
}

func leaseRecommendedActionV0(
	action orquestacoreleases.AgentTimeoutDecisionV0,
) (orquestacoreworkflow.AgentLeaseRecommendedActionV0, bool) {
	switch action {
	case orquestacoreleases.AgentTimeoutDecisionRetryV0:
		return orquestacoreworkflow.AgentLeaseActionRetryV0, true
	case orquestacoreleases.AgentTimeoutDecisionAskDirectorV0:
		return orquestacoreworkflow.AgentLeaseActionAskDirectorV0, true
	case orquestacoreleases.AgentTimeoutDecisionStopAgentV0:
		return orquestacoreworkflow.AgentLeaseActionStopAgentV0, true
	case orquestacoreleases.AgentTimeoutDecisionMarkFailedV0:
		return orquestacoreworkflow.AgentLeaseActionMarkFailedV0, true
	case orquestacoreleases.AgentTimeoutDecisionMarkStoppedV0:
		return orquestacoreworkflow.AgentLeaseActionMarkStoppedV0, true
	case orquestacoreleases.AgentTimeoutDecisionReplanTaskV0:
		return orquestacoreworkflow.AgentLeaseActionReplanTaskV0, true
	case orquestacoreleases.AgentTimeoutDecisionAlertOnlyV0:
		return orquestacoreworkflow.AgentLeaseActionAlertOnlyV0, true
	default:
		return "", false
	}
}

func leaseActionAlreadyExpiredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	leaseRef string,
) bool {
	leaseRef = strings.TrimSpace(leaseRef)
	for _, projection := range run.AgentLeaseExpirations {
		projection = strings.TrimSpace(projection)
		if projection == leaseRef {
			return true
		}
		prefix, _, ok := strings.Cut(projection, leaseActionProjectionAgentSeparatorV0)
		if ok && strings.TrimSpace(prefix) == leaseRef {
			return true
		}
	}
	return false
}

func leaseActionQuestionRefV0(
	action orquestacoreworkflow.AgentLeaseRecommendedActionV0,
	leaseRef string,
) string {
	if action != orquestacoreworkflow.AgentLeaseActionAskDirectorV0 {
		return ""
	}
	return "question-ref-lease-" + leaseActionSafeRefPartV0(leaseRef)
}

func leaseActionCandidateRefV0(leaseRef string) string {
	return "lease-action-candidate-ref-" + leaseActionSafeRefPartV0(leaseRef)
}

func leaseActionRequestedByV0(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "orquesta-nucleo-leases"
}

func leaseActionSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
