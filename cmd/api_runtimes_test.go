/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestAPIRuntimesReadOnly(t *testing.T) {
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-runtime-api",
		ResumenContinuidad: "api runtime",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtime == nil {
		t.Fatalf("runtime nil")
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	assertRuntimeOK := func(path string, key string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode json %s: %v", path, err)
		}
		if _, ok := body[key]; !ok {
			t.Fatalf("respuesta %s sin clave %q: %s", path, key, rec.Body.String())
		}
	}

	assertRuntimeOK("/api/runtimes", "runtimes")
	assertRuntimeOK("/api/runtimes?proyecto=orquestador&activos=true", "runtimes")
	assertRuntimeOK("/api/runtimes/tree?proyecto=orquestador", "runtimes")
	assertRuntimeOK("/api/runtimes/"+itoa(runtime.ID), "runtime")
}
