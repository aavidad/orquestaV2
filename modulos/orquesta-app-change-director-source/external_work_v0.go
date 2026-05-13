package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func appChangeContractSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
	if appChangeIsDraftContentBlockWorkV0(request) {
		return "Contrato para redactar bloque documental externo con paquete de dominio suficiente."
	}
	if appChangeIsDocumentaryExternalWorkV0(request) {
		return "Contrato para resolver trabajo documental externo con paquete de dominio suficiente."
	}
	if appChangeHasExternalWorkV0(request) {
		return "Contrato para resolver trabajo externo de dominio sin acoplar Orquesta a la app."
	}
	return "Contrato para aplicar cambio aislado."
}

func appChangeFunctionNamesV0(request orquestaappchange.AppChangeRequestV0) []string {
	if appChangeHasExternalWorkV0(request) {
		return []string{"ApplyExternalDomainWorkV0"}
	}
	return []string{"ApplyAppChangeV0"}
}

func appChangeTaskTitleV0(request orquestaappchange.AppChangeRequestV0) string {
	if !appChangeHasExternalWorkV0(request) {
		return "Aplicar cambio de app"
	}
	switch strings.TrimSpace(request.ExternalWork.WorkKind) {
	case "draft_content_block":
		return "Redactar bloque documental " + appChangeExternalWorkTitleScopeV0(request.ExternalWork)
	case "summarize_block":
		return "Resumir bloque documental externo"
	case "summarize_chapter":
		return "Resumir capitulo documental externo"
	case "summarize_topic":
		return "Resumir tema documental externo"
	case "create_exam_outline":
		return "Crear esquema de examen externo"
	case "research_sources":
		return "Resolver investigacion externa"
	case "split_syllabus_topic":
		return "Dividir tema de temario externo"
	case "draft_topic_outline":
		return "Preparar esquema de tema externo"
	case "review_legal":
		return "Resolver revision legal externa"
	case "review_pedagogical":
		return "Resolver revision pedagogica externa"
	case "review_quality":
		return "Resolver revision de calidad externa"
	case "documentation":
		return "Resolver trabajo documental " + appChangeExternalWorkTitleScopeV0(request.ExternalWork)
	case "generation":
		return "Resolver trabajo de generacion externo"
	case "review":
		return "Resolver revision externa"
	default:
		return "Resolver trabajo externo de app"
	}
}

func appChangeTaskSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
	if appChangeIsDraftContentBlockWorkV0(request) {
		return "Redactar unidad editorial amplia con paquete de dominio suficiente; no trocear un temario largo en parrafos sin continuidad."
	}
	if appChangeIsSummaryExternalWorkV0(request) {
		return "Crear resumen derivado compacto con trazabilidad a bloques, capitulos, tema y fuentes de origen."
	}
	if appChangeIsDocumentaryExternalWorkV0(request) {
		return "Resolver trabajo documental con paquete de dominio suficiente: temario, esquema, objetivo, fuentes, criterios y longitud si llegan."
	}
	if appChangeHasExternalWorkV0(request) {
		return "Coordinar agentes sobre contrato externo y devolver entrega verificable a la app propietaria."
	}
	return "Implementar solo el cambio aceptado."
}

func appChangeTaskCriteriaV0(request orquestaappchange.AppChangeRequestV0) []string {
	criteria := []string{"Mantener arquitectura hexagonal e i18n si aplica."}
	criteria = append(criteria, appChangeExternalWorkCriteriaV0(request)...)
	return append(criteria, request.AcceptanceCriteria...)
}

