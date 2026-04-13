package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/capacidadapp"
	"orquesta/conectoresapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/microprogramacionapp"
	"orquesta/runtimeagente"
	"orquesta/runtimesapp"
	"orquesta/sesionesapp"
)

func runGitCmdTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, strings.TrimSpace(string(out)))
	}
}

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
		switch r.URL.Path {
		case "/api/chat":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]any{
					"role":    "assistant",
					"content": "PATCH: listo",
				},
			})
		case "/api/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"done": true})
		default:
			t.Fatalf("ruta ollama inesperada: %s", r.URL.Path)
		}
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
	if input["delivery_state"] != "notified" || input["delivery_async"] != true {
		t.Fatalf("respuesta inesperada: %+v", input)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	var transcript []*db.RuntimeTranscriptEntry
	for i := 0; i < 20; i++ {
		transcript, err = db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: apuntarString("Gemma1"), ProyectoID: &proyecto.ID, Limit: 10})
		if err != nil {
			t.Fatalf("listar transcript: %v", err)
		}
		if len(transcript) > 0 && strings.Contains(transcript[0].Text, "PATCH: listo") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(transcript) == 0 || !strings.Contains(transcript[0].Text, "PATCH: listo") {
		t.Fatalf("faltaba salida observada en transcript: %+v", transcript)
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
	metadata, _ := status["metadata"].(map[string]any)
	if got, ok := metadata["timeout_ms"].(float64); !ok || int64(got) != timeoutClienteOllamaLocalMS() {
		t.Fatalf("timeout_ms inesperado en status: %+v", metadata)
	}
	if got, _ := metadata["endpoint"].(string); got != "http://example.com" {
		t.Fatalf("endpoint inesperado en status: %+v", metadata)
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

func TestAPIOllamaPoolInputMarcaDispatchNotificadoSiRecibeRuntimeOrderID(t *testing.T) {
	prepararDBTemporalCmd(t)
	asegurarProyectoOrquestadorCmdTest(t)
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente gemma1: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]any{
					"role":    "assistant",
					"content": "PATCH: listo",
				},
			})
		case "/api/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"done": true})
		default:
			t.Fatalf("ruta ollama inesperada: %s", r.URL.Path)
		}
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
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Gemma1",
		ProyectoID:  &proyecto.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Gemma1","texto":"Implementa la funcion"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}

	resp := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"Implementa la funcion","runtime_order_id":`+strconv.FormatInt(orderID, 10)+`}`), http.StatusOK)
	if got, _ := resp["runtime_order_id"].(float64); int64(got) != orderID {
		t.Fatalf("runtime_order_id inesperado en respuesta: %+v", resp)
	}
	order, err := db.GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("estado inesperado tras notify: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"dispatch_state":"notified"`) || !strings.Contains(order.ResultadoJSON, `"delivery_state":"notified"`) {
		t.Fatalf("resultado sin dispatch notified: %s", order.ResultadoJSON)
	}
}

func apuntarString(v string) *string { return &v }

