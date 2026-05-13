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
	for _, issue := range result.Issues {
		code := strings.TrimSpace(issue.Code)
		if code != "" {
			values = append(values, "gate-issue:"+code)
		}
	}
	return compactCodexDeliveryRefsV0(values)
}
