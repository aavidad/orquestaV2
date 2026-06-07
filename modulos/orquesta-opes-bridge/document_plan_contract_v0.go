package orquestaopesbridge

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func withOPESGlobalEditorialPolicyFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if fieldHasNameV0(fields, "opes_global_editorial_policy_2026_05_18") {
		return fields
	}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields)+1)
	out = append(out, orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_global_editorial_policy_2026_05_18",
		Values: opesGlobalEditorialPolicyV0(),
	})
	out = append(out, fields...)
	return out
}

func withOPESHTMLPublicationPolicyFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if fieldHasNameV0(fields, "opes_html_publication_policy_2026_05_19") {
		return fields
	}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields)+1)
	out = append(out, orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_html_publication_policy_2026_05_19",
		Values: opesHTMLPublicationPolicyV0(),
	})
	out = append(out, fields...)
	return out
}

func withOPESHTMLTopicTemplateFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if fieldHasNameV0(fields, "opes_html_topic_template_v1") {
		return fields
	}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields)+1)
	out = append(out, orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_html_topic_template_v1",
		Values: opesHTMLTopicTemplateV1(),
	})
	out = append(out, fields...)
	return out
}

func withOPESTemarioAgentRulesFieldV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	if fieldHasNameV0(fields, "opes_temario_agent_rules_2026_06_04") {
		return fields
	}
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields)+1)
	out = append(out, orquestadomainwork.DomainWorkFieldV0{
		Name:   "opes_temario_agent_rules_2026_06_04",
		Values: opesTemarioAgentRulesV0(),
	})
	out = append(out, fields...)
	return out
}

func OPESGlobalEditorialPolicyV0() []string {
	return append([]string(nil), opesGlobalEditorialPolicyV0()...)
}

func OPESHTMLPublicationPolicyV0() []string {
	return append([]string(nil), opesHTMLPublicationPolicyV0()...)
}

func OPESHTMLTopicTemplateV1() []string {
	return append([]string(nil), opesHTMLTopicTemplateV1()...)
}

func OPESTemarioAgentRulesV0() []string {
	return append([]string(nil), opesTemarioAgentRulesV0()...)
}

func appendDocumentPlanContractFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	jobType string,
) []orquestadomainwork.DomainWorkFieldV0 {
	if !isDocumentPlanJobTypeV0(jobType) {
		return fields
	}
	if !fieldHasNameV0(fields, "expected_schema") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:  "expected_schema",
			Value: orquestadomainwork.DomainDocumentPlanSchemaV0,
		})
	}
	if !fieldHasNameV0(fields, "required_plan_parts") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_plan_parts",
			Values: documentPlanRequiredPartsV0(),
		})
	}
	if !fieldHasNameV0(fields, "allowed_document_plan_work_kinds") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "allowed_document_plan_work_kinds",
			Values: documentPlanAllowedWorkKindsV0(),
		})
	}
	if !fieldHasNameV0(fields, "minimum_quality_gates") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "minimum_quality_gates",
			Values: documentPlanQualityGatesV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_editorial_workflow") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_editorial_workflow",
			Values: documentPlanOPESEditorialWorkflowV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_level_derivation_policy") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_level_derivation_policy",
			Values: documentPlanOPESLevelDerivationPolicyV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_assimilation_method") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_assimilation_method",
			Values: documentPlanOPESAssimilationMethodV0(),
		})
	}
	if !fieldHasNameV0(fields, "opes_quality_requirements") {
		fields = append(fields, orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_quality_requirements",
			Values: documentPlanOPESQualityRequirementsV0(),
		})
	}
	return fields
}

func isDocumentPlanJobTypeV0(jobType string) bool {
	switch strings.TrimSpace(jobType) {
	case orquestadomainwork.DomainWorkKindPlanDocumentV0,
		orquestadomainwork.DomainWorkKindPlanTopicV0,
		orquestadomainwork.DomainWorkKindPlanSyllabusV0:
		return true
	default:
		return false
	}
}

func documentPlanRequiredPartsV0() []string {
	return []string{
		"sections",
		"deliverables",
		"quality_criteria",
		"review_steps",
		"visuals_when_useful",
	}
}

func documentPlanAllowedWorkKindsV0() []string {
	return []string{
		"root: plan_documento|plan_tema|plan_temario",
		"research: research_exam_precedents|research_exam_results|research_related_administration_exams",
		"sections: draft_content_block",
		"visuals: generate_visual_asset",
		"tests: generate_question_bank|generate_topic_tests|create_topic_tests",
		"agent_candidates: generate_agent_candidate_codex|generate_agent_candidate_gemini|generate_agent_candidate_claude",
		"candidate_votes: vote_agent_candidates_codex|vote_agent_candidates_gemini|vote_agent_candidates_claude|select_agent_candidate_director",
		"review_steps: review_legal|review_pedagogical|review_quality|review_codex|review_gemini|review_claude|review_pair_codex_gemini|review_pair_codex_claude|review_pair_gemini_claude|review_director_consolidation|validate_topic",
		"assembly: assemble_topic|generate_audio_asset|generate_tutor_assets|generate_learning_games|generate_html_site|generate_help_manual_assets",
		"closure: finalize_temario_package|close_temario_package",
	}
}

