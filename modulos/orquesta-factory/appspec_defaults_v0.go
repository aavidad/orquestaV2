package orquestafactory

import "strings"

func (n appSpecNormalizerV0) defaultsApplied() []DefaultAppliedV0 {
	defaults := []DefaultAppliedV0{
		{Campo: "architecture.patron", Valor: "hexagonal", Motivo: "Default obligatorio de OrquestaV2."},
	}
	if n.req.I18N.Enabled == nil {
		defaults = append(defaults, DefaultAppliedV0{Campo: "i18n.enabled", Valor: true, Motivo: "i18n se activa por defecto."})
	}
	if strings.TrimSpace(n.req.I18N.DefaultLocale) == "" {
		defaults = append(defaults, DefaultAppliedV0{Campo: "i18n.default_locale", Valor: strings.TrimSpace(n.req.Locale), Motivo: "Se usa el locale principal de la peticion."})
	}
	if len(n.req.Documentacion.Locales) == 0 {
		defaults = append(defaults, DefaultAppliedV0{Campo: "docs.locales", Valor: n.i18n().Locales, Motivo: "La documentacion hereda los locales i18n."})
	}
	if strings.TrimSpace(n.req.Deploy.Target) == "" {
		defaults = append(defaults, DefaultAppliedV0{Campo: "deploy.target", Valor: "sin_preferencia", Motivo: "El deploy se decide por conector en una fase posterior."})
	}
	if strings.TrimSpace(n.req.RequestKind) == "" {
		defaults = append(defaults, DefaultAppliedV0{Campo: "request_kind", Valor: DefaultRequestKindV0, Motivo: "Una solicitud de nueva app ejecuta el flujo completo por defecto."})
	}
	if strings.TrimSpace(n.req.ExecutionMode) == "" {
		defaults = append(defaults, DefaultAppliedV0{Campo: "execution_mode", Valor: DefaultExecutionModeV0, Motivo: "El modo normal exige minimos completos salvo debug explicito."})
	}
	return defaults
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func boolDefaultV0(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func docsLocalesV0(req AppSpecRequestV0, fallback []string) []string {
	if locales := compactUniqueV0(req.Documentacion.Locales); len(locales) > 0 {
		return locales
	}
	return fallback
}

func defaultPlatformsV0(tipoApp string) []string {
	switch strings.TrimSpace(tipoApp) {
	case "web":
		return []string{"web"}
	case "api":
		return []string{"server"}
	case "cli":
		return []string{"cli"}
	case "desktop":
		return []string{"desktop"}
	case "mobile":
		return []string{"mobile"}
	case "automation":
		return []string{"automation"}
	case "data":
		return []string{"data"}
	case "plugin":
		return []string{"plugin"}
	default:
		return []string{"mixed"}
	}
}
