package orquestacapacity

import "strings"

type CapacityPolicyRequestV0 struct {
	CapacityRequestID          string   `json:"capacity_request_id"`
	DecisionRef                string   `json:"decision_ref,omitempty"`
	RunRef                     string   `json:"run_ref"`
	PhaseRef                   string   `json:"phase_ref"`
	TaskRef                    string   `json:"task_ref,omitempty"`
	ReasonRef                  string   `json:"reason_ref"`
	Summary                    string   `json:"summary"`
	MinimumRecommendedCapacity string   `json:"minimum_recommended_capacity,omitempty"`
	DefaultTier                string   `json:"default_tier,omitempty"`
	DefaultReasoningEffort     string   `json:"default_reasoning_effort,omitempty"`
	EvidenceRefs               []string `json:"evidence_refs,omitempty"`
}

type CapacityPolicyDecisionV0 struct {
	DecisionRef     string   `json:"decision_ref"`
	NivelCapacidad  string   `json:"nivel_capacidad"`
	ReasoningEffort string   `json:"reasoning_effort"`
	PolicyRef       string   `json:"policy_ref"`
	PoolRef         string   `json:"pool_ref"`
	ModelRef        string   `json:"model_ref"`
	QuotaRef        string   `json:"quota_ref"`
	Motivos         []string `json:"motivos"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type DefaultCapacityPolicyV0 struct {
	PolicyRef              string
	PoolRef                string
	ModelRef               string
	QuotaRef               string
	DefaultTier            string
	DefaultReasoningEffort string
	Summary                string
	EvidenceRefs           []string
}

func (policy DefaultCapacityPolicyV0) DecideCapacityV0(
	request CapacityPolicyRequestV0,
) (CapacityPolicyDecisionV0, error) {
	request = normalizeCapacityPolicyRequestV0(request)
	policyRef := capacityPolicyRefOrDefaultV0(policy.PolicyRef, "capacity-policy-ref-default-v0")
	poolRef := capacityPolicyRefOrDefaultV0(policy.PoolRef, "capacity-pool-ref-default-v0")
	modelRef := capacityPolicyRefOrDefaultV0(policy.ModelRef, "capacity-model-ref-default-v0")
	quotaRef := capacityPolicyRefOrDefaultV0(policy.QuotaRef, "capacity-quota-ref-default-v0")
	var issues []CapacityDecisionIssueV0
	issues = appendCapacityPolicyRefIssueV0(issues, "policy_ref", policyRef)
	issues = appendCapacityPolicyRefIssueV0(issues, "pool_ref", poolRef)
	issues = appendCapacityPolicyRefIssueV0(issues, "model_ref", modelRef)
	issues = appendCapacityPolicyRefIssueV0(issues, "quota_ref", quotaRef)
	if len(issues) > 0 {
		return CapacityPolicyDecisionV0{}, CapacityDecisionValidationErrorV0{Issues: issues}
	}
	tier := chooseCapacityPolicyLevelV0(policy.DefaultTier, request.DefaultTier, request.MinimumRecommendedCapacity)
	reasoning := chooseCapacityPolicyLevelV0(policy.DefaultReasoningEffort, request.DefaultReasoningEffort, tier)
	motivos := []string{"capacity-policy-port-v0", "request-capacity-decision"}
	if tier == "xhigh" && len(compactCapacityPolicyStringsV0(append(request.EvidenceRefs, policy.EvidenceRefs...))) == 0 {
		tier = "high"
		reasoning = chooseCapacityPolicyLevelV0("", reasoning, tier)
		motivos = append(motivos, "xhigh-degraded-without-evidence")
	}
	decisionRef := strings.TrimSpace(request.DecisionRef)
	if decisionRef == "" {
		decisionRef = "capacity-decision-ref-" + strings.TrimSpace(request.CapacityRequestID)
	}
	return CapacityPolicyDecisionV0{
		DecisionRef:     decisionRef,
		NivelCapacidad:  tier,
		ReasoningEffort: reasoning,
		PolicyRef:       policyRef,
		PoolRef:         poolRef,
		ModelRef:        modelRef,
		QuotaRef:        quotaRef,
		Motivos:         compactCapacityPolicyStringsV0(motivos),
		EvidenceRefs:    compactCapacityPolicyStringsV0(append(request.EvidenceRefs, policy.EvidenceRefs...)),
	}, nil
}

func appendCapacityPolicyRefIssueV0(issues []CapacityDecisionIssueV0, field string, value string) []CapacityDecisionIssueV0 {
	if strings.TrimSpace(value) == "" || looksLikeSecretV0(value) || looksLikeHomePathV0(value) ||
		!opaqueRefPatternV0.MatchString(value) || containsForbiddenBrandV0(value) {
		return append(issues, CapacityDecisionIssueV0{Code: ErrReferenciaNoOpacaV0, Field: field})
	}
	return issues
}

func normalizeCapacityPolicyRequestV0(request CapacityPolicyRequestV0) CapacityPolicyRequestV0 {
	request.CapacityRequestID = strings.TrimSpace(request.CapacityRequestID)
	request.DecisionRef = strings.TrimSpace(request.DecisionRef)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.PhaseRef = strings.TrimSpace(request.PhaseRef)
	request.TaskRef = strings.TrimSpace(request.TaskRef)
	request.ReasonRef = strings.TrimSpace(request.ReasonRef)
	request.Summary = strings.TrimSpace(request.Summary)
	request.MinimumRecommendedCapacity = strings.TrimSpace(request.MinimumRecommendedCapacity)
	request.DefaultTier = strings.TrimSpace(request.DefaultTier)
	request.DefaultReasoningEffort = strings.TrimSpace(request.DefaultReasoningEffort)
	request.EvidenceRefs = compactCapacityPolicyStringsV0(request.EvidenceRefs)
	return request
}

func chooseCapacityPolicyLevelV0(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if isOneOfV0(trimmed, "low", "medium", "high", "xhigh") {
			return trimmed
		}
	}
	return "medium"
}

func capacityPolicyRefOrDefaultV0(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func compactCapacityPolicyStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}
