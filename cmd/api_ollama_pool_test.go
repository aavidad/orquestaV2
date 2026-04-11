package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"orquesta/capacidadapp"
	"orquesta/conectoresapp"
	"orquesta/db"
	"orquesta/sesionesapp"
)

func asegurarConectorPoolLocalCmdTest(t *testing.T) {
	t.Helper()
	if _, err := conectoresService.SaveConnector(conectoresapp.SaveConnectorInput{
		Slug:         "ollama_pool_local",
		Nombre:       "Ollama Pool Local",
		Transporte:   "api",
		Comando:      "ollama",
		MetadataJSON: `{"familia":"ollama","reanudable":false,"pool_compartido":true,"default_task_profile":"implementacion","default_reasoning_effort":"high"}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("asegurar conector ollama_pool_local: %v", err)
	}
}

func asegurarProyectoOrquestadorCmdTest(t *testing.T) {
	t.Helper()
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("asegurar proyecto orquestador: %v", err)
	}
}

func TestAPIOllamaPoolCycle(t *testing.T) {
	prepararDBTemporalCmd(t)
	asegurarProyectoOrquestadorCmdTest(t)
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente gemma1: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("ruta ollama inesperada: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "PATCH: listo",
			},
		})
	}))
	defer ollama.Close()
	ollamaPoolManager = newTestOllamaPoolManager(ollama.URL)
	if _, err := capacidadService.AsegurarPoolLocalCompartido(capacidadapp.EntradaAsegurarPoolLocalCompartido{
		PoolSlug:         "ollama-gemma4",
		ModeloPreferente: "gemma4:26b",
		SlotsMaximos:     1,
	}); err != nil {
		t.Fatalf("asegurar pool local: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	post := func(path string, body []byte, want int) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return payload
	}

	launch := post("/api/runtime/ollama-pool/launch", []byte(`{"agente":"Gemma1","proyecto":"orquestador","plan":{"modelo":"gemma4:26b","perfil_tarea":"implementacion","razonamiento":"high"}}`), http.StatusOK)
	handleRef := launch["handle_ref"].(string)
	if handleRef == "" {
		t.Fatal("faltaba handle_ref")
	}
	sesionActiva, err := sesionesAPIService.GetActiveSession("Gemma1", "orquestador")
	if err != nil {
		t.Fatalf("sesion activa launch: %v", err)
	}
	if sesionActiva == nil || sesionActiva.ExternalSessionID == "" {
		t.Fatalf("faltaba sesion activa persistida: %+v", sesionActiva)
	}
	if got := sesionActiva.ResumePayloadJSON; got == "" || !bytes.Contains([]byte(got), []byte(`"modelo":"gemma4:26b"`)) || !bytes.Contains([]byte(got), []byte(`"driver":"ollama_pool_local"`)) {
		t.Fatalf("faltaba resume payload canónico: %+v", sesionActiva)
	}
	input := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"Implementa la funcion"}`), http.StatusOK)
	if input["respuesta"] != "PATCH: listo" {
		t.Fatalf("respuesta inesperada: %+v", input)
	}
	sesionActiva, err = sesionesAPIService.GetActiveSession("Gemma1", "orquestador")
	if err != nil {
		t.Fatalf("sesion activa input: %v", err)
	}
	if sesionActiva == nil || sesionActiva.ResumenContinuidad == "" {
		t.Fatalf("faltaba resumen continuidad persistido: %+v", sesionActiva)
	}
	if got := sesionActiva.ResumePayloadJSON; got == "" || !bytes.Contains([]byte(got), []byte(`"perfil_tarea":"implementacion"`)) {
		t.Fatalf("faltaba perfil persistido tras input: %+v", sesionActiva)
	}
	status := post("/api/runtime/ollama-pool/status", []byte(`{"handle_ref":"`+handleRef+`"}`), http.StatusOK)
	if status["logical_state"] != "ready" {
		t.Fatalf("status inesperado: %+v", status)
	}
	detalle, err := capacidadService.DescribirPoolLocalCompartido("ollama-gemma4")
	if err != nil {
		t.Fatalf("DescribirPoolLocalCompartido: %v", err)
	}
	if detalle == nil || detalle.Telemetria == nil || detalle.Telemetria.SlotsActivos != 1 || detalle.Telemetria.SesionesReady != 1 {
		t.Fatalf("telemetria de pool inesperada: %+v", detalle)
	}
	_ = post("/api/runtime/ollama-pool/stop", []byte(`{"handle_ref":"`+handleRef+`"}`), http.StatusOK)
	status = post("/api/runtime/ollama-pool/status", []byte(`{"handle_ref":"`+handleRef+`"}`), http.StatusOK)
	if status["logical_state"] != "stopped" {
		t.Fatalf("status detenido inesperado: %+v", status)
	}
	sesionActiva, err = sesionesAPIService.GetActiveSession("Gemma1", "orquestador")
	if err != nil {
		t.Fatalf("sesion activa stop: %v", err)
	}
	if sesionActiva == nil || sesionActiva.Estado != "cerrada" {
		t.Fatalf("estado sesion activa stop inesperado: %+v", sesionActiva)
	}
}

