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
	switch normalizeDomainWorkDeliveryAliasV0(value) {
	case "content_block", "draft_content_block", "generate_block", "generate_program_topic_draft",
		"bloque", "bloque_contenido", "contenido_bloque":
		return "content_block"
	case "visual_asset", "generate_visual_asset", "visual", "recurso_visual", "activo_visual":
		return "visual_asset"
	case "block_revision", "revision", "review_legal", "review_pedagogical", "review_quality",
		"validate_topic", "revision_bloque", "revision_texto":
		return "block_revision"
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
	case "assembled_topic", "assemble_topic", "tema_ensamblado":
		return "assembled_topic"
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
