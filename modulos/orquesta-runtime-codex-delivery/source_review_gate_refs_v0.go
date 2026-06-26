package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexReviewGateReviewRequestIDV0(deliveryRef string) string {
	return "review-request-ref-" + strings.TrimSpace(deliveryRef)
}

func codexReviewGateReviewResultRefV0(deliveryRef string) string {
	return "review-result-ref-" + strings.TrimSpace(deliveryRef)
}

func codexReviewGateAcceptedReviewRefV0(deliveryRef string) string {
	return "accepted-review-ref-" + strings.TrimSpace(deliveryRef)
}

func codexReviewGateSummaryV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
) string {
	if result.Accepted && result.RequiresFollowup {
		return "Entrega aceptada con rail blando para follow-up no bloqueante."
	}
	if result.Accepted {
		return "Entrega aceptada por gate de revision."
	}
	if len(result.Issues) == 1 {
		return "Entrega requiere revision: " + strings.TrimSpace(result.Issues[0].Code)
	}
	return "Entrega requiere revision: multiples incidencias."
}

func codexReviewGateEvidenceRefsV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
) []string {
	values := []string{strings.TrimSpace(ack.AckRef)}
	values = append(values, orquestaruntimecodex.CodexAgentAckPendingRailEvidenceRefsV0(ack)...)
	if result.RecommendedAction != "" {
		values = append(values, "gate-action:"+string(result.RecommendedAction))
	}
	if result.RequiresFollowup {
		values = append(values, "gate-followup-required")
	}
	for _, issue := range result.Issues {
		ref := codexReviewGateIssueEvidenceRefV0(issue.Code)
		if ref != "" {
			values = append(values, ref)
		}
	}
	values = codexDeliveryWithWriteSetEscapeAliasV0(values)
	return compactCodexDeliveryRefsV0(values)
}

func codexReviewGateIssueEvidenceRefV0(code string) string {
	code = strings.TrimSpace(code)
	for {
		if !strings.HasPrefix(strings.ToLower(code), "gate-issue:") {
			break
		}
		code = strings.TrimSpace(code[len("gate-issue:"):])
	}
	if code == "" {
		return ""
	}
	return "gate-issue:" + code
}
