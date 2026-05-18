package orquestadocumentplanexpander

import (
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

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
	plan := orquestadomainwork.NormalizeDomainDocumentPlanV0(request.Plan)
	result := DomainDocumentPlanExpansionResultV0{
		SchemaVersion: DomainDocumentPlanExpansionSchemaVersionV0,
	}
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

func ExpectedArtifactTypeForDocumentPlanWorkKindV0(workKind string) string {
	switch strings.TrimSpace(workKind) {
	case "draft_content_block", "generate_block", "generate_program_topic_draft":
		return "content_block"
	case "generate_visual_asset":
		return "visual_asset"
	case "review_legal", "review_pedagogical", "review_quality", "validate_topic":
		return "block_revision"
	case "research_sources", "download_source", "verify_sources":
		return "source"
	case "summarize_block", "summarize_chapter", "summarize_topic", "create_exam_outline":
		return "topic_summary"
	case "expand_topic_from_summary":
		return "topic_expansion_package"
	case "plan_documento", "plan_tema", "plan_temario":
		return orquestadomainwork.DomainDocumentPlanArtifactTypeV0
	case "assemble_topic":
		return "assembled_topic"
	default:
		return "work_delivery"
	}
}

func expandDocumentPlanJobsV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
) []orquestadomainwork.DomainWorkJobRequestV0 {
	jobs := make([]orquestadomainwork.DomainWorkJobRequestV0, 0,
		len(plan.Sections)+len(plan.Visuals)+len(plan.ReviewSteps))
	for _, section := range plan.Sections {
		jobs = append(jobs, documentPlanSectionJobV0(request, plan, section))
	}
	for _, visual := range plan.Visuals {
		jobs = append(jobs, documentPlanVisualJobV0(request, plan, visual))
	}
	for _, review := range plan.ReviewSteps {
		jobs = append(jobs, documentPlanReviewJobV0(request, plan, review))
	}
	return jobs
}

func documentPlanSectionJobV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
	section orquestadomainwork.DomainDocumentPlanSectionV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	artifactType := ExpectedArtifactTypeForDocumentPlanWorkKindV0(section.WorkKind)
	return documentPlanBaseJobV0(
		request,
		plan,
		"section",
		section.SectionRef,
		section.WorkKind,
		section.Objective,
		artifactType,
		documentPlanRefsV0(plan.PlanRef, plan.ScopeRef, section.SectionRef, section.ParentRef),
		documentPlanRefsV0(append(plan.SourceRefs, section.SourceRefs...)...),
		documentPlanSectionFieldsV0(section, artifactType),
		append(documentPlanRefsV0(plan.QualityCriteria...), section.AcceptanceCriteria...),
		[]orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "section_ref", Ref: section.SectionRef},
		},
	)
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

func documentPlanJobRefV0(planRef string, itemKind string, itemRef string) string {
	return "docplan-job-" + strings.TrimSpace(planRef) + "-" +
		strings.TrimSpace(itemKind) + "-" + strings.TrimSpace(itemRef)
}

func documentPlanCorrelationIDV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
) string {
	if value := strings.TrimSpace(request.CorrelationID); value != "" {
		return value
	}
	return "docplan-expansion-" + plan.PlanRef
}

func documentPlanRequestedByV0(request DomainDocumentPlanExpansionRequestV0) string {
	if value := strings.TrimSpace(request.RequestedBy); value != "" {
		return value
	}
	return DomainDocumentPlanExpansionRequestedByV0
}

func positiveIntStringV0(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func appendDocumentPlanFieldsV0(
	base []orquestadomainwork.DomainWorkFieldV0,
	values ...orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	out := append([]orquestadomainwork.DomainWorkFieldV0(nil), base...)
	for _, value := range values {
		field := orquestadomainwork.DomainWorkFieldV0{
			Name:      strings.TrimSpace(value.Name),
			Value:     strings.TrimSpace(value.Value),
			Values:    documentPlanRefsV0(value.Values...),
			ValueJSON: append([]byte(nil), value.ValueJSON...),
		}
		if field.Name == "" ||
			(field.Value == "" && len(field.Values) == 0 && len(field.ValueJSON) == 0) {
			continue
		}
		out = append(out, field)
	}
	return out
}

func documentPlanCommonExternalRefsV0(
	request DomainDocumentPlanExpansionRequestV0,
	plan orquestadomainwork.DomainDocumentPlanV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	refs := []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "plan_ref", Ref: plan.PlanRef},
		{Kind: "document_kind", Ref: plan.DocumentKind},
	}
	if plan.ScopeRef != "" {
		refs = append(refs, orquestadomainwork.DomainWorkExternalRefV0{
			Kind: "scope_ref",
			Ref:  plan.ScopeRef,
		})
	}
	return appendDocumentPlanExternalRefsV0(refs, request.ExternalRefs...)
}

func appendDocumentPlanExternalRefsV0(
	base []orquestadomainwork.DomainWorkExternalRefV0,
	values ...orquestadomainwork.DomainWorkExternalRefV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	seen := map[string]struct{}{}
	out := make([]orquestadomainwork.DomainWorkExternalRefV0, 0, len(base)+len(values))
	for _, value := range append(append([]orquestadomainwork.DomainWorkExternalRefV0(nil), base...), values...) {
		ref := orquestadomainwork.DomainWorkExternalRefV0{
			Kind: strings.TrimSpace(value.Kind),
			Ref:  strings.TrimSpace(value.Ref),
		}
		if ref.Kind == "" || ref.Ref == "" {
			continue
		}
		key := ref.Kind + "\x00" + ref.Ref
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func documentPlanRefsV0(values ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func documentPlanSectionRefsV0(
	sections []orquestadomainwork.DomainDocumentPlanSectionV0,
) []string {
	refs := make([]string, 0, len(sections))
	for _, section := range sections {
		refs = append(refs, section.SectionRef)
	}
	return documentPlanRefsV0(refs...)
}

func documentPlanVisualRefsV0(
	visuals []orquestadomainwork.DomainDocumentPlanVisualV0,
) []string {
	refs := make([]string, 0, len(visuals))
	for _, visual := range visuals {
		refs = append(refs, visual.VisualRef)
	}
	return documentPlanRefsV0(refs...)
}

func documentPlanDeliverableRefsV0(
	deliverables []orquestadomainwork.DomainDocumentPlanDeliverableV0,
) []string {
	refs := make([]string, 0, len(deliverables))
	for _, deliverable := range deliverables {
		refs = append(refs, deliverable.DeliverableRef)
	}
	return documentPlanRefsV0(refs...)
}

func documentPlanDeliverableArtifactTypesV0(
	deliverables []orquestadomainwork.DomainDocumentPlanDeliverableV0,
) []string {
	refs := make([]string, 0, len(deliverables))
	for _, deliverable := range deliverables {
		refs = append(refs, deliverable.ArtifactType)
	}
	return documentPlanRefsV0(refs...)
}
