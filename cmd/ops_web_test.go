package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestAPIOpsListados(t *testing.T) {
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
	if _, err := db.DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?,?,?,?)`,
		"codex1", proyectoID, "activa", "principal"); err != nil {
		t.Fatalf("insert asignacion: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO sesiones (agente, activa, proyecto_id, estado, herramienta, host, branch) VALUES (?,?,?,?,?,?,?)`,
		"codex1", 1, proyectoID, "activa", "codex", "localhost", "main"); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	rec := httptest.NewRecorder()
	webHandlerAPIAgentesLista(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api/agentes status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/asignaciones?estado=activa&agente=codex1", nil)
	rec = httptest.NewRecorder()
	webHandlerAPIAsignaciones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api/asignaciones status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var asignaciones struct {
		Items []db.Asignacion `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &asignaciones); err != nil {
		t.Fatalf("json asignaciones: %v", err)
	}
	if len(asignaciones.Items) != 1 {
		t.Fatalf("asignaciones inesperadas: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sesiones", nil)
	rec = httptest.NewRecorder()
	webHandlerAPISesiones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api/sesiones status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var sesiones struct {
		Items []db.SesionActiva `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sesiones); err != nil {
		t.Fatalf("json sesiones: %v", err)
	}
	if len(sesiones.Items) != 1 {
		t.Fatalf("sesiones inesperadas: %s", rec.Body.String())
	}
}

func TestAPIAgentesMutaciones(t *testing.T) {
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
	if err := db.RegistrarAgente("codex9", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agentes", bytes.NewBufferString(`{"nombre":"codex10","rol":"documentador"}`))
	rec := httptest.NewRecorder()
	webHandlerAPIAgentesLista(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("crear agente status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/agentes/codex9/retirar", nil)
	webHandlerAPIAgenteRetirar(rec, req, "codex9")
	if rec.Code != http.StatusOK {
		t.Fatalf("retirar agente status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/agentes/codex9/rehabilitar", nil)
	webHandlerAPIAgenteRehabilitar(rec, req, "codex9")
	if rec.Code != http.StatusOK {
		t.Fatalf("rehabilitar agente status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
}

func TestWebSesionesFiltrosYDetalle(t *testing.T) {
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
	pid := int64(4321)
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                "/tmp/orquestador",
		Herramienta:        "codex",
		ExternalSessionID:  "sess-123",
		ResumePayloadJSON:  `{"resume":"ok"}`,
		ResumenContinuidad: "continuidad de prueba",
		Branch:             "main",
		Host:               "localhost",
		PID:                &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/sesiones?agente=Codex1&proyecto=orquestador&activa=true", nil)
	rec := httptest.NewRecorder()
	webHandlerSesiones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sesiones status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "sess-123") || !strings.Contains(body, "/sesiones/"+itoa(sesion.ID)) {
		t.Fatalf("listado de sesiones incompleto: %s", body)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/sesiones/"+itoa(sesion.ID), nil)
	webRouterSesiones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle sesion status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if !strings.Contains(body, "continuidad de prueba") || !strings.Contains(body, "resume") {
		t.Fatalf("detalle de sesión incompleto: %s", body)
	}
}
