package orquestaweb

func appChangeTextV0(locale string, key string) string {
	catalog := map[string]map[string]string{
		"es": {
			"title":                   "Solicitar cambio de app",
			"submit":                  "Enviar al director",
			"run_ref":                 "Run",
			"app_ref":                 "App",
			"change_ref":              "Cambio",
			"locale":                  "Idioma",
			"user_intent":             "Mensaje para el director",
			"target_area":             "Area",
			"criteria":                "Criterios de aceptacion",
			"write_set":               "Write-set permitido",
			"current_refs":            "Refs de estado",
			"constraints":             "Restricciones",
			"accepted":                "Cambio enviado al director",
			"initial":                 "Pendiente de solicitud",
			"error":                   "El cambio no se ha aceptado",
			"help_run":                "Identificador opaco del run existente.",
			"help_change":             "Identificador idempotente de esta solicitud.",
			"help_intent":             "Escribe aqui los cambios, instrucciones o peticiones sobre la app existente.",
			"help_write_set":          "Rutas relativas separadas por coma si quieres acotar el cambio.",
			"external_work":           "Trabajo externo",
			"external_project_ref":    "Proyecto externo",
			"external_interface_refs": "Contratos externos",
			"external_work_kind":      "Tipo de trabajo externo",
			"external_work_refs":      "Refs de trabajo externo",
			"help_external_work":      "Refs opacas para que Orquesta coordine una app externa sin conocer su nucleo interno.",
		},
		"en": {
			"title":                   "Request app change",
			"submit":                  "Send to director",
			"run_ref":                 "Run",
			"app_ref":                 "App",
			"change_ref":              "Change",
			"locale":                  "Locale",
			"user_intent":             "Message to the director",
			"target_area":             "Area",
			"criteria":                "Acceptance criteria",
			"write_set":               "Allowed write-set",
			"current_refs":            "State refs",
			"constraints":             "Constraints",
			"accepted":                "Change sent to director",
			"initial":                 "Waiting for request",
			"error":                   "Change was not accepted",
			"help_run":                "Opaque identifier of the existing run.",
			"help_change":             "Idempotent identifier for this request.",
			"help_intent":             "Write the changes, instructions, or requests for the existing app here.",
			"help_write_set":          "Comma-separated relative paths if you want to scope the change.",
			"external_work":           "External work",
			"external_project_ref":    "External project",
			"external_interface_refs": "External contracts",
			"external_work_kind":      "External work kind",
			"external_work_refs":      "External work refs",
			"help_external_work":      "Opaque refs so Orquesta can coordinate an external app without knowing its internals.",
		},
	}
	if catalog[locale] != nil && catalog[locale][key] != "" {
		return catalog[locale][key]
	}
	return catalog["es"][key]
}
