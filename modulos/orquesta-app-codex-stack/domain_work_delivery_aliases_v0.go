package orquestaappcodexstack

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func domainWorkDeliveryArtifactTypeMatchesV0(actual string, expected string) bool {
	actual = domainWorkDeliveryCanonicalArtifactTypeV0(actual)
	expected = domainWorkDeliveryCanonicalArtifactTypeV0(expected)
	return actual != "" && actual == expected
}

func domainWorkDeliveryCanonicalArtifactTypeV0(value string) string {
	key := normalizeDomainWorkDeliveryAliasV0(value)
	if artifactType := domainWorkArtifactTypeForWorkKindV0(key); artifactType != orquestadomainwork.DomainWorkArtifactTypeGenericWorkDeliveryV0 {
		return artifactType
	}
	switch key {
	case "content_block", "draft_content_block", "generate_block", "generate_program_topic_draft",
		"bloque", "bloque_contenido", "contenido_bloque", "redaccion_tema",
		"redaccion_documental", "investigacion_y_redaccion", "sintesis_pedagogica",
		"control_editorial", "desarrollo_contenido", "tema_grande":
		return "content_block"
	case "visual_asset", "generate_visual_asset", "visual", "recurso_visual", "activo_visual":
		return "visual_asset"
	case "block_revision", "revision", "review_legal", "review_pedagogical", "review_quality",
		"validate_topic", "revision_bloque", "revision_texto", "revision_legal",
		"revision_legal_deontologica", "revision_pedagogica", "revision_pedagogical",
		"revision_psicologia", "revision_contenido", "revision_contenido_tecnico",
		"revision_editorial", "revision_calidad", "validacion_contrato",
		"validacion_documental", "validacion_tema", "validar_tema":
		return "block_revision"
	case "agent_review_report", "review_agent_independent", "review_independent_agent",
		"review_codex", "review_gemini", "review_claude":
		return "agent_review_report"
	case "agent_pair_review_report", "review_agent_pair", "review_peer_pair", "review_pair",
		"review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude":
		return "agent_pair_review_report"
	case "director_review_matrix", "review_director_consolidation", "review_consensus_director":
		return "director_review_matrix"
	case "completed_syllabus_package", "finalize_temario_package", "finalize_syllabus_package",
		"close_temario_package":
		return "completed_syllabus_package"
	case "source", "fuente", "research_sources", "download_source", "verify_sources":
		return "source"
	case "topic_summary", "summary", "resumen", "summarize_block", "summarize_chapter",
		"summarize_topic", "create_exam_outline":
		return "topic_summary"
	case "topic_expansion_package", "expand_topic_from_summary", "paquete_tema",
		"tema_completo", "desarrollo_tema":
		return "topic_expansion_package"
	case "document_plan", "plan_documento", "plan_tema", "plan_temario", "plan",
		"planificacion", "planificacion_documental":
		return orquestadomainwork.DomainDocumentPlanArtifactTypeV0
	case "assembled_topic", "assemble_topic", "tema_ensamblado", "ensamblado",
		"ensamblado_y_exportacion", "exportacion":
		return "assembled_topic"
	case "audio_asset", "generate_audio_asset", "generate_topic_audio", "create_topic_audio",
		"create_audio_asset", "synthesize_topic_audio", "narrate_topic", "tts_topic",
		"generacion_audio", "generar_audio", "crear_audio_tema", "audio_tema",
		"narracion_tema", "sintesis_voz_tema", "topic_audio":
		return "audio_asset"
	case "work_delivery", "entrega", "resultado":
		return "work_delivery"
	default:
		return strings.TrimSpace(value)
	}
}

