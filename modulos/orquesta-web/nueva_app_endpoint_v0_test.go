package orquestaweb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestNuevaAppWebEndpointV0GETRenderInicialLocalizadoYOpcionesDelForm(t *testing.T) {
	client := &fakeNuevaAppClientV0{}
	endpoint := NewNuevaAppWebEndpointV0(client)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nueva-app?locale=en", nil)

	endpoint.ServeHTTP(rec, req)

	page := decodeNuevaAppWebPageTestV0(t, rec)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.calls != 0 {
		t.Fatalf("GET no debe llamar al cliente: calls=%d", client.calls)
	}
	if page.SchemaVersion != NuevaAppWebEndpointSchemaV0 ||
		page.Locale != NuevaAppI18nEnglishLocaleV0 ||
		page.Titulo != "New app" ||
		page.ViewModel.Estado != WebNuevaAppEstadoInicial ||
		page.Textos.Estado != "Ready to complete" {
		t.Fatalf("page inicial inesperada: %+v", page)
	}
	if page.Formulario.Contrato != "WebNuevaAppFormV0" || page.Formulario.Acciones.Submit != "Request app" {
		t.Fatalf("formulario inesperado: %+v", page.Formulario)
	}
	nombre := nuevaAppCampoByPathTestV0(page.Formulario.Campos, "nombre")
	if nombre.Label != "Name" || !nombre.Requerido || nombre.Tipo != "texto" {
		t.Fatalf("campo nombre inesperado: %+v", nombre)
	}
	datos := nuevaAppCampoByPathTestV0(page.Formulario.Campos, "datos.db_required")
	if datos.Label != "Needs persistence" || datos.Tipo != "booleano" {
		t.Fatalf("campo datos inesperado: %+v", datos)
	}
	requestKind := nuevaAppCampoByPathTestV0(page.Formulario.Campos, "request_kind")
	if requestKind.Label != "Request type" || requestKind.Tipo != "texto" {
		t.Fatalf("campo request_kind inesperado: %+v", requestKind)
	}
	integracionTipo := nuevaAppCampoByPathTestV0(page.Formulario.Campos, "integraciones.0.tipo")
	if integracionTipo.Label != "Integration type" || integracionTipo.Tipo != "texto" {
		t.Fatalf("campo integracion tipo inesperado: %+v", integracionTipo)
	}
	projectSource := nuevaAppCampoByPathTestV0(page.Formulario.Campos, "project_source")
	if projectSource.Label != "Project source" || projectSource.Tipo != "objeto" {
		t.Fatalf("campo project_source inesperado: %+v", projectSource)
	}
	projectGitURL := nuevaAppCampoByPathTestV0(page.Formulario.Campos, "project_source.git_url")
	if projectGitURL.Label != "Git URL" || projectGitURL.Tipo != "texto" {
		t.Fatalf("campo project_source.git_url inesperado: %+v", projectGitURL)
	}
	for _, field := range page.Formulario.Campos {
		if field.Path == "director_execution_mode" {
			t.Fatalf("director_execution_mode no debe aparecer como campo normal: %+v", field)
		}
		if field.Label == "" {
			t.Fatalf("campo sin label i18n: %+v", field)
		}
	}
	if !nuevaAppHasOptionTestV0(page.Opciones.Locales, NuevaAppI18nDefaultLocaleV0) ||
		!nuevaAppHasOptionTestV0(page.Opciones.Locales, NuevaAppI18nEnglishLocaleV0) {
		t.Fatalf("locales incompletos: %+v", page.Opciones.Locales)
	}
	if !nuevaAppHasOptionTestV0(page.Opciones.Estados, string(WebNuevaAppEstadoRequiereDatos)) ||
		!nuevaAppHasOptionTestV0(page.Opciones.Errores, WebNuevaAppErrTransporteV0) ||
		!nuevaAppHasOptionTestV0(page.Opciones.Errores, orquestafactory.ErrAppSpecInvalida) {
		t.Fatalf("opciones incompletas: estados=%+v errores=%+v", page.Opciones.Estados, page.Opciones.Errores)
	}
}
