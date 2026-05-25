package orquestaserver

import (
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func residentOperationalEstadoV0(state StateV0) string {
	if strings.TrimSpace(state.LastError) != "" || state.SupervisorErrorTicks > 0 {
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
	total := state.SupervisorTicks + state.IdleSelfImprovementTarget
	if total <= 0 {
		total = 1
	}
	completed := state.SupervisorExecutions + state.IdleSelfImprovementOK
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
	return map[string]float64{
		"supervisor_ticks":          float64(nonNegativeServerIntV0(state.SupervisorTicks)),
		"supervisor_error_ticks":    float64(nonNegativeServerIntV0(state.SupervisorErrorTicks)),
		"supervisor_executions":     float64(nonNegativeServerIntV0(state.SupervisorExecutions)),
		"supervisor_skips":          float64(nonNegativeServerIntV0(state.SupervisorSkips)),
		"queue_size":                float64(nonNegativeServerIntV0(state.LastSupervisorQueueSize)),
		"idle_improvement_runs":     float64(nonNegativeServerIntV0(state.IdleSelfImprovementRuns)),
		"idle_improvement_accepted": float64(nonNegativeServerIntV0(state.IdleSelfImprovementOK)),
	}
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
