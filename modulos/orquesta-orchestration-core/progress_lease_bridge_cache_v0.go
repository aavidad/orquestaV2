package orquestacionnucleoapp

import (
	"strconv"
	"strings"
)

func (bridge *AgentProgressLeaseBridgeV0) storeProgressLeaseObservationsV0(
	key string,
	observations []AgentProgressObservationV0,
) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	bridge.cache = agentProgressLeaseBridgeCacheV0{
		Key:          key,
		Valid:        true,
		Observations: cloneProgressLeaseObservationsV0(observations),
	}
}

func (bridge *AgentProgressLeaseBridgeV0) cachedProgressLeaseObservationsV0(
	key string,
) ([]AgentProgressObservationV0, bool) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	if !bridge.cache.Valid || bridge.cache.Key != key {
		return nil, false
	}
	return cloneProgressLeaseObservationsV0(bridge.cache.Observations), true
}

func cloneProgressLeaseObservationsV0(
	observations []AgentProgressObservationV0,
) []AgentProgressObservationV0 {
	out := make([]AgentProgressObservationV0, len(observations))
	copy(out, observations)
	for i := range out {
		out[i].EvidenceRefs = compactStringsV0(out[i].EvidenceRefs)
		out[i].Report.EvidenceRefs = compactStringsV0(out[i].Report.EvidenceRefs)
	}
	return out
}

func progressLeaseBridgeKeyFromProgressV0(
	request AgentProgressObservationRequestV0,
) string {
	return strings.Join(compactStringsV0([]string{
		request.Run.RunID,
		strconv.Itoa(request.StepNumber),
		strings.TrimSpace(request.OccurredAt),
		strings.TrimSpace(request.CorrelationID),
	}), "|")
}

func progressLeaseBridgeKeyFromLeaseV0(
	request AgentLeaseAssessmentRequestV0,
) string {
	return strings.Join(compactStringsV0([]string{
		request.Run.RunID,
		strconv.Itoa(request.StepNumber),
		strings.TrimSpace(request.OccurredAt),
		strings.TrimSpace(request.CorrelationID),
	}), "|")
}
