package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

// codexReceiptAckIssuesAllRecoverableV0 amplia el corte historico: conserva
// como `gate-issue` las discrepancias recuperables de ACK (tests/receipts
// incompletos, ruta normalizable y detalle sensible redactable) para que el
// Director/review decida sin colgar el tick. Mantiene corte duro para:
// JSON/forma ilegible, correlacion ajena, salida cruda, artefactos de control y
// efectos fuera del proyecto/write-set.
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
	if issue.Code == orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckForbiddenV0) {
		return codexReceiptAckIssueHasEvidenceV0(issue, "forbidden_sensitive_detail")
	}
	if issue.Code != orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0) {
		// Forma rota (ack_invalido) y correlacion (causalidad rota) siguen siendo
		// cortes duros legitimos.
		return false
	}
	field := strings.TrimSpace(issue.Field)
	switch field {
	case "tests", "test_receipts":
		return codexReceiptAckTestIssueRecoverableV0(issue)
	case "files":
		// Solo la forma/normalizacion de la ruta es recuperable. Un artefacto
		// de control o fuera de write-set (`local_artifact_excluded:*`) sigue
		// siendo corte duro por efecto fuera del proyecto/control.
		return codexReceiptAckIssueHasEvidenceV0(issue, "artifact_path_invalid")
	default:
		return false
	}
}

func codexReceiptAckTestIssueRecoverableV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	return codexReceiptAckIssueHasEvidenceV0(issue,
		"required",
		"failed_test_evidence",
		"missing_required_test_receipt",
		"required_test_receipt_mismatch",
		"required_test_receipt_schema_invalid",
		"required_test_receipt_command_required",
		"required_test_receipt_not_passed",
		"required_test_receipt_exit_code_invalid",
		"required_test_receipt_evidence_required",
		"required_test_receipt_order_required",
		"required_test_receipt_output_redaction_required",
	)
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
