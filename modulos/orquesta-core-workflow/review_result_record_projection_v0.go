package orquestacoreworkflow

import "strings"

const (
	reviewResultProjectionStatusSeparatorV0        = "#review_result:"
	reviewResultProjectionReviewRequestSeparatorV0 = "#review_request:"
	reviewResultProjectionDeliverySeparatorV0      = "#delivery:"
)

type reviewResultProjectionPartsV0 struct {
	ReviewResultRef string
	Status          ReviewResultStatusV0
	ReviewRequestID string
	DeliveryRef     string
}

func reviewResultProjectionRefV0(payload ReviewResultV0) string {
	normalized := NormalizeReviewResultV0(payload)
	return normalized.ReviewResultRef +
		reviewResultProjectionStatusSeparatorV0 + string(normalized.Status) +
		reviewResultProjectionReviewRequestSeparatorV0 + normalized.ReviewRequestID +
		reviewResultProjectionDeliverySeparatorV0 + normalized.DeliveryRef
}

func reviewResultProjectionPartsFromRefV0(ref string) (reviewResultProjectionPartsV0, bool) {
	resultRef, tail, ok := strings.Cut(strings.TrimSpace(ref), reviewResultProjectionStatusSeparatorV0)
	if !ok {
		return reviewResultProjectionPartsV0{}, false
	}
	status, tail, ok := strings.Cut(tail, reviewResultProjectionReviewRequestSeparatorV0)
	if !ok {
		return reviewResultProjectionPartsV0{}, false
	}
	reviewRequestID, deliveryRef, ok := strings.Cut(tail, reviewResultProjectionDeliverySeparatorV0)
	if !ok {
		return reviewResultProjectionPartsV0{}, false
	}
	parts := reviewResultProjectionPartsV0{
		ReviewResultRef: strings.TrimSpace(resultRef),
		Status:          ReviewResultStatusV0(strings.TrimSpace(status)),
		ReviewRequestID: strings.TrimSpace(reviewRequestID),
		DeliveryRef:     strings.TrimSpace(deliveryRef),
	}
	return parts, parts.ReviewResultRef != "" && parts.ReviewRequestID != "" && parts.DeliveryRef != ""
}

func reviewResultProjectionForRefV0(current OrchestrationRunV0, reviewResultRef string) (string, bool) {
	reviewResultRef = strings.TrimSpace(reviewResultRef)
	for _, existing := range current.ReviewResults {
		parts, ok := reviewResultProjectionPartsFromRefV0(existing)
		if ok && parts.ReviewResultRef == reviewResultRef {
			return strings.TrimSpace(existing), true
		}
	}
	return "", false
}

func acceptedReviewResultAlreadyReflectedV0(current OrchestrationRunV0, reviewRequestID string, deliveryRef string) bool {
	reviewRequestID = strings.TrimSpace(reviewRequestID)
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, existing := range current.ReviewResults {
		parts, ok := reviewResultProjectionPartsFromRefV0(existing)
		if ok &&
			parts.Status == ReviewResultStatusAcceptedV0 &&
			parts.ReviewRequestID == reviewRequestID &&
			parts.DeliveryRef == deliveryRef {
			return true
		}
	}
	return false
}

func reviewResultRecordMatchesPayloadV0(current OrchestrationRunV0, payload ReviewResultV0) (bool, bool) {
	existing, ok := reviewResultProjectionForRefV0(current, payload.ReviewResultRef)
	if !ok {
		return false, false
	}
	expected := reviewResultProjectionRefV0(payload)
	return existing == expected, existing != expected
}

func reviewResultProjectionFieldUnsafeV0(payload ReviewResultV0) bool {
	return reviewResultProjectionContainsSeparatorV0(payload.ReviewResultRef) ||
		reviewResultProjectionContainsSeparatorV0(payload.ReviewRequestID) ||
		reviewResultProjectionContainsSeparatorV0(payload.DeliveryRef)
}

func reviewResultProjectionContainsSeparatorV0(value string) bool {
	return strings.Contains(value, reviewResultProjectionStatusSeparatorV0) ||
		strings.Contains(value, reviewResultProjectionReviewRequestSeparatorV0) ||
		strings.Contains(value, reviewResultProjectionDeliverySeparatorV0)
}
