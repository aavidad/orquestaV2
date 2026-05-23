package orquestadirectoroperativo

func operationalDirectorWorkProfileKindV0(
	plan OperationalDirectorPlanV0,
	kind OperationalDirectorStepKindV0,
) string {
	switch kind {
	case OperationalDirectorStepGatherContextV0,
		OperationalDirectorStepSplitWorkV0,
		OperationalDirectorStepRequestDomainContextV0:
		return "code_study"
	case OperationalDirectorStepLaunchSubagentsV0:
		if plan.Mode == OperationalDirectorModeDomainWorkV0 {
			return "domain_work"
		}
		return "implementation"
	case OperationalDirectorStepRunRequiredTestsV0:
		return "required_tests"
	case OperationalDirectorStepReviewDeliveriesV0,
		OperationalDirectorStepReplanOrCloseV0:
		return "review"
	default:
		return ""
	}
}

func operationalDirectorAcceptanceCriteriaV0(
	plan OperationalDirectorPlanV0,
) []string {
	criteria := []string{
		"Mantener el objetivo actual: " + plan.Objective,
		"Reparar antes que rechazar si es seguro: normalizar, pedir correccion dirigida, delegar revision o secuenciar otro paso.",
		"Cortar fuerte solo por seguridad, causalidad rota, refs imposibles o efecto externo no autorizado.",
	}
	if plan.Mode == OperationalDirectorModeProgrammingV0 {
		return append(criteria,
			"Usar el write-set como alcance declarado y justificar cualquier ampliacion necesaria para cumplir el objetivo o arreglar pruebas.",
			"Ejecutar y registrar tests obligatorios antes de cerrar.",
		)
	}
	if plan.Status == OperationalDirectorPlanNeedsContextV0 {
		return append(criteria, "No lanzar agentes hasta recibir el contexto faltante.")
	}
	return append(criteria,
		"Usar solo refs de dominio y campos entregados por la app externa.",
		"No inventar contexto de dominio ausente.",
	)
}
