package orquestaweb

import (
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestWebNuevaAppIntakeGuidedTurnV0RellenaAppMovilAlquileresConMapas(t *testing.T) {
	turn := NewWebNuevaAppIntakeGuidedTurnV0("Quiero una app para movil que muestre datos de los pisos en alquiler cercanos con OpenStreetMap")
	session := NewWebNuevaAppIntakeSessionV0("session-guided-1", "es", "", "")

	session = ApplyWebNuevaAppIntakeGuidedTurnV0(session, turn)

	if turn.SchemaVersion != WebNuevaAppIntakeGuidedTurnSchemaV0 || len(turn.Followups) == 0 {
		t.Fatalf("turn incompleto: %+v", turn)
	}
	if session.Estado != WebNuevaAppIntakeEstadoRequiereDatos ||
		!stringSliceHasV0(session.PendingQuestions, "preferencias_tecnicas") ||
		!stringSliceHasV0(session.PendingQuestions, "deploy") ||
		!stringSliceHasV0(session.PendingQuestions, "calidad") {
		t.Fatalf("estado=%q pending=%+v", session.Estado, session.PendingQuestions)
	}
	if session.Form.Nombre != "Alquileres cercanos" ||
		session.Form.TipoApp != "mobile" ||
		!stringSliceHasV0(session.Form.Plataformas, "mobile") {
		t.Fatalf("identidad/plataformas no inferidas: %+v", session.Form)
	}
	if !session.Form.Datos.DBRequired ||
		len(session.Form.Datos.TiposDetallados) != 2 ||
		session.Form.Datos.TiposDetallados[0].Nombre != "Pisos en alquiler" ||
		session.Form.Datos.TiposDetallados[1].Sensibilidad != "personal" ||
		len(session.Form.Datos.Storage) != 2 ||
		session.Form.Datos.Storage[0].Tipo != "relacional" ||
		session.Form.Datos.Storage[1].Tipo != "busqueda" {
		t.Fatalf("datos guiados no aplicados: %+v", session.Form.Datos)
	}
	if len(session.Form.Integraciones) == 0 ||
		session.Form.Integraciones[0].Tipo != "maps" ||
		!stringSliceHasV0(session.Form.Integraciones[0].Restricciones, "preferir OpenStreetMap si el adaptador autorizado lo soporta") {
		t.Fatalf("mapas no aplicados como capacidad: %+v", session.Form.Integraciones)
	}
	req := session.Form.ToAppSpecRequestV0()
	if issues := orquestafactory.ValidateAppSpecRequestV0(req); len(issues) != 0 {
		t.Fatalf("request guiada debe ser valida para factory: %+v", issues)
	}
}

func TestWebNuevaAppIntakeGuidedActionV0AplicaDudasSinCambiarContratoFinal(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-2", "es", "Portal", "Publicar viviendas")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: "web"})

	session = ApplyWebNuevaAppIntakeGuidedActionV0(session, "architecture_event")
	session = ApplyWebNuevaAppIntakeGuidedActionV0(session, "quality_public")
	session = ApplyWebNuevaAppIntakeGuidedActionV0(session, "data_external")

	if session.Form.PreferenciasTecnicas.Arquitectura != "event_driven" {
		t.Fatalf("arquitectura=%+v", session.Form.PreferenciasTecnicas)
	}
	if session.Form.Calidad.Pruebas != "alta" ||
		session.Form.Calidad.Accesibilidad != "wcag_aa" ||
		!stringSliceHasV0(session.Form.Calidad.AccesibilidadOpciones, "normal") {
		t.Fatalf("calidad=%+v", session.Form.Calidad)
	}
	if len(session.Form.Integraciones) < 2 ||
		session.Form.Integraciones[1].Tipo != "api" ||
		session.Form.Integraciones[1].Nombre != "datos externos de alquileres" {
		t.Fatalf("datos externos=%+v", session.Form.Integraciones)
	}
	req := session.Form.ToAppSpecRequestV0()
	if req.PreferenciasTecnicas.Arquitectura != "event_driven" ||
		req.Calidad.Accesibilidad != "wcag_aa" ||
		len(req.Calidad.AccesibilidadOpciones) != 2 {
		t.Fatalf("request no conserva decisiones guiadas: %+v", req)
	}
}

func TestWebNuevaAppIntakeGuidedActionV0CubreOpcionesExpertasV0(t *testing.T) {
	architectures := map[string]string{
		"architecture_clean":         "clean_architecture",
		"architecture_onion":         "onion",
		"architecture_layered":       "layered",
		"architecture_microservices": "microservices",
		"architecture_serverless":    "serverless",
		"architecture_plugin":        "plugin_based",
		"architecture_data_pipeline": "data_pipeline",
	}
	for action, want := range architectures {
		session := NewWebNuevaAppIntakeSessionV0("session-guided-architecture-"+action, "es", "Portal", "Publicar viviendas")
		session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: "web"})
		session = ApplyWebNuevaAppIntakeGuidedActionV0(session, action)
		if session.Form.PreferenciasTecnicas.Arquitectura != want {
			t.Fatalf("action=%s arquitectura=%+v want %s", action, session.Form.PreferenciasTecnicas, want)
		}
		req := session.Form.ToAppSpecRequestV0()
		if issues := orquestafactory.ValidateAppSpecRequestV0(req); len(issues) != 0 {
			t.Fatalf("action=%s request invalida: %+v", action, issues)
		}
	}

	session := NewWebNuevaAppIntakeSessionV0("session-guided-quality-expert", "es", "Servicio", "Procesar datos internos")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: "api"})
	session = ApplyWebNuevaAppIntakeGuidedActionV0(session, "quality_regulated")
	if session.Form.Calidad.Pruebas != "alta" ||
		session.Form.Calidad.Observabilidad == nil ||
		!*session.Form.Calidad.Observabilidad ||
		!session.Form.Datos.Operacion.Auditoria ||
		!stringSliceHasV0(session.Form.Calidad.Compliance, "proteccion_datos") {
		t.Fatalf("quality_regulated=%+v datos=%+v", session.Form.Calidad, session.Form.Datos)
	}
	session = ApplyWebNuevaAppIntakeGuidedActionV0(session, "accessibility_none")
	if session.Form.Calidad.Accesibilidad != "no_aplica" ||
		!stringSliceHasV0(session.Form.Calidad.AccesibilidadOpciones, "no_aplica") {
		t.Fatalf("accessibility_none=%+v", session.Form.Calidad)
	}
	req := session.Form.ToAppSpecRequestV0()
	if issues := orquestafactory.ValidateAppSpecRequestV0(req); len(issues) != 0 {
		t.Fatalf("request de calidad experta invalida: %+v", issues)
	}
}

