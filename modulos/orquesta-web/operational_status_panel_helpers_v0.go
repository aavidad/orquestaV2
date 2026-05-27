package orquestaweb

import (
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func webOperationalStatusEstadoV0(value string) string {
	switch trimOperationalStatusV0(value) {
	case orquestaobservability.DiagnosticoEstadoOKV0:
		return WebOperationalStatusEstadoOKV0
	case orquestaobservability.DiagnosticoEstadoDegradedV0:
		return WebOperationalStatusEstadoDegradedV0
	case orquestaobservability.DiagnosticoEstadoBlockedV0:
		return WebOperationalStatusEstadoBlockedV0
	case orquestaobservability.DiagnosticoEstadoFailedV0:
		return WebOperationalStatusEstadoFailedV0
	default:
		return WebOperationalStatusEstadoUnknownV0
	}
}

func webOperationalStatusVisualV0(value string) string {
	switch webOperationalStatusEstadoV0(value) {
	case WebOperationalStatusEstadoOKV0:
		return "success"
	case WebOperationalStatusEstadoDegradedV0:
		return "warning"
	case WebOperationalStatusEstadoBlockedV0:
		return "blocked"
	case WebOperationalStatusEstadoFailedV0:
		return "error"
	default:
		return "unknown"
	}
}

func webOperationalStatusProgressV0(value orquestaobservability.DiagnosticoProgresoV0) WebOperationalStatusProgressV0 {
	return WebOperationalStatusProgressV0{
		Completed: value.Completed,
		Total:     value.Total,
		Percent:   value.Percent,
		Summary:   trimOperationalStatusV0(value.Summary),
	}
}

func webOperationalStatusPhasesV0(phase string, estado string, progress WebOperationalStatusProgressV0) []WebOperationalStatusPhaseV0 {
	if phase == "" {
		return []WebOperationalStatusPhaseV0{}
	}
	return []WebOperationalStatusPhaseV0{{
		Key:      phase,
		Estado:   webOperationalStatusEstadoV0(estado),
		Actual:   true,
		Progress: progress,
	}}
}

func webOperationalStatusBlockersV0(values []orquestaobservability.DiagnosticoBloqueoV0) []WebOperationalStatusBlockerV0 {
	out := make([]WebOperationalStatusBlockerV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebOperationalStatusBlockerV0{
			Ref:          trimOperationalStatusV0(value.BlockerRef),
			Severity:     trimOperationalStatusV0(value.Severity),
			OwnerArea:    trimOperationalStatusV0(value.OwnerArea),
			Summary:      trimOperationalStatusV0(value.Summary),
			EvidenceRefs: compactOperationalStringsV0(value.EvidenceRefs),
		})
	}
	if out == nil {
		return []WebOperationalStatusBlockerV0{}
	}
	return out
}

func webOperationalStatusHealthV0(values []orquestaobservability.DiagnosticoSaludCheckV0) []WebOperationalStatusHealthV0 {
	out := make([]WebOperationalStatusHealthV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebOperationalStatusHealthV0{
			Area:         trimOperationalStatusV0(value.Area),
			Severity:     trimOperationalStatusV0(value.Severity),
			Estado:       webOperationalStatusEstadoV0(value.Estado),
			I18nKey:      trimOperationalStatusV0(value.I18nKey),
			EvidenceRefs: compactOperationalStringsV0(value.EvidenceRefs),
		})
	}
	if out == nil {
		return []WebOperationalStatusHealthV0{}
	}
	return out
}

func webOperationalStatusActivitiesV0(values []orquestaobservability.DiagnosticoActividadV0) []WebOperationalStatusActivityV0 {
	out := make([]WebOperationalStatusActivityV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebOperationalStatusActivityV0{
			Ref:         trimOperationalStatusV0(value.ActivityRef),
			OccurredAt:  trimOperationalStatusV0(value.OccurredAt),
			Area:        trimOperationalStatusV0(value.Area),
			Summary:     trimOperationalStatusV0(value.Summary),
			EventRef:    trimOperationalStatusV0(value.EventRef),
			ArtifactRef: trimOperationalStatusV0(value.ArtifactRef),
		})
	}
	if out == nil {
		return []WebOperationalStatusActivityV0{}
	}
	return out
}

func webOperationalStatusWarningsV0(values []orquestaobservability.DiagnosticoWarningV0) []WebOperationalStatusWarningV0 {
	out := make([]WebOperationalStatusWarningV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebOperationalStatusWarningV0{
			Code:    trimOperationalStatusV0(value.Code),
			Section: trimOperationalStatusV0(value.Section),
			Summary: trimOperationalStatusV0(value.Summary),
		})
	}
	if out == nil {
		return []WebOperationalStatusWarningV0{}
	}
	return out
}

func webOperationalStatusFreshnessV0(value orquestaobservability.DiagnosticoFreshnessV0) WebOperationalStatusFreshnessV0 {
	return WebOperationalStatusFreshnessV0{
		WatermarkRef:  trimOperationalStatusV0(value.WatermarkRef),
		MaxAgeSeconds: value.MaxAgeSeconds,
		Partial:       value.Partial,
		Stale:         value.Stale,
	}
}

func webOperationalStatusPrivacyOKV0(value orquestaobservability.DiagnosticoPrivacyV0) bool {
	return !value.ContainsSecret &&
		!value.ContainsTranscript &&
		!value.ContainsPrompt &&
		!value.ContainsCompletion &&
		!value.ContainsConnectionDetail
}

func webOperationalStatusRedactionLevelV0(value orquestaobservability.DiagnosticoPrivacyV0) string {
	return orquestaobservability.DiagnosticoPrivacyRedactionLevelV0(value)
}

func compactOperationalStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := trimOperationalStatusV0(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func trimOperationalStatusV0(value string) string {
	return strings.TrimSpace(value)
}
