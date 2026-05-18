package orquestaappcodexstack

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func canonicalDomainWorkDeliveryPayloadBodyV0(
	artifactType string,
	body string,
	input DomainWorkArtifactSubmissionBuildInputV0,
) string {
	if strings.TrimSpace(artifactType) == "content_block" {
		canonical, ok := canonicalDomainWorkContentBlockPayloadJSONV0(body)
		if ok {
			return canonical
		}
		return body
	}
	if strings.TrimSpace(artifactType) != orquestadomainwork.DomainDocumentPlanArtifactTypeV0 {
		return body
	}
	canonical, ok := orquestadomainwork.CanonicalDomainDocumentPlanPayloadJSONV0(
		body,
		documentPlanPayloadDefaultsFromInputV0(input),
	)
	if !ok {
		return body
	}
	return canonical
}

func documentPlanPayloadDefaultsFromInputV0(
	input DomainWorkArtifactSubmissionBuildInputV0,
) orquestadomainwork.DomainDocumentPlanPayloadDefaultsV0 {
	work := input.Record.Request.ExternalWork
	if work == nil {
		return orquestadomainwork.DomainDocumentPlanPayloadDefaultsV0{}
	}
	fields := copyDomainWorkFieldsForContextV0(work.InputFields)
	return orquestadomainwork.DomainDocumentPlanPayloadDefaultsV0{
		PlanRef:           "plan-" + strings.TrimSpace(work.JobRef),
		DomainRef:         strings.TrimSpace(work.ProjectRef),
		WorkKind:          strings.TrimSpace(work.WorkKind),
		DocumentKind:      firstNonEmptyDomainPlanDefaultV0(domainWorkFieldStringValueV0(fields, "document_kind"), "document"),
		ScopeRef:          firstNonEmptyDomainPlanDefaultV0(domainWorkFieldStringValueV0(fields, "topic_id"), domainWorkFieldStringValueV0(fields, "program_id")),
		LanguageCode:      firstNonEmptyDomainPlanDefaultV0(domainWorkFieldStringValueV0(fields, "language_code"), "es"),
		Title:             firstNonEmptyDomainPlanDefaultV0(domainWorkFieldStringValueV0(fields, "topic_title"), input.Task.Title),
		Objective:         firstNonEmptyDomainPlanDefaultV0(domainWorkFieldStringValueV0(fields, "official_outline"), input.Record.Request.UserIntent),
		TargetAudience:    domainWorkFieldStringValueV0(fields, "level"),
		EstimatedPagesMin: domainWorkFieldIntV0(fields, "target_pages_min"),
		EstimatedPagesMax: domainWorkFieldIntV0(fields, "target_pages_max"),
	}
}

func firstNonEmptyDomainPlanDefaultV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
