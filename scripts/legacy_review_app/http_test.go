// Este fichero prueba la frontera local, sus negativos y el HTML accesible.
package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestHTTPRequiresExactHostAndAuthorization(t *testing.T) {
	app, _ := testApplication(t)
	unauthorized := performRequest(app, http.MethodGet, "http://127.0.0.1:8787/", nil, false)
	if unauthorized.Code != http.StatusUnauthorized ||
		!strings.Contains(unauthorized.Header().Get("WWW-Authenticate"), app.catalog.text("auth_realm")) {
		t.Fatalf("autorización no exigida: código=%d cabeceras=%v", unauthorized.Code, unauthorized.Header())
	}

	request := httptest.NewRequest(http.MethodGet, "http://otro.invalid/", nil)
	request.Host = "otro.invalid"
	request.SetBasicAuth("local", testAuthorization)
	response := httptest.NewRecorder()
	app.handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest ||
		!strings.Contains(response.Body.String(), app.catalog.text("error_host")) {
		t.Fatalf("host alternativo no rechazado: %d %s", response.Code, response.Body.String())
	}
}

func TestHTTPRejectsTraversalOriginCSRFAndContentType(t *testing.T) {
	app, item := testApplication(t)
	traversal := performRequest(app, http.MethodGet, "http://127.0.0.1:8787/%2e%2e/secreto", nil, true)
	if traversal.Code != http.StatusBadRequest ||
		!strings.Contains(traversal.Body.String(), app.catalog.text("error_traversal")) {
		t.Fatalf("traversal no rechazado: %d %s", traversal.Code, traversal.Body.String())
	}

	form := validForm(app, item, 0, strings.Repeat("b", 32))
	noOrigin := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8787/proposal", strings.NewReader(form.Encode()))
	noOrigin.Host = app.allowedHost
	noOrigin.SetBasicAuth("local", testAuthorization)
	noOrigin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	noOriginResponse := httptest.NewRecorder()
	app.handler().ServeHTTP(noOriginResponse, noOrigin)
	if noOriginResponse.Code != http.StatusForbidden {
		t.Fatalf("origen ausente aceptado: %d", noOriginResponse.Code)
	}

	badCSRF := cloneForm(form)
	badCSRF.Set("csrf_token", "incorrecto")
	response := performRequest(app, http.MethodPost, "http://127.0.0.1:8787/proposal", badCSRF, true)
	if response.Code != http.StatusForbidden ||
		!strings.Contains(response.Body.String(), app.catalog.text("error_csrf")) {
		t.Fatalf("CSRF inválido no rechazado: %d %s", response.Code, response.Body.String())
	}

	plain := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8787/proposal", strings.NewReader("x"))
	plain.Host = app.allowedHost
	plain.SetBasicAuth("local", testAuthorization)
	plain.Header.Set("Origin", app.allowedOrigin)
	plain.Header.Set("Content-Type", "text/plain")
	plainResponse := httptest.NewRecorder()
	app.handler().ServeHTTP(plainResponse, plain)
	if plainResponse.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("tipo de contenido inseguro aceptado: %d", plainResponse.Code)
	}
}

func TestHTTPRendersInventoryDetailAndKeyboardSemantics(t *testing.T) {
	app, item := testApplication(t)
	list := performRequest(app, http.MethodGet, "http://127.0.0.1:8787/?q=agentes&family=agentes", nil, true)
	body := list.Body.String()
	for _, expected := range []string{
		app.catalog.text("list_title"),
		app.catalog.text("filter_search"),
		app.catalog.text("action_review"),
		"&lt;script&gt;alert",
		`href="#contenido"`,
		`<label for="q">`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("la lista no contiene %q", expected)
		}
	}
	if strings.Contains(body, "<script>") || list.Header().Get("Content-Security-Policy") == "" ||
		list.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("renderizado o cabeceras inseguros: %s", body)
	}

	detail := performRequest(
		app,
		http.MethodGet,
		"http://127.0.0.1:8787/item?id="+url.QueryEscape(item.ID),
		nil,
		true,
	)
	detailBody := detail.Body.String()
	for _, expected := range []string{
		app.catalog.text("sources_title"),
		app.catalog.text("attempts_title"),
		app.catalog.text("uncertainties_title"),
		"legacy/ref uno",
		"intento v1",
		"cuota no acreditada",
		`<fieldset class="rules">`,
		`<label for="reason">`,
		app.catalog.text("proposal_boundary"),
	} {
		if !strings.Contains(detailBody, expected) {
			t.Errorf("el detalle no contiene %q", expected)
		}
	}
	if strings.Contains(detailBody, "<script>") {
		t.Fatal("el detalle no escapó contenido del inventario")
	}
}

