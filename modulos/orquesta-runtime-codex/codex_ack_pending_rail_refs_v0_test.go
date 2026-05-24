package orquestaruntimecodex

import "testing"

func TestCodexAgentAckPendingRailEvidenceRefsV0ClasificaSinValores(t *testing.T) {
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0(
		"rail pendiente: token provider home prompt access_token=redacted sin valor real",
	))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("rail pendiente no debe bloquear issues=%+v", issues)
	}

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	for _, want := range []string{
		CodexAgentAckPendingRailEvidenceRefV0,
		"ack-pending-rail:token",
		"ack-pending-rail:provider",
		"ack-pending-rail:home",
		"ack-pending-rail:prompt",
	} {
		if !codexAckPendingRailRefsContainForTestV0(refs, want) {
			t.Fatalf("refs sin %q: %v", want, refs)
		}
	}
}

func TestCodexAgentAckPendingRailEvidenceRefsV0NoClasificaSecretoEfectivo(t *testing.T) {
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0("access_token=abc123"))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	requireCodexIssueV0(t, issues, CodexConnectorAckForbiddenV0)

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	if len(refs) != 0 {
		t.Fatalf("secreto efectivo no debe generar refs de rail blando: %v", refs)
	}
}

func TestCodexAgentAckPendingRailEvidenceRefsV0ConservaRailExplicitoSinCategoria(t *testing.T) {
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0(
		"rail pendiente: detector externo dudoso sin categoria sensible",
	))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("rail pendiente explicito no debe bloquear issues=%+v", issues)
	}
	if !codexAckBytesContainForbiddenDetailV0(data) {
		t.Fatalf("rail pendiente explicito no fue detectado")
	}

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	if !codexAckPendingRailRefsContainForTestV0(refs, CodexAgentAckPendingRailEvidenceRefV0) {
		t.Fatalf("rail pendiente explicito no conservado: %v", refs)
	}
}

func TestCodexAgentAckPendingRailEvidenceRefsV0ConservaAuthorizationYPrivateKeyRedactados(t *testing.T) {
	spec := codexSpecForTestV0()
	data := []byte(codexAckJSONWithNoteForPendingRailTestV0(
		"authorization: redacted; private key placeholder",
	))
	ack, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	if len(issues) != 0 {
		t.Fatalf("rail redactado no debe bloquear issues=%+v", issues)
	}

	refs := CodexAgentAckPendingRailEvidenceRefsV0(ack)
	for _, want := range []string{
		"ack-pending-rail:token",
		"ack-pending-rail:secret",
	} {
		if !codexAckPendingRailRefsContainForTestV0(refs, want) {
			t.Fatalf("refs sin %q: %v", want, refs)
		}
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
