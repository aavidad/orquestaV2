package orquestacionnucleoapp

import (
	"strings"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func normalizeDirectorBriefingExecutionRequestV0(
	request DirectorBriefingExecutionRequestV0,
) DirectorBriefingExecutionRequestV0 {
	request.ActionRef = strings.TrimSpace(request.ActionRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	request.WaitAgentRefs = compactStringsV0(request.WaitAgentRefs)
	request.Briefing.SchemaVersion = strings.TrimSpace(request.Briefing.SchemaVersion)
	request.Briefing.RunRef = strings.TrimSpace(request.Briefing.RunRef)
	request.Briefing.EvidenceRefs = compactStringsV0(request.Briefing.EvidenceRefs)
	request.Briefing.PendingOutboxRefs = compactStringsV0(request.Briefing.PendingOutboxRefs)
	request.Briefing.WaitingReasons = compactStringsV0(request.Briefing.WaitingReasons)
	request.Briefing.BlockedRefs = compactStringsV0(request.Briefing.BlockedRefs)
	request.Briefing.ContextRefs = compactStringsV0(request.Briefing.ContextRefs)
	if request.Briefing.NextAction != nil {
		next := normalizeDirectorBriefingActionV0(*request.Briefing.NextAction, request.Briefing.RunRef)
		request.Briefing.NextAction = &next
	}
	for i := range request.Briefing.ActionQueue {
		request.Briefing.ActionQueue[i] = normalizeDirectorBriefingActionV0(
			request.Briefing.ActionQueue[i],
			request.Briefing.RunRef,
		)
	}
	return request
}

func selectDirectorBriefingExecutionActionV0(
	request DirectorBriefingExecutionRequestV0,
) (orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0, error) {
	if request.Briefing.SchemaVersion != orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0 {
		return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{},
			errorV0(ErrDirectorBriefingExecutionInvalidV0, "briefing.schema_version", "schema_version no soportado")
	}
	if request.Briefing.RunRef == "" {
		return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{},
			errorV0(ErrDirectorBriefingExecutionInvalidV0, "briefing.run_ref", "run_ref requerido")
	}
	action, ok := findDirectorBriefingExecutionActionV0(request)
	if !ok {
		return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{},
			errorV0(ErrDirectorBriefingExecutionInvalidV0, "action_ref", "accion no encontrada")
	}
	if action.Kind == "" {
		return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{},
			errorV0(ErrDirectorBriefingExecutionInvalidV0, "action.kind", "kind requerido")
	}
	if action.RunRef != "" && action.RunRef != request.Briefing.RunRef {
		return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{},
			errorV0(ErrDirectorBriefingExecutionInvalidV0, "action.run_ref", "accion de otro run")
	}
	action.RunRef = request.Briefing.RunRef
	return action, nil
}

func findDirectorBriefingExecutionActionV0(
	request DirectorBriefingExecutionRequestV0,
) (orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0, bool) {
	if request.ActionRef != "" {
		for _, action := range request.Briefing.ActionQueue {
			if action.ActionRef == request.ActionRef {
				return action, true
			}
		}
		if request.Briefing.NextAction != nil && request.Briefing.NextAction.ActionRef == request.ActionRef {
			return *request.Briefing.NextAction, true
		}
		return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{}, false
	}
	if request.Briefing.NextAction != nil {
		return *request.Briefing.NextAction, true
	}
	if len(request.Briefing.ActionQueue) > 0 {
		return request.Briefing.ActionQueue[0], true
	}
	return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{}, false
}

func normalizeDirectorBriefingActionV0(
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
	defaultRunRef string,
) orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0 {
	action.ActionRef = strings.TrimSpace(action.ActionRef)
	action.Kind = strings.TrimSpace(action.Kind)
	action.RunRef = strings.TrimSpace(action.RunRef)
	if action.RunRef == "" {
		action.RunRef = defaultRunRef
	}
	action.ReasonCode = strings.TrimSpace(action.ReasonCode)
	action.TargetRefs = compactStringsV0(action.TargetRefs)
	action.EvidenceRefs = compactStringsV0(action.EvidenceRefs)
	return action
}

func mergeBriefingExecutionEvidenceV0(
	request DirectorBriefingExecutionRequestV0,
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
) []string {
	refs := append([]string(nil), request.Briefing.EvidenceRefs...)
	refs = append(refs, request.EvidenceRefs...)
	refs = append(refs, action.EvidenceRefs...)
	return compactStringsV0(refs)
}

func mergeBriefingDispatchEvidenceV0(
	request DirectorBriefingExecutionRequestV0,
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
	dispatched progressiveDispatchWaitResultV0,
) []string {
	refs := mergeBriefingExecutionEvidenceV0(request, action)
	for _, once := range dispatched.Once {
		refs = append(refs, once.EvidenceRefs...)
	}
	return compactStringsV0(refs)
}
