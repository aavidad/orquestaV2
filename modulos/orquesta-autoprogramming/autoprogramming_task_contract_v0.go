package orquestaautoprogramming

import (
	"fmt"
	"strings"
)

func autoprogrammingTitleForGroupV0(
	group AutoprogrammingTaskGroupV0,
	groupIndex int,
) string {
	for _, task := range group.Tasks {
		if task.Title != "" {
			return task.Title
		}
	}
	return fmt.Sprintf("Cambio acotado %02d", groupIndex+1)
}

func autoprogrammingObjectiveForGroupV0(
	group AutoprogrammingTaskGroupV0,
) string {
	lines := []string{"Resolver refs de tarea asignadas dentro del write-set validado."}
	for _, task := range group.Tasks {
		if task.Objective != "" {
			lines = append(lines, "Objetivo "+task.TaskRef+": "+task.Objective)
		}
		for _, context := range task.Context {
			lines = append(lines, "Contexto "+task.TaskRef+": "+context)
		}
		for _, rule := range task.CompactRules {
			lines = append(lines, "Regla compacta "+task.TaskRef+": "+rule)
		}
	}
	return strings.Join(compactStringsV0(lines), "\n")
}

func autoprogrammingSummaryForGroupV0(
	group AutoprogrammingTaskGroupV0,
) string {
	details := make([]string, 0, len(group.Tasks)*3)
	for _, task := range group.Tasks {
		if task.Objective != "" {
			details = append(details, task.Objective)
		}
		details = append(details, task.Context...)
		if len(task.CompactRules) > 0 {
			details = append(details, "reglas: "+strings.Join(task.CompactRules, "; "))
		}
	}
	if len(details) == 0 {
		return "Autoprogramacion acotada con worktree aislada y rama opaca preservada."
	}
	return "Autoprogramacion acotada: " + strings.Join(compactStringsV0(details), " | ")
}

func autoprogrammingRequiredTestsForGroupV0(
	requiredTests []string,
	group AutoprogrammingTaskGroupV0,
) []string {
	out := append([]string(nil), requiredTests...)
	for _, task := range group.Tasks {
		out = append(out, task.RequiredTests...)
	}
	return compactStringsV0(out)
}

func autoprogrammingContextRefsForGroupV0(
	request AutoprogrammingRequestV0,
	group AutoprogrammingTaskGroupV0,
) []string {
	refs := autoprogrammingProgrammableContextRefsV0(request)
	for _, task := range group.Tasks {
		refs = append(refs, "source_task_ref:"+task.TaskRef)
		refs = append(refs, task.ContextRefs...)
	}
	return compactStringsV0(refs)
}

func autoprogrammingAcceptanceCriteriaForGroupV0(
	group AutoprogrammingTaskGroupV0,
) []string {
	out := []string{
		"Cambios limitados al write-set asignado al grupo.",
		"ACK declara pruebas requeridas ejecutadas y resultado.",
		"Refs de tarea origen preservadas: " + strings.Join(group.TaskRefs, ","),
	}
	for _, task := range group.Tasks {
		for _, criterion := range task.AcceptanceCriteria {
			out = append(out, "Criterio "+task.TaskRef+": "+criterion)
		}
		for _, rule := range task.CompactRules {
			out = append(out, "Regla compacta "+task.TaskRef+": "+rule)
		}
	}
	return compactStringsV0(out)
}
