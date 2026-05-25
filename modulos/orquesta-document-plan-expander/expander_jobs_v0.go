package orquestadocumentplanexpander

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

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
