package orquestadocumentplanexpander

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

const (
	DomainDocumentPlanExpansionSchemaVersionV0 = "domain_document_plan_expansion.v0"
	DomainDocumentPlanExpansionRequestedByV0   = "orquesta-document-plan-expander"
)

type DomainDocumentPlanExpansionRequestV0 struct {
	Plan          orquestadomainwork.DomainDocumentPlanV0
	CorrelationID string
	RequestedBy   string
	InterfaceRefs []string
	InputFields   []orquestadomainwork.DomainWorkFieldV0
	ExternalRefs  []orquestadomainwork.DomainWorkExternalRefV0
	EvidenceRefs  []string
}

type DomainDocumentPlanExpansionResultV0 struct {
	SchemaVersion string                                      `json:"schema_version"`
	Jobs          []orquestadomainwork.DomainWorkJobRequestV0 `json:"jobs,omitempty"`
	Issues        []orquestadomainwork.DomainWorkIssueV0      `json:"issues,omitempty"`
}

func ExpandDomainDocumentPlanV0(
	request DomainDocumentPlanExpansionRequestV0,
) DomainDocumentPlanExpansionResultV0 {
	result := DomainDocumentPlanExpansionResultV0{
		SchemaVersion: DomainDocumentPlanExpansionSchemaVersionV0,
	}
	if issues := validateRawDocumentPlanExpansionRefsV0(request.Plan); len(issues) > 0 {
		result.Issues = issues
		return result
	}
	plan := orquestadomainwork.NormalizeDomainDocumentPlanV0(request.Plan)
	if issues := orquestadomainwork.ValidateDomainDocumentPlanV0(plan); len(issues) > 0 {
		result.Issues = issues
		return result
	}
	jobs := expandDocumentPlanJobsV0(request, plan)
	var issues []orquestadomainwork.DomainWorkIssueV0
	normalized := make([]orquestadomainwork.DomainWorkJobRequestV0, 0, len(jobs))
	for _, job := range jobs {
		job = orquestadomainwork.NormalizeDomainWorkJobRequestV0(job)
		if jobIssues := orquestadomainwork.ValidateDomainWorkJobRequestV0(job); len(jobIssues) > 0 {
			issues = append(issues, jobIssues...)
			continue
		}
		normalized = append(normalized, job)
	}
	if len(issues) > 0 {
		result.Issues = issues
		return result
	}
	result.Jobs = normalized
	return result
}

func documentPlanVisualJobV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
	visual orquestadomainwork.DomainDocumentPlanVisualV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	artifactType := ExpectedArtifactTypeForDocumentPlanWorkKindV0(visual.WorkKind)
	return documentPlanBaseJobV0(
		request,
		plan,
		"visual",
		visual.VisualRef,
		visual.WorkKind,
		visual.Objective,
		artifactType,
		documentPlanRefsV0(plan.PlanRef, plan.ScopeRef, visual.VisualRef, visual.PlacementRef),
		documentPlanRefsV0(append(plan.SourceRefs, visual.SourceRefs...)...),
		documentPlanVisualFieldsV0(visual, artifactType),
		append(documentPlanRefsV0(plan.QualityCriteria...), visual.AcceptanceCriteria...),
		[]orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "visual_ref", Ref: visual.VisualRef},
		},
	)
}

func documentPlanReviewJobV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
	review orquestadomainwork.DomainDocumentPlanReviewV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	artifactType := ExpectedArtifactTypeForDocumentPlanWorkKindV0(review.WorkKind)
	return documentPlanBaseJobV0(
		request,
		plan,
		"review",
		review.ReviewRef,
		review.WorkKind,
		review.Objective,
		artifactType,
		documentPlanRefsV0(plan.PlanRef, plan.ScopeRef, review.ReviewRef),
		documentPlanRefsV0(plan.SourceRefs...),
		documentPlanReviewFieldsV0(plan, review, artifactType),
		append(documentPlanRefsV0(plan.QualityCriteria...), review.AcceptanceCriteria...),
		[]orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "review_ref", Ref: review.ReviewRef},
		},
	)
}

func documentPlanBaseJobV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
	itemKind string,
	itemRef string,
	workKind string,
	objective string,
	artifactType string,
	workRefs []string,
	inputRefs []string,
	itemFields []orquestadomainwork.DomainWorkFieldV0,
	acceptanceCriteria []string,
	externalRefs []orquestadomainwork.DomainWorkExternalRefV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	jobRef := documentPlanJobRefV0(plan.PlanRef, itemKind, itemRef)
	return orquestadomainwork.DomainWorkJobRequestV0{
		RequestID:          jobRef,
		CorrelationID:      documentPlanCorrelationIDV0(request, plan),
		IdempotencyKey:     jobRef,
		RequestedBy:        documentPlanRequestedByV0(request),
		DomainRef:          plan.DomainRef,
		InterfaceRefs:      documentPlanRefsV0(request.InterfaceRefs...),
		WorkKind:           workKind,
		WorkRefs:           documentPlanRefsV0(workRefs...),
		Objective:          objective,
		InputFields:        appendDocumentPlanFieldsV0(documentPlanCommonFieldsV0(request, plan), itemFields...),
		InputRefs:          documentPlanRefsV0(inputRefs...),
		Constraints:        documentPlanRefsV0(plan.Constraints...),
		AcceptanceCriteria: documentPlanRefsV0(acceptanceCriteria...),
		ExternalRefs: appendDocumentPlanExternalRefsV0(
			documentPlanCommonExternalRefsV0(request, plan),
			externalRefs...,
		),
		EvidenceRefs: documentPlanRefsV0(append(request.EvidenceRefs, plan.EvidenceRefs...)...),
	}
}

