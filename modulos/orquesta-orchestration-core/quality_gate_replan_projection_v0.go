package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	qualityGateReplanGateDecisionSeparatorV0 = "#decision:"
	qualityGateReplanGateSubjectSeparatorV0  = "#subject:"

	qualityGateReplanDecisionSourceSeparatorV0    = "#source:"
	qualityGateReplanDecisionTaskSeparatorV0      = "#task:"
	qualityGateReplanDecisionActionSeparatorV0    = "#action:"
	qualityGateReplanDecisionFollowupsSeparatorV0 = "#followups:"
	qualityGateReplanDecisionFollowupsJoinerV0    = "+"
)

type qualityGateReplanGateProjectionPartsV0 struct {
	GateRef    string
	Decision   orquestacoreworkflow.QualityGateDecisionV0
	SubjectRef string
}

type qualityGateReplanDecisionProjectionPartsV0 struct {
	ReplanRef      string
	SourceRef      string
	TaskRef        string
	AcceptedAction orquestacoreworkflow.ReplanDecisionActionV0
	FollowupRefs   []string
}

func qualityGateReplanCausalDecisionsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []qualityGateReplanDecisionProjectionPartsV0 {
	blockedGates := qualityGateReplanPendingBlockedGateSetV0(run)
	if len(blockedGates) == 0 {
		return nil
	}
	result := []qualityGateReplanDecisionProjectionPartsV0{}
	for _, projection := range run.ReplanDecisions {
		parts, ok := qualityGateReplanDecisionProjectionPartsFromRefV0(projection)
		if ok && blockedGates[parts.SourceRef] {
			result = append(result, parts)
		}
	}
	return result
}

func qualityGateReplanPendingBlockedGateSetV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) map[string]bool {
	subjectOrder := make([]string, 0, len(run.QualityGates))
	lastBySubject := map[string]qualityGateReplanGateProjectionPartsV0{}
	for _, projection := range run.QualityGates {
		parts, ok := qualityGateReplanGateProjectionPartsFromRefV0(projection)
		if !ok {
			continue
		}
		if _, exists := lastBySubject[parts.SubjectRef]; !exists {
			subjectOrder = append(subjectOrder, parts.SubjectRef)
		}
		lastBySubject[parts.SubjectRef] = parts
	}
	result := map[string]bool{}
	for _, subjectRef := range subjectOrder {
		parts := lastBySubject[subjectRef]
		if parts.Decision == orquestacoreworkflow.QualityGateDecisionBlockedV0 {
			result[parts.GateRef] = true
		}
	}
	return result
}

func qualityGateReplanGateProjectionPartsFromRefV0(
	ref string,
) (qualityGateReplanGateProjectionPartsV0, bool) {
	gateRef, tail, ok := strings.Cut(strings.TrimSpace(ref), qualityGateReplanGateDecisionSeparatorV0)
	if !ok {
		return qualityGateReplanGateProjectionPartsV0{}, false
	}
	decision, subjectRef, ok := strings.Cut(tail, qualityGateReplanGateSubjectSeparatorV0)
	if !ok {
		return qualityGateReplanGateProjectionPartsV0{}, false
	}
	parts := qualityGateReplanGateProjectionPartsV0{
		GateRef:    strings.TrimSpace(gateRef),
		Decision:   orquestacoreworkflow.QualityGateDecisionV0(strings.TrimSpace(decision)),
		SubjectRef: strings.TrimSpace(subjectRef),
	}
	return parts, parts.GateRef != "" && parts.SubjectRef != "" &&
		(parts.Decision == orquestacoreworkflow.QualityGateDecisionAcceptedV0 ||
			parts.Decision == orquestacoreworkflow.QualityGateDecisionBlockedV0)
}

func qualityGateReplanDecisionProjectionPartsFromRefV0(
	ref string,
) (qualityGateReplanDecisionProjectionPartsV0, bool) {
	replanRef, tail, ok := strings.Cut(strings.TrimSpace(ref), qualityGateReplanDecisionSourceSeparatorV0)
	if !ok {
		return qualityGateReplanDecisionProjectionPartsV0{}, false
	}
	sourceRef, tail, ok := strings.Cut(tail, qualityGateReplanDecisionTaskSeparatorV0)
	if !ok {
		return qualityGateReplanDecisionProjectionPartsV0{}, false
	}
	taskRef, tail, ok := strings.Cut(tail, qualityGateReplanDecisionActionSeparatorV0)
	if !ok {
		return qualityGateReplanDecisionProjectionPartsV0{}, false
	}
	action, followups, ok := strings.Cut(tail, qualityGateReplanDecisionFollowupsSeparatorV0)
	if !ok {
		return qualityGateReplanDecisionProjectionPartsV0{}, false
	}
	parts := qualityGateReplanDecisionProjectionPartsV0{
		ReplanRef:      strings.TrimSpace(replanRef),
		SourceRef:      strings.TrimSpace(sourceRef),
		TaskRef:        strings.TrimSpace(taskRef),
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionV0(strings.TrimSpace(action)),
		FollowupRefs:   qualityGateReplanProjectionFollowupsV0(followups),
	}
	return parts, parts.ReplanRef != "" && parts.SourceRef != "" && parts.TaskRef != "" &&
		qualityGateReplanActionSupportedV0(parts.AcceptedAction) && len(parts.FollowupRefs) > 0
}

func qualityGateReplanProjectionFollowupsV0(joined string) []string {
	return compactStringsV0(strings.Split(strings.TrimSpace(joined), qualityGateReplanDecisionFollowupsJoinerV0))
}
