package orquestafactory

import "strings"

const DefaultAccessibilityLevelV0 = "basica"

var supportedAccessibilityLevelsV0 = []string{
	"no_aplica",
	"basica",
	"normal",
	"wcag_aa",
}

func NormalizeAccessibilityLevelV0(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case "":
		return ""
	case "none", "no", "no_aplica", "no_aplicable":
		return "no_aplica"
	case "basic", "basica", "básica":
		return "basica"
	case "normal", "media", "standard":
		return "normal"
	case "aa", "wcag", "wcag_aa":
		return "wcag_aa"
	default:
		return normalized
	}
}

func AccessibilityLevelSupportedV0(value string) bool {
	normalized := NormalizeAccessibilityLevelV0(value)
	for _, candidate := range supportedAccessibilityLevelsV0 {
		if normalized == candidate {
			return true
		}
	}
	return false
}

func accessibilityLevelOrDefaultV0(value string) string {
	if normalized := NormalizeAccessibilityLevelV0(value); normalized != "" {
		return normalized
	}
	return DefaultAccessibilityLevelV0
}

func normalizeAccessibilityOptionsV0(calidad CalidadRequestV0) []string {
	values := make([]string, 0, len(calidad.AccesibilidadOpciones)+1)
	selected := accessibilityLevelOrDefaultV0(calidad.Accesibilidad)
	values = append(values, selected)
	for _, value := range calidad.AccesibilidadOpciones {
		if normalized := NormalizeAccessibilityLevelV0(value); normalized != "" {
			values = append(values, normalized)
		}
	}
	return compactUniqueV0(values)
}
