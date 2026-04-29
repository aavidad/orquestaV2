package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/notificaciones"
)

var openClawOperatorSnapshotTimeout = 1200 * time.Millisecond
var openClawOperatorSnapshotCoreBuilder = buildOpenClawOperatorSnapshotCore
var openClawOperatorReviewSnapshotBuilder = buildOpenClawReviewSnapshotSafeWithStatus
var openClawOperatorRichDefault = false

func mcpOpenClawOperatorTool() mcpTool {
	return mcpTool{
		Name:        "orquesta.openclaw.operator",
		Title:       "Snapshot operador OpenClaw",
		Description: "Devuelve el snapshot canónico de operador OpenClaw para decisión remota por MCP",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"supervisor": map[string]any{"type": "string"},
				"rich":       map[string]any{"type": "boolean"},
			},
			"additionalProperties": false,
		},
	}
}

func buildOpenClawOperatorSnapshotFallback(supervisor string, apiStatus apiStatusResponse) map[string]any {
	status := buildOpenClawBaseStatusFromAPIStatus(apiStatus)
	statusResumen := buildOpenClawOperatorStatusBase(status, nil)
	operationalInfo := buildOpenClawOperationalInfoWithStatus(apiStatus)
	actionQueue, safeActionQueue, nextAction, nextSafeAction := fillOpenClawOperationalQueuesFromStatusFallback(apiStatus, statusResumen.MailboxPendiente, nil, nil, nil, nil)
	nextRecoveryAction := buildOpenClawNextRecoveryAction(operationalInfo)
	queueSummary := buildOpenClawQueueSummaryFromActions(actionQueue, safeActionQueue)
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
		"review":                     map[string]any{},
		"next_action":                nextAction,
		"next_recovery_action":       nextRecoveryAction,
		"action_queue":               actionQueue,
		"next_safe_action":           nextSafeAction,
		"safe_action_queue":          safeActionQueue,
		"queue_summary":              queueSummary,
		"capacity_summary":           statusResumen.CapacitySummary,
		"saturated_agents":           statusResumen.AgentesSaturados,
		"notificaciones":             notificaciones.EstadoNotificaciones{},
		"entregas":                   notificaciones.OutboxSummary{},
		"eventos_normalizados":       []openClawNormalizedEvent{},
		"thread_sessions":            map[string]any{},
		"subagentes":                 []map[string]any{},
		"subagent_store":             supervisorSubagentStoreSummary{},
		"subagent_profiles":          []db.SupervisorSubagentToolProfile{},
		"session_candidates":         []apiOpenClawSessionCandidate{},
		"worktree_drift":             []apiOpenClawWorktreeDrift{},
		"pipeline_state":             map[string]any{},
		"pipeline_followup":          map[string]any{},
		"subagentes_followup":        []map[string]any{},
	}
}

func buildOpenClawOperatorSnapshotBestEffort(supervisor string, apiStatus apiStatusResponse) map[string]any {
	if snapshot, err := runAPITimeboxed(openClawOperatorSnapshotTimeout, func() (map[string]any, error) {
		return openClawOperatorSnapshotCoreBuilder(supervisor, apiStatus), nil
	}, errStatusFetchTimeout); err == nil && snapshot != nil {
		return snapshot
	}
	return buildOpenClawOperatorSnapshotFallback(supervisor, apiStatus)
}

func buildOpenClawOperatorSnapshot(supervisor string, apiStatus apiStatusResponse, rich bool) map[string]any {
	if !rich {
		return buildOpenClawOperatorSnapshotFallback(supervisor, apiStatus)
	}
	return buildOpenClawOperatorSnapshotBestEffort(supervisor, apiStatus)
}

func buildMCPOpenClawOperatorSnapshot(supervisor string, rich bool) (map[string]any, error) {
	supervisor = resolveOpenClawOperatorSupervisor(supervisor)
	apiStatus := resolveOpenClawAPIStatus()
	if !rich && openClawOperatorRichDefault {
		rich = true
	}
	return buildOpenClawOperatorSnapshot(supervisor, apiStatus, rich), nil
}

