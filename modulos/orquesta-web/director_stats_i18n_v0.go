package orquestaweb

func directorStatsTextsV0(locale string) WebDirectorStatsTextsV0 {
	switch normalizeDirectorStatsLocaleV0(locale) {
	case "en-US":
		return WebDirectorStatsTextsV0{
			Title:         "Director statistics",
			Counts:        "Counts",
			TaskProgress:  "Task progress",
			AgentProgress: "Agent progress",
			PublicErrors:  "Public errors",
		}
	default:
		return WebDirectorStatsTextsV0{
			Title:         "Estadisticas del director",
			Counts:        "Contadores",
			TaskProgress:  "Progreso de tareas",
			AgentProgress: "Progreso de agentes",
			PublicErrors:  "Errores publicos",
		}
	}
}

func normalizeDirectorStatsLocaleV0(locale string) string {
	switch trimDirectorStatsV0(locale) {
	case "en", "en-US":
		return "en-US"
	case "es", "es-ES":
		return "es-ES"
	default:
		return "es-ES"
	}
}

func firstDirectorStatsNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := trimDirectorStatsV0(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
