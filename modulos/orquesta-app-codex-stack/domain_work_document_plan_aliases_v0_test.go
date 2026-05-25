package orquestaappcodexstack

import (
	"encoding/json"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestCanonicalDomainWorkDeliveryPayloadBodyV0NormalizaPlanOPESReal(t *testing.T) {
	body := `{
		"schema_version":"domain_document_plan.v0",
		"artifact_type":"document_plan",
		"work_kind":"planificacion_documental",
		"plan_ref":"plan-ref-opes-080",
		"domain_ref":"opes",
		"document_kind":"tema_oposicion",
		"language_code":"es",
		"title":"Evaluacion diagnostica",
		"objective":"Planificar un tema completo sin redactarlo.",
		"estimated_pages_min":45,
		"estimated_pages_max":60,
		"sections":[{"section_ref":"sec-01","order":1,"title":"Marco","objective":"Crear base conceptual","work_kind":"redaccion_tema","target_words_min":900,"target_words_max":1200}],
		"visuals":[{"visual_ref":"vis-01","visual_type":"diagrama_flujo","placement_ref":"sec-01","objective":"Mostrar proceso","work_kind":"visual_asset_plan"}],
		"review_steps":[{"review_ref":"rev-01","order":1,"work_kind":"revision_pedagogica","objective":"Revisar pedagogia"}],
		"deliverables":[{"deliverable_ref":"del-01","artifact_type":"tema_grande","title":"Tema grande","required":true}]
	}`
	input := DomainWorkArtifactSubmissionBuildInputV0{
		Record: orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "opes",
					JobRef:     "job-ref-plan-001",
					WorkKind:   "plan_tema",
				},
			},
		},
	}

	canonical := canonicalDomainWorkDeliveryPayloadBodyV0(
		orquestadomainwork.DomainDocumentPlanArtifactTypeV0,
		body,
		input,
	)
	if err := validateDomainWorkDocumentPlanDeliveryV0(canonical); err != nil {
		t.Fatalf("quality gate: %v\n%s", err, canonical)
	}
	var plan orquestadomainwork.DomainDocumentPlanV0
	if err := json.Unmarshal([]byte(canonical), &plan); err != nil {
		t.Fatalf("json canonico invalido: %v", err)
	}
	if plan.WorkKind != "plan_tema" ||
		plan.Sections[0].WorkKind != "draft_content_block" ||
		plan.Visuals[0].WorkKind != "generate_visual_asset" ||
		plan.ReviewSteps[0].WorkKind != "review_pedagogical" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestCanonicalDomainWorkDeliveryPayloadBodyV0ReconoceDerivadosTemarioCompletoOPES(t *testing.T) {
	body := `{
		"schema_version":"domain_document_plan.v0",
		"artifact_type":"document_plan",
		"work_kind":"plan_temario",
		"plan_ref":"plan-ref-temario-completo-001",
		"domain_ref":"opes",
		"document_kind":"temario_oposicion",
		"scope_ref":"program-ref-001",
		"language_code":"es",
		"title":"Temario completo fake",
		"objective":"Planificar un temario completo con derivados esperados.",
		"estimated_pages_min":120,
		"estimated_pages_max":180,
		"sections":[{
			"section_ref":"sec-01",
			"order":1,
			"title":"Tema 1",
			"objective":"Redactar bloque base.",
			"work_kind":"draft_content_block",
			"target_words_min":900
		}],
		"visuals":[{
			"visual_ref":"vis-01",
			"visual_type":"diagrama",
			"placement_ref":"sec-01",
			"objective":"Crear apoyo visual.",
			"work_kind":"generate_visual_asset"
		}],
		"review_steps":[
			{"review_ref":"rev-legal","order":1,"work_kind":"review_legal","objective":"Revisar encaje legal."},
			{"review_ref":"rev-pedagogical","order":2,"work_kind":"review_pedagogical","objective":"Revisar enfoque pedagogico."},
			{"review_ref":"rev-quality","order":3,"work_kind":"review_quality","objective":"Revisar calidad editorial."},
			{"review_ref":"validate-topic","order":4,"work_kind":"validate_topic","objective":"Validar tema contra contrato."},
			{"review_ref":"assemble-topic","order":5,"work_kind":"assemble_topic","objective":"Ensamblar tema final."}
		],
		"deliverables":[{
			"deliverable_ref":"del-assembled-topic",
			"artifact_type":"assembled_topic",
			"title":"Tema ensamblado",
			"required":true
		}]
	}`
	input := DomainWorkArtifactSubmissionBuildInputV0{
		Record: orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.AppChangeRequestV0{
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "opes",
					JobRef:     "job-ref-temario-completo-001",
					WorkKind:   "plan_temario",
				},
			},
		},
	}

	canonical := canonicalDomainWorkDeliveryPayloadBodyV0(
		orquestadomainwork.DomainDocumentPlanArtifactTypeV0,
		body,
		input,
	)
	if err := validateDomainWorkDocumentPlanDeliveryV0(canonical); err != nil {
		t.Fatalf("quality gate: %v\n%s", err, canonical)
	}
	var plan orquestadomainwork.DomainDocumentPlanV0
	if err := json.Unmarshal([]byte(canonical), &plan); err != nil {
		t.Fatalf("json canonico invalido: %v", err)
	}

	recognizedWorkKinds := map[string]bool{
		plan.Sections[0].WorkKind: true,
		plan.Visuals[0].WorkKind:  true,
	}
	for _, review := range plan.ReviewSteps {
		recognizedWorkKinds[review.WorkKind] = true
	}
	for _, expected := range []string{
		"draft_content_block",
		"generate_visual_asset",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
	} {
		if !recognizedWorkKinds[expected] {
			t.Fatalf("work_kind esperado no reconocido %q en plan=%+v", expected, plan)
		}
	}
	if len(plan.Deliverables) != 1 || plan.Deliverables[0].ArtifactType != "assembled_topic" {
		t.Fatalf("deliverables=%+v", plan.Deliverables)
	}

	for _, workKind := range []string{
		"plan_temario",
		"draft_content_block",
		"generate_visual_asset",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
	} {
		expectedArtifact := orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
		if artifact := domainWorkArtifactTypeForWorkKindV0(workKind); artifact != expectedArtifact {
			t.Fatalf("work_kind %q artifact=%q, esperado %q", workKind, artifact, expectedArtifact)
		}
	}
}
