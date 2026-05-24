package orquestaruntimecodex

import "testing"

func TestCodexAckPendingRailFalsePositiveVocabularyV0(t *testing.T) {
	spec := codexSpecForTestV0()
	cases := []string{
		"secrets_policy",
		"oauth-client-secret-policy-doc",
		"credential",
		"provider ref opaco",
		"prompt policy",
		"transcript policy",
		"codex_home policy",
	}

	for _, note := range cases {
		data := []byte(codexAckJSONWithNoteForPendingRailTestV0("rail pendiente: " + note))
		ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
		if len(issues) != 0 {
			t.Fatalf("vocabulario operativo no debe bloquear issues=%+v note=%q", issues, note)
		}
		if !CodexAgentAckHasPendingRailV0(ack) {
			t.Fatalf("rail pendiente no conservado note=%q ack=%+v", note, ack)
		}
		observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
		if len(issues) != 0 {
			t.Fatalf("observacion no debe bloquear issues=%+v note=%q", issues, note)
		}
		if codexDeliveryObservationUnsafeForCoreV0(observation) {
			t.Fatalf("observacion marca rail blando como unsafe note=%q observation=%+v", note, observation)
		}
	}
}
