package orquestaopesbridge

import (
	"sort"
	"strings"
	"unicode"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	opesArtifactTypeLearningGamesPackageV0     = "learning_games_package"
	opesArtifactTypeHelpManualPackageV0        = "help_manual_package"
	opesArtifactTypeCompletedSyllabusPackageV0 = "completed_syllabus_package"
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
	switch strings.TrimSpace(jobType) {
	case "review_codex", "review_gemini", "review_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0
	case "review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0
	case "review_director_consolidation", "review_director_final":
		return orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0
	case "generate_agent_candidate_codex", "generate_agent_candidate_gemini",
		"generate_agent_candidate_claude", "generate_provider_candidate":
		return orquestadomainwork.DomainWorkArtifactTypeAgentCandidateV0
	case "vote_agent_candidates_codex", "vote_agent_candidates_gemini",
		"vote_agent_candidates_claude", "vote_provider_candidates":
		return orquestadomainwork.DomainWorkArtifactTypeAgentCandidateVoteV0
	case "select_provider_candidate_director":
		return orquestadomainwork.DomainWorkArtifactTypeAgentCandidateSelectV0
	case "generate_learning_games", "create_learning_games", "generate_course_games":
		return opesArtifactTypeLearningGamesPackageV0
	case "generate_help_manual_assets", "generate_help_manuals", "create_help_manuals":
		return opesArtifactTypeHelpManualPackageV0
	case "finalize_temario_package", "close_temario_package":
		return opesArtifactTypeCompletedSyllabusPackageV0
	}
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
		"generate_agent_candidate_codex",
		"generate_agent_candidate_gemini",
		"generate_agent_candidate_claude",
		"vote_agent_candidates_codex",
		"vote_agent_candidates_gemini",
		"vote_agent_candidates_claude",
		"select_agent_candidate_director",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"review_codex",
		"review_gemini",
		"review_claude",
		"review_pair_codex_gemini",
		"review_pair_codex_claude",
		"review_pair_gemini_claude",
		"review_director_consolidation",
		"validate_topic",
		"assemble_topic",
		"generate_html_site",
		"generate_audio_asset",
		"generate_topic_audio",
		"generate_tutor_assets",
		"generate_learning_games",
		"generate_help_manual_assets",
		"finalize_temario_package",
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
		"resolver marcadores editoriales visibles cuando existan; no bloquear ni descartar trabajo por busquedas literales de palabras",
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
	case "generate_agent_candidate_codex", "generate_agent_candidate_gemini",
		"generate_agent_candidate_claude":
		criteria = append(criteria, agentCandidateAcceptanceCriteriaV0(jobType)...)
	case "vote_agent_candidates_codex", "vote_agent_candidates_gemini",
		"vote_agent_candidates_claude":
		criteria = append(criteria, agentCandidateVoteAcceptanceCriteriaV0(jobType)...)
	case "select_agent_candidate_director", "select_provider_candidate_director":
		criteria = append(criteria, agentCandidateSelectionAcceptanceCriteriaV0()...)
	case "generate_html_site", "generate_local_html_site", "assemble_local_html_site":
		criteria = append(criteria, localHTMLSiteAcceptanceCriteriaV0()...)
	case "generate_audio_asset", "generate_topic_audio":
		criteria = append(criteria, topicAudioAcceptanceCriteriaV0()...)
	case "generate_tutor_assets", "configure_temario_tutor", "configure_temario_bots":
		criteria = append(criteria, tutorBotAcceptanceCriteriaV0()...)
	case "generate_learning_games", "create_learning_games", "generate_course_games":
		criteria = append(criteria, learningGamesAcceptanceCriteriaV0()...)
	case "generate_help_manual_assets", "generate_help_manuals", "create_help_manuals":
		criteria = append(criteria, helpManualPackageAcceptanceCriteriaV0()...)
	case "review_codex", "review_gemini", "review_claude":
		criteria = append(criteria, independentAgentReviewAcceptanceCriteriaV0(jobType)...)
	case "review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude":
		criteria = append(criteria, pairedAgentReviewAcceptanceCriteriaV0(jobType)...)
	case "review_director_consolidation", "review_director_final":
		criteria = append(criteria, directorConsolidationAcceptanceCriteriaV0()...)
	case "finalize_temario_package", "close_temario_package":
		criteria = append(criteria, finalizedTemarioPackageAcceptanceCriteriaV0()...)
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
		"integrar infografias con criterio editorial en puntos importantes, dificiles, comparativos o procedimentales donde aporten aprendizaje",
		"no saturar el tema con demasiadas infografias ni crear visuales de relleno; cada visual debe tener utilidad didactica clara",
		"cubrir los temas o apartados marcados por el document_plan, incluidos los criticos cuando proceda, sin limitarse solo a ellos",
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
		"usar guide_ref=" + opesTCAETestCreationGuideRefV0 + " como referencia de criterios de test cuando el temario sea TCAE o la composicion OPES no aporte una guia mas especifica; no exponer rutas locales en payloads publicos",
		"entregar JSON por tema, HTML revisable por tema, index.html, metadata.json e informe Markdown",
		"ejecutar o pedir validacion estructural: temas esperados, 50 preguntas por tema en banco estandar, 4 opciones por pregunta y 1 correcta por pregunta",
		"ejecutar o pedir validacion de dificultad/proximidad; corregir sin tirar bancos aprovechables ni parar por coincidencias literales reparables",
		"si el adaptador OPES/USO importa en Postgres local, exigir backup previo, SQL con DELETE acotado al patron del banco nuevo y verificacion de conteos",
		"separar banco privado de afiliados del HTML publico abierto",
	}
}

