package orquestadocumentplanexpander

import (
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestExpandDomainDocumentPlanV0ProduceJobsDerivadosCompatibles(t *testing.T) {
	plan := validDocumentPlanForExpanderTestV0()
	result := ExpandDomainDocumentPlanV0(DomainDocumentPlanExpansionRequestV0{
		Plan:          plan,
		CorrelationID: "corr-docplan-001",
		RequestedBy:   "test",
		InterfaceRefs: []string{"domain_work.v0"},
		InputFields: []orquestadomainwork.DomainWorkFieldV0{{
			Name:  "context_budget_profile",
			Value: "standard",
		}},
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "external_job_ref",
			Ref:  "job-plan-001",
		}},
		EvidenceRefs: []string{"evidence-expansion-001"},
	})
	if len(result.Issues) != 0 {
		t.Fatalf("issues=%+v", result.Issues)
	}
	wantKinds := []string{
		"draft_content_block",
		"generate_visual_asset",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
	}
	if len(result.Jobs) != len(wantKinds) {
		t.Fatalf("jobs=%d want=%d %+v", len(result.Jobs), len(wantKinds), result.Jobs)
	}
	for i, want := range wantKinds {
		job := result.Jobs[i]
		if job.WorkKind != want {
			t.Fatalf("job[%d].work_kind=%q want=%q", i, job.WorkKind, want)
		}
		if issues := orquestadomainwork.ValidateDomainWorkJobRequestV0(job); len(issues) != 0 {
			t.Fatalf("job[%d] issues=%+v job=%+v", i, issues, job)
		}
		if job.CorrelationID != "corr-docplan-001" ||
			job.RequestedBy != "test" ||
			job.DomainRef != "dominio-demo" {
			t.Fatalf("job[%d]=%+v", i, job)
		}
		if !stringInDocumentPlanExpanderSetV0(job.WorkRefs, "plan-temario-001") {
			t.Fatalf("job[%d].work_refs=%v", i, job.WorkRefs)
		}
		if fieldValueForDocumentPlanExpanderTestV0(job.InputFields, "expected_artifact_type") == "work_delivery" {
			t.Fatalf("job[%d] cayo a work_delivery: %+v", i, job)
		}
	}
	for _, job := range result.Jobs {
		want := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(job.WorkKind)
		if got := fieldValueForDocumentPlanExpanderTestV0(job.InputFields, "expected_artifact_type"); got != want {
			t.Fatalf("work_kind=%s expected_artifact_type=%q want=%q", job.WorkKind, got, want)
		}
	}
	assemble := result.Jobs[len(result.Jobs)-1]
	if assemble.WorkKind != "assemble_topic" ||
		!fieldValuesContainForDocumentPlanExpanderTestV0(assemble.InputFields, "deliverable_artifact_types", "assembled_topic") {
		t.Fatalf("assemble job=%+v", assemble)
	}
}

func TestExpandDomainDocumentPlanV0RechazaPlanInvalidoSinJobsParciales(t *testing.T) {
	plan := validDocumentPlanForExpanderTestV0()
	plan.Sections = nil

	result := ExpandDomainDocumentPlanV0(DomainDocumentPlanExpansionRequestV0{Plan: plan})
	if len(result.Issues) == 0 {
		t.Fatalf("expected issues")
	}
	if len(result.Jobs) != 0 {
		t.Fatalf("jobs parciales=%+v", result.Jobs)
	}
}

func TestExpandDomainDocumentPlanV0RechazaRefsInvalidasEnJobsDerivados(t *testing.T) {
	plan := validDocumentPlanForExpanderTestV0()
	plan.Sections[0].SourceRefs = []string{"source/invalid"}

	result := ExpandDomainDocumentPlanV0(DomainDocumentPlanExpansionRequestV0{Plan: plan})
	if len(result.Issues) == 0 {
		t.Fatalf("expected issues")
	}
	if len(result.Jobs) != 0 {
		t.Fatalf("jobs parciales=%+v", result.Jobs)
	}
}

