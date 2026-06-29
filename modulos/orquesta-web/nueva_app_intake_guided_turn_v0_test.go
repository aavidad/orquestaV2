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
	if session.Estado != WebNuevaAppIntakeEstadoLista {
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

func TestWebNuevaAppIntakeGuidedActionV0IgnoraAccionDesconocida(t *testing.T) {
	session := NewWebNuevaAppIntakeSessionV0("session-guided-3", "es", "Portal", "Publicar viviendas")
	before := session.Form

	session = ApplyWebNuevaAppIntakeGuidedActionV0(session, "accion_inexistente")

	if session.Form.Nombre != before.Nombre || session.Form.Objetivo != before.Objetivo {
		t.Fatalf("accion desconocida cambio la sesion: before=%+v after=%+v", before, session.Form)
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

func stringSliceHasV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
