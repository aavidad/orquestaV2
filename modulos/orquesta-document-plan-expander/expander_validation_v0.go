package orquestadocumentplanexpander

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func validateRawDocumentPlanExpansionRefsV0(
	plan orquestadomainwork.DomainDocumentPlanV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	for _, item := range []struct {
		Values []string
		Field  string
	}{
		{plan.SourceRefs, "source_refs"},
		{plan.EvidenceRefs, "evidence_refs"},
	} {
		if issues := validateRawDocumentPlanExpansionRefListV0(item.Values, item.Field); len(issues) > 0 {
			return issues
		}
	}
	for _, section := range plan.Sections {
		for _, item := range []struct {
			Values []string
			Field  string
		}{
			{section.DependsOn, "sections.depends_on"},
			{section.SourceRefs, "sections.source_refs"},
		} {
			if issues := validateRawDocumentPlanExpansionRefListV0(item.Values, item.Field); len(issues) > 0 {
				return issues
			}
		}
	}
	for _, visual := range plan.Visuals {
		if issues := validateRawDocumentPlanExpansionRefListV0(
			visual.SourceRefs,
			"visuals.source_refs",
		); len(issues) > 0 {
			return issues
		}
	}
	return nil
}

func validateRawDocumentPlanExpansionRefListV0(
	values []string,
	field string,
) []orquestadomainwork.DomainWorkIssueV0 {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if strings.ContainsAny(trimmed, " /\\\t\n\r") {
			return []orquestadomainwork.DomainWorkIssueV0{{
				Code:  orquestadomainwork.ErrDomainDocumentPlanRefInvalidV0,
				Field: field,
			}}
		}
	}
	return nil
}

func validateDocumentPlanExpandedJobIdentitiesV0(
	jobs []orquestadomainwork.DomainWorkJobRequestV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	seenRequestIDs := map[string]struct{}{}
	seenIdempotencyKeys := map[string]struct{}{}
	for _, job := range jobs {
		if issue := validateDocumentPlanExpandedJobIdentityV0(
			seenRequestIDs,
			job.RequestID,
			"jobs.request_id",
		); issue != nil {
			return []orquestadomainwork.DomainWorkIssueV0{*issue}
		}
		if issue := validateDocumentPlanExpandedJobIdentityV0(
			seenIdempotencyKeys,
			job.IdempotencyKey,
			"jobs.idempotency_key",
		); issue != nil {
			return []orquestadomainwork.DomainWorkIssueV0{*issue}
		}
	}
	return nil
}

func validateDocumentPlanExpandedJobIdentityV0(
	seen map[string]struct{},
	value string,
	field string,
) *orquestadomainwork.DomainWorkIssueV0 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if _, ok := seen[trimmed]; ok {
		return &orquestadomainwork.DomainWorkIssueV0{
			Code:  ErrDomainDocumentPlanDerivedJobDuplicateV0,
			Field: field,
		}
	}
	seen[trimmed] = struct{}{}
	return nil
}
