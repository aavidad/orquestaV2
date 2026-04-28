package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/tareasapp"
)

func TestResolverProyectoRepoServerUsaPrepareLiteTrasTimeout(t *testing.T) {
	prevTimeout := apiRuntimeProjectLookupTimeout
	prevPrimary := apiProjectLookupFn
	prevFallback := apiProjectLookupPrepareLiteFn
	t.Cleanup(func() {
		apiRuntimeProjectLookupTimeout = prevTimeout
		apiProjectLookupFn = prevPrimary
		apiProjectLookupPrepareLiteFn = prevFallback
	})

	apiRuntimeProjectLookupTimeout = 5 * time.Millisecond
	apiProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		time.Sleep(25 * time.Millisecond)
		return &db.Proyecto{ID: 1, Slug: ref, RutaAbs: "/slow", Tipo: db.ProyectoRepo}, nil
	}
	apiProjectLookupPrepareLiteFn = func(ref string) (*db.Proyecto, error) {
		return &db.Proyecto{ID: 7, Slug: ref, RutaAbs: "/fast", Tipo: db.ProyectoRepo}, nil
	}

	proyecto, materializado, err := resolverProyectoRepoServer("demo", "", "", "", "")
	if err != nil {
		t.Fatalf("resolverProyectoRepoServer: %v", err)
	}
	if materializado != nil {
		t.Fatalf("materializado inesperado: %+v", materializado)
	}
	if proyecto == nil || proyecto.RutaAbs != "/fast" {
		t.Fatalf("fallback prepare-lite no aplicado: %+v", proyecto)
	}
}

func TestAPIReposMaterializarReturns400WhenPathAndGitAreBothSet(t *testing.T) {
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/materializar", bytes.NewReader([]byte(`{"path":"/tmp/repo","git":"https://example.invalid/repo.git"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("usa exactamente uno de path o git")) {
		t.Fatalf("respuesta 400 sin detalle esperado: %s", rec.Body.String())
	}
}

func TestAPIReposMaterializarReturns500OnInjectedInternalError(t *testing.T) {
	prev := apiRepoAddServerFn
	t.Cleanup(func() {
		apiRepoAddServerFn = prev
	})

	apiRepoAddServerFn = func(localPath, remoteURL, branch, destino string) (*repoAddResult, error) {
		return nil, errors.New("injected internal failure")
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/materializar", bytes.NewReader([]byte(`{"path":"/tmp/repo"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("respuesta 500 sin clave error: %+v", payload)
	}
}

func TestAPIReposRevisarReturns400OnResolverBadRequest(t *testing.T) {
	prev := apiRepoResolverFn
	t.Cleanup(func() {
		apiRepoResolverFn = prev
	})

	apiRepoResolverFn = func(proyecto, localPath, remoteURL, branch, destino string) (*db.Proyecto, *repoAddResult, error) {
		return nil, nil, repoBadRequestf("usa proyecto o bien path/git, pero no ambos")
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/revisar", bytes.NewReader([]byte(`{"proyecto":"demo","path":"/tmp/repo","plan":true}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIReposRevisarReturns500OnInjectedPlanError(t *testing.T) {
	prevResolver := apiRepoResolverFn
	prevPlan := apiRepoPlanFn
	t.Cleanup(func() {
		apiRepoResolverFn = prevResolver
		apiRepoPlanFn = prevPlan
	})

	apiRepoResolverFn = func(proyecto, localPath, remoteURL, branch, destino string) (*db.Proyecto, *repoAddResult, error) {
		return &db.Proyecto{ID: 7, Slug: "demo", Tipo: db.ProyectoRepo, RutaAbs: "/tmp/repo"}, nil, nil
	}
	apiRepoPlanFn = func(proyecto string) (*capacidadapp.PasoPipelineLocalDeterminista, error) {
		return nil, errors.New("planner exploded")
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/revisar", bytes.NewReader([]byte(`{"proyecto":"demo","plan":true}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIReposRevisarReturns500WhenPlanReturnsNilStep(t *testing.T) {
	prevResolver := apiRepoResolverFn
	prevPlan := apiRepoPlanFn
	t.Cleanup(func() {
		apiRepoResolverFn = prevResolver
		apiRepoPlanFn = prevPlan
	})

	apiRepoResolverFn = func(proyecto, localPath, remoteURL, branch, destino string) (*db.Proyecto, *repoAddResult, error) {
		return &db.Proyecto{ID: 7, Slug: "demo", Tipo: db.ProyectoRepo, RutaAbs: "/tmp/repo"}, nil, nil
	}
	apiRepoPlanFn = func(proyecto string) (*capacidadapp.PasoPipelineLocalDeterminista, error) {
		return nil, nil
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/revisar", bytes.NewReader([]byte(`{"proyecto":"demo","plan":true}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIReposMejorarReturns400WhenTituloMissing(t *testing.T) {
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/mejorar", bytes.NewReader([]byte(`{"proyecto":"demo"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIReposMejorarReturns500OnInjectedCreateError(t *testing.T) {
	prevResolver := apiRepoResolverFn
	prevCreate := apiRepoTaskCreateFn
	t.Cleanup(func() {
		apiRepoResolverFn = prevResolver
		apiRepoTaskCreateFn = prevCreate
	})

	apiRepoResolverFn = func(proyecto, localPath, remoteURL, branch, destino string) (*db.Proyecto, *repoAddResult, error) {
		return &db.Proyecto{ID: 7, Slug: "demo", Tipo: db.ProyectoRepo, RutaAbs: "/tmp/repo"}, nil, nil
	}
	apiRepoTaskCreateFn = func(input tareasapp.CreateTaskInput) (int64, error) {
		return 0, errors.New("task create exploded")
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/mejorar", bytes.NewReader([]byte(`{"proyecto":"demo","titulo":"Mejora demo"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIReposMejorarReturns500WhenTaskReloadReturnsNil(t *testing.T) {
	prevResolver := apiRepoResolverFn
	prevCreate := apiRepoTaskCreateFn
	prevGet := apiRepoTaskGetFn
	t.Cleanup(func() {
		apiRepoResolverFn = prevResolver
		apiRepoTaskCreateFn = prevCreate
		apiRepoTaskGetFn = prevGet
	})

	apiRepoResolverFn = func(proyecto, localPath, remoteURL, branch, destino string) (*db.Proyecto, *repoAddResult, error) {
		return &db.Proyecto{ID: 7, Slug: "demo", Tipo: db.ProyectoRepo, RutaAbs: "/tmp/repo"}, nil, nil
	}
	apiRepoTaskCreateFn = func(input tareasapp.CreateTaskInput) (int64, error) {
		return 99, nil
	}
	apiRepoTaskGetFn = func(id int64) (*db.Tarea, error) {
		return nil, nil
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/repos/mejorar", bytes.NewReader([]byte(`{"proyecto":"demo","titulo":"Mejora demo","despachar":false}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}
