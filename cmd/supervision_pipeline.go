package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/db"
)

func buildSupervisorPipelineSnapshot(supervisor, proyectoSlug string, limit int) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	items, err := db.ListarSupervisorPipelineStates(db.FiltroSupervisorPipelineStates{
		Supervisor:   strings.TrimSpace(supervisor),
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"supervisor":   supervisor,
		"proyecto":     strings.TrimSpace(proyectoSlug),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"pipelines":    items,
	}, nil
}

func buildSupervisorPipelineOverview(supervisor string) (string, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorPipelineSnapshot(supervisor, "", 20)
	if err != nil {
		return "", err
	}
	items, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState)
	var b strings.Builder
	fmt.Fprintf(&b, "# Pipeline del supervisor: %s\n\n", supervisor)
	if len(items) == 0 {
		b.WriteString("- Sin pipeline explícita registrada.\n")
		return b.String(), nil
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		fmt.Fprintf(&b, "- %s", strings.TrimSpace(item.PipelineName))
		if strings.TrimSpace(item.ProyectoSlug) != "" {
			fmt.Fprintf(&b, " proyecto=%s", strings.TrimSpace(item.ProyectoSlug))
		}
		if strings.TrimSpace(item.CurrentPhase) != "" {
			fmt.Fprintf(&b, " fase=%s", strings.TrimSpace(item.CurrentPhase))
		}
		fmt.Fprintf(&b, " estado=%s\n", strings.TrimSpace(item.Status))
	}
	return strings.TrimSpace(b.String()) + "\n", nil
}
