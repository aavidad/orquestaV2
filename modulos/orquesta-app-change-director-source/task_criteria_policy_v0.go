package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

const (
	maxAppChangeTaskCriteriaV0       = 10
	maxAppChangeTaskCriterionCharsV0 = 260
)

func appChangeTaskCriteriaV0(request orquestaappchange.AppChangeRequestV0) []string {
	criteria := []string{"Mantener arquitectura hexagonal e i18n si aplica."}
	criteria = append(criteria, appChangeExternalWorkCriteriaV0(request)...)
	criteria = append(criteria, request.AcceptanceCriteria...)
	return compactAppChangeTaskCriteriaV0(criteria)
}

func compactAppChangeTaskCriteriaV0(criteria []string) []string {
	criteria = compactAppChangeSourceRefsV0(criteria)
	criteria = sanitizeAppChangeTaskCriteriaV0(criteria)
	if len(criteria) <= maxAppChangeTaskCriteriaV0 {
		return criteria
	}
	return append([]string(nil), criteria[:maxAppChangeTaskCriteriaV0]...)
}

func sanitizeAppChangeTaskCriteriaV0(criteria []string) []string {
	out := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		sanitized := strings.TrimSpace(compactAppChangeTaskCriterionForDirectorV0(
			sanitizeAppChangeTaskCriterionV0(criterion),
		))
		if sanitized != "" {
			out = append(out, sanitized)
		}
	}
	return out
}

func sanitizeAppChangeTaskCriterionV0(value string) string {
	return strings.TrimSpace(value)
}

func compactAppChangeTaskCriterionForDirectorV0(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) <= maxAppChangeTaskCriterionCharsV0 {
		return value
	}
	suffix := " (detalle completo en paquete externo)"
	limit := maxAppChangeTaskCriterionCharsV0 - len(suffix)
	if limit < 1 {
		return strings.TrimSpace(suffix)
	}
	prefix := strings.TrimSpace(value[:limit])
	if cut := strings.LastIndex(prefix, " "); cut > 80 {
		prefix = strings.TrimSpace(prefix[:cut])
	}
	return prefix + suffix
}
