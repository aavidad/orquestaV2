package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type CapacityDecisionPolicyPortV0 interface {
	DecideCapacityV0(context.Context, CapacityDecisionPolicyRequestV0) (CapacityDecisionPolicyDecisionV0, error)
}

type CapacityDecisionPolicyRequestV0 struct {
	Request                orquestacoreworkflow.CapacityDecisionRequestV0
	DecisionRef            string
	DefaultTier            orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	DefaultReasoningEffort orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	EvidenceRefs           []string
}

type CapacityDecisionPolicyDecisionV0 struct {
	DecisionRef     string
	Tier            orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	ReasoningEffort orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	Summary         string
	PolicyRef       string
	PoolRef         string
	ModelRef        string
	QuotaRef        string
	Motivos         []string
	EvidenceRefs    []string
}

func (decision CapacityDecisionPolicyDecisionV0) normalizeV0(
	fallback CapacityDecisionPolicyRequestV0,
) CapacityDecisionPolicyDecisionV0 {
	decision.DecisionRef = strings.TrimSpace(decision.DecisionRef)
	if decision.DecisionRef == "" {
		decision.DecisionRef = strings.TrimSpace(fallback.DecisionRef)
	}
	decision.Tier = capacityDecisionTierV0(decision.Tier, fallback.Request.MinimumRecommendedCapacity)
	decision.ReasoningEffort = capacityDecisionTierV0(decision.ReasoningEffort, decision.Tier)
	decision.Summary = strings.TrimSpace(decision.Summary)
	decision.PolicyRef = strings.TrimSpace(decision.PolicyRef)
	decision.PoolRef = strings.TrimSpace(decision.PoolRef)
	decision.ModelRef = strings.TrimSpace(decision.ModelRef)
	decision.QuotaRef = strings.TrimSpace(decision.QuotaRef)
	decision.Motivos = compactStringsV0(decision.Motivos)
	decision.EvidenceRefs = compactStringsV0(decision.EvidenceRefs)
	return decision
}
