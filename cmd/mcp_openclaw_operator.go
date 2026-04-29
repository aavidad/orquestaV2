package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/notificaciones"
)

func mcpOpenClawOperatorTool() mcpTool {
	return mcpTool{
		Name:        "orquesta.openclaw.operator",
		Title:       "Snapshot operador OpenClaw",
		Description: "Devuelve el snapshot canónico de operador OpenClaw para decisión remota por MCP",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"supervisor": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		},
	}
}

func buildMCPOpenClawOperatorSnapshot(supervisor string) (map[string]any, error) {
	supervisor = resolveOpenClawOperatorSupervisor(supervisor)
	status := buildOpenClawBaseStatus()
	revision := buildOpenClawReviewSnapshotSafe(supervisor)
	mailboxPendiente, mailboxKnown := openClawPendingMailboxFromReviewSnapshot(revision)

	var panelRows []agentesapp.Row
	if rows, err := runAPITimeboxed(200*time.Millisecond, func() ([]agentesapp.Row, error) {
		return fetchAgentPanelRowsCached(150 * time.Millisecond)
	}, errStatusFetchTimeout); err == nil {
		panelRows = rows
	}

	statusResumen := buildOpenClawOperatorStatusBase(status, panelRows)
	if summary, err := runAPITimeboxed(200*time.Millisecond, func() (apiOpenClawStatusLite, error) {
		return buildOpenClawOperatorStatusWithRowsAndMailbox(status, panelRows, mailboxPendiente, mailboxKnown)
	}, errStatusFetchTimeout); err == nil {
		statusResumen = summary
	}

	eventos := openClawEventsFromReviewSnapshot(revision)
	threads := supervisorThreadsFromReviewSnapshot(revision)
	pipeline := supervisorPipelineFromReviewSnapshot(revision)
	worktreeDrift := openClawWorktreeDriftFromReviewSnapshot(revision)
	subagents := supervisorSubagentsFromReviewSnapshot(revision)
	if len(subagents) == 0 {
		if snapshot, err := runAPITimeboxed(150*time.Millisecond, func() (map[string]any, error) {
			return buildSupervisorSubagentsSnapshot(supervisor, "", "", 100)
		}, errStatusFetchTimeout); err == nil && snapshot != nil {
			subagents = snapshot
		}
	}
	if len(pipeline) == 0 {
		if snapshot, err := runAPITimeboxed(150*time.Millisecond, func() (map[string]any, error) {
			return buildSupervisorPipelineSnapshot(supervisor, "", 20)
		}, errStatusFetchTimeout); err == nil && snapshot != nil {
			pipeline = snapshot
		}
	}

	subagentFollowups := buildOpenClawSubagentFollowupsCompact(subagents)
	if len(subagentFollowups) == 0 {
		subagentFollowups = buildOpenClawSubagentFollowupsDirect(supervisor)
	}
	pipelineFollowup := buildOpenClawPipelineFollowupCompact(pipeline)
	if len(pipelineFollowup) == 0 && len(subagentFollowups) > 0 {
		pipelineFollowup = buildOpenClawPipelineFollowupFromSubagents(subagentFollowups)
	}
	if observed, ok := threads["observed_agent_sessions"].([]*supervisorObservedAgentSessionSummary); ok {
		threads["observed_agent_sessions"] = alignSupervisorObservedSessionsWithStatus(observed, status.AgentesActivos)
	}

	reviewCompact := buildOpenClawReviewCompact(revision)
	queueSummary := buildOpenClawQueueSummaryFromReviewSnapshot(revision)
	sessionCandidates := alignOpenClawSessionCandidatesWithStatus(buildOpenClawSessionCandidates(threads), status.AgentesActivos)
	operationalInfo := buildOpenClawOperationalInfo()

	estadoNotifs := notificaciones.EstadoNotificaciones{}
	if estado, err := runAPITimeboxed(100*time.Millisecond, func() (notificaciones.EstadoNotificaciones, error) {
		return notificaciones.DescribirConfiguracion(), nil
	}, errStatusFetchTimeout); err == nil {
		estadoNotifs = estado
	}
	entregas := notificaciones.OutboxSummary{}
	if outbox, err := runAPITimeboxed(100*time.Millisecond, func() (notificaciones.OutboxSummary, error) {
		return notificaciones.DescribirOutbox(10), nil
	}, errStatusFetchTimeout); err == nil {
		entregas = outbox
	}

	return map[string]any{
		"status":                     statusResumen,
		"server_operational":         operationalInfo,
		"server_operational_summary": buildOpenClawOperationalSummary(operationalInfo),
		"agentesActivos":             statusResumen.AgentesActivos,
		"agentesTrabajando":          statusResumen.AgentesTrabajando,
		"agentesAuthManual":          statusResumen.AgentesAuthManual,
		"agentesQuotaBlocked":        statusResumen.AgentesQuotaBlocked,
		"enCuota":                    statusResumen.EnCuota,
		"mailboxPendiente":           statusResumen.MailboxPendiente,
		"retenidasPorCuota":          statusResumen.RetenidasPorCuota,
		"tareasActivas":              statusResumen.TareasActivas,
		"tareasReservadas":           statusResumen.TareasReservadas,
		"propuestasAbiertas":         statusResumen.PropuestasAbiertas,
		"review":                     reviewCompact,
		"next_action":                revision["next_action"],
		"action_queue":               revision["action_queue"],
		"next_safe_action":           revision["next_safe_action"],
		"safe_action_queue":          revision["safe_action_queue"],
		"queue_summary":              queueSummary,
		"capacity_summary":           statusResumen.CapacitySummary,
		"saturated_agents":           statusResumen.AgentesSaturados,
		"notificaciones":             estadoNotifs,
		"entregas":                   entregas,
		"eventos_normalizados":       eventos,
		"thread_sessions":            threads,
		"subagentes":                 subagents["subagents"],
		"subagent_store":             subagents["store"],
		"subagent_profiles":          subagents["tool_profiles"],
		"session_candidates":         sessionCandidates,
		"worktree_drift":             worktreeDrift,
		"pipeline_state":             pipeline,
		"pipeline_followup":          pipelineFollowup,
		"subagentes_followup":        subagentFollowups,
	}, nil
}

