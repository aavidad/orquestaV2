package orquestacionnucleoapp

import (
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func (executor CapacityDecisionExecutorV0) capacityDecisionCommandV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	request orquestacoreworkflow.CapacityDecisionRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	payload := orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
		CapacityRequestID: strings.TrimSpace(request.CapacityRequestID),
		DecisionRef:       capacityDecisionRefV0(intent.MessageID),
		Tier:              capacityDecisionTierV0(executor.Tier, request.MinimumRecommendedCapacity),
		ReasoningEffort:   capacityDecisionTierV0(executor.ReasoningEffort, request.MinimumRecommendedCapacity),
		Summary:           capacityDecisionSummaryV0(executor.Summary),
		EvidenceRefs:      executor.capacityDecisionEvidenceRefsV0(intent, request),
	}
	return orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		executor.capacityDecisionCommandMetaV0(intent),
		payload,
	)
}

func (executor CapacityDecisionExecutorV0) capacityDecisionCommandMetaV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-" + capacityDecisionRefV0(intent.MessageID),
		RunID:          intent.RunID,
		IdempotencyKey: "idem-" + capacityDecisionRefV0(intent.MessageID),
		CorrelationID:  capacityDecisionCorrelationIDV0(executor.CorrelationID, intent),
		RequestedBy:    capacityDecisionRequestedByV0(executor.RequestedBy),
		OccurredAt:     strings.TrimSpace(executor.OccurredAt),
	}
}

func (executor CapacityDecisionExecutorV0) capacityDecisionEvidenceRefsV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	request orquestacoreworkflow.CapacityDecisionRequestV0,
) []string {
	refs := append([]string(nil), executor.EvidenceRefs...)
	refs = append(refs, request.EvidenceRefs...)
	refs = append(refs, "evidence-ref-"+capacityDecisionRefV0(intent.MessageID))
	return compactStringsV0(refs)
}

func capacityDecisionTierV0(
	configured orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
	requested orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(configured)) != "" {
		return configured
	}
	if strings.TrimSpace(string(requested)) != "" {
		return requested
	}
	return orquestacoreworkflow.OrchestrationCapacityMediumV0
}

func capacityDecisionSummaryV0(summary string) string {
	if strings.TrimSpace(summary) != "" {
		return strings.TrimSpace(summary)
	}
	return "Decision compacta de capacidad."
}

func capacityDecisionCorrelationIDV0(
	configured string,
	intent orquestaoutboxdispatch.DispatchIntentV0,
) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	return "corr-" + strings.TrimSpace(intent.MessageID)
}

func capacityDecisionRequestedByV0(requestedBy string) string {
	if strings.TrimSpace(requestedBy) != "" {
		return strings.TrimSpace(requestedBy)
	}
	return "orquesta-capacity-adapter"
}

func capacityDecisionDispatchRefV0(messageID string) string {
	return "dispatch-ref-" + capacityDecisionRefV0(messageID)
}

func capacityDecisionRefV0(messageID string) string {
	return fmt.Sprintf("capacity-decision-ref-%s", strings.TrimSpace(messageID))
}