func documentPlanQualityGatesV0() []string {
	return []string{
		"schema_version=domain_document_plan.v0",
		"artifact_type=document_plan",
		"work_kind raiz debe coincidir con el job_type recibido",
		"raiz DomainDocumentPlanV0: plan_ref, domain_ref, work_kind, document_kind, language_code, title, objective, estimated_pages_min, estimated_pages_max",
		"sections DomainDocumentPlanSectionV0: section_ref, order, title, objective, work_kind, target_words_min, target_words_max, required_elements, acceptance_criteria",
		"visuals DomainDocumentPlanVisualV0: visual_ref, visual_type, placement_ref, objective, work_kind",
		"review_steps DomainDocumentPlanReviewV0: review_ref, order, work_kind, objective",
		"work_kind de sections/visuals/review_steps debe usar allowed_document_plan_work_kinds",
		"deliverables DomainDocumentPlanDeliverableV0: deliverable_ref, artifact_type, title, required",
		"sections ejecutables con work_kind y acceptance_criteria",
		"deliverables obligatorios declarados",
		"plan_temario OPES debe incluir investigacion de examenes relacionados, redaccion, infografias por tema, banco de tests, revisiones independientes Codex/Gemini/Claude, revisiones por pares Codex-Gemini/Codex-Claude/Gemini-Claude, candidatos y votacion solo cuando una pieza concreta sea floja, consolidacion del Director, ensamblado, audio por apartado tras texto aprobado, tutor/bots, juegos/retos/revision de errores, HTML local operativo, manuales graficos de ayuda USO y paquete final 100% antes de produccion",
		"sin redactar contenido final en la planificacion",
		"quality_criteria/constraints deben incorporar politica editorial OPES cuando domain_ref=opes",
	}
}

func documentPlanAcceptanceCriteriaV0() []string {
	return []string{
		"devolver DomainDocumentPlanV0 valido",
		"incluir sections y deliverables obligatorios",
		"incluir quality_criteria y review_steps aplicables",
		"para OPES, incluir research_exam_precedents para buscar examenes, convocatorias y temarios de administraciones relacionadas por internet usando fuentes publicas verificables",
		"para OPES, incluir generate_visual_asset para crear infografias utiles y no excesivas en puntos importantes, dificiles, comparativos o procedimentales donde aporten aprendizaje, incluidos apartados criticos cuando proceda",
		"para OPES, incluir generate_question_bank y deliverable question_bank para tests por tema",
		"para OPES, incluir generate_audio_asset y deliverable audio_asset para audio accesible por tema y por apartado/seccion",
		"para OPES, incluir generate_tutor_assets para tutor y bots del temario",
		"para OPES, incluir generate_learning_games y deliverable learning_games_package para juegos, retos y revision de errores como modulos reutilizables del curso",
		"para OPES, incluir generate_html_site y deliverable local_html_site para HTML local operativo con logos USO, aspecto USO/TCAE promocion interna y capa protegida #uso-material-watermark antes de subir a produccion",
		"para OPES, incluir generate_help_manual_assets y deliverable help_manual_package para manuales graficos de ayuda USO derivados del HTML local, con YAML, capturas, index.html, manual.pdf y manual.md",
		"para OPES, incluir review_codex, review_gemini y review_claude como revisiones independientes obligatorias del curso y de los tests publicables",
		"para OPES, incluir review_pair_codex_gemini, review_pair_codex_claude y review_pair_gemini_claude como revisiones por pares con matriz de acuerdos, discrepancias y rework causal",
		"para OPES, cuando un banco de tests, tutor, texto, visual o pieza concreta sea flojo, permitir generate_agent_candidate_codex, generate_agent_candidate_gemini y generate_agent_candidate_claude para crear candidatos alternativos y conservarlos todos",
		"para OPES, si existen candidatos alternativos, incluir vote_agent_candidates_codex, vote_agent_candidates_gemini y vote_agent_candidates_claude para que los tres agentes voten el mejor candidato con evidencia antes de que el Director seleccione o fusione",
		"para OPES, incluir select_agent_candidate_director cuando existan candidatos alternativos; el Director elige ganador, fusiona piezas aprovechables o pide rework causal",
		"para OPES, incluir review_director_consolidation para consolidar revisiones, aceptar artefactos, pedir rework localizado y conservar material recuperable",
		"para OPES, incluir finalize_temario_package y deliverable completed_syllabus_package para entregar el temario terminado al 100% antes de produccion",
		"no redactar el documento final dentro del plan",
		"para OPES, respetar flujo editorial: inventario, investigacion externa, agrupacion, mapa de dependencias, temas maestros, derivacion por nivel, redaccion, infografias, tests, revision independiente, revision por pares, candidatos alternativos y votacion triple solo si hay material flojo, consolidacion del Director, ensamblado, audio tras texto aprobado, tutor/bots, juegos/retos/revision de errores, HTML local publicable, manuales de ayuda y cierre 100%",
		"si existe maestro A1/A2 o A1 equivalente, planificar primero ese maestro y despues derivar B/C1/C2/AP por resumen, reduccion editorial y adaptacion de nivel",
		"si no existe equivalente superior, marcar creacion_directa_nivel en criterios, constraints o secciones",
		"aplicar metodo OPES de asimilacion: recuperacion activa, repaso espaciado, ejemplos trabajados, carga cognitiva controlada, visuales utiles, elaboracion e intercalado",
	}
}
