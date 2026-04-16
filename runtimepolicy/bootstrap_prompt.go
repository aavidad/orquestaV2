package runtimepolicy

import (
	"path/filepath"
	"strings"

	"orquesta/runtimeagente"
)

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

func RuntimeLaunchBootstrapPromptHasActiveTask(taskStates []string) bool {
	for _, state := range taskStates {
		switch strings.ToLower(strings.TrimSpace(state)) {
		case "asignada", "en_progreso", "bloqueada":
			return true
		}
	}
	return false
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
