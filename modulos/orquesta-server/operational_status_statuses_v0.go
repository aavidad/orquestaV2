package orquestaserver

import (
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func startupEstadoV0(state StateV0) string {
	if state.StartupReady {
		return orquestaobservability.DiagnosticoEstadoOKV0
	}
	if strings.TrimSpace(state.StartupStatus) == "" {
		return orquestaobservability.DiagnosticoEstadoUnknownV0
	}
	return orquestaobservability.DiagnosticoEstadoBlockedV0
}

func supervisorEstadoV0(state StateV0) string {
	if strings.TrimSpace(state.LastSupervisorStatus) == "error" ||
		strings.TrimSpace(state.LastSupervisorError) != "" {
		return orquestaobservability.DiagnosticoEstadoDegradedV0
	}
	if strings.TrimSpace(state.LastSupervisorAt) == "" {
		return orquestaobservability.DiagnosticoEstadoUnknownV0
	}
	return orquestaobservability.DiagnosticoEstadoOKV0
}

func startupSeverityV0(state StateV0) string {
	if state.StartupReady {
		return "info"
	}
	return "warning"
}

func supervisorSeverityV0(state StateV0) string {
	if strings.TrimSpace(state.LastSupervisorStatus) == "error" ||
		strings.TrimSpace(state.LastSupervisorError) != "" {
		return "warning"
	}
	return "info"
}

func sanitizeServerEvidenceRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || !serverEvidenceRefPatternV0.MatchString(value) {
			continue
		}
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}
