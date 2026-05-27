package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

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
