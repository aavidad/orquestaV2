package orquestaautoprogramming

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	autoprogrammingWorkflowTitleLimitV0     = 80
	autoprogrammingWorkflowSummaryLimitV0   = 160
	autoprogrammingWorkflowObjectiveLimitV0 = 600
	autoprogrammingWorkflowCriteriaLimitV0  = 100
	autoprogrammingWorkflowCriteriaMaxV0    = 14
)

func autoprogrammingTitleForGroupV0(
	group AutoprogrammingTaskGroupV0,
	groupIndex int,
) string {
	for _, task := range group.Tasks {
		if task.Title != "" {
			return compactAutoprogrammingWorkflowTextV0(task.Title, autoprogrammingWorkflowTitleLimitV0)
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
	return compactAutoprogrammingWorkflowTextV0(
		strings.Join(compactStringsV0(lines), "\n"),
		autoprogrammingWorkflowObjectiveLimitV0,
	)
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
	return compactAutoprogrammingWorkflowTextV0(
		"Autoprogramacion acotada: "+strings.Join(compactStringsV0(details), " | "),
		autoprogrammingWorkflowSummaryLimitV0,
	)
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

func compactAutoprogrammingWorkflowTextsV0(values []string, limit int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if compact := compactAutoprogrammingWorkflowTextV0(value, limit); compact != "" {
			out = append(out, compact)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func compactAutoprogrammingWorkflowTextV0(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return strings.TrimSpace(string(runes[:limit-3])) + "..."
}

func autoprogrammingContextRefsForGroupV0(
	request AutoprogrammingRequestV0,
	group AutoprogrammingTaskGroupV0,
) []string {
	refs := autoprogrammingSafeWorkflowContextRefsV0(
		autoprogrammingProgrammableContextRefsV0(request),
	)
	for _, task := range group.Tasks {
		refs = append(refs, "source_task_ref:"+task.TaskRef)
		refs = append(refs, autoprogrammingSafeWorkflowContextRefsV0(task.ContextRefs)...)
	}
	return compactStringsV0(refs)
}

func autoprogrammingSafeWorkflowContextRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if ref := autoprogrammingSafeWorkflowContextRefV0(value); ref != "" {
			out = append(out, ref)
		}
	}
	return compactStringsV0(out)
}

func autoprogrammingSafeWorkflowContextRefV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if autoprogrammingWorkflowContextRefIsCompactV0(value) {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	prefix := autoprogrammingWorkflowContextRefPrefixV0(value)
	return "context_ref:" + prefix + "-" + hex.EncodeToString(sum[:])[:12]
}

func autoprogrammingWorkflowContextRefIsCompactV0(value string) bool {
	return len(value) <= 600 && !strings.ContainsAny(value, " /\\\t\r\n")
}

func autoprogrammingWorkflowContextRefPrefixV0(value string) string {
	prefix := value
	if before, _, ok := strings.Cut(value, ":"); ok {
		prefix = before
	}
	prefix = normalizeAutoprogrammingTaskAreaV0(prefix)
	if prefix == "" {
		return "context"
	}
	return prefix
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
	return compactAutoprogrammingWorkflowTextsLimitedV0(
		compactStringsV0(out),
		autoprogrammingWorkflowCriteriaLimitV0,
		autoprogrammingWorkflowCriteriaMaxV0,
	)
}

func compactAutoprogrammingWorkflowTextsLimitedV0(values []string, limit int, maxItems int) []string {
	out := compactAutoprogrammingWorkflowTextsV0(values, limit)
	if maxItems <= 0 || len(out) <= maxItems {
		return out
	}
	hash := autoprogrammingWorkflowListHashV0(out[maxItems:])
	out = append([]string(nil), out[:maxItems]...)
	out = append(out, "criterios_extra_ref:"+hash)
	return out
}
