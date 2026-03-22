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
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestWebRuntimesPageYFiltros(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	// Runtime activo.
	sesionActiva, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-web-001",
		ResumenContinuidad: "runtime activo",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}
	runtimeActivo, err := db.GetRuntimeBySesionID(sesionActiva.ID)
	if err != nil || runtimeActivo == nil {
		t.Fatalf("get runtime activo: %v", err)
	}

	// Runtime cerrado.
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex2",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-web-002",
		ResumenContinuidad: "runtime cerrado",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion cerrada: %v", err)
	}
	if err := db.FinSesion("Codex2"); err != nil {
		t.Fatalf("cerrar sesion codex2: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/tareas", webHandlerTareas)
	mux.HandleFunc("/tareas/", webRouterTareas)
	mux.HandleFunc("/propuestas", webHandlerPropuestas)
	mux.HandleFunc("/propuestas/", webRouterPropuestas)
	mux.HandleFunc("/runtimes", webHandlerRuntimes)
	mux.HandleFunc("/runtimes/", webRouterRuntimes)
	registerAPIRoutes(mux)

	assertContains := func(path, needle string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), needle) {
			t.Fatalf("respuesta %s no contiene %q: %s", path, needle, rec.Body.String())
		}
	}

	assertContains("/runtimes", "Runtimes")
	assertContains("/runtimes", "Codex1")
	assertContains("/runtimes?activos=true", "Codex1")
	assertContains("/runtimes?activos=false", "Codex2")
	assertContains("/runtimes/"+itoa(runtimeActivo.ID), "Runtime #")
}