func localHTMLSiteAcceptanceCriteriaV0() []string {
	return []string{
		"crear HTML local operativo del temario completo antes de produccion",
		"usar el formato real de curso USO/TCAE aportado por el adaptador OPES/USO; no entregar visores single-file ni maquetas con estructura visual propia si existe plantilla web de curso",
		"materializar estructura de curso revisable: index.html, html_final por tema, assets locales, audio/manifests por tema y locales/i18n para controles visibles",
		"incluir la capa protegida tipo USO/TCAE cuando el curso sea material de estudio: #uso-material-watermark, marca diagonal visible y comportamiento coherente con la web de afiliados",
		"integrar temas, infografias, tests visibles permitidos, tutor, audios por apartado y navegacion local",
		"ubicar cada infografia junto al apartado o parrafo que explica; no agrupar varias infografias al inicio del tema salvo que sean mapa inicial justificado",
		"todo texto visible de interfaz debe ser localizable/i18n y reutilizar las claves o convenciones de la web USO cuando existan",
		"no mostrar al alumnado notas de generacion, reutilizacion de comunes, refs, manifests, trazabilidad tecnica, OPES, agentes, backend, staging ni decisiones internas; esa informacion queda en metadata o informes internos",
		"validar HTML offline, enlaces relativos, manifests de audio, assets locales, responsive movil y ausencia de rutas internas",
	}
}

