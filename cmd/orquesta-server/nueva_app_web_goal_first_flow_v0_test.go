package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestServerNuevaAppHTMLGoalFirstPOSTRenderizaYObservaV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")

	backend := &goalFirstHTTPBackendForTestV0{}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/nueva-app?locale=es", nil))
	if get.Code != http.StatusOK ||
		!strings.Contains(get.Body.String(), `<form method="post" action="/nueva-app" novalidate>`) ||
		!strings.Contains(get.Body.String(), `/api/v0/apps/director/goal/observe`) {
		t.Fatalf("GET /nueva-app inesperado status=%d body=%s", get.Code, get.Body.String())
	}

	post := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", strings.NewReader(nuevaAppGoalFirstFormValuesForTestV0().Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html")
	handler.ServeHTTP(post, req)
	body := post.Body.String()
	if post.Code != http.StatusOK {
		t.Fatalf("POST /nueva-app status=%d body=%s", post.Code, body)
	}
	if !strings.Contains(post.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type=%q", post.Header().Get("Content-Type"))
	}
	if backend.packet.GoalRef == "" || backend.packet.Objective == "" {
		t.Fatalf("goal packet incompleto: %+v\n%s", backend.packet, body)
	}
	runRef := nuevaAppHTMLDataAttrForTestV0(t, body, "data-run-ref")
	goalRef := nuevaAppHTMLDataAttrForTestV0(t, body, "data-goal-ref")
	if runRef == "" || goalRef != backend.packet.GoalRef {
		t.Fatalf("refs html run=%q goal=%q packet=%+v\n%s", runRef, goalRef, backend.packet, body)
	}
	for _, want := range []string{
		"Director arrancado",
		"Director y goal",
		`data-goal-panel`,
		`data-goal-auto-poll="true"`,
		runRef,
		backend.packet.GoalRef,
		"thread-ref-http-goal-first-001",
		orquestagoal.GoalStatusRunningV0,
		`/api/v0/apps/director/goal/observe`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("POST HTML no contiene %q\n%s", want, body)
		}
	}
	if strings.Contains(body, `legacy_director_loop</code>`) {
		t.Fatalf("POST HTML reintroduce modo legacy como resultado normal\n%s", body)
	}

	observed := postGoalFirstObserveForTestV0(t, handler, runRef)
	if observed.GoalRef != backend.packet.GoalRef ||
		observed.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		observed.RunStatus != "cerrada" ||
		observed.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		!observed.ClosureAccepted {
		t.Fatalf("observed=%+v packet=%+v", observed, backend.packet)
	}
}

func nuevaAppHTMLDataAttrForTestV0(t *testing.T, body string, attr string) string {
	t.Helper()
	pattern := regexp.MustCompile(regexp.QuoteMeta(attr) + `="([^"]+)"`)
	match := pattern.FindStringSubmatch(body)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func nuevaAppGoalFirstFormValuesForTestV0() url.Values {
	values := url.Values{}
	values.Set("request_id", "request-ref-server-nueva-app-goal-first-001")
	values.Set("locale", "es")
	values.Set("nombre", "Alquileres cercanos")
	values.Set("objetivo", "Mostrar pisos en alquiler cercanos en una app movil con mapa y filtros.")
	values.Set("descripcion", "El usuario quiere ver alquileres cercanos y guardar favoritos.")
	values.Set("tipo_app", "mobile")
	values.Add("plataformas", "ios")
	values.Add("plataformas", "android")
	values.Set("integraciones.0.tipo", "maps")
	values.Set("integraciones.0.nombre", "mapas")
	values.Set("integraciones.0.proposito", "Mostrar viviendas cercanas en mapa.")
	values.Set("integraciones.0.requerido", "true")
	values.Set("preferencias_tecnicas.arquitectura", "")
	values.Set("preferencias_tecnicas.lenguaje", "go")
	values.Set("datos.db_required", "true")
	values.Set("datos.necesidad_funcional", "Guardar viviendas, favoritos y busquedas.")
	values.Add("datos.tipos_datos", "pisos")
	values.Add("datos.tipos_datos", "usuarios")
	values.Set("datos.tipos_detallados.0.nombre", "vivienda")
	values.Set("datos.tipos_detallados.0.proposito", "Ficha de piso en alquiler.")
	values.Set("datos.tipos_detallados.0.sensibilidad", "publica")
	values.Set("datos.storage.0.tipo", "relacional")
	values.Set("datos.storage.0.proposito", "Consultas filtradas por zona y precio.")
	values.Set("datos.storage.0.requerido", "true")
	values.Set("calidad.pruebas", "media")
	values.Add("calidad.accesibilidad_opciones", "normal")
	values.Add("calidad.accesibilidad_opciones", "wcag_aa")
	values.Set("calidad.observabilidad", "true")
	values.Set("agentes.autonomia", "media")
	return values
}
