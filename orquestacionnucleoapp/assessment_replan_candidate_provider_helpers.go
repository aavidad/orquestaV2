package orquestacionnucleoapp

import (
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeAgentAssessmentReplanPlanV0(plan AgentAssessmentReplanPlanV0) AgentAssessmentReplanPlanV0 {
	plan.CandidateRef = strings.TrimSpace(plan.CandidateRef)
	plan.ReplanRef = strings.TrimSpace(plan.ReplanRef)
	plan.SignalRef = strings.TrimSpace(plan.SignalRef)
	plan.TaskRef = strings.TrimSpace(plan.TaskRef)
	plan.ReasonRef = strings.TrimSpace(plan.ReasonRef)
	plan.RequestedAction = orquestacorereplanner.ReplanRecommendedActionV0(strings.TrimSpace(string(plan.RequestedAction)))
	plan.ReplacementRole = strings.TrimSpace(plan.ReplacementRole)
	plan.CapacityRequestRef = strings.TrimSpace(plan.CapacityRequestRef)
	plan.AgentRequestID = strings.TrimSpace(plan.AgentRequestID)
	plan.MinimumRecommendedCapacity = orquestacoreworkflow.OrchestrationCapacityRecommendationV0(strings.TrimSpace(string(plan.MinimumRecommendedCapacity)))
	plan.Summary = strings.TrimSpace(plan.Summary)
	plan.EvidenceRefs = compactStringsV0(plan.EvidenceRefs)
	plan.Assessment.AssessmentRef = strings.TrimSpace(plan.Assessment.AssessmentRef)
	plan.Assessment.PhaseID = strings.TrimSpace(plan.Assessment.PhaseID)
	plan.Assessment.AgentRequestID = strings.TrimSpace(plan.Assessment.AgentRequestID)
	plan.Assessment.TaskRef = strings.TrimSpace(plan.Assessment.TaskRef)
	plan.Assessment.DeliveryRef = strings.TrimSpace(plan.Assessment.DeliveryRef)
	plan.Assessment.Verdict = strings.TrimSpace(plan.Assessment.Verdict)
	plan.Assessment.Action = strings.TrimSpace(plan.Assessment.Action)
	plan.Assessment.Severity = strings.TrimSpace(plan.Assessment.Severity)
	plan.Assessment.Summary = strings.TrimSpace(plan.Assessment.Summary)
	plan.Assessment.EvidenceRefs = compactStringsV0(plan.Assessment.EvidenceRefs)
	return plan
}

func workflowReplanActionFromProposalV0(
	action orquestacorereplanner.ReplanRecommendedActionV0,
) (orquestacoreworkflow.ReplanDecisionActionV0, bool) {
	switch action {
	case orquestacorereplanner.ReplanActionRetryTaskV0,
		orquestacorereplanner.ReplanActionReplaceAgentV0,
		orquestacorereplanner.ReplanActionEscalateCapacityV0:
		return orquestacoreworkflow.ReplanDecisionActionV0(action), true
	default:
		return "", false
	}
}

func assessmentReplanActionRequiresCapacityV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionEscalateCapacityV0
}

func assessmentReplanActionRequiresAgentV0(action orquestacoreworkflow.ReplanDecisionActionV0) bool {
	return action == orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		action == orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0
}

func assessmentReplanFollowupRefsV0(plan AgentAssessmentReplanPlanV0) []string {
	return compactStringsV0([]string{plan.CapacityRequestRef, plan.AgentRequestID})
}

func (provider AgentAssessmentReplanCandidateProviderV0) assessmentReplanCapacityV0(
	plan AgentAssessmentReplanPlanV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(plan.MinimumRecommendedCapacity)) != "" {
		return plan.MinimumRecommendedCapacity
	}
	if strings.TrimSpace(string(provider.DefaultCapacity)) != "" {
		return provider.DefaultCapacity
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}

func assessmentReplanRoleV0(
	plan AgentAssessmentReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
) string {
	if proposal.ReplacementRole != "" {
		return proposal.ReplacementRole
	}
	if plan.ReplacementRole != "" {
		return plan.ReplacementRole
	}
	return "implementacion"
}

func assessmentReplanCandidateRefV0(
	plan AgentAssessmentReplanPlanV0,
	proposal orquestacorereplanner.ReplanProposalV0,
) string {
	if plan.CandidateRef != "" {
		return plan.CandidateRef
	}
	return "assessment-replan-candidate-ref-" + assessmentReplanSafeRefPartV0(proposal.ReplanRef)
}

func assessmentReplanRequestedByV0(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "orquesta-nucleo-replanner"
}

func assessmentReplanSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
