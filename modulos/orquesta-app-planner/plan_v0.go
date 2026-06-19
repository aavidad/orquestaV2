package orquestaappplanner

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func BuildGoAPIWebMicrotaskPlanV0(request AppPlanRequestV0) (AppMicrotaskPlanV0, error) {
	request = normalizeAppPlanRequestV0(request)
	if err := validateAppPlanRequestV0(request); err != nil {
		return AppMicrotaskPlanV0{}, err
	}
	plan := AppMicrotaskPlanV0{
		SchemaVersion: AppMicrotaskPlanSchemaVersionV0,
		RunRef:        request.RunRef,
		AppRef:        request.AppRef,
		Units:         appPlanUnitsV0(request),
		EvidenceRefs:  []string{"evidence-ref-app-plan-v0"},
	}
	if err := validateAppMicrotaskPlanV0(plan); err != nil {
		return AppMicrotaskPlanV0{}, err
	}
	return plan, nil
}

func appPlanUnitsV0(request AppPlanRequestV0) []AppWorkUnitV0 {
	if request.Scale == AppPlanScaleLargeV0 {
		return appLargePlanUnitsV0(request)
	}
	return []AppWorkUnitV0{
		appUnitBootstrapV0(request),
		appUnitDomainV0(request),
		appUnitWebV0(request),
		appUnitAPIV0(request),
		appUnitDocsV0(request),
		appUnitReviewV0(request),
	}
}

func appUnitBootstrapV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "bootstrap", "preparacion", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Preparar base Go minima",
		"Crear modulo Go autonomo, contexto local y contrato hexagonal inicial.",
		[]string{"go.mod", "AGENTS.md", "README.md", "docs/contratos.md", "docs/tareas.md", "docs/pruebas.md", "docs/decisiones.md"},
		[]string{
			"go.mod creado con modulo canonico.",
			"AGENTS.md y docs/contratos.md declaran arquitectura hexagonal estricta como condicion de aceptacion.",
			"docs/contratos.md separa domain, application, ports, adapters y bootstrap.",
			"README inicial presente.",
		},
		nil,
	)
}

func appUnitDomainV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "agenda-core", "implementacion", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Crear nucleo hexagonal de aplicacion",
		"Implementar dominio, casos de uso y puertos en modulos internos pequenos.",
		[]string{"internal/domain", "internal/application", "internal/ports"},
		[]string{
			"Dominio probado sin imports de adapters, HTTP, DB, filesystem, runtime ni UI.",
			"Casos de uso dependen de puertos/interfaces, no de repositorios concretos.",
			"Puertos de entrada/salida definidos con DTOs de aplicacion.",
		},
		[]string{deliveryRefV0(request, "bootstrap")},
	)
}

func appUnitWebV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "web", "implementacion", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Crear web de agenda",
		"Implementar superficie web estatica simple.",
		[]string{"web"},
		[]string{"Web estatica presente.", "Textos listos para i18n si crece."},
		[]string{deliveryRefV0(request, "bootstrap")},
	)
}

func appUnitAPIV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "api", "implementacion", orquestacoreworkflow.OrchestrationCapacityHighV0,
		"Crear API REST hexagonal",
		"Implementar handlers finos conectados a casos de uso y bootstrap de composicion.",
		[]string{"internal/adapters/http", "internal/app/bootstrap", "cmd/server"},
		[]string{
			"API REST compilable.",
			"Entrypoint bajo cmd/server y composicion bajo internal/app/bootstrap.",
			"Handlers finos: no construyen repositorios, autenticacion, fixtures ni reglas de negocio.",
			"Imports de modulo, sin imports relativos ../.",
			"`go test ./...` declarado en ACK.",
		},
		[]string{deliveryRefV0(request, "agenda-core"), deliveryRefV0(request, "web")},
	)
}

func appUnitDocsV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "docs", "documentacion", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Documentar uso de agenda",
		"Actualizar README con ejecucion, API y alcance.",
		[]string{"README.md"},
		[]string{"README profesional actualizado.", "Incluye comandos de prueba."},
		[]string{deliveryRefV0(request, "api")},
	)
}

func appUnitReviewV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "review", "revision", orquestacoreworkflow.OrchestrationCapacityHighV0,
		"Revisar mini app de agenda",
		"Registrar revision final compacta de arquitectura, pruebas y riesgos.",
		[]string{"docs/revision.md"},
		[]string{
			"Revision final escrita.",
			"La revision no acepta la app si dominio/application importan adapters, HTTP, DB, filesystem, runtime o UI.",
			"La revision no acepta handlers con composicion de repositorios, autenticacion, fixtures o reglas de negocio.",
			"`go test ./...` declarado en ACK.",
		},
		[]string{deliveryRefV0(request, "docs")},
	)
}

func appUnitV0(
	request AppPlanRequestV0,
	key string,
	role string,
	capacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0,
	title string,
	summary string,
	writeSet []string,
	criteria []string,
	deps []string,
) AppWorkUnitV0 {
	return AppWorkUnitV0{
		TaskRef:             taskRefV0(request, key),
		ClaimRef:            claimRefV0(request, key),
		AgentRequestID:      agentRefV0(request, key),
		DeliveryRef:         deliveryRefV0(request, key),
		PhaseID:             appPhaseForRoleV0(role),
		WorkProfileKind:     appWorkProfileKindForRoleV0(role),
		Role:                role,
		Capacity:            capacity,
		Title:               title,
		Summary:             summary,
		WriteSet:            compactAppPlannerStringsV0(writeSet),
		AcceptanceCriteria:  compactAppPlannerStringsV0(criteria),
		RequiredTests:       []string{"go test ./..."},
		DependsOnDeliveries: compactAppPlannerStringsV0(deps),
		EvidenceRefs:        []string{"evidence-ref-" + request.AppRef + "-" + key},
	}
}

func appWorkProfileKindForRoleV0(role string) orquestacoreworkflow.WorkProfileKindV0 {
	switch role {
	case "arquitectura", "preparacion":
		return orquestacoreworkflow.WorkProfileCodeStudyV0
	case "documentacion":
		return orquestacoreworkflow.WorkProfileDocumentationV0
	case "revision":
		return orquestacoreworkflow.WorkProfileReviewV0
	default:
		return orquestacoreworkflow.WorkProfileImplementationV0
	}
}

func appPhaseForRoleV0(role string) orquestacoreworkflow.OrchestrationPhaseIDV0 {
	switch role {
	case "documentacion":
		return orquestacoreworkflow.OrchestrationPhaseDocumentacionV0
	case "integracion", "entorno":
		return orquestacoreworkflow.OrchestrationPhaseIntegracionV0
	case "revision":
		return orquestacoreworkflow.OrchestrationPhaseRevisionV0
	default:
		return orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	}
}
