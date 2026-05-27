package orquestaappcodexstack

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func directorTitleV0(area string) string {
	if area == "director" {
		return "Dirigir arquitectura inicial de la app"
	}
	return "Definir " + area + " de la app"
}

func directorObjectiveV0(payload orquestaruntime.LaunchRuntimeAgentRequestV0, area string) string {
	kind := directorRequestKindV0(payload)
	mode := directorExecutionModeV0(payload)
	lines := []string{
		"Actua como agente director de Orquesta para una app solicitada por web/API/MCP.",
		"Area: " + area + ".",
		"Tipo de peticion: " + kind + ".",
		"Modo de ejecucion: " + mode + ".",
		"Resumen de Orquesta: " + strings.TrimSpace(payload.Summary),
		"Reglas: hexagonal, i18n si aplica, persistencia solo por puerto/conector, funciones pequenas, sin archivos gigantes.",
		"No asumas app nueva ni Go por defecto: request_kind decide si crear, modificar, refactorizar, reimplementar, documentar, analizar, probar, asegurar o preparar deploy.",
		"Separa brainstorming, documentacion, programacion, pruebas, seguridad y revision final.",
		"Cumple los minimos del tipo de peticion; solo puedes recortar alcance si execution_mode=debug y debes listar lo omitido.",
		"Si falta informacion no inferible, deja CONSULTA AL DIRECTOR en el documento.",
	}
	if area == "director" {
		lines = append(lines,
			"No eres un worker de area: puedes producir entregables globales dentro de tu write-set.",
			"No te limites a planificar si se piden artefactos reales; crea documentos minimos y evidencias.",
		)
	}
	lines = append(lines, directorRequestKindInstructionsV0(kind, mode)...)
	if area == "director" {
		lines = append(lines, directorDecisionInstructionsV0(payload)...)
	}
	return strings.Join(compactStringsV0(lines), "\n")
}

func directorDoneCriteriaV0(payload orquestaruntime.LaunchRuntimeAgentRequestV0, area string) []string {
	policy := orquestafactory.ResolveRequestPolicyV0(directorRequestKindV0(payload), directorExecutionModeV0(payload))
	criteria := []string{
		"Documento del area creado dentro del write-set.",
		"Decisiones, riesgos y consultas al director quedan documentadas.",
		"El alcance documentado respeta request_kind y execution_mode.",
		"El plan cubre minimos: " + strings.Join(policy.MinimumDeliverables, ", ") + ".",
		"agent_ack.json escrito con status completed.",
	}
	if area == "director" {
		criteria = append(criteria, "director_decisions.json escrito en decision_path con decisiones ejecutables.")
	}
	if policy.OmissionReportRequired {
		return append(criteria, "Si se recorta el alcance, el ACK enumera entregables omitidos por debug.")
	}
	return criteria
}

func directorRequestKindInstructionsV0(kind string, mode string) []string {
	policy := orquestafactory.ResolveRequestPolicyV0(kind, mode)
	lines := []string{
		"Minimos " + policy.RequestKind + ": " + strings.Join(policy.MinimumDeliverables, ", ") + ".",
	}
	if policy.OmissionReportRequired {
		lines = append(lines, "DEBUG: puedes reducir entregables, pero no marques cierre productivo y lista exactamente que falta.")
	}
	return lines
}

func directorRequestKindV0(payload orquestaruntime.LaunchRuntimeAgentRequestV0) string {
	return directorSummaryTokenV0(payload.Summary, "request_kind", orquestafactory.DefaultRequestKindV0)
}

func directorExecutionModeV0(payload orquestaruntime.LaunchRuntimeAgentRequestV0) string {
	return directorSummaryTokenV0(payload.Summary, "execution_mode", orquestafactory.DefaultExecutionModeV0)
}

func directorSummaryTokenV0(summary string, key string, fallback string) string {
	prefix := key + "="
	for _, part := range strings.Fields(summary) {
		value := strings.Trim(strings.TrimSpace(part), ".,;")
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix)
		}
	}
	return fallback
}

func directorWriteSetV0(payload orquestaruntime.LaunchRuntimeAgentRequestV0, area string) []string {
	switch area {
	case "web":
		return []string{"docs/web.md"}
	case "api":
		return []string{"docs/api.md"}
	case "persistencia":
		return []string{"docs/persistencia.md"}
	case "i18n":
		return []string{"docs/i18n.md"}
	case "calidad":
		return []string{"docs/calidad.md"}
	default:
		return directorGlobalWriteSetV0(directorRequestKindV0(payload))
	}
}

func directorGlobalWriteSetV0(kind string) []string {
	paths := []string{
		"docs/arquitectura.md",
		"docs/plan_tareas.md",
		"docs/decisiones.md",
		"docs/pruebas.md",
		"docs/pendientes.md",
	}
	switch orquestafactory.NormalizeRequestKindV0(kind) {
	case orquestafactory.RequestKindDocumentarAppV0,
		orquestafactory.RequestKindCrearAppCompletaV0,
		orquestafactory.RequestKindPlanificarAppV0:
		paths = append(paths,
			"docs/manual_usuario.md",
			"docs/manual_desarrollador.md",
			"docs/manual_sistemas_deploy.md",
		)
	}
	return paths
}
