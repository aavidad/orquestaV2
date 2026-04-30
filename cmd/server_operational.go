package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

var (
	serverOperationalListAgentsFetcher = db.ListarAgentesEstadoLigero
	serverOperationalCountTasksFetcher = db.ContarTareasPorEstado
	serverOperationalDispatchFetcher   = statusDispatchSummaryFetcher
	serverOperationalOptionalTimeout   = statusOptionalSectionTimeout
	serverOperationalReviewSnapshotFn  = buildSupervisorReviewSnapshot
	serverOperationalApplyNextActionFn = applySupervisorNextAction
)

type serverOperationalInfo struct {
	State                string                           `json:"state"`
	Operational          bool                             `json:"operational"`
	Reason               string                           `json:"reason,omitempty"`
	NextQuotaResetAt     string                           `json:"nextQuotaResetAt,omitempty"`
	Generated            string                           `json:"generated,omitempty"`
	AutonomyHighlights   []string                         `json:"autonomyHighlights,omitempty"`
	CriticalProjectRisk  *workspaceAutonomyProjectSummary `json:"criticalProjectRisk,omitempty"`
	Recovery             *serverOperationalRecoveryHint   `json:"recovery,omitempty"`
	NextRecoveryPlan     *serverOperationalRecoveryPlan   `json:"nextRecoveryPlan,omitempty"`
	RegisteredAgents     int                              `json:"registeredAgents"`
	ActiveAgents         int                              `json:"activeAgents"`
	WorkingAgents        int                              `json:"workingAgents"`
	ConnectedWorkers     int                              `json:"connectedWorkers"`
	WorkingWorkers       int                              `json:"workingWorkers"`
	SaturatedAgents      int                              `json:"saturatedAgents"`
	StuckAgents          int                              `json:"stuckAgents"`
	AuthAgents           int                              `json:"authAgents"`
	QuotaAgents          int                              `json:"quotaAgents"`
	PausedAgents         int                              `json:"pausedAgents"`
	TasksInProgress      int                              `json:"tasksInProgress"`
	ReservedTasks        int                              `json:"reservedTasks"`
	BlockedTasks         int                              `json:"blockedTasks"`
	CompactionDebtAgents int                              `json:"compactionDebtAgents"`
	CompactionDebtTasks  int                              `json:"compactionDebtTasks"`
	DispatchPending      int                              `json:"dispatchPending"`
	DispatchNotified     int                              `json:"dispatchNotified"`
	DispatchFailed       int                              `json:"dispatchFailed"`
	DispatchConfirmed    int                              `json:"dispatchConfirmed"`
	AutonomySupervising  int                              `json:"autonomySupervising"`
	AutonomyContinuing   int                              `json:"autonomyContinuing"`
	AutonomyPending      int                              `json:"autonomyPending"`
	AutonomyConfirmed    int                              `json:"autonomyConfirmed"`
	AutonomyHandoffs     int                              `json:"autonomyHandoffs"`
	Rearm                *serverOperationalRearmHint      `json:"rearm,omitempty"`
}

type serverOperationalRearmHint struct {
	Needed         bool                         `json:"needed"`
	Available      bool                         `json:"available"`
	Supervisor     string                       `json:"supervisor,omitempty"`
	Reason         string                       `json:"reason,omitempty"`
	Tool           string                       `json:"tool,omitempty"`
	Endpoint       string                       `json:"endpoint,omitempty"`
	Method         string                       `json:"method,omitempty"`
	NextSafeAction *supervisorRecommendedAction `json:"nextSafeAction,omitempty"`
}

type serverOperationalRecoveryHint struct {
	Kind               string `json:"kind"`
	AffectedTasks      int    `json:"affectedTasks,omitempty"`
	MissingWorkers     int    `json:"missingWorkers,omitempty"`
	QuotaBlockedAgents int    `json:"quotaBlockedAgents,omitempty"`
	AuthBlockedAgents  int    `json:"authBlockedAgents,omitempty"`
	StuckAgents        int    `json:"stuckAgents,omitempty"`
	RearmAvailable     bool   `json:"rearmAvailable,omitempty"`
	SuggestedAction    string `json:"suggestedAction,omitempty"`
	Detail             string `json:"detail,omitempty"`
}

type serverOperationalRecoveryPlan struct {
	Kind               string                          `json:"kind"`
	Action             string                          `json:"action,omitempty"`
	Target             string                          `json:"target,omitempty"`
	Priority           string                          `json:"priority,omitempty"`
	Assignee           string                          `json:"assignee,omitempty"`
	Summary            string                          `json:"summary,omitempty"`
	Detail             string                          `json:"detail,omitempty"`
	BlockingReason     string                          `json:"blockingReason,omitempty"`
	RequiresRearm      bool                            `json:"requiresRearm,omitempty"`
	RearmAvailable     bool                            `json:"rearmAvailable,omitempty"`
	AutoExecutable     bool                            `json:"autoExecutable,omitempty"`
	Tool               string                          `json:"tool,omitempty"`
	Endpoint           string                          `json:"endpoint,omitempty"`
	Method             string                          `json:"method,omitempty"`
	NextQuotaResetAt   string                          `json:"nextQuotaResetAt,omitempty"`
	AffectedTasks      int                             `json:"affectedTasks,omitempty"`
	MissingWorkers     int                             `json:"missingWorkers,omitempty"`
	QuotaBlockedAgents int                             `json:"quotaBlockedAgents,omitempty"`
	AuthBlockedAgents  int                             `json:"authBlockedAgents,omitempty"`
	StuckAgents        int                             `json:"stuckAgents,omitempty"`
	Steps              []serverOperationalRecoveryStep `json:"steps,omitempty"`
}

