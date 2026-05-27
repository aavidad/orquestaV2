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

func OPESGlobalEditorialPolicyV0() []string {
	return append([]string(nil), opesGlobalEditorialPolicyV0()...)
}

func OPESHTMLPublicationPolicyV0() []string {
	return append([]string(nil), opesHTMLPublicationPolicyV0()...)
}

func OPESHTMLTopicTemplateV1() []string {
	return append([]string(nil), opesHTMLTopicTemplateV1()...)
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
		"sections: draft_content_block",
		"visuals: generate_visual_asset",
		"review_steps: review_legal|review_pedagogical|review_quality|validate_topic|assemble_topic",
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
		"sin redactar contenido final en la planificacion",
		"quality_criteria/constraints deben incorporar politica editorial OPES cuando domain_ref=opes",
	}
}

func documentPlanAcceptanceCriteriaV0() []string {
	return []string{
		"devolver DomainDocumentPlanV0 valido",
		"incluir sections y deliverables obligatorios",
		"incluir quality_criteria y review_steps aplicables",
		"no redactar el documento final dentro del plan",
		"para OPES, respetar flujo editorial: inventario, agrupacion, mapa de dependencias, temas maestros, derivacion por nivel, revision y HTML publicable",
		"si existe maestro A1/A2 o A1 equivalente, planificar primero ese maestro y despues derivar B/C1/C2/AP por resumen, reduccion editorial y adaptacion de nivel",
		"si no existe equivalente superior, marcar creacion_directa_nivel en criterios, constraints o secciones",
		"aplicar metodo OPES de asimilacion: recuperacion activa, repaso espaciado, ejemplos trabajados, carga cognitiva controlada, visuales utiles, elaboracion e intercalado",
	}
}
