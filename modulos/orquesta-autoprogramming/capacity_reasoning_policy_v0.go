package orquestaautoprogramming

import "strings"

const (
	CapacityPolicyRefBackgroundMediumV0 = "capacity-policy-ref-background-medium-v0"
	CapacityPolicyRefHighRiskV0         = "capacity-policy-ref-high-risk-v0"
	CapacityPolicyRefOPESDocumentV0     = "capacity-policy-ref-opes-document-v0"
)

type CapacityReasoningPolicyInputV0 struct {
	WorkProfileKind string   `json:"work_profile_kind,omitempty"`
	DomainRefs      []string `json:"domain_refs,omitempty"`
	TaskRef         string   `json:"task_ref,omitempty"`
	Title           string   `json:"title,omitempty"`
	Objective       string   `json:"objective,omitempty"`
	WriteSet        []string `json:"write_set,omitempty"`
	Risk            string   `json:"risk,omitempty"`
}

type CapacityReasoningPolicyDecisionV0 struct {
	CapacityLevel   string   `json:"capacity_level"`
	ReasoningEffort string   `json:"reasoning_effort"`
	PolicyRef       string   `json:"policy_ref"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

func DecideCapacityReasoningPolicyV0(
	input CapacityReasoningPolicyInputV0,
) CapacityReasoningPolicyDecisionV0 {
	switch {
	case capacityReasoningContainsAnyV0(input, []string{
		"opes", "plan_temario", "plan_tema", "plan_documento", "document_plan",
	}):
		return capacityReasoningDecisionV0("xhigh", CapacityPolicyRefOPESDocumentV0)
	case capacityReasoningContainsAnyV0(input, []string{
		"architecture_decision", "arquitectura amplia", "risk:high", "riesgo_alto",
	}):
		return capacityReasoningDecisionV0("high", CapacityPolicyRefHighRiskV0)
	default:
		return capacityReasoningDecisionV0("medium", CapacityPolicyRefBackgroundMediumV0)
	}
}

func capacityReasoningDecisionV0(level string, policyRef string) CapacityReasoningPolicyDecisionV0 {
	return CapacityReasoningPolicyDecisionV0{
		CapacityLevel:   level,
		ReasoningEffort: level,
		PolicyRef:       policyRef,
		EvidenceRefs:    []string{"evidence-ref-" + policyRef},
	}
}

func capacityReasoningContainsAnyV0(input CapacityReasoningPolicyInputV0, markers []string) bool {
	value := strings.ToLower(strings.Join(append(
		[]string{
			input.WorkProfileKind,
			strings.Join(input.DomainRefs, "\n"),
			input.TaskRef,
			input.Title,
			input.Objective,
			input.Risk,
		},
		input.WriteSet...,
	), "\n"))
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}
