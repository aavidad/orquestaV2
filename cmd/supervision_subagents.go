package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/db"
)

func buildSupervisorSubagentsSnapshot(supervisor, proyectoSlug, sessionID string, limit int) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	if limit <= 0 {
		limit = 100
	}
	items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
		Supervisor:   strings.TrimSpace(supervisor),
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		SessionID:    strings.TrimSpace(sessionID),
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"supervisor":    supervisor,
		"proyecto":      strings.TrimSpace(proyectoSlug),
		"session_id":    strings.TrimSpace(sessionID),
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"subagents":     items,
		"tool_profiles": db.ListSupervisorSubagentToolProfiles(),
		"store":         mustSupervisorSubagentStoreSummary(),
	}, nil
}

func buildSupervisorSubagentsOverview(supervisor string) (string, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorSubagentsSnapshot(supervisor, "", "", 100)
	if err != nil {
		return "", err
	}
	items, _ := snapshot["subagents"].([]*db.SupervisorSubagent)
	var b strings.Builder
	fmt.Fprintf(&b, "# Subagentes del supervisor: %s\n\n", supervisor)
	if len(items) == 0 {
		b.WriteString("- Sin subagentes explícitos registrados.\n")
		return b.String(), nil
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		fmt.Fprintf(&b, "- %s", strings.TrimSpace(item.ThreadID))
		if name := strings.TrimSpace(item.SubagentName); name != "" {
			fmt.Fprintf(&b, " nombre=%s", name)
		}
		fmt.Fprintf(&b, " tipo=%s estado=%s", strings.TrimSpace(item.SubagentType), strings.TrimSpace(item.Status))
		if parent := strings.TrimSpace(item.ParentThreadID); parent != "" {
			fmt.Fprintf(&b, " padre=%s", parent)
		}
		if project := strings.TrimSpace(item.ProyectoSlug); project != "" {
			fmt.Fprintf(&b, " proyecto=%s", project)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()) + "\n", nil
}

func mustSupervisorSubagentStoreSummary() supervisorSubagentStoreSummary {
	summary, _, err := inspectClaudeSubagentStore()
	if err != nil {
		return supervisorSubagentStoreSummary{}
	}
	return summary
}
