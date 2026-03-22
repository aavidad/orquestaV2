/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"orquesta/db"
)

func prepararDBTemporalCmd(t *testing.T) string {
	t.Helper()

	anteriorDB := os.Getenv("ORQUESTA_DB")
	anteriorRoot := os.Getenv("ORQUESTA_WORKSPACE_ROOT")
	t.Cleanup(func() {
		db.Close()
		db.DB = nil
		if anteriorDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", anteriorDB)
		}
		if anteriorRoot == "" {
			_ = os.Unsetenv("ORQUESTA_WORKSPACE_ROOT")
		} else {
			_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", anteriorRoot)
		}
	})

	db.Close()
	db.DB = nil

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-api-test.db")
	if err := os.Setenv("ORQUESTA_DB", dbPath); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_WORKSPACE_ROOT", tmp); err != nil {
		t.Fatalf("setenv ORQUESTA_WORKSPACE_ROOT: %v", err)
	}
	if err := db.Open(); err != nil {
		t.Fatalf("open db temporal: %v", err)
	}
	return tmp
}

func TestAPIAgentesListaJSON(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); len(got) < 16 || got[:16] != "application/json" {
		t.Fatalf("content-type inesperado: %s", got)
	}
}

func TestAPIAsignacionActivarYSesionInicio(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.GetProyecto("autofirmav2"); err != nil {
		t.Fatalf("get proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	asignacionBody, _ := json.Marshal(apiAsignacionActivarRequest{
		Agente:   "codex1",
		Proyecto: "autofirmav2",
		Nota:     "slot 1",
	})
	recAsignacion := httptest.NewRecorder()
	reqAsignacion := httptest.NewRequest(http.MethodPost, "/api/asignaciones/activar", bytes.NewReader(asignacionBody))
	mux.ServeHTTP(recAsignacion, reqAsignacion)
	if recAsignacion.Code != http.StatusOK {
		t.Fatalf("status asignacion inesperado: %d body=%s", recAsignacion.Code, recAsignacion.Body.String())
	}

	sesionBody, _ := json.Marshal(apiSesionInicioRequest{
		Agente:            "codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-cli",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID: "sess-001",
		Resumen:           "continuar desde API",
	})
	recSesion := httptest.NewRecorder()
	reqSesion := httptest.NewRequest(http.MethodPost, "/api/sesiones/inicio", bytes.NewReader(sesionBody))
	mux.ServeHTTP(recSesion, reqSesion)
	if recSesion.Code != http.StatusCreated {
		t.Fatalf("status sesion inesperado: %d body=%s", recSesion.Code, recSesion.Body.String())
	}

	sesion, err := db.GetSesionActiva("codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	if sesion.ExternalSessionID != "sess-001" {
		t.Fatalf("external session id inesperado: %s", sesion.ExternalSessionID)
	}
}

func TestAPIPropuestaReabrirYRepararVotos(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &db.Propuesta{
		Titulo:       "Propuesta API",
		Descripcion:  "Reabrir y reparar desde API",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if _, err := db.DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor2"); err != nil {
		t.Fatalf("delete voto faltante: %v", err)
	}
	if err := db.CerrarPropuesta(propuesta.Codigo, string(db.PropuestaBacklog), "alberto"); err != nil {
		t.Fatalf("cerrar propuesta: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	reabrirBody, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion: "reabrir",
		Agente: "alberto",
	})
	recReabrir := httptest.NewRecorder()
	reqReabrir := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(reabrirBody))
	mux.ServeHTTP(recReabrir, reqReabrir)
	if recReabrir.Code != http.StatusOK {
		t.Fatalf("status reabrir inesperado: %d body=%s", recReabrir.Code, recReabrir.Body.String())
	}

	reabierta, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta reabierta: %v", err)
	}
	if reabierta.Estado != db.PropuestaAbierta {
		t.Fatalf("estado inesperado tras reabrir: %s", reabierta.Estado)
	}

	if _, err := db.DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1"); err != nil {
		t.Fatalf("delete segundo voto faltante: %v", err)
	}

	repararBody, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion: "reparar_votos",
		Agente: "alberto",
	})
	recReparar := httptest.NewRecorder()
	reqReparar := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(repararBody))
	mux.ServeHTTP(recReparar, reqReparar)
	if recReparar.Code != http.StatusOK {
		t.Fatalf("status reparar inesperado: %d body=%s", recReparar.Code, recReparar.Body.String())
	}

	var total int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1").Scan(&total); err != nil {
		t.Fatalf("contar voto reparado: %v", err)
	}
	if total != 1 {
		t.Fatalf("conteo inesperado tras reparar via API: %d", total)
	}
}
