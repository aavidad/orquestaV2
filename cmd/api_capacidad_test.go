package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"orquesta/db"
)

func TestAPICapacidadEndpoints(t *testing.T) {
	prepararDBTemporalCmd(t)

	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:                "codex",
		Proveedor:           "OpenAI",
		Runtime:             "codex",
		Plan:                "default",
		EsDePago:            true,
		CapacidadTotal:      4,
		CapacidadReservada:  1,
		PermiteHijos:        true,
		PermiteModelosMulti: true,
		PermiteSobrecoste:   false,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        "{}",
		Activo:              true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := db.GuardarPoolModelo("codex", &db.PoolModelo{
		ModelSlug:          "gpt-5.4",
		Activo:             true,
		Prioridad:          10,
		CosteRelativo:      1.5,
		LimiteConocidoJSON: "{}",
	}); err != nil {
		t.Fatalf("guardar pool modelo: %v", err)
	}
	if _, err := db.GuardarPoliticaModelo(&db.PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "programador",
		PerfilTarea:     "programador",
		PoolSlug:        "codex",
		ModelSlug:       "gpt-5.4",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
		MetadataJSON:    "{}",
	}); err != nil {
		t.Fatalf("guardar politica modelo: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	assertKey := func(method, path string, body []byte, key string, wantCode int) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		mux.ServeHTTP(rec, req)
		if rec.Code != wantCode {
			t.Fatalf("status inesperado %s %s: %d body=%s", method, path, rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode json %s %s: %v", method, path, err)
		}
		if _, ok := payload[key]; !ok {
			t.Fatalf("respuesta %s %s sin clave %q: %s", method, path, key, rec.Body.String())
		}
	}

	assertKey(http.MethodGet, "/api/pools", nil, "pools", http.StatusOK)
	assertKey(http.MethodGet, "/api/pools/codex", nil, "detalle", http.StatusOK)
	assertKey(http.MethodGet, "/api/pools/codex/modelos", nil, "modelos", http.StatusOK)
	assertKey(http.MethodPost, "/api/pools/local-compartido", []byte(`{"pool_slug":"ollama-gemma4","proveedor":"Ollama","runtime":"ollama","modelo_preferente":"gemma4:26b","slots_maximos":1,"conector_canonico":"ollama_pool_local","conector_compatibilidad":"ollama-cli","experimental_compat":true}`), "pool_local", http.StatusCreated)
	assertKey(http.MethodGet, "/api/pools/ollama-gemma4/local", nil, "pool_local", http.StatusOK)
	assertKey(http.MethodPost, "/api/pools", []byte(`{"slug":"claude","proveedor":"Anthropic","runtime":"claude","plan":"default","es_de_pago":true,"capacidad_total":1,"capacidad_reservada":0,"permite_hijos":true,"permite_modelos_multi":true,"permite_sobrecoste":false,"politica_handoff":"preventivo","fuente_telemetria":"manual","metadata_json":"{}","activo":true}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/pools/claude/modelos", []byte(`{"model_slug":"claude-opus","activo":true,"prioridad":20,"coste_relativo":2.0,"limite_conocido_json":"{}"}`), "id", http.StatusCreated)
	assertKey(http.MethodGet, "/api/politicas-modelo?scope_tipo=perfil&scope_ref=programador", nil, "politicas", http.StatusOK)
	assertKey(http.MethodPost, "/api/politicas-modelo", []byte(`{"scope_tipo":"perfil","scope_ref":"documentador","perfil_tarea":"documentador","pool_slug":"codex","model_slug":"gpt-5.4","reasoning_effort":"medium","prioridad":20,"activa":true,"metadata_json":"{}"}`), "id", http.StatusCreated)
	assertKey(http.MethodGet, "/api/modelo/resolver?perfil=programador", nil, "resolucion", http.StatusOK)
	assertKey(http.MethodGet, "/api/modelo/pipeline-local?proyecto=orquestador", nil, "pipeline", http.StatusOK)
	assertKey(http.MethodGet, "/api/modelo/pipeline-local/paso?proyecto=orquestador", nil, "paso", http.StatusOK)
	assertKey(http.MethodPost, "/api/modelo/pipeline-local/ejecutar?proyecto=orquestador", []byte(`{}`), "resultado", http.StatusOK)
	assertKey(http.MethodPost, "/api/modelo/pipeline-local/despachar?proyecto=orquestador", []byte(`{}`), "resultado", http.StatusOK)
	assertKey(http.MethodGet, "/api/modelo/runtime-activos", nil, "modelos", http.StatusOK)
	assertKey(http.MethodPost, "/api/modelo/runtime-descargar", []byte(`{}`), "descargados", http.StatusOK)
}

func TestAPIPipelineLocalEjecutarIncluyeDespacho(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: wd,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: proyecto=%+v err=%v", proyecto, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Preparar backlog",
		Estado:     db.TareaLibre,
		Prioridad:  db.PrioridadAlta,
		ProyectoID: &proyecto.ID,
	})
	if err != nil {
		t.Fatalf("registrar tarea: %v", err)
	}
	if tareaID <= 0 {
		t.Fatalf("id tarea invalido: %d", tareaID)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/modelo/pipeline-local/ejecutar?proyecto=orquestador", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Resultado struct {
			Despacho map[string]any `json:"despacho"`
		} `json:"resultado"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Resultado.Despacho == nil {
		t.Fatalf("respuesta sin despacho: %s", rec.Body.String())
	}
	if payload.Resultado.Despacho["carril"] != "premium_worktree" {
		t.Fatalf("carril inesperado: %+v", payload.Resultado.Despacho)
	}
	if payload.Resultado.Despacho["entrega_canonica"] != "git_worktree" {
		t.Fatalf("entrega inesperada: %+v", payload.Resultado.Despacho)
	}
}

func TestAPIPipelineLocalDespacharIncluyeEstadoRuntime(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: wd,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: proyecto=%+v err=%v", proyecto, err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Preparar backlog",
		Estado:     db.TareaLibre,
		Prioridad:  db.PrioridadAlta,
		ProyectoID: &proyecto.ID,
	}); err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/modelo/pipeline-local/despachar?proyecto=orquestador", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Resultado struct {
			DispatchRuntime map[string]any `json:"dispatch_runtime"`
		} `json:"resultado"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Resultado.DispatchRuntime == nil {
		t.Fatalf("respuesta sin dispatch_runtime: %s", rec.Body.String())
	}
	if payload.Resultado.DispatchRuntime["estado"] == nil {
		t.Fatalf("dispatch runtime sin estado: %s", rec.Body.String())
	}
}
