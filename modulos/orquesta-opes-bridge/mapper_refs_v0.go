package orquestaopesbridge

import (
	"sort"
	"strings"
	"unicode"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func appendOpaqueExecutionRefFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	externalRefs map[string]string,
) ([]orquestadomainwork.DomainWorkFieldV0, bool) {
	refs := opaqueExecutionRefsFromExternalRefsV0(externalRefs)
	for _, name := range opaqueExecutionRefFieldNamesV0() {
		value := strings.TrimSpace(refs[name])
		if value == "" {
			value = fieldValueV0(fields, name)
		}
		if value == "" {
			continue
		}
		if !isOpaqueExecutionRefV0(value) {
			return fields, false
		}
		fields = appendFieldIfMissingV0(fields, name, value)
	}
	return fields, true
}

func opaqueExecutionRefsFromExternalRefsV0(refs map[string]string) map[string]string {
	if len(refs) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(refs))
	for key := range refs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := map[string]string{}
	for _, key := range keys {
		normalized := strings.TrimSpace(key)
		switch normalized {
		case "worktree_ref", "branch_ref":
			out[normalized] = strings.TrimSpace(refs[key])
		}
	}
	return out
}

func opaqueExecutionRefFieldNamesV0() []string {
	return []string{"worktree_ref", "branch_ref"}
}

func isOpaqueExecutionRefV0(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.ContainsAny(value, " /\\\t\n\r")
}

func expectedArtifactTypeV0(jobType string) string {
	return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(jobType)
}

func contextProfileForJobTypeV0(jobType string) string {
	switch strings.TrimSpace(jobType) {
	case "summarize_topic",
		"expand_topic_from_summary",
		"plan_documento",
		"plan_tema",
		"plan_temario",
		"research_exam_precedents",
		"research_exam_results",
		"draft_content_block",
		"generate_question_bank",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
		"generate_html_site",
		"generate_audio_asset",
		"generate_topic_audio",
		"generate_tutor_assets",
		"configure_temario_bots":
		return "large"
	default:
		return "standard"
	}
}

func userIntentForJobV0(jobType string) string {
	return "Resolver job OPES " + strings.TrimSpace(jobType) +
		" y devolver artefacto " + expectedArtifactTypeV0(jobType) +
		" por el contrato publico OPES."
}

func acceptanceCriteriaForJobV0(jobType string) []string {
	criteria := []string{
		"devolver artifact_type=" + expectedArtifactTypeV0(jobType),
		"payload_json valido y trazable",
		"sin placeholders",
		"sin leer internals de OPES",
		"entrega en fichero unico bajo allowed_write_set",
	}
	if strings.TrimSpace(jobType) == "expand_topic_from_summary" {
		criteria = append(criteria, expansionAcceptanceCriteriaV0()...)
	}
	if isDocumentPlanJobTypeV0(jobType) {
		criteria = append(criteria, documentPlanAcceptanceCriteriaV0()...)
	}
	switch strings.TrimSpace(jobType) {
	case "research_exam_precedents", "research_exam_results", "research_related_administration_exams":
		criteria = append(criteria, examResearchAcceptanceCriteriaV0()...)
	case "generate_visual_asset":
		criteria = append(criteria, visualAssetAcceptanceCriteriaV0()...)
	case "generate_question_bank", "generate_topic_tests", "create_topic_tests":
		criteria = append(criteria, questionBankAcceptanceCriteriaV0()...)
	case "generate_html_site", "generate_local_html_site", "assemble_local_html_site":
		criteria = append(criteria, localHTMLSiteAcceptanceCriteriaV0()...)
	case "generate_audio_asset", "generate_topic_audio":
		criteria = append(criteria, topicAudioAcceptanceCriteriaV0()...)
	case "generate_tutor_assets", "configure_temario_tutor", "configure_temario_bots":
		criteria = append(criteria, tutorBotAcceptanceCriteriaV0()...)
	}
	return criteria
}

func examResearchAcceptanceCriteriaV0() []string {
	return []string{
		"buscar examenes, convocatorias, temarios y pruebas publicas de administraciones relacionadas con la OPE",
		"priorizar boletines oficiales, sedes administrativas, tribunales, institutos publicos y sindicatos con documentacion verificable",
		"devolver informe con URLs publicas, fecha de consulta, administracion, anio, cuerpo/categoria, coincidencia de epigrafes y utilidad editorial",
		"separar evidencia confirmada de inferencias; no inventar examenes ni preguntas",
	}
}

