package orquestaserver

import (
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func (tracker *StatusTrackerV0) MarkSelfWatchdogV0(
	decision SelfWatchdogDecisionV0,
	now time.Time,
) StateV0 {
	decision = normalizeSelfWatchdogDecisionV0(decision)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.SelfWatchdogStatus = decision.Status
		state.SelfWatchdogReason = decision.ReasonCode
		state.SelfWatchdogObservedAt = decision.ObservedAt
		state.SelfWatchdogHighCPUSince = decision.HighCPUSince
		state.SelfWatchdogCPUPercent = decision.CPUPercent
		state.SelfWatchdogShutdownRequested = decision.ShouldRequestShutdown
		state.SelfWatchdogEvidenceRefs = compactServerStringsV0(decision.EvidenceRefs)
		state.SelfWatchdogOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "self_watchdog",
				ReasonCode:   decision.ReasonCode,
				Status:       decision.Status,
				Message:      "self_watchdog_" + decision.ReasonCode,
				EvidenceRefs: decision.EvidenceRefs,
				Counters: map[string]int{
					"cpu_percent": decision.CPUPercent,
				},
			},
		)
		if decision.ShouldRequestShutdown {
			state.Status = "unhealthy"
			state.LastError = "self_watchdog_" + decision.ReasonCode
			state.LastErrorOperationalMessage = copyServerOperationalMessageV0(state.SelfWatchdogOperationalMessage)
			appendRecentServerErrorV0(state, now, decision.ReasonCode, "self_watchdog", state.LastError, decision.EvidenceRefs)
		}
	})
}

func normalizeSelfWatchdogDecisionV0(decision SelfWatchdogDecisionV0) SelfWatchdogDecisionV0 {
	decision.Status = strings.TrimSpace(decision.Status)
	if decision.Status == "" {
		decision.Status = SelfWatchdogStatusOKV0
	}
	decision.ReasonCode = strings.TrimSpace(decision.ReasonCode)
	if decision.ReasonCode == "" {
		decision.ReasonCode = SelfWatchdogReasonCPUWithinLimitV0
	}
	decision.ObservedAt = strings.TrimSpace(decision.ObservedAt)
	decision.HighCPUSince = strings.TrimSpace(decision.HighCPUSince)
	decision.CPUPercent = nonNegativeServerIntV0(decision.CPUPercent)
	decision.EvidenceRefs = compactServerStringsV0(decision.EvidenceRefs)
	return decision
}

func selfWatchdogHealthCheckV0(state StateV0) orquestaobservability.DiagnosticoSaludCheckV0 {
	status := strings.TrimSpace(state.SelfWatchdogStatus)
	if status == "" {
		return orquestaobservability.DiagnosticoSaludCheckV0{}
	}
	return orquestaobservability.DiagnosticoSaludCheckV0{
		Area:         "system",
		Severity:     selfWatchdogSeverityV0(state),
		Estado:       selfWatchdogEstadoV0(state),
		I18nKey:      "server.health.self_watchdog",
		EvidenceRefs: compactServerStringsV0(state.SelfWatchdogEvidenceRefs),
	}
}

func selfWatchdogSeverityV0(state StateV0) string {
	if state.SelfWatchdogShutdownRequested {
		return "error"
	}
	switch strings.TrimSpace(state.SelfWatchdogStatus) {
	case SelfWatchdogStatusObservingHighCPUV0, SelfWatchdogStatusHighCPUWithCauseV0:
		return "warning"
	default:
		return "info"
	}
}

func selfWatchdogEstadoV0(state StateV0) string {
	if state.SelfWatchdogShutdownRequested {
		return orquestaobservability.DiagnosticoEstadoBlockedV0
	}
	switch strings.TrimSpace(state.SelfWatchdogStatus) {
	case SelfWatchdogStatusDisabledV0:
		return orquestaobservability.DiagnosticoEstadoOKV0
	case SelfWatchdogStatusObservingHighCPUV0, SelfWatchdogStatusHighCPUWithCauseV0:
		return orquestaobservability.DiagnosticoEstadoDegradedV0
	default:
		return orquestaobservability.DiagnosticoEstadoOKV0
	}
}
