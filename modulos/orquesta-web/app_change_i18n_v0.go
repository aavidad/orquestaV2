package orquestaweb

func appChangeTextV0(locale string, key string) string {
	catalog := map[string]map[string]string{
		"es": {
			"title":          "Solicitar cambio de app",
			"submit":         "Enviar al director",
			"run_ref":        "Run",
			"app_ref":        "App",
			"change_ref":     "Cambio",
			"locale":         "Idioma",
			"user_intent":    "Mensaje para el director",
			"target_area":    "Area",
			"criteria":       "Criterios de aceptacion",
			"write_set":      "Write-set permitido",
			"current_refs":   "Refs de estado",
			"constraints":    "Restricciones",
			"accepted":       "Cambio enviado al director",
			"initial":        "Pendiente de solicitud",
			"error":          "El cambio no se ha aceptado",
			"help_run":       "Identificador opaco del run existente.",
			"help_change":    "Identificador idempotente de esta solicitud.",
			"help_intent":    "Escribe aqui los cambios, instrucciones o peticiones sobre la app existente.",
			"help_write_set": "Rutas relativas separadas por coma si quieres acotar el cambio.",
		},
		"en": {
			"title":          "Request app change",
			"submit":         "Send to director",
			"run_ref":        "Run",
			"app_ref":        "App",
			"change_ref":     "Change",
			"locale":         "Locale",
			"user_intent":    "Message to the director",
			"target_area":    "Area",
			"criteria":       "Acceptance criteria",
			"write_set":      "Allowed write-set",
			"current_refs":   "State refs",
			"constraints":    "Constraints",
			"accepted":       "Change sent to director",
			"initial":        "Waiting for request",
			"error":          "Change was not accepted",
			"help_run":       "Opaque identifier of the existing run.",
			"help_change":    "Idempotent identifier for this request.",
			"help_intent":    "Write the changes, instructions, or requests for the existing app here.",
			"help_write_set": "Comma-separated relative paths if you want to scope the change.",
		},
	}
	if catalog[locale] != nil && catalog[locale][key] != "" {
		return catalog[locale][key]
	}
	return catalog["es"][key]
}
