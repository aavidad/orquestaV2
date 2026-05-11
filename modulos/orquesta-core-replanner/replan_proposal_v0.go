package orquestacorereplanner

import "strings"

func NewReplanProposalV0(proposal ReplanProposalV0) (ReplanProposalV0, error) {
	normalized := NormalizeReplanProposalV0(proposal)
	if err := ValidateReplanProposalV0(normalized); err != nil {
		return ReplanProposalV0{}, err
	}
	return normalized, nil
}

func NormalizeReplanProposalV0(proposal ReplanProposalV0) ReplanProposalV0 {
	return ReplanProposalV0{
		ReplanRef:          strings.TrimSpace(proposal.ReplanRef),
		RunRef:             strings.TrimSpace(proposal.RunRef),
		TaskRef:            strings.TrimSpace(proposal.TaskRef),
		SourceRef:          strings.TrimSpace(proposal.SourceRef),
		ReasonCode:         strings.TrimSpace(proposal.ReasonCode),
		RecommendedAction:  ReplanRecommendedActionV0(strings.TrimSpace(string(proposal.RecommendedAction))),
		CapacityRequestRef: strings.TrimSpace(proposal.CapacityRequestRef),
		ReplacementRole:    strings.TrimSpace(proposal.ReplacementRole),
		Summary:            strings.TrimSpace(proposal.Summary),
		EvidenceRefs:       normalizeReplanProposalStringsV0(proposal.EvidenceRefs),
	}
}

func ValidateReplanProposalV0(proposal ReplanProposalV0) error {
	if err := validateReplanProposalRequiredFieldsV0(proposal); err != nil {
		return err
	}
	if err := ValidateReplanRecommendedActionV0(proposal.RecommendedAction); err != nil {
		return err
	}
	if err := validateReplanProposalActionFieldsV0(proposal); err != nil {
		return err
	}
	if err := validateReplanProposalCollectionsV0(proposal); err != nil {
		return err
	}
	if replanProposalHasLongStringV0(replanProposalTextFieldsV0(proposal)) {
		return replanProposalErrorV0(ErrReplanProposalPayloadInvalidoV0, "payload")
	}
	if replanProposalHasForbiddenDetailsV0(replanProposalTextFieldsV0(proposal)) {
		return replanProposalErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateReplanProposalCompactPayloadV0(proposal)
}

func IsSupportedReplanRecommendedActionV0(action ReplanRecommendedActionV0) bool {
	switch ReplanRecommendedActionV0(strings.TrimSpace(string(action))) {
	case ReplanActionSplitTaskV0,
		ReplanActionRetryTaskV0,
		ReplanActionReplaceAgentV0,
		ReplanActionEscalateCapacityV0,
		ReplanActionAskDirectorV0,
		ReplanActionAbortTaskV0:
		return true
	default:
		return false
	}
}

func ValidateReplanRecommendedActionV0(action ReplanRecommendedActionV0) error {
	if !IsSupportedReplanRecommendedActionV0(action) {
		return replanProposalErrorV0(ErrReplanActionNoSoportadaV0, "recommended_action")
	}
	return nil
}
