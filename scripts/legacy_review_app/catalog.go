// Este fichero es el único catálogo de textos visibles de la herramienta.
// No escribe estado; acredita español predeterminado y de reserva en sus pruebas.
package main

type catalog struct {
	values map[string]string
}

func newCatalog(_ string) catalog {
	return catalog{values: map[string]string{
		"app_title":                  "Revisión del inventario histórico",
		"skip_content":               "Saltar al contenido principal",
		"nav_inventory":              "Inventario",
		"list_title":                 "Elementos pendientes de revisión",
		"filter_title":               "Filtros",
		"filter_search":              "Buscar",
		"filter_family":              "Familia",
		"filter_disposition":         "Disposición propuesta",
		"filter_all":                 "Todas",
		"filter_apply":               "Aplicar filtros",
		"table_item":                 "Elemento",
		"table_family":               "Familia",
		"table_revision":             "Revisión",
		"table_disposition":          "Propuesta vigente",
		"table_action":               "Acción",
		"action_review":              "Revisar",
		"empty_list":                 "No hay elementos que coincidan con los filtros.",
		"detail_title":               "Detalle del elemento",
		"field_identifier":           "Identificador",
		"field_revision":             "Revisión inmutable",
		"field_family":               "Familia",
		"field_summary":              "Resumen",
		"sources_title":              "Fuentes",
		"attempts_title":             "Intentos históricos",
		"uncertainties_title":        "Incertidumbres",
		"raw_title":                  "Registro original",
		"empty_sources":              "No hay fuentes estructuradas declaradas.",
		"empty_attempts":             "No hay intentos históricos estructurados.",
		"empty_uncertainties":        "No hay incertidumbres estructuradas.",
		"proposal_history_title":     "Historial de propuestas",
		"empty_proposals":            "Todavía no hay propuestas para este elemento.",
		"proposal_form_title":        "Nueva propuesta de clasificación",
		"proposal_boundary":          "Esta herramienta solo conserva una propuesta. No modifica el inventario, no cambia el catálogo y no crea tareas.",
		"field_disposition":          "Disposición",
		"field_reason":               "Razón fundada",
		"field_solution":             "Solución fundada o tratamiento propuesto",
		"field_confidence":           "Confianza, de 0 a 100",
		"field_rule_notes":           "Explicación del cumplimiento o de sus dudas",
		"rules_title":                "Cumplimiento de reglas del proyecto",
		"field_expected_revision":    "Revisión esperada de propuestas",
		"field_actor":                "Actor",
		"field_project":              "Proyecto",
		"field_created":              "Fecha conservada",
		"field_rules_result":         "Resultado por regla",
		"action_submit":              "Conservar propuesta",
		"saved_notice":               "La propuesta se conservó sin modificar ninguna autoridad del producto.",
		"error_title":                "No se pudo completar la operación",
		"error_generic":              "La operación no pudo completarse de forma segura.",
		"error_not_found":            "El elemento solicitado no existe.",
		"error_unauthorized":         "Se necesita la autorización local explícita.",
		"error_host":                 "El destino HTTP no coincide con el enlace local autorizado.",
		"error_origin":               "El origen de la petición no está autorizado.",
		"error_csrf":                 "La protección de la petición no es válida.",
		"error_content_type":         "El formato de la petición no está permitido.",
		"error_invalid_form":         "La propuesta está incompleta o contiene valores no válidos.",
		"error_stale_inventory":      "El inventario cambió; reinicia la herramienta antes de proponer.",
		"error_revision_conflict":    "Otra propuesta cambió la revisión esperada. Recarga el detalle.",
		"error_idempotency_conflict": "La misma clave de repetición corresponde a otro contenido.",
		"error_traversal":            "La ruta solicitada no está permitida.",
		"back_detail":                "Volver al detalle",
		"back_list":                  "Volver al inventario",
		"value_yes":                  "Cumple",
		"value_no":                   "No cumple o existe duda",
		"auth_realm":                 "revisión local del inventario",
		"cli_started":                "Aplicación disponible en",
		"cli_login":                  "Usuario de autorización HTTP",
		"cli_start_failed":           "No se pudo iniciar la aplicación local.",
		"cli_stopped":                "La aplicación local se detuvo.",
		"cli_usage":                  "Uso: legacy_review_app --inventory RUTA --proposals RUTA --authorization-file RUTA --actor-ref REF --project-ref REF [--listen 127.0.0.1:8787]",
		"cli_flag_inventory":         "JSONL de inventario, abierto en solo lectura",
		"cli_flag_proposals":         "JSONL donde conservar propuestas",
		"cli_flag_authorization":     "Fichero privado con la autorización local",
		"cli_flag_actor":             "Referencia opaca del actor",
		"cli_flag_project":           "Referencia opaca del proyecto",
		"cli_flag_listen":            "Dirección numérica de escucha local",
		"missing_text":               "Texto no disponible",

		"disposition_none":                "Sin propuesta",
		"disposition_admitido":            "Admitido",
		"disposition_en_estudio":          "En estudio, sin tarea",
		"disposition_rechazado":           "Rechazado con razón",
		"disposition_evidencia_historica": "Evidencia histórica",
		"disposition_duplicado":           "Duplicado trazado",
		"rule_hexagonal":                  "Arquitectura hexagonal de dentro hacia fuera",
		"rule_i18n":                       "Internacionalización completa con español de reserva",
		"rule_single_lifecycle":           "Goal como único ciclo de vida",
		"rule_single_writer_scheduler":    "Un único escritor y un único planificador",
		"rule_config_credentials":         "Configuración única y secretos mediante referencias",
		"rule_identity_permissions":       "Identidad, proyecto y permisos explícitos",
		"rule_governed_effects":           "Efectos externos gobernados y justificables",
		"rule_no_legacy_dependency":       "Ninguna dependencia ni escritura alternativa hacia el legado",
	}}
}

func (c catalog) text(key string) string {
	if value := c.values[key]; value != "" {
		return value
	}
	return c.values["missing_text"]
}

func (c catalog) disposition(value disposition) string {
	return c.text("disposition_" + string(value))
}

func (c catalog) rule(value string) string {
	return c.text("rule_" + value)
}