func TestAPIOllamaPoolInputMaterializaFicheros(t *testing.T) {
	prepararDBTemporalCmd(t)
	dirProyecto := t.TempDir()
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: dirProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("asegurar proyecto: %v", err)
	}
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	codigoGenerado := "// FILE: pkg/hola/hola.go\npackage hola\n\nfunc Hola() string { return \"hola\" }\n"

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{"role": "assistant", "content": codigoGenerado},
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
		req.Host = "example.com"
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

	especificacionID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "Materializar Hola",
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		Descripcion:       "Crear funcion Hola",
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		WriteSet:          []string{"pkg/hola/hola.go"},
		CreadoPor:         "test",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	if _, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "Implementa Hola",
		EspecificacionID:  especificacionID,
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		WriteSet:          []string{"pkg/hola/hola.go"},
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	}); err != nil {
		t.Fatalf("despachar microprogramacion: %v", err)
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: apuntarString("Gemma1"), ProyectoID: &projectID})
	if err != nil || len(orders) == 0 {
		t.Fatalf("listar ordenes tras despacho: %+v err=%v", orders, err)
	}
	orderID := orders[0].ID

	resp := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"Implementa hola"}`), http.StatusOK)
	if resp["delivery_state"] != "notified" || resp["delivery_async"] != true {
		t.Fatalf("respuesta asincrona inesperada: %+v", resp)
	}

	// Verificar que el fichero existe físicamente en el directorio del proyecto
	destino := filepath.Join(dirProyecto, "pkg", "hola", "hola.go")
	var contenido []byte
	for i := 0; i < 30; i++ {
		contenido, err = os.ReadFile(destino)
		if err == nil && strings.Contains(string(contenido), "func Hola()") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("fichero no escrito en disco: %v", err)
	}
	if !strings.Contains(string(contenido), "func Hola()") {
		t.Errorf("contenido del fichero incorrecto: %s", string(contenido))
	}
	var order *db.RuntimeOrder
	for i := 0; i < 30; i++ {
		order, err = runtimesService.GetRuntimeOrder(orderID)
		if err != nil || order == nil {
			t.Fatalf("get runtime order final: %+v err=%v", order, err)
		}
		if order.Estado == "completada" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if order.Estado != "completada" {
		t.Fatalf("la orden deberia quedar completada tras materializar: %+v", order)
	}
}

func TestAPIOllamaPoolInputNoMaterializaFueraDeWriteSet(t *testing.T) {
	prepararDBTemporalCmd(t)
	dirProyecto := t.TempDir()
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: dirProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("asegurar proyecto: %v", err)
	}
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	codigoGenerado := "// FILE: cmd/api.go\npackage cmd\n"
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{"role": "assistant", "content": codigoGenerado},
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
	especificacionID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "No tocar cmd/api.go",
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		Descripcion:       "Crear funcion Hola",
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		WriteSet:          []string{"pkg/hola/hola.go"},
		CreadoPor:         "test",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	if _, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "Implementa Hola",
		EspecificacionID:  especificacionID,
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		WriteSet:          []string{"pkg/hola/hola.go"},
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	}); err != nil {
		t.Fatalf("despachar microprogramacion: %v", err)
	}

	resp := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"Implementa hola"}`), http.StatusOK)
	materializados, _ := resp["ficheros_materializados"].([]any)
	if len(materializados) != 0 {
		t.Fatalf("no deberia materializar fuera del write_set: %+v", resp)
	}
	if _, err := os.Stat(filepath.Join(dirProyecto, "cmd", "api.go")); !os.IsNotExist(err) {
		t.Fatalf("se escribió un fichero fuera del write_set: %v", err)
	}
}

