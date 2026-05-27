package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
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
