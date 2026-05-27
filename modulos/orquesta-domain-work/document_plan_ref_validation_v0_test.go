package orquestadomainwork

import (
	"encoding/json"
	"testing"
)

func TestDomainDocumentPlanV0RechazaRefsDuplicadosPorTipo(t *testing.T) {
	for _, tc := range []struct {
		name   string
		field  string
		mutate func(*DomainDocumentPlanV0)
	}{
		{
			name:  "sections",
			field: "sections.section_ref",
			mutate: func(plan *DomainDocumentPlanV0) {
				plan.Sections = append(plan.Sections, plan.Sections[0])
			},
		},
		{
			name:  "visuals",
			field: "visuals.visual_ref",
			mutate: func(plan *DomainDocumentPlanV0) {
				plan.Visuals = append(plan.Visuals, plan.Visuals[0])
			},
		},
		{
			name:  "reviews",
			field: "review_steps.review_ref",
			mutate: func(plan *DomainDocumentPlanV0) {
				plan.ReviewSteps = append(plan.ReviewSteps, plan.ReviewSteps[0])
			},
		},
		{
			name:  "deliverables",
			field: "deliverables.deliverable_ref",
			mutate: func(plan *DomainDocumentPlanV0) {
				plan.Deliverables = append(plan.Deliverables, plan.Deliverables[0])
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := validDomainDocumentPlanForTestV0()
			tc.mutate(&plan)

			issues := ValidateDomainDocumentPlanV0(plan)

			if !domainDocumentPlanIssueForTestV0(issues, ErrDomainDocumentPlanRefDuplicateV0, tc.field) {
				t.Fatalf("issues=%+v", issues)
			}
		})
	}
}

func TestCanonicalDomainDocumentPlanPayloadJSONV0DiagnosticaArraysRawInvalidos(t *testing.T) {
	payload := `{
		"schema_version":"domain_document_plan.v0",
		"plan_ref":"plan-ref-001",
		"domain_ref":"opes",
		"work_kind":"plan_tema",
		"document_kind":"tema",
		"language_code":"es",
		"title":"Tema",
		"objective":"Planificar tema.",
		"sections":{"section_ref":"sec-01"},
		"deliverables":[{"deliverable_ref":"del-01","artifact_type":"markdown","title":"Markdown"}]
	}`

	_, ok, issues := CanonicalDomainDocumentPlanPayloadJSONWithIssuesV0(
		payload,
		DomainDocumentPlanPayloadDefaultsV0{},
	)

	if ok || !domainDocumentPlanIssueForTestV0(issues, ErrDomainDocumentPlanArrayInvalidV0, "sections") {
		t.Fatalf("ok=%v issues=%+v", ok, issues)
	}
}

func TestCanonicalDomainDocumentPlanPayloadJSONV0DerivaRefsUnicos(t *testing.T) {
	payload := `{
		"schema_version":"domain_document_plan.v0",
		"plan_ref":"plan-ref-001",
		"domain_ref":"opes",
		"work_kind":"plan_tema",
		"document_kind":"tema",
		"language_code":"es",
		"title":"Tema",
		"objective":"Planificar tema.",
		"sections":[
			{"title":"Marco comun","objective":"Primer bloque","work_kind":"draft_content_block"},
			{"title":"Marco comun","objective":"Segundo bloque","work_kind":"draft_content_block"}
		],
		"deliverables":[
			{"name":"markdown","title":"Markdown"},
			{"name":"markdown","title":"Markdown"}
		]
	}`

	canonical, ok, issues := CanonicalDomainDocumentPlanPayloadJSONWithIssuesV0(
		payload,
		DomainDocumentPlanPayloadDefaultsV0{},
	)
	if !ok {
		t.Fatalf("issues=%+v", issues)
	}
	var plan DomainDocumentPlanV0
	if err := json.Unmarshal([]byte(canonical), &plan); err != nil {
		t.Fatalf("json canonico invalido: %v", err)
	}
	if plan.Sections[0].SectionRef != "section-marco_comun" ||
		plan.Sections[1].SectionRef != "section-marco_comun-2" ||
		plan.Deliverables[0].DeliverableRef != "deliverable-markdown" ||
		plan.Deliverables[1].DeliverableRef != "deliverable-markdown-2" {
		t.Fatalf("plan=%+v", plan)
	}
}

func domainDocumentPlanIssueForTestV0(
	issues []DomainWorkIssueV0,
	code string,
	field string,
) bool {
	for _, issue := range issues {
		if issue.Code == code && issue.Field == field {
			return true
		}
	}
	return false
}
