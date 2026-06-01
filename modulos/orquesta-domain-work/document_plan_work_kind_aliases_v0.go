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
	case "ensamblado_y_exportacion", "ensamblado", "exportacion":
		return "assemble_topic"
	case "generate_topic_audio", "generacion_audio", "generar_audio",
		"crear_audio_tema", "audio_tema", "narracion_tema", "sintesis_voz_tema",
		"topic_audio", "audio_asset", "tts_topic":
		return "generate_audio_asset"
	default:
		return compactDocumentPlanRefTextV0(value)
	}
}
