package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type supervisorObservedAgentSessionSummary struct {
	Agente                 string     `json:"agente"`
	Activo                 bool       `json:"activo"`
	Herramienta            string     `json:"herramienta,omitempty"`
	Host                   string     `json:"host,omitempty"`
	ExternalSessionID      string     `json:"external_session_id,omitempty"`
	ObservedSessionPath    string     `json:"observed_session_path,omitempty"`
	ObservedUsageMessages  *int       `json:"observed_usage_messages,omitempty"`
	ObservedUsageTurns     *int       `json:"observed_usage_turns,omitempty"`
	ObservedUsageUpdatedAt *time.Time `json:"observed_usage_updated_at,omitempty"`
	UsageSummary           string     `json:"usage_summary,omitempty"`
}

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
	observedSessions, err := buildSupervisorObservedAgentSessions()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"supervisor":               supervisor,
		"session_id":               strings.TrimSpace(sessionID),
		"active_window_s":          int(db.DefaultSupervisorThreadActiveWindow / time.Second),
		"sessions":                 summaries,
		"threads":                  items,
		"observed_agent_sessions":  observedSessions,
	}, nil
}

func buildSupervisorObservedAgentSessions() ([]*supervisorObservedAgentSessionSummary, error) {
	agentes, err := db.ListarAgentes()
	if err != nil {
		return nil, err
	}
	sesionesActivas, err := db.ListarSesionesActivasOperativas()
	if err != nil {
		return nil, err
	}
	sesionesPorAgente := map[string]*db.Sesion{}
	for _, sesion := range sesionesActivas {
		if sesion == nil {
			continue
		}
		agente := strings.TrimSpace(sesion.Agente)
		if agente == "" {
			continue
		}
		actual := sesionesPorAgente[agente]
		if actual == nil || actual.Inicio.Before(sesion.Inicio) {
			sesionesPorAgente[agente] = sesion
		}
	}
	out := make([]*supervisorObservedAgentSessionSummary, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		item := &supervisorObservedAgentSessionSummary{
			Agente:                 strings.TrimSpace(agente.Nombre),
			Activo:                 agente.Activo,
			ObservedSessionPath:    strings.TrimSpace(agente.ObservedSessionPath),
			ObservedUsageMessages:  agente.ObservedUsageMessages,
			ObservedUsageTurns:     agente.ObservedUsageTurns,
			ObservedUsageUpdatedAt: agente.ObservedUsageUpdatedAt,
		}
		if sesion := sesionesPorAgente[item.Agente]; sesion != nil {
			item.Herramienta = strings.TrimSpace(sesion.Herramienta)
			item.Host = strings.TrimSpace(sesion.Host)
			item.ExternalSessionID = strings.TrimSpace(sesion.ExternalSessionID)
		}
		usage := make([]string, 0, 2)
		if item.ObservedUsageMessages != nil {
			usage = append(usage, fmt.Sprintf("%d msg", *item.ObservedUsageMessages))
		}
		if item.ObservedUsageTurns != nil {
			usage = append(usage, fmt.Sprintf("%d turns", *item.ObservedUsageTurns))
		}
		item.UsageSummary = strings.Join(usage, " · ")
		if item.ExternalSessionID == "" && item.ObservedSessionPath == "" {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Agente) < strings.ToLower(out[j].Agente)
	})
	return out, nil
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
