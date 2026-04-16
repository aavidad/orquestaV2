package runtimepolicy

import (
	"fmt"
	"path/filepath"
	"strings"

	"orquesta/runtimeagente"
)

type RuntimeLaunchBootstrapTask struct {
	ID          int64
	Estado      string
	Titulo      string
	Descripcion string
}

func RuntimeLaunchBootstrapPromptCompact(plan *runtimeagente.LaunchPlan) bool {
	if plan == nil {
		return false
	}
	if plan.CanSendInput != nil && !*plan.CanSendInput &&
		strings.EqualFold(strings.TrimSpace(plan.MailboxDeliveryMode), runtimeagente.MailboxDeliveryBootstrapOnly) {
		return true
	}
	return runtimeLaunchBootstrapPromptLooksLikeOrchestratedAgentCLI(plan)
}

func RuntimeLaunchBootstrapPromptLooksLikeOrchestratedAgentCLI(plan *runtimeagente.LaunchPlan) bool {
	return runtimeLaunchBootstrapPromptLooksLikeOrchestratedAgentCLI(plan)
}

func RuntimeLaunchBootstrapPromptLooksLikeOllamaCLI(plan *runtimeagente.LaunchPlan) bool {
	if plan == nil {
		return false
	}
	parts := make([]string, 0, 1+len(plan.Args))
	if cmd := strings.ToLower(strings.TrimSpace(plan.Comando)); cmd != "" {
		parts = append(parts, cmd)
	}
	for _, arg := range plan.Args {
		arg = strings.ToLower(strings.TrimSpace(arg))
		if arg != "" {
			parts = append(parts, arg)
		}
	}
	for _, part := range parts {
		base := strings.ToLower(strings.TrimSpace(filepath.Base(part)))
		if part == "ollama" || base == "ollama" ||
			strings.Contains(part, "ollama-cli") || strings.Contains(part, "ollama-perfil") ||
			strings.Contains(base, "ollama-cli") || strings.Contains(base, "ollama-perfil") {
			return true
		}
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(plan.RemoteConfigJSON)), "ollama") {
		return true
	}
	return false
}

func RuntimeLaunchBootstrapPromptHasActiveTask(taskStates []string) bool {
	for _, state := range taskStates {
		switch strings.ToLower(strings.TrimSpace(state)) {
		case "asignada", "en_progreso", "bloqueada":
			return true
		}
	}
	return false
}

func RuntimeLaunchBootstrapPromptCompactTaskSummary(tareas []RuntimeLaunchBootstrapTask) string {
	if len(tareas) == 0 {
		return "No hay una tarea activa única; consulta Orquesta antes de desviarte."
	}
	for _, tarea := range tareas {
		switch strings.ToLower(strings.TrimSpace(tarea.Estado)) {
		case "asignada", "en_progreso", "bloqueada":
			partes := []string{fmt.Sprintf("Tarea activa: #%d [%s] %s.", tarea.ID, strings.TrimSpace(tarea.Estado), strings.TrimSpace(tarea.Titulo))}
			if descripcion := runtimeLaunchBootstrapPromptCompactTaskDescription(strings.TrimSpace(tarea.Descripcion)); descripcion != "" {
				partes = append(partes, "Alcance inmediato: "+descripcion+".")
			}
			return strings.Join(partes, " ")
		}
	}
	return "No hay una tarea activa única; consulta Orquesta antes de desviarte."
}

func runtimeLaunchBootstrapPromptLooksLikeOrchestratedAgentCLI(plan *runtimeagente.LaunchPlan) bool {
	if plan == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(plan.Driver), "cli") {
		return false
	}
	parts := make([]string, 0, 1+len(plan.Args))
	if cmd := strings.ToLower(strings.TrimSpace(plan.Comando)); cmd != "" {
		parts = append(parts, cmd)
	}
	for _, arg := range plan.Args {
		arg = strings.ToLower(strings.TrimSpace(arg))
		if arg != "" {
			parts = append(parts, arg)
		}
	}
	for _, part := range parts {
		base := strings.ToLower(strings.TrimSpace(filepath.Base(part)))
		if strings.Contains(part, "codex-perfil") || strings.Contains(part, "claude-perfil") || strings.Contains(part, "gemini-perfil") ||
			strings.Contains(base, "codex-perfil") || strings.Contains(base, "claude-perfil") || strings.Contains(base, "gemini-perfil") {
			return true
		}
		if part == "codex" || part == "claude" || part == "gemini" || part == "ollama" ||
			base == "codex" || base == "claude" || base == "gemini" || base == "ollama" {
			return true
		}
		if strings.Contains(part, "ollama-cli") || strings.Contains(part, "ollama-perfil") ||
			strings.Contains(base, "ollama-cli") || strings.Contains(base, "ollama-perfil") {
			return true
		}
	}
	return false
}

