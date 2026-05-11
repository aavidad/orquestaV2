package orquestacorereplanner

import "strings"

func NewReplanDecisionV0(decision ReplanDecisionV0) (ReplanDecisionV0, error) {
	normalized := NormalizeReplanDecisionV0(decision)
	if err := ValidateReplanDecisionV0(normalized); err != nil {
		return ReplanDecisionV0{}, err
	}
	return normalized, nil
}

func NormalizeReplanDecisionV0(decision ReplanDecisionV0) ReplanDecisionV0 {
	return ReplanDecisionV0{
		ReplanRef:      strings.TrimSpace(decision.ReplanRef),
		RunRef:         strings.TrimSpace(decision.RunRef),
		TaskRef:        strings.TrimSpace(decision.TaskRef),
		SourceRef:      strings.TrimSpace(decision.SourceRef),
		AcceptedAction: ReplanRecommendedActionV0(strings.TrimSpace(string(decision.AcceptedAction))),
		FollowupRefs:   normalizeReplanProposalStringsV0(decision.FollowupRefs),
		Summary:        strings.TrimSpace(decision.Summary),
		EvidenceRefs:   normalizeReplanProposalStringsV0(decision.EvidenceRefs),
	}
}

func ValidateReplanDecisionV0(decision ReplanDecisionV0) error {
	if err := validateReplanDecisionRequiredFieldsV0(decision); err != nil {
		return err
	}
	if err := validateReplanDecisionActionV0(decision.AcceptedAction); err != nil {
		return err
	}
	if err := validateReplanDecisionCollectionsV0(decision); err != nil {
		return err
	}
	if replanProposalHasLongStringV0(replanDecisionTextFieldsV0(decision)) {
		return replanDecisionErrorV0(ErrReplanDecisionPayloadInvalidoV0, "payload")
	}
	if replanProposalHasForbiddenDetailsV0(replanDecisionTextFieldsV0(decision)) {
		return replanDecisionErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateReplanDecisionCompactPayloadV0(decision)
}
