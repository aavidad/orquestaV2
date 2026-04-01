package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

func resolveSupervisorName(supervisor string) string {
	supervisor = strings.TrimSpace(supervisor)
	if supervisor != "" {
		return supervisor
	}
	if cfg, err := db.ConfigGet("openclaw_gateway_operator"); err == nil && strings.TrimSpace(cfg) != "" {
		return strings.TrimSpace(cfg)
	}
	return "OpenClaw"
}

func registrarSupervisorThreadLigero(supervisor, proyectoSlug, sessionID, threadID, kind, mode, status, source, turnID string) (*db.SupervisorThread, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, nil
	}
	supervisor = resolveSupervisorName(supervisor)
	if strings.TrimSpace(kind) == "" {
		kind = "subagent"
	}
	if strings.TrimSpace(status) == "" {
		status = "active"
	}
	if strings.TrimSpace(source) == "" {
		source = "server"
	}
	return db.RecordSupervisorThreadTurn(db.RecordSupervisorThreadInput{
		Supervisor:   supervisor,
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		SessionID:    strings.TrimSpace(sessionID),
		ThreadID:     threadID,
		Kind:         strings.TrimSpace(kind),
		Mode:         strings.TrimSpace(mode),
		Status:       strings.TrimSpace(status),
		Source:       strings.TrimSpace(source),
		TurnID:       strings.TrimSpace(turnID),
		Timestamp:    time.Now().UTC(),
	})
}

func buildSupervisorThreadsSnapshot(supervisor, sessionID string, limit int) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	if limit <= 0 {
		limit = 100
	}
	items, err := db.ListarSupervisorThreads(db.FiltroSupervisorThreads{
		Supervisor: strings.TrimSpace(supervisor),
		SessionID:  strings.TrimSpace(sessionID),
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	sessionIndex := map[string][]*db.SupervisorThread{}
	for _, item := range items {
		if item == nil {
			continue
		}
		key := strings.TrimSpace(item.SessionID)
		if key == "" {
			key = "_default"
		}
		sessionIndex[key] = append(sessionIndex[key], item)
	}
	sessionKeys := make([]string, 0, len(sessionIndex))
	for key := range sessionIndex {
		sessionKeys = append(sessionKeys, key)
	}
	sort.Strings(sessionKeys)
	summaries := make([]*db.SupervisorThreadSessionSummary, 0, len(sessionKeys))
	for _, key := range sessionKeys {
		actualSessionID := key
		if key == "_default" {
			actualSessionID = ""
		}
		summary, err := db.SummarizeSupervisorThreadSession(supervisor, actualSessionID, db.DefaultSupervisorThreadActiveWindow)
		if err != nil {
			return nil, err
		}
		if summary == nil {
			continue
		}
		if actualSessionID == "" {
			summary.SessionID = ""
		}
		summaries = append(summaries, summary)
	}
	return map[string]any{
		"supervisor":      supervisor,
		"session_id":      strings.TrimSpace(sessionID),
		"active_window_s": int(db.DefaultSupervisorThreadActiveWindow / time.Second),
		"sessions":        summaries,
		"threads":         items,
	}, nil
}

func buildSupervisorThreadsOverview(supervisor string) (string, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorThreadsSnapshot(supervisor, "", 100)
	if err != nil {
		return "", err
	}
	summaries, _ := snapshot["sessions"].([]*db.SupervisorThreadSessionSummary)
	var b strings.Builder
	fmt.Fprintf(&b, "## Threads y subagentes del supervisor\n")
	if len(summaries) == 0 {
		b.WriteString("- Sin threads registradas todavía.\n")
		return b.String(), nil
	}
	for _, summary := range summaries {
		if summary == nil {
			continue
		}
		sessionLabel := strings.TrimSpace(summary.SessionID)
		if sessionLabel == "" {
			sessionLabel = "<default>"
		}
		fmt.Fprintf(&b, "- sesión=%s", sessionLabel)
		if strings.TrimSpace(summary.LeaderThreadID) != "" {
			fmt.Fprintf(&b, " líder=%s", strings.TrimSpace(summary.LeaderThreadID))
		}
		fmt.Fprintf(&b, " subagentes=%d activos=%d\n", len(summary.AllSubagentThreadIDs), len(summary.ActiveSubagentThreadIDs))
	}
	return strings.TrimSpace(b.String()), nil
}