func topicAudioAcceptanceCriteriaV0() []string {
	return []string{
		"derivar el audio desde el tema ensamblado aprobado o refs de paquete final",
		"antes de generar TTS resolver common_topic_ref, source_content_ref, audio_manifest_ref y audio_ref existentes; reutilizar audio comun compatible cuando exista",
		"si un comun no encaja exactamente, conservarlo como candidato y crear derivacion localizada; no regenerar ni tirar trabajo por alias, orden, titulo o metadatos reparables",
		"crear audio por tema y por apartado/seccion cuando el tema este dividido en apartados",
		"devolver manifest de audio con idioma, formatos, duracion aproximada, audio_ref global si aplica y segments con una entrada por apartado/seccion narrable",
		"cada segmento debe conservar section_ref, audio_ref, manifest_ref si aplica, duracion, hash/ref de texto y estado reused_common/generated/derived",
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

func learningGamesAcceptanceCriteriaV0() []string {
	return []string{
		"crear paquete modular de juegos y retos para el curso, sin tocar textos base ni tests originales",
		"incluir parejas, completa la frase, retos por tema y revision de errores cuando haya datos suficientes",
		"cada juego debe consumir temas, tests, errores y tutor por refs/manifests, no por SQL directo ni rutas internas",
		"incluir i18n, assets locales, manifest de registro y criterios de insercion en HTML",
		"si un juego no tiene datos suficientes, conservar plantilla/tarea derivada en vez de bloquear el paquete completo",
	}
}

func helpManualPackageAcceptanceCriteriaV0() []string {
	return []string{
		"crear paquete de manuales graficos de ayuda para USO cuando el temario o artefacto local ya tenga HTML revisable",
		"usar como referencia SCREENSHOT_HELP_MANUALS.md: YAML de escenario -> capturas/anotaciones -> index.html canonico -> manual.pdf exportado desde HTML -> manual.md",
		"aplicar regla de marca USO: logo en cabecera, marca de agua cuando proceda, pie con web, correo uso@dipgra.es, telefono si esta disponible y datos del sindicato",
		"mantener comentarios de campos fuera de la captura mediante notas numeradas; no tapar campos importantes con textos largos dentro de la imagen",
		"difuminar o tapar datos personales, claves, sesiones, cookies o tokens en capturas privadas; usar datos de prueba cuando sea posible",
		"entregar scenario.yaml, index.html, manual.pdf, manual.md, img anotadas, raw originales y reporte de revision visual",
		"validar abriendo HTML y PDF: imagenes ajustadas a A4, margenes correctos, marca visible, sin datos reales no autorizados y texto final apto para usuarios",
		"no exponer rutas locales ni perfiles privados en el payload publico; usar refs opacas para outputs revisables",
	}
}

func independentAgentReviewAcceptanceCriteriaV0(jobType string) []string {
	return []string{
		"emitir revision independiente del curso o artefacto asignado con rol=" + strings.TrimPrefix(strings.TrimSpace(jobType), "review_"),
		"revisar contenido, fuentes, tests, visuales, audios, tutor, HTML, manuales, i18n, accesibilidad y paquete segun el alcance recibido",
		"para bancos de test publicables, revisar el 100% de preguntas, opciones, respuesta correcta, distractores y explicaciones tutor; si hay limite externo, partir en lotes y conservar evidencia",
		"para visuales, revisar legibilidad interna del asset, textos dentro de cajas, placement_ref, alt text, utilidad didactica y ausencia de visuales decorativos",
		"para audios, comprobar manifest, MP3 por apartado cuando proceda, lectura de tablas/listas/esquemas y estado de QA o transcripcion",
		"clasificar hallazgos como aceptar, rework localizado, reutilizar como insumo o bloqueo real; no tirar trabajo recuperable por alias, formato reparable o palabra suelta",
		"devolver informe compacto con decision, issue_refs, evidence_refs, rework_refs y elementos aceptados",
	}
}

func agentCandidateAcceptanceCriteriaV0(jobType string) []string {
	provider := strings.TrimPrefix(strings.TrimSpace(jobType), "generate_agent_candidate_")
	return []string{
		"crear candidato alternativo de " + provider + " solo para la pieza asignada: tests, tutor, texto, visual, audio narrable o bloque concreto",
		"conservar y referenciar el artefacto original; no sobrescribir ni borrar candidatos de otros agentes",
		"explicar por que el candidato mejora el material existente: claridad, rigor, distractores, pedagogia, visual, accesibilidad o ajuste al programa",
		"devolver agent_candidate_artifact con candidate_ref, source_artifact_ref, target_part_ref, provider_role=" + provider + ", payload_ref, strengths, risks y evidence_refs",
		"si el candidato no supera al original, conservarlo como borrador o insumo y declararlo; no parar el ciclo por no ganar",
	}
}

func agentCandidateVoteAcceptanceCriteriaV0(jobType string) []string {
	voter := strings.TrimPrefix(strings.TrimSpace(jobType), "vote_agent_candidates_")
	return []string{
		"votar candidatos alternativos desde el criterio de " + voter + " sobre la misma pieza asignada",
		"comparar candidato Codex, candidato Gemini, candidato Claude y original cuando exista",
		"puntuar con evidencia: correccion, ajuste al temario, utilidad pedagogica, calidad de distractores o visual, claridad, mantenibilidad e integracion",
		"devolver agent_candidate_vote_report con ranked_candidate_refs, winner_ref recomendado, mergeable_parts, rejected_parts, risks y evidence_refs",
		"no bloquear por gustos de estilo o palabra suelta; si varias piezas aportan valor, proponer fusion y rework causal",
	}
}

func agentCandidateSelectionAcceptanceCriteriaV0() []string {
	return []string{
		"consolidar votos de Codex, Gemini y Claude sobre candidatos alternativos",
		"elegir ganador, fusionar partes aprovechables o pedir nuevo candidato acotado cuando no haya suficiente calidad",
		"mantener trazabilidad de original, candidatos, votos y decision final",
		"devolver agent_candidate_selection_matrix con selected_candidate_ref, merged_refs, discarded_as_draft_refs, rework_refs y evidence_refs",
		"solo el Director convierte un candidato en artefacto final; los demas quedan conservados como borrador o evidencia",
	}
}

func pairedAgentReviewAcceptanceCriteriaV0(jobType string) []string {
	return []string{
		"ejecutar revision por pares " + strings.TrimPrefix(strings.TrimSpace(jobType), "review_pair_") + " sobre el mismo paquete o artefacto",
		"comparar criterios de ambos roles y registrar acuerdos, desacuerdos, riesgos no vistos por una parte y decision propuesta",
		"en tests, contrastar pregunta por pregunta y opcion por opcion cuando el banco sea publicable; dividir por lotes si hace falta",
		"en visuales y HTML, contrastar captura/legibilidad/placement con criterio editorial y visual; no aceptar maquetas como arte final",
		"en audios, contrastar manifest, segmentos, comprensibilidad y correspondencia con el texto final visible",
		"proponer rework causal y acotado; conservar borradores, insumos y piezas recuperables antes de pedir rehacer",
		"devolver agent_pair_review_report con matriz de consenso, discrepancias, accepted_refs, rework_refs, blocked_refs y evidence_refs",
	}
}

func directorConsolidationAcceptanceCriteriaV0() []string {
	return []string{
		"consolidar revisiones independientes de Codex, Gemini y Claude y revisiones por pares Codex-Gemini, Codex-Claude y Gemini-Claude",
		"resolver discrepancias con decision del Director: aceptar, pedir rework, derivar tarea, conservar como insumo o bloquear por causa real",
		"verificar que no quedan P0/P1 abiertos en contenido, tests, visuales, audios, tutor, HTML, manuales, i18n, accesibilidad ni paquete",
		"confirmar que todo bloqueo restante es seguridad real, datos sensibles, causalidad rota, refs imposibles, efecto externo no autorizado o falta de entorno",
		"devolver director_review_matrix con decision final, rework_refs, accepted_artifact_refs y evidence_refs",
	}
}

func finalizedTemarioPackageAcceptanceCriteriaV0() []string {
	return []string{
		"entregar paquete de temario terminado al 100% para revision local del operador antes de produccion",
		"incluir temario resumido y ampliado separados, fuentes, exam_research_report, visuales finales, question_bank, audios por apartado, tutor/bots, HTML local, manuales graficos y manifest de trazabilidad",
		"incorporar solo artefactos aceptados por Director o marcados como recuperables en ubicacion interna; no mostrar trazabilidad tecnica al alumnado",
		"verificar HTML local, responsive, enlaces, assets comprimidos, audio/manifests, locales/i18n, watermark USO cuando proceda y ausencia de rutas internas",
		"exigir triple visto bueno y revisiones por pares cerradas; si falta una revision obligatoria, estado pendiente_continuar, no ready",
		"devolver completed_syllabus_package con package_ref, manifest_ref, checksum_refs, validation_report_ref, review_matrix_ref y estado listo_para_revision_operador",
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