func TestWebNuevaAppIntakeDecisionV0ConservaPreferenciasTecnicasEnHandoff(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-tech", "es", "Portal", "Publicar viviendas")
	decisions := []WebNuevaAppIntakeDecisionV0{
		{Field: "tipo_app", Value: "web"},
		{Field: "preferencias_tecnicas.lenguaje", Value: "Go"},
		{Field: "preferencias_tecnicas.framework", Value: "HTMX"},
		{Field: "preferencias_tecnicas.restricciones", Value: "sin SPA pesada, sin ORM"},
		{Field: "preferencias_tecnicas.preferencias", Value: "render server-side, colas internas"},
	}
	for _, decision := range decisions {
		session = session.ApplyDecisionV0(decision)
	}

	if session.Form.PreferenciasTecnicas.Lenguaje != "Go" ||
		session.Form.PreferenciasTecnicas.Framework != "HTMX" ||
		!stringSliceHasV0(session.Form.PreferenciasTecnicas.Restricciones, "sin SPA pesada") ||
		!stringSliceHasV0(session.Form.PreferenciasTecnicas.Preferencias, "colas internas") {
		t.Fatalf("preferencias tecnicas no aplicadas al formulario: %+v", session.Form.PreferenciasTecnicas)
	}
	req := session.Form.ToAppSpecRequestV0()
	if req.PreferenciasTecnicas.Lenguaje != "Go" ||
		req.PreferenciasTecnicas.Framework != "HTMX" ||
		!stringSliceHasV0(req.PreferenciasTecnicas.Restricciones, "sin ORM") ||
		!stringSliceHasV0(req.PreferenciasTecnicas.Preferencias, "render server-side") {
		t.Fatalf("preferencias tecnicas no conservadas en handoff: %+v", req.PreferenciasTecnicas)
	}
	if session.AppSpecPartial.PreferenciasTecnicas.Lenguaje != "Go" ||
		session.AppSpecPartial.PreferenciasTecnicas.Framework != "HTMX" {
		t.Fatalf("app spec parcial no refrescada: %+v", session.AppSpecPartial.PreferenciasTecnicas)
	}
}

func TestWebNuevaAppIntakeGuidedActionV0IgnoraAccionDesconocida(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-3", "es", "Portal", "Publicar viviendas")
	before := session.Form

	session = ApplyWebNuevaAppIntakeGuidedActionV0(session, "accion_inexistente")

	if session.Form.Nombre != before.Nombre || session.Form.Objetivo != before.Objetivo {
		t.Fatalf("accion desconocida cambio la sesion: before=%+v after=%+v", before, session.Form)
	}
}

func TestApplyWebNuevaAppIntakeGuidedAnswerV0NormalizaTipoAppLibreV0(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-free-type", "es", "Portal", "Publicar viviendas")

	session = ApplyWebNuevaAppIntakeGuidedAnswerV0(session, "tipo_app", "aplicación web con API")

	if session.Form.TipoApp != "web" || session.Estado != WebNuevaAppIntakeEstadoRequiereDatos {
		t.Fatalf("respuesta libre tipo_app no normalizada: %+v", session.Form)
	}
	req := session.Form.ToAppSpecRequestV0()
	if issues := orquestafactory.ValidateAppSpecRequestV0(req); len(issues) != 0 {
		t.Fatalf("request debe ser valida tras alias recuperable: %+v", issues)
	}
}

