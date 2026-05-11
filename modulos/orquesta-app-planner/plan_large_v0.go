package orquestaappplanner

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func appLargePlanUnitsV0(request AppPlanRequestV0) []AppWorkUnitV0 {
	return []AppWorkUnitV0{
		appUnitBootstrapV0(request),
		appLargeArchitectureUnitV0(request),
		appLargeDomainUnitV0(request),
		appLargePersistenceUnitV0(request),
		appLargeAPIUnitV0(request),
		appLargeWebUnitV0(request),
		appLargeI18NUnitV0(request),
		appLargeDeployUnitV0(request),
		appLargeIntegrationUnitV0(request),
		appLargeDocsUnitV0(request),
		appLargeReviewUnitV0(request),
	}
}

func appLargeArchitectureUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "architecture", "arquitectura", orquestacoreworkflow.OrchestrationCapacityXHighV0,
		"Disenar arquitectura de app grande",
		"Documentar fronteras hexagonales, modulos, puertos y riesgos antes de programar.",
		[]string{"docs/arquitectura.md", "docs/contratos.md", "docs/decisiones.md"},
		[]string{"Fronteras y conectores definidos.", "No hay proveedor DB/runtime hardcodeado."},
		[]string{deliveryRefV0(request, "bootstrap")},
	)
}

func appLargeDomainUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "domain", "implementacion", orquestacoreworkflow.OrchestrationCapacityHighV0,
		"Crear dominio hexagonal",
		"Implementar entidades, servicios puros y puertos sin adaptadores concretos.",
		[]string{"internal/domain", "internal/ports"},
		[]string{"Dominio probado.", "Puertos definidos sin DB concreta."},
		[]string{deliveryRefV0(request, "architecture")},
	)
}

func appLargePersistenceUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "persistence-port", "implementacion", orquestacoreworkflow.OrchestrationCapacityHighV0,
		"Preparar contrato de persistencia",
		"Crear puerto de persistencia y adaptador fake de test sin elegir proveedor.",
		[]string{"internal/persistence", "internal/testadapters"},
		[]string{"Persistencia expresada por interfaz.", "No aparece sqlite/postgres/mysql hardcodeado."},
		[]string{deliveryRefV0(request, "architecture")},
	)
}

func appLargeAPIUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "api", "implementacion", orquestacoreworkflow.OrchestrationCapacityHighV0,
		"Crear API REST modular",
		"Implementar handlers pequenos conectados a puertos de aplicacion y entrypoint Go idiomatico.",
		[]string{"internal/api", "cmd/server"},
		[]string{"API compilable.", "Entrypoint bajo cmd/server.", "Imports de modulo, sin imports relativos ../.", "Handlers pequenos y testeados."},
		[]string{deliveryRefV0(request, "domain"), deliveryRefV0(request, "persistence-port")},
	)
}

func appLargeWebUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "web", "implementacion", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Crear web modular",
		"Implementar shell web y cliente API sin acoplarse al servidor interno.",
		[]string{"web"},
		[]string{"Web navegable.", "Textos pasan por i18n local."},
		[]string{deliveryRefV0(request, "architecture")},
	)
}

func appLargeI18NUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "i18n", "implementacion", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Preparar i18n",
		"Crear catalogos de textos y contrato de uso para web/API.",
		[]string{"i18n", "docs/i18n.md"},
		[]string{"Locales documentados.", "No hay textos principales sin clave i18n."},
		[]string{deliveryRefV0(request, "architecture")},
	)
}

func appLargeDeployUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "deploy", "entorno", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Preparar entorno de ejecucion",
		"Crear contrato de despliegue y scripts seguros sin imponer proveedor.",
		[]string{"deploy", "docs/deploy.md"},
		[]string{"Entorno documentado.", "Deploy queda como conector configurable."},
		[]string{deliveryRefV0(request, "architecture")},
	)
}

func appLargeIntegrationUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "integration", "integracion", orquestacoreworkflow.OrchestrationCapacityHighV0,
		"Integrar modulos de app",
		"Conectar dominio, puertos, API, web e i18n con tests de flujo.",
		[]string{"internal/app", "internal/integration"},
		[]string{"Flujo principal probado.", "`go test ./...` declarado en ACK."},
		[]string{
			deliveryRefV0(request, "domain"),
			deliveryRefV0(request, "persistence-port"),
			deliveryRefV0(request, "api"),
			deliveryRefV0(request, "web"),
			deliveryRefV0(request, "i18n"),
		},
	)
}

func appLargeDocsUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "docs", "documentacion", orquestacoreworkflow.OrchestrationCapacityMediumV0,
		"Documentar app grande",
		"Completar manual de desarrollo, operacion, pruebas y decisiones.",
		[]string{"README.md", "docs/operacion.md", "docs/pruebas.md"},
		[]string{"Documentacion profesional.", "Incluye gates y comandos de validacion."},
		[]string{deliveryRefV0(request, "integration"), deliveryRefV0(request, "deploy")},
	)
}

func appLargeReviewUnitV0(request AppPlanRequestV0) AppWorkUnitV0 {
	return appUnitV0(request, "review", "revision", orquestacoreworkflow.OrchestrationCapacityXHighV0,
		"Revisar app grande",
		"Ejecutar revision final de arquitectura, seguridad, tests y tamano.",
		[]string{"docs/revision.md"},
		[]string{"Revision final aceptada.", "Riesgos y rework documentados."},
		[]string{deliveryRefV0(request, "docs")},
	)
}