type serverOperationalRecoveryStep struct {
	Name           string `json:"name"`
	Action         string `json:"action"`
	Target         string `json:"target,omitempty"`
	Detail         string `json:"detail,omitempty"`
	AutoExecutable bool   `json:"autoExecutable,omitempty"`
}

func registeredAgentCountFromStatus(status apiStatusResponse) int {
	seen := map[string]struct{}{}
	add := func(items []*db.Agente) {
		for _, agente := range items {
			if agente == nil {
				continue
			}
			nombre := strings.ToLower(strings.TrimSpace(agente.Nombre))
			if nombre != "" {
				seen[nombre] = struct{}{}
			}
		}
	}
	add(status.Agentes)
	add(status.AgentesActivos)
	add(status.AgentesTrabajando)
	add(status.AgentesSaturados)
	add(status.AgentesAtascados)
	add(status.AgentesAuthManual)
	add(status.AgentesQuotaBlocked)
	return len(seen)
}

func buildServerOperationalInfo(status apiStatusResponse) serverOperationalInfo {
	_, autonomyHighlights, criticalProjectRisk := serverOperationalAutonomyContext(status)
	tasksInProgress := len(status.TareasEnProgreso)
	if tasksInProgress == 0 && status.TareasPorEstado != nil {
		tasksInProgress = status.TareasPorEstado[string(db.TareaEnProgreso)]
	}
	if tasksInProgress == 0 {
		tasksInProgress = len(filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaEnProgreso))
	}
	reservedTasks := len(status.TareasReservadas)
	if reservedTasks == 0 && status.TareasPorEstado != nil {
		reservedTasks = status.TareasPorEstado[string(db.TareaAsignada)]
	}
	if reservedTasks == 0 {
		reservedTasks = len(filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaAsignada))
	}
	blockedTasks := 0
	if status.TareasPorEstado != nil {
		blockedTasks = status.TareasPorEstado[string(db.TareaBloqueada)]
	}
	quotaBlockedVisible := agentesBloqueadosPorCuotaVisibles(&estadoResumen{
		Agentes:             status.Agentes,
		AgentesQuotaBlocked: status.AgentesQuotaBlocked,
	})
	quotaAgents := len(quotaBlockedVisible)
	nextQuotaResetAt := nextQuotaResetVisible(quotaBlockedVisible)
	pausedAgents := len(agentesNoActivosEnPausaOperativa(status.Agentes))
	activeAgents := len(status.AgentesActivos)
	workingAgents := len(status.AgentesTrabajando)
	activeWorkers, workingWorkers, _ := reconciledVisibleWorkerCounters(
		status.WorkersConectados,
		status.WorkersTrabajando,
		status.SupervisoresActivos,
		status.AgentesActivos,
		status.AgentesTrabajando,
		status.Autonomia,
	)
	saturatedAgents := len(status.AgentesSaturados)
	stuckAgents := len(status.AgentesAtascados)
	authAgents := len(status.AgentesAuthManual)
	compactionDebtAgents, compactionDebtTasks := serverOperationalCompactionDebt(statusNowFunc().UTC())

	state := "ready"
	reason := "control_plane_responsive"
	operational := true
	switch {
	case authAgents > 0:
		state = "degraded"
		reason = "workers_require_manual_auth"
		operational = false
	case stuckAgents > 0:
		state = "degraded"
		reason = "workers_stuck"
		operational = false
	case tasksInProgress > 0 && workingWorkers < tasksInProgress && status.Autonomia.WorkConfirmed < tasksInProgress:
		state = "degraded"
		reason = "tasks_without_workers"
		operational = false
	case activeWorkers == 0 && reservedTasks > 0:
		state = "degraded"
		reason = "reserved_without_connected_workers"
		operational = false
	case activeAgents == 0 && quotaAgents > 0:
		state = "idle"
		reason = "workers_quota_blocked"
	case activeAgents == 0 && tasksInProgress == 0 && reservedTasks == 0:
		state = "idle"
		reason = "no_active_workers"
	}

	return serverOperationalInfo{
		State:                state,
		Operational:          operational,
		Reason:               reason,
		NextQuotaResetAt:     nextQuotaResetAt,
		Generated:            status.Generado,
		AutonomyHighlights:   autonomyHighlights,
		CriticalProjectRisk:  criticalProjectRisk,
		RegisteredAgents:     registeredAgentCountFromStatus(status),
		ActiveAgents:         activeAgents,
		WorkingAgents:        workingAgents,
		ConnectedWorkers:     activeWorkers,
		WorkingWorkers:       workingWorkers,
		SaturatedAgents:      saturatedAgents,
		StuckAgents:          stuckAgents,
		AuthAgents:           authAgents,
		QuotaAgents:          quotaAgents,
		PausedAgents:         pausedAgents,
		TasksInProgress:      tasksInProgress,
		ReservedTasks:        reservedTasks,
		BlockedTasks:         blockedTasks,
		CompactionDebtAgents: compactionDebtAgents,
		CompactionDebtTasks:  compactionDebtTasks,
		DispatchPending:      status.DeudaDispatch.Pendientes,
		DispatchNotified:     status.DeudaDispatch.Notificadas,
		DispatchFailed:       status.DeudaDispatch.Fallidas,
		DispatchConfirmed:    status.DeudaDispatch.WorkConfirmed,
		AutonomySupervising:  status.Autonomia.Supervisando,
		AutonomyContinuing:   status.Autonomia.Continuando,
		AutonomyPending:      status.Autonomia.ContinuidadPendiente,
		AutonomyConfirmed:    status.Autonomia.WorkConfirmed,
		AutonomyHandoffs:     status.Autonomia.Handoffs,
	}
}

