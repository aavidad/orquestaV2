package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

const maxAppChangeTaskCriteriaV0 = 10

type appChangeExternalWorkKindClassV0 string

const (
	appChangeExternalWorkKindGenericV0           appChangeExternalWorkKindClassV0 = "generic"
	appChangeExternalWorkKindDocumentaryV0       appChangeExternalWorkKindClassV0 = "documentary"
	appChangeExternalWorkKindDraftContentBlockV0 appChangeExternalWorkKindClassV0 = "draft_content_block"
	appChangeExternalWorkKindSummaryV0           appChangeExternalWorkKindClassV0 = "summary"
	appChangeExternalWorkKindExpansionV0         appChangeExternalWorkKindClassV0 = "expansion"
	appChangeExternalWorkKindDocumentPlanV0      appChangeExternalWorkKindClassV0 = "document_plan"
	appChangeExternalWorkKindVisualV0            appChangeExternalWorkKindClassV0 = "visual"
)

type appChangeExternalWorkKindRuleV0 struct {
	Class             appChangeExternalWorkKindClassV0
	Title             string
	ScopedTitlePrefix string
}

// Known external domain-work aliases. Keeping them in one internal table avoids
// spreading documentary source rules through generic app-change projection code.
var appChangeExternalWorkKindRulesV0 = map[string]appChangeExternalWorkKindRuleV0{
	"draft_content_block":       {Class: appChangeExternalWorkKindDraftContentBlockV0, ScopedTitlePrefix: "Redactar bloque documental "},
	"generate_visual_asset":     {Class: appChangeExternalWorkKindVisualV0, ScopedTitlePrefix: "Generar recurso visual "},
	"summarize_block":           {Class: appChangeExternalWorkKindSummaryV0, Title: "Resumir bloque documental externo"},
	"summarize_chapter":         {Class: appChangeExternalWorkKindSummaryV0, Title: "Resumir capitulo documental externo"},
	"summarize_topic":           {Class: appChangeExternalWorkKindSummaryV0, Title: "Resumir tema documental externo"},
	"expand_topic_from_summary": {Class: appChangeExternalWorkKindExpansionV0, Title: "Ampliar tema documental externo"},
	"plan_documento":            {Class: appChangeExternalWorkKindDocumentPlanV0, Title: "Planificar documento externo"},
	"plan_tema":                 {Class: appChangeExternalWorkKindDocumentPlanV0, Title: "Planificar tema documental externo"},
	"plan_temario":              {Class: appChangeExternalWorkKindDocumentPlanV0, Title: "Planificar temario externo"},
	"create_exam_outline":       {Class: appChangeExternalWorkKindSummaryV0, Title: "Crear esquema de examen externo"},
	"research_sources":          {Class: appChangeExternalWorkKindDocumentaryV0, Title: "Resolver investigacion externa"},
	"split_syllabus_topic":      {Class: appChangeExternalWorkKindDocumentaryV0, Title: "Dividir tema de temario externo"},
	"draft_topic_outline":       {Class: appChangeExternalWorkKindDocumentaryV0, Title: "Preparar esquema de tema externo"},
	"review_legal":              {Class: appChangeExternalWorkKindDocumentaryV0, Title: "Resolver revision legal externa"},
	"review_pedagogical":        {Class: appChangeExternalWorkKindDocumentaryV0, Title: "Resolver revision pedagogica externa"},
	"review_quality":            {Class: appChangeExternalWorkKindDocumentaryV0, Title: "Resolver revision de calidad externa"},
	"validate_topic":            {Class: appChangeExternalWorkKindDocumentaryV0},
	"assemble_topic":            {Class: appChangeExternalWorkKindDocumentaryV0},
	"export_topic":              {Class: appChangeExternalWorkKindDocumentaryV0},
	"verify_sources":            {Class: appChangeExternalWorkKindDocumentaryV0},
	"documentation":             {Class: appChangeExternalWorkKindDocumentaryV0, ScopedTitlePrefix: "Resolver trabajo documental "},
	"generation":                {Class: appChangeExternalWorkKindGenericV0, Title: "Resolver trabajo de generacion externo"},
	"review":                    {Class: appChangeExternalWorkKindGenericV0, Title: "Resolver revision externa"},
}

func appChangeContractSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
	if appChangeIsDraftContentBlockWorkV0(request) {
		return "Contrato para redactar bloque documental externo con paquete de dominio suficiente."
	}
	if appChangeIsVisualExternalWorkV0(request) {
		return "Contrato para generar recurso visual pedagogico externo con entrega auditable."
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

func appChangeTaskWorkProfileKindV0(request orquestaappchange.AppChangeRequestV0) string {
	if appChangeHasExternalWorkV0(request) {
		return "domain_work"
	}
	return "implementation"
}

func appChangeTaskTitleV0(request orquestaappchange.AppChangeRequestV0) string {
	if !appChangeHasExternalWorkV0(request) {
		return "Aplicar cambio de app"
	}
	if rule, ok := appChangeExternalWorkKindRuleForRequestV0(request); ok {
		if rule.ScopedTitlePrefix != "" {
			return rule.ScopedTitlePrefix + appChangeExternalWorkTitleScopeV0(request.ExternalWork)
		}
		if rule.Title != "" {
			return rule.Title
		}
	}
	return "Resolver trabajo externo de app"
}

func appChangeTaskSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
	if appChangeIsDraftContentBlockWorkV0(request) {
		return "Redactar unidad editorial amplia con paquete de dominio suficiente; no trocear un temario largo en parrafos sin continuidad."
	}
	if appChangeIsVisualExternalWorkV0(request) {
		return appChangeVisualTaskSummaryV0()
	}
	if appChangeIsSummaryExternalWorkV0(request) {
		return "Crear resumen derivado compacto con trazabilidad a bloques, capitulos, tema y fuentes de origen."
	}
	if appChangeIsExpansionExternalWorkV0(request) {
		return "Ampliar tema documental completo desde resumen trazable y paquete de dominio suficiente."
	}
	if appChangeIsDocumentPlanWorkV0(request) {
		return "Crear plan documental validable; no redactar ni ensamblar el documento final en esta tarea."
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
	criteria = append(criteria, request.AcceptanceCriteria...)
	return compactAppChangeTaskCriteriaV0(criteria)
}

func compactAppChangeTaskCriteriaV0(criteria []string) []string {
	criteria = compactAppChangeSourceRefsV0(criteria)
	criteria = sanitizeAppChangeTaskCriteriaV0(criteria)
	if len(criteria) <= maxAppChangeTaskCriteriaV0 {
		return criteria
	}
	return append([]string(nil), criteria[:maxAppChangeTaskCriteriaV0]...)
}

func sanitizeAppChangeTaskCriteriaV0(criteria []string) []string {
	out := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		sanitized := strings.TrimSpace(sanitizeAppChangeTaskCriterionV0(criterion))
		if sanitized != "" {
			out = append(out, sanitized)
		}
	}
	return out
}

func sanitizeAppChangeTaskCriterionV0(value string) string {
	value = sanitizeAppChangeOperationalTermsV0(value)
	replacer := strings.NewReplacer(
		"base de datos", "almacen interno",
		"Base de datos", "Almacen interno",
		"BASE DE DATOS", "ALMACEN INTERNO",
		"database", "almacen interno",
		"Database", "Almacen interno",
		"DATABASE", "ALMACEN INTERNO",
		"DB", "almacen interno",
		"db", "almacen interno",
		"SQL", "consulta interna",
		"sql", "consulta interna",
		"prompt", "instrucciones",
		"Prompt", "Instrucciones",
		"transcript", "registro externo",
		"Transcript", "Registro externo",
		"runtime", "ejecucion interna",
		"Runtime", "Ejecucion interna",
		"provider", "adaptador",
		"Provider", "Adaptador",
		"proveedor", "adaptador",
		"Proveedor", "Adaptador",
		"model", "capacidad",
		"Model", "Capacidad",
		"modelo", "capacidad",
		"Modelo", "Capacidad",
		"Codex", "agente externo",
		"codex", "agente externo",
		"Claude", "agente externo",
		"claude", "agente externo",
		"Gemini", "agente externo",
		"gemini", "agente externo",
		"Ollama", "agente externo",
		"ollama", "agente externo",
		"vLLM", "agente externo",
		"vllm", "agente externo",
		"HOME", "directorio interno",
		"home", "directorio interno",
		"OAuth", "identidad externa",
		"oauth", "identidad externa",
		"Docker", "contenedor",
		"docker", "contenedor",
		"tmux", "multiplexor externo",
		"secret", "dato sensible",
		"Secret", "Dato sensible",
		"secreto", "dato sensible",
		"Secreto", "Dato sensible",
		"token", "dato sensible",
		"Token", "Dato sensible",
		"password", "dato sensible",
		"Password", "Dato sensible",
		"credential", "dato sensible",
		"Credential", "Dato sensible",
		"credencial", "dato sensible",
		"Credencial", "Dato sensible",
		"api_key", "clave externa",
		"API_KEY", "clave externa",
	)
	return replacer.Replace(value)
}

