package orquestacoreworkflow

import "strings"

const capacityDecisionProjectionSeparatorV0 = "#capacity_decision:"

func normalizeRegisterCapacityDecisionPayloadV0(payload RegisterCapacityDecisionCommandPayloadV0) RegisterCapacityDecisionCommandPayloadV0 {
	return RegisterCapacityDecisionCommandPayloadV0{
		CapacityRequestID: strings.TrimSpace(payload.CapacityRequestID),
		DecisionRef:       strings.TrimSpace(payload.DecisionRef),
		Tier:              OrchestrationCapacityRecommendationV0(strings.TrimSpace(string(payload.Tier))),
		ReasoningEffort:   OrchestrationCapacityRecommendationV0(strings.TrimSpace(string(payload.ReasoningEffort))),
		Summary:           strings.TrimSpace(payload.Summary),
		EvidenceRefs:      compactStringsV0(payload.EvidenceRefs),
	}
}

func normalizeCapacityDecidedPayloadV0(payload CapacityDecidedPayloadV0) CapacityDecidedPayloadV0 {
	normalized := normalizeRegisterCapacityDecisionPayloadV0(RegisterCapacityDecisionCommandPayloadV0(payload))
	return capacityDecidedPayloadFromCommandV0(normalized)
}

func capacityDecisionTextFieldsV0(payload RegisterCapacityDecisionCommandPayloadV0) []string {
	values := []string{
		payload.CapacityRequestID,
		payload.DecisionRef,
		string(payload.Tier),
		string(payload.ReasoningEffort),
		payload.Summary,
	}
	return append(values, payload.EvidenceRefs...)
}

func capacityDecisionProjectionRefV0(payload RegisterCapacityDecisionCommandPayloadV0) string {
	return strings.TrimSpace(payload.CapacityRequestID) + capacityDecisionProjectionSeparatorV0 + strings.TrimSpace(payload.DecisionRef)
}

func capacityDecisionProjectionRequestIDV0(ref string) string {
	requestID, _, ok := strings.Cut(strings.TrimSpace(ref), capacityDecisionProjectionSeparatorV0)
	if ok {
		return strings.TrimSpace(requestID)
	}
	return strings.TrimSpace(ref)
}

func capacityDecisionProjectionForRequestV0(current OrchestrationRunV0, requestID string) (string, bool) {
	requestID = strings.TrimSpace(requestID)
	for _, existing := range current.CapacityDecisions {
		if capacityDecisionProjectionRequestIDV0(existing) == requestID {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func capacityDecisionAlreadyReflectedV0(current OrchestrationRunV0, requestID string) bool {
	_, ok := capacityDecisionProjectionForRequestV0(current, requestID)
	return ok
}

func capacityDecisionMatchesPayloadV0(current OrchestrationRunV0, payload RegisterCapacityDecisionCommandPayloadV0) (bool, bool) {
	existing, ok := capacityDecisionProjectionForRequestV0(current, payload.CapacityRequestID)
	if !ok {
		return false, false
	}
	return existing == capacityDecisionProjectionRefV0(payload), existing != capacityDecisionProjectionRefV0(payload)
}
