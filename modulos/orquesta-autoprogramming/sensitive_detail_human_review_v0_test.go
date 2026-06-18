package orquestaautoprogramming

import "testing"

// Politica de revision (decidida 2026-06-18): el detalle sensible y el artefacto
// sin ACK conservan la entrega (no se veta al agente) pero EXIGEN revision antes
// del cierre. La revision la hace OTRO AGENTE por defecto (request_changes/rework);
// solo escala a humano si el Director no puede resolverlo. Por tanto NO son
// advisory (no se aceptan en silencio) pero TAMPOCO bloquean cierre en duro hacia
// humano: derivan a review/rework por agente.

func TestSensitiveDetailGateIssueExigeRevision(t *testing.T) {
	for _, ref := range []string{
		"gate-issue:ack_sensitive_detail_redacted",
		"gate-issue:forbidden_sensitive_detail",
		"review-gate-issue:detalle_sensible",
	} {
		if AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(ref) {
			t.Fatalf("%q no debe ser advisory: exige revision, no aceptacion en silencio", ref)
		}
		if AutoprogrammingReviewGateIssueCodeBlocksClosureV0(ref) {
			t.Fatalf("%q no debe bloquear en duro hacia humano: la revision la hace un agente primero", ref)
		}
	}
}

func TestArtifactWithoutAckGateIssueExigeRevision(t *testing.T) {
	for _, ref := range []string{
		"gate-issue:artifact_without_ack_requires_review",
		"gate-issue:artifact_without_ack",
		"review-gate-issue:artefacto_sin_ack",
	} {
		if AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(ref) {
			t.Fatalf("%q no debe ser advisory: exige revision por agente antes del cierre", ref)
		}
		if AutoprogrammingReviewGateIssueCodeBlocksClosureV0(ref) {
			t.Fatalf("%q no debe bloquear en duro hacia humano: la revision la hace un agente primero", ref)
		}
	}
}
