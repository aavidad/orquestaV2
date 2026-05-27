package orquestadomainwork

import "strings"

func ValidateDomainDocumentPlanV0(plan DomainDocumentPlanV0) []DomainWorkIssueV0 {
	plan = NormalizeDomainDocumentPlanV0(plan)
	var issues []DomainWorkIssueV0
	issues = append(issues, requiredDomainWorkRefV0(
		plan.PlanRef,
		"plan_ref",
		ErrDomainDocumentPlanRefRequiredV0,
		ErrDomainDocumentPlanRefInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkRefV0(
		plan.DomainRef,
		"domain_ref",
		ErrDomainWorkDomainRefRequiredV0,
		ErrDomainWorkDomainRefInvalidV0,
	)...)
	issues = append(issues, validateDocumentPlanWorkKindV0(plan.WorkKind)...)
	issues = append(issues, requiredDomainWorkRefV0(
		plan.DocumentKind,
		"document_kind",
		ErrDomainDocumentPlanDocumentKindRequiredV0,
		ErrDomainDocumentPlanRefInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		plan.Title,
		"title",
		ErrDomainDocumentPlanTitleRequiredV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		plan.Objective,
		"objective",
		ErrDomainDocumentPlanObjectiveRequiredV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		plan.LanguageCode,
		"language_code",
		ErrDomainDocumentPlanLanguageRequiredV0,
	)...)
	issues = append(issues, validateDomainDocumentPlanRangeV0(
		plan.EstimatedPagesMin,
		plan.EstimatedPagesMax,
		"estimated_pages",
	)...)
	issues = append(issues, validateDomainDocumentPlanRefsV0(plan.SourceRefs, "source_refs")...)
	issues = append(issues, validateDomainDocumentPlanRefsV0(plan.EvidenceRefs, "evidence_refs")...)
	issues = append(issues, validateDocumentPlanSectionsV0(plan.Sections)...)
	issues = append(issues, validateDocumentPlanUniqueRefsV0(
		documentPlanSectionRefsForValidationV0(plan.Sections),
		"sections.section_ref",
	)...)
	issues = append(issues, validateDocumentPlanVisualsV0(plan.Visuals)...)
	issues = append(issues, validateDocumentPlanUniqueRefsV0(
		documentPlanVisualRefsForValidationV0(plan.Visuals),
		"visuals.visual_ref",
	)...)
	issues = append(issues, validateDocumentPlanReviewsV0(plan.ReviewSteps)...)
	issues = append(issues, validateDocumentPlanUniqueRefsV0(
		documentPlanReviewRefsForValidationV0(plan.ReviewSteps),
		"review_steps.review_ref",
	)...)
	issues = append(issues, validateDocumentPlanDeliverablesV0(plan.Deliverables)...)
	issues = append(issues, validateDocumentPlanUniqueRefsV0(
		documentPlanDeliverableRefsForValidationV0(plan.Deliverables),
		"deliverables.deliverable_ref",
	)...)
	return issues
}

func validateDocumentPlanWorkKindV0(workKind string) []DomainWorkIssueV0 {
	workKind = strings.TrimSpace(workKind)
	if workKind == "" {
		return []DomainWorkIssueV0{{
			Code:  ErrDomainDocumentPlanWorkKindRequiredV0,
			Field: "work_kind",
		}}
	}
	if !isCompactDomainWorkRefV0(workKind) || !strings.HasPrefix(workKind, "plan_") {
		return []DomainWorkIssueV0{{
			Code:  ErrDomainDocumentPlanWorkKindInvalidV0,
			Field: "work_kind",
		}}
	}
	return nil
}

func validateDocumentPlanSectionsV0(
	sections []DomainDocumentPlanSectionV0,
) []DomainWorkIssueV0 {
	if len(sections) == 0 {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanSectionsRequiredV0, Field: "sections"}}
	}
	for _, section := range sections {
		if issues := validateDocumentPlanSectionV0(section); len(issues) > 0 {
			return issues
		}
	}
	return nil
}

