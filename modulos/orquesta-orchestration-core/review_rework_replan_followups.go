package orquestacionnucleoapp

import (
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkOpenPhaseCandidateV0(
	request SchedulerCandidateRequestV0,
	proposal orquestacorereplanner.ReplanProposalV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) *orquestadirector.ReplanOpenPhaseCandidateV0 {
	if !reviewReworkActionReturnsToProgrammingV0(action) {
		return nil
	}
	return &orquestadirector.ReplanOpenPhaseCandidateV0{
		CommandMeta: provider.reviewReworkCommandMetaV0(request, "open-programacion", proposal.ReplanRef),
		Payload: orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Reabrir programacion tras retrabajo.",
		},
	}
}

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkCapacityCandidateV0(
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) *orquestadirector.ReplanCapacityCandidateV0 {
	if !reviewReworkActionRequiresCapacityV0(action) || plan.CapacityRequestRef == "" {
		return nil
	}
	return &orquestadirector.ReplanCapacityCandidateV0{
		CommandMeta: provider.reviewReworkCommandMetaV0(request, "capacity", plan.CapacityRequestRef),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          plan.CapacityRequestRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    proposal.TaskRef,
			ReasonCode:                 proposal.ReasonCode,
			Summary:                    reviewReworkSummaryV0(plan, "Capacidad para repetir tarea tras revision."),
			MinimumRecommendedCapacity: provider.reviewReworkCapacityV0(plan),
			EvidenceRefs:               compactStringsV0(proposal.EvidenceRefs),
		},
	}
}

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkAgentCandidateV0(
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) *orquestadirector.ReplanAgentCandidateV0 {
	if !reviewReworkActionRequiresAgentV0(action) || plan.AgentRequestID == "" {
		return nil
	}
	return &orquestadirector.ReplanAgentCandidateV0{
		CommandMeta: provider.reviewReworkCommandMetaV0(request, "agent", plan.AgentRequestID),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     plan.AgentRequestID,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            proposal.TaskRef,
			CapacityRequestRef: plan.CapacityRequestRef,
			Role:               reviewReworkRoleV0(plan),
			Summary:            reviewReworkSummaryV0(plan, "Agente para repetir tarea tras revision."),
			EvidenceRefs:       compactStringsV0(proposal.EvidenceRefs),
		},
	}
}

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkAskDirectorCandidateV0(
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) *orquestadirector.ReplanAskDirectorCandidateV0 {
	if !reviewReworkActionRequiresDirectorQuestionV0(action) || plan.AskDirectorQuestionID == "" {
		return nil
	}
	if action == orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 && len(plan.SplitTasks) > 0 {
		return nil
	}
	return &orquestadirector.ReplanAskDirectorCandidateV0{
		CommandMeta: provider.reviewReworkCommandMetaV0(request, "ask-director", plan.AskDirectorQuestionID),
		Payload: orquestacoreworkflow.AskDirectorCommandPayloadV0{
			QuestionID:   plan.AskDirectorQuestionID,
			SourceGroup:  "orquesta-nucleo-review-rework",
			Summary:      "Resolver retrabajo de revision.",
			Options:      []string{"repetir_tarea", "dividir_tarea", "cancelar_tarea"},
			EvidenceRefs: compactStringsV0(proposal.EvidenceRefs),
			Blocking:     true,
		},
	}
}

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkCommandMetaV0(
	request SchedulerCandidateRequestV0,
	kind string,
	ref string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	suffix := reviewReworkReplanSafeRefPartV0(kind + "-" + ref)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-review-rework-" + suffix,
		RunID:          request.Run.RunID,
		IdempotencyKey: "idem-review-rework-" + suffix,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    reviewReworkReplanRequestedByV0(provider.RequestedBy),
		OccurredAt:     strings.TrimSpace(request.OccurredAt),
	}
}

func (provider ReviewReworkReplanCandidateProviderV0) reviewReworkCapacityV0(
	plan ReviewReworkReplanPlanV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(plan.MinimumRecommendedCapacity)) != "" {
		return plan.MinimumRecommendedCapacity
	}
	if strings.TrimSpace(string(provider.DefaultCapacity)) != "" {
		return provider.DefaultCapacity
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}

func reviewReworkFollowupRefsV0(
	plan ReviewReworkReplanPlanV0,
	action orquestacoreworkflow.ReplanDecisionActionV0,
) []string {
	if action == orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 && len(plan.SplitTasks) > 0 {
		return reviewReworkSplitTaskRefsV0(plan.SplitTasks)
	}
	if reviewReworkActionRequiresDirectorQuestionV0(action) {
		return compactStringsV0([]string{plan.AskDirectorQuestionID})
	}
	return compactStringsV0([]string{plan.CapacityRequestRef, plan.AgentRequestID})
}

func reviewReworkCandidateRefV0(
	plan ReviewReworkReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
) string {
	if plan.CandidateRef != "" {
		return plan.CandidateRef
	}
	return "review-rework-replan-candidate-ref-" + reviewReworkReplanSafeRefPartV0(proposal.ReplanRef)
}

func reviewReworkRoleV0(plan ReviewReworkReplanPlanV0) string {
	if plan.AgentRole != "" {
		return plan.AgentRole
	}
	return "implementacion"
}

func reviewReworkSummaryV0(plan ReviewReworkReplanPlanV0, fallback string) string {
	if strings.TrimSpace(plan.Summary) != "" {
		return strings.TrimSpace(plan.Summary)
	}
	return fallback
}

func reviewReworkActionReturnsToProgrammingV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionSplitTaskV0
}

func reviewReworkActionRequiresCapacityV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionEscalateCapacityV0
}

func reviewReworkActionRequiresAgentV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0
}

func reviewReworkActionRequiresDirectorQuestionV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionAskDirectorV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionAbortTaskV0
}
