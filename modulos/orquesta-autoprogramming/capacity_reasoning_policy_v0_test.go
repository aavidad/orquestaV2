package orquestaautoprogramming

import "testing"

func TestDecideCapacityReasoningPolicyV0DefaultMediumParaFondo(t *testing.T) {
	got := DecideCapacityReasoningPolicyV0(CapacityReasoningPolicyInputV0{
		WorkProfileKind: "documentation",
		TaskRef:         "task-ref-scanner",
		Title:           "Escanear backlog",
	})

	if got.CapacityLevel != "medium" ||
		got.ReasoningEffort != "medium" ||
		got.PolicyRef != CapacityPolicyRefBackgroundMediumV0 ||
		len(got.EvidenceRefs) == 0 {
		t.Fatalf("policy=%+v", got)
	}
}

func TestDecideCapacityReasoningPolicyV0OPESDocumentalOptInXHigh(t *testing.T) {
	got := DecideCapacityReasoningPolicyV0(CapacityReasoningPolicyInputV0{
		DomainRefs: []string{"domain-ref-opes"},
		WriteSet:   []string{"external/opes/plan_temario/job-ref-001"},
	})

	if got.CapacityLevel != "xhigh" ||
		got.ReasoningEffort != "xhigh" ||
		got.PolicyRef != CapacityPolicyRefOPESDocumentV0 {
		t.Fatalf("policy=%+v", got)
	}
}

func TestDecideCapacityReasoningPolicyV0NoEscalaOPESPorSubcadena(t *testing.T) {
	got := DecideCapacityReasoningPolicyV0(CapacityReasoningPolicyInputV0{
		WorkProfileKind: "implementation",
		Objective:       "Ajustar write-set scopes para autoprogramacion generica.",
		WriteSet:        []string{"modulos/orquesta-autoprogramming/scope_policy.go"},
	})

	if got.CapacityLevel != "medium" ||
		got.ReasoningEffort != "medium" ||
		got.PolicyRef != CapacityPolicyRefBackgroundMediumV0 {
		t.Fatalf("policy=%+v", got)
	}
}

func TestDecideCapacityReasoningPolicyV0RiesgoAltoUsaHighNoXHigh(t *testing.T) {
	got := DecideCapacityReasoningPolicyV0(CapacityReasoningPolicyInputV0{
		Objective: "Decidir arquitectura amplia para runtime externo",
	})

	if got.CapacityLevel != "high" ||
		got.ReasoningEffort != "high" ||
		got.PolicyRef != CapacityPolicyRefHighRiskV0 {
		t.Fatalf("policy=%+v", got)
	}
}
