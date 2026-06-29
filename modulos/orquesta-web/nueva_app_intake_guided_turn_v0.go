package orquestaweb

import "strings"

const WebNuevaAppIntakeGuidedTurnSchemaV0 = "web_nueva_app_intake_guided_turn.v0"

type WebNuevaAppIntakeGuidedTurnV0 struct {
	SchemaVersion string                            `json:"schema_version"`
	Need          string                            `json:"need"`
	Decisions     []WebNuevaAppIntakeDecisionV0     `json:"decisions"`
	Followups     []WebNuevaAppIntakeGuidedActionV0 `json:"followups"`
	Messages      []string                          `json:"messages"`
}

type WebNuevaAppIntakeGuidedActionV0 struct {
	ID       string                        `json:"id"`
	LabelKey string                        `json:"label_key"`
	Decides  []WebNuevaAppIntakeDecisionV0 `json:"decides"`
}

func NewWebNuevaAppIntakeGuidedTurnV0(need string) WebNuevaAppIntakeGuidedTurnV0 {
	trimmed := trimV0(need)
	turn := WebNuevaAppIntakeGuidedTurnV0{
		SchemaVersion: WebNuevaAppIntakeGuidedTurnSchemaV0,
		Need:          trimmed,
		Decisions:     guidedInitialDecisionsV0(trimmed),
		Followups:     guidedFollowupActionsV0(),
		Messages:      []string{"nueva_app.wizard.guided_msg_analyzed"},
	}
	if turn.Decisions == nil {
		turn.Decisions = []WebNuevaAppIntakeDecisionV0{}
	}
	return turn
}

func ApplyWebNuevaAppIntakeGuidedTurnV0(
	session WebNuevaAppIntakeSessionV0,
	turn WebNuevaAppIntakeGuidedTurnV0,
) WebNuevaAppIntakeSessionV0 {
	for _, decision := range turn.Decisions {
		session = session.ApplyDecisionV0(decision)
	}
	return session
}

func ApplyWebNuevaAppIntakeGuidedActionV0(
	session WebNuevaAppIntakeSessionV0,
	actionID string,
) WebNuevaAppIntakeSessionV0 {
	for _, action := range guidedFollowupActionsV0() {
		if action.ID != trimV0(actionID) {
			continue
		}
		for _, decision := range action.Decides {
			session = session.ApplyDecisionV0(decision)
		}
		return session
	}
	return session
}

func ApplyWebNuevaAppIntakeGuidedAnswerV0(
	session WebNuevaAppIntakeSessionV0,
	field string,
	answer string,
) WebNuevaAppIntakeSessionV0 {
	answer = trimV0(answer)
	if answer == "" {
		return session
	}
	field = trimV0(field)
	if field == "" && len(session.PendingQuestions) > 0 {
		field = session.PendingQuestions[0]
	}
	if field == "" {
		return session
	}
	return session.ApplyDecisionV0(guidedAnswerDecisionV0(field, answer))
}

func guidedAnswerDecisionV0(field string, answer string) WebNuevaAppIntakeDecisionV0 {
	field = trimV0(field)
	answer = trimV0(answer)
	switch field {
	case "tipo_app":
		return WebNuevaAppIntakeDecisionV0{
			Field: field,
			Value: normalizeGuidedTipoAppAnswerV0(answer),
		}
	case "usuarios_objetivo", "plataformas", "restricciones":
		return WebNuevaAppIntakeDecisionV0{
			Field:  field,
			Value:  answer,
			Values: splitGuidedAnswerValuesV0(field, answer),
		}
	default:
		return WebNuevaAppIntakeDecisionV0{
			Field:  field,
			Value:  answer,
			Values: splitGuidedAnswerValuesV0(field, answer),
		}
	}
}

func normalizeGuidedTipoAppAnswerV0(answer string) string {
	normalized := normalizeGuidedNeedV0(answer)
	switch {
	case guidedContainsAnyV0(normalized, "web", "portal", "panel", "frontend", "aplicacion web", "aplicación web"):
		return "web"
	case guidedContainsAnyV0(normalized, "api", "rest", "backend", "servicio"):
		return "api"
	case guidedContainsAnyV0(normalized, "movil", "mobile", "android", "ios", "iphone", "móvil"):
		return "mobile"
	case guidedContainsAnyV0(normalized, "cli", "terminal", "consola"):
		return "cli"
	case guidedContainsAnyV0(normalized, "libreria", "library", "sdk"):
		return "library"
	default:
		return answer
	}
}

