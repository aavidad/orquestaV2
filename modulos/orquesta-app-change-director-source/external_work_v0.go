package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func appChangeContractSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
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
		return "Resolver bloque documental externo"
	case "research_sources":
		return "Resolver investigacion externa"
	case "review_legal":
		return "Resolver revision legal externa"
	case "review_pedagogical":
		return "Resolver revision pedagogica externa"
	case "review_quality":
		return "Resolver revision de calidad externa"
	case "documentation":
		return "Resolver trabajo documental externo"
	case "generation":
		return "Resolver trabajo de generacion externo"
	case "review":
		return "Resolver revision externa"
	default:
		return "Resolver trabajo externo de app"
	}
}

func appChangeTaskSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
	if appChangeHasExternalWorkV0(request) {
		return "Coordinar agentes sobre contrato externo y devolver entrega verificable a la app propietaria."
	}
	return "Implementar solo el cambio aceptado."
}

func appChangeHasExternalWorkV0(request orquestaappchange.AppChangeRequestV0) bool {
	return request.ExternalWork != nil &&
		(request.ExternalWork.ProjectRef != "" ||
			request.ExternalWork.WorkKind != "" ||
			len(request.ExternalWork.InterfaceRefs) > 0 ||
			len(request.ExternalWork.WorkRefs) > 0)
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