func domainWorkDeliveryCanonicalPayloadFieldNameV0(artifactType string, name string) string {
	key := normalizeDomainWorkDeliveryAliasV0(name)
	if key == "" {
		return ""
	}
	switch domainWorkDeliveryCanonicalArtifactTypeV0(artifactType) {
	case "content_block":
		switch key {
		case "topic_id", "topicid", "topic", "tema_id", "id_tema", "idtema":
			return "topic_id"
		case "chapter_id", "chapterid", "chapter", "capitulo_id", "id_capitulo", "idcapitulo":
			return "chapter_id"
		case "block_type", "type", "tipo", "tipo_bloque", "kind":
			return "block_type"
		case "title", "titulo", "nombre", "name", "heading", "encabezado":
			return "title"
		case "markdown", "body", "content", "contenido", "texto", "text":
			return "body"
		case "language_code", "language", "locale", "idioma":
			return "language_code"
		case "source_refs", "sources", "fuentes", "source_ids", "source_references":
			return "source_refs"
		case "citations", "citas":
			return "citations"
		}
	case "visual_asset":
		switch key {
		case "topic_id", "topicid", "topic", "tema_id", "id_tema", "idtema":
			return "topic_id"
		case "chapter_id", "chapterid", "chapter", "capitulo_id", "id_capitulo", "idcapitulo":
			return "chapter_id"
		case "asset_type", "visual_type", "tipo_visual", "tipo", "kind":
			return "asset_type"
		case "format", "formato":
			return "format"
		case "title", "titulo", "nombre", "name":
			return "title"
		case "caption", "leyenda", "pie", "pie_de_figura":
			return "caption"
		case "alt_text", "alttext", "texto_alternativo", "descripcion_accesible":
			return "alt_text"
		case "body", "content", "contenido", "markdown":
			return "body"
		case "svg":
			return "svg"
		case "mermaid":
			return "mermaid"
		case "html":
			return "html"
		case "image_data_uri", "imagedatauri", "data_uri":
			return "image_data_uri"
		case "placement", "ubicacion", "posicion":
			return "placement"
		case "language_code", "language", "locale", "idioma":
			return "language_code"
		case "source_refs", "sources", "fuentes", "source_ids", "source_references":
			return "source_refs"
		case "citations", "citas":
			return "citations"
		}
	case "source":
		switch key {
		case "title", "titulo", "nombre", "name":
			return "title"
		case "type", "tipo", "source_type":
			return "type"
		case "organization", "organizacion", "organismo", "publisher":
			return "organization"
		case "original_url", "url", "enlace":
			return "url"
		case "language_code", "language", "locale", "idioma":
			return "language_code"
		}
	case "topic_summary":
		switch key {
		case "topic_id", "topicid", "tema_id", "id_tema", "idtema":
			return "topic_id"
		case "title", "titulo", "nombre", "name":
			return "title"
		case "markdown", "body", "content", "contenido", "texto", "text":
			return "markdown"
		case "source_refs", "sources", "fuentes", "source_ids", "source_references":
			return "source_refs"
		}
	case "audio_asset":
		switch key {
		case "topic_id", "topicid", "topic", "tema_id", "id_tema", "idtema":
			return "topic_id"
		case "assembled_topic_artifact_id", "assembled_topic_ref", "assembled_ref",
			"tema_ensamblado_ref", "artifact_assembled_topic", "source_artifact_ref":
			return "assembled_topic_artifact_id"
		case "language_code", "language", "locale", "idioma":
			return "language_code"
		case "voice_profile_ref", "voice_ref", "voz_ref", "perfil_voz":
			return "voice_profile_ref"
		case "audio_profile_ref", "profile_ref", "perfil_audio":
			return "audio_profile_ref"
		case "format", "formato":
			return "format"
		case "mime_type", "mimetype", "content_type", "tipo_mime":
			return "mime_type"
		case "duration_seconds", "duration", "duracion", "duracion_segundos":
			return "duration_seconds"
		case "audio_ref", "audio_asset_ref", "ref_audio":
			return "audio_ref"
		case "manifest_ref", "ref_manifest", "manifest":
			return "manifest_ref"
		case "checksum", "sha256", "digest":
			return "checksum"
		case "segments", "segmentos", "audio_segments", "segmentos_audio",
			"section_audio_segments", "section_audio_links", "section_audios",
			"apartado_audio_segments", "apartado_audios", "mapa_apartados_audio",
			"section_audio_map", "audio_manifest_segments":
			return "segments"
		case "source_refs", "sources", "fuentes", "source_ids", "source_references":
			return "source_refs"
		}
	default:
		switch key {
		case "topic_id", "topicid", "tema_id", "id_tema", "idtema":
			return "topic_id"
		case "chapter_id", "chapterid", "capitulo_id", "id_capitulo", "idcapitulo":
			return "chapter_id"
		case "language_code", "language", "locale", "idioma":
			return "language_code"
		case "title", "titulo", "nombre", "name":
			return "title"
		case "markdown", "body", "content", "contenido", "texto", "text":
			return "body"
		case "source_refs", "sources", "fuentes", "source_ids", "source_references":
			return "source_refs"
		}
	}
	return strings.TrimSpace(name)
}

func normalizeDomainWorkDeliveryAliasV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
		"ñ", "n",
		"-", "_",
		" ", "_",
		".", "_",
	)
	return strings.Trim(replacer.Replace(value), "_")
}
