package orquestadirectoragent

import "testing"

func TestValidateDirectorAgentDecisionV0PermiteRuntimeYProviderComoRefsOpacas(t *testing.T) {
	decision := validDirectorAgentDecisionV0()
	decision.Summary = "usar runtime y provider como refs opacas"
	decision.EvidenceRefs = []string{"evidence-ref-runtime-provider-model"}

	if issues := ValidateDirectorAgentDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("issues inesperados: %+v", issues)
	}
}