func callMCPOpenClawOperator(args map[string]any) (map[string]any, error) {
	payload, err := buildMCPOpenClawOperatorSnapshot(optionalStringArg(args, "supervisor"))
	if err != nil {
		return nil, err
	}
	return toolResult(prettyJSON(payload), payload, false), nil
}

func buildMCPOpenClawOperatorOverview(supervisor string) (string, error) {
	payload, err := buildMCPOpenClawOperatorSnapshot(supervisor)
	if err != nil {
		return "", err
	}

	supervisor = resolveOpenClawOperatorSupervisor(supervisor)
	status, _ := payload["status"].(apiOpenClawStatusLite)
	operational, _ := payload["server_operational"].(serverOperationalInfo)
	queueSummary, _ := payload["queue_summary"].(apiOpenClawQueueSummary)
	sessionCandidates, _ := payload["session_candidates"].([]apiOpenClawSessionCandidate)
	mailboxPendiente, _ := payload["mailboxPendiente"].([]apiOpenClawMailboxLite)
	nextSafeAction, _ := parseSupervisorRecommendedActionSnapshotValue(payload["next_safe_action"])

	var b strings.Builder
	b.WriteString("Snapshot operador OpenClaw: ")
	b.WriteString(supervisor)
	b.WriteString("\n\n")
	b.WriteString("Operación:\n")
	b.WriteString("- ")
	b.WriteString(compactMCPLine(buildOpenClawOperationalSummary(operational), 220))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("- agentes_activos=%d · agentes_trabajando=%d · quota=%d · auth_manual=%d\n",
		len(status.AgentesActivos),
		len(status.AgentesTrabajando),
		len(status.AgentesQuotaBlocked),
		len(status.AgentesAuthManual),
	))
	b.WriteString(fmt.Sprintf("- tareas_activas=%d · reservadas=%d · mailbox_pendiente=%d · candidatos_sesion=%d\n",
		len(status.TareasActivas),
		len(status.TareasReservadas),
		len(mailboxPendiente),
		len(sessionCandidates),
	))
	b.WriteString(fmt.Sprintf("- queue_safe=%d · queue_manual=%d · saturated_agents=%d\n",
		queueSummary.Safe,
		queueSummary.Manual,
		len(status.AgentesSaturados),
	))

	if nextSafeAction != nil {
		b.WriteString("\nSiguiente acción segura:\n")
		b.WriteString("- ")
		b.WriteString(strings.TrimSpace(nextSafeAction.Target))
		b.WriteString(" -> ")
		b.WriteString(strings.TrimSpace(nextSafeAction.Action))
		if note := compactMCPLine(strings.TrimSpace(nextSafeAction.Reason), 180); note != "" {
			b.WriteString(" · ")
			b.WriteString(note)
		}
		b.WriteString("\n")
	}

	if len(mailboxPendiente) > 0 {
		first := mailboxPendiente[0]
		b.WriteString("\nMailbox más vieja:\n")
		b.WriteString(fmt.Sprintf("- %s · count=%d", strings.TrimSpace(first.Agente), first.Count))
		if strings.TrimSpace(first.ContextsCSV) != "" {
			b.WriteString(" · ")
			b.WriteString(strings.TrimSpace(first.ContextsCSV))
		}
		b.WriteString("\n")
	}

	b.WriteString("\nUso:\n")
	b.WriteString("- Esta surface sirve como snapshot único para decidir acción por MCP sin reconstruir `/api/openclaw/operator`.\n")
	b.WriteString("- Si necesitas actuar, enlázala con `orquesta.supervision.*`, `orquesta.agentes.*` y `orquesta.runtime.*`.\n")
	return b.String(), nil
}
