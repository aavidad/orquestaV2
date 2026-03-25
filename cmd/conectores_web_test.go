/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxConectoresWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/conectores", webHandlerConectores)
	mux.HandleFunc("/conectores/", webRouterConectores)
	registerAPIRoutes(mux)
	return mux
}

func TestWebConectoresListaYGuardaPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)

	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    "codex",
		ArgsJSON:   "[]",
		EnvJSON:    "{}",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/conectores?lang=en", nil)
	testMuxConectoresWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("conectores status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"Codex CLI", "/conectores/codex", "<html lang=\"en\">"} {
		if !strings.Contains(body, token) {
			t.Fatalf("listado de conectores incompleto, falta %q:\n%s", token, body)
		}
	}

	form := url.Values{
		"slug":          {"claude"},
		"nombre":        {"Claude CLI"},
		"transporte":    {"cli"},
		"comando":       {"claude"},
		"args_json":     {"[]"},
		"env_json":      {"{}"},
		"metadata_json": {"{}"},
		"activo":        {"1"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/conectores/nuevo", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxConectoresWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar conector status=%d body=%s", rec.Code, rec.Body.String())
	}
	conector, err := db.GetConector("claude")
	if err != nil {
		t.Fatalf("get conector guardado: %v", err)
	}
	if conector == nil || conector.Comando != "claude" || !conector.Activo {
		t.Fatalf("conector guardado inesperado: %+v", conector)
	}
}
