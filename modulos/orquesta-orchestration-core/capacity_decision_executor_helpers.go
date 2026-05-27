package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func (executor CapacityDecisionExecutorV0) capacityDecisionCommandV0(
	ctx context.Context,
	intent orquestaoutboxdispatch.DispatchIntentV0,
	request orquestacoreworkflow.CapacityDecisionRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	decision, err := executor.capacityPolicyDecisionV0(ctx, intent, request)
	if err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{}, err
	}
	payload := orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
		CapacityRequestID: strings.TrimSpace(request.CapacityRequestID),
		DecisionRef:       decision.DecisionRef,
		Tier:              decision.Tier,
		ReasoningEffort:   decision.ReasoningEffort,
		Summary:           capacityDecisionSummaryV0(decision.Summary),
		EvidenceRefs:      executor.capacityDecisionEvidenceRefsFromPolicyV0(intent, request, decision),
	}
	return orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		executor.capacityDecisionCommandMetaV0(intent),
		payload,
	)
}

func (executor CapacityDecisionExecutorV0) capacityPolicyDecisionV0(
	ctx context.Context,
	intent orquestaoutboxdispatch.DispatchIntentV0,
	request orquestacoreworkflow.CapacityDecisionRequestV0,
) (CapacityDecisionPolicyDecisionV0, error) {
	policyRequest := CapacityDecisionPolicyRequestV0{
		Request:                request,
		DecisionRef:            capacityDecisionRefV0(intent.MessageID),
		DefaultTier:            executor.Tier,
		DefaultReasoningEffort: executor.ReasoningEffort,
		EvidenceRefs:           executor.capacityDecisionEvidenceRefsV0(intent, request),
	}
	if executor.Policy == nil {
		return executor.staticCapacityDecisionV0(policyRequest), nil
	}
	decision, err := executor.Policy.DecideCapacityV0(ctx, policyRequest)
	if err != nil {
		return CapacityDecisionPolicyDecisionV0{}, err
	}
	return decision.normalizeV0(policyRequest), nil
}

func (executor CapacityDecisionExecutorV0) staticCapacityDecisionV0(
	request CapacityDecisionPolicyRequestV0,
) CapacityDecisionPolicyDecisionV0 {
	return CapacityDecisionPolicyDecisionV0{
		DecisionRef:     request.DecisionRef,
		Tier:            capacityDecisionTierV0(executor.Tier, request.Request.MinimumRecommendedCapacity),
		ReasoningEffort: capacityDecisionTierV0(executor.ReasoningEffort, request.Request.MinimumRecommendedCapacity),
		Summary:         capacityDecisionSummaryV0(executor.Summary),
		EvidenceRefs:    request.EvidenceRefs,
		Motivos:         []string{"legacy-static-capacity-decision"},
	}.normalizeV0(request)
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

func (executor CapacityDecisionExecutorV0) capacityDecisionEvidenceRefsFromPolicyV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
	request orquestacoreworkflow.CapacityDecisionRequestV0,
	decision CapacityDecisionPolicyDecisionV0,
) []string {
	refs := executor.capacityDecisionEvidenceRefsV0(intent, request)
	refs = append(refs, decision.EvidenceRefs...)
	refs = appendCapacityDecisionTaggedRefV0(refs, "capacity_policy_ref", decision.PolicyRef)
	refs = appendCapacityDecisionTaggedRefV0(refs, "capacity_pool_ref", decision.PoolRef)
	refs = appendCapacityDecisionTaggedRefV0(refs, "capacity_model_ref", decision.ModelRef)
	refs = appendCapacityDecisionTaggedRefV0(refs, "capacity_quota_ref", decision.QuotaRef)
	for _, motivo := range decision.Motivos {
		refs = appendCapacityDecisionTaggedRefV0(refs, "capacity_motivo", motivo)
	}
	return compactStringsV0(refs)
}

func appendCapacityDecisionTaggedRefV0(refs []string, tag string, ref string) []string {
	if strings.TrimSpace(ref) == "" {
		return refs
	}
	return append(refs, strings.TrimSpace(tag)+":"+strings.TrimSpace(ref))
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
