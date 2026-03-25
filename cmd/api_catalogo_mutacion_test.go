package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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
	assertKey(http.MethodPost, "/api/skills/"+strconv.FormatInt(skillID, 10), []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-rg","descripcion":"busqueda rapida","cuando_usar":"buscar texto","escenario":"investigacion","prioridad":10,"aliases_json":"[\"ripgrep\"]","herramientas_json":"[\"rg\"]","activa":true}`), "id", http.StatusOK)
	assertKey(http.MethodPost, "/api/skills/"+strconv.FormatInt(skillID, 10)+"/activa", []byte(`{"actor":"Codex1","activa":false}`), "ok", http.StatusOK)
	assertKey(http.MethodGet, "/api/skills/"+strconv.FormatInt(skillID, 10)+"/versiones", nil, "versiones", http.StatusOK)

	workflowID := createID("/api/workflows", []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-inicio-sesion","descripcion":"flujo base","pasos_json":"[\"leer\",\"votar\"]"}`))
	assertKey(http.MethodGet, "/api/workflows/"+strconv.FormatInt(workflowID, 10), nil, "workflow", http.StatusOK)
	assertKey(http.MethodPost, "/api/workflows/"+strconv.FormatInt(workflowID, 10), []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-inicio-sesion","descripcion":"flujo base","pasos_json":"[\"leer\",\"votar\",\"programar\"]","activo":true}`), "id", http.StatusOK)
	assertKey(http.MethodPost, "/api/workflows/"+strconv.FormatInt(workflowID, 10)+"/activa", []byte(`{"actor":"Codex1","activa":false}`), "ok", http.StatusOK)
	assertKey(http.MethodGet, "/api/workflows/"+strconv.FormatInt(workflowID, 10)+"/versiones", nil, "versiones", http.StatusOK)
	assertKey(http.MethodPost, "/api/permisos-catalogo", []byte(`{"actor":"alberto","entidad":"reglas","rol":"programador","alcance":"mismo_rol","puede_crear":true,"puede_editar":true,"puede_activar":true,"puede_versionar":true}`), "ok", http.StatusOK)
}

func TestAPISkillRechazaDuplicadoEquivalente(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	bodyBase := []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"catalogo-rg","descripcion":"busqueda rapida","cuando_usar":"buscar texto","escenario":"investigacion","prioridad":10,"aliases_json":"[\"ripgrep\"]","herramientas_json":"[\"rg\"]"}`)
	recBase := httptest.NewRecorder()
	reqBase := httptest.NewRequest(http.MethodPost, "/api/skills", bytes.NewReader(bodyBase))
	reqBase.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recBase, reqBase)
	if recBase.Code != http.StatusCreated {
		t.Fatalf("status crear base inesperado: %d body=%s", recBase.Code, recBase.Body.String())
	}

	bodyDup := []byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"ripgrep","descripcion":"duplicada","cuando_usar":"buscar texto","escenario":"investigacion","prioridad":20,"herramientas_json":"[\"rg\"]"}`)
	recDup := httptest.NewRecorder()
	reqDup := httptest.NewRequest(http.MethodPost, "/api/skills", bytes.NewReader(bodyDup))
	reqDup.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recDup, reqDup)
	if recDup.Code != http.StatusBadRequest {
		t.Fatalf("status duplicado inesperado: %d body=%s", recDup.Code, recDup.Body.String())
	}
}

func TestAPISkillExternaQuedaPendienteYSoloAdminLaActiva(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body := []byte(`{"actor":"Codex2","tipo_agente":"programador","nombre":"external-linter","descripcion":"linter externo","cuando_usar":"validar dependencias","escenario":"qa","prioridad":30,"herramientas_json":"[\"vendor-lint\"]","origen":"third_party"}`)
	recCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/skills", bytes.NewReader(body))
	reqCreate.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("status crear skill externa inesperado: %d body=%s", recCreate.Code, recCreate.Body.String())
	}

	var created apiCatalogoMutationResponse
	if err := json.Unmarshal(recCreate.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode crear skill externa: %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("id invalido para skill externa: %s", recCreate.Body.String())
	}

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/skills/"+strconv.FormatInt(created.ID, 10), nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("status get skill externa inesperado: %d body=%s", recGet.Code, recGet.Body.String())
	}
	var resp apiSkillResponse
	if err := json.Unmarshal(recGet.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode skill externa: %v", err)
	}
	if resp.Skill == nil || resp.Skill.Activa || !resp.Skill.RequiereAprobacion {
		t.Fatalf("estado inicial externo inesperado: %+v", resp.Skill)
	}

	recActivateUser := httptest.NewRecorder()
	reqActivateUser := httptest.NewRequest(http.MethodPost, "/api/skills/"+strconv.FormatInt(created.ID, 10)+"/activa", bytes.NewReader([]byte(`{"actor":"Codex2","activa":true}`)))
	reqActivateUser.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recActivateUser, reqActivateUser)
	if recActivateUser.Code != http.StatusBadRequest {
		t.Fatalf("status activar skill externa sin admin inesperado: %d body=%s", recActivateUser.Code, recActivateUser.Body.String())
	}

	recActivateAdmin := httptest.NewRecorder()
	reqActivateAdmin := httptest.NewRequest(http.MethodPost, "/api/skills/"+strconv.FormatInt(created.ID, 10)+"/activa", bytes.NewReader([]byte(`{"actor":"alberto","activa":true}`)))
	reqActivateAdmin.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recActivateAdmin, reqActivateAdmin)
	if recActivateAdmin.Code != http.StatusOK {
		t.Fatalf("status activar skill externa como admin inesperado: %d body=%s", recActivateAdmin.Code, recActivateAdmin.Body.String())
	}
}

func TestAPISkillDetectaCarenciaYPreparaInvocacion(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recBase := httptest.NewRecorder()
	reqBase := httptest.NewRequest(http.MethodPost, "/api/skills", bytes.NewReader([]byte(`{"actor":"Codex1","tipo_agente":"programador","nombre":"gofmt","descripcion":"formateo","cuando_usar":"formatear codigo go","escenario":"codigo","herramientas_json":"[\"gofmt\"]"}`)))
	reqBase.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recBase, reqBase)
	if recBase.Code != http.StatusCreated {
		t.Fatalf("status crear base inesperado: %d body=%s", recBase.Code, recBase.Body.String())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/detectar-carencia", bytes.NewReader([]byte(`{"tipo_agente":"programador","nombre":"goimports","descripcion":"ordenar imports y formatear go","cuando_usar":"corregir imports en codigo go","escenario":"codigo","herramientas_json":"[\"goimports\"]"}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status deteccion inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiSkillDeteccionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode deteccion: %v", err)
	}
	if resp.Resultado == nil || !resp.Resultado.Falta {
		t.Fatalf("resultado de deteccion inesperado: %+v", resp.Resultado)
	}
	if !strings.Contains(resp.Resultado.InvocacionCreador, "$skill-creator") {
		t.Fatalf("invocacion inesperada: %+v", resp.Resultado)
	}
}
