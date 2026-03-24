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

func TestAPIGobernanzaReglaCrearYListar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	body := bytes.NewBufferString(`{"tipo_agente":"programador","categoria":"arquitectura","titulo":"Puerto","descripcion":"No acoplar","activa":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/gobernanza/reglas", body)
	rec := httptest.NewRecorder()
	webRouterAPIGobernanza(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d cuerpo=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/gobernanza/reglas?tipo_agente=programador", nil)
	rec = httptest.NewRecorder()
	webRouterAPIGobernanza(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado listando: %d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Items []db.Regla `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatalf("sin reglas en payload")
	}
}

func TestAPIGobernanzaWorkflowCrear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	body := bytes.NewBufferString(`{"tipo_agente":"programador","nombre":"workflow-api","descripcion":"flujo","pasos":["uno","dos"],"activo":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/gobernanza/workflows", body)
	rec := httptest.NewRecorder()
	webRouterAPIGobernanza(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d cuerpo=%s", rec.Code, rec.Body.String())
	}

	items, err := gobernanzaService.ListWorkflows("programador", nil)
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if len(items) == 0 || items[len(items)-1].Nombre != "workflow-api" {
		t.Fatalf("workflow no persistido: %+v", items)
	}
}
