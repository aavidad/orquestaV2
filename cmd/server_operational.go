package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/db"
)

type serverOperationalInfo struct {
	State               string `json:"state"`
	Operational         bool   `json:"operational"`
	Reason              string `json:"reason,omitempty"`
	NextQuotaResetAt    string `json:"nextQuotaResetAt,omitempty"`
	Generated           string `json:"generated,omitempty"`
	RegisteredAgents    int    `json:"registeredAgents"`
	ActiveAgents        int    `json:"activeAgents"`
	WorkingAgents       int    `json:"workingAgents"`
	SaturatedAgents     int    `json:"saturatedAgents"`
	StuckAgents         int    `json:"stuckAgents"`
	AuthAgents          int    `json:"authAgents"`
	QuotaAgents         int    `json:"quotaAgents"`
	PausedAgents        int    `json:"pausedAgents"`
	TasksInProgress     int    `json:"tasksInProgress"`
	ReservedTasks       int    `json:"reservedTasks"`
	BlockedTasks        int    `json:"blockedTasks"`
	DispatchPending     int    `json:"dispatchPending"`
	DispatchNotified    int    `json:"dispatchNotified"`
	DispatchFailed      int    `json:"dispatchFailed"`
	DispatchConfirmed   int    `json:"dispatchConfirmed"`
	AutonomySupervising int    `json:"autonomySupervising"`
	AutonomyContinuing  int    `json:"autonomyContinuing"`
	AutonomyPending     int    `json:"autonomyPending"`
	AutonomyConfirmed   int    `json:"autonomyConfirmed"`
	AutonomyHandoffs    int    `json:"autonomyHandoffs"`
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
	tasksInProgress := len(status.TareasEnProgreso)
	if tasksInProgress == 0 {
		tasksInProgress = len(filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaEnProgreso))
	}
	reservedTasks := len(status.TareasReservadas)
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
	saturatedAgents := len(status.AgentesSaturados)
	stuckAgents := len(status.AgentesAtascados)
	authAgents := len(status.AgentesAuthManual)

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
	case activeAgents == 0 && quotaAgents > 0:
		state = "idle"
		reason = "workers_quota_blocked"
	case tasksInProgress > 0 && workingAgents == 0 && status.Autonomia.WorkConfirmed == 0:
		state = "degraded"
		reason = "tasks_without_workers"
		operational = false
	case activeAgents == 0 && reservedTasks > 0:
		state = "degraded"
		reason = "reserved_without_connected_workers"
		operational = false
	case activeAgents == 0 && tasksInProgress == 0 && reservedTasks == 0:
		state = "idle"
		reason = "no_active_workers"
	}

	return serverOperationalInfo{
		State:               state,
		Operational:         operational,
		Reason:              reason,
		NextQuotaResetAt:    nextQuotaResetAt,
		Generated:           status.Generado,
		RegisteredAgents:    registeredAgentCountFromStatus(status),
		ActiveAgents:        activeAgents,
		WorkingAgents:       workingAgents,
		SaturatedAgents:     saturatedAgents,
		StuckAgents:         stuckAgents,
		AuthAgents:          authAgents,
		QuotaAgents:         quotaAgents,
		PausedAgents:        pausedAgents,
		TasksInProgress:     tasksInProgress,
		ReservedTasks:       reservedTasks,
		BlockedTasks:        blockedTasks,
		DispatchPending:     status.DeudaDispatch.Pendientes,
		DispatchNotified:    status.DeudaDispatch.Notificadas,
		DispatchFailed:      status.DeudaDispatch.Fallidas,
		DispatchConfirmed:   status.DeudaDispatch.WorkConfirmed,
		AutonomySupervising: status.Autonomia.Supervisando,
		AutonomyContinuing:  status.Autonomia.Continuando,
		AutonomyPending:     status.Autonomia.ContinuidadPendiente,
		AutonomyConfirmed:   status.Autonomia.WorkConfirmed,
		AutonomyHandoffs:    status.Autonomia.Handoffs,
	}
}

func formatServerOperationalSummary(info *serverOperationalInfo) string {
	if info == nil {
		return "desconocida"
	}
	parts := []string{
		fmt.Sprintf("%d conectados", info.ActiveAgents),
		fmt.Sprintf("%d registrados", info.RegisteredAgents),
	}
	if info.WorkingAgents > 0 {
		parts = append(parts, fmt.Sprintf("%d trabajando", info.WorkingAgents))
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
	if info.NextQuotaResetAt != "" {
		parts = append(parts, "quota_reset "+info.NextQuotaResetAt)
	}
	return fmt.Sprintf("%s (%s)", strings.ToUpper(strings.TrimSpace(info.State)), strings.Join(parts, ", "))
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
