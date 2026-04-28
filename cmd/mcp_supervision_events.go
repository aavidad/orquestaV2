package cmd

import (
	"fmt"
	"strings"
)

const mcpSupervisorEventsDefaultLimit = 20

func buildSupervisorEventsOverview(supervisor string, limit int) (string, error) {
	supervisor = resolveSupervisorName(supervisor)
	items, err := buildOpenClawNormalizedEvents(limit)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("Eventos del supervisor: ")
	b.WriteString(supervisor)
	b.WriteString("\n\n")
	if len(items) == 0 {
		b.WriteString("No hay eventos normalizados recientes.\n")
		return b.String(), nil
	}

	b.WriteString("Señal reciente normalizada para decisión operativa:\n")
	for i, item := range items {
		b.WriteString(fmt.Sprintf("%d. %s", i+1, strings.TrimSpace(item.NormalizedEvent)))
		if item.Project != "" {
			b.WriteString(" · proyecto=" + strings.TrimSpace(item.Project))
		}
		if item.Agent != "" {
			b.WriteString(" · agente=" + strings.TrimSpace(item.Agent))
		}
		if item.Status != "" {
			b.WriteString(" · estado=" + strings.TrimSpace(item.Status))
		}
		if item.SuggestedAction != "" {
			b.WriteString(" · accion=" + strings.TrimSpace(item.SuggestedAction))
		}
		if item.Message != "" {
			b.WriteString(" · ")
			b.WriteString(compactMCPLine(item.Message, 180))
		}
		b.WriteString("\n")
	}

	b.WriteString("\nCriterio de uso:\n")
	b.WriteString("- Prioriza `notification.failed`, `review.blocked`, `supervisor.approval_required` y `merge.failed`.\n")
	b.WriteString("- Usa esta cola como señal operativa rápida; no sustituye la revisión completa de pipeline/review.\n")
	return b.String(), nil
}

func callMCPSupervisionEvents(args map[string]any) (map[string]any, error) {
	supervisor := optionalStringArg(args, "supervisor")
	limit := intArgOrDefault(args, "limit", mcpSupervisorEventsDefaultLimit)
	items, err := buildOpenClawNormalizedEvents(limit)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"supervisor":        resolveSupervisorName(supervisor),
		"limit":             limit,
		"normalized_events": items,
	}
	return toolResult(prettyJSON(payload), payload, false), nil
}
