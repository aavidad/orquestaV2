package orquestaruntimecodex

import "testing"

func TestCodexAgentAckPendingRailEvidenceRefsV0ClasificaSinValores(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0(
		"rail pendiente: token provider home prompt access_token=redacted sin valor real",
	))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("rail pendiente no debe bloquear issues=%+v", issues)
	}

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	if len(refs) != 0 {
		t.Fatalf("rails quitados no deben generar refs: %v", refs)
	}
}

func TestCodexAgentAckPendingRailEvidenceRefsV0NoClasificaSecretoEfectivo(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0("access_token=abc123"))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("rails quitados no deben bloquear secreto efectivo: %+v", issues)
	}

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	if len(refs) != 0 {
		t.Fatalf("secreto efectivo no debe generar refs de rail blando: %v", refs)
	}
}

func TestCodexAgentAckPendingRailEvidenceRefsV0ConservaRailExplicitoSinCategoria(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0(
		"rail pendiente: detector externo dudoso sin categoria sensible",
	))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("rail pendiente explicito no debe bloquear issues=%+v", issues)
	}
	if codexAckBytesContainForbiddenDetailV0(data) {
		t.Fatalf("rail pendiente explicito no debe detectarse")
	}

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	if len(refs) != 0 {
		t.Fatalf("rail pendiente explicito no debe conservarse: %v", refs)
	}
}

func TestCodexAgentAckPendingRailEvidenceRefsV0ConservaAuthorizationYPrivateKeyRedactados(t *testing.T) {
	enableCodexRailsModeEnforcedForTestV0(t)
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0(
		"authorization: redacted; private key placeholder",
	))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("rail redactado no debe bloquear issues=%+v", issues)
	}

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	if len(refs) != 0 {
		t.Fatalf("rail redactado no debe generar refs: %v", refs)
	}
}

func codexAckPendingRailRefsContainForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