func runtimeLaunchBootstrapPromptCompactTaskDescription(raw string) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if raw == "" {
		return ""
	}
	raw = strings.TrimSpace(strings.TrimRight(raw, ".;"))
	const maxRunes = 360
	prioritized := runtimeLaunchBootstrapPromptCompactTaskDescriptionPrioritaria(raw, maxRunes)
	if prioritized != "" {
		return prioritized
	}
	runes := []rune(raw)
	if len(runes) <= maxRunes {
		return raw
	}
	return string(runes[:maxRunes-1]) + "…"
}

func runtimeLaunchBootstrapPromptCompactTaskDescriptionPrioritaria(raw string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	clauses := runtimeLaunchBootstrapPromptSplitTaskClauses(raw)
	if len(clauses) == 0 {
		return ""
	}
	priorityMatchers := []string{
		"simbolos foco:",
		"write-set",
		"write_set",
		"tests minimos",
		"tests mínimos",
		"regla arquitectonica de este frente:",
		"regla arquitectónica de este frente:",
	}
	seen := map[string]struct{}{}
	selected := make([]string, 0, len(clauses))
	appendIfFits := func(clause string) bool {
		clause = runtimeLaunchBootstrapPromptCompactTaskClause(strings.TrimSpace(strings.TrimRight(clause, ".;")))
		if clause == "" {
			return false
		}
		if _, ok := seen[clause]; ok {
			return true
		}
		candidate := clause
		if len(selected) > 0 {
			candidate = strings.Join(append(append([]string(nil), selected...), clause), ". ")
		}
		if len([]rune(candidate)) > maxRunes {
			return false
		}
		selected = append(selected, clause)
		seen[clause] = struct{}{}
		return true
	}
	for _, matcher := range priorityMatchers {
		for _, clause := range clauses {
			lower := strings.ToLower(strings.TrimSpace(clause))
			if !strings.Contains(lower, matcher) {
				continue
			}
			if !appendIfFits(clause) {
				break
			}
		}
	}
	if len(selected) == 0 {
		return ""
	}
	resumen := strings.Join(selected, ". ")
	resumen = strings.TrimSpace(strings.TrimRight(resumen, ".;"))
	if resumen == "" {
		return ""
	}
	return resumen
}

func runtimeLaunchBootstrapPromptCompactTaskClause(clause string) string {
	clause = strings.TrimSpace(strings.TrimRight(clause, ".;"))
	if clause == "" {
		return ""
	}
	lower := strings.ToLower(clause)
	if strings.HasPrefix(lower, "tests minimos") || strings.HasPrefix(lower, "tests mínimos") {
		clause = strings.Replace(clause, "Tests minimos del slice:", "Tests minimos:", 1)
		clause = strings.Replace(clause, "Tests mínimos del slice:", "Tests mínimos:", 1)
		clause = strings.ReplaceAll(clause, " -count=1", "")
		clause = strings.ReplaceAll(clause, " y go test ", " ; go test ")
		clause = strings.Join(strings.Fields(clause), " ")
	}
	return clause
}

func runtimeLaunchBootstrapPromptSplitTaskClauses(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "\n", ". ")
	partes := strings.Split(raw, ". ")
	out := make([]string, 0, len(partes))
	for _, parte := range partes {
		parte = strings.TrimSpace(strings.TrimRight(parte, ".;"))
		if parte == "" {
			continue
		}
		out = append(out, parte)
	}
	return out
}