func serverOperationalAutonomyContext(status apiStatusResponse) (*autonomySurface, []string, *workspaceAutonomyProjectSummary) {
	surface, autonomyHighlights, criticalProjectRisk := normalizeStatusAutonomyPayload(
		status.AutonomySurface,
		status.AutonomyHighlights,
		status.CriticalProjectRisk,
	)
	if surface != nil || len(autonomyHighlights) > 0 || criticalProjectRisk != nil {
		return surface, autonomyHighlights, criticalProjectRisk
	}
	if statusAutonomySurfaceFetcher == nil {
		return nil, nil, nil
	}
	surface, ok := runStatusOptional(serverOperationalOptionalTimeout, statusAutonomySurfaceFetcher)
	if !ok || surface == nil {
		return nil, nil, nil
	}
	riskSummary := buildStatusWorkspaceRiskSummary(surface)
	surface, riskSummary = canonicalizeStatusAutonomyRisk(surface, riskSummary)
	return normalizeStatusAutonomyPayload(surface, riskSummary.Highlights, riskSummary.CriticalProjectRisk)
}

func serverOperationalRiskContext(info *serverOperationalInfo) ([]string, *workspaceAutonomyProjectSummary) {
	if info == nil {
		return nil, nil
	}
	_, autonomyHighlights, criticalProjectRisk := normalizeStatusAutonomyPayload(
		nil,
		info.AutonomyHighlights,
		info.CriticalProjectRisk,
	)
	return autonomyHighlights, criticalProjectRisk
}

func normalizeServerOperationalInfo(info serverOperationalInfo) serverOperationalInfo {
	autonomyHighlights, criticalProjectRisk := serverOperationalRiskContext(&info)
	info.AutonomyHighlights = autonomyHighlights
	info.CriticalProjectRisk = criticalProjectRisk
	info.Rearm = buildServerOperationalRearmHint(info)
	info.Recovery = buildServerOperationalRecoveryHint(info)
	info.NextRecoveryPlan = buildServerOperationalRecoveryPlan(info, buildOpenClawNextRecoveryAction(info))
	return info
}

