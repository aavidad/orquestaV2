package orquestaopesbridge

import (
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestBuildExternalWorkRunRequestV0MapeaSummarizeTopic(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestWithContextV0(orquestaopesconnector.ExternalJobV0{
		ID:             "job-ref-summary-001",
		Type:           "summarize_topic",
		Status:         "pending",
		ExecutionMode:  "external",
		CorrelationID:  "corr-ref-summary-001",
		IdempotencyKey: "idem-summary-001",
		RequestedBy:    "opes",
		PayloadJSON: `{
			"program_id":"program-ref-001",
			"topic_id":"topic-ref-001",
			"official_order":90,
			"quality_criteria":["derivar solo del temario contrastado","no inventar"]
		}`,
	}, JobRunConfigV0{PriorityScore: 80}, JobContextV0{
		TopicBlocks: []orquestaopesconnector.TopicBlockV0{{
			ID:         "block-ref-001",
			StableID:   "stable-ref-001",
			Title:      "Bloque 1",
			Markdown:   "Texto del bloque.",
			SourceRefs: []string{"source-ref-001"},
		}},
	})

	if !ok {
		t.Fatalf("request no construida")
	}
	work := req.AppChangeRequest.ExternalWork
	if req.ProjectRef != "opes" ||
		req.AppChangeRequest.AppRef != "opes" ||
		req.AppChangeRequest.ChangeRef != "opes-job-job-ref-summary-001" ||
		req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/summarize_topic/job-ref-summary-001" ||
		work == nil ||
		work.JobRef != "job-ref-summary-001" ||
		work.WorkKind != "summarize_topic" ||
		!fieldValueForTestV0(work.InputFields, "expected_artifact_type", "topic_summary") ||
		!fieldValueForTestV0(work.InputFields, "context_budget_profile", "large") ||
		!fieldValuesForTestV0(work.InputFields, "quality_criteria", []string{"derivar solo del temario contrastado", "no inventar"}) ||
		!fieldJSONForTestV0(work.InputFields, "topic_blocks") ||
		!containsStringForTestV0(work.WorkRefs, "opes-topic_id-topic-ref-001") {
		t.Fatalf("request=%+v work=%+v", req, work)
	}
}

func TestBuildExternalWorkRunRequestV0MapeaExpansionComoLarge(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-ref-expansion-001",
		Type: "expand_topic_from_summary",
		PayloadJSON: `{
			"topic_id":"topic-ref-001",
			"summary_payload_json":{"markdown":"Resumen"},
			"output_contract":["artifact_type=topic_expansion_package"]
		}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	fields := req.AppChangeRequest.ExternalWork.InputFields
	if req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/expand_topic_from_summary/job-ref-expansion-001" ||
		!fieldValueForTestV0(fields, "expected_artifact_type", "topic_expansion_package") ||
		!fieldValueForTestV0(fields, "context_budget_profile", "large") ||
		!fieldValuesContainForTestV0(fields, "opes_global_editorial_policy_2026_05_18", []string{
			"20.250",
			"modo tutor completo",
			"50 preguntas",
			"no infantilizar",
		}) ||
		!fieldValuesContainForTestV0(fields, "opes_html_publication_policy_2026_05_19", []string{
			"patron web tipo Tema 11",
			"primera lectura activa por defecto",
			"banco de preguntas i18n externo",
			"4 opciones",
		}) ||
		!fieldValuesContainForTestV0(fields, "opes_html_topic_template_v1", []string{
			"modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py",
			"modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py",
			"template_ref=opes_html_topic_template_v1",
			"50 items",
			"estructura fija",
		}) ||
		!fieldValueForTestV0(fields, "target_words_min", defaultExpansionTargetWordsMinV0) ||
		!fieldValuesForTestV0(fields, "minimum_quality_gates", expansionQualityGatesV0()) ||
		!fieldValuesForTestV0(fields, "required_document_variants", []string{
			"tema_grande",
			"tema_mediano",
			"resumen",
			"esquema_repaso",
			"plan_visuales",
		}) ||
		!fieldJSONForTestV0(fields, "summary_payload_json") {
		t.Fatalf("request=%+v", req)
	}
	if !containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "incluir resumen/memoria de repaso derivado del tema desarrollado") {
		t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
	}
	if !containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "tema_grande debe alcanzar target_words_min si esta declarado") {
		t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
	}
}

func TestBuildExternalWorkRunRequestV0MarcaContratoSeisSubrolesOPES(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:          "job-tema-001",
		Type:        "draft_content_block",
		PayloadJSON: `{"topic_id":"tema-001","subroles_required":6}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("BuildExternalWorkRunRequestV0 ok=false")
	}
	work := req.AppChangeRequest.ExternalWork
	if work == nil ||
		!containsStringForTestV0(work.InterfaceRefs, "opes-rest-v0") ||
		!containsStringForTestV0(work.InterfaceRefs, "opes-mcp-v0") ||
		!containsStringForTestV0(work.InterfaceRefs, "opes.padre-tema-6-subroles.v1") {
		t.Fatalf("external_work=%+v", work)
	}
}

