package orquestadomainwork

import "strings"

func NormalizeDomainDocumentPlanV0(
	plan DomainDocumentPlanV0,
) DomainDocumentPlanV0 {
	plan.SchemaVersion = defaultDomainWorkSchemaV0(plan.SchemaVersion, DomainDocumentPlanSchemaV0)
	plan.PlanRef = strings.TrimSpace(plan.PlanRef)
	plan.DomainRef = strings.TrimSpace(plan.DomainRef)
	plan.WorkKind = strings.TrimSpace(plan.WorkKind)
	plan.DocumentKind = strings.TrimSpace(plan.DocumentKind)
	plan.ScopeRef = strings.TrimSpace(plan.ScopeRef)
	plan.LanguageCode = strings.TrimSpace(plan.LanguageCode)
	plan.Title = strings.TrimSpace(plan.Title)
	plan.Objective = strings.TrimSpace(plan.Objective)
	plan.TargetAudience = strings.TrimSpace(plan.TargetAudience)
	plan.Sections = compactDocumentPlanSectionsV0(plan.Sections)
	plan.Visuals = compactDocumentPlanVisualsV0(plan.Visuals)
	plan.ReviewSteps = compactDocumentPlanReviewsV0(plan.ReviewSteps)
	plan.Deliverables = compactDocumentPlanDeliverablesV0(plan.Deliverables)
	plan.QualityCriteria = compactDomainWorkStringsV0(plan.QualityCriteria)
	plan.Constraints = compactDomainWorkStringsV0(plan.Constraints)
	plan.SourceRefs = compactDocumentPlanRefsV0(plan.SourceRefs)
	plan.EvidenceRefs = compactDocumentPlanRefsV0(plan.EvidenceRefs)
	return plan
}

func compactDocumentPlanSectionsV0(
	values []DomainDocumentPlanSectionV0,
) []DomainDocumentPlanSectionV0 {
	out := make([]DomainDocumentPlanSectionV0, 0, len(values))
	for _, value := range values {
		section := DomainDocumentPlanSectionV0{
			SectionRef:         strings.TrimSpace(value.SectionRef),
			ParentRef:          strings.TrimSpace(value.ParentRef),
			Order:              value.Order,
			Title:              strings.TrimSpace(value.Title),
			Objective:          strings.TrimSpace(value.Objective),
			WorkKind:           documentPlanSectionWorkKindV0(value.WorkKind),
			DependsOn:          compactDocumentPlanRefsV0(value.DependsOn),
			TargetWordsMin:     value.TargetWordsMin,
			TargetWordsMax:     value.TargetWordsMax,
			RequiredElements:   compactDomainWorkStringsV0(value.RequiredElements),
			AcceptanceCriteria: compactDomainWorkStringsV0(value.AcceptanceCriteria),
			SourceRefs:         compactDocumentPlanRefsV0(value.SourceRefs),
		}
		if section.SectionRef == "" && section.Title == "" && section.Objective == "" {
			continue
		}
		out = append(out, section)
	}
	if out == nil {
		return []DomainDocumentPlanSectionV0{}
	}
	return out
}

func compactDocumentPlanVisualsV0(
	values []DomainDocumentPlanVisualV0,
) []DomainDocumentPlanVisualV0 {
	out := make([]DomainDocumentPlanVisualV0, 0, len(values))
	for _, value := range values {
		visual := DomainDocumentPlanVisualV0{
			VisualRef:          strings.TrimSpace(value.VisualRef),
			VisualType:         strings.TrimSpace(value.VisualType),
			PlacementRef:       strings.TrimSpace(value.PlacementRef),
			Objective:          strings.TrimSpace(value.Objective),
			WorkKind:           documentPlanVisualWorkKindV0(value.WorkKind),
			AcceptanceCriteria: compactDomainWorkStringsV0(value.AcceptanceCriteria),
			SourceRefs:         compactDocumentPlanRefsV0(value.SourceRefs),
		}
		if visual.VisualRef == "" && visual.Objective == "" {
			continue
		}
		out = append(out, visual)
	}
	if out == nil {
		return []DomainDocumentPlanVisualV0{}
	}
	return out
}

func compactDocumentPlanReviewsV0(
	values []DomainDocumentPlanReviewV0,
) []DomainDocumentPlanReviewV0 {
	out := make([]DomainDocumentPlanReviewV0, 0, len(values))
	for _, value := range values {
		review := DomainDocumentPlanReviewV0{
			ReviewRef:          strings.TrimSpace(value.ReviewRef),
			Order:              value.Order,
			WorkKind:           documentPlanReviewWorkKindV0(value.WorkKind),
			Objective:          strings.TrimSpace(value.Objective),
			AcceptanceCriteria: compactDomainWorkStringsV0(value.AcceptanceCriteria),
		}
		if review.ReviewRef == "" && review.Objective == "" {
			continue
		}
		out = append(out, review)
	}
	if out == nil {
		return []DomainDocumentPlanReviewV0{}
	}
	return out
}

func compactDocumentPlanDeliverablesV0(
	values []DomainDocumentPlanDeliverableV0,
) []DomainDocumentPlanDeliverableV0 {
	out := make([]DomainDocumentPlanDeliverableV0, 0, len(values))
	for _, value := range values {
		deliverable := DomainDocumentPlanDeliverableV0{
			DeliverableRef: strings.TrimSpace(value.DeliverableRef),
			ArtifactType:   strings.TrimSpace(value.ArtifactType),
			Title:          strings.TrimSpace(value.Title),
			Required:       value.Required,
		}
		if deliverable.DeliverableRef == "" && deliverable.ArtifactType == "" {
			continue
		}
		out = append(out, deliverable)
	}
	if out == nil {
		return []DomainDocumentPlanDeliverableV0{}
	}
	return out
}
