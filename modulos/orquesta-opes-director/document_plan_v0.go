package orquestaopesdirector

import (
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func documentPlanFromArtifactRecordV0(
	record OPESCausalArtifactRecordV0,
) (orquestadomainwork.DomainDocumentPlanV0, bool, []orquestadomainwork.DomainWorkIssueV0) {
	fields := cloneFieldsV0(record.PayloadFields)
	plan := orquestadomainwork.DomainDocumentPlanV0{
		SchemaVersion:     fieldStringV0(fields, "schema_version"),
		PlanRef:           firstNonEmptyV0(fieldStringV0(fields, "plan_ref"), "plan-"+safeRefV0(record.JobRef)),
		DomainRef:         firstNonEmptyV0(fieldStringV0(fields, "domain_ref"), record.DomainRef),
		WorkKind:          firstNonEmptyV0(fieldStringV0(fields, "work_kind"), orquestadomainwork.DomainWorkKindPlanSyllabusV0),
		DocumentKind:      firstNonEmptyV0(fieldStringV0(fields, "document_kind"), "temario_oposicion"),
		ScopeRef:          fieldStringV0(fields, "scope_ref"),
		LanguageCode:      firstNonEmptyV0(fieldStringV0(fields, "language_code"), "es"),
		Title:             firstNonEmptyV0(fieldStringV0(fields, "title"), record.Summary),
		Objective:         firstNonEmptyV0(fieldStringV0(fields, "objective"), "Planificar trabajos derivados OPES."),
		TargetAudience:    fieldStringV0(fields, "target_audience"),
		EstimatedPagesMin: fieldIntV0(fields, "estimated_pages_min"),
		EstimatedPagesMax: fieldIntV0(fields, "estimated_pages_max"),
		QualityCriteria:   fieldStringsV0(fields, "quality_criteria"),
		Constraints:       fieldStringsV0(fields, "constraints"),
		SourceRefs:        fieldStringsV0(fields, "source_refs"),
		EvidenceRefs:      fieldStringsV0(fields, "evidence_refs"),
	}
	fieldJSONV0(fields, "sections", &plan.Sections)
	fieldJSONV0(fields, "visuals", &plan.Visuals)
	fieldJSONV0(fields, "review_steps", &plan.ReviewSteps)
	fieldJSONV0(fields, "deliverables", &plan.Deliverables)
	plan = orquestadomainwork.NormalizeDomainDocumentPlanV0(plan)
	if issues := orquestadomainwork.ValidateDomainDocumentPlanV0(plan); len(issues) > 0 {
		return plan, false, append([]orquestadomainwork.DomainWorkIssueV0{issueV0(ErrOPESCausalArtifactInvalidV0, "document_plan")}, issues...)
	}
	return plan, true, nil
}
