package orquestacapacity

import "testing"

func TestDefaultCapacityPolicyV0DecideConRefsOpacasYEvidencia(t *testing.T) {
	decision, err := DefaultCapacityPolicyV0{
		PolicyRef:              "capacity-policy-ref-test-v0",
		PoolRef:                "capacity-pool-ref-test-v0",
		ModelRef:               "capacity-model-ref-test-v0",
		QuotaRef:               "capacity-quota-ref-test-v0",
		DefaultTier:            "high",
		DefaultReasoningEffort: "medium",
		EvidenceRefs:           []string{"evidence-ref-policy-test"},
	}.DecideCapacityV0(CapacityPolicyRequestV0{
		CapacityRequestID: "capacity-ref-test-001",
		RunRef:            "run-ref-test-001",
		PhaseRef:          "programacion",
		ReasonRef:         "implementation",
		Summary:           "Decision de capacidad para prueba.",
		EvidenceRefs:      []string{"evidence-ref-request-test"},
	})
	if err != nil {
		t.Fatalf("DecideCapacityV0: %v", err)
	}
	if decision.NivelCapacidad != "high" || decision.ReasoningEffort != "medium" {
		t.Fatalf("decision=%+v", decision)
	}
	if decision.PolicyRef != "capacity-policy-ref-test-v0" ||
		decision.PoolRef != "capacity-pool-ref-test-v0" ||
		decision.ModelRef != "capacity-model-ref-test-v0" ||
		decision.QuotaRef != "capacity-quota-ref-test-v0" {
		t.Fatalf("refs=%+v", decision)
	}
	if !capacityPolicyTestContainsV0(decision.EvidenceRefs, "evidence-ref-policy-test") ||
		!capacityPolicyTestContainsV0(decision.Motivos, "capacity-policy-port-v0") {
		t.Fatalf("evidence=%v motivos=%v", decision.EvidenceRefs, decision.Motivos)
	}
}

func TestDefaultCapacityPolicyV0DegradaXHighSinEvidencia(t *testing.T) {
	decision, err := DefaultCapacityPolicyV0{
		DefaultTier: "xhigh",
	}.DecideCapacityV0(CapacityPolicyRequestV0{CapacityRequestID: "capacity-ref-test-002"})
	if err != nil {
		t.Fatalf("DecideCapacityV0: %v", err)
	}
	if decision.NivelCapacidad != "high" {
		t.Fatalf("nivel=%s", decision.NivelCapacidad)
	}
	if !capacityPolicyTestContainsV0(decision.Motivos, "xhigh-degraded-without-evidence") {
		t.Fatalf("motivos=%v", decision.Motivos)
	}
}

func capacityPolicyTestContainsV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
