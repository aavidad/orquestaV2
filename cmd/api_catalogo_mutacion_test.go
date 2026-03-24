package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	assertKey(http.MethodPost, "/api/reglas", []byte(`{"actor":"Codex1","tipo_agente":"programador","categoria":"calidad","titulo":"No romper tests","descripcion":"Mantener verde"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/skills", []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"rg","descripcion":"busqueda rapida","cuando_usar":"buscar texto"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/workflows", []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"inicio-sesion","descripcion":"flujo base","pasos_json":"[\"leer\",\"votar\"]"}`), "id", http.StatusCreated)
	assertKey(http.MethodPost, "/api/permisos-catalogo", []byte(`{"actor":"alberto","entidad":"reglas","rol":"programador","alcance":"mismo_rol","puede_crear":true,"puede_editar":true,"puede_activar":true,"puede_versionar":true}`), "ok", http.StatusOK)
}
