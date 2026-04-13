package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/microprogramacionapp"
)

func TestAPIMicroprogramacionEspecificacionesCRUD(t *testing.T) {
	prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil || proyectoID <= 0 {
		t.Fatalf("upsert proyecto: id=%d err=%v", proyectoID, err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	crearBody := []byte(`{
		"tarea_id": 505,
		"proyecto": "orquestador",
		"titulo": "runtimeagente/driver.go::BuildSpec",
		"archivo_objetivo": "runtimeagente/driver.go",
		"simbolo_objetivo": "BuildSpec",
		"descripcion": "Crear la especificacion del siguiente slice",
		"tests_obligatorios": ["go test ./runtimeagente -run TestBuildSpec"],
		"write_set": ["runtimeagente/driver.go", "runtimeagente/driver_test.go"],
		"dependencias_permitidas": ["orquesta/runtimesapp"],
		"dependencias_prohibidas": ["orquesta/db"]
	}`)

	crearReq := httptest.NewRequest(http.MethodPost, "/api/microprogramacion/especificaciones", bytes.NewReader(crearBody))
	crearReq.Header.Set("Content-Type", "application/json")
	crearRec := httptest.NewRecorder()
	mux.ServeHTTP(crearRec, crearReq)
	if crearRec.Code != http.StatusCreated {
		t.Fatalf("crear status=%d body=%s", crearRec.Code, crearRec.Body.String())
	}
	var crearResp apiEspecificacionFuncionCreateResponse
	if err := json.Unmarshal(crearRec.Body.Bytes(), &crearResp); err != nil {
		t.Fatalf("decode crear: %v", err)
	}
	if crearResp.ID <= 0 || crearResp.Especificacion == nil {
		t.Fatalf("respuesta crear inesperada: %+v", crearResp)
	}
	if crearResp.Especificacion.ArchivoObjetivo != "runtimeagente/driver.go" {
		t.Fatalf("archivo objetivo inesperado: %+v", crearResp.Especificacion)
	}
	if crearResp.Especificacion.ProyectoID == nil || *crearResp.Especificacion.ProyectoID != proyectoID {
		t.Fatalf("proyecto inesperado: %+v", crearResp.Especificacion)
	}

	listarReq := httptest.NewRequest(http.MethodGet, "/api/microprogramacion/especificaciones?tarea_id=505&proyecto=orquestador", nil)
	listarRec := httptest.NewRecorder()
	mux.ServeHTTP(listarRec, listarReq)
	if listarRec.Code != http.StatusOK {
		t.Fatalf("listar status=%d body=%s", listarRec.Code, listarRec.Body.String())
	}
	var listarResp apiEspecificacionesFuncionResponse
	if err := json.Unmarshal(listarRec.Body.Bytes(), &listarResp); err != nil {
		t.Fatalf("decode listar: %v", err)
	}
	if len(listarResp.Especificaciones) != 1 {
		t.Fatalf("especificaciones inesperadas: %+v", listarResp.Especificaciones)
	}

	verReq := httptest.NewRequest(http.MethodGet, "/api/microprogramacion/especificaciones/"+itoa(crearResp.ID), nil)
	verRec := httptest.NewRecorder()
	mux.ServeHTTP(verRec, verReq)
	if verRec.Code != http.StatusOK {
		t.Fatalf("ver status=%d body=%s", verRec.Code, verRec.Body.String())
	}
	var verResp apiEspecificacionFuncionResponse
	if err := json.Unmarshal(verRec.Body.Bytes(), &verResp); err != nil {
		t.Fatalf("decode ver: %v", err)
	}
	if verResp.Especificacion == nil || verResp.Especificacion.ID != crearResp.ID {
		t.Fatalf("detalle inesperado: %+v", verResp.Especificacion)
	}
}

func TestAPIMicroprogramacionEspecificacionesValidaEntrada(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/microprogramacion/especificaciones", bytes.NewBufferString(`{"archivo_objetivo":"/tmp/x.go"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIMicroprogramacionEmitirDevuelveMicrotarea(t *testing.T) {
	prepararDBTemporalCmd(t)

	id, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		Titulo:            "EmitirSlice",
		ArchivoObjetivo:   "cmd/api.go",
		SimboloObjetivo:   "apiHandlerX",
		Descripcion:       "Implementar solo el siguiente slice",
		TestsObligatorios: []string{"go test ./cmd -run TestAPIX -count=1"},
		WriteSet:          []string{"cmd/api.go", "cmd/api_test.go"},
		CreadoPor:         "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body := []byte(`{"contexto":"No toques nada fuera del write-set."}`)
	req := httptest.NewRequest(http.MethodPost, "/api/microprogramacion/especificaciones/"+itoa(id)+"/emitir", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("emitir status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiMicrotareaEmitidaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode emitir: %v", err)
	}
	if resp.Microtarea == nil || resp.Microtarea.EspecificacionID != id {
		t.Fatalf("microtarea inesperada: %+v", resp.Microtarea)
	}
	if !strings.Contains(resp.Microtarea.Mensaje, "No toques nada fuera del write-set.") {
		t.Fatalf("mensaje sin contexto esperado: %s", resp.Microtarea.Mensaje)
	}
}

func TestAPIMicroprogramacionDespacharCreaRuntimeOrder(t *testing.T) {
	prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	id, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		Titulo:            "EmitirSlice",
		ProyectoID:        &proyectoID,
		ArchivoObjetivo:   "cmd/api.go",
		SimboloObjetivo:   "apiHandlerX",
		Descripcion:       "Implementar solo el siguiente slice",
		TestsObligatorios: []string{"go test ./cmd -run TestAPIX -count=1"},
		WriteSet:          []string{"cmd/api.go", "cmd/api_test.go"},
		CreadoPor:         "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body := []byte(`{"agente":"Gemma1","contexto":"No toques nada fuera del write-set."}`)
	req := httptest.NewRequest(http.MethodPost, "/api/microprogramacion/especificaciones/"+itoa(id)+"/despachar", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("despachar status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiMicrotareaDespachadaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode despachar: %v", err)
	}
	if resp.Despacho == nil || resp.Despacho.AgenteDestino != "Gemma1" || resp.Despacho.RuntimeOrderID <= 0 {
		t.Fatalf("despacho inesperado: %+v", resp.Despacho)
	}
	orden, err := runtimesService.GetRuntimeOrder(resp.Despacho.RuntimeOrderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	if orden == nil || orden.Tipo != "send_instruction" {
		t.Fatalf("runtime order inesperada: %+v", orden)
	}
	if !strings.Contains(orden.PayloadJSON, `"source":"microprogramacion"`) || !strings.Contains(orden.PayloadJSON, `"especificacion_id":`) {
		t.Fatalf("payload sin trazabilidad microprogramada: %s", orden.PayloadJSON)
	}
	if !strings.Contains(rec.Body.String(), `"runtime_order_id":`) || strings.Contains(rec.Body.String(), `"RuntimeOrderID"`) {
		t.Fatalf("respuesta API sin contrato snake_case: %s", rec.Body.String())
	}
}

func TestAPIMicroprogramacionValidarEntregaDevuelveHallazgos(t *testing.T) {
	prepararDBTemporalCmd(t)

	id, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		Titulo:                 "ValidarEntregaSlice",
		ArchivoObjetivo:        "cmd/api.go",
		SimboloObjetivo:        "apiHandlerY",
		Descripcion:            "Implementar solo el handler Y",
		DependenciasPermitidas: []string{"orquesta/runtimesapp"},
		DependenciasProhibidas: []string{"orquesta/db"},
		TestsObligatorios:      []string{"go test ./cmd -run TestAPIY -count=1"},
		WriteSet:               []string{"cmd/api.go", "cmd/api_test.go"},
		CreadoPor:              "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body := []byte(`{
		"simbolo_entregado":"apiHandlerZ",
		"write_set_entregado":["cmd/api.go","db/agentes.go"],
		"dependencias_usadas":["orquesta/db"],
		"tests_ejecutados":["go test ./cmd -run TestOtra -count=1"],
		"tests_fallidos":["go test ./cmd -run TestAPIY -count=1"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/microprogramacion/especificaciones/"+itoa(id)+"/validar-entrega", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validar status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiValidacionEntregaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode validar: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.Valida {
		t.Fatalf("resultado inesperado: %+v", resp.Resultado)
	}
	if len(resp.Resultado.Hallazgos) == 0 {
		t.Fatalf("se esperaban hallazgos de validacion")
	}
}
