package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexReceiptAckIssuesOnlyReviewableFailedTestEvidenceV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if issue.Code != orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0) {
			return false
		}
		field := strings.TrimSpace(issue.Field)
		if field != "tests" && field != "test_receipts" {
			return false
		}
		if !codexReceiptAckIssueHasEvidenceV0(issue,
			"failed_test_evidence",
			"required_test_receipt_not_passed",
			"required_test_receipt_exit_code_invalid",
		) {
			return false
		}
	}
	return true
}

// codexReceiptAckIssuesAllRecoverableV0 amplia el corte historico: ademas de la
// evidencia reviewable de tests, trata como recuperables las discrepancias de
// forma de ruta/nombre cercano del ACK (`artifact_path_invalid`) y el detalle
// sensible (`forbidden_sensitive_detail`). Esas discrepancias se normalizan/
// redactan y se conservan como `gate-issue` para que el Director/review decida,
// en lugar de colgar al agente (la entrega ya esta en disco; vetar el ACK no
// protege nada). Mantiene corte duro para: JSON/forma ilegible, correlacion
// ajena (causalidad rota) y artefactos de control/fuera de write-set
// (`local_artifact_excluded`). Alinea el codigo con
// docs/estado_actual_2026-05-17.md:39-64.
func codexReceiptAckIssuesAllRecoverableV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if codexReceiptAckIssueRecoverableV0(issue) {
			continue
		}
		return false
	}
	return true
}

func codexReceiptAckIssueRecoverableV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	// Detalle sensible: recuperable. No vetamos la entrega entera por un
	// patron tipo `password:`/`api_key=` que puede aparecer en texto pedagogico
	// legitimo (p. ej. un temario). Se conserva como gate-issue redactado y la
	// review/Director decide. Solo se proyecta la categoria, nunca el valor.
	if issue.Code == orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckForbiddenV0) {
		return true
	}
	if issue.Code != orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0) {
		// Forma rota (ack_invalido) y correlacion (causalidad rota) siguen siendo
		// cortes duros legitimos.
		return false
	}
	field := strings.TrimSpace(issue.Field)
	switch field {
	case "tests", "test_receipts":
		return codexReceiptAckIssueHasEvidenceV0(issue,
			"failed_test_evidence",
			"required_test_receipt_not_passed",
			"required_test_receipt_exit_code_invalid",
		)
	case "files":
		// Solo la forma/normalizacion de la ruta es recuperable. Un artefacto
		// de control o fuera de write-set (`local_artifact_excluded:*`) sigue
		// siendo corte duro por efecto fuera del proyecto/control.
		return codexReceiptAckIssueHasEvidenceV0(issue, "artifact_path_invalid")
	default:
		return false
	}
}

// codexReceiptAckIssueGateRefsV0 proyecta los issues recuperables del ACK como
// refs compactas `gate-issue:ack_*` para que viajen como evidencia de review en
// lugar de perderse o colgar al agente.
func codexReceiptAckIssueGateRefsV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) []string {
	refs := make([]string, 0, len(issues))
	for _, issue := range issues {
		field := strings.TrimSpace(issue.Field)
		if field == "" {
			field = "agent_ack"
		}
		// Para detalle sensible solo emitimos un marcador de categoria redactado;
		// nunca su evidencia, que podria arrastrar el valor sensible al ref.
		if issue.Code == orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckForbiddenV0) {
			refs = append(refs, "gate-issue:ack_sensitive_detail_redacted")
			continue
		}
		refs = append(refs, "gate-issue:ack_"+field)
		for _, evidence := range issue.Evidence {
			evidence = strings.TrimSpace(evidence)
			if !codexReceiptWorktreeIssueEvidenceSafeV0(evidence) {
				continue
			}
			refs = append(refs, "gate-issue:ack_"+field+":"+evidence)
		}
	}
	return compactCodexDeliveryRefsV0(refs)
}

func codexReceiptAckIssueHasEvidenceV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
	allowed ...string,
) bool {
	for _, evidence := range issue.Evidence {
		for _, item := range allowed {
			if evidence == item {
				return true
			}
		}
	}
	return false
}
