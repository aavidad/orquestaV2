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

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config?lang=en", nil)
	testMuxConfigWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("config status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"workspace_root", "/tmp/orquesta", "<html lang=\"en\">"} {
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
}
