package orquestadomainwork

import (
	"bytes"
	"encoding/json"
	"strings"
)

func CanonicalDomainDocumentPlanPayloadJSONV0(
	body string,
	defaults DomainDocumentPlanPayloadDefaultsV0,
) (string, bool) {
	canonical, ok, _ := CanonicalDomainDocumentPlanPayloadJSONWithIssuesV0(body, defaults)
	return canonical, ok
}

func CanonicalDomainDocumentPlanPayloadJSONWithIssuesV0(
	body string,
	defaults DomainDocumentPlanPayloadDefaultsV0,
) (string, bool, []DomainWorkIssueV0) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", false, nil
	}
	var plan DomainDocumentPlanV0
	if json.Unmarshal([]byte(body), &plan) == nil {
		plan = applyDocumentPlanDefaultsV0(plan, defaults)
		if issues := ValidateDomainDocumentPlanV0(plan); len(issues) == 0 {
			canonical, ok := marshalCanonicalDocumentPlanV0(plan)
			return canonical, ok, nil
		}
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &raw) != nil || len(raw) == 0 {
		return "", false, nil
	}
	if issues := validateDocumentPlanRawArrayFieldsV0(raw); len(issues) > 0 {
		return "", false, issues
	}
	plan = documentPlanFromRawPayloadV0(raw, defaults)
	if issues := ValidateDomainDocumentPlanV0(plan); len(issues) > 0 {
		return "", false, issues
	}
	canonical, ok := marshalCanonicalDocumentPlanV0(plan)
	return canonical, ok, nil
}

func validateDocumentPlanRawArrayFieldsV0(
	raw map[string]json.RawMessage,
) []DomainWorkIssueV0 {
	for _, field := range []string{"sections", "visuals", "review_steps", "deliverables"} {
		value, ok := raw[field]
		if !ok {
			continue
		}
		trimmed := bytes.TrimSpace(value)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			continue
		}
		var values []map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &values); err != nil {
			return []DomainWorkIssueV0{{
				Code:  ErrDomainDocumentPlanArrayInvalidV0,
				Field: field,
			}}
		}
	}
	return nil
}

func documentPlanFromRawPayloadV0(
	raw map[string]json.RawMessage,
	defaults DomainDocumentPlanPayloadDefaultsV0,
) DomainDocumentPlanV0 {
	min, max := documentPlanEstimatedPagesV0(raw, defaults)
	plan := DomainDocumentPlanV0{
		SchemaVersion:     firstDocumentPlanStringV0(raw, "schema_version"),
		PlanRef:           firstDocumentPlanStringV0(raw, "plan_ref", "plan_id"),
		DomainRef:         firstDocumentPlanStringV0(raw, "domain_ref"),
		WorkKind:          firstDocumentPlanStringV0(raw, "work_kind"),
		DocumentKind:      firstDocumentPlanStringV0(raw, "document_kind"),
		ScopeRef:          firstDocumentPlanStringV0(raw, "scope_ref", "topic_id", "program_id"),
		LanguageCode:      firstDocumentPlanStringV0(raw, "language_code", "locale"),
		Title:             firstDocumentPlanStringV0(raw, "title", "topic_title"),
		Objective:         firstDocumentPlanStringV0(raw, "objective", "official_outline"),
		TargetAudience:    firstDocumentPlanStringV0(raw, "target_audience", "level"),
		EstimatedPagesMin: min,
		EstimatedPagesMax: max,
		Sections:          documentPlanSectionsFromRawV0(raw["sections"]),
		Visuals:           documentPlanVisualsFromRawV0(raw["visuals"]),
		ReviewSteps:       documentPlanReviewsFromRawV0(raw["review_steps"]),
		Deliverables:      documentPlanDeliverablesFromRawV0(raw["deliverables"]),
		QualityCriteria:   firstDocumentPlanStringsV0(raw, "quality_criteria"),
		Constraints: compactDomainWorkStringsV0(append(
			firstDocumentPlanStringsV0(raw, "constraints"),
			firstDocumentPlanStringsV0(raw, "scope_exclusions")...,
		)),
		SourceRefs:   firstDocumentPlanRefsV0(raw, "source_refs"),
		EvidenceRefs: firstDocumentPlanRefsV0(raw, "evidence_refs"),
	}
	if plan.PlanRef == "" {
		plan.PlanRef = prefixedDocumentPlanRefV0("plan", firstDocumentPlanStringV0(raw, "job_id"))
	}
	return applyDocumentPlanDefaultsV0(plan, defaults)
}

func applyDocumentPlanDefaultsV0(
	plan DomainDocumentPlanV0,
	defaults DomainDocumentPlanPayloadDefaultsV0,
) DomainDocumentPlanV0 {
	if plan.PlanRef == "" {
		plan.PlanRef = defaults.PlanRef
	}
	if plan.DomainRef == "" {
		plan.DomainRef = defaults.DomainRef
	}
	if plan.WorkKind == "" {
		plan.WorkKind = defaults.WorkKind
	}
	plan.WorkKind = documentPlanRootWorkKindV0(plan.WorkKind, defaults.WorkKind)
	if plan.DocumentKind == "" {
		plan.DocumentKind = defaults.DocumentKind
	}
	if plan.ScopeRef == "" {
		plan.ScopeRef = defaults.ScopeRef
	}
	if plan.LanguageCode == "" {
		plan.LanguageCode = defaults.LanguageCode
	}
	if plan.Title == "" {
		plan.Title = defaults.Title
	}
	if plan.Objective == "" {
		plan.Objective = defaults.Objective
	}
	if plan.TargetAudience == "" {
		plan.TargetAudience = defaults.TargetAudience
	}
	if plan.EstimatedPagesMin == 0 {
		plan.EstimatedPagesMin = defaults.EstimatedPagesMin
	}
	if plan.EstimatedPagesMax == 0 {
		plan.EstimatedPagesMax = defaults.EstimatedPagesMax
	}
	return NormalizeDomainDocumentPlanV0(plan)
}

func marshalCanonicalDocumentPlanV0(plan DomainDocumentPlanV0) (string, bool) {
	data, err := json.Marshal(NormalizeDomainDocumentPlanV0(plan))
	if err != nil {
		return "", false
	}
	return string(data), true
}
