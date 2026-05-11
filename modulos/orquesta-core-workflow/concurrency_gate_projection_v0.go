package orquestacoreworkflow

import "strings"

const (
	concurrencyGateProjectionDecisionSeparatorV0 = "#decision:"
	concurrencyGateProjectionPlanSeparatorV0     = "#plan:"
)

type concurrencyGateProjectionPartsV0 struct {
	GateRef  string
	Decision ConcurrencyGateDecisionV0
	PlanRef  string
}

func concurrencyGateProjectionRefV0(payload ConcurrencyGateRecordedPayloadV0) string {
	normalized := normalizeConcurrencyGateRecordedPayloadV0(payload)
	return normalized.GateRef +
		concurrencyGateProjectionDecisionSeparatorV0 + string(normalized.Decision) +
		concurrencyGateProjectionPlanSeparatorV0 + normalized.PlanRef
}

func concurrencyGateProjectionPartsFromRefV0(ref string) (concurrencyGateProjectionPartsV0, bool) {
	gateRef, tail, ok := strings.Cut(strings.TrimSpace(ref), concurrencyGateProjectionDecisionSeparatorV0)
	if !ok {
		return concurrencyGateProjectionPartsV0{}, false
	}
	decision, planRef, ok := strings.Cut(tail, concurrencyGateProjectionPlanSeparatorV0)
	if !ok {
		return concurrencyGateProjectionPartsV0{}, false
	}
	parts := concurrencyGateProjectionPartsV0{
		GateRef:  strings.TrimSpace(gateRef),
		Decision: ConcurrencyGateDecisionV0(strings.TrimSpace(decision)),
		PlanRef:  strings.TrimSpace(planRef),
	}
	return parts, parts.GateRef != "" && parts.PlanRef != "" && isSupportedConcurrencyGateDecisionV0(parts.Decision)
}

func concurrencyGateProjectionForGateV0(current OrchestrationRunV0, gateRef string) (string, bool) {
	gateRef = strings.TrimSpace(gateRef)
	for _, existing := range current.ConcurrencyGates {
		parts, ok := concurrencyGateProjectionPartsFromRefV0(existing)
		if ok && parts.GateRef == gateRef {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func concurrencyGateMatchesPayloadV0(current OrchestrationRunV0, payload RecordConcurrencyGateCommandPayloadV0) (bool, bool) {
	existing, ok := concurrencyGateProjectionForGateV0(current, payload.GateRef)
	if !ok {
		return false, false
	}
	expected := concurrencyGateProjectionRefV0(concurrencyGateRecordedPayloadFromCommandV0(payload))
	return existing == expected, existing != expected
}

func concurrencyGateEventMatchesPayloadV0(current OrchestrationRunV0, payload ConcurrencyGateRecordedPayloadV0) (bool, bool) {
	existing, ok := concurrencyGateProjectionForGateV0(current, payload.GateRef)
	if !ok {
		return false, false
	}
	expected := concurrencyGateProjectionRefV0(payload)
	return existing == expected, existing != expected
}

func concurrencyGateRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.ConcurrencyGates {
		parts, ok := concurrencyGateProjectionPartsFromRefV0(projection)
		if !ok || seen[parts.GateRef] {
			return true
		}
		seen[parts.GateRef] = true
	}
	return false
}

func concurrencyGateProjectionFieldUnsafeV0(payload RecordConcurrencyGateCommandPayloadV0) bool {
	values := []string{payload.RunRef, payload.GateRef, payload.PlanRef, string(payload.Decision)}
	for _, value := range values {
		if concurrencyGateProjectionContainsSeparatorV0(value) {
			return true
		}
	}
	return false
}

func concurrencyGateProjectionContainsSeparatorV0(value string) bool {
	return strings.Contains(value, concurrencyGateProjectionDecisionSeparatorV0) ||
		strings.Contains(value, concurrencyGateProjectionPlanSeparatorV0)
}

func stringSetFromStringsV0(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[strings.TrimSpace(value)] = true
	}
	return set
}

func allStringsInSetV0(values []string, set map[string]bool) bool {
	for _, value := range values {
		if !set[strings.TrimSpace(value)] {
			return false
		}
	}
	return true
}

func anyStringInSetV0(values []string, set map[string]bool) bool {
	for _, value := range values {
		if set[strings.TrimSpace(value)] {
			return true
		}
	}
	return false
}