func buildServerOperationalRecoveryHint(info serverOperationalInfo) *serverOperationalRecoveryHint {
	if info.Operational &&
		strings.TrimSpace(info.Reason) == "control_plane_responsive" &&
		info.CompactionDebtTasks > 0 {
		return &serverOperationalRecoveryHint{
			Kind:            "compaction_debt",
			AffectedTasks:   info.CompactionDebtTasks,
			SuggestedAction: "compact_or_reassign_active_tasks",
			Detail:          fmt.Sprintf("%d tarea(s) abiertas exceden la señal real de trabajo en %d agente(s)", max(info.CompactionDebtTasks, 1), max(info.CompactionDebtAgents, 1)),
		}
	}
	reason := strings.TrimSpace(info.Reason)
	if reason == "" {
		return nil
	}
	withFallback := func(value, fallback string) string {
		value = strings.TrimSpace(value)
		if value == "" {
			return fallback
		}
		return value
	}
	hint := &serverOperationalRecoveryHint{
		QuotaBlockedAgents: info.QuotaAgents,
		AuthBlockedAgents:  info.AuthAgents,
		StuckAgents:        info.StuckAgents,
	}
	if info.Rearm != nil && info.Rearm.Needed {
		hint.RearmAvailable = info.Rearm.Available
	}
	switch reason {
	case "tasks_without_workers":
		hint.Kind = "worker_gap"
		hint.AffectedTasks = max(info.TasksInProgress, 0)
		hint.MissingWorkers = max(info.TasksInProgress-max(info.WorkingWorkers, info.AutonomyConfirmed), 0)
		if hint.MissingWorkers == 0 && hint.AffectedTasks > 0 {
			hint.MissingWorkers = hint.AffectedTasks
		}
		if hint.RearmAvailable {
			hint.SuggestedAction = "server_rearm"
			hint.Detail = fmt.Sprintf("%d tarea(s) en progreso sin worker útil visible; hay rearm seguro disponible", max(hint.AffectedTasks, 1))
			return hint
		}
		if info.AuthAgents > 0 {
			hint.SuggestedAction = "complete_manual_auth"
			hint.Detail = fmt.Sprintf("%d worker(s) requieren autenticación manual antes de retomar el trabajo", info.AuthAgents)
			return hint
		}
		if info.ConnectedWorkers > info.WorkingWorkers {
			hint.SuggestedAction = "inspect_connected_idle_workers"
			hint.Detail = fmt.Sprintf("%d worker(s) conectados pero solo %d trabajando; conviene consumir primero la capacidad ya visible antes de esperar cuota", info.ConnectedWorkers, info.WorkingWorkers)
			return hint
		}
		if info.QuotaAgents > 0 {
			hint.SuggestedAction = "wait_quota_reset"
			hint.Detail = fmt.Sprintf("%d worker(s) bloqueados por cuota; próximo reset visible %s", info.QuotaAgents, withFallback(info.NextQuotaResetAt, "pendiente"))
			return hint
		}
		if info.StuckAgents > 0 {
			hint.SuggestedAction = "inspect_stuck_workers"
			hint.Detail = fmt.Sprintf("%d worker(s) atascados siguen bloqueando %d tarea(s) en progreso", info.StuckAgents, max(hint.AffectedTasks, 1))
			return hint
		}
		if info.ConnectedWorkers > 0 {
			hint.SuggestedAction = "inspect_connected_idle_workers"
			hint.Detail = fmt.Sprintf("%d worker(s) conectados pero solo %d trabajando; revisar runtime, resume o start", info.ConnectedWorkers, info.WorkingWorkers)
			return hint
		}
		hint.SuggestedAction = "start_or_assign_workers"
		hint.Detail = fmt.Sprintf("%d tarea(s) en progreso sin worker visible", max(hint.AffectedTasks, 1))
		return hint
	case "reserved_without_connected_workers":
		hint.Kind = "reserved_gap"
		hint.AffectedTasks = max(info.ReservedTasks, 0)
		hint.MissingWorkers = max(info.ReservedTasks-info.ConnectedWorkers, 0)
		if hint.MissingWorkers == 0 && hint.AffectedTasks > 0 {
			hint.MissingWorkers = hint.AffectedTasks
		}
		if hint.RearmAvailable {
			hint.SuggestedAction = "server_rearm"
			hint.Detail = fmt.Sprintf("%d tarea(s) reservadas sin worker conectado; hay rearm seguro disponible", max(hint.AffectedTasks, 1))
			return hint
		}
		if info.QuotaAgents > 0 {
			hint.SuggestedAction = "wait_quota_reset"
			hint.Detail = fmt.Sprintf("%d worker(s) bloqueados por cuota; próximo reset visible %s", info.QuotaAgents, withFallback(info.NextQuotaResetAt, "pendiente"))
			return hint
		}
		hint.SuggestedAction = "start_or_assign_workers"
		hint.Detail = fmt.Sprintf("%d tarea(s) reservadas esperan worker conectado", max(hint.AffectedTasks, 1))
		return hint
	case "workers_require_manual_auth":
		hint.Kind = "manual_auth"
		hint.SuggestedAction = "complete_manual_auth"
		hint.Detail = fmt.Sprintf("%d worker(s) requieren autenticación manual", max(info.AuthAgents, 1))
		return hint
	case "workers_stuck":
		hint.Kind = "stuck_workers"
		if hint.RearmAvailable {
			hint.SuggestedAction = "server_rearm"
			hint.Detail = fmt.Sprintf("%d worker(s) atascados; hay rearm seguro disponible", max(info.StuckAgents, 1))
			return hint
		}
		hint.SuggestedAction = "inspect_stuck_workers"
		hint.Detail = fmt.Sprintf("%d worker(s) atascados requieren inspección", max(info.StuckAgents, 1))
		return hint
	case "workers_quota_blocked":
		hint.Kind = "quota_cooldown"
		hint.SuggestedAction = "wait_quota_reset"
		hint.Detail = fmt.Sprintf("%d worker(s) bloqueados por cuota; próximo reset visible %s", max(info.QuotaAgents, 1), withFallback(info.NextQuotaResetAt, "pendiente"))
		return hint
	default:
		if !hint.RearmAvailable {
			return nil
		}
		hint.Kind = "rearm"
		hint.SuggestedAction = "server_rearm"
		if info.Rearm != nil && strings.TrimSpace(info.Rearm.Reason) != "" {
			hint.Detail = strings.TrimSpace(info.Rearm.Reason)
		} else {
			hint.Detail = fmt.Sprintf("estado %s degradado con rearm seguro disponible", withFallback(reason, "operational"))
		}
		return hint
	}
}

