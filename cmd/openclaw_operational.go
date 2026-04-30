package cmd

import (
	"strings"
	"time"
)

var openClawOperationalInfoTimeout = 150 * time.Millisecond

func buildOpenClawOperationalInfo() serverOperationalInfo {
	return buildOpenClawOperationalInfoWithStatus(apiStatusResponse{})
}

func buildOpenClawOperationalInfoWithStatus(status apiStatusResponse) serverOperationalInfo {
	fallback := normalizeServerOperationalInfo(buildServerOperationalInfo(status))
	if controlPlaneOperationalInfoFetcher == nil {
		if fallback.Generated == "" && !statusCarriesOperationalSignal(status) {
			return degradedServerOperationalInfo()
		}
		return fallback
	}
	info, err := runAPITimeboxed(openClawOperationalInfoTimeout, controlPlaneOperationalInfoFetcher, errStatusFetchTimeout)
	if err != nil {
		if fallback.Generated == "" && !statusCarriesOperationalSignal(status) {
			return degradedServerOperationalInfo()
		}
		return fallback
	}
	info = normalizeServerOperationalInfo(info)
	if openClawOperationalFallbackIsNewer(status, info.Generated) {
		return fallback
	}
	return info
}

func openClawOperationalFallbackIsNewer(status apiStatusResponse, generated string) bool {
	statusGenerated := strings.TrimSpace(status.Generado)
	infoGenerated := strings.TrimSpace(generated)
	if statusGenerated == "" || infoGenerated == "" {
		return false
	}
	statusAt, ok := parseOpenClawOperationalGenerated(statusGenerated)
	if !ok {
		return false
	}
	infoAt, ok := parseOpenClawOperationalGenerated(infoGenerated)
	if !ok {
		return false
	}
	return statusAt.After(infoAt)
}

func parseOpenClawOperationalGenerated(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func statusCarriesOperationalSignal(status apiStatusResponse) bool {
	if strings.TrimSpace(status.Generado) != "" {
		return true
	}
	if len(status.Agentes) > 0 || len(status.AgentesActivos) > 0 || len(status.AgentesTrabajando) > 0 ||
		len(status.AgentesSaturados) > 0 || len(status.AgentesAtascados) > 0 ||
		len(status.AgentesAuthManual) > 0 || len(status.AgentesQuotaBlocked) > 0 {
		return true
	}
	if len(status.TareasActivas) > 0 || len(status.TareasEnProgreso) > 0 || len(status.TareasReservadas) > 0 ||
		len(status.TareasPorEstado) > 0 || len(status.ConteoTareas) > 0 || len(status.ResumenTareas) > 0 {
		return true
	}
	if status.WorkersConectados > 0 || status.WorkersTrabajando > 0 || status.SupervisoresActivos > 0 {
		return true
	}
	if status.AutonomySurface != nil || status.CriticalProjectRisk != nil || len(status.AutonomyHighlights) > 0 {
		return true
	}
	return false
}

func buildOpenClawOperationalSummary(info serverOperationalInfo) string {
	info = normalizeServerOperationalInfo(info)
	return formatServerOperationalSummary(&info)
}
