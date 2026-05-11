package orquestacorereplanner

import (
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func ValidateReviewReworkSignalV0(signal ReviewReworkSignalV0) error {
	if err := validateReviewReworkSignalRequiredFieldsV0(signal); err != nil {
		return err
	}
	if err := ValidateReviewReworkStatusV0(signal.ReviewStatus); err != nil {
		return err
	}
	if err := ValidateReviewReworkActionV0(signal.ReviewStatus, signal.RequestedAction); err != nil {
		return err
	}
	if err := validateReviewReworkSignalCollectionsV0(signal); err != nil {
		return err
	}
	if replanProposalHasLongStringV0(reviewReworkSignalTextFieldsV0(signal)) {
		return reviewReworkSignalErrorV0(ErrReviewReworkSignalPayloadInvalidoV0, "payload")
	}
	if replanProposalHasForbiddenDetailsV0(reviewReworkSignalTextFieldsV0(signal)) {
		return reviewReworkSignalErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateReviewReworkSignalCompactPayloadV0(signal)
}

func ValidateReviewReworkStatusV0(status orquestacoreworkflow.ReviewResultStatusV0) error {
	switch orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(status))) {
	case orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		orquestacoreworkflow.ReviewResultStatusRejectedV0:
		return nil
	default:
		return reviewReworkSignalErrorV0(ErrReviewReworkStatusNoSoportadoV0, "review_status")
	}
}

func ValidateReviewReworkActionV0(status orquestacoreworkflow.ReviewResultStatusV0, action ReplanRecommendedActionV0) error {
	status = orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(status)))
	action = ReplanRecommendedActionV0(strings.TrimSpace(string(action)))

	if status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 {
		switch action {
		case ReplanActionSplitTaskV0, ReplanActionRetryTaskV0, ReplanActionAskDirectorV0:
			return nil
		default:
			return reviewReworkSignalErrorV0(ErrReviewReworkActionNoSoportadaV0, "requested_action")
		}
	}

	if status == orquestacoreworkflow.ReviewResultStatusRejectedV0 {
		switch action {
		case ReplanActionSplitTaskV0, ReplanActionRetryTaskV0, ReplanActionAskDirectorV0:
			return nil
		default:
			return reviewReworkSignalErrorV0(ErrReviewReworkActionNoSoportadaV0, "requested_action")
		}
	}

	return reviewReworkSignalErrorV0(ErrReviewReworkStatusNoSoportadoV0, "review_status")
}

func validateReviewReworkSignalRequiredFieldsV0(signal ReviewReworkSignalV0) error {
	fields := map[string]string{
		"signal_ref":         signal.SignalRef,
		"run_ref":            signal.RunRef,
		"task_ref":           signal.TaskRef,
		"review_request_ref": signal.ReviewRequestRef,
		"delivery_ref":       signal.DeliveryRef,
		"review_result_ref":  signal.ReviewResultRef,
		"review_status":      string(signal.ReviewStatus),
		"requested_action":   string(signal.RequestedAction),
		"summary":            signal.Summary,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return reviewReworkSignalErrorV0(ErrReviewReworkSignalInvalidoV0, field)
		}
	}
	return nil
}

func validateReviewReworkSignalCollectionsV0(signal ReviewReworkSignalV0) error {
	if len(signal.EvidenceRefs) > maxReplanProposalEvidenceRefsV0 {
		return reviewReworkSignalErrorV0(ErrReviewReworkSignalInvalidoV0, "evidence_refs")
	}
	for _, ref := range signal.EvidenceRefs {
		if strings.TrimSpace(ref) == "" {
			return reviewReworkSignalErrorV0(ErrReviewReworkSignalInvalidoV0, "evidence_refs")
		}
	}
	return nil
}

func validateReviewReworkSignalCompactPayloadV0(signal ReviewReworkSignalV0) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return reviewReworkSignalErrorV0(ErrReviewReworkSignalPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReplanProposalPayloadBytesV0 {
		return reviewReworkSignalErrorV0(ErrReviewReworkSignalPayloadInvalidoV0, "payload")
	}
	return nil
}

func reviewReworkSignalTextFieldsV0(signal ReviewReworkSignalV0) []string {
	values := []string{
		signal.SignalRef,
		signal.RunRef,
		signal.TaskRef,
		signal.ReviewRequestRef,
		signal.DeliveryRef,
		signal.ReviewResultRef,
		string(signal.ReviewStatus),
		string(signal.RequestedAction),
		signal.ReasonRef,
		signal.Summary,
	}
	return append(values, signal.EvidenceRefs...)
}

func reviewReworkSignalErrorV0(code string, field string) ReviewReworkSignalErrorV0 {
	return ReviewReworkSignalErrorV0{Code: code, Field: field}
}