func sanitizeAppChangeOperationalTermsV0(value string) string {
	replacer := strings.NewReplacer(
		"token economy", "economia de contexto",
		"Token economy", "Economia de contexto",
		"TOKEN ECONOMY", "ECONOMIA DE CONTEXTO",
		"economia de tokens", "economia de contexto",
		"Economia de tokens", "Economia de contexto",
		"ECONOMIA DE TOKENS", "ECONOMIA DE CONTEXTO",
	)
	return replacer.Replace(value)
}

func appChangeExternalWorkCriteriaV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	if !appChangeHasExternalWorkV0(request) {
		return nil
	}
	criteria := []string{"Resolver solo el contrato externo de dominio con refs opacas."}
	if appChangeIsVisualExternalWorkV0(request) {
		return append(criteria, appChangeVisualWorkCriteriaV0(request)...)
	}
	if !appChangeIsDocumentaryExternalWorkV0(request) {
		return criteria
	}
	scope := appChangeExternalWorkTitleScopeV0(request.ExternalWork)
	criteria = append(criteria,
		"Tratar el paquete de dominio "+scope+" como entrada suficiente, no como contexto minimo.",
	)
	if appChangeIsDraftContentBlockWorkV0(request) {
		owner := appChangeExternalWorkOwnerLabelV0(request.ExternalWork)
		criteria = append(criteria,
			"Para draft_content_block, usar paquete editorial suficiente y no sobreatomizar: bloque, subcapitulo o capitulo coherente si "+owner+" lo envio asi.",
			"No redactar un tema de 50 folios sin paquete suficiente; pedir division editorial a "+owner+" si excede contexto o trazabilidad.",
		)
	} else if appChangeIsDocumentPlanWorkV0(request) {
		criteria = append(criteria,
			"Devolver artifact_type=document_plan compatible con DomainDocumentPlanV0.",
			"Incluir sections, deliverables, quality_criteria y review_steps suficientes para ejecutar despues.",
			"Planificar visuales, revisiones, ensamblado y exportacion cuando el objetivo lo requiera.",
			"No redactar el documento final en esta tarea; solo plan verificable.",
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
	return appChangeExternalWorkKindHasClassV0(
		request,
		appChangeExternalWorkKindDraftContentBlockV0,
	)
}

func appChangeIsDocumentaryExternalWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	rule, ok := appChangeExternalWorkKindRuleForRequestV0(request)
	return ok && rule.isDocumentaryV0()
}

func appChangeIsExpansionExternalWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return appChangeExternalWorkKindHasClassV0(
		request,
		appChangeExternalWorkKindExpansionV0,
	)
}

func appChangeIsDocumentPlanWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return appChangeExternalWorkKindHasClassV0(
		request,
		appChangeExternalWorkKindDocumentPlanV0,
	)
}

func appChangeIsSummaryExternalWorkV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return appChangeExternalWorkKindHasClassV0(
		request,
		appChangeExternalWorkKindSummaryV0,
	)
}

func appChangeExternalWorkKindHasClassV0(
	request orquestaappchange.AppChangeRequestV0,
	class appChangeExternalWorkKindClassV0,
) bool {
	rule, ok := appChangeExternalWorkKindRuleForRequestV0(request)
	return ok && rule.Class == class
}

func appChangeExternalWorkKindRuleForRequestV0(
	request orquestaappchange.AppChangeRequestV0,
) (appChangeExternalWorkKindRuleV0, bool) {
	if !appChangeHasExternalWorkV0(request) {
		return appChangeExternalWorkKindRuleV0{}, false
	}
	rule, ok := appChangeExternalWorkKindRulesV0[strings.TrimSpace(request.ExternalWork.WorkKind)]
	return rule, ok
}

func (rule appChangeExternalWorkKindRuleV0) isDocumentaryV0() bool {
	switch rule.Class {
	case appChangeExternalWorkKindDocumentaryV0,
		appChangeExternalWorkKindDraftContentBlockV0,
		appChangeExternalWorkKindSummaryV0,
		appChangeExternalWorkKindExpansionV0,
		appChangeExternalWorkKindDocumentPlanV0:
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

func appChangeExternalWorkOwnerLabelV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) string {
	scope := appChangeExternalWorkTitleScopeV0(work)
	if scope == "externo" {
		return "la app externa"
	}
	return scope
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
