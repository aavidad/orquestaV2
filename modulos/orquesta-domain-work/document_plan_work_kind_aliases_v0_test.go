package orquestadomainwork

import (
	"encoding/json"
	"testing"
)

func TestCanonicalDomainDocumentPlanPayloadJSONV0NormalizaWorkKindsDeAgente(t *testing.T) {
	payload := `{
		"schema_version":"domain_document_plan.v0",
		"artifact_type":"document_plan",
		"work_kind":"planificacion_documental",
		"plan_ref":"plan-ref-learning-080",
		"domain_ref":"domain-ref-learning",
		"document_kind":"learning_topic",
		"language_code":"es",
		"title":"Evaluacion diagnostica",
		"objective":"Planificar un tema completo sin redactarlo.",
		"estimated_pages_min":45,
		"estimated_pages_max":60,
		"sections":[
			{"section_ref":"sec-01","order":1,"title":"Marco","objective":"Crear base conceptual","work_kind":"redaccion_tema","target_words_min":900,"target_words_max":1200},
			{"section_ref":"sec-02","order":2,"title":"Sintesis","objective":"Preparar repaso","work_kind":"sintesis_pedagogica","target_words_min":500,"target_words_max":800}
		],
		"visuals":[
			{"visual_ref":"vis-01","visual_type":"diagrama_flujo","placement_ref":"sec-01","objective":"Mostrar el proceso","work_kind":"visual_asset_plan"}
		],
		"review_steps":[
			{"review_ref":"rev-01","order":1,"work_kind":"validacion_contrato","objective":"Validar contrato"},
			{"review_ref":"rev-02","order":2,"work_kind":"revision_psicologia","objective":"Revisar contenido"},
			{"review_ref":"rev-03","order":3,"work_kind":"revision_pedagogica","objective":"Revisar pedagogia"},
			{"review_ref":"rev-04","order":4,"work_kind":"ensamblado_y_exportacion","objective":"Preparar ensamblado"},
			{"review_ref":"rev-05","order":5,"work_kind":"generacion_audio","objective":"Crear audio accesible del tema"},
			{"review_ref":"rev-06","order":6,"work_kind":"practice_package","objective":"Crear practica interactiva"},
			{"review_ref":"rev-07","order":7,"work_kind":"help_package","objective":"Crear paquete de ayuda"},
			{"review_ref":"rev-08","order":8,"work_kind":"review_codex","objective":"Revision independiente"},
			{"review_ref":"rev-09","order":9,"work_kind":"review_pair_codex_gemini","objective":"Revision cruzada"},
			{"review_ref":"rev-10","order":10,"work_kind":"generate_question_bank","objective":"Crear tests"},
			{"review_ref":"rev-11","order":11,"work_kind":"generate_rag_assets","objective":"Crear RAG"},
			{"review_ref":"rev-12","order":12,"work_kind":"generate_learning_games","objective":"Crear juegos"},
			{"review_ref":"rev-13","order":13,"work_kind":"generate_html_site","objective":"Crear HTML"},
			{"review_ref":"rev-14","order":14,"work_kind":"generate_help_manual_assets","objective":"Crear manuales"},
			{"review_ref":"rev-15","order":15,"work_kind":"finalize_topic_package","objective":"Cerrar paquete"}
		],
		"quality_criteria":[
			{"title":"Lectura pedagogica y clara","rule":"Debe leerse con facilidad"},
			{"title":"Autores relevantes y teorias","rule":"Debe cubrir autores"}
		],
		"deliverables":[
			{"deliverable_ref":"del-01","artifact_type":"tema_grande","title":"Tema grande","required":true},
		{"deliverable_ref":"del-02","artifact_type":"audio_asset","title":"Audio accesible del tema","required":true},
		{"deliverable_ref":"del-03","artifact_type":"interactive_practice_package","title":"Practica interactiva","required":true},
		{"deliverable_ref":"del-04","artifact_type":"help_package","title":"Ayuda","required":true}
		]
	}`

	canonical, ok := CanonicalDomainDocumentPlanPayloadJSONV0(payload, DomainDocumentPlanPayloadDefaultsV0{
		WorkKind: "plan_tema",
	})
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
	if plan.WorkKind != "plan_tema" ||
		plan.Sections[0].WorkKind != "draft_content_block" ||
		plan.Sections[1].WorkKind != "draft_content_block" ||
		plan.Visuals[0].WorkKind != "generate_visual_asset" ||
		plan.ReviewSteps[0].WorkKind != "validate_topic" ||
		plan.ReviewSteps[1].WorkKind != "review_quality" ||
		plan.ReviewSteps[2].WorkKind != "review_pedagogical" ||
		plan.ReviewSteps[3].WorkKind != "assemble_topic" ||
		plan.ReviewSteps[4].WorkKind != "generate_audio_asset" ||
		plan.ReviewSteps[5].WorkKind != "generate_interactive_practice" ||
		plan.ReviewSteps[6].WorkKind != "generate_help_package" ||
		plan.ReviewSteps[7].WorkKind != "review_agent_independent" ||
		plan.ReviewSteps[8].WorkKind != "review_agent_pair" ||
		plan.ReviewSteps[9].WorkKind != "generate_question_bank" ||
		plan.ReviewSteps[10].WorkKind != "generate_tutor_assets" ||
		plan.ReviewSteps[11].WorkKind != "generate_interactive_practice" ||
		plan.ReviewSteps[12].WorkKind != "generate_html_site" ||
		plan.ReviewSteps[13].WorkKind != "generate_help_package" ||
		plan.ReviewSteps[14].WorkKind != "finalize_domain_package" ||
		len(plan.QualityCriteria) != 2 {
		t.Fatalf("plan=%+v", plan)
	}
}
