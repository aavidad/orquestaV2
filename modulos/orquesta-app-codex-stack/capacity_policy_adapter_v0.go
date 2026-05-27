package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type StackCapacityPolicyAdapterV0 struct {
	Policy orquestacapacity.DefaultCapacityPolicyV0
}

var _ orquestacionnucleoapp.CapacityDecisionPolicyPortV0 = StackCapacityPolicyAdapterV0{}

func (adapter StackCapacityPolicyAdapterV0) DecideCapacityV0(
	ctx context.Context,
	request orquestacionnucleoapp.CapacityDecisionPolicyRequestV0,
) (orquestacionnucleoapp.CapacityDecisionPolicyDecisionV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.CapacityDecisionPolicyDecisionV0{}, err
	}
	decision, err := adapter.Policy.DecideCapacityV0(capacityPolicyRequestFromCoreV0(request))
	if err != nil {
		return orquestacionnucleoapp.CapacityDecisionPolicyDecisionV0{}, err
	}
	return capacityPolicyDecisionToCoreV0(request.DecisionRef, decision), nil
}

func capacityDecisionPolicyV0(
	config CapacityConfigV0,
) orquestacionnucleoapp.CapacityDecisionPolicyPortV0 {
	if config.DecisionPolicy != nil {
		return config.DecisionPolicy
	}
	return StackCapacityPolicyAdapterV0{Policy: orquestacapacity.DefaultCapacityPolicyV0{
		PolicyRef:              config.PolicyRef,
		PoolRef:                config.PoolRef,
		ModelRef:               config.ModelRef,
		QuotaRef:               config.QuotaRef,
		DefaultTier:            string(config.Tier),
		DefaultReasoningEffort: string(config.ReasoningEffort),
		Summary:                config.Summary,
		EvidenceRefs:           append([]string(nil), config.EvidenceRefs...),
	}}
}

func capacityPolicyRequestFromCoreV0(
	request orquestacionnucleoapp.CapacityDecisionPolicyRequestV0,
) orquestacapacity.CapacityPolicyRequestV0 {
	coreRequest := request.Request
	return orquestacapacity.CapacityPolicyRequestV0{
		CapacityRequestID:          coreRequest.CapacityRequestID,
		DecisionRef:                request.DecisionRef,
		RunRef:                     coreRequest.RunID,
		PhaseRef:                   coreRequest.PhaseID,
		TaskRef:                    coreRequest.TaskRef,
		ReasonRef:                  coreRequest.ReasonCode,
		Summary:                    coreRequest.Summary,
		MinimumRecommendedCapacity: string(coreRequest.MinimumRecommendedCapacity),
		DefaultTier:                string(request.DefaultTier),
		DefaultReasoningEffort:     string(request.DefaultReasoningEffort),
		EvidenceRefs:               append([]string(nil), request.EvidenceRefs...),
	}
}

func capacityPolicyDecisionToCoreV0(
	fallbackDecisionRef string,
	decision orquestacapacity.CapacityPolicyDecisionV0,
) orquestacionnucleoapp.CapacityDecisionPolicyDecisionV0 {
	decisionRef := strings.TrimSpace(decision.DecisionRef)
	if decisionRef == "" {
		decisionRef = strings.TrimSpace(fallbackDecisionRef)
	}
	return orquestacionnucleoapp.CapacityDecisionPolicyDecisionV0{
		DecisionRef:     decisionRef,
		Tier:            orquestacoreworkflow.OrchestrationCapacityRecommendationV0(decision.NivelCapacidad),
		ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityRecommendationV0(decision.ReasoningEffort),
		Summary:         "Decision de capacidad por puerto de politica.",
		PolicyRef:       decision.PolicyRef,
		PoolRef:         decision.PoolRef,
		ModelRef:        decision.ModelRef,
		QuotaRef:        decision.QuotaRef,
		Motivos:         append([]string(nil), decision.Motivos...),
		EvidenceRefs:    append([]string(nil), decision.EvidenceRefs...),
	}
}
