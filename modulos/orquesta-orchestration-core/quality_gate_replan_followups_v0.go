package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func (provider QualityGateReplanCandidateProviderV0) qualityGateReplanFollowupCandidateV0(
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
