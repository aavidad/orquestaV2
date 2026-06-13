package orquestadomainwork

import "strings"

func documentPlanRootWorkKindV0(value string, fallback string) string {
	value = compactDocumentPlanRefTextV0(value)
	if strings.HasPrefix(value, "plan_") {
		return value
	}
	fallback = compactDocumentPlanRefTextV0(fallback)
	if strings.HasPrefix(fallback, "plan_") {
		return fallback
	}
	return value
}

func documentPlanSectionWorkKindV0(value string) string {
	switch compactDocumentPlanRefTextV0(value) {
	case "",
		"redaccion_tema",
		"redaccion_documental",
		"investigacion_y_redaccion",
		"sintesis_pedagogica",
		"control_editorial",
		"desarrollo_contenido",
		"tema_grande":
		return "draft_content_block"
	default:
		return compactDocumentPlanRefTextV0(value)
	}
}

func documentPlanVisualWorkKindV0(value string) string {
	switch compactDocumentPlanRefTextV0(value) {
	case "",
		"visual_asset_plan",
		"plan_visual",
		"visual_assets",
		"revision_visual_assets",
		"esquema_estudio",
		"diagrama_flujo":
		return "generate_visual_asset"
	default:
		return compactDocumentPlanRefTextV0(value)
	}
}

func documentPlanReviewWorkKindV0(value string) string {
	switch compactDocumentPlanRefTextV0(value) {
	case "":
		return "review_quality"
	case "validacion_contrato", "validacion_documental", "validar_tema":
		return "validate_topic"
	case "revision_psicologia", "revision_contenido", "revision_contenido_tecnico",
		"revision_visual_assets", "revision_visual", "revision_editorial":
		return "review_quality"
	case "revision_legal", "revision_legal_deontologica":
		return "review_legal"
	case "revision_pedagogica", "revision_pedagogical":
		return "review_pedagogical"
	case "review_codex", "revision_codex", "review_gemini", "revision_gemini",
		"review_claude", "revision_claude":
		return "review_agent_independent"
	case "review_pair_codex_gemini", "review_pair_codex_claude",
		"review_pair_gemini_claude", "revision_cruzada", "revision_por_pares":
		return "review_agent_pair"
	case "review_director_consolidation", "revision_director_consolidation",
		"consolidacion_director":
		return "review_director_consolidation"
	case "ensamblado_y_exportacion", "ensamblado", "exportacion":
		return "assemble_topic"
	case "generate_question_bank", "generate_topic_tests", "create_topic_tests",
		"banco_preguntas_tema", "crear_tests_tema":
		return "generate_question_bank"
	case "generate_topic_audio", "generacion_audio", "generar_audio",
		"crear_audio_tema", "audio_tema", "narracion_tema", "sintesis_voz_tema",
		"topic_audio", "audio_asset", "tts_topic":
		return "generate_audio_asset"
	case "generate_tutor_assets", "generate_rag_assets", "generate_rag_tutor_assets",
		"configure_temario_tutor", "configure_temario_bots", "rag_tutor":
		return "generate_tutor_assets"
	case "interactive_practice", "practice_package", "create_interactive_practice":
		return "generate_interactive_practice"
	case "generate_learning_games", "learning_games", "juegos_temario":
		return "generate_interactive_practice"
	case "generate_html_site", "generate_local_html_site", "html_temario_local",
		"crear_html_temario":
		return "generate_html_site"
	case "help_package", "create_help_package", "generate_help_manual_assets",
		"manuales_ayuda":
		return "generate_help_package"
	case "finalize_topic_package", "paquete_final_tema",
		"finalize_temario_package", "finalize_syllabus_package",
		"completed_syllabus_package", "paquete_final_temario":
		return "finalize_domain_package"
	default:
		return compactDocumentPlanRefTextV0(value)
	}
}
