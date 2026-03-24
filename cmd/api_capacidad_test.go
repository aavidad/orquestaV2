package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	assertKey(http.MethodPost, "/api/pools", []byte(`{"slug":"claude","proveedor":"Anthropic","runtime":"claude","plan":"default","es_de_pago":true,"capacidad_total":1,"capacidad_reservada":0,"permite_hijos":true,"permite_modelos_multi":true,"permite_sobrecoste":false,"politica_handoff":"preventivo","fuente_telemetria":"manual","metadata_json":"{}","activo":true}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/pools/claude/modelos", []byte(`{"model_slug":"claude-opus","activo":true,"prioridad":20,"coste_relativo":2.0,"limite_conocido_json":"{}"}`), "id", http.StatusCreated)
	assertKey(http.MethodGet, "/api/politicas-modelo?scope_tipo=perfil&scope_ref=programador", nil, "politicas", http.StatusOK)
	assertKey(http.MethodPost, "/api/politicas-modelo", []byte(`{"scope_tipo":"perfil","scope_ref":"documentador","perfil_tarea":"documentador","pool_slug":"codex","model_slug":"gpt-5.4","reasoning_effort":"medium","prioridad":20,"activa":true,"metadata_json":"{}"}`), "id", http.StatusCreated)
	assertKey(http.MethodGet, "/api/modelo/resolver?perfil=programador", nil, "resolucion", http.StatusOK)
}
