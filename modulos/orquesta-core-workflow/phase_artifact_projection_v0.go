package orquestacoreworkflow

import "strings"

const (
	phaseArtifactProjectionPhaseSeparatorV0 = "#phase:"
	phaseArtifactProjectionAgentSeparatorV0 = "#agent:"
)

type phaseArtifactProjectionPartsV0 struct {
	ArtifactRef string
	PhaseID     OrchestrationPhaseIDV0
	AgentRef    string
}

func phaseArtifactProjectionRefV0(payload PhaseArtifactRegisteredPayloadV0) string {
	normalized := normalizePhaseArtifactRegisteredPayloadV0(payload)
	return normalized.ArtifactRef +
		phaseArtifactProjectionPhaseSeparatorV0 + normalized.PhaseID +
		phaseArtifactProjectionAgentSeparatorV0 + normalized.AgentRef
}

func phaseArtifactProjectionPartsFromRefV0(ref string) (phaseArtifactProjectionPartsV0, bool) {
	artifactRef, tail, ok := strings.Cut(strings.TrimSpace(ref), phaseArtifactProjectionPhaseSeparatorV0)
	if !ok {
		return phaseArtifactProjectionPartsV0{}, false
	}
	phaseID, agentRef, ok := strings.Cut(tail, phaseArtifactProjectionAgentSeparatorV0)
	if !ok {
		return phaseArtifactProjectionPartsV0{}, false
	}
	parts := phaseArtifactProjectionPartsV0{
		ArtifactRef: strings.TrimSpace(artifactRef),
		PhaseID:     OrchestrationPhaseIDV0(strings.TrimSpace(phaseID)),
		AgentRef:    strings.TrimSpace(agentRef),
	}
	return parts, parts.ArtifactRef != "" && parts.AgentRef != "" &&
		IsSupportedOrchestrationPhaseV0(parts.PhaseID) &&
		parts.PhaseID != OrchestrationPhaseProgramacionV0
}

func phaseArtifactProjectionForRefV0(current OrchestrationRunV0, artifactRef string) (string, bool) {
	artifactRef = strings.TrimSpace(artifactRef)
	for _, existing := range current.PhaseArtifacts {
		parts, ok := phaseArtifactProjectionPartsFromRefV0(existing)
		if ok && parts.ArtifactRef == artifactRef {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func phaseArtifactMatchesPayloadV0(current OrchestrationRunV0, payload RegisterPhaseArtifactCommandPayloadV0) (bool, bool) {
	existing, ok := phaseArtifactProjectionForRefV0(current, payload.ArtifactRef)
	if !ok {
		return false, false
	}
	expected := phaseArtifactProjectionRefV0(phaseArtifactRegisteredPayloadFromCommandV0(payload))
	return existing == expected, existing != expected
}

func phaseArtifactEventMatchesPayloadV0(current OrchestrationRunV0, payload PhaseArtifactRegisteredPayloadV0) (bool, bool) {
	existing, ok := phaseArtifactProjectionForRefV0(current, payload.ArtifactRef)
	if !ok {
		return false, false
	}
	expected := phaseArtifactProjectionRefV0(payload)
	return existing == expected, existing != expected
}

func phaseArtifactRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.PhaseArtifacts {
		parts, ok := phaseArtifactProjectionPartsFromRefV0(projection)
		if !ok || seen[parts.ArtifactRef] {
			return true
		}
		if !agentRequestAlreadyReflectedV0(run, parts.AgentRef) ||
			!agentStartedAlreadyReflectedV0(run, parts.AgentRef) {
			return true
		}
		seen[parts.ArtifactRef] = true
	}
	return false
}

func phaseArtifactProjectionFieldUnsafeV0(values ...string) bool {
	for _, value := range values {
		if strings.Contains(value, phaseArtifactProjectionPhaseSeparatorV0) ||
			strings.Contains(value, phaseArtifactProjectionAgentSeparatorV0) {
			return true
		}
	}
	return false
}
