package orquestacoreworkflow

import "strings"

const (
	qualityGateProjectionDecisionSeparatorV0 = "#decision:"
	qualityGateProjectionSubjectSeparatorV0  = "#subject:"
)

type qualityGateProjectionPartsV0 struct {
	GateRef    string
	Decision   QualityGateDecisionV0
	SubjectRef string
}

func qualityGateProjectionRefV0(payload QualityGateRecordedPayloadV0) string {
	normalized := normalizeQualityGateRecordedPayloadV0(payload)
	return normalized.GateRef +
		qualityGateProjectionDecisionSeparatorV0 + string(normalized.Decision) +
		qualityGateProjectionSubjectSeparatorV0 + normalized.SubjectRef
}

func qualityGateProjectionPartsFromRefV0(ref string) (qualityGateProjectionPartsV0, bool) {
	gateRef, tail, ok := strings.Cut(strings.TrimSpace(ref), qualityGateProjectionDecisionSeparatorV0)
	if !ok {
		return qualityGateProjectionPartsV0{}, false
	}
	decision, subjectRef, ok := strings.Cut(tail, qualityGateProjectionSubjectSeparatorV0)
	if !ok {
		return qualityGateProjectionPartsV0{}, false
	}
	parts := qualityGateProjectionPartsV0{
		GateRef:    strings.TrimSpace(gateRef),
		Decision:   QualityGateDecisionV0(strings.TrimSpace(decision)),
		SubjectRef: strings.TrimSpace(subjectRef),
	}
	return parts, parts.GateRef != "" &&
		parts.SubjectRef != "" &&
		isSupportedQualityGateDecisionV0(parts.Decision)
}

func qualityGateProjectionForGateV0(current OrchestrationRunV0, gateRef string) (string, bool) {
	gateRef = strings.TrimSpace(gateRef)
	for _, existing := range current.QualityGates {
		parts, ok := qualityGateProjectionPartsFromRefV0(existing)
		if ok && parts.GateRef == gateRef {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func qualityGateBlockedAlreadyReflectedV0(current OrchestrationRunV0, gateRef string) bool {
	projection, ok := qualityGateProjectionForGateV0(current, gateRef)
	if !ok {
		return false
	}
	parts, ok := qualityGateProjectionPartsFromRefV0(projection)
	return ok && parts.Decision == QualityGateDecisionBlockedV0
}

func qualityGateMatchesPayloadV0(current OrchestrationRunV0, payload RecordQualityGateCommandPayloadV0) (bool, bool) {
	existing, ok := qualityGateProjectionForGateV0(current, payload.GateRef)
	if !ok {
		return false, false
	}
	expected := qualityGateProjectionRefV0(qualityGateRecordedPayloadFromCommandV0(payload))
	return existing == expected, existing != expected
}

func qualityGateEventMatchesPayloadV0(current OrchestrationRunV0, payload QualityGateRecordedPayloadV0) (bool, bool) {
	existing, ok := qualityGateProjectionForGateV0(current, payload.GateRef)
	if !ok {
		return false, false
	}
	expected := qualityGateProjectionRefV0(payload)
	return existing == expected, existing != expected
}

func qualityGateRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.QualityGates {
		parts, ok := qualityGateProjectionPartsFromRefV0(projection)
		if !ok || seen[parts.GateRef] {
			return true
		}
		seen[parts.GateRef] = true
	}
	return false
}

func qualityGateProjectionFieldUnsafeV0(payload RecordQualityGateCommandPayloadV0) bool {
	values := []string{payload.RunRef, payload.GateRef, payload.PhaseID, payload.SubjectRef, string(payload.Decision)}
	for _, value := range values {
		if strings.Contains(value, qualityGateProjectionDecisionSeparatorV0) ||
			strings.Contains(value, qualityGateProjectionSubjectSeparatorV0) {
			return true
		}
	}
	return false
}
