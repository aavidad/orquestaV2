package orquestacoreworkflow

import "strings"

const (
	agentLeaseExpiredProjectionAgentSeparatorV0    = "#agent:"
	agentLeaseExpiredProjectionActionSeparatorV0   = "#action:"
	agentLeaseExpiredProjectionReasonSeparatorV0   = "#reason:"
	agentLeaseExpiredProjectionObservedSeparatorV0 = "#observed:"
)

type agentLeaseExpiredProjectionPartsV0 struct {
	LeaseRef          string
	AgentRequestID    string
	RecommendedAction AgentLeaseRecommendedActionV0
	ReasonCode        string
	ObservedAt        string
}

func agentLeaseExpiredProjectionRefV0(payload AgentLeaseExpiredPayloadV0) string {
	normalized := normalizeAgentLeaseExpiredPayloadV0(payload)
	return normalized.LeaseRef +
		agentLeaseExpiredProjectionAgentSeparatorV0 + normalized.AgentRequestID +
		agentLeaseExpiredProjectionActionSeparatorV0 + string(normalized.RecommendedAction) +
		agentLeaseExpiredProjectionReasonSeparatorV0 + normalized.ReasonCode +
		agentLeaseExpiredProjectionObservedSeparatorV0 + normalized.ObservedAt
}

func agentLeaseExpiredProjectionPartsFromRefV0(ref string) (agentLeaseExpiredProjectionPartsV0, bool) {
	leaseRef, tail, ok := strings.Cut(strings.TrimSpace(ref), agentLeaseExpiredProjectionAgentSeparatorV0)
	if !ok {
		return agentLeaseExpiredProjectionPartsV0{}, false
	}
	agentRequestID, tail, ok := strings.Cut(tail, agentLeaseExpiredProjectionActionSeparatorV0)
	if !ok {
		return agentLeaseExpiredProjectionPartsV0{}, false
	}
	action, tail, ok := strings.Cut(tail, agentLeaseExpiredProjectionReasonSeparatorV0)
	if !ok {
		return agentLeaseExpiredProjectionPartsV0{}, false
	}
	reasonCode, observedAt, ok := strings.Cut(tail, agentLeaseExpiredProjectionObservedSeparatorV0)
	if !ok {
		return agentLeaseExpiredProjectionPartsV0{}, false
	}
	parts := agentLeaseExpiredProjectionPartsV0{
		LeaseRef:          strings.TrimSpace(leaseRef),
		AgentRequestID:    strings.TrimSpace(agentRequestID),
		RecommendedAction: AgentLeaseRecommendedActionV0(strings.TrimSpace(action)),
		ReasonCode:        strings.TrimSpace(reasonCode),
		ObservedAt:        strings.TrimSpace(observedAt),
	}
	return parts, parts.LeaseRef != "" && parts.AgentRequestID != "" &&
		isSupportedAgentLeaseRecommendedActionV0(parts.RecommendedAction) &&
		parts.ReasonCode != "" && validAgentLeaseObservedAtV0(parts.ObservedAt)
}

func agentLeaseExpiredProjectionForLeaseV0(current OrchestrationRunV0, leaseRef string) (string, bool) {
	leaseRef = strings.TrimSpace(leaseRef)
	for _, existing := range current.AgentLeaseExpirations {
		parts, ok := agentLeaseExpiredProjectionPartsFromRefV0(existing)
		if ok && parts.LeaseRef == leaseRef {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func agentLeaseExpiredMatchesPayloadV0(current OrchestrationRunV0, payload RegisterAgentLeaseExpiredCommandPayloadV0) (bool, bool) {
	existing, ok := agentLeaseExpiredProjectionForLeaseV0(current, payload.LeaseRef)
	if !ok {
		return false, false
	}
	expected := agentLeaseExpiredProjectionRefV0(agentLeaseExpiredPayloadFromCommandV0(payload))
	return existing == expected, existing != expected
}

func agentLeaseExpiredEventMatchesPayloadV0(current OrchestrationRunV0, payload AgentLeaseExpiredPayloadV0) (bool, bool) {
	existing, ok := agentLeaseExpiredProjectionForLeaseV0(current, payload.LeaseRef)
	if !ok {
		return false, false
	}
	expected := agentLeaseExpiredProjectionRefV0(payload)
	return existing == expected, existing != expected
}

func agentLeaseExpirationRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.AgentLeaseExpirations {
		parts, ok := agentLeaseExpiredProjectionPartsFromRefV0(projection)
		if !ok || seen[parts.LeaseRef] {
			return true
		}
		if !agentRequestAlreadyReflectedV0(run, parts.AgentRequestID) {
			return true
		}
		seen[parts.LeaseRef] = true
	}
	return false
}

func agentLeaseExpiredProjectionFieldUnsafeV0(payload RegisterAgentLeaseExpiredCommandPayloadV0) bool {
	values := []string{
		payload.RunRef,
		payload.AgentRequestID,
		payload.LeaseRef,
		payload.ReasonCode,
		payload.ObservedAt,
		string(payload.RecommendedAction),
	}
	for _, value := range values {
		if agentLeaseExpiredProjectionContainsSeparatorV0(value) {
			return true
		}
	}
	return false
}

func agentLeaseExpiredProjectionContainsSeparatorV0(value string) bool {
	return strings.Contains(value, agentLeaseExpiredProjectionAgentSeparatorV0) ||
		strings.Contains(value, agentLeaseExpiredProjectionActionSeparatorV0) ||
		strings.Contains(value, agentLeaseExpiredProjectionReasonSeparatorV0) ||
		strings.Contains(value, agentLeaseExpiredProjectionObservedSeparatorV0)
}
