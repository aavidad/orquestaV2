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
		TaskRef:         task.TaskID,
		Priority:        "alta",
		Title:           task.Title,
		Objective:       programmingObjectiveV0(task, payload),
		WriteSet:        append([]string(nil), task.WriteSet...),
		ParentTaskRef:   task.ParentTaskRef,
		CohortRef:       task.CohortRef,
		WaveRef:         task.WaveRef,
		DelegationDepth: task.DelegationDepth,
		MaxChildAgents:  task.MaxChildAgents,
		ChildTaskRefs:   append([]string(nil), task.ChildTaskRefs...),
		RequiredTests: append([]string(nil),
			task.RequiredTests...,
		),
		DoneCriteria: append(
			append([]string{"agent_ack.json escrito con status completed."}, task.AcceptanceCriteria...),
			autoprogrammingScopeDoneCriteriaV0(task)...,
		),
	}, nil
}

func programmingObjectiveV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) string {
	unit := "contrato de esta tarea completa"
	if workflowTaskHasDomainWorkContractV0(task) {
		unit = "contrato de esta unidad de trabajo externa"
	}
	lines := []string{
		strings.TrimSpace(task.Summary),
		programmingProfileObjectiveLineV0(task, unit),
		strictWriteSetObjectiveLineV0(),
		agentDelegationObjectiveLineV0(task.MaxChildAgents, task.MaxSubagentsPerAgent),
		"Ejecuta pruebas focales razonables y registra el resultado en el ACK.",
	}
	if programmingTaskRequiresCompleteGoAppV0(task) {
		lines = append(lines, "App Go completa: modulo autonomo con go.mod, entrypoint bajo cmd/server o equivalente documentado, imports de modulo y sin imports relativos ../.")
	} else {
		lines = append(lines, "No conviertas refactors, revisiones, documentacion o cambios de app existente en una app Go nueva salvo que el contrato lo pida claramente.")
	}
	if scopeLine := autoprogrammingScopeObjectiveLineV0(task); scopeLine != "" {
		lines = append(lines, scopeLine)
	}
	if context := programmingReworkContextV0(payload); context != "" {
		lines = append(lines, context)
	}
	return strings.Join(lines, "\n")
}

func agentDelegationObjectiveLineV0(maxChildAgents int, maxSubagentsPerAgent int) string {
	limit := agentDelegationLimitV0(maxChildAgents, maxSubagentsPerAgent)
	return fmt.Sprintf(
		"Delegacion operativa: si necesitas ayuda y el runtime lo permite, activa subagentes para paralelizar analisis, implementacion, pruebas o revision; limite %d subagentes, conservando refs/parentesco, write-set, presupuesto y evidencia en el ACK.",
		limit,
	)
}

func agentDelegationLimitV0(maxChildAgents int, maxSubagentsPerAgent int) int {
	limit := maxChildAgents
	if maxSubagentsPerAgent > 0 && (limit <= 0 || maxSubagentsPerAgent < limit) {
		limit = maxSubagentsPerAgent
	}
	if limit <= 0 || limit > 6 {
		limit = 6
	}
	return limit
}

func programmingTaskRequiresCompleteGoAppV0(task orquestacoreworkflow.WorkflowTaskV0) bool {
	if programmingTaskWriteSetContainsV0(task, "go.mod") {
		return true
	}
	text := strings.ToLower(strings.Join([]string{
		task.Summary,
		task.Title,
		strings.Join(task.AcceptanceCriteria, " "),
	}, " "))
	if strings.Contains(text, "go.mod") ||
		strings.Contains(text, "app go completa") ||
		strings.Contains(text, "modulo go autonomo") {
		return true
	}
	return strings.Contains(text, "request_kind=crear_app_completa") && strings.Contains(text, " go")
}

func programmingTaskWriteSetContainsV0(task orquestacoreworkflow.WorkflowTaskV0, want string) bool {
	want = strings.Trim(strings.TrimSpace(want), "/")
	for _, path := range task.WriteSet {
		if strings.Trim(strings.TrimSpace(path), "/") == want {
			return true
		}
	}
	return false
}

func programmingProfileObjectiveLineV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	unit string,
) string {
	switch task.WorkProfileKind {
	case orquestacoreworkflow.WorkProfileCodeStudyV0:
		return "Estudia el codigo y documenta el mapa de cambio del " + unit + "; no cambies implementacion salvo que sea imprescindible para producir evidencia verificable."
	case orquestacoreworkflow.WorkProfileRefactorV0:
		return "Refactoriza solo el " + unit + " conservando comportamiento publico; si falta alcance, pide decision del director en el ACK."
	case orquestacoreworkflow.WorkProfileRequiredTestsV0:
		return "Completa o ejecuta las pruebas requeridas del " + unit + " y deja evidencia clara del resultado."
	default:
		return "Implementa solo el " + unit + "."
	}
}

func strictWriteSetObjectiveLineV0() string {
	return "Usa el write-set como alcance cerrado. Si falta alcance para cumplir el objetivo o arreglar pruebas, no edites fuera: deja CONSULTA AL DIRECTOR en el ACK y cierra failed salvo decision explicita del director o policy opt-in distinta."
}

func programmingReworkContextV0(payload orquestaruntime.LaunchRuntimeAgentRequestV0) string {
	summary := strings.TrimSpace(payload.Summary)
	if summary == "" {
		return ""
	}
	lower := strings.ToLower(summary + "\n" + strings.Join(payload.EvidenceRefs, "\n"))
	if !strings.Contains(lower, "revision") &&
		!strings.Contains(lower, "rework") &&
		!strings.Contains(lower, "retrabajo") &&
		!strings.Contains(lower, "faltante") &&
		!strings.Contains(lower, "missing") {
		return ""
	}
	return "Contexto de revision/rework: " + summary +
		" Corrige la entrega existente; conserva lo valido y completa lo indicado."
}

func directorTaskV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
	area string,
) orquestaruntime.AgentStartTaskV0 {
	writeSet := directorWriteSetV0(payload, area)
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