func buildOpenClawOperatorSnapshotCore(supervisor string, apiStatus apiStatusResponse) map[string]any {
	status := buildOpenClawBaseStatusFromAPIStatus(apiStatus)
	revision := openClawOperatorReviewSnapshotBuilder(supervisor, apiStatus)
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

	subagentFollowups := buildOpenClawSubagentFollowupsCompact(subagents)
	pipelineFollowup := buildOpenClawPipelineFollowupCompact(pipeline)
	if len(pipelineFollowup) == 0 && len(subagentFollowups) > 0 {
		pipelineFollowup = buildOpenClawPipelineFollowupFromSubagents(subagentFollowups)
	}
	if observed, ok := threads["observed_agent_sessions"].([]*supervisorObservedAgentSessionSummary); ok {
		threads["observed_agent_sessions"] = alignSupervisorObservedSessionsWithStatus(observed, status.AgentesActivos)
	}

	reviewCompact := buildOpenClawReviewCompact(revision)
	actionQueue := supervisorActionQueueFromReviewOrPipelineSnapshot(revision, pipeline)
	safeActionQueue := supervisorSafeActionQueueFromReviewSnapshot(revision)
	nextAction := supervisorNextActionFromReviewOrPipelineSnapshot(revision, pipeline)
	nextSafeAction := supervisorNextSafeActionFromReviewSnapshot(revision)
	operationalInfo := buildOpenClawOperationalInfoWithStatus(apiStatus)
	actionQueue, safeActionQueue, nextAction, nextSafeAction = fillOpenClawOperationalQueuesFromStatusFallback(apiStatus, statusResumen.MailboxPendiente, actionQueue, safeActionQueue, nextAction, nextSafeAction)
	nextRecoveryAction := buildOpenClawNextRecoveryAction(operationalInfo)
	queueSummary := buildOpenClawQueueSummaryFromActions(actionQueue, safeActionQueue)
	sessionCandidates := alignOpenClawSessionCandidatesWithStatus(buildOpenClawSessionCandidates(threads), status.AgentesActivos)

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
		"next_action":                nextAction,
		"next_recovery_action":       nextRecoveryAction,
		"action_queue":               actionQueue,
		"next_safe_action":           nextSafeAction,
		"safe_action_queue":          safeActionQueue,
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
	}
}

func buildOpenClawNextRecoveryAction(info serverOperationalInfo) *supervisorRecommendedAction {
	recovery := info.Recovery
	if recovery == nil || strings.TrimSpace(recovery.SuggestedAction) == "" {
		return nil
	}
	action := &supervisorRecommendedAction{
		Kind:     "recovery",
		Target:   "server:operational",
		Action:   strings.TrimSpace(recovery.SuggestedAction),
		Reason:   strings.TrimSpace(recovery.Detail),
		Priority: "media",
		Assignee: resolveSupervisorName(""),
	}
	switch strings.TrimSpace(recovery.Kind) {
	case "worker_gap", "stuck_workers":
		action.Priority = "alta"
	case "quota_cooldown":
		action.Priority = "baja"
	}
	if strings.TrimSpace(action.Reason) == "" {
		action.Reason = strings.TrimSpace(info.Reason)
	}
	return action
}

func callMCPOpenClawOperator(args map[string]any) (map[string]any, error) {
	rich, _ := boolArg(args, "rich")
	payload, err := buildMCPOpenClawOperatorSnapshot(optionalStringArg(args, "supervisor"), rich)
	if err != nil {
		return nil, err
	}
	return toolResult(prettyJSON(payload), payload, false), nil
}

func buildMCPOpenClawOperatorOverview(supervisor string) (string, error) {
	payload, err := buildMCPOpenClawOperatorSnapshot(supervisor, false)
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