func documentPlanCommonFieldsV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	fields := appendDocumentPlanFieldsV0(nil,
		orquestadomainwork.DomainWorkFieldV0{Name: "plan_ref", Value: plan.PlanRef},
		orquestadomainwork.DomainWorkFieldV0{Name: "document_kind", Value: plan.DocumentKind},
		orquestadomainwork.DomainWorkFieldV0{Name: "scope_ref", Value: plan.ScopeRef},
		orquestadomainwork.DomainWorkFieldV0{Name: "language_code", Value: plan.LanguageCode},
		orquestadomainwork.DomainWorkFieldV0{Name: "plan_title", Value: plan.Title},
		orquestadomainwork.DomainWorkFieldV0{Name: "plan_objective", Value: plan.Objective},
		orquestadomainwork.DomainWorkFieldV0{Name: "target_audience", Value: plan.TargetAudience},
		orquestadomainwork.DomainWorkFieldV0{Name: "quality_criteria", Values: plan.QualityCriteria},
	)
	return appendDocumentPlanFieldsV0(fields, request.InputFields...)
}

func documentPlanSectionFieldsV0(
	section orquestadomainwork.DomainDocumentPlanSectionV0,
	artifactType string,
) []orquestadomainwork.DomainWorkFieldV0 {
	return appendDocumentPlanFieldsV0(nil,
		orquestadomainwork.DomainWorkFieldV0{Name: "expected_artifact_type", Value: artifactType},
		orquestadomainwork.DomainWorkFieldV0{Name: "section_ref", Value: section.SectionRef},
		orquestadomainwork.DomainWorkFieldV0{Name: "parent_ref", Value: section.ParentRef},
		orquestadomainwork.DomainWorkFieldV0{Name: "section_order", Value: positiveIntStringV0(section.Order)},
		orquestadomainwork.DomainWorkFieldV0{Name: "section_title", Value: section.Title},
		orquestadomainwork.DomainWorkFieldV0{Name: "target_words_min", Value: positiveIntStringV0(section.TargetWordsMin)},
		orquestadomainwork.DomainWorkFieldV0{Name: "target_words_max", Value: positiveIntStringV0(section.TargetWordsMax)},
		orquestadomainwork.DomainWorkFieldV0{Name: "required_elements", Values: section.RequiredElements},
		orquestadomainwork.DomainWorkFieldV0{Name: "depends_on", Values: section.DependsOn},
	)
}

func documentPlanVisualFieldsV0(
	visual orquestadomainwork.DomainDocumentPlanVisualV0,
	artifactType string,
) []orquestadomainwork.DomainWorkFieldV0 {
	return appendDocumentPlanFieldsV0(nil,
		orquestadomainwork.DomainWorkFieldV0{Name: "expected_artifact_type", Value: artifactType},
		orquestadomainwork.DomainWorkFieldV0{Name: "visual_ref", Value: visual.VisualRef},
		orquestadomainwork.DomainWorkFieldV0{Name: "visual_type", Value: visual.VisualType},
		orquestadomainwork.DomainWorkFieldV0{Name: "placement_ref", Value: visual.PlacementRef},
	)
}

func documentPlanReviewFieldsV0(
	plan orquestadomainwork.DomainDocumentPlanV0,
	review orquestadomainwork.DomainDocumentPlanReviewV0,
	artifactType string,
) []orquestadomainwork.DomainWorkFieldV0 {
	return appendDocumentPlanFieldsV0(nil,
		orquestadomainwork.DomainWorkFieldV0{Name: "expected_artifact_type", Value: artifactType},
		orquestadomainwork.DomainWorkFieldV0{Name: "review_ref", Value: review.ReviewRef},
		orquestadomainwork.DomainWorkFieldV0{Name: "review_order", Value: positiveIntStringV0(review.Order)},
		orquestadomainwork.DomainWorkFieldV0{Name: "section_refs", Values: documentPlanSectionRefsV0(plan.Sections)},
		orquestadomainwork.DomainWorkFieldV0{Name: "visual_refs", Values: documentPlanVisualRefsV0(plan.Visuals)},
		orquestadomainwork.DomainWorkFieldV0{Name: "deliverable_refs", Values: documentPlanDeliverableRefsV0(plan.Deliverables)},
		orquestadomainwork.DomainWorkFieldV0{Name: "deliverable_artifact_types", Values: documentPlanDeliverableArtifactTypesV0(plan.Deliverables)},
	)
}
