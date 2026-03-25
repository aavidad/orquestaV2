package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebRenderResuelveIdiomaPorQuery(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?lang=en", nil)

	webRender(rec, req, webTplLayout+`{{define "content"}}<p>{{tr "Tareas"}}</p>{{end}}`, map[string]any{})

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<html lang=\"en\">") {
		t.Fatalf("html lang inesperado: %s", body)
	}
	if !strings.Contains(body, ">Tasks<") {
		t.Fatalf("traduccion inesperada: %s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q, want en", got)
	}
	found := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == webLangCookieName && cookie.Value == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("se esperaba cookie %s=en", webLangCookieName)
	}
}

func TestWebRenderResuelveIdiomaPorCookieYAceptaHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: webLangCookieName, Value: "en"})

	webRender(rec, req, webTplLayout+`{{define "content"}}<p>{{tr "Agentes"}}</p>{{end}}`, map[string]any{})

	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q, want en", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, ">Agents<") {
		t.Fatalf("traduccion inesperada por cookie: %s", body)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,es;q=0.7")

	webRender(rec, req, webTplLayout+`{{define "content"}}<p>{{tr "Agentes"}}</p>{{end}}`, map[string]any{})

	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language por header=%q, want en", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, ">Agents<") || !strings.Contains(body, "<html lang=\"en\">") {
		t.Fatalf("traduccion inesperada por Accept-Language: %s", body)
	}
}
