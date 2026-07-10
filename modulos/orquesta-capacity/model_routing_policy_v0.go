package orquestacapacity

import "strings"

// ModelRoutingLevelV0 describes work severity without provider or model terms.
type ModelRoutingLevelV0 string

const (
	ModelRoutingLevelNormalV0   ModelRoutingLevelV0 = "normal"
	ModelRoutingLevelComplexV0  ModelRoutingLevelV0 = "complex"
	ModelRoutingLevelCriticalV0 ModelRoutingLevelV0 = "critical"
)

type ModelRoutingRequestV0 struct {
	TaskRef               string              `json:"task_ref"`
	Level                 ModelRoutingLevelV0 `json:"level"`
	Trivial               bool                `json:"trivial,omitempty"`
	ReasonRef             string              `json:"reason_ref,omitempty"`
	EvidenceRefs          []string            `json:"evidence_refs,omitempty"`
	RequestedEffort       string              `json:"requested_effort,omitempty"`
	XHighAuthorizationRef string              `json:"xhigh_authorization_ref,omitempty"`
}

type ModelRoutingPolicyV0 struct {
	PolicyRef        string `json:"policy_ref"`
	Strict           bool   `json:"strict"`
	TrivialModelRef  string `json:"trivial_model_ref"`
	NormalModelRef   string `json:"normal_model_ref"`
	CriticalModelRef string `json:"critical_model_ref"`
	TrivialEffort    string `json:"trivial_effort"`
	NormalEffort     string `json:"normal_effort"`
	ComplexEffort    string `json:"complex_effort"`
	CriticalEffort   string `json:"critical_effort"`
}

type ModelRoutingDecisionV0 struct {
	TaskRef               string              `json:"task_ref"`
	Level                 ModelRoutingLevelV0 `json:"level"`
	Trivial               bool                `json:"trivial,omitempty"`
	SelectedModelRef      string              `json:"selected_model_ref"`
	ReasoningEffort       string              `json:"reasoning_effort"`
	PolicyRef             string              `json:"policy_ref"`
	ReasonRef             string              `json:"reason_ref,omitempty"`
	EvidenceRefs          []string            `json:"evidence_refs,omitempty"`
	XHighAuthorizationRef string              `json:"xhigh_authorization_ref,omitempty"`
	Rejected              bool                `json:"rejected,omitempty"`
	RejectionRef          string              `json:"rejection_ref,omitempty"`
}

func ValidModelRoutingLevelV0(level ModelRoutingLevelV0) bool {
	switch level {
	case ModelRoutingLevelNormalV0, ModelRoutingLevelComplexV0, ModelRoutingLevelCriticalV0:
		return true
	default:
		return false
	}
}

func ResolveModelRoutingV0(policy ModelRoutingPolicyV0, request ModelRoutingRequestV0) ModelRoutingDecisionV0 {
	decision := ModelRoutingDecisionV0{
		TaskRef:               strings.TrimSpace(request.TaskRef),
		Level:                 ModelRoutingLevelV0(strings.TrimSpace(string(request.Level))),
		Trivial:               request.Trivial,
		PolicyRef:             strings.TrimSpace(policy.PolicyRef),
		ReasonRef:             strings.TrimSpace(request.ReasonRef),
		EvidenceRefs:          compactCapacityPolicyStringsV0(request.EvidenceRefs),
		XHighAuthorizationRef: strings.TrimSpace(request.XHighAuthorizationRef),
	}
	if decision.TaskRef == "" {
		return rejectModelRoutingV0(decision, "model-routing-task-ref-missing")
	}
	if decision.PolicyRef == "" {
		return rejectModelRoutingV0(decision, "model-routing-policy-ref-missing")
	}
	if !ValidModelRoutingLevelV0(decision.Level) {
		return rejectModelRoutingV0(decision, "model-routing-level-invalid")
	}
	if rejection := validateModelRoutingPolicyV0(policy); rejection != "" {
		return rejectModelRoutingV0(decision, rejection)
	}
	if decision.Trivial && decision.Level != ModelRoutingLevelNormalV0 {
		return rejectModelRoutingV0(decision, "model-routing-trivial-level-invalid")
	}
	if decision.Level == ModelRoutingLevelCriticalV0 && !modelRoutingCausalityCompleteV0(decision) {
		return rejectModelRoutingV0(decision, "model-routing-critical-authorization-missing")
	}

	switch {
	case decision.Trivial:
		decision.SelectedModelRef = strings.TrimSpace(policy.TrivialModelRef)
		decision.ReasoningEffort = strings.TrimSpace(policy.TrivialEffort)
	case decision.Level == ModelRoutingLevelCriticalV0:
		decision.SelectedModelRef = strings.TrimSpace(policy.CriticalModelRef)
		decision.ReasoningEffort = strings.TrimSpace(policy.CriticalEffort)
	case decision.Level == ModelRoutingLevelComplexV0:
		decision.SelectedModelRef = strings.TrimSpace(policy.NormalModelRef)
		decision.ReasoningEffort = strings.TrimSpace(policy.ComplexEffort)
	default:
		decision.SelectedModelRef = strings.TrimSpace(policy.NormalModelRef)
		decision.ReasoningEffort = strings.TrimSpace(policy.NormalEffort)
	}

	requestedEffort := strings.TrimSpace(request.RequestedEffort)
	if requestedEffort == "" {
		return decision
	}
	if requestedEffort != "xhigh" {
		return rejectModelRoutingV0(decision, "model-routing-effort-override-prohibited")
	}
	if decision.Trivial || decision.XHighAuthorizationRef == "" || !modelRoutingCausalityCompleteV0(decision) {
		return rejectModelRoutingV0(decision, "model-routing-xhigh-authorization-missing")
	}
	decision.ReasoningEffort = "xhigh"
	return decision
}

func validateModelRoutingPolicyV0(policy ModelRoutingPolicyV0) string {
	if !policy.Strict {
		return "model-routing-policy-not-strict"
	}
	for _, value := range []string{policy.TrivialModelRef, policy.NormalModelRef, policy.CriticalModelRef} {
		if strings.TrimSpace(value) == "" {
			return "model-routing-model-ref-missing"
		}
	}
	if strings.TrimSpace(policy.TrivialEffort) != "low" ||
		strings.TrimSpace(policy.NormalEffort) != "medium" ||
		strings.TrimSpace(policy.ComplexEffort) != "high" ||
		strings.TrimSpace(policy.CriticalEffort) != "high" {
		return "model-routing-policy-effort-invalid"
	}
	return ""
}

func modelRoutingCausalityCompleteV0(decision ModelRoutingDecisionV0) bool {
	return decision.TaskRef != "" && decision.ReasonRef != "" && len(decision.EvidenceRefs) > 0
}

func rejectModelRoutingV0(decision ModelRoutingDecisionV0, ref string) ModelRoutingDecisionV0 {
	decision.SelectedModelRef = ""
	decision.ReasoningEffort = ""
	decision.Rejected = true
	decision.RejectionRef = ref
	return decision
}
