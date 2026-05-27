package orquestadomainwork

import "strings"

func validateDocumentPlanUniqueRefsV0(values []string, field string) []DomainWorkIssueV0 {
	seen := map[string]struct{}{}
	for _, value := range values {
		ref := strings.TrimSpace(value)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			return []DomainWorkIssueV0{{
				Code:  ErrDomainDocumentPlanRefDuplicateV0,
				Field: field,
			}}
		}
		seen[ref] = struct{}{}
	}
	return nil
}

func documentPlanSectionRefsForValidationV0(
	sections []DomainDocumentPlanSectionV0,
) []string {
	refs := make([]string, 0, len(sections))
	for _, section := range sections {
		refs = append(refs, section.SectionRef)
	}
	return refs
}

func documentPlanVisualRefsForValidationV0(
	visuals []DomainDocumentPlanVisualV0,
) []string {
	refs := make([]string, 0, len(visuals))
	for _, visual := range visuals {
		refs = append(refs, visual.VisualRef)
	}
	return refs
}

func documentPlanReviewRefsForValidationV0(
	reviews []DomainDocumentPlanReviewV0,
) []string {
	refs := make([]string, 0, len(reviews))
	for _, review := range reviews {
		refs = append(refs, review.ReviewRef)
	}
	return refs
}

func documentPlanDeliverableRefsForValidationV0(
	deliverables []DomainDocumentPlanDeliverableV0,
) []string {
	refs := make([]string, 0, len(deliverables))
	for _, deliverable := range deliverables {
		refs = append(refs, deliverable.DeliverableRef)
	}
	return refs
}