func visualAssetAcceptanceCriteriaV0() []string {
	return []string{
		"crear o especificar infografias utiles para el tema completo, no decorativas",
		"cubrir todos los temas o apartados marcados por el document_plan",
		"devolver assets o prompts trazables con placement_ref, texto alternativo y objetivo didactico",
		"si se usa Gemini, Claude, Codex u otro proveedor, Orquesta lo decide por rol; OPES solo recibe visual_asset",
	}
}

func questionBankAcceptanceCriteriaV0() []string {
	return []string{
		"crear banco de tests por tema con minimo configurable, por defecto 50 preguntas",
		"crear salida nueva separada por bank_id/test_id; no sobrescribir ni borrar bancos originales",
		"cada pregunta debe tener 4 opciones A, B, C y D, una sola respuesta correcta exacta y distractores plausibles de dificultad real",
		"evitar opciones ridiculas, obvias, mecanicas, descartables por longitud o repetidas por plantilla",
		"todo texto visible debe ser localizable/i18n: preguntas, opciones, explicaciones, titulos, feedback y mensajes",
		"incluir explicacion tutor con respuesta correcta, por que fallan las distractoras y donde repasar en el temario",
		"en temas comunes no mencionar TCAE si el contenido debe valer para cualquier OPE; en temas especificos TCAE no usar Auxiliar de Enfermeria como nombre principal",
		"entregar JSON por tema, HTML revisable por tema, index.html, metadata.json e informe Markdown",
		"ejecutar o pedir validacion estructural: temas esperados, 50 preguntas por tema en banco estandar, 4 opciones por pregunta y 1 correcta por pregunta",
		"ejecutar o pedir validacion de dificultad/proximidad y busqueda rg de patrones prohibidos; corregir sin tirar bancos aprovechables",
		"si el adaptador OPES/USO importa en Postgres local, exigir backup previo, SQL con DELETE acotado al patron del banco nuevo y verificacion de conteos",
		"separar banco privado de afiliados del HTML publico abierto",
	}
}

func localHTMLSiteAcceptanceCriteriaV0() []string {
	return []string{
		"crear HTML local operativo del temario completo antes de produccion",
		"usar logos USO y aspecto visual coherente con la web USO/TCAE promocion interna aportada por el adaptador OPES/USO",
		"integrar temas, infografias, tests visibles permitidos, tutor, audios por apartado y navegacion local",
		"validar HTML offline, enlaces relativos, assets locales, responsive movil y ausencia de rutas internas",
	}
}

func topicAudioAcceptanceCriteriaV0() []string {
	return []string{
		"derivar el audio desde el tema ensamblado aprobado o refs de paquete final",
		"crear audio por tema y por apartado/seccion cuando el tema este dividido en apartados",
		"devolver manifest de audio con idioma, formatos, duracion aproximada, mapa section_ref -> audio_ref y checksum o refs de artefactos",
		"mantener texto narrado trazable a secciones del tema sin inventar contenido nuevo",
		"revisar lectura de numeros romanos como numeros antes de TTS",
		"validar escucha/transcripcion automatica cuando el adaptador OPES lo soporte",
		"no incluir rutas locales, proveedor, GPU, modelo ni procesos internos en el payload publico",
	}
}

func tutorBotAcceptanceCriteriaV0() []string {
	return []string{
		"crear paquete de tutor y bots del temario para uso local antes de produccion",
		"incluir intents, prompts o configuracion opaca, mapa tema/apartado, fuentes permitidas y limites de respuesta",
		"el tutor debe explicar errores de test, proponer repaso, responder dudas por tema y no inventar fuera de fuentes",
		"devolver configuracion portable sin secretos, rutas internas, proveedor ni modelo fijado en OPES",
	}
}

func constraintsForJobV0() []string {
	return []string{
		"no inventar contenido",
		"no leer DB ni ficheros internos de OPES",
		"usar solo el paquete de dominio recibido",
		"si falta contexto obligatorio declarar bloqueo",
	}
}

func opesJobWriteSetV0(workKind string, safeJob string) string {
	return "external/opes/" + strings.TrimSpace(workKind) + "/" + strings.TrimSpace(safeJob)
}

func compactOPESBridgeRefV0(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == '.', r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "opes"
	}
	return out
}

func compactStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