func TestBuildExternalWorkRunRequestV0UsaTopicDirSeguroComoWriteSetProducto(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-tema-producto-032",
		Type: "draft_content_block",
		PayloadJSON: `{
			"topic_id":"tema-032",
			"topic_dir":"temas/tema_032"
		}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	if len(req.AppChangeRequest.AllowedWriteSet) != 1 ||
		req.AppChangeRequest.AllowedWriteSet[0] != "temas/tema_032" {
		t.Fatalf("allowed_write_set=%+v", req.AppChangeRequest.AllowedWriteSet)
	}
}

func TestBuildExternalWorkRunRequestV0IgnoraWriteSetProductoInseguro(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-tema-producto-inseguro-032",
		Type: "draft_content_block",
		PayloadJSON: `{
			"topic_id":"tema-032",
			"topic_dir":"../tema_032",
			"product_write_set":"/tmp/tema_032",
			"allowed_write_set":"C:\\tmp\\tema_032"
		}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	if len(req.AppChangeRequest.AllowedWriteSet) != 1 ||
		req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/draft_content_block/job-tema-producto-inseguro-032" {
		t.Fatalf("allowed_write_set=%+v", req.AppChangeRequest.AllowedWriteSet)
	}
}

func TestBuildExternalWorkRunRequestV0MapeaPlanTemaComoDocumentPlan(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-ref-plan-001",
		Type: "plan_tema",
		PayloadJSON: `{
			"program_id":"program-ref-001",
			"topic_id":"topic-ref-080",
			"topic_title":"Evaluacion diagnostica en psicologia",
			"quality_criteria":["pedagogico","autores y teorias","legislacion asociada"]
		}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	fields := req.AppChangeRequest.ExternalWork.InputFields
	if req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/plan_tema/job-ref-plan-001" ||
		!fieldValueForTestV0(fields, "expected_artifact_type", orquestadomainwork.DomainDocumentPlanArtifactTypeV0) ||
		!fieldValueForTestV0(fields, "context_budget_profile", "large") ||
		!fieldValueForTestV0(fields, "expected_schema", orquestadomainwork.DomainDocumentPlanSchemaV0) ||
		!fieldValuesForTestV0(fields, "required_plan_parts", documentPlanRequiredPartsV0()) ||
		!fieldValuesForTestV0(fields, "allowed_document_plan_work_kinds", documentPlanAllowedWorkKindsV0()) ||
		!fieldValuesForTestV0(fields, "opes_level_derivation_policy", documentPlanOPESLevelDerivationPolicyV0()) ||
		!fieldValuesForTestV0(fields, "opes_assimilation_method", documentPlanOPESAssimilationMethodV0()) ||
		!containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "devolver DomainDocumentPlanV0 valido") ||
		!containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "no redactar el documento final dentro del plan") ||
		!containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "si existe maestro A1/A2 o A1 equivalente, planificar primero ese maestro y despues derivar B/C1/C2/AP por resumen, reduccion editorial y adaptacion de nivel") {
		t.Fatalf("request=%+v", req)
	}
}

func TestBuildExternalWorkRunRequestV0MapeaPlanTemarioOperadoresComoDocumentPlan(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-ref-plan-operadores-001",
		Type: "plan_temario",
		PayloadJSON: `{
			"program_id":"program-ref-operadores-001",
			"document_kind":"temario_oposicion",
			"language_code":"es",
			"title":"Temario operadores",
			"official_outline":"Operadores, tipos, precedencia, asociatividad y usos.",
			"quality_criteria":["completo","sin placeholders","derivado del programa oficial"],
			"target_pages_min":20,
			"target_pages_max":40
		}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	fields := req.AppChangeRequest.ExternalWork.InputFields
	criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
	if req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/plan_temario/job-ref-plan-operadores-001" ||
		req.AppChangeRequest.ExternalWork.WorkKind != "plan_temario" ||
		!fieldValueForTestV0(fields, "expected_artifact_type", orquestadomainwork.DomainDocumentPlanArtifactTypeV0) ||
		!fieldValueForTestV0(fields, "context_budget_profile", "large") ||
		!fieldValueForTestV0(fields, "expected_schema", orquestadomainwork.DomainDocumentPlanSchemaV0) ||
		!fieldValueForTestV0(fields, "document_kind", "temario_oposicion") ||
		!fieldValuesContainForTestV0(fields, "opes_global_editorial_policy_2026_05_18", []string{
			"notas de test separadas",
			"50 preguntas",
			"visuales utiles no decorativos",
			"no infantilizar",
		}) ||
		!fieldValuesContainForTestV0(fields, "opes_html_publication_policy_2026_05_19", []string{
			"barra lateral plegable",
			"notas de test ocultables",
			"4 opciones",
			"html final debe parsear",
		}) ||
		!fieldValuesContainForTestV0(fields, "opes_html_topic_template_v1", []string{
			"modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py",
			"modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py",
			"mode-bar sticky",
			"50 items",
			"validacion obligatoria",
		}) ||
		!fieldValuesContainForTestV0(fields, "opes_temario_agent_rules_2026_06_04", []string{
			"alcance_busqueda",
			"course_id/topic_id/programa/canon",
			"backups, paquetes historicos",
			"hallazgos fuera de alcance",
		}) ||
		!fieldValuesForTestV0(fields, "allowed_document_plan_work_kinds", documentPlanAllowedWorkKindsV0()) ||
		!fieldValuesForTestV0(fields, "opes_editorial_workflow", documentPlanOPESEditorialWorkflowV0()) ||
		!fieldValuesForTestV0(fields, "opes_level_derivation_policy", documentPlanOPESLevelDerivationPolicyV0()) ||
		!fieldValuesForTestV0(fields, "opes_quality_requirements", documentPlanOPESQualityRequirementsV0()) ||
		!containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "devolver DomainDocumentPlanV0 valido") ||
		!strings.Contains(criteriaText, "incluir research_exam_precedents") ||
		!strings.Contains(criteriaText, "incluir generate_question_bank") ||
		!strings.Contains(criteriaText, "incluir generate_tutor_assets") ||
		!strings.Contains(criteriaText, "incluir generate_learning_games") ||
		!strings.Contains(criteriaText, "incluir generate_html_site") ||
		!strings.Contains(criteriaText, "incluir generate_help_manual_assets") ||
		!strings.Contains(criteriaText, "incluir review_codex, review_gemini y review_claude") ||
		!strings.Contains(criteriaText, "incluir review_pair_codex_gemini, review_pair_codex_claude y review_pair_gemini_claude") ||
		!strings.Contains(criteriaText, "incluir review_director_consolidation") ||
		!strings.Contains(criteriaText, "incluir finalize_temario_package") ||
		!strings.Contains(criteriaText, "respetar flujo editorial: inventario, investigacion externa") {
		t.Fatalf("request=%+v", req)
	}
}

