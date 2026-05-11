package orquestafactory

type RequestPolicyV0 struct {
	RequestKind              string   `json:"request_kind"`
	ExecutionMode            string   `json:"execution_mode"`
	MinimumDeliverables      []string `json:"minimum_deliverables"`
	AllowsReducedScope       bool     `json:"allows_reduced_scope"`
	ProductiveClosureAllowed bool     `json:"productive_closure_allowed"`
	OmissionReportRequired   bool     `json:"omission_report_required"`
}

func ResolveRequestPolicyV0(kind string, mode string) RequestPolicyV0 {
	mode = NormalizeExecutionModeV0(mode)
	return RequestPolicyV0{
		RequestKind:              NormalizeRequestKindV0(kind),
		ExecutionMode:            mode,
		MinimumDeliverables:      MinimumDeliverablesForRequestKindV0(kind),
		AllowsReducedScope:       mode == ExecutionModeDebugV0,
		ProductiveClosureAllowed: mode != ExecutionModeDebugV0,
		OmissionReportRequired:   mode == ExecutionModeDebugV0,
	}
}

func MinimumDeliverablesForRequestKindV0(kind string) []string {
	switch NormalizeRequestKindV0(kind) {
	case RequestKindDocumentarAppV0:
		return requestPolicyCopyV0(
			"manual_usuario",
			"manual_desarrollador",
			"manual_sistemas_deploy",
			"decisiones",
			"pruebas_documentales",
			"pendientes",
		)
	case RequestKindAnalizarAppV0:
		return requestPolicyCopyV0(
			"inventario",
			"hallazgos",
			"riesgos",
			"seguridad",
			"deuda",
			"recomendaciones",
			"preguntas_bloqueantes",
		)
	case RequestKindBrainstormingArquitecturaV0:
		return requestPolicyCopyV0(
			"opciones_comparadas",
			"votacion",
			"decision_aceptada",
			"tradeoffs",
			"contratos_afectados",
		)
	case RequestKindPlanificarAppV0:
		return requestPolicyCopyV0(
			"alcance",
			"arquitectura",
			"contratos",
			"microtareas",
			"bloqueos",
			"criterios_de_cierre",
		)
	case RequestKindCrearAppCompletaV0:
		return requestPolicyCopyV0(
			"arquitectura",
			"contratos",
			"programacion",
			"pruebas",
			"seguridad",
			"documentacion",
			"deploy_si_aplica",
			"revision_final",
		)
	case RequestKindProgramarModuloV0:
		return requestPolicyCopyV0(
			"contrato_modulo",
			"write_set_pequeno",
			"implementacion",
			"pruebas_focales",
			"documentacion_local",
		)
	case RequestKindModificarAppExistenteV0, RequestKindMigracionRefactorV0:
		return requestPolicyCopyV0(
			"contexto_actual",
			"plan_cambio",
			"microtareas",
			"implementacion",
			"pruebas_regresion",
			"riesgos",
		)
	case RequestKindRevisarCodigoV0:
		return requestPolicyCopyV0("hallazgos", "riesgos", "evidencias", "tests_recomendados", "decision")
	case RequestKindPruebasYValidacionV0:
		return requestPolicyCopyV0("plan_pruebas", "ejecucion", "evidencias", "fallos", "aceptacion_o_rework")
	case RequestKindSeguridadV0:
		return requestPolicyCopyV0("amenazas", "datos_sensibles", "permisos", "dependencias", "evidencias")
	case RequestKindDeployV0, RequestKindOperacionSoporteV0:
		return requestPolicyCopyV0("entorno", "configuracion", "healthcheck", "rollback", "manual_operacion")
	case RequestKindIntegracionExternaV0:
		return requestPolicyCopyV0("contrato_conector", "fixtures", "errores", "pruebas_contrato", "documentacion")
	case RequestKindI18NL10NV0:
		return requestPolicyCopyV0("catalogos", "locales", "fallbacks", "pruebas_i18n", "documentacion")
	case RequestKindInvestigacionTecnicaV0:
		return requestPolicyCopyV0("opciones", "evidencias", "comparativa", "riesgos", "recomendacion")
	default:
		return requestPolicyCopyV0("objetivo", "alcance", "entregables", "evidencias", "riesgos")
	}
}

func requestPolicyCopyV0(values ...string) []string {
	return append([]string(nil), values...)
}