func TestHTTPWritesReplayAndRevisionConflictWithoutCreatingOtherEffects(t *testing.T) {
	app, item := testApplication(t)
	inventoryBefore, err := os.ReadFile(app.inventory.path)
	if err != nil {
		t.Fatal(err)
	}
	nonce := strings.Repeat("c", 32)
	form := validForm(app, item, 0, nonce)
	first := performRequest(app, http.MethodPost, "http://127.0.0.1:8787/proposal", form, true)
	if first.Code != http.StatusSeeOther {
		t.Fatalf("propuesta válida rechazada: %d %s", first.Code, first.Body.String())
	}
	replay := performRequest(app, http.MethodPost, "http://127.0.0.1:8787/proposal", form, true)
	if replay.Code != http.StatusSeeOther {
		t.Fatalf("repetición idempotente rechazada: %d %s", replay.Code, replay.Body.String())
	}
	proposals, err := app.store.list()
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 || proposals[0].ActorRef != app.actorRef || proposals[0].ProjectRef != app.projectRef {
		t.Fatalf("historial HTTP incorrecto: %#v", proposals)
	}
	inventoryAfter, err := os.ReadFile(app.inventory.path)
	if err != nil || !reflect.DeepEqual(inventoryBefore, inventoryAfter) {
		t.Fatalf("la propuesta modificó el inventario: error=%v", err)
	}
	filtered := performRequest(
		app,
		http.MethodGet,
		"http://127.0.0.1:8787/?disposition="+string(dispositionStudy),
		nil,
		true,
	)
	if !strings.Contains(filtered.Body.String(), item.ID) ||
		!strings.Contains(filtered.Body.String(), app.catalog.disposition(dispositionStudy)) {
		t.Fatal("el filtro por propuesta no muestra el elemento conservado")
	}
	detail := performRequest(
		app,
		http.MethodGet,
		"http://127.0.0.1:8787/item?id="+url.QueryEscape(item.ID),
		nil,
		true,
	)
	for _, expected := range []string{
		proposals[0].Reason,
		app.catalog.text("value_yes"),
		app.catalog.disposition(dispositionStudy),
	} {
		if !strings.Contains(detail.Body.String(), expected) {
			t.Errorf("el historial renderizado no contiene %q", expected)
		}
	}

	stale := validForm(app, item, 0, strings.Repeat("d", 32))
	conflict := performRequest(app, http.MethodPost, "http://127.0.0.1:8787/proposal", stale, true)
	if conflict.Code != http.StatusConflict ||
		!strings.Contains(conflict.Body.String(), app.catalog.text("error_revision_conflict")) {
		t.Fatalf("revisión caducada no rechazada: %d %s", conflict.Code, conflict.Body.String())
	}
}

func TestHTTPRejectsChangedInventoryBeforeWriting(t *testing.T) {
	app, item := testApplication(t)
	if err := os.WriteFile(app.inventory.path, []byte("{\"id\":\"cambiado\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	form := validForm(app, item, 0, strings.Repeat("e", 32))
	response := performRequest(app, http.MethodPost, "http://127.0.0.1:8787/proposal", form, true)
	if response.Code != http.StatusConflict ||
		!strings.Contains(response.Body.String(), app.catalog.text("error_stale_inventory")) {
		t.Fatalf("inventario cambiado no rechazado: %d %s", response.Code, response.Body.String())
	}
	proposals, err := app.store.list()
	if err != nil || len(proposals) != 0 {
		t.Fatalf("se escribió pese al cambio de inventario: propuestas=%#v error=%v", proposals, err)
	}
}

func TestCatalogUsesSpanishAsDefaultAndFallback(t *testing.T) {
	spanish := newCatalog("es")
	fallback := newCatalog("fr-FR")
	if !reflect.DeepEqual(spanish.values, fallback.values) {
		t.Fatal("el catálogo de reserva no conserva paridad con español")
	}
	if spanish.text("app_title") == "" || spanish.text("clave_ausente") != spanish.text("missing_text") {
		t.Fatal("el catálogo no aplica el texto de reserva")
	}
	expression := regexp.MustCompile(`text "([^"]+)"`)
	keys := 0
	for _, parsed := range pageTemplate.Templates() {
		for _, match := range expression.FindAllStringSubmatch(parsed.Tree.Root.String(), -1) {
			keys++
			if spanish.text(match[1]) == spanish.text("missing_text") {
				t.Errorf("la plantilla usa una clave sin catálogo: %s", match[1])
			}
		}
	}
	if keys == 0 {
		t.Fatal("la prueba no encontró claves de catálogo en las plantillas")
	}
}

func cloneForm(source url.Values) url.Values {
	result := url.Values{}
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}
