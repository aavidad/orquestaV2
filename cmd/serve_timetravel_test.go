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

func TestWebTimeTravelPageYDetalle(t *testing.T) {
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
	if _, err := db.UpsertEntidadMemoria(&db.EntidadMemoria{
		Nombre:        "Core_API",
		Tipo:          string(db.EntidadMemoriaAPI),
		ValorJSON:     `{"version":"v2"}`,
		MetadataJSON:  `{"fuente":"test"}`,
		VerificadoPor: "Codex1",
		ProyectoID:    &proyectoID,
	}); err != nil {
		t.Fatalf("upsert entidad memoria: %v", err)
	}

	cp1ID, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "checkpoint",
		Resumen:        "analisis inicial",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"step":1}`,
		ResumeStrategy: "resume-thread",
		Source:         "manual:test",
	})
	if err != nil {
		t.Fatalf("crear checkpoint 1: %v", err)
	}
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex2",
		ProyectoID:     &proyectoID,
		CheckpointKind: "handoff",
		Resumen:        "handoff a Codex2",
		Branch:         "feature/handoff",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"from":"Codex1"}`,
		ResumeStrategy: "take-over",
		Source:         "runtime_order:1",
	}); err != nil {
		t.Fatalf("crear checkpoint 2: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/time-travel", webHandlerTimeTravel)
	mux.HandleFunc("/time-travel/", webRouterTimeTravel)

	assertContains := func(path string, needles ...string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		for _, needle := range needles {
			if !strings.Contains(body, needle) {
				t.Fatalf("respuesta %s no contiene %q: %s", path, needle, body)
			}
		}
	}

	assertNotContains := func(path string, needle string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), needle) {
			t.Fatalf("respuesta %s contiene inesperadamente %q: %s", path, needle, rec.Body.String())
		}
	}

	assertContains("/time-travel", "Time Travel", "Codex1", "handoff")
	assertContains("/time-travel?agente=Codex1", "Codex1")
	assertNotContains("/time-travel?agente=Codex1", "handoff a Codex2")
	assertContains("/time-travel?kind=handoff", "Codex2")
	assertNotContains("/time-travel?kind=handoff", "analisis inicial")
	assertContains("/time-travel/"+itoa(cp1ID), "Checkpoint #", "analisis inicial", "Core_API", "step", "version")
}

func TestWebDashboardMuestraCheckpointsRecientes(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
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
	cpID, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "checkpoint",
		Resumen:        "preparar rollback seguro",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"step":"checkpoint"}`,
		ResumeStrategy: "resume-thread",
		Source:         "manual:dashboard",
	})
	if err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	webHandlerDash(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado dashboard: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, needle := range []string{"Checkpoints recientes", "preparar rollback seguro", "/time-travel/" + itoa(cpID)} {
		if !strings.Contains(body, needle) {
			t.Fatalf("dashboard sin %q: %s", needle, body)
		}
	}
}
