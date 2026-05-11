package orquestacoreworkflow

import "strings"

func NewReviewResultV0(result ReviewResultV0) (ReviewResultV0, error) {
	normalized := NormalizeReviewResultV0(result)
	if err := ValidateReviewResultV0(normalized); err != nil {
		return ReviewResultV0{}, err
	}
	return normalized, nil
}

func NormalizeReviewResultV0(result ReviewResultV0) ReviewResultV0 {
	return ReviewResultV0{
		ReviewResultRef: strings.TrimSpace(result.ReviewResultRef),
		ReviewRequestID: strings.TrimSpace(result.ReviewRequestID),
		DeliveryRef:     strings.TrimSpace(result.DeliveryRef),
		Status:          ReviewResultStatusV0(strings.TrimSpace(string(result.Status))),
		Summary:         strings.TrimSpace(result.Summary),
		EvidenceRefs:    normalizeReviewResultStringsV0(result.EvidenceRefs),
		QualityGateRef:  strings.TrimSpace(result.QualityGateRef),
	}
}

func ValidateReviewResultV0(result ReviewResultV0) error {
	if err := validateReviewResultRequiredFieldsV0(result); err != nil {
		return err
	}
	if err := ValidateReviewResultStatusV0(result.Status); err != nil {
		return err
	}
	if err := validateReviewResultCollectionsV0(result); err != nil {
		return err
	}
	if reviewResultHasLongStringV0(reviewResultTextFieldsV0(result)) {
		return reviewResultErrorV0(ErrReviewResultPayloadInvalidoV0, "payload")
	}
	if reviewResultHasForbiddenDetailsV0(reviewResultTextFieldsV0(result)) {
		return reviewResultErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateReviewResultCompactPayloadV0(result)
}

func IsSupportedReviewResultStatusV0(status ReviewResultStatusV0) bool {
	switch ReviewResultStatusV0(strings.TrimSpace(string(status))) {
	case ReviewResultStatusAcceptedV0,
		ReviewResultStatusChangesRequestedV0,
		ReviewResultStatusRejectedV0:
		return true
	default:
		return false
	}
}

func ValidateReviewResultStatusV0(status ReviewResultStatusV0) error {
	if !IsSupportedReviewResultStatusV0(status) {
		return reviewResultErrorV0(ErrReviewResultStatusNoSoportadoV0, "status")
	}
	return nil
}

func ReviewResultIsAcceptedV0(result ReviewResultV0) bool {
	normalized := NormalizeReviewResultV0(result)
	if normalized.Status != ReviewResultStatusAcceptedV0 {
		return false
	}
	return ValidateReviewResultV0(normalized) == nil
}