func buildServerOperationalRecoveryPlan(info serverOperationalInfo, action *supervisorRecommendedAction) *serverOperationalRecoveryPlan {
	recovery := info.Recovery
	if recovery == nil {
		return nil
	}
	kind := strings.TrimSpace(recovery.Kind)
	plan := &serverOperationalRecoveryPlan{
		Kind:               kind,
		Action:             strings.TrimSpace(recovery.SuggestedAction),
		Target:             "server:operational",
		BlockingReason:     strings.TrimSpace(info.Reason),
		Detail:             strings.TrimSpace(recovery.Detail),
		RearmAvailable:     recovery.RearmAvailable,
		NextQuotaResetAt:   strings.TrimSpace(info.NextQuotaResetAt),
		AffectedTasks:      recovery.AffectedTasks,
		MissingWorkers:     recovery.MissingWorkers,
		QuotaBlockedAgents: recovery.QuotaBlockedAgents,
		AuthBlockedAgents:  recovery.AuthBlockedAgents,
		StuckAgents:        recovery.StuckAgents,
	}
	if action != nil {
		if value := strings.TrimSpace(action.Action); value != "" {
			plan.Action = value
		}
		if value := strings.TrimSpace(action.Target); value != "" {
			plan.Target = value
		}
		if value := strings.TrimSpace(action.Priority); value != "" {
			plan.Priority = value
		}
		if value := strings.TrimSpace(action.Assignee); value != "" {
			plan.Assignee = value
		}
		if plan.Detail == "" {
			plan.Detail = strings.TrimSpace(action.Reason)
		}
	}
	if plan.Action == "" {
		return nil
	}
	if plan.Priority == "" {
		plan.Priority = defaultServerOperationalRecoveryPriority(kind)
	}
	if plan.Assignee == "" {
		plan.Assignee = resolveSupervisorName("")
	}
	plan.RequiresRearm = plan.Action == "server_rearm"
	plan.AutoExecutable = plan.RequiresRearm && plan.RearmAvailable
	if plan.RequiresRearm && info.Rearm != nil {
		plan.Tool = strings.TrimSpace(info.Rearm.Tool)
		plan.Endpoint = strings.TrimSpace(info.Rearm.Endpoint)
		plan.Method = strings.TrimSpace(info.Rearm.Method)
	}
	plan.Summary = buildServerOperationalRecoveryPlanSummary(plan)
	plan.Steps = buildServerOperationalRecoveryPlanSteps(plan)
	return plan
}

func defaultServerOperationalRecoveryPriority(kind string) string {
	switch strings.TrimSpace(kind) {
	case "worker_gap", "reserved_gap", "manual_auth", "stuck_workers":
		return "alta"
	case "quota_cooldown":
		return "baja"
	default:
		return "media"
	}
}

func buildServerOperationalRecoveryPlanSummary(plan *serverOperationalRecoveryPlan) string {
	if plan == nil {
		return ""
	}
	switch strings.TrimSpace(plan.Kind) {
	case "worker_gap":
		return fmt.Sprintf("Recuperar workers útiles para %d tarea(s) en progreso", max(plan.AffectedTasks, 1))
	case "reserved_gap":
		return fmt.Sprintf("Recuperar workers conectados para %d tarea(s) reservadas", max(plan.AffectedTasks, 1))
	case "manual_auth":
		return fmt.Sprintf("Completar autenticación manual de %d worker(s)", max(plan.AuthBlockedAgents, 1))
	case "quota_cooldown":
		return fmt.Sprintf("Esperar reset de cuota para %d worker(s) bloqueados", max(plan.QuotaBlockedAgents, 1))
	case "stuck_workers":
		return fmt.Sprintf("Desbloquear %d worker(s) atascados", max(plan.StuckAgents, 1))
	case "compaction_debt":
		return fmt.Sprintf("Compactar o reasignar %d tarea(s) con deuda operativa", max(plan.AffectedTasks, 1))
	case "rearm":
		return "Aplicar el rearm seguro del control plane"
	default:
		if detail := strings.TrimSpace(plan.Detail); detail != "" {
			return detail
		}
		return strings.TrimSpace(plan.Action)
	}
}

func buildServerOperationalRecoveryPlanSteps(plan *serverOperationalRecoveryPlan) []serverOperationalRecoveryStep {
	if plan == nil || strings.TrimSpace(plan.Action) == "" {
		return nil
	}
	steps := []serverOperationalRecoveryStep{{
		Name:           serverOperationalRecoveryPrimaryStepName(plan.Action),
		Action:         strings.TrimSpace(plan.Action),
		Target:         strings.TrimSpace(plan.Target),
		Detail:         buildServerOperationalRecoveryPrimaryStepDetail(plan),
		AutoExecutable: plan.AutoExecutable,
	}}
	recheckDetail := "Volver a consultar `orquesta.server.operational` y confirmar si desaparece la degradación."
	if strings.TrimSpace(plan.Action) == "wait_quota_reset" && strings.TrimSpace(plan.NextQuotaResetAt) != "" {
		recheckDetail = "Reevaluar `orquesta.server.operational` después del próximo reset visible de cuota."
	}
	steps = append(steps, serverOperationalRecoveryStep{
		Name:   "recheck_operational_state",
		Action: "server_operational_refresh",
		Target: "server:operational",
		Detail: recheckDetail,
	})
	return steps
}

func serverOperationalRecoveryPrimaryStepName(action string) string {
	switch strings.TrimSpace(action) {
	case "server_rearm":
		return "server_rearm"
	case "complete_manual_auth":
		return "complete_manual_auth"
	case "wait_quota_reset":
		return "wait_quota_reset"
	case "inspect_stuck_workers":
		return "inspect_stuck_workers"
	case "inspect_connected_idle_workers":
		return "inspect_connected_idle_workers"
	case "start_or_assign_workers":
		return "start_or_assign_workers"
	case "compact_or_reassign_active_tasks":
		return "compact_or_reassign_active_tasks"
	default:
		return "apply_recovery_action"
	}
}

