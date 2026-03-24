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

func TestAPIProyectoDetalleDevuelveMemoria(t *testing.T) {
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
	var proyectoID int64
	if err := db.DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	if _, err := db.GuardarDecisionProyecto(&db.DecisionProyecto{
		ProyectoID: proyectoID,
		Titulo:     "Arquitectura base",
		Solucion:   "Puerto de almacenamiento",
	}); err != nil {
		t.Fatalf("GuardarDecisionProyecto: %v", err)
	}
	if _, err := db.GuardarDocumentoExterno(&db.DocumentoExterno{
		ProyectoID: proyectoID,
		Titulo:     "ADR",
		RutaRef:    "/tmp/adr.md",
		Resumen:    "Resumen",
	}); err != nil {
		t.Fatalf("GuardarDocumentoExterno: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador", nil)
	rec := httptest.NewRecorder()
	webRouterAPIProyectos(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Proyecto   db.Proyecto           `json:"proyecto"`
		Decisiones []db.DecisionProyecto `json:"decisiones"`
		Documentos []db.DocumentoExterno `json:"documentos"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if payload.Proyecto.Slug != "orquestador" {
		t.Fatalf("slug inesperado: %s", payload.Proyecto.Slug)
	}
	if len(payload.Decisiones) != 1 || len(payload.Documentos) != 1 {
		t.Fatalf("payload incompleto: %s", rec.Body.String())
	}
}

func TestAPIProyectoDecisionCrear(t *testing.T) {
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
	if _, err := db.DB.Exec(`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1)`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo"); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	body := bytes.NewBufferString(`{"titulo":"BD","solucion":"Puerto","motivo":"Soportar SQLite y MySQL"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquestador/decisiones", body)
	rec := httptest.NewRecorder()
	webRouterAPIProyectos(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d cuerpo=%s", rec.Code, rec.Body.String())
	}
	items, err := memoriaProyectoService.ListDecisions("orquestador")
	if err != nil {
		t.Fatalf("ListDecisions: %v", err)
	}
	if len(items) != 1 || items[0].Titulo != "BD" {
		t.Fatalf("decision no persistida: %+v", items)
	}
}
