package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
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

type QualityGateReplanCandidateProviderV0 struct {
	Base            CandidateProviderPortV0
	RequestedBy     string
	DefaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
}

var _ CandidateProviderPortV0 = QualityGateReplanCandidateProviderV0{}

func (provider QualityGateReplanCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return candidates, nil
	}
	blockedGates := qualityGateReplanPendingBlockedGateSetV0(request.Run)
	if len(blockedGates) == 0 {
		return candidates, nil
	}
	for _, projection := range request.Run.ReplanDecisions {
		parts, ok := qualityGateReplanDecisionProjectionPartsFromRefV0(projection)
		if !ok || !blockedGates[parts.SourceRef] {
			continue
		}
		candidate, ok := provider.qualityGateReplanCandidateV0(request, parts)
		if !ok {
			continue
		}
		candidates.ReplanFollowupCandidates = append(candidates.ReplanFollowupCandidates, candidate)
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
	}
	return candidates, nil
}

func (provider QualityGateReplanCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func (provider QualityGateReplanCandidateProviderV0) qualityGateReplanCandidateV0(
	request SchedulerCandidateRequestV0,
	parts qualityGateReplanDecisionProjectionPartsV0,
) (orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0, bool) {
	capacityRef, agentRef := qualityGateReplanFollowupRefsV0(parts.FollowupRefs)
	if !qualityGateReplanHasEssentialFollowupsV0(parts.AcceptedAction, capacityRef, agentRef) {
		return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{}, false
	}
	input := orquestadirector.ReplanFollowupsInputV0{
		DecisionCommandMeta: provider.qualityGateReplanCommandMetaV0(request, "replan", parts.ReplanRef),
		DecisionPayload: orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
			ReplanRef:      parts.ReplanRef,
			RunRef:         request.Run.RunID,
			TaskRef:        parts.TaskRef,
			SourceRef:      parts.SourceRef,
			AcceptedAction: parts.AcceptedAction,
			FollowupRefs:   compactStringsV0(parts.FollowupRefs),
			Summary:        "Replan registrado para desbloquear quality gate.",
			EvidenceRefs:   compactStringsV0(request.EvidenceRefs),
		},
		SourceKind:        orquestadirector.ReplanFollowupSourceQualityGateBlockedV0,
		CapacityCandidate: provider.qualityGateReplanCapacityCandidateV0(request, parts, capacityRef),
		AgentCandidate:    provider.qualityGateReplanAgentCandidateV0(request, parts, capacityRef, agentRef),
	}
	return orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0{
		CandidateRef:         qualityGateReplanCandidateRefV0(parts),
		ReplanFollowupsInput: input,
		EvidenceRefs:         compactStringsV0(request.EvidenceRefs),
	}, true
}

func (provider QualityGateReplanCandidateProviderV0) qualityGateReplanCapacityCandidateV0(
	request SchedulerCandidateRequestV0,
	parts qualityGateReplanDecisionProjectionPartsV0,
	capacityRef string,
) *orquestadirector.ReplanCapacityCandidateV0 {
	if !qualityGateReplanActionRequiresCapacityV0(parts.AcceptedAction) || capacityRef == "" {
		return nil
	}
	return &orquestadirector.ReplanCapacityCandidateV0{
		CommandMeta: provider.qualityGateReplanCommandMetaV0(request, "capacity", capacityRef),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          capacityRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    parts.TaskRef,
			ReasonCode:                 "quality_gate_blocked",
			Summary:                    "Capacidad para desbloquear quality gate.",
			MinimumRecommendedCapacity: provider.qualityGateReplanCapacityV0(),
			EvidenceRefs:               compactStringsV0(request.EvidenceRefs),
		},
	}
}