func TestAPIOllamaPoolLaunchAceptaLaunchPlanCanonicoCompleto(t *testing.T) {
	prepararDBTemporalCmd(t)
	asegurarProyectoOrquestadorCmdTest(t)
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente gemma1: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("ruta ollama inesperada: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "PATCH: listo",
			},
		})
	}))
	defer ollama.Close()
	ollamaPoolManager = newTestOllamaPoolManager(ollama.URL)
	if _, err := capacidadService.AsegurarPoolLocalCompartido(capacidadapp.EntradaAsegurarPoolLocalCompartido{
		PoolSlug:         "ollama-gemma4",
		ModeloPreferente: "gemma4:26b",
		SlotsMaximos:     1,
	}); err != nil {
		t.Fatalf("asegurar pool local: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/ollama-pool/launch", bytes.NewReader([]byte(`{
		"agente":"Gemma1",
		"proyecto":"orquestador",
		"plan":{
			"driver":"ollama_pool_local",
			"transporte":"api",
			"modo":"launch",
			"working_dir":"/home/alberto/Trabajo/orquesta",
			"modelo":"gemma4:26b",
			"perfil_tarea":"implementacion",
			"razonamiento":"high",
			"bootstrap_prompt":"microtarea cerrada"
		}
	}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("launch canónico status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIOllamaPoolLaunchFallaSinSlots(t *testing.T) {
	prepararDBTemporalCmd(t)
	asegurarProyectoOrquestadorCmdTest(t)
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente gemma1: %v", err)
	}
	if err := agentesService.RegisterAgent("Gemma2", "programador"); err != nil {
		t.Fatalf("registrar agente gemma2: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollamaPoolManager = newTestOllamaPoolManager("http://127.0.0.1:11434")
	if _, err := capacidadService.AsegurarPoolLocalCompartido(capacidadapp.EntradaAsegurarPoolLocalCompartido{
		PoolSlug:         "ollama-gemma4",
		ModeloPreferente: "gemma4:26b",
		SlotsMaximos:     1,
	}); err != nil {
		t.Fatalf("asegurar pool local: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/ollama-pool/launch", bytes.NewReader([]byte(`{"agente":"Gemma1","proyecto":"orquestador","plan":{"modelo":"gemma4:26b","perfil_tarea":"implementacion","razonamiento":"high"}}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("primer launch status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/runtime/ollama-pool/launch", bytes.NewReader([]byte(`{"agente":"Gemma2","proyecto":"orquestador","plan":{"modelo":"gemma4:26b","perfil_tarea":"implementacion","razonamiento":"high"}}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("segundo launch deberia fallar por slots, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIOllamaPoolLaunchHeredaResumenContinuidadPersistido(t *testing.T) {
	prepararDBTemporalCmd(t)
	asegurarProyectoOrquestadorCmdTest(t)
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente gemma1: %v", err)
	}
	if _, err := sesionesAPIService.StartContext(sesionesapp.StartContextInput{
		Agente:   "Gemma1",
		Proyecto: "orquestador",
		Conector: "ollama_pool_local",
		Resumen:  "Firma previa pendiente y helper ya extraido",
	}); err != nil {
		t.Fatalf("sembrar sesion previa: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })
	ollamaPoolManager = newTestOllamaPoolManager("http://127.0.0.1:11434")
	if _, err := capacidadService.AsegurarPoolLocalCompartido(capacidadapp.EntradaAsegurarPoolLocalCompartido{
		PoolSlug:         "ollama-gemma4",
		ModeloPreferente: "gemma4:26b",
		SlotsMaximos:     1,
	}); err != nil {
		t.Fatalf("asegurar pool local: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/ollama-pool/launch", bytes.NewReader([]byte(`{"agente":"Gemma1","proyecto":"orquestador","plan":{"modelo":"gemma4:26b","perfil_tarea":"implementacion","razonamiento":"high"}}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("launch status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode launch: %v", err)
	}
	handleRef, _ := payload["handle_ref"].(string)
	estado, err := ollamaPoolManager.Estado(handleRef)
	if err != nil {
		t.Fatalf("estado pool: %v", err)
	}
	if got := estado.ResumenContinuidad; got != "Firma previa pendiente y helper ya extraido" {
		t.Fatalf("resumen continuidad heredado inesperado: %q", got)
	}
}