func splitGuidedAnswerValuesV0(field string, answer string) []string {
	switch field {
	case "usuarios_objetivo", "plataformas", "restricciones":
		return normalizeGuidedAnswerValuesV0(field, splitFreeListTextV0(answer))
	default:
		return nil
	}
}

func normalizeGuidedAnswerValuesV0(field string, values []string) []string {
	if field != "plataformas" {
		return values
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeGuidedNeedV0(value)
		switch {
		case guidedContainsAnyV0(normalized, "web", "portal", "panel", "frontend"):
			out = append(out, "web")
		case guidedContainsAnyV0(normalized, "api", "rest", "backend", "servicio"):
			out = append(out, "api")
		case guidedContainsAnyV0(normalized, "movil", "mobile", "android", "ios", "iphone"):
			out = append(out, "mobile")
		default:
			out = append(out, value)
		}
	}
	return compactStringsV0(out)
}

func splitFreeListTextV0(raw string) []string {
	normalized := strings.NewReplacer(
		"\n", ",",
		";", ",",
		" y ", ",",
		" e ", ",",
		" and ", ",",
		" & ", ",",
		" + ", ",",
	).Replace(raw)
	return splitCSVTextV0(normalized)
}

func splitCSVTextV0(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := trimV0(part)
		if value != "" {
			out = append(out, value)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func guidedInitialDecisionsV0(need string) []WebNuevaAppIntakeDecisionV0 {
	if trimV0(need) == "" {
		return []WebNuevaAppIntakeDecisionV0{}
	}
	normalized := normalizeGuidedNeedV0(need)
	decisions := []WebNuevaAppIntakeDecisionV0{
		{Field: "objetivo", Value: need},
		{Field: "descripcion", Value: need},
		{Field: "nombre", Value: guidedTitleFromNeedV0(need)},
	}
	var platforms []string
	appType := ""
	if guidedContainsAnyV0(normalized, "movil", "mobile", "android", "ios", "iphone", "apple") {
		appType = "mobile"
		platforms = append(platforms, "mobile")
	}
	if guidedContainsAnyV0(normalized, "api", "backend", "servicio") {
		if appType == "" {
			appType = "api"
		}
		platforms = append(platforms, "api")
	}
	if guidedContainsAnyV0(normalized, "web", "panel", "portal", "gestion") {
		if appType == "" {
			appType = "web"
		}
		platforms = append(platforms, "web")
	}
	if appType != "" {
		decisions = append(decisions, WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: appType})
	}
	if len(platforms) > 0 {
		decisions = append(decisions, WebNuevaAppIntakeDecisionV0{Field: "plataformas", Values: compactStringsV0(platforms)})
	}
	nextIntegrationIndex := 0
	if guidedContainsAnyV0(normalized, "piso", "pisos", "alquiler", "alquileres", "vivienda", "rent") {
		decisions = append(decisions, guidedRentalDataDecisionsV0()...)
	}
	if guidedContainsAnyV0(normalized, "mapa", "mapas", "map", "maps", "cerca", "cercano", "cercanos", "ubicacion", "geo", "openstreet") {
		decisions = append(decisions, guidedMapDecisionsV0(strings.Contains(normalized, "openstreet"))...)
		nextIntegrationIndex = 1
	}
	if guidedContainsAnyV0(normalized, "calendario", "calendar", "agenda", "cita", "citas") {
		decisions = append(decisions, guidedCapabilityIntegrationDecisionsV0(
			nextIntegrationIndex,
			"calendar",
			"calendario",
			"Sincronizar eventos, citas o agenda con adaptador autorizado.",
			"oauth gestionado",
		)...)
		nextIntegrationIndex++
	}
	if guidedContainsAnyV0(normalized, "pago", "pagos", "payment", "payments", "cobro", "stripe") {
		decisions = append(decisions, guidedCapabilityIntegrationDecisionsV0(
			nextIntegrationIndex,
			"payments",
			"pagos",
			"Procesar pagos mediante un adaptador externo autorizado.",
			"proveedor gestionado",
		)...)
		nextIntegrationIndex++
	}
	if guidedContainsAnyV0(normalized, "login", "auth", "autenticacion", "permisos", "roles", "sso") {
		decisions = append(decisions, guidedCapabilityIntegrationDecisionsV0(
			nextIntegrationIndex,
			"auth",
			"autenticacion",
			"Gestionar acceso, permisos, roles o SSO sin acoplar secretos al nucleo.",
			"sso u oauth gestionado",
		)...)
		nextIntegrationIndex++
	}
	if guidedContainsAnyV0(normalized, "notificacion", "notificaciones", "notification", "notifications", "alerta", "alertas") {
		decisions = append(decisions, guidedCapabilityIntegrationDecisionsV0(
			nextIntegrationIndex,
			"notifications",
			"notificaciones",
			"Enviar avisos o alertas por canales autorizados.",
			"proveedor gestionado",
		)...)
	}
	return decisions
}

func guidedFollowupActionsV0() []WebNuevaAppIntakeGuidedActionV0 {
	return []WebNuevaAppIntakeGuidedActionV0{
		{
			ID:       "mobile_both",
			LabelKey: "nueva_app.wizard.guided_mobile_both",
			Decides: []WebNuevaAppIntakeDecisionV0{
				{Field: "tipo_app", Value: "mobile"},
				{Field: "plataformas", Values: []string{"mobile"}},
				{Field: "preferencias_tecnicas.preferencias", Values: []string{"plataformas moviles: iOS y Android"}},
			},
		},
		{
			ID:       "mobile_ios",
			LabelKey: "nueva_app.wizard.guided_mobile_ios",
			Decides: []WebNuevaAppIntakeDecisionV0{
				{Field: "tipo_app", Value: "mobile"},
				{Field: "plataformas", Values: []string{"mobile"}},
				{Field: "preferencias_tecnicas.preferencias", Values: []string{"plataforma movil: iOS"}},
			},
		},
		{
			ID:       "mobile_android",
			LabelKey: "nueva_app.wizard.guided_mobile_android",
			Decides: []WebNuevaAppIntakeDecisionV0{
				{Field: "tipo_app", Value: "mobile"},
				{Field: "plataformas", Values: []string{"mobile"}},
				{Field: "preferencias_tecnicas.preferencias", Values: []string{"plataforma movil: Android"}},
			},
		},
		{
			ID:       "data_external",
			LabelKey: "nueva_app.wizard.guided_data_external",
			Decides: []WebNuevaAppIntakeDecisionV0{
				{Field: "datos.db_required", Value: "false"},
				{Field: "integraciones.1.tipo", Value: "api"},
				{Field: "integraciones.1.nombre", Value: "datos externos de alquileres"},
				{Field: "integraciones.1.proposito", Value: "Consultar o importar datos existentes de pisos en alquiler."},
			},
		},
		{
			ID:       "data_management",
			LabelKey: "nueva_app.wizard.guided_data_management",
			Decides:  guidedRentalDataDecisionsV0(),
		},
		{
			ID:       "maps_generic",
			LabelKey: "nueva_app.wizard.guided_maps_generic",
			Decides:  guidedMapDecisionsV0(false),
		},
		{
			ID:       "maps_osm",
			LabelKey: "nueva_app.wizard.guided_maps_osm",
			Decides:  guidedMapDecisionsV0(true),
		},
		{
			ID:       "architecture_default",
			LabelKey: "nueva_app.wizard.guided_architecture_default",
			Decides:  []WebNuevaAppIntakeDecisionV0{{Field: "preferencias_tecnicas.arquitectura", Value: "hexagonal"}},
		},
		{
			ID:       "architecture_event",
			LabelKey: "nueva_app.wizard.guided_architecture_event",
			Decides:  []WebNuevaAppIntakeDecisionV0{{Field: "preferencias_tecnicas.arquitectura", Value: "event_driven"}},
		},
		{
			ID:       "architecture_modular",
			LabelKey: "nueva_app.wizard.guided_architecture_modular",
			Decides:  []WebNuevaAppIntakeDecisionV0{{Field: "preferencias_tecnicas.arquitectura", Value: "modular_monolith"}},
		},
		{
			ID:       "quality_public",
			LabelKey: "nueva_app.wizard.guided_quality_public",
			Decides: []WebNuevaAppIntakeDecisionV0{
				{Field: "calidad.pruebas", Value: "alta"},
				{Field: "calidad.accesibilidad", Value: "wcag_aa"},
				{Field: "calidad.accesibilidad_opciones", Values: []string{"normal", "wcag_aa"}},
			},
		},
	}
}

func guidedRentalDataDecisionsV0() []WebNuevaAppIntakeDecisionV0 {
	return []WebNuevaAppIntakeDecisionV0{
		{Field: "datos.db_required", Value: "true"},
		{Field: "datos.necesidad_funcional", Value: "Gestionar y consultar pisos en alquiler, ubicaciones, favoritos y alertas."},
		{Field: "datos.tipos_datos", Values: []string{"pisos", "alquileres", "ubicaciones", "favoritos", "alertas"}},
		{Field: "datos.tipos_detallados.0.nombre", Value: "Pisos en alquiler"},
		{Field: "datos.tipos_detallados.0.proposito", Value: "Mostrar pisos cercanos y sus datos principales."},
		{Field: "datos.tipos_detallados.0.sensibilidad", Value: "publica"},
		{Field: "datos.tipos_detallados.0.retencion", Value: "mientras el anuncio este activo"},
		{Field: "datos.tipos_detallados.0.volumen", Value: "alto"},
		{Field: "datos.tipos_detallados.1.nombre", Value: "Usuarios y favoritos"},
		{Field: "datos.tipos_detallados.1.proposito", Value: "Guardar favoritos, busquedas y alertas."},
		{Field: "datos.tipos_detallados.1.sensibilidad", Value: "personal"},
		{Field: "datos.storage.0.tipo", Value: "relacional"},
		{Field: "datos.storage.0.proposito", Value: "Consultas consistentes de anuncios, usuarios y favoritos."},
		{Field: "datos.storage.0.requerido", Value: "true"},
		{Field: "datos.storage.1.tipo", Value: "busqueda"},
		{Field: "datos.storage.1.proposito", Value: "Filtrado por ubicacion, precio y preferencias."},
	}
}

func guidedMapDecisionsV0(preferOpenStreetMap bool) []WebNuevaAppIntakeDecisionV0 {
	decisions := []WebNuevaAppIntakeDecisionV0{
		{Field: "integraciones.0.tipo", Value: "maps"},
		{Field: "integraciones.0.nombre", Value: "capacidad de mapas"},
		{Field: "integraciones.0.proposito", Value: "Mostrar ubicacion, cercania y rutas aproximadas sin acoplar el dominio a un proveedor."},
		{Field: "integraciones.0.requerido", Value: "true"},
	}
	if preferOpenStreetMap {
		decisions = append(decisions,
			WebNuevaAppIntakeDecisionV0{Field: "integraciones.0.restricciones", Values: []string{"preferir OpenStreetMap si el adaptador autorizado lo soporta"}},
			WebNuevaAppIntakeDecisionV0{Field: "preferencias_tecnicas.preferencias", Values: []string{"mapas: OpenStreetMap preferido por adaptador"}},
		)
	}
	return decisions
}

func guidedCapabilityIntegrationDecisionsV0(
	index int,
	kind string,
	name string,
	purpose string,
	auth string,
) []WebNuevaAppIntakeDecisionV0 {
	if index < 0 || index > 3 {
		return nil
	}
	prefix := "integraciones." + string(rune('0'+index)) + "."
	return []WebNuevaAppIntakeDecisionV0{
		{Field: prefix + "tipo", Value: kind},
		{Field: prefix + "nombre", Value: name},
		{Field: prefix + "proposito", Value: purpose},
		{Field: prefix + "auth", Value: auth},
	}
}

func guidedTitleFromNeedV0(need string) string {
	normalized := normalizeGuidedNeedV0(need)
	if guidedContainsAnyV0(normalized, "piso", "pisos", "alquiler", "alquileres", "vivienda") {
		return "Alquileres cercanos"
	}
	fields := strings.Fields(strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r == ':' || r == ';' || r == ',' || r == '.' {
			return ' '
		}
		return r
	}, trimV0(need)))
	if len(fields) > 5 {
		fields = fields[:5]
	}
	if len(fields) == 0 {
		return "Nueva app"
	}
	return strings.Join(fields, " ")
}

func normalizeGuidedNeedV0(value string) string {
	normalized := strings.ToLower(trimV0(value))
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
		"ñ", "n",
	)
	return replacer.Replace(normalized)
}

func guidedContainsAnyV0(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
