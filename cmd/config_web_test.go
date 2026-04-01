package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxConfigWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerConfigGuardar(w, r)
			return
		}
		webHandlerConfig(w, r)
	})
	registerAPIRoutes(mux)
	return mux
}

func TestWebConfigListaYGuardaPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.ConfigSet("workspace_root", "/tmp/orquesta"); err != nil {
		t.Fatalf("config set inicial: %v", err)
	}
	if err := db.ConfigSet("integration_openclaw_transport", "mcp_stdio"); err != nil {
		t.Fatalf("config set openclaw inicial: %v", err)
	}
	if err := db.ConfigSet("integration_openclaw_endpoint", webOpenClawDefaultEndpoint()); err != nil {
		t.Fatalf("config set openclaw endpoint: %v", err)
	}
	if err := db.ConfigSet("integration_openclaw_require_identity", "true"); err != nil {
		t.Fatalf("config set openclaw identidad: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config?lang=en", nil)
	testMuxConfigWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("config status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"workspace_root",
		"/tmp/orquesta",
		"OpenClaw integration",
		"action=\"/config?lang=en\"",
		"integration_openclaw_transport",
		"mcp_stdio",
		"integration_openclaw_endpoint",
		webOpenClawDefaultEndpoint(),
		"integration_openclaw_require_identity",
		"value=\"true\"",
		"<html lang=\"en\">",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("config web incompleta, falta %q:\n%s", token, body)
		}
	}

	form := url.Values{
		"clave": {"workspace_root"},
		"valor": {"/srv/orquesta"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxConfigWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar config status=%d body=%s", rec.Code, rec.Body.String())
	}
	valor, err := db.ConfigGet("workspace_root")
	if err != nil {
		t.Fatalf("config get actualizada: %v", err)
	}
	if valor != "/srv/orquesta" {
		t.Fatalf("valor inesperado: %s", valor)
	}

	form = url.Values{
		"clave": {"integration_openclaw_transport"},
		"valor": {"mcp_http"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxConfigWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar config openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	valor, err = db.ConfigGet("integration_openclaw_transport")
	if err != nil {
		t.Fatalf("config get openclaw actualizada: %v", err)
	}
	if valor != "mcp_http" {
		t.Fatalf("valor openclaw inesperado: %s", valor)
	}
}

func TestWebConfigGuardarValidaClaveConI18n(t *testing.T) {
	prepararDBTemporalCmd(t)

	form := url.Values{
		"clave": {"   "},
		"valor": {"ignorado"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/config?lang=en", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxConfigWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("guardar config sin clave status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "lang=en") {
		t.Fatalf("redirect sin lang: %s", location)
	}
	if !strings.Contains(location, "Key+is+required") {
		t.Fatalf("redirect inesperado: %s", location)
	}
}

func TestWebConfigAplicaPresetOpenClaw(t *testing.T) {
	prepararDBTemporalCmd(t)

	form := url.Values{
		"preset": {"openclaw_server_first"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/config?lang=en", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxConfigWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("aplicar preset status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "lang=en") {
		t.Fatalf("redirect preset sin lang: %s", location)
	}
	if !strings.Contains(location, "OpenClaw+server-first+preset+applied") {
		t.Fatalf("redirect preset inesperado: %s", location)
	}

	for clave, want := range map[string]string{
		"integration_openclaw_enabled":          "true",
		"integration_openclaw_transport":        "mcp_http",
		"integration_openclaw_endpoint":         webOpenClawDefaultEndpoint(),
		"integration_openclaw_workspace_mode":   "worktree",
		"integration_openclaw_agent_prefix":     "OpenClaw-",
		"integration_openclaw_require_identity": "true",
	} {
		got, err := db.ConfigGet(clave)
		if err != nil {
			t.Fatalf("config get %s: %v", clave, err)
		}
		if got != want {
			t.Fatalf("%s=%q, want %q", clave, got, want)
		}
	}
}

func TestWebConfigRechazaPresetDesconocido(t *testing.T) {
	prepararDBTemporalCmd(t)

	form := url.Values{
		"preset": {"desconocido"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/config?lang=en", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxConfigWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("preset desconocido status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "lang=en") {
		t.Fatalf("redirect preset desconocido sin lang: %s", location)
	}
	if !strings.Contains(location, "Unknown+preset%3A+desconocido") {
		t.Fatalf("redirect inesperado para preset desconocido: %s", location)
	}
}