func appChangeExternalWorkCriteriaV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	if !appChangeHasExternalWorkV0(request) {
		return nil
	}
	criteria := []string{"Resolver solo el contrato externo de dominio con refs opacas."}
	if !appChangeIsDocumentaryExternalWorkV0(request) {
		return criteria
	}
	scope := appChangeExternalWorkTitleScopeV0(request.ExternalWork)
	criteria = append(criteria,
		"Tratar el paquete de dominio "+scope+" como entrada suficiente, no como contexto minimo.",
	)
	if appChangeIsDraftContentBlockWorkV0(request) {
		criteria = append(criteria,
			"Para draft_content_block, usar paquete editorial suficiente y no sobreatomizar: bloque, subcapitulo o capitulo coherente si OPES lo envio asi.",
			"No redactar un tema de 50 folios sin paquete suficiente; pedir division editorial a OPES si excede contexto o trazabilidad.",
		)
	} else if appChangeIsSummaryExternalWorkV0(request) {
		criteria = append(criteria,
			"Para summarize_* y create_exam_outline, aceptar granularidad pequena porque son artefactos derivados y trazables.",
			"Conservar matices, excepciones, plazos, organos, fuentes criticas y advertencias de examen del material de origen.",
		)
	} else {
		criteria = append(criteria,
			"Usar temario, esquema, objetivo, fuentes, criterios y longitud si llegan en input_fields.",
		)
	}
	return append(criteria,
		"Si falta un campo de input_fields requerido por el job, declararlo como bloqueo de dominio y no inventarlo.",
	)
}

func appChangeIsDraftContentBlockWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return appChangeHasExternalWorkV0(request) &&
		strings.TrimSpace(request.ExternalWork.WorkKind) == "draft_content_block"
}

func appChangeIsDocumentaryExternalWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	if !appChangeHasExternalWorkV0(request) {
		return false
	}
	switch strings.TrimSpace(request.ExternalWork.WorkKind) {
	case "draft_content_block",
		"documentation",
		"research_sources",
		"split_syllabus_topic",
		"draft_topic_outline",
		"summarize_block",
		"summarize_chapter",
		"summarize_topic",
		"create_exam_outline",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"validate_topic",
		"assemble_topic",
		"export_topic",
		"verify_sources":
		return true
	default:
		return false
	}
}

func appChangeIsSummaryExternalWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	if !appChangeHasExternalWorkV0(request) {
		return false
	}
	switch strings.TrimSpace(request.ExternalWork.WorkKind) {
	case "summarize_block", "summarize_chapter", "summarize_topic", "create_exam_outline":
		return true
	default:
		return false
	}
}

func appChangeExternalWorkTitleScopeV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) string {
	if work == nil {
		return "externo"
	}
	project := appChangeScopePartV0(work.ProjectRef)
	project = strings.TrimPrefix(project, "project-ref-")
	project = strings.TrimPrefix(project, "project-")
	if project == "" || strings.Contains(project, "-") || len(project) > 12 {
		return "externo"
	}
	return strings.ToUpper(project)
}

func appChangeHasExternalWorkV0(request orquestaappchange.AppChangeRequestV0) bool {
	return request.ExternalWork != nil &&
		(request.ExternalWork.ProjectRef != "" ||
			request.ExternalWork.JobRef != "" ||
			request.ExternalWork.WorkKind != "" ||
			len(request.ExternalWork.InterfaceRefs) > 0 ||
			len(request.ExternalWork.WorkRefs) > 0 ||
			len(request.ExternalWork.InputFields) > 0)
}

func appChangeTaskWriteSetV0(request orquestaappchange.AppChangeRequestV0) []string {
	if len(request.AllowedWriteSet) > 0 {
		return append([]string(nil), request.AllowedWriteSet...)
	}
	if !appChangeHasExternalWorkV0(request) {
		return nil
	}
	return appChangeExternalWorkScopesV0(request.ExternalWork)
}

func appChangeExternalWorkScopesV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) []string {
	if work == nil {
		return nil
	}
	project := appChangeScopePartV0(work.ProjectRef)
	if project == "" {
		project = "external"
	}
	scopes := []string{}
	if kind := appChangeScopePartV0(work.WorkKind); kind != "" {
		scopes = append(scopes, "external/"+project+"/"+kind)
	}
	if job := appChangeScopePartV0(work.JobRef); job != "" {
		scopes = append(scopes, "external/"+project+"/"+job)
	}
	for _, ref := range work.WorkRefs {
		part := appChangeScopePartV0(ref)
		if part == "" {
			continue
		}
		scopes = append(scopes, "external/"+project+"/"+part)
	}
	return compactAppChangeSourceRefsV0(scopes)
}

func appChangeScopePartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(
		" ", "-",
		"\t", "-",
		"\n", "-",
		"\r", "-",
		"/", "-",
		"\\", "-",
	).Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return ""
	}
	return value
}