func TestBuildExternalWorkRunRequestV0MapeaAssembleTopicComoAssembledTopic(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-ref-assemble-001",
		Type: "assemble_topic",
		PayloadJSON: `{
			"topic_id":"topic-ref-001",
			"document_plan_artifact_id":"artifact-plan-001"
		}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	fields := req.AppChangeRequest.ExternalWork.InputFields
	if req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/assemble_topic/job-ref-assemble-001" ||
		!fieldValueForTestV0(fields, "expected_artifact_type", "assembled_topic") ||
		!fieldValueForTestV0(fields, "context_budget_profile", "large") ||
		!containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "devolver artifact_type=assembled_topic") ||
		containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "devolver artifact_type=work_delivery") {
		t.Fatalf("request=%+v", req)
	}
}

func TestBuildExternalWorkRunRequestV0MapeaDerivadosOPESConArtefactosEsperados(t *testing.T) {
	cases := []struct {
		workKind     string
		artifactType string
		context      string
	}{
		{workKind: "research_exam_precedents", artifactType: "exam_research_report", context: "large"},
		{workKind: "draft_content_block", artifactType: "content_block", context: "large"},
		{workKind: "generate_visual_asset", artifactType: "visual_asset", context: "standard"},
		{workKind: "generate_question_bank", artifactType: "question_bank", context: "large"},
		{workKind: "review_legal", artifactType: "block_revision", context: "large"},
		{workKind: "review_pedagogical", artifactType: "block_revision", context: "large"},
		{workKind: "review_quality", artifactType: "block_revision", context: "large"},
		{workKind: "review_codex", artifactType: "agent_review_report", context: "large"},
		{workKind: "review_gemini", artifactType: "agent_review_report", context: "large"},
		{workKind: "review_claude", artifactType: "agent_review_report", context: "large"},
		{workKind: "review_pair_codex_gemini", artifactType: "agent_pair_review_report", context: "large"},
		{workKind: "review_pair_codex_claude", artifactType: "agent_pair_review_report", context: "large"},
		{workKind: "review_pair_gemini_claude", artifactType: "agent_pair_review_report", context: "large"},
		{workKind: "review_director_consolidation", artifactType: "director_review_matrix", context: "large"},
		{workKind: "validate_topic", artifactType: "block_revision", context: "large"},
		{workKind: "assemble_topic", artifactType: "assembled_topic", context: "large"},
		{workKind: "generate_audio_asset", artifactType: "audio_asset", context: "large"},
		{workKind: "generate_topic_audio", artifactType: "audio_asset", context: "large"},
		{workKind: "generate_tutor_assets", artifactType: "tutor_bot_package", context: "large"},
		{workKind: "generate_learning_games", artifactType: "learning_games_package", context: "large"},
		{workKind: "generate_html_site", artifactType: "local_html_site", context: "large"},
		{workKind: "generate_help_manual_assets", artifactType: "help_manual_package", context: "large"},
		{workKind: "finalize_topic_package", artifactType: "final_domain_package", context: "large"},
		{workKind: "finalize_temario_package", artifactType: "completed_syllabus_package", context: "large"},
	}
	for _, tc := range cases {
		t.Run(tc.workKind, func(t *testing.T) {
			req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
				ID:   "job-ref-" + strings.ReplaceAll(tc.workKind, "_", "-") + "-001",
				Type: tc.workKind,
				PayloadJSON: `{
					"program_id":"program-ref-operadores-001",
					"topic_id":"topic-ref-operadores-001",
					"document_plan_artifact_id":"artifact-plan-operadores-001"
				}`,
			}, JobRunConfigV0{})

			if !ok {
				t.Fatalf("request no construida")
			}
			work := req.AppChangeRequest.ExternalWork
			if work == nil ||
				work.WorkKind != tc.workKind ||
				!fieldValueForTestV0(work.InputFields, "expected_artifact_type", tc.artifactType) ||
				!fieldValueForTestV0(work.InputFields, "context_budget_profile", tc.context) ||
				!containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "devolver artifact_type="+tc.artifactType) ||
				strings.Contains(req.AppChangeRequest.AllowedWriteSet[0], "work_delivery") {
				t.Fatalf("req=%+v work=%+v", req, work)
			}
			if tc.artifactType == "audio_asset" &&
				(!strings.Contains(strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n"), "devolver manifest de audio") ||
					!strings.Contains(strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n"), "reutilizar audio comun compatible") ||
					!strings.Contains(strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n"), "no incluir rutas locales, proveedor, GPU, modelo ni procesos internos")) {
				t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
			}
			if tc.artifactType == "local_html_site" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "formato real de curso USO/TCAE") ||
					!strings.Contains(criteriaText, "audio/manifests") ||
					!strings.Contains(criteriaText, "locales/i18n") ||
					!strings.Contains(criteriaText, "#uso-material-watermark") ||
					!strings.Contains(criteriaText, "ubicar cada infografia junto al apartado") ||
					!strings.Contains(criteriaText, "no mostrar al alumnado notas de generacion") ||
					!strings.Contains(criteriaText, "no entregar visores single-file") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "visual_asset" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "no saturar el tema") ||
					!strings.Contains(criteriaText, "visuales de relleno") ||
					!strings.Contains(criteriaText, "puntos importantes, dificiles, comparativos o procedimentales") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "question_bank" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "no sobrescribir ni borrar bancos originales") ||
					!strings.Contains(criteriaText, "4 opciones A, B, C y D") ||
					!strings.Contains(criteriaText, "localizable/i18n") ||
					!strings.Contains(criteriaText, opesTCAETestCreationGuideRefV0) ||
					!strings.Contains(criteriaText, "validacion de dificultad/proximidad") ||
					!strings.Contains(criteriaText, "backup previo") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "exam_research_report" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "empezar por course_id") ||
					!strings.Contains(criteriaText, "excluir por defecto backups") ||
					!strings.Contains(criteriaText, "runtime_orquesta") ||
					!strings.Contains(criteriaText, "evidencia blanda") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "help_manual_package" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "SCREENSHOT_HELP_MANUALS.md") ||
					!strings.Contains(criteriaText, "manual.pdf exportado desde HTML") ||
					!strings.Contains(criteriaText, "correo uso@dipgra.es") ||
					!strings.Contains(criteriaText, "datos personales") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "agent_review_report" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "revision independiente") ||
					!strings.Contains(criteriaText, "100% de preguntas") ||
					!strings.Contains(criteriaText, "no tirar trabajo recuperable") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "agent_pair_review_report" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "revision por pares") ||
					!strings.Contains(criteriaText, "acuerdos, desacuerdos") ||
					!strings.Contains(criteriaText, "agent_pair_review_report") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "director_review_matrix" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "Codex, Gemini y Claude") ||
					!strings.Contains(criteriaText, "Codex-Gemini, Codex-Claude y Gemini-Claude") ||
					!strings.Contains(criteriaText, "director_review_matrix") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.artifactType == "completed_syllabus_package" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "temario terminado al 100%") ||
					!strings.Contains(criteriaText, "convertir pendientes causales") ||
					!strings.Contains(criteriaText, "RAG/corpus") ||
					!strings.Contains(criteriaText, "triple visto bueno") ||
					!strings.Contains(criteriaText, "listo_para_revision_operador") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
			if tc.workKind == "finalize_topic_package" {
				criteriaText := strings.Join(req.AppChangeRequest.AcceptanceCriteria, "\n")
				if !strings.Contains(criteriaText, "paquete local verificable de un tema") ||
					!strings.Contains(criteriaText, "followup_refs causales") ||
					!strings.Contains(criteriaText, "no declararlo temario completo") {
					t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
				}
			}
		})
	}
}
