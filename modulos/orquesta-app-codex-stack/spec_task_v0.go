package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func (resolver CodexLaunchSpecResolverV0) agentTaskV0(
	ctx context.Context,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
	area string,
) (orquestaruntime.AgentStartTaskV0, error) {
	if isProgrammingPhaseV0(payload.PhaseID) {
		return resolver.programmingTaskV0(ctx, payload)
	}
	return directorTaskV0(payload, area), nil
}

func (resolver CodexLaunchSpecResolverV0) programmingTaskV0(
	ctx context.Context,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) (orquestaruntime.AgentStartTaskV0, error) {
	if resolver.TaskStore == nil {
		return orquestaruntime.AgentStartTaskV0{}, fmt.Errorf("task_store requerido")
	}
	tasks, err := resolver.TaskStore.LoadWorkflowTasksV0(
		ctx,
		strings.TrimSpace(payload.RunID),
		[]string{strings.TrimSpace(payload.TaskRef)},
	)
	if err != nil {
		return orquestaruntime.AgentStartTaskV0{}, err
	}
	if len(tasks) != 1 {
		return orquestaruntime.AgentStartTaskV0{}, fmt.Errorf("workflow_task no encontrada")
	}
	task := tasks[0]
	return orquestaruntime.AgentStartTaskV0{
		TaskRef:   task.TaskID,
		Priority:  "alta",
		Title:     task.Title,
		Objective: programmingObjectiveV0(task),
		WriteSet:  append([]string(nil), task.WriteSet...),
		RequiredTests: append([]string(nil),
			task.RequiredTests...,
		),
		DoneCriteria: append([]string{"agent_ack.json escrito con status completed."}, task.AcceptanceCriteria...),
	}, nil
}

func programmingObjectiveV0(task orquestacoreworkflow.WorkflowTaskV0) string {
	unit := "contrato de esta microtarea"
	if workflowTaskHasDomainWorkContractV0(task) {
		unit = "contrato de esta unidad de trabajo externa"
	}
	return strings.Join([]string{
		strings.TrimSpace(task.Summary),
		"Implementa solo el " + unit + ".",
		"No cambies ficheros fuera del write-set.",
		"Si la app es Go completa, debe quedar como modulo autonomo con go.mod, entrypoint bajo cmd/server o equivalente documentado, imports de modulo y sin imports relativos ../.",
		"Ejecuta pruebas focales razonables y registra el resultado en el ACK.",
	}, "\n")
}

func directorTaskV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
	area string,
) orquestaruntime.AgentStartTaskV0 {
	writeSet := directorWriteSetV0(area)
	return orquestaruntime.AgentStartTaskV0{
		TaskRef:      strings.TrimSpace(payload.TaskRef),
		Priority:     "alta",
		Title:        directorTitleV0(area),
		Objective:    directorObjectiveV0(payload, area),
		WriteSet:     writeSet,
		DoneCriteria: directorDoneCriteriaV0(payload, area),
	}
}

func directorTitleV0(area string) string {
	if area == "director" {
		return "Dirigir arquitectura inicial de la app"
	}
	return "Definir " + area + " de la app"
}

func directorObjectiveV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
	area string,
) string {
	kind := directorRequestKindV0(payload)
	mode := directorExecutionModeV0(payload)
	lines := []string{
		"Actua como agente director de Orquesta para una app solicitada por web/API/MCP.",
		"Area: " + area + ".",
		"Tipo de peticion: " + kind + ".",
		"Modo de ejecucion: " + mode + ".",
		"Resumen de Orquesta: " + strings.TrimSpace(payload.Summary),
		"Reglas: hexagonal, i18n si aplica, persistencia solo por puerto/conector, funciones pequenas, sin archivos gigantes.",
		"Para una app Go completa exige modulo autonomo: go.mod, entrypoint cmd/server o equivalente documentado, imports de modulo y `go test ./...` como prueba de cierre.",
		"Separa brainstorming, documentacion, programacion, pruebas, seguridad y revision final.",
		"Cumple los minimos del tipo de peticion; solo puedes recortar alcance si execution_mode=debug y debes listar lo omitido.",
		"Si falta informacion no inferible, deja CONSULTA AL DIRECTOR en el documento.",
	}
	lines = append(lines, directorRequestKindInstructionsV0(kind, mode)...)
	if area == "director" {
		lines = append(lines, directorDecisionInstructionsV0(payload)...)
	}
	return strings.Join(compactStringsV0(lines), "\n")
}

func directorDoneCriteriaV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
	area string,
) []string {
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

func directorWriteSetV0(area string) []string {
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
		return []string{"docs/arquitectura.md", "docs/plan_microtareas.md"}
	}
}
