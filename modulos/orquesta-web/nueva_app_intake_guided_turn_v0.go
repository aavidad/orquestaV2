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
	if guidedContainsAnyV0(normalized, "movil", "mobile", "android", "ios", "iphone", "apple") {
		decisions = append(decisions, WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: "mobile"})
		platforms = append(platforms, "mobile")
	}
	if guidedContainsAnyV0(normalized, "api", "backend", "servicio") {
		platforms = append(platforms, "api")
	}
	if guidedContainsAnyV0(normalized, "web", "panel", "gestion") {
		platforms = append(platforms, "web")
	}
	if len(platforms) > 0 {
		decisions = append(decisions, WebNuevaAppIntakeDecisionV0{Field: "plataformas", Values: compactStringsV0(platforms)})
	}
	if guidedContainsAnyV0(normalized, "piso", "pisos", "alquiler", "alquileres", "vivienda", "rent") {
		decisions = append(decisions, guidedRentalDataDecisionsV0()...)
	}
	if guidedContainsAnyV0(normalized, "mapa", "mapas", "map", "maps", "cerca", "cercano", "cercanos", "ubicacion", "geo", "openstreet") {
		decisions = append(decisions, guidedMapDecisionsV0(strings.Contains(normalized, "openstreet"))...)
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
