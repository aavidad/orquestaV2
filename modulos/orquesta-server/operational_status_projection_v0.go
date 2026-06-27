package orquestaserver

import (
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func residentOperationalEstadoV0(state StateV0) string {
	if strings.TrimSpace(state.LastError) != "" ||
		strings.TrimSpace(state.LastSupervisorStatus) == "error" ||
		strings.TrimSpace(state.LastSupervisorError) != "" ||
		strings.TrimSpace(state.ResidentDirectorStatus) == "error" ||
		strings.TrimSpace(state.ResidentDirectorLastError) != "" ||
		strings.TrimSpace(state.ExternalBridgeStatus) == "blocked" ||
		strings.TrimSpace(state.ExternalBridgeStatus) == "degraded" ||
		strings.TrimSpace(state.ExternalBridgeStatus) == "timeout" ||
		strings.TrimSpace(state.StatePersistStatus) == "degraded" ||
		strings.TrimSpace(state.AuditStatus) == "degraded" ||
		residentOperationalDirectorDisabledWaitingOutboxV0(state) ||
		residentOperationalGoalFirstCapabilityMissingV0(state) {
		return orquestaobservability.DiagnosticoEstadoDegradedV0
	}
	switch strings.TrimSpace(state.Status) {
	case "running":
		return orquestaobservability.DiagnosticoEstadoOKV0
	case "startup_blocked":
		return orquestaobservability.DiagnosticoEstadoBlockedV0
	case "stopped":
		return orquestaobservability.DiagnosticoEstadoUnknownV0
	case "":
		return orquestaobservability.DiagnosticoEstadoUnknownV0
	default:
		return orquestaobservability.DiagnosticoEstadoDegradedV0
	}
}

func residentOperationalProgressV0(state StateV0) orquestaobservability.DiagnosticoProgresoV0 {
	total := state.SupervisorTicks + state.ResidentDirectorTicks + state.IdleSelfImprovementTarget
	if total <= 0 {
		total = 1
	}
	completed := state.SupervisorExecutions + state.ResidentDirectorExecutedActions + state.IdleSelfImprovementOK
	if completed > total {
		total = completed
	}
	percent := float64(completed) * 100 / float64(total)
	if percent > 100 {
		percent = 100
	}
	return orquestaobservability.DiagnosticoProgresoV0{
		Completed: completed,
		Total:     total,
		Percent:   &percent,
		Phase:     firstNonEmptyServerDiagnosticV0(state.StartupStatus, state.Status, "unknown"),
		Summary:   "estado residente compacto sin paths ni detalles privados",
	}
}

func residentOperationalCountersV0(state StateV0) map[string]float64 {
	counters := map[string]float64{
		"supervisor_ticks":          float64(nonNegativeServerIntV0(state.SupervisorTicks)),
		"supervisor_error_ticks":    float64(nonNegativeServerIntV0(state.SupervisorErrorTicks)),
		"supervisor_executions":     float64(nonNegativeServerIntV0(state.SupervisorExecutions)),
		"supervisor_skips":          float64(nonNegativeServerIntV0(state.SupervisorSkips)),
		"queue_size":                float64(nonNegativeServerIntV0(state.LastSupervisorQueueSize)),
		"idle_improvement_runs":     float64(nonNegativeServerIntV0(state.IdleSelfImprovementRuns)),
		"idle_improvement_accepted": float64(nonNegativeServerIntV0(state.IdleSelfImprovementOK)),
		"resident_director_ticks":   float64(nonNegativeServerIntV0(state.ResidentDirectorTicks)),
		"resident_director_errors":  float64(nonNegativeServerIntV0(state.ResidentDirectorErrorTicks)),
		"resident_director_actions": float64(nonNegativeServerIntV0(state.ResidentDirectorExecutedActions)),
		"resident_pending_without_dispatch": float64(
			residentPendingReasonCounterV0(state.IdleSelfImprovementReason),
		),
		"shutdown_async_work_active": float64(nonNegativeServerIntV0(state.ShutdownAsyncWorkActive)),
		"external_bridge_ticks":      float64(nonNegativeServerIntV0(state.ExternalBridgeTicks)),
		"external_bridge_errors":     float64(nonNegativeServerIntV0(state.ExternalBridgeErrorTicks)),
		"state_persist_failures":     float64(nonNegativeServerIntV0(state.StatePersistFailures)),
		"audit_failures":             float64(nonNegativeServerIntV0(state.AuditFailures)),
	}
	goal := residentOperationalGoalProjectionV0(state)
	if !residentOperationalGoalProjectionHasDurableStateV0(goal) {
		return counters
	}
	counters["goal_spec_present"] = residentOperationalBoolCounterV0(goal.SpecPresent)
	counters["goal_receipt_present"] = residentOperationalBoolCounterV0(goal.ReceiptPresent)
	counters["goal_result_present"] = residentOperationalBoolCounterV0(goal.ResultPresent)
	counters["goal_closure_present"] = residentOperationalBoolCounterV0(goal.ClosurePresent)
	counters["goal_closure_accepted"] = residentOperationalBoolCounterV0(goal.ClosureAccepted)
	return counters
}

func residentPendingReasonCounterV0(reason string) int {
	if residentPendingValueV0(reason) {
		return 1
	}
	return 0
}

func residentOperationalDirectorDisabledWaitingOutboxV0(state StateV0) bool {
	return !serverPublicResidentDirectorEnabledV0(state.EffectiveConfig) &&
		strings.TrimSpace(state.LastSupervisorStatus) == SupervisorPublicStatusWaitingOutboxV0
}

func residentOperationalAgeSecondsV0(lastHeartbeat string, now time.Time) int {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(lastHeartbeat))
	if err != nil || parsed.IsZero() || now.IsZero() {
		return 0
	}
	seconds := int(now.UTC().Sub(parsed.UTC()).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func nonNegativeServerIntV0(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func residentOperationalBoolCounterV0(value bool) float64 {
	return float64(boolToServerCounterV0(value))
}
