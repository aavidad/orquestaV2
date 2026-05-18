package orquestadomainwork

import (
	"encoding/json"
	"testing"
)

func TestDomainDocumentPlanV0ValidaPlanTemaCompleto(t *testing.T) {
	plan := validDomainDocumentPlanForTestV0()

	normalized := NormalizeDomainDocumentPlanV0(plan)
	if normalized.SchemaVersion != DomainDocumentPlanSchemaV0 ||
		normalized.WorkKind != DomainWorkKindPlanTopicV0 ||
		normalized.Sections[0].SectionRef != "sec-diagnostico-001" ||
		normalized.Visuals[0].WorkKind != "generate_visual_asset" ||
		len(normalized.QualityCriteria) != 4 {
		t.Fatalf("normalized=%+v", normalized)
	}
	var tema PlanTemaV0 = normalized
	if tema.PlanRef != "plan-tema-psicologia-001" {
		t.Fatalf("alias PlanTemaV0 no conserva contrato: %+v", tema)
	}
	if issues := ValidateDomainDocumentPlanV0(normalized); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainDocumentPlanV0RechazaPlanSinSecciones(t *testing.T) {
	plan := validDomainDocumentPlanForTestV0()
	plan.Sections = nil

	issues := ValidateDomainDocumentPlanV0(plan)

	if len(issues) == 0 ||
		issues[0].Code != ErrDomainDocumentPlanSectionsRequiredV0 ||
		issues[0].Field != "sections" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainDocumentPlanV0RechazaWorkKindQueNoEsPlan(t *testing.T) {
	plan := validDomainDocumentPlanForTestV0()
	plan.WorkKind = "draft_content_block"

	issues := ValidateDomainDocumentPlanV0(plan)

	if len(issues) == 0 ||
		issues[0].Code != ErrDomainDocumentPlanWorkKindInvalidV0 ||
		issues[0].Field != "work_kind" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainDocumentPlanV0RechazaRangosInvalidos(t *testing.T) {
	plan := validDomainDocumentPlanForTestV0()
	plan.EstimatedPagesMin = 50
	plan.EstimatedPagesMax = 45

	issues := ValidateDomainDocumentPlanV0(plan)

	if len(issues) == 0 ||
		issues[0].Code != ErrDomainDocumentPlanRangeInvalidV0 ||
		issues[0].Field != "estimated_pages" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainDocumentPlanV0RequiereEntregables(t *testing.T) {
	plan := validDomainDocumentPlanForTestV0()
	plan.Deliverables = nil

	issues := ValidateDomainDocumentPlanV0(plan)

	if len(issues) == 0 ||
		issues[0].Code != ErrDomainDocumentPlanDeliverablesRequiredV0 ||
		issues[0].Field != "deliverables" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestCanonicalDomainDocumentPlanPayloadJSONV0ConvierteAliasDeAgente(t *testing.T) {
	payload := `{
		"schema_version":"domain_document_plan.v0",
		"artifact_type":"document_plan",
		"work_kind":"plan_tema",
		"job_id":"job-ref-plan-001",
		"topic_id":"topic-ref-080",
		"topic_title":"Evaluacion diagnostica en psicologia",
		"document_kind":"tema_oposicion",
		"language_code":"es",
		"target_pages":{"min":45,"max":50},
		"sections":[{
			"section_id":"sec-01-presentacion",
			"title":"Presentacion",
			"objective":"Situar el proceso diagnostico.",
			"work_kind":"draft_content_block",
			"planned_pages":2,
			"required_points":["definiciones"],
			"acceptance_criteria":["lectura facil"]
		}],
		"visuals":[{
			"visual_id":"vis-01-flujo",
			"asset_type":"diagrama de flujo",
			"target_section_id":"sec-01-presentacion",
			"brief":"Mostrar entrevista, hipotesis e informe.",
			"work_kind":"generate_visual_asset"
		}],
		"review_steps":[{
			"review_id":"rev-01-legal",
			"work_kind":"review_legal",
			"scope":"Normativa y proteccion de datos."
		}],
		"deliverables":[{
			"deliverable_id":"del-01-topic-expansion-package",
			"name":"topic_expansion_package",
			"description":"Tema grande completo."
		}]
	}`
	canonical, ok := CanonicalDomainDocumentPlanPayloadJSONV0(
		payload,
		DomainDocumentPlanPayloadDefaultsV0{
			DomainRef:    "opes",
			Objective:    "Planificar tema completo.",
			DocumentKind: "tema_oposicion",
		},
	)
	if !ok {
		t.Fatalf("payload no canonicalizado")
	}
	var plan DomainDocumentPlanV0
	if err := json.Unmarshal([]byte(canonical), &plan); err != nil {
		t.Fatalf("json canonico invalido: %v", err)
	}
	if issues := ValidateDomainDocumentPlanV0(plan); len(issues) != 0 {
		t.Fatalf("issues=%+v canonical=%s", issues, canonical)
	}
	if plan.Sections[0].SectionRef != "sec-01-presentacion" ||
		plan.Visuals[0].VisualType != "diagrama_de_flujo" ||
		plan.Deliverables[0].ArtifactType != "topic_expansion_package" {
		t.Fatalf("plan=%+v", plan)
	}
}

func validDomainDocumentPlanForTestV0() DomainDocumentPlanV0 {
	return DomainDocumentPlanV0{
		PlanRef:           " plan-tema-psicologia-001 ",
		DomainRef:         " opes ",
		WorkKind:          " plan_tema ",
		DocumentKind:      " topic ",
		ScopeRef:          " topic-psicologia-001 ",
		LanguageCode:      " es ",
		Title:             " Tema de evaluacion psicologica ",
		Objective:         " Planificar un tema completo con calidad de oposicion. ",
		TargetAudience:    " Personas opositoras A1/A2. ",
		EstimatedPagesMin: 45,
		EstimatedPagesMax: 50,
		Sections: []DomainDocumentPlanSectionV0{{
			SectionRef:       " sec-diagnostico-001 ",
			Order:            1,
			Title:            " Diagnostico psicologico ",
			Objective:        " Explicar definiciones, autores, teorias y pasos aplicables. ",
			WorkKind:         " draft_content_block ",
			TargetWordsMin:   1400,
			TargetWordsMax:   2200,
			RequiredElements: []string{"autores", "teorias", "legislacion", "ejemplos"},
			AcceptanceCriteria: []string{
				"Lectura facil y pedagogica.",
				"Incluye pasos concretos de diagnostico.",
			},
			SourceRefs: []string{"source-ref-manual-001"},
		}},
		Visuals: []DomainDocumentPlanVisualV0{{
			VisualRef:    " visual-entrevista-001 ",
			VisualType:   " vignette ",
			PlacementRef: " sec-diagnostico-001 ",
			Objective:    " Mostrar entrevista diagnostica sin texto visible. ",
			WorkKind:     " generate_visual_asset ",
			AcceptanceCriteria: []string{
				"SVG o imagen pequena apta para PDF.",
			},
		}},
		ReviewSteps: []DomainDocumentPlanReviewV0{{
			ReviewRef: " review-pedagogico-001 ",
			Order:     1,
			WorkKind:  " validate_topic ",
			Objective: " Revisar que autores, teorias y legislacion no falten. ",
			AcceptanceCriteria: []string{
				"No cerrar si faltan elementos obligatorios.",
			},
		}},
		Deliverables: []DomainDocumentPlanDeliverableV0{
			{
				DeliverableRef: "deliverable-topic-large-001",
				ArtifactType:   "topic_expansion_package",
				Title:          "Tema grande completo",
				Required:       true,
			},
			{
				DeliverableRef: "deliverable-topic-pdf-001",
				ArtifactType:   "pdf",
				Title:          "PDF final",
				Required:       true,
			},
		},
		QualityCriteria: []string{
			"autores_relevantes",
			"teorias",
			"legislacion",
			"lectura_facil",
		},
		Constraints:  []string{"hexagonal", "sin_db_hardcodeada"},
		SourceRefs:   []string{"source-ref-manual-001"},
		EvidenceRefs: []string{"evidence-ref-plan-001"},
	}
}
