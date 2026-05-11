package orquestacoreworkflow

import "strings"

const (
	reworkRequestProjectionResultSeparatorV0   = "#review_result:"
	reworkRequestProjectionReviewSeparatorV0   = "#review_request:"
	reworkRequestProjectionDeliverySeparatorV0 = "#delivery:"
)

type reworkRequestProjectionPartsV0 struct {
	ReworkRequestRef string
	ReviewResultRef  string
	ReviewRequestID  string
	DeliveryRef      string
}

func reworkRequestProjectionRefV0(payload ReworkRequestedPayloadV0) string {
	normalized := normalizeReworkRequestedPayloadV0(payload)
	return normalized.ReworkRequestRef +
		reworkRequestProjectionResultSeparatorV0 + normalized.ReviewResultRef +
		reworkRequestProjectionReviewSeparatorV0 + normalized.ReviewRequestID +
		reworkRequestProjectionDeliverySeparatorV0 + normalized.DeliveryRef
}

func reworkRequestProjectionPartsFromRefV0(ref string) (reworkRequestProjectionPartsV0, bool) {
	reworkRef, tail, ok := strings.Cut(strings.TrimSpace(ref), reworkRequestProjectionResultSeparatorV0)
	if !ok {
		return reworkRequestProjectionPartsV0{}, false
	}
	resultRef, tail, ok := strings.Cut(tail, reworkRequestProjectionReviewSeparatorV0)
	if !ok {
		return reworkRequestProjectionPartsV0{}, false
	}
	reviewRequestID, deliveryRef, ok := strings.Cut(tail, reworkRequestProjectionDeliverySeparatorV0)
	if !ok {
		return reworkRequestProjectionPartsV0{}, false
	}
	parts := reworkRequestProjectionPartsV0{
		ReworkRequestRef: strings.TrimSpace(reworkRef),
		ReviewResultRef:  strings.TrimSpace(resultRef),
		ReviewRequestID:  strings.TrimSpace(reviewRequestID),
		DeliveryRef:      strings.TrimSpace(deliveryRef),
	}
	return parts, parts.ReworkRequestRef != "" && parts.ReviewResultRef != "" &&
		parts.ReviewRequestID != "" && parts.DeliveryRef != ""
}

func reworkRequestProjectionForRefV0(current OrchestrationRunV0, reworkRequestRef string) (string, bool) {
	reworkRequestRef = strings.TrimSpace(reworkRequestRef)
	for _, existing := range current.ReworkRequests {
		parts, ok := reworkRequestProjectionPartsFromRefV0(existing)
		if ok && parts.ReworkRequestRef == reworkRequestRef {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func reworkRequestAlreadyReflectedV0(current OrchestrationRunV0, reworkRequestRef string) bool {
	_, ok := reworkRequestProjectionForRefV0(current, reworkRequestRef)
	return ok
}

func reworkRequestMatchesPayloadV0(current OrchestrationRunV0, payload RequestReworkCommandPayloadV0) (bool, bool) {
	existing, ok := reworkRequestProjectionForRefV0(current, payload.ReworkRequestRef)
	if !ok {
		return false, false
	}
	expected := reworkRequestProjectionRefV0(reworkRequestedPayloadFromCommandV0(payload))
	return existing == expected, existing != expected
}

func reworkRequestEventMatchesPayloadV0(current OrchestrationRunV0, payload ReworkRequestedPayloadV0) (bool, bool) {
	existing, ok := reworkRequestProjectionForRefV0(current, payload.ReworkRequestRef)
	if !ok {
		return false, false
	}
	expected := reworkRequestProjectionRefV0(payload)
	return existing == expected, existing != expected
}

func reviewResultSupportsReworkV0(current OrchestrationRunV0, reviewResultRef string, reviewRequestID string, deliveryRef string) bool {
	parts, ok := reviewResultPartsForResultRefV0(current, reviewResultRef)
	if !ok || !reviewResultStatusSupportsReworkV0(parts.Status) {
		return false
	}
	return parts.ReviewRequestID == strings.TrimSpace(reviewRequestID) &&
		parts.DeliveryRef == strings.TrimSpace(deliveryRef)
}

func reviewResultStatusSupportsReworkV0(status ReviewResultStatusV0) bool {
	return status == ReviewResultStatusChangesRequestedV0 || status == ReviewResultStatusRejectedV0
}

func reviewResultPartsForResultRefV0(current OrchestrationRunV0, reviewResultRef string) (reviewResultProjectionPartsV0, bool) {
	projection, ok := reviewResultProjectionForRefV0(current, reviewResultRef)
	if !ok {
		return reviewResultProjectionPartsV0{}, false
	}
	return reviewResultProjectionPartsFromRefV0(projection)
}

func reworkRequestRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.ReworkRequests {
		parts, ok := reworkRequestProjectionPartsFromRefV0(projection)
		if !ok || seen[parts.ReworkRequestRef] {
			return true
		}
		if !reviewResultSupportsReworkV0(run, parts.ReviewResultRef, parts.ReviewRequestID, parts.DeliveryRef) {
			return true
		}
		seen[parts.ReworkRequestRef] = true
	}
	return false
}

func reworkRequestProjectionFieldUnsafeV0(payload RequestReworkCommandPayloadV0) bool {
	return reworkProjectionContainsSeparatorV0(payload.ReworkRequestRef) ||
		reworkProjectionContainsSeparatorV0(payload.ReviewResultRef) ||
		reworkProjectionContainsSeparatorV0(payload.ReviewRequestID) ||
		reworkProjectionContainsSeparatorV0(payload.DeliveryRef)
}

func reworkRequestedProjectionFieldUnsafeV0(payload ReworkRequestedPayloadV0) bool {
	return reworkProjectionContainsSeparatorV0(payload.ReworkRequestRef) ||
		reworkProjectionContainsSeparatorV0(payload.ReviewResultRef) ||
		reworkProjectionContainsSeparatorV0(payload.ReviewRequestID) ||
		reworkProjectionContainsSeparatorV0(payload.DeliveryRef)
}

func reworkProjectionContainsSeparatorV0(value string) bool {
	return strings.Contains(value, reworkRequestProjectionResultSeparatorV0) ||
		strings.Contains(value, reworkRequestProjectionReviewSeparatorV0) ||
		strings.Contains(value, reworkRequestProjectionDeliverySeparatorV0)
}
