package orquestacoreworkflow

import "strings"

const agentPhaseProjectionPhaseSeparatorV0 = "#phase:"

type agentPhaseProjectionPartsV0 struct {
	AgentRef string
	PhaseID  OrchestrationPhaseIDV0
}

func agentPhaseProjectionRefV0(payload AgentRequestedPayloadV0) string {
	normalized := normalizeAgentRequestedPayloadV0(payload)
	return normalized.AgentRequestID +
		agentPhaseProjectionPhaseSeparatorV0 + normalized.PhaseID
}

func agentPhaseProjectionPartsFromRefV0(ref string) (agentPhaseProjectionPartsV0, bool) {
	agentRef, phaseID, ok := strings.Cut(strings.TrimSpace(ref), agentPhaseProjectionPhaseSeparatorV0)
	if !ok {
		return agentPhaseProjectionPartsV0{}, false
	}
	parts := agentPhaseProjectionPartsV0{
		AgentRef: strings.TrimSpace(agentRef),
		PhaseID:  OrchestrationPhaseIDV0(strings.TrimSpace(phaseID)),
	}
	return parts, parts.AgentRef != "" && IsSupportedOrchestrationPhaseV0(parts.PhaseID)
}

func agentRequestedForPhaseAlreadyReflectedV0(
	current OrchestrationRunV0,
	agentRef string,
	phaseID OrchestrationPhaseIDV0,
) bool {
	agentRef = strings.TrimSpace(agentRef)
	phaseID = normalizePhaseIDV0(phaseID)
	for _, projection := range current.AgentPhaseRefs {
		parts, ok := agentPhaseProjectionPartsFromRefV0(projection)
		if ok && parts.AgentRef == agentRef && normalizePhaseIDV0(parts.PhaseID) == phaseID {
			return true
		}
	}
	return false
}

func agentPhaseRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.AgentPhaseRefs {
		parts, ok := agentPhaseProjectionPartsFromRefV0(projection)
		if !ok || seen[parts.AgentRef] {
			return true
		}
		if !agentRequestAlreadyReflectedV0(run, parts.AgentRef) ||
			!runContainsPhaseV0(run, parts.PhaseID) {
			return true
		}
		seen[parts.AgentRef] = true
	}
	return false
}
