package orquestafactory

import "strings"

const DefaultAccessibilityLevelV0 = "basica"

var supportedAccessibilityLevelsV0 = []string{
	"no_aplica",
	"basica",
	"normal",
	"wcag_aa",
}

var supportedAccessibilityOptionsV0 = []string{
	"no_aplica",
	"basica",
	"normal",
	"wcag_aa",
	"teclado",
	"lectores_pantalla",
	"contraste_alto",
	"movimiento_reducido",
	"subtitulos_transcripciones",
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

func NormalizeAccessibilityOptionV0(value string) string {
	normalized := NormalizeAccessibilityLevelV0(value)
	switch normalized {
	case "keyboard", "navegacion_por_teclado", "navegación_por_teclado":
		return "teclado"
	case "screen_reader", "screen_readers", "lectores", "lector_pantalla", "lectores_pantalla":
		return "lectores_pantalla"
	case "high_contrast", "contraste", "contraste_alto":
		return "contraste_alto"
	case "reduced_motion", "reduce_motion", "movimiento_reducido":
		return "movimiento_reducido"
	case "captions", "subtitles", "transcripciones", "subtitulos", "subtítulos", "subtitulos_transcripciones", "subtítulos_transcripciones":
		return "subtitulos_transcripciones"
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

func AccessibilityOptionSupportedV0(value string) bool {
	normalized := NormalizeAccessibilityOptionV0(value)
	for _, candidate := range supportedAccessibilityOptionsV0 {
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
		if normalized := NormalizeAccessibilityOptionV0(value); normalized != "" {
			values = append(values, normalized)
		}
	}
	return compactUniqueV0(values)
}
