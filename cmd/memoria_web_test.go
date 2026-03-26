package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxMemoriaWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/memoria", webHandlerMemoria)
	mux.HandleFunc("/memoria/", webRouterMemoria)
	registerAPIRoutes(mux)
	return mux
}

func TestWebMemoriaListaDetalleYGuardarPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertEntidadMemoria(&db.EntidadMemoria{
		Nombre:        "Core_API",
		Tipo:          "api",
		ValorJSON:     `{"estado":"ok"}`,
		MetadataJSON:  `{"fuente":"test"}`,
		VerificadoPor: "Codex1",
		ProyectoID:    &proyectoID,
	}); err != nil {
		t.Fatalf("upsert memoria: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/memoria?proyecto=orquestador&lang=en", nil)
	testMuxMemoriaWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("memoria status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"Core_API", "/memoria/Core_API?proyecto=orquestador", "<html lang=\"en\">"} {
		if !strings.Contains(body, token) {
			t.Fatalf("listado de memoria incompleto, falta %q:\n%s", token, body)
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/memoria/Core_API?proyecto=orquestador&lang=en", nil)
	testMuxMemoriaWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle memoria status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Core_API") || !strings.Contains(rec.Body.String(), "Codex1") {
		t.Fatalf("detalle de memoria inesperado:\n%s", rec.Body.String())
	}

	form := url.Values{
		"proyecto":       {"orquestador"},
		"nombre":         {"UI_Map"},
		"tipo":           {"negocio"},
		"valor":          {`{"pantallas":3}`},
		"metadata":       {`{"fuente":"web"}`},
		"verificado_por": {"Codex2"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/memoria/guardar", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxMemoriaWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar memoria status=%d body=%s", rec.Code, rec.Body.String())
	}

	entidades, err := db.ListarEntidadesMemoria(db.FiltroEntidadesMemoria{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar memoria guardada: %v", err)
	}
	var entidad *db.EntidadMemoria
	for _, item := range entidades {
		if item != nil && item.Nombre == "UI_Map" {
			entidad = item
			break
		}
	}
	if entidad == nil || entidad.Tipo != "negocio" || entidad.VerificadoPor != "Codex2" {
		t.Fatalf("entidad guardada inesperada: %+v lista=%+v", entidad, entidades)
	}
}
