package orquestaruntimecodex

import "testing"

func TestCodexAckPendingRailFalsePositiveVocabularyV0(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
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
		if CodexAgentAckHasPendingRailV0(ack) {
			t.Fatalf("rail pendiente no debe conservarse note=%q ack=%+v", note, ack)
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

func TestCodexAckPendingRailFalsePositiveVocabularySinMarcadorExplicitoV0(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	spec := codexSpecForTestV0()
	cases := []string{
		"token budget policy",
		"secret handling docs",
		"prompt policy",
		"provider ref opaco",
		"codex home redacted policy",
	}

	for _, note := range cases {
		data := []byte(codexAckJSONWithNoteForPendingRailTestV0(note))
		ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
		if len(issues) != 0 {
			t.Fatalf("vocabulario generico no debe bloquear issues=%+v note=%q", issues, note)
		}
		if CodexAgentAckHasPendingRailV0(ack) {
			t.Fatalf("rail pendiente generico no debe conservarse note=%q ack=%+v", note, ack)
		}
		observation, issues := BuildCodexDeliveryObservationV0(ack, spec)
		if len(issues) != 0 {
			t.Fatalf("observacion no debe bloquear issues=%+v note=%q", issues, note)
		}
		if codexDeliveryObservationUnsafeForCoreV0(observation) {
			t.Fatalf("observacion marca vocabulario generico como unsafe note=%q observation=%+v", note, observation)
		}
	}
}