func TestApplyWebNuevaAppIntakeGuidedAnswerV0NormalizaArquitecturasExpertasV0(t *testing.T) {
	cases := map[string]string{
		"microservicios con despliegues independientes": "microservices",
		"serverless por eventos":                        "serverless",
		"plugins instalables":                           "plugin_based",
		"pipeline de ingesta ETL":                       "data_pipeline",
		"arquitectura onion":                            "onion",
	}
	for answer, want := range cases {
		session := NewWebNuevaAppIntakeSessionV0("session-guided-free-architecture", "es", "Portal", "Publicar viviendas")
		session = ApplyWebNuevaAppIntakeGuidedAnswerV0(session, "preferencias_tecnicas", answer)
		if session.Form.PreferenciasTecnicas.Arquitectura != want {
			t.Fatalf("answer=%q arquitectura=%q want %q", answer, session.Form.PreferenciasTecnicas.Arquitectura, want)
		}
	}
}

func TestApplyWebNuevaAppIntakeGuidedAnswerV0NormalizaSDKComoPluginVisibleV0(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-free-sdk", "es", "SDK", "Crear una libreria para integraciones internas")

	session = ApplyWebNuevaAppIntakeGuidedAnswerV0(session, "tipo_app", "library sdk reutilizable")

	if session.Form.TipoApp != "plugin" || session.Estado != WebNuevaAppIntakeEstadoRequiereDatos {
		t.Fatalf("sdk/library debe normalizarse a opcion visible plugin: %+v", session.Form)
	}
	req := session.Form.ToAppSpecRequestV0()
	if issues := orquestafactory.ValidateAppSpecRequestV0(req); len(issues) != 0 {
		t.Fatalf("request debe ser valida tras alias sdk/library: %+v", issues)
	}
}

func TestApplyWebNuevaAppIntakeGuidedAnswerV0SplitLibrePlataformasV0(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-free-platforms", "es", "Portal", "Publicar viviendas")

	session = ApplyWebNuevaAppIntakeGuidedAnswerV0(session, "plataformas", "web y móvil, API")

	if !stringSliceHasV0(session.Form.Plataformas, "web") ||
		!stringSliceHasV0(session.Form.Plataformas, "mobile") ||
		!stringSliceHasV0(session.Form.Plataformas, "api") {
		t.Fatalf("plataformas no normalizadas: %+v", session.Form.Plataformas)
	}
}

func TestWebNuevaAppIntakeGuidedTurnV0ReconoceIntegracionesFrecuentes(t *testing.T) {
	turn := NewWebNuevaAppIntakeGuidedTurnV0("portal con calendario, pagos, login, permisos y notificaciones")
	session := NewWebNuevaAppIntakeSessionV0("session-guided-integraciones", "es", "", "")

	session = ApplyWebNuevaAppIntakeGuidedTurnV0(session, turn)

	if len(session.Form.Integraciones) != 4 {
		t.Fatalf("integraciones=%+v", session.Form.Integraciones)
	}
	want := []string{"calendar", "payments", "auth", "notifications"}
	for index, kind := range want {
		if session.Form.Integraciones[index].Tipo != kind ||
			session.Form.Integraciones[index].Nombre == "" ||
			session.Form.Integraciones[index].Proposito == "" ||
			session.Form.Integraciones[index].Auth == "" {
			t.Fatalf("integracion %d=%+v want kind=%s", index, session.Form.Integraciones[index], kind)
		}
	}
	req := session.Form.ToAppSpecRequestV0()
	if issues := orquestafactory.ValidateAppSpecRequestV0(req); len(issues) != 0 {
		t.Fatalf("request guiada debe ser valida para factory: %+v", issues)
	}
}

func TestGuidedCapabilityIntegrationDecisionsV0PermiteSeisFilasV0(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-six", "es", "Portal", "Publicar viviendas")
	session = session.ApplyDecisionV0(WebNuevaAppIntakeDecisionV0{Field: "tipo_app", Value: "web"})
	for _, decision := range guidedCapabilityIntegrationDecisionsV0(5, "storage", "documentos", "guardar adjuntos", "signed-url") {
		session = session.ApplyDecisionV0(decision)
	}
	for _, decision := range guidedCapabilityIntegrationDecisionsV0(6, "llm", "asistente", "redactar borradores", "oauth") {
		session = session.ApplyDecisionV0(decision)
	}

	if len(session.Form.Integraciones) != 6 ||
		session.Form.Integraciones[5].Tipo != "storage" ||
		session.Form.Integraciones[5].Nombre != "documentos" ||
		session.Form.Integraciones[5].Proposito != "guardar adjuntos" ||
		session.Form.Integraciones[5].Auth != "signed-url" {
		t.Fatalf("sexta integracion guiada no aplicada: %+v", session.Form.Integraciones)
	}
}

func stringSliceHasV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
