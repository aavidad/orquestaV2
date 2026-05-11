package orquestadirectortickinput

import "strings"

const (
	capacityDecisionProjectionSeparatorV0 = "#capacity_decision:"
	concurrencyGateProjectionSeparatorV0  = "#decision:"
	agentLeaseProjectionSeparatorV0       = "#agent:"
	phaseArtifactProjectionSeparatorV0    = "#phase:"
	replanProjectionSeparatorV0           = "#source:"
	qualityGateDecisionSeparatorV0        = "#decision:"
	qualityGateSubjectSeparatorV0         = "#subject:"
	qualityGateDecisionAcceptedV0         = "accepted"
	qualityGateDecisionReworkRequiredV0   = "rework_required"
	qualityGateDecisionBlockedV0          = "blocked"
	qualityGateDecisionAskDirectorV0      = "ask_director"
)

func schedulerCapacityDecisionRefsV0(refs []string) []string {
	return projectionPrefixRefsV0(refs, capacityDecisionProjectionSeparatorV0)
}

func schedulerConcurrencyGateRefsV0(refs []string) []string {
	return projectionPrefixRefsV0(refs, concurrencyGateProjectionSeparatorV0)
}

func schedulerExpiredLeaseRefsV0(refs []string) []string {
	return projectionPrefixRefsV0(refs, agentLeaseProjectionSeparatorV0)
}

func schedulerPhaseArtifactRefsV0(refs []string) []string {
	return projectionPrefixRefsV0(refs, phaseArtifactProjectionSeparatorV0)
}

func schedulerReplanRefsV0(refs []string) []string {
	return projectionPrefixRefsV0(refs, replanProjectionSeparatorV0)
}

func schedulerBlockingQualityGateRefsV0(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	subjectOrder := make([]string, 0, len(refs))
	lastBySubject := map[string]qualityGateProjectionRefV0{}
	for _, ref := range refs {
		parts, ok := qualityGateProjectionRefFromCompactV0(ref)
		if !ok {
			continue
		}
		if _, exists := lastBySubject[parts.SubjectRef]; !exists {
			subjectOrder = append(subjectOrder, parts.SubjectRef)
		}
		lastBySubject[parts.SubjectRef] = parts
	}
	pendingRefs := make([]string, 0, len(lastBySubject))
	for _, subjectRef := range subjectOrder {
		parts := lastBySubject[subjectRef]
		if qualityGateDecisionIsPendingV0(parts.Decision) {
			pendingRefs = append(pendingRefs, parts.GateRef)
		}
	}
	return compactTickInputRefsV0(pendingRefs)
}

type qualityGateProjectionRefV0 struct {
	GateRef    string
	Decision   string
	SubjectRef string
}

func qualityGateProjectionRefFromCompactV0(ref string) (qualityGateProjectionRefV0, bool) {
	gateRef, tail, ok := strings.Cut(strings.TrimSpace(ref), qualityGateDecisionSeparatorV0)
	if !ok {
		return qualityGateProjectionRefV0{}, false
	}
	decision, subjectRef, ok := strings.Cut(tail, qualityGateSubjectSeparatorV0)
	if !ok {
		return qualityGateProjectionRefV0{}, false
	}
	parts := qualityGateProjectionRefV0{
		GateRef:    strings.TrimSpace(gateRef),
		Decision:   strings.TrimSpace(decision),
		SubjectRef: strings.TrimSpace(subjectRef),
	}
	return parts, parts.GateRef != "" && parts.Decision != "" && parts.SubjectRef != ""
}

func qualityGateDecisionIsPendingV0(decision string) bool {
	switch decision {
	case qualityGateDecisionReworkRequiredV0,
		qualityGateDecisionBlockedV0,
		qualityGateDecisionAskDirectorV0:
		return true
	default:
		return false
	}
}