func buildServerOperationalRecoveryPrimaryStepDetail(plan *serverOperationalRecoveryPlan) string {
	if plan == nil {
		return ""
	}
	if detail := strings.TrimSpace(plan.Detail); detail != "" {
		return detail
	}
	switch strings.TrimSpace(plan.Action) {
	case "server_rearm":
		return "Aplicar el siguiente rearm seguro disponible del supervisor."
	case "complete_manual_auth":
		return "Completar la autenticación manual pendiente antes de retomar trabajo."
	case "wait_quota_reset":
		return "Esperar al próximo reset visible de cuota para reactivar workers."
	case "inspect_stuck_workers":
		return "Inspeccionar workers atascados y desbloquear la ejecución."
	case "inspect_connected_idle_workers":
		return "Revisar workers conectados sin trabajo útil y relanzar runtime si hace falta."
	case "start_or_assign_workers":
		return "Asignar o arrancar workers útiles para retomar las tareas afectadas."
	case "compact_or_reassign_active_tasks":
		return "Compactar deuda operativa o reasignar tareas activas inconsistentes."
	default:
		return ""
	}
}

func buildServerOperationalRearmHint(info serverOperationalInfo) *serverOperationalRearmHint {
	if info.Operational {
		return nil
	}
	supervisor := resolveSupervisorName("")
	hint := &serverOperationalRearmHint{
		Needed:     true,
		Supervisor: supervisor,
		Reason:     strings.TrimSpace(info.Reason),
		Tool:       "orquesta.server.rearm",
		Endpoint:   "/api/server/rearm",
		Method:     "POST",
	}
	snapshot, ok := runStatusOptional(serverOperationalOptionalTimeout, func() (map[string]any, error) {
		return serverOperationalReviewSnapshotFn(supervisor)
	})
	if !ok || snapshot == nil {
		return hint
	}
	if nextSafeAction := supervisorNextSafeActionFromSnapshot(snapshot); nextSafeAction != nil {
		hint.Available = true
		hint.NextSafeAction = nextSafeAction
		if reason := strings.TrimSpace(nextSafeAction.Reason); reason != "" {
			hint.Reason = reason
		}
	}
	return hint
}

func serverOperationalApplyNextAction(supervisor string) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	return serverOperationalApplyNextActionFn(supervisor)
}

func visibleWorkerCount(totalVisible, supervisors int) int {
	if totalVisible <= 0 {
		return 0
	}
	if supervisors <= 0 {
		return totalVisible
	}
	out := totalVisible - supervisors
	if out < 0 {
		return 0
	}
	return out
}

func configuredSupervisorAgentSet() map[string]struct{} {
	names := statusSupervisorAgentNames()
	out := make(map[string]struct{}, len(names))
	for _, nombre := range names {
		nombre = nombreAgenteCanonico(nombre)
		if nombre == "" {
			continue
		}
		out[nombre] = struct{}{}
	}
	return out
}

func visibleNonSupervisorAgents(items []*db.Agente) []*db.Agente {
	if len(items) == 0 {
		return nil
	}
	supervisores := configuredSupervisorAgentSet()
	if len(supervisores) == 0 {
		return items
	}
	out := make([]*db.Agente, 0, len(items))
	for _, agente := range items {
		if agente == nil {
			continue
		}
		if _, ok := supervisores[nombreAgenteCanonico(agente.Nombre)]; ok {
			continue
		}
		out = append(out, agente)
	}
	return out
}

func buildServerOperationalInfoFastFromDB() (serverOperationalInfo, error) {
	if snapshot, ok := readStatusSnapshotFreshUsable(); ok {
		reconcileStatusSnapshotWithFreshPanel(&snapshot)
		return buildServerOperationalInfo(snapshot), nil
	}
	if snapshot, ok := readStatusSnapshotAny(); ok && !statusSnapshotNeedsImmediateRefresh(snapshot) {
		reconcileStatusSnapshotWithFreshPanel(&snapshot)
		return buildServerOperationalInfo(snapshot), nil
	}
	agentes, err := serverOperationalListAgentsFetcher()
	if err != nil {
		return serverOperationalInfo{}, err
	}
	cuentas, err := serverOperationalCountTasksFetcher()
	if err != nil {
		return serverOperationalInfo{}, err
	}
	agentesActivos := make([]*db.Agente, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil || !agenteCuentaComoConectado(agente) {
			continue
		}
		agentesActivos = append(agentesActivos, agente)
	}

	status := apiStatusResponse{
		Agentes:             agentes,
		AgentesActivos:      agentesActivos,
		AgentesQuotaBlocked: agentesNoActivosConCuotaConResumen(agentes, nil),
		TareasPorEstado:     cuentas,
		Generado:            time.Now().UTC().Format(time.RFC3339),
	}
	if cuentas != nil {
		status.TareasEnProgreso = make([]tareaLite, cuentas[string(db.TareaEnProgreso)])
		status.TareasReservadas = make([]tareaLite, cuentas[string(db.TareaAsignada)])
	}
	if rows, ok := readAgentPanelSnapshotFresh(); ok {
		status.Agentes = mergeServerOperationalAgentsWithPanelRows(status.Agentes, rows)
		sanitizeServerOperationalQuotaFromPanelRows(status.Agentes, rows)
		aplicarVisibilidadOperativaAgentes(status.Agentes, rows)
		activos, trabajando, saturados, atascados, authManual, quotaBlocked, _ := agentesVisiblesPorEstadoOperativoRows(status.Agentes, rows)
		activos, trabajando, saturados = normalizarAgentesVisiblesStatus(activos, trabajando, saturados)
		status.AgentesActivos = activos
		status.AgentesTrabajando = trabajando
		status.AgentesSaturados = saturados
		status.AgentesAtascados = atascados
		status.AgentesAuthManual = authManual
		if len(quotaBlocked) > 0 {
			status.AgentesQuotaBlocked = quotaBlocked
		}
		status.Autonomia = resumirAutonomiaRows(rows, statusNowFunc().UTC())
	}
	if summary, ok := runStatusOptional(serverOperationalOptionalTimeout, serverOperationalDispatchFetcher); ok {
		status.DeudaDispatch = summary.Deuda
		if summary.Handoffs > status.Autonomia.Handoffs {
			status.Autonomia.Handoffs = summary.Handoffs
		}
	}
	return buildServerOperationalInfo(status), nil
}

