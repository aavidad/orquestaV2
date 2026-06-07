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

func TestOperationalPrivacyTaxonomyV0RailsOfflineNoClasificaValoresEfectivos(t *testing.T) {
	for _, value := range []string{
		"access_token=abc123456",
		"prompt=texto completo",
		"transcript completo",
		"/home/user/private",
	} {
		t.Run(value, func(t *testing.T) {
			if finding := ClassifyOperationalPrivacyTextV0(value); finding.RedactionLevel != OperationalPrivacyRedactionNoneV0 ||
				finding.ContainsSecret ||
				finding.ContainsPrompt ||
				finding.ContainsTranscript ||
				finding.ContainsConnectionDetail {
				t.Fatalf("rails offline no debe clasificar ni bloquear valor: %+v", finding)
			}
		})
	}
}
