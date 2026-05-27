package orquestaruntimecodexdelivery

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func codexReviewGateTerminalProjectedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
	status orquestacoreworkflow.ReviewResultStatusV0,
) bool {
	reviewResultRef := codexReviewGateReviewResultRefV0(deliveryRef)
	if status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 &&
		codexReviewGateReworkForDeliveryProjectedV0(run.ReworkRequests, deliveryRef) {
		return true
	}
	switch status {
	case orquestacoreworkflow.ReviewResultStatusAcceptedV0:
		return stringInCodexDeliverySetV0(
			run.AcceptedReviews,
			codexReviewGateAcceptedReviewRefV0(deliveryRef),
		)
	case orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		orquestacoreworkflow.ReviewResultStatusRejectedV0:
		return false
	default:
		return codexReviewGateResultRefInSetV0(run.ReviewResults, reviewResultRef)
	}
}

func codexReviewGateReworkForDeliveryProjectedV0(values []string, deliveryRef string) bool {
	deliveryRef = strings.TrimSpace(deliveryRef)
	if deliveryRef == "" {
		return false
	}
	for _, value := range values {
		if strings.Contains(strings.TrimSpace(value), "#delivery:"+deliveryRef) {
			return true
		}
	}
	return false
}

func codexReviewGateResultRefInSetV0(values []string, reviewResultRef string) bool {
	reviewResultRef = strings.TrimSpace(reviewResultRef)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == reviewResultRef || strings.HasPrefix(value, reviewResultRef+"#") {
			return true
		}
	}
	return false
}