func formatServerOperationalSummary(info *serverOperationalInfo) string {
	if info == nil {
		return "desconocida"
	}
	parts := []string{
		fmt.Sprintf("%d conectados", info.ActiveAgents),
		fmt.Sprintf("%d registrados", info.RegisteredAgents),
	}
	if info.ConnectedWorkers > 0 {
		parts = append(parts, fmt.Sprintf("%d workers", info.ConnectedWorkers))
	}
	if info.WorkingAgents > 0 {
		parts = append(parts, fmt.Sprintf("%d trabajando", info.WorkingAgents))
	}
	if info.WorkingWorkers > 0 {
		parts = append(parts, fmt.Sprintf("%d workers_activos", info.WorkingWorkers))
	}
	if info.StuckAgents > 0 {
		parts = append(parts, fmt.Sprintf("%d atascados", info.StuckAgents))
	}
	if info.AuthAgents > 0 {
		parts = append(parts, fmt.Sprintf("%d requieren_auth", info.AuthAgents))
	}
	if info.QuotaAgents > 0 {
		parts = append(parts, fmt.Sprintf("%d bloqueados_cuota", info.QuotaAgents))
	}
	if info.SaturatedAgents > 0 {
		parts = append(parts, fmt.Sprintf("%d saturados", info.SaturatedAgents))
	}
	if info.ReservedTasks > 0 {
		parts = append(parts, fmt.Sprintf("%d reservadas", info.ReservedTasks))
	}
	if info.DispatchConfirmed > 0 || info.DispatchPending > 0 || info.DispatchNotified > 0 || info.DispatchFailed > 0 {
		parts = append(parts, fmt.Sprintf("dispatch p:%d n:%d f:%d c:%d", info.DispatchPending, info.DispatchNotified, info.DispatchFailed, info.DispatchConfirmed))
	}
	if info.AutonomySupervising > 0 || info.AutonomyContinuing > 0 || info.AutonomyPending > 0 || info.AutonomyConfirmed > 0 || info.AutonomyHandoffs > 0 {
		parts = append(parts, fmt.Sprintf("autonomia s:%d c:%d p:%d ok:%d h:%d", info.AutonomySupervising, info.AutonomyContinuing, info.AutonomyPending, info.AutonomyConfirmed, info.AutonomyHandoffs))
	}
	if info.TasksInProgress > 0 {
		parts = append(parts, fmt.Sprintf("%d en_progreso", info.TasksInProgress))
	}
	if info.CompactionDebtTasks > 0 {
		parts = append(parts, fmt.Sprintf("compactacion_pendiente:%d", info.CompactionDebtTasks))
	}
	if info.NextQuotaResetAt != "" {
		parts = append(parts, "quota_reset "+info.NextQuotaResetAt)
	}
	if info.Recovery != nil {
		recovery := strings.TrimSpace(info.Recovery.Kind)
		action := strings.TrimSpace(info.Recovery.SuggestedAction)
		switch {
		case recovery != "" && action != "":
			parts = append(parts, fmt.Sprintf("recovery %s->%s", recovery, action))
		case recovery != "":
			parts = append(parts, "recovery "+recovery)
		case action != "":
			parts = append(parts, "recovery "+action)
		}
		if info.Recovery.MissingWorkers > 0 {
			parts = append(parts, fmt.Sprintf("faltan_workers:%d", info.Recovery.MissingWorkers))
		}
	}
	return fmt.Sprintf("%s (%s)", strings.ToUpper(strings.TrimSpace(info.State)), strings.Join(parts, ", "))
}

func serverOperationalCompactionDebt(now time.Time) (int, int) {
	rows, ok := readAgentPanelSnapshotFresh()
	if !ok {
		return 0, 0
	}
	return serverOperationalCompactionDebtFromRows(rows, now)
}

func serverOperationalCompactionDebtFromRows(rows []agentesapp.Row, now time.Time) (int, int) {
	agents := 0
	tasks := 0
	for _, row := range rows {
		if !rowHasCompactionDebt(row, now) {
			continue
		}
		agents++
		tasks += row.OpenTasks - 1
	}
	return agents, tasks
}