func TestAPIOllamaPoolInputNoReencolaCorreccionSiRespuestaEsSoloProsa(t *testing.T) {
	prepararDBTemporalCmd(t)
	dirProyecto := t.TempDir()
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: dirProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("asegurar proyecto: %v", err)
	}
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "He analizado el problema. Propongo extraer un helper y revisar el caso borde antes de tocar el fichero.",
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

	especificacionID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "Prosa sin correccion",
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		Descripcion:       "No debe autocorregir si la respuesta no intenta entregar codigo",
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		WriteSet:          []string{"pkg/hola/hola.go"},
		FormatoSalida:     "ficheros+evidencia",
		CreadoPor:         "test",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	if _, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "Implementa Hola",
		EspecificacionID:  especificacionID,
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		WriteSet:          []string{"pkg/hola/hola.go"},
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	}); err != nil {
		t.Fatalf("despachar microprogramacion: %v", err)
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: apuntarString("Gemma1"), ProyectoID: &projectID})
	if err != nil || len(orders) == 0 {
		t.Fatalf("listar ordenes tras despacho: %+v err=%v", orders, err)
	}
	orderID := orders[0].ID

	resp := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"Implementa hola","runtime_order_id":`+strconv.FormatInt(orderID, 10)+`}`), http.StatusOK)
	if resp["delivery_state"] != "notified" || resp["delivery_async"] != true {
		t.Fatalf("respuesta asincrona inesperada: %+v", resp)
	}

	var order *db.RuntimeOrder
	for i := 0; i < 30; i++ {
		order, err = runtimesService.GetRuntimeOrder(orderID)
		if err != nil || order == nil {
			t.Fatalf("get runtime order: %+v err=%v", order, err)
		}
		if strings.Contains(order.ResultadoJSON, `"dispatch_state":"notified"`) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(order.ResultadoJSON, `"dispatch_state":"notified"`) {
		t.Fatalf("faltaba dispatch_state notified: %+v", order)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("la orden no deberia cambiar de estado por prosa sin entrega: %+v", order)
	}

	for i := 0; i < 20; i++ {
		orders, err = runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: apuntarString("Gemma1"), ProyectoID: &projectID})
		if err != nil {
			t.Fatalf("listar ordenes finales: %v", err)
		}
		if len(orders) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(orders) != 1 {
		t.Fatalf("no deberia crear correccion automatica para prosa: %+v", orders)
	}
	if _, err := os.Stat(filepath.Join(dirProyecto, "pkg", "hola", "hola.go")); !os.IsNotExist(err) {
		t.Fatalf("no deberia materializar fichero con una respuesta en prosa: %v", err)
	}
}

func TestAPIOllamaPoolInputRehidrataSesionPersistidaTrasReinicio(t *testing.T) {
	prepararDBTemporalCmd(t)
	asegurarProyectoOrquestadorCmdTest(t)
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente gemma1: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "PATCH: rehidratada",
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
		req.Host = "example.com"
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

	ollamaPoolManager = newTestOllamaPoolManager(ollama.URL)

	input := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"Implementa la funcion"}`), http.StatusOK)
	if input["delivery_state"] != "notified" || input["delivery_async"] != true {
		t.Fatalf("respuesta inesperada tras rehidratacion: %+v", input)
	}
	var status map[string]any
	for i := 0; i < 30; i++ {
		status = post("/api/runtime/ollama-pool/status", []byte(`{"handle_ref":"`+handleRef+`"}`), http.StatusOK)
		if status["logical_state"] == "ready" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if status["logical_state"] != "ready" {
		t.Fatalf("status inesperado tras rehidratacion: %+v", status)
	}
}

