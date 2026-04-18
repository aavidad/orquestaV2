package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestAPIAgenteHandoff(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		CWD:         tmp,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("sesion origen: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex2",
		CWD:         tmp,
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("sesion destino: %v", err)
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

	body, err := json.Marshal(apiRuntimeHandoffRequest{
		AgenteOrigen:      "Codex1",
		AgenteDestino:     "Codex2",
		TareaID:           tareaID,
		Motivo:            "prueba api",
		Resumen:           "continua desde resumen",
		ExternalSessionID: "sess-api",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agente/handoff", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	apiHandlerAgenteHandoff(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal resp: %v", err)
	}
	if _, ok := resp["id"]; !ok {
		t.Fatalf("respuesta sin id: %+v", resp)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex2" {
		t.Fatalf("agente reasignado inesperado: %+v", tarea.Agente)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "handoff" {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
}

func TestAPIAgenteControl(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}

	body, err := json.Marshal(apiAgenteControlRequest{
		Agente:       "Codex1",
		Proyecto:     "orquestador",
		Accion:       "arrancar",
		Conector:     "codex-cli",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		Perfil:       "implementacion",
		Motivo:       "arranque de prueba",
		Por:          "test",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agente/control", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	apiHandlerAgenteControl(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"conector":"codex-cli"`) {
		t.Fatalf("payload inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestAPIAgenteControlReturns503OnTimeout(t *testing.T) {
	prevEnqueueFunc := agentControlEnqueueFunc
	prevTimeout := agentControlEnqueueTimeout
	defer func() {
		agentControlEnqueueFunc = prevEnqueueFunc
		agentControlEnqueueTimeout = prevTimeout
	}()
	agentControlEnqueueTimeout = 20 * time.Millisecond
	block := make(chan struct{})
	agentControlEnqueueFunc = func(req apiAgenteControlRequest) (int64, string, error) {
		<-block
		return 0, "", nil
	}

	body, err := json.Marshal(apiAgenteControlRequest{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Accion:   "arrancar",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/control", bytes.NewReader(body))
	apiHandlerAgenteControl(rec, req)
	close(block)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
}