func rowHasCompactionDebt(row agentesapp.Row, now time.Time) bool {
	if row.Agente == nil || row.CurrentTask == nil || row.OpenTasks <= 1 {
		return false
	}
	agente := nombreAgenteCanonico(row.Agente.Nombre)
	if agente == "" || !agenteVisibleEnStatusFleet(agente) {
		return false
	}
	if rowEsResiduoPausadoSinTrabajo(row, now) || row.SupervisorRoleActive(now) || row.EffectiveContinuityPending(now) {
		return false
	}
	switch strings.TrimSpace(row.EstadoOperativo) {
	case "retirado", "bloqueado", "bloqueado_por_runtime", "bloqueado_por_cuota", "caido":
		return false
	}
	if row.WorkerFresh(now) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(row.LastAutonomyState), "work_confirmed")
}

func mergeServerOperationalAgentsWithPanelRows(agentes []*db.Agente, rows []agentesapp.Row) []*db.Agente {
	if len(rows) == 0 {
		return agentes
	}
	byName := make(map[string]*db.Agente, len(agentes)+len(rows))
	order := make([]string, 0, len(agentes)+len(rows))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(agente.Nombre)
		if nombre == "" {
			continue
		}
		if _, ok := byName[nombre]; !ok {
			order = append(order, nombre)
		}
		byName[nombre] = preferAgenteStatusCanonico(byName[nombre], agente)
	}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(row.Agente.Nombre)
		if nombre == "" {
			continue
		}
		if _, ok := byName[nombre]; !ok {
			order = append(order, nombre)
		}
		byName[nombre] = preferAgenteStatusCanonico(byName[nombre], row.Agente)
	}
	if len(byName) == 0 {
		return nil
	}
	out := make([]*db.Agente, 0, len(order))
	for _, nombre := range order {
		if agente := byName[nombre]; agente != nil {
			out = append(out, agente)
		}
	}
	return out
}

func reconcileStatusSnapshotWithFreshPanel(snapshot *apiStatusResponse) {
	if snapshot == nil {
		return
	}
	rows, ok := readAgentPanelSnapshotFresh()
	if !ok {
		return
	}
	snapshot.Agentes = mergeServerOperationalAgentsWithPanelRows(snapshot.Agentes, rows)
	sanitizeServerOperationalQuotaFromPanelRows(snapshot.Agentes, rows)
	aplicarVisibilidadOperativaAgentes(snapshot.Agentes, rows)
	activos, trabajando, saturados, atascados, authManual, quotaBlocked, _ := agentesVisiblesPorEstadoOperativoRows(snapshot.Agentes, rows)
	activos, trabajando, saturados = normalizarAgentesVisiblesStatus(activos, trabajando, saturados)
	snapshot.AgentesActivos = activos
	snapshot.AgentesTrabajando = trabajando
	snapshot.AgentesSaturados = saturados
	snapshot.AgentesAtascados = atascados
	snapshot.AgentesAuthManual = authManual
	snapshot.AgentesQuotaBlocked = quotaBlocked
	snapshot.Autonomia = resumirAutonomiaRows(rows, statusNowFunc().UTC())
	snapshot.TareasActivas = reconciliarTareasActivasConPanelRows(snapshot.TareasActivas, rows)
	snapshot.TareasEnProgreso = filtrarOpenClawTareasPorEstado(snapshot.TareasActivas, db.TareaEnProgreso)
	snapshot.TareasReservadas = filtrarOpenClawTareasPorEstado(snapshot.TareasActivas, db.TareaAsignada)
	snapshot.TareasPorEstado = reconciliarConteoTareasActivasVisible(snapshot.TareasPorEstado, snapshot.TareasActivas)
	snapshot.WorkersConectados, snapshot.WorkersTrabajando, snapshot.SupervisoresActivos = statusVisibleWorkerCounters(
		snapshot.AgentesActivos,
		snapshot.AgentesTrabajando,
		snapshot.Autonomia,
	)
}

func sanitizeServerOperationalQuotaFromPanelRows(agentes []*db.Agente, rows []agentesapp.Row) {
	if len(agentes) == 0 || len(rows) == 0 {
		return
	}
	rowPorNombre := make(map[string]agentesapp.Row, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(row.Agente.Nombre)
		if nombre == "" {
			continue
		}
		rowPorNombre[nombre] = row
	}
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		row, ok := rowPorNombre[nombreAgenteCanonico(agente.Nombre)]
		if !ok {
			continue
		}
		switch strings.TrimSpace(row.EstadoOperativo) {
		case "arrancando", "trabajando", "saturado", "disponible", "atascado", "mailbox_atascada", "bloqueado_por_runtime":
			agente.EstadoCuota = "activo"
			agente.ReanimarAt = nil
			agente.MotivoPausa = ""
		}
	}
}

func nextQuotaResetVisible(agentes []*db.Agente) string {
	var earliest time.Time
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		resetAt := cooldownVisibleAgente(agente)
		if resetAt == nil || resetAt.IsZero() {
			continue
		}
		ts := resetAt.UTC()
		if earliest.IsZero() || ts.Before(earliest) {
			earliest = ts
		}
	}
	if earliest.IsZero() {
		return ""
	}
	return earliest.Format(time.RFC3339)
}