func TestAPIOllamaPoolInputMaterializaContraEspecificacionCoincidentePorWriteSet(t *testing.T) {
	prepararDBTemporalCmd(t)
	dirProyecto := t.TempDir()
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: dirProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("asegurar proyecto: %v", err)
	}
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{"role": "assistant", "content": "// FILE: cabeceras/merge_headers.go\npackage cabeceras\n\nfunc FusionarCabecerasCanonicas(base, sobreescrituras map[string]string) map[string]string { return map[string]string{} }\nEVIDENCIA:\n- cambio\n"},
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

	especificacionVieja, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "Microtarea vieja",
		ArchivoObjetivo:   "tmp/ollama_smoke_eval/normalizar_identificador.go",
		SimboloObjetivo:   "NormalizarIdentificadorTecnico",
		Descripcion:       "Vieja",
		TestsObligatorios: []string{"go test ./tmp/ollama_smoke_eval -count=1"},
		WriteSet:          []string{"tmp/ollama_smoke_eval/normalizar_identificador.go"},
		CreadoPor:         "test",
	})
	if err != nil {
		t.Fatalf("crear especificacion vieja: %v", err)
	}
	especificacionNueva, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "Microtarea nueva",
		ArchivoObjetivo:   "cabeceras/merge_headers.go",
		SimboloObjetivo:   "FusionarCabecerasCanonicas",
		Descripcion:       "Nueva",
		TestsObligatorios: []string{"go test ./cabeceras -run TestFusionarCabecerasCanonicas -count=1"},
		WriteSet:          []string{"cabeceras/merge_headers.go"},
		FormatoSalida:     "ficheros+evidencia",
		CreadoPor:         "test",
	})
	if err != nil {
		t.Fatalf("crear especificacion nueva: %v", err)
	}

	if _, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "Vieja",
		EspecificacionID:  especificacionVieja,
		ArchivoObjetivo:   "tmp/ollama_smoke_eval/normalizar_identificador.go",
		SimboloObjetivo:   "NormalizarIdentificadorTecnico",
		WriteSet:          []string{"tmp/ollama_smoke_eval/normalizar_identificador.go"},
		TestsObligatorios: []string{"go test ./tmp/ollama_smoke_eval -count=1"},
		FormatoSalida:     "patch+evidencia",
	}); err != nil {
		t.Fatalf("despachar especificacion vieja: %v", err)
	}
	if _, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "Nueva",
		EspecificacionID:  especificacionNueva,
		ArchivoObjetivo:   "cabeceras/merge_headers.go",
		SimboloObjetivo:   "FusionarCabecerasCanonicas",
		WriteSet:          []string{"cabeceras/merge_headers.go"},
		TestsObligatorios: []string{"go test ./cabeceras -run TestFusionarCabecerasCanonicas -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	}); err != nil {
		t.Fatalf("despachar especificacion nueva: %v", err)
	}

	resp := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"Implementa FusionarCabecerasCanonicas"}`), http.StatusOK)
	if resp["delivery_state"] != "notified" || resp["delivery_async"] != true {
		t.Fatalf("respuesta asincrona inesperada: %+v", resp)
	}

	destino := filepath.Join(dirProyecto, "cabeceras", "merge_headers.go")
	var contenido []byte
	for i := 0; i < 30; i++ {
		contenido, err = os.ReadFile(destino)
		if err == nil && strings.Contains(string(contenido), "FusionarCabecerasCanonicas") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("fichero materializado inexistente: %v", err)
	}
	if !strings.Contains(string(contenido), "FusionarCabecerasCanonicas") {
		t.Fatalf("contenido inesperado: %s", string(contenido))
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

func TestAPIOllamaPoolInputRegistraEntregaGitDesdeWorktree(t *testing.T) {
	prepararDBTemporalCmd(t)
	repoDir := t.TempDir()
	runGitCmdTest(t, "", "init", "-b", "main", repoDir)
	runGitCmdTest(t, repoDir, "config", "user.name", "Orquesta Test")
	runGitCmdTest(t, repoDir, "config", "user.email", "orquesta@example.test")
	if err := os.MkdirAll(filepath.Join(repoDir, "pkg", "hola"), 0o755); err != nil {
		t.Fatalf("mkdir pkg/hola: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "pkg", "hola", "hola.go"), []byte("package hola\n\nfunc Hola() string { return \"base\" }\n"), 0o644); err != nil {
		t.Fatalf("write hola.go: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "base")

	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("asegurar proyecto: %v", err)
	}
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	prev := ollamaPoolManager
	t.Cleanup(func() { ollamaPoolManager = prev })

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{"role": "assistant", "content": "He modificado la worktree y ejecutado los tests."},
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

	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Gemma1",
		Reason:     "test_entrega_git",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "pkg", "hola", "hola.go"), []byte("package hola\n\nfunc Hola() string { return \"gemma\" }\n"), 0o644); err != nil {
		t.Fatalf("rewrite worktree hola.go: %v", err)
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

	especificacionID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "Entrega git hola",
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		Descripcion:       "Cambiar la funcion Hola usando entrega git",
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		WriteSet:          []string{"pkg/hola/hola.go"},
		FormatoSalida:     "git_worktree+evidencia",
		CreadoPor:         "test",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	if _, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "Modifica Hola en tu worktree",
		EspecificacionID:  especificacionID,
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		WriteSet:          []string{"pkg/hola/hola.go"},
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		FormatoSalida:     "git_worktree+evidencia",
	}); err != nil {
		t.Fatalf("despachar microprogramacion: %v", err)
	}
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: apuntarString("Gemma1"), ProyectoID: &projectID})
	if err != nil || len(orders) == 0 {
		t.Fatalf("listar ordenes: %+v err=%v", orders, err)
	}
	orderID := orders[0].ID

	resp := post("/api/runtime/ollama-pool/input", []byte(`{"handle_ref":"`+handleRef+`","texto":"He terminado"}`), http.StatusOK)
	if resp["delivery_state"] != "notified" || resp["delivery_async"] != true {
		t.Fatalf("respuesta asincrona inesperada: %+v", resp)
	}

	var order *db.RuntimeOrder
	for i := 0; i < 30; i++ {
		order, err = runtimesService.GetRuntimeOrder(orderID)
		if err != nil || order == nil {
			t.Fatalf("get runtime order: %+v err=%v", order, err)
		}
		if order.Estado == "completada" && strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if order.Estado != "completada" {
		t.Fatalf("la orden deberia quedar completada: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
		t.Fatalf("resultado sin receipt git: %s", order.ResultadoJSON)
	}
	merges, err := db.ListarGitMerges(&projectID, "pendiente")
	if err != nil || len(merges) == 0 {
		t.Fatalf("listar git merges: %+v err=%v", merges, err)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "main" {
		t.Fatalf("merge inesperado: %+v", merges[0])
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

func TestAPIOllamaPoolLaunchPublicaYPersisteWorktreeActiva(t *testing.T) {
	prepararDBTemporalCmd(t)
	repoDir := t.TempDir()
	runGitCmdTest(t, "", "init", "-b", "main", repoDir)
	runGitCmdTest(t, repoDir, "config", "user.name", "Orquesta Test")
	runGitCmdTest(t, repoDir, "config", "user.email", "orquesta@example.test")
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module demo\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "base")

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("asegurar proyecto: %v", err)
	}
	asegurarConectorPoolLocalCmdTest(t)
	if err := agentesService.RegisterAgent("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente gemma1: %v", err)
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
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Gemma1",
		Reason:     "test_launch_worktree",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
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
	metadata, _ := payload["metadata"].(map[string]any)
	if int(metadata["worktree_id"].(float64)) != int(worktree.ID) {
		t.Fatalf("metadata sin worktree_id esperado: %+v", metadata)
	}
	if metadata["ruta_worktree"] != worktree.Path {
		t.Fatalf("metadata sin ruta_worktree esperada: %+v", metadata)
	}
	if metadata["branch_worktree"] != worktree.Branch || metadata["base_ref_worktree"] != worktree.BaseRef {
		t.Fatalf("metadata worktree inesperada: %+v", metadata)
	}
	if metadata["mailbox_delivery_mode"] != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("metadata sin mailbox_delivery_mode interactivo: %+v", metadata)
	}
	if metadata["input_path"] != "/api/runtime/ollama-pool/input" || metadata["status_path"] != "/api/runtime/ollama-pool/status" || metadata["stop_path"] != "/api/runtime/ollama-pool/stop" {
		t.Fatalf("metadata sin rutas remotas canónicas: %+v", metadata)
	}

	activa, err := sesionesAPIService.GetActiveSession("Gemma1", "orquestador")
	if err != nil || activa == nil {
		t.Fatalf("sesion activa: %+v err=%v", activa, err)
	}
	envelope := db.ParseResumePayloadEnvelope(activa.ResumePayloadJSON)
	worktreePayload, _ := envelope["worktree"].(map[string]any)
	if int64(worktree.ID) != int64(worktreePayload["id"].(float64)) || worktree.Path != worktreePayload["ruta"] || worktree.Branch != worktreePayload["branch"] || worktree.BaseRef != worktreePayload["base_ref"] {
		t.Fatalf("resume_payload sin worktree persistida: %s", activa.ResumePayloadJSON)
	}
	if envelope["mailbox_delivery_mode"] != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("resume_payload sin mailbox_delivery_mode interactivo: %s", activa.ResumePayloadJSON)
	}
	if envelope["endpoint"] == "" || envelope["input_path"] != "/api/runtime/ollama-pool/input" || envelope["status_path"] != "/api/runtime/ollama-pool/status" || envelope["stop_path"] != "/api/runtime/ollama-pool/stop" {
		t.Fatalf("resume_payload sin rutas remotas canónicas: %s", activa.ResumePayloadJSON)
	}
}
