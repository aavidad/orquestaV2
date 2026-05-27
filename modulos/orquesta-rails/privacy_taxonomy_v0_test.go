package orquestarails

import "testing"

func TestOperationalPrivacyTaxonomyV0DistingueRefsDeValores(t *testing.T) {
	refs := []string{"token_policy_ref", "payload_redaction_ref", "home_ref", "dsn_ref"}
	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			if finding := ClassifyOperationalPrivacyTextV0(ref); finding.RedactionLevel != OperationalPrivacyRedactionNoneV0 ||
				finding.ContainsSecret ||
				finding.ContainsConnectionDetail {
				t.Fatalf("ref opaca clasificada como valor privado: %+v", finding)
			}
		})
	}
}

func TestOperationalPrivacyTaxonomyV0ClasificaValoresEfectivos(t *testing.T) {
	tests := []struct {
		value string
		check func(OperationalPrivacyFindingV0) bool
	}{
		{value: "access_token=abc123456", check: func(f OperationalPrivacyFindingV0) bool { return f.ContainsSecret }},
		{value: "prompt=texto completo", check: func(f OperationalPrivacyFindingV0) bool { return f.ContainsPrompt }},
		{value: "transcript completo", check: func(f OperationalPrivacyFindingV0) bool { return f.ContainsTranscript }},
		{value: "/home/user/private", check: func(f OperationalPrivacyFindingV0) bool { return f.ContainsConnectionDetail }},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			if finding := ClassifyOperationalPrivacyTextV0(test.value); !test.check(finding) {
				t.Fatalf("valor no clasificado: %+v", finding)
			}
		})
	}
}
