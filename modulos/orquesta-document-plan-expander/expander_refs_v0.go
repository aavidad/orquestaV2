package orquestadocumentplanexpander

import (
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

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
