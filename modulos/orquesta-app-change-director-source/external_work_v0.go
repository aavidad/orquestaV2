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
