package orquestaweb

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestNuevaAppHTMLHandlerV0GETMuestraFormularioUsableSinDelegar(t *testing.T) {
	client := &fakeNuevaAppClientV0{}
	handler := NewNuevaAppHTMLHandlerV0(client)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nueva-app?locale=es", nil)

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
	if client.calls != 0 {
		t.Fatalf("GET no debe delegar: calls=%d", client.calls)
	}
	for _, want := range []string{
		`<form method="post" action="/nueva-app">`,
		`name="request_id"`,
		`name="request_kind"`,
		`name="execution_mode"`,
		`title="Define si Orquesta debe crear una app completa`,
		`name="locale"`,
		`name="nombre"`,
		`name="objetivo"`,
		`name="tipo_app"`,
		`name="plataformas"`,
		`name="preferencias_tecnicas.arquitectura"`,
		`name="i18n.enabled"`,
		`name="datos.db_required"`,
		`name="deploy.target"`,
		`name="calidad.pruebas"`,
		`name="agentes.autonomia"`,
		`name="integraciones.0.tipo"`,
		`Vista previa del backlog`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET HTML no contiene %q\n%s", want, body)
		}
	}
}

func TestNuevaAppHTMLHandlerV0POSTValidoDelegaYRenderizaResultado(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		vm: NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0()),
	}
	handler := NewNuevaAppHTMLHandlerV0(client)
	values := nuevaAppHTMLValidFormValuesV0()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", bytes.NewBufferString(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
	if client.received.Nombre != "Agenda" ||
		client.received.RequestKind != "crear_app_completa" ||
		client.received.ExecutionMode != "normal" ||
		client.received.PreferenciasTecnicas.Arquitectura != "hexagonal" ||
		!client.received.Datos.DBRequired ||
		client.received.Deploy.Target != "contenedor" ||
		client.received.Agentes.Autonomia != "media" ||
		len(client.received.Integraciones) != 1 ||
		client.received.Integraciones[0].Tipo != "api" {
		t.Fatalf("form delegado inesperado: %+v", client.received)
	}
	for _, want := range []string{"Lista para revisar", "Agenda", "BLG-001", "producto / AppSpecV0"} {
		if !strings.Contains(body, want) {
			t.Fatalf("POST valido no contiene %q\n%s", want, body)
		}
	}
}

func TestNuevaAppHTMLHandlerV0POSTInvalidoRenderizaErrorPublico(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		vm: NewWebNuevaAppErrorViewModelV0("req-invalid", "es", []orquestafactory.ValidationIssue{{
			Code:    orquestafactory.ErrAppSpecInvalida,
			Field:   "nombre",
			Message: "campo obligatorio",
		}}),
	}
	handler := NewNuevaAppHTMLHandlerV0(client)
	values := nuevaAppHTMLValidFormValuesV0()
	values.Set("request_id", "req-invalid")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", bytes.NewBufferString(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
	for _, want := range []string{"Necesita correcciones", "app_spec_invalida", "La solicitud de app no es valida."} {
		if !strings.Contains(body, want) {
			t.Fatalf("POST invalido no contiene %q\n%s", want, body)
		}
	}
}

func nuevaAppHTMLValidFormValuesV0() url.Values {
	values := url.Values{}
	values.Set("request_id", "req-html-001")
	values.Set("locale", "es")
	values.Set("request_kind", "crear_app_completa")
	values.Set("execution_mode", "normal")
	values.Set("nombre", "Agenda")
	values.Set("objetivo", "Coordinar ensayos")
	values.Set("descripcion", "Gestion operativa")
	values.Set("tipo_app", "web")
	values.Add("plataformas", "web")
	values.Set("preferencias_tecnicas.arquitectura", "hexagonal")
	values.Set("i18n.enabled", "true")
	values.Set("i18n.default_locale", "es")
	values.Set("datos.db_required", "true")
	values.Set("datos.necesidad_funcional", "guardar disponibilidad")
	values.Set("deploy.target", "contenedor")
	values.Set("calidad.pruebas", "alta")
	values.Set("calidad.observabilidad", "true")
	values.Set("agentes.revision_humana", "true")
	values.Set("agentes.autonomia", "media")
	values.Set("integraciones.0.tipo", "api")
	values.Set("integraciones.0.nombre", "crm")
	values.Set("integraciones.0.proposito", "sincronizar ensayos")
	return values
}
