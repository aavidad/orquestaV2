package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxCapacidadWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/pools", webHandlerPools)
	mux.HandleFunc("/pools/", webRouterPools)
	mux.HandleFunc("/modelo", webHandlerModelo)
	registerAPIRoutes(mux)
	return mux
}

func TestWebPoolsListaDetalleYGuardarPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)

	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:               "codex",
		Proveedor:          "OpenAI",
		Runtime:            "codex",
		Plan:               "default",
		CapacidadTotal:     4,
		CapacidadReservada: 1,
		PoliticaHandoff:    "preventivo",
		FuenteTelemetria:   "manual",
		MetadataJSON:       "{}",
		Activo:             true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := db.GuardarPoolModelo("codex", &db.PoolModelo{
		ModelSlug:          "gpt-5",
		Activo:             true,
		Prioridad:          1,
		CosteRelativo:      1.0,
		LimiteConocidoJSON: "{}",
	}); err != nil {
		t.Fatalf("guardar modelo pool: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pools?lang=en", nil)
	testMuxCapacidadWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pools status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"codex", "/pools/codex", "<html lang=\"en\">"} {
		if !strings.Contains(body, token) {
			t.Fatalf("listado de pools incompleto, falta %q:\n%s", token, body)
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/pools/codex?lang=en", nil)
	testMuxCapacidadWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle pool status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "gpt-5") {
		t.Fatalf("detalle de pool inesperado:\n%s", rec.Body.String())
	}

	form := url.Values{
		"slug":                {"claude"},
		"proveedor":           {"Anthropic"},
		"runtime":             {"claude"},
		"plan":                {"default"},
		"capacidad_total":     {"3"},
		"capacidad_reservada": {"0"},
		"politica_handoff":    {"preventivo"},
		"fuente_telemetria":   {"manual"},
		"metadata_json":       {"{}"},
		"activo":              {"1"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/pools/guardar", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxCapacidadWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar pool status=%d body=%s", rec.Code, rec.Body.String())
	}
	detail, err := capacidadService.GetPoolDetail("claude")
	if err != nil {
		t.Fatalf("detalle pool guardado: %v", err)
	}
	if detail == nil || detail.Pool == nil || detail.Pool.Proveedor != "Anthropic" {
		t.Fatalf("pool guardado inesperado: %+v", detail)
	}
}

func TestWebModeloResuelvePorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)

	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:               "codex",
		Proveedor:          "OpenAI",
		Runtime:            "codex",
		Plan:               "default",
		CapacidadTotal:     4,
		CapacidadReservada: 0,
		PoliticaHandoff:    "preventivo",
		FuenteTelemetria:   "manual",
		MetadataJSON:       "{}",
		Activo:             true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := db.GuardarPoolModelo("codex", &db.PoolModelo{
		ModelSlug:          "gpt-5",
		Activo:             true,
		Prioridad:          1,
		CosteRelativo:      1.0,
		LimiteConocidoJSON: "{}",
	}); err != nil {
		t.Fatalf("guardar modelo pool: %v", err)
	}
	if _, err := db.GuardarPoliticaModelo(&db.PoliticaModelo{
		ScopeTipo:       "global",
		PerfilTarea:     "programador",
		PoolSlug:        "codex",
		ModelSlug:       "gpt-5",
		ReasoningEffort: "high",
		Prioridad:       1,
		Activa:          true,
		MetadataJSON:    "{}",
	}); err != nil {
		t.Fatalf("guardar politica modelo: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/modelo?perfil=programador&lang=en", nil)
	testMuxCapacidadWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("modelo status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"gpt-5", "codex", "high"} {
		if !strings.Contains(body, token) {
			t.Fatalf("resolucion de modelo incompleta, falta %q:\n%s", token, body)
		}
	}
}