func (provider QualityGateReplanCandidateProviderV0) qualityGateReplanAgentCandidateV0(
	request SchedulerCandidateRequestV0,
	parts qualityGateReplanDecisionProjectionPartsV0,
	capacityRef string,
	agentRef string,
) *orquestadirector.ReplanAgentCandidateV0 {
	if !qualityGateReplanActionRequiresAgentV0(parts.AcceptedAction) || capacityRef == "" || agentRef == "" {
		return nil
	}
	return &orquestadirector.ReplanAgentCandidateV0{
		CommandMeta: provider.qualityGateReplanCommandMetaV0(request, "agent", agentRef),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     agentRef,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            parts.TaskRef,
			CapacityRequestRef: capacityRef,
			Role:               "implementacion",
			Summary:            "Agente para desbloquear quality gate.",
			EvidenceRefs:       compactStringsV0(request.EvidenceRefs),
		},
	}
}

func (provider QualityGateReplanCandidateProviderV0) qualityGateReplanCommandMetaV0(
	request SchedulerCandidateRequestV0,
	kind string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	suffix := qualityGateReplanSafeRefPartV0(kind + "-" + ref)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-quality-gate-replan-" + suffix,
		RunID:          request.Run.RunID,
		IdempotencyKey: "idem-quality-gate-replan-" + suffix,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    qualityGateReplanRequestedByV0(provider.RequestedBy),
		OccurredAt:     strings.TrimSpace(request.OccurredAt),
	}
}

func (provider QualityGateReplanCandidateProviderV0) qualityGateReplanCapacityV0() orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(provider.DefaultCapacity)) != "" {
		return provider.DefaultCapacity
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}

type qualityGateReplanGateProjectionPartsV0 struct {
	GateRef    string
	Decision   orquestacoreworkflow.QualityGateDecisionV0
	SubjectRef string
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

type qualityGateReplanDecisionProjectionPartsV0 struct {
	ReplanRef      string
	SourceRef      string
	TaskRef        string
	AcceptedAction orquestacoreworkflow.ReplanDecisionActionV0
	FollowupRefs   []string
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

func qualityGateReplanFollowupRefsV0(followupRefs []string) (string, string) {
	var capacityRef string
	var agentRef string
	for _, ref := range compactStringsV0(followupRefs) {
		switch {
		case capacityRef == "" && qualityGateReplanLooksLikeCapacityRefV0(ref):
			capacityRef = ref
		case agentRef == "" && qualityGateReplanLooksLikeAgentRefV0(ref):
			agentRef = ref
		}
	}
	return capacityRef, agentRef
}

func qualityGateReplanLooksLikeCapacityRefV0(ref string) bool {
	return strings.HasPrefix(strings.TrimSpace(ref), "capacity-ref") ||
		strings.HasPrefix(strings.TrimSpace(ref), "capacity-request-ref")
}

func qualityGateReplanLooksLikeAgentRefV0(ref string) bool {
	return strings.HasPrefix(strings.TrimSpace(ref), "agent-ref") ||
		strings.HasPrefix(strings.TrimSpace(ref), "agent-request-ref")
}

func qualityGateReplanHasEssentialFollowupsV0(
	action orquestacoreworkflow.ReplanDecisionActionV0,
	capacityRef string,
	agentRef string,
) bool {
	if qualityGateReplanActionRequiresAgentV0(action) {
		return capacityRef != "" && agentRef != ""
	}
	if qualityGateReplanActionRequiresCapacityV0(action) {
		return capacityRef != ""
	}
	return false
}

func qualityGateReplanActionSupportedV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return qualityGateReplanActionRequiresCapacityV0(action) ||
		qualityGateReplanActionRequiresAgentV0(action)
}

func qualityGateReplanActionRequiresCapacityV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionEscalateCapacityV0
}

func qualityGateReplanActionRequiresAgentV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0
}

func qualityGateReplanCandidateRefV0(parts qualityGateReplanDecisionProjectionPartsV0) string {
	return "quality-gate-replan-candidate-ref-" + qualityGateReplanSafeRefPartV0(parts.ReplanRef)
}

func qualityGateReplanRequestedByV0(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "orquesta-nucleo-quality-gate-replan"
}

func qualityGateReplanSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
