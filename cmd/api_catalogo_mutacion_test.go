package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestAPICatalogoMutaciones(t *testing.T) {
	prepararDBTemporalCmd(t)

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

	createID := func(path string, body []byte) int64 {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status inesperado POST %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		var payload apiCatalogoMutationResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode json POST %s: %v", path, err)
		}
		if payload.ID <= 0 {
			t.Fatalf("POST %s sin id valido: %s", path, rec.Body.String())
		}
		return payload.ID
	}

	reglaID := createID("/api/reglas", []byte(`{"actor":"Codex1","tipo_agente":"programador","categoria":"calidad","titulo":"No romper tests","descripcion":"Mantener verde"}`))
	assertKey(http.MethodGet, "/api/reglas/"+strconv.FormatInt(reglaID, 10), nil, "regla", http.StatusOK)
	assertKey(http.MethodPost, "/api/reglas/"+strconv.FormatInt(reglaID, 10), []byte(`{"actor":"Codex1","tipo_agente":"programador","categoria":"calidad","titulo":"No romper tests nunca","descripcion":"Mantener verde","activa":true}`), "id", http.StatusOK)
	assertKey(http.MethodPost, "/api/reglas/"+strconv.FormatInt(reglaID, 10)+"/activa", []byte(`{"actor":"Codex1","activa":false}`), "ok", http.StatusOK)
	assertKey(http.MethodGet, "/api/reglas/"+strconv.FormatInt(reglaID, 10)+"/versiones", nil, "versiones", http.StatusOK)

	skillID := createID("/api/skills", []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-rg","descripcion":"busqueda rapida","cuando_usar":"buscar texto"}`))
	assertKey(http.MethodGet, "/api/skills/"+strconv.FormatInt(skillID, 10), nil, "skill", http.StatusOK)
	assertKey(http.MethodPost, "/api/skills/"+strconv.FormatInt(skillID, 10), []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-rg","descripcion":"busqueda rapida","cuando_usar":"buscar texto","activa":true}`), "id", http.StatusOK)
	assertKey(http.MethodPost, "/api/skills/"+strconv.FormatInt(skillID, 10)+"/activa", []byte(`{"actor":"Codex1","activa":false}`), "ok", http.StatusOK)
	assertKey(http.MethodGet, "/api/skills/"+strconv.FormatInt(skillID, 10)+"/versiones", nil, "versiones", http.StatusOK)

	workflowID := createID("/api/workflows", []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-inicio-sesion","descripcion":"flujo base","pasos_json":"[\"leer\",\"votar\"]"}`))
	assertKey(http.MethodGet, "/api/workflows/"+strconv.FormatInt(workflowID, 10), nil, "workflow", http.StatusOK)
	assertKey(http.MethodPost, "/api/workflows/"+strconv.FormatInt(workflowID, 10), []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-inicio-sesion","descripcion":"flujo base","pasos_json":"[\"leer\",\"votar\",\"programar\"]","activo":true}`), "id", http.StatusOK)
	assertKey(http.MethodPost, "/api/workflows/"+strconv.FormatInt(workflowID, 10)+"/activa", []byte(`{"actor":"Codex1","activa":false}`), "ok", http.StatusOK)
	assertKey(http.MethodGet, "/api/workflows/"+strconv.FormatInt(workflowID, 10)+"/versiones", nil, "versiones", http.StatusOK)
	assertKey(http.MethodPost, "/api/permisos-catalogo", []byte(`{"actor":"alberto","entidad":"reglas","rol":"programador","alcance":"mismo_rol","puede_crear":true,"puede_editar":true,"puede_activar":true,"puede_versionar":true}`), "ok", http.StatusOK)
}