func validateDocumentPlanSectionV0(
	section DomainDocumentPlanSectionV0,
) []DomainWorkIssueV0 {
	if issues := requiredDomainWorkRefV0(
		section.SectionRef,
		"sections.section_ref",
		ErrDomainDocumentPlanRefRequiredV0,
		ErrDomainDocumentPlanRefInvalidV0,
	); len(issues) > 0 {
		return issues
	}
	if section.ParentRef != "" && !isCompactDomainWorkRefV0(section.ParentRef) {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRefInvalidV0, Field: "sections.parent_ref"}}
	}
	if section.Order <= 0 {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRangeInvalidV0, Field: "sections.order"}}
	}
	if issues := requiredDomainWorkTextV0(
		section.Title,
		"sections.title",
		ErrDomainDocumentPlanTitleRequiredV0,
	); len(issues) > 0 {
		return issues
	}
	if issues := requiredDomainWorkTextV0(
		section.Objective,
		"sections.objective",
		ErrDomainDocumentPlanObjectiveRequiredV0,
	); len(issues) > 0 {
		return issues
	}
	if issues := requiredDomainWorkRefV0(
		section.WorkKind,
		"sections.work_kind",
		ErrDomainWorkWorkKindRequiredV0,
		ErrDomainWorkWorkKindInvalidV0,
	); len(issues) > 0 {
		return issues
	}
	if issues := validateDomainDocumentPlanRangeV0(
		section.TargetWordsMin,
		section.TargetWordsMax,
		"sections.target_words",
	); len(issues) > 0 {
		return issues
	}
	return validateDomainDocumentPlanRefsV0(section.DependsOn, "sections.depends_on")
}

func validateDocumentPlanVisualsV0(
	visuals []DomainDocumentPlanVisualV0,
) []DomainWorkIssueV0 {
	for _, visual := range visuals {
		if issues := validateDocumentPlanVisualV0(visual); len(issues) > 0 {
			return issues
		}
	}
	return nil
}

func validateDocumentPlanVisualV0(
	visual DomainDocumentPlanVisualV0,
) []DomainWorkIssueV0 {
	for _, item := range []struct {
		Value string
		Field string
	}{
		{visual.VisualRef, "visuals.visual_ref"},
		{visual.VisualType, "visuals.visual_type"},
		{visual.WorkKind, "visuals.work_kind"},
	} {
		if !isCompactDomainWorkRefV0(item.Value) {
			return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRefInvalidV0, Field: item.Field}}
		}
	}
	if visual.PlacementRef != "" && !isCompactDomainWorkRefV0(visual.PlacementRef) {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRefInvalidV0, Field: "visuals.placement_ref"}}
	}
	return requiredDomainWorkTextV0(
		visual.Objective,
		"visuals.objective",
		ErrDomainDocumentPlanObjectiveRequiredV0,
	)
}

func validateDocumentPlanReviewsV0(
	reviews []DomainDocumentPlanReviewV0,
) []DomainWorkIssueV0 {
	for _, review := range reviews {
		if issues := validateDocumentPlanReviewV0(review); len(issues) > 0 {
			return issues
		}
	}
	return nil
}

func validateDocumentPlanReviewV0(
	review DomainDocumentPlanReviewV0,
) []DomainWorkIssueV0 {
	if !isCompactDomainWorkRefV0(review.ReviewRef) {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRefInvalidV0, Field: "review_steps.review_ref"}}
	}
	if review.Order <= 0 {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRangeInvalidV0, Field: "review_steps.order"}}
	}
	if !isCompactDomainWorkRefV0(review.WorkKind) {
		return []DomainWorkIssueV0{{Code: ErrDomainWorkWorkKindInvalidV0, Field: "review_steps.work_kind"}}
	}
	return requiredDomainWorkTextV0(
		review.Objective,
		"review_steps.objective",
		ErrDomainDocumentPlanObjectiveRequiredV0,
	)
}

func validateDocumentPlanDeliverablesV0(
	deliverables []DomainDocumentPlanDeliverableV0,
) []DomainWorkIssueV0 {
	if len(deliverables) == 0 {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanDeliverablesRequiredV0, Field: "deliverables"}}
	}
	for _, deliverable := range deliverables {
		for _, item := range []struct {
			Value string
			Field string
		}{
			{deliverable.DeliverableRef, "deliverables.deliverable_ref"},
			{deliverable.ArtifactType, "deliverables.artifact_type"},
		} {
			if !isCompactDomainWorkRefV0(item.Value) {
				return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRefInvalidV0, Field: item.Field}}
			}
		}
		if strings.TrimSpace(deliverable.Title) == "" {
			return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanTitleRequiredV0, Field: "deliverables.title"}}
		}
	}
	return nil
}

func validateDomainDocumentPlanRangeV0(min int, max int, field string) []DomainWorkIssueV0 {
	if min < 0 || max < 0 || (min > 0 && max > 0 && min > max) {
		return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRangeInvalidV0, Field: field}}
	}
	return nil
}

func validateDomainDocumentPlanRefsV0(values []string, field string) []DomainWorkIssueV0 {
	for _, value := range values {
		if !isCompactDomainWorkRefV0(value) {
			return []DomainWorkIssueV0{{Code: ErrDomainDocumentPlanRefInvalidV0, Field: field}}
		}
	}
	return nil
}
