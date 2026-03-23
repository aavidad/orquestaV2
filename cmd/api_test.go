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
	var resp apiSesionInicioResponse
	if err := json.Unmarshal(recSesion.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode sesion inicio: %v", err)
	}
	if resp.Sesion == nil || resp.Sesion.Agente != "codex1" {
		t.Fatalf("respuesta de sesion inesperada: %+v", resp.Sesion)
	}
	if resp.Rol == "" {
		t.Fatalf("se esperaba rol en la respuesta de sesion inicio")
	}
	if len(resp.Reglas) == 0 {
		t.Fatalf("se esperaban reglas en la respuesta de sesion inicio")
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

func TestAPIProyectoDescubrirYGestionAgentes(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(filepath.Join(proyectoDir, ".git"), 0o755); err != nil {
		t.Fatalf("crear proyecto demo: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	descubrirBody, _ := json.Marshal(apiProyectoDescubrirRequest{Ruta: tmp})
	recDescubrir := httptest.NewRecorder()
	reqDescubrir := httptest.NewRequest(http.MethodPost, "/api/proyectos/descubrir", bytes.NewReader(descubrirBody))
	mux.ServeHTTP(recDescubrir, reqDescubrir)
	if recDescubrir.Code != http.StatusCreated {
		t.Fatalf("status descubrir inesperado: %d body=%s", recDescubrir.Code, recDescubrir.Body.String())
	}

	recProyecto := httptest.NewRecorder()
	reqProyecto := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo", nil)
	mux.ServeHTTP(recProyecto, reqProyecto)
	if recProyecto.Code != http.StatusOK {
		t.Fatalf("status proyecto inesperado: %d body=%s", recProyecto.Code, recProyecto.Body.String())
	}

	agenteBody, _ := json.Marshal(apiAgenteRequest{Nombre: "temporal", Rol: "programador"})
	recAlta := httptest.NewRecorder()
	reqAlta := httptest.NewRequest(http.MethodPost, "/api/agentes", bytes.NewReader(agenteBody))
	mux.ServeHTTP(recAlta, reqAlta)
	if recAlta.Code != http.StatusCreated {
		t.Fatalf("status alta agente inesperado: %d body=%s", recAlta.Code, recAlta.Body.String())
	}

	recRetirar := httptest.NewRecorder()
	reqRetirar := httptest.NewRequest(http.MethodPost, "/api/agentes/temporal/retirar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recRetirar, reqRetirar)
	if recRetirar.Code != http.StatusOK {
		t.Fatalf("status retirar inesperado: %d body=%s", recRetirar.Code, recRetirar.Body.String())
	}

	recRehabilitar := httptest.NewRecorder()
	reqRehabilitar := httptest.NewRequest(http.MethodPost, "/api/agentes/temporal/rehabilitar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recRehabilitar, reqRehabilitar)
	if recRehabilitar.Code != http.StatusOK {
		t.Fatalf("status rehabilitar inesperado: %d body=%s", recRehabilitar.Code, recRehabilitar.Body.String())
	}
}

func TestAPIAgenteControlPlaneAcciones(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if _, err := db.IniciarSesion("Codex1"); err != nil {
		t.Fatalf("iniciar sesion origen: %v", err)
	}
	if _, err := db.IniciarSesion("Codex2"); err != nil {
		t.Fatalf("iniciar sesion destino: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Handoff API",
		Modulo:    "orquestador",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	pausaBody, _ := json.Marshal(apiAgentePausarRequest{Agente: "Codex1", Minutos: 15, Motivo: "rate limit"})
	recPausa := httptest.NewRecorder()
	reqPausa := httptest.NewRequest(http.MethodPost, "/api/agente/pausar", bytes.NewReader(pausaBody))
	mux.ServeHTTP(recPausa, reqPausa)
	if recPausa.Code != http.StatusOK {
		t.Fatalf("status pausar inesperado: %d body=%s", recPausa.Code, recPausa.Body.String())
	}

	recReset := httptest.NewRecorder()
	reqReset := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex1/reset-reanimacion", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recReset, reqReset)
	if recReset.Code != http.StatusOK {
		t.Fatalf("status reset inesperado: %d body=%s", recReset.Code, recReset.Body.String())
	}

	handoffBody, _ := json.Marshal(apiRuntimeHandoffRequest{
		AgenteOrigen:  "Codex1",
		AgenteDestino: "Codex2",
		TareaID:       tareaID,
		Motivo:        "cambio de turno",
		Resumen:       "seguir desde api",
	})
	recHandoff := httptest.NewRecorder()
	reqHandoff := httptest.NewRequest(http.MethodPost, "/api/agente/handoff", bytes.NewReader(handoffBody))
	mux.ServeHTTP(recHandoff, reqHandoff)
	if recHandoff.Code != http.StatusCreated {
		t.Fatalf("status handoff inesperado: %d body=%s", recHandoff.Code, recHandoff.Body.String())
	}

	var resp struct {
		ID      int64 `json:"id"`
		OrderID int64 `json:"order_id"`
	}
	if err := json.Unmarshal(recHandoff.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode handoff: %v", err)
	}
	if resp.OrderID == 0 && resp.ID == 0 {
		t.Fatalf("order id inesperado: %d", resp.OrderID)
	}

	recEliminar := httptest.NewRecorder()
	reqEliminar := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex3/eliminar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recEliminar, reqEliminar)
	if recEliminar.Code != http.StatusOK {
		t.Fatalf("status eliminar inesperado: %d body=%s", recEliminar.Code, recEliminar.Body.String())
	}
}

func TestAPIAgenteTickAutoPausaPorAgotamiento(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-tick-auto-pausa",
		ResumenContinuidad: "tick final",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(map[string]any{
		"agente":     "Codex1",
		"proyecto":   "orquestador",
		"cuota_pct":  3,
		"finalizado": true,
		"motivo":     "rate limit de proveedor",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/tick", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status tick inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("estado_cuota inesperado: %s", agente.EstadoCuota)
	}
	if agente.ReanimarAt == nil {
		t.Fatalf("reanimar_at no deberia ser nil")
	}
	if agente.MotivoPausa == "" || agente.MotivoPausa != "Auto-pausa por agotamiento: rate limit de proveedor" {
		t.Fatalf("motivo_pausa inesperado: %q", agente.MotivoPausa)
	}
}
