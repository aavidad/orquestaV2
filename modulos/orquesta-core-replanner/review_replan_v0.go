package orquestacorereplanner

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func NewReviewReworkSignalV0(signal ReviewReworkSignalV0) (ReviewReworkSignalV0, error) {
	normalized := NormalizeReviewReworkSignalV0(signal)
	if err := ValidateReviewReworkSignalV0(normalized); err != nil {
		return ReviewReworkSignalV0{}, err
	}
	return normalized, nil
}

func NormalizeReviewReworkSignalV0(signal ReviewReworkSignalV0) ReviewReworkSignalV0 {
	return ReviewReworkSignalV0{
		SignalRef:        strings.TrimSpace(signal.SignalRef),
		RunRef:           strings.TrimSpace(signal.RunRef),
		TaskRef:          strings.TrimSpace(signal.TaskRef),
		ReviewRequestRef: strings.TrimSpace(signal.ReviewRequestRef),
		DeliveryRef:      strings.TrimSpace(signal.DeliveryRef),
		ReviewResultRef:  strings.TrimSpace(signal.ReviewResultRef),
		ReviewStatus:     orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(string(signal.ReviewStatus))),
		RequestedAction:  ReplanRecommendedActionV0(strings.TrimSpace(string(signal.RequestedAction))),
		ReasonRef:        strings.TrimSpace(signal.ReasonRef),
		Summary:          strings.TrimSpace(signal.Summary),
		EvidenceRefs:     normalizeReplanProposalStringsV0(signal.EvidenceRefs),
	}
}

func ReviewResultToReplanProposalV0(input ReviewResultReplanInputV0) (*ReplanProposalV0, error) {
	result, err := orquestacoreworkflow.NewReviewResultV0(input.ReviewResult)
	if err != nil {
		return nil, err
	}
	if result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		return nil, nil
	}

	signal, err := ReviewReworkSignalFromReviewResultV0(input, result)
	if err != nil {
		return nil, err
	}

	proposal, err := ReplanProposalFromReviewReworkSignalV0(input.ReplanRef, signal)
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

func ReviewReworkSignalFromReviewResultV0(input ReviewResultReplanInputV0, result orquestacoreworkflow.ReviewResultV0) (ReviewReworkSignalV0, error) {
	normalizedResult, err := orquestacoreworkflow.NewReviewResultV0(result)
	if err != nil {
		return ReviewReworkSignalV0{}, err
	}
	result = normalizedResult

	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		summary = result.Summary
	}

	signal := ReviewReworkSignalV0{
		SignalRef:        input.SignalRef,
		RunRef:           input.RunRef,
		TaskRef:          input.TaskRef,
		ReviewRequestRef: result.ReviewRequestID,
		DeliveryRef:      result.DeliveryRef,
		ReviewResultRef:  result.ReviewResultRef,
		ReviewStatus:     result.Status,
		RequestedAction:  input.RequestedAction,
		ReasonRef:        input.ReasonRef,
		Summary:          summary,
		EvidenceRefs:     reviewReworkEvidenceRefsV0(result.EvidenceRefs, input.EvidenceRefs),
	}
	return NewReviewReworkSignalV0(signal)
}

func ReplanProposalFromReviewReworkSignalV0(replanRef string, signal ReviewReworkSignalV0) (ReplanProposalV0, error) {
	normalized, err := NewReviewReworkSignalV0(signal)
	if err != nil {
		return ReplanProposalV0{}, err
	}

	proposal := ReplanProposalV0{
		ReplanRef:         replanRef,
		RunRef:            normalized.RunRef,
		TaskRef:           normalized.TaskRef,
		SourceRef:         normalized.ReviewResultRef,
		ReasonCode:        reviewReworkReasonCodeV0(normalized),
		RecommendedAction: normalized.RequestedAction,
		Summary:           normalized.Summary,
		EvidenceRefs:      reviewReworkEvidenceRefsV0([]string{normalized.SignalRef}, normalized.EvidenceRefs),
	}
	return NewReplanProposalV0(proposal)
}

func reviewReworkReasonCodeV0(signal ReviewReworkSignalV0) string {
	if strings.TrimSpace(signal.ReasonRef) != "" {
		return signal.ReasonRef
	}
	return "review_" + string(signal.ReviewStatus)
}

func reviewReworkEvidenceRefsV0(groups ...[]string) []string {
	seen := map[string]bool{}
	var refs []string
	for _, group := range groups {
		for _, raw := range group {
			ref := strings.TrimSpace(raw)
			if ref == "" || seen[ref] {
				continue
			}
			seen[ref] = true
			refs = append(refs, ref)
		}
	}
	return refs
}
