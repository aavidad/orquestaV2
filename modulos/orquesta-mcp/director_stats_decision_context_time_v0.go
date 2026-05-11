package orquestamcp

import (
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func mcpDirectorDecisionWarningsV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
	observedAt string,
) []orquestaobservability.DirectorDecisionWarningV0 {
	var warnings []orquestaobservability.DirectorDecisionWarningV0
	if strings.TrimSpace(observedAt) == "" {
		warnings = append(warnings, directorDecisionWarningMCPV0("timing_partial"))
	}
	if stats.Progress.SourceStatus == orquestacionnucleoapp.DirectorProgressSourceNotConfiguredV0 {
		warnings = append(warnings, directorDecisionWarningMCPV0("progress_source_not_configured"))
	}
	if stats.Counts.AgentsInFlight > 0 &&
		stats.Counts.AgentsControlRegistered == 0 &&
		stats.Counts.AgentsControlMissing == 0 {
		warnings = append(warnings, directorDecisionWarningMCPV0("process_registry_not_loaded"))
	}
	return warnings
}

func directorDecisionWarningMCPV0(code string) orquestaobservability.DirectorDecisionWarningV0 {
	return orquestaobservability.DirectorDecisionWarningV0{
		Code:       code,
		SummaryKey: "director.decision_context.warning." + code,
	}
}

func mcpDirectorObservedAtV0(value string) string {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "Z") {
		return ""
	}
	if _, err := time.Parse(time.RFC3339Nano, trimmed); err != nil {
		return ""
	}
	return trimmed
}

func mcpDirectorDurationSecondsV0(openedAt string, closedAt string, observedAt string) int64 {
	start, ok := parseMCPDirectorTimeV0(openedAt)
	if !ok {
		return 0
	}
	end, ok := parseMCPDirectorTimeV0(firstNonEmptyMCPV0(closedAt, observedAt))
	if !ok || end.Before(start) {
		return 0
	}
	return int64(end.Sub(start).Seconds())
}

func parseMCPDirectorTimeV0(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "Z") {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	return parsed, err == nil
}

func mcpDirectorCauseForIndexV0(values []string, index int) string {
	values = compactStringsMCPV0(values)
	if index >= 0 && index < len(values) {
		return values[index]
	}
	if len(values) == 1 {
		return values[0]
	}
	return ""
}

func maxMCPDirectorV0(a int, b int) int {
	if b > a {
		return b
	}
	return a
}

func nonNegativeMCPDirectorV0(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