func TestExpandDomainDocumentPlanV0RechazaIdentidadDuplicada(t *testing.T) {
	plan := validDocumentPlanForExpanderTestV0()
	plan.Sections = append(plan.Sections, plan.Sections[0])

	result := ExpandDomainDocumentPlanV0(DomainDocumentPlanExpansionRequestV0{Plan: plan})

	if len(result.Issues) == 0 ||
		result.Issues[0].Code != orquestadomainwork.ErrDomainDocumentPlanRefDuplicateV0 ||
		result.Issues[0].Field != "sections.section_ref" {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Jobs) != 0 {
		t.Fatalf("jobs parciales=%+v", result.Jobs)
	}
}

func validDocumentPlanForExpanderTestV0() orquestadomainwork.DomainDocumentPlanV0 {
	return orquestadomainwork.DomainDocumentPlanV0{
		PlanRef:           "plan-temario-001",
		DomainRef:         "dominio-demo",
		WorkKind:          "plan_temario",
		DocumentKind:      "documento_formativo",
		ScopeRef:          "programa-001",
		LanguageCode:      "es",
		Title:             "Temario completo",
		Objective:         "Planificar un temario con derivados ejecutables.",
		TargetAudience:    "Opositores",
		EstimatedPagesMin: 120,
		EstimatedPagesMax: 180,
		Sections: []orquestadomainwork.DomainDocumentPlanSectionV0{{
			SectionRef:       "sec-01",
			Order:            1,
			Title:            "Tema 1",
			Objective:        "Redactar bloque base.",
			WorkKind:         "redaccion_tema",
			TargetWordsMin:   900,
			TargetWordsMax:   1200,
			RequiredElements: []string{"autores", "legislacion"},
			AcceptanceCriteria: []string{
				"Incluye estructura docente.",
			},
			SourceRefs: []string{"source-001"},
		}},
		Visuals: []orquestadomainwork.DomainDocumentPlanVisualV0{{
			VisualRef:    "vis-01",
			VisualType:   "diagrama",
			PlacementRef: "sec-01",
			Objective:    "Crear apoyo visual.",
			WorkKind:     "visual_asset_plan",
			AcceptanceCriteria: []string{
				"Puede insertarse en el tema.",
			},
			SourceRefs: []string{"source-001"},
		}},
		ReviewSteps: []orquestadomainwork.DomainDocumentPlanReviewV0{
			{
				ReviewRef: "rev-legal",
				Order:     1,
				WorkKind:  "revision_legal_deontologica",
				Objective: "Revisar encaje legal.",
			},
			{
				ReviewRef: "rev-pedagogical",
				Order:     2,
				WorkKind:  "revision_pedagogica",
				Objective: "Revisar enfoque pedagogico.",
			},
			{
				ReviewRef: "rev-quality",
				Order:     3,
				WorkKind:  "",
				Objective: "Revisar calidad editorial.",
			},
			{
				ReviewRef: "validate-topic",
				Order:     4,
				WorkKind:  "validar_tema",
				Objective: "Validar tema contra contrato.",
			},
			{
				ReviewRef: "assemble-topic",
				Order:     5,
				WorkKind:  "ensamblado_y_exportacion",
				Objective: "Ensamblar tema final.",
			},
		},
		Deliverables: []orquestadomainwork.DomainDocumentPlanDeliverableV0{{
			DeliverableRef: "del-assembled-topic",
			ArtifactType:   "assembled_topic",
			Title:          "Tema ensamblado",
			Required:       true,
		}},
		QualityCriteria: []string{"calidad_editorial", "trazabilidad"},
		Constraints:     []string{"sin_placeholder"},
		SourceRefs:      []string{"source-001"},
		EvidenceRefs:    []string{"evidence-plan-001"},
	}
}

func fieldValueForDocumentPlanExpanderTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) string {
	for _, field := range fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}

func fieldValuesContainForDocumentPlanExpanderTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) bool {
	for _, field := range fields {
		if field.Name != name {
			continue
		}
		return stringInDocumentPlanExpanderSetV0(field.Values, value)
	}
	return false
}

func stringInDocumentPlanExpanderSetV0(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
