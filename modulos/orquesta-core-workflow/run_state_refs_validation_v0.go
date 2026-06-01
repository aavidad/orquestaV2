package orquestacoreworkflow

import "strings"

func capacityDecisionRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, decision := range run.CapacityDecisions {
		requestID := capacityDecisionProjectionRequestIDV0(decision)
		if strings.TrimSpace(requestID) == "" || !capacityRequestAlreadyReflectedV0(run, requestID) {
			return true
		}
		if seen[requestID] {
			return true
		}
		seen[requestID] = true
	}
	return false
}

func directorAnsweredQuestionRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, questionID := range run.DirectorAnsweredQuestions {
		if !directorQuestionRefAlreadyReflectedV0(run, questionID) {
			return true
		}
	}
	return false
}

func reviewResultRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.ReviewResults {
		parts, ok := reviewResultProjectionPartsFromRefV0(projection)
		if !ok || !IsSupportedReviewResultStatusV0(parts.Status) {
			return true
		}
		if reviewResultProjectionInvalidV0(run, parts, seen) {
			return true
		}
		seen[parts.ReviewResultRef] = true
	}
	return false
}

func reviewResultProjectionInvalidV0(
	run OrchestrationRunV0,
	parts reviewResultProjectionPartsV0,
	seen map[string]bool,
) bool {
	return !reviewRequestAlreadyReflectedV0(run, parts.ReviewRequestID) ||
		!deliveryAlreadyReflectedV0(run, parts.DeliveryRef) ||
		seen[parts.ReviewResultRef]
}

func stoppedAgentRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, stopped := range run.StoppedAgents {
		if !agentRequestAlreadyReflectedV0(run, stopped) {
			return true
		}
	}
	return false
}

func lostAgentRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, lost := range run.LostAgents {
		if lostAgentRefInvalidV0(run, lost) {
			return true
		}
	}
	return false
}

func lostAgentRefInvalidV0(run OrchestrationRunV0, lost string) bool {
	return !agentRequestAlreadyReflectedV0(run, lost) ||
		!agentStartedAlreadyReflectedV0(run, lost) ||
		agentFailedAlreadyReflectedV0(run, lost) ||
		agentStopConfirmedAlreadyReflectedV0(run, lost) ||
		agentDeliveredAlreadyReflectedV0(run, lost)
}

func agentStopRequestRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projectionRef := range run.AgentStopRequests {
		projection, ok := ParseAgentStopRequestProjectionV0(projectionRef)
		if !ok || agentStopRequestProjectionInvalidV0(run, projection, seen) {
			return true
		}
		seen[projection.AgentRequestID] = true
	}
	return false
}

func agentStopRequestProjectionInvalidV0(
	run OrchestrationRunV0,
	projection AgentStopRequestProjectionV0,
	seen map[string]bool,
) bool {
	return !agentStopAlreadyReflectedV0(run, projection.AgentRequestID) ||
		seen[projection.AgentRequestID]
}

func confirmedStoppedAgentRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, stopped := range run.ConfirmedStoppedAgents {
		if !agentStopAlreadyReflectedV0(run, stopped) {
			return true
		}
	}
	return false
}

func refsContainForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		if stringHasForbiddenRunDetailV0(value) {
			return true
		}
	}
	return false
}

func stringHasForbiddenRunDetailV0(value string) bool {
	return textContainsForbiddenOperationalSensitiveDetailV0(value)
}
