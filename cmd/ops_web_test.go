package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orquesta/db"
)

func TestWebSesionesFiltrosYDetalle(t *testing.T) {
	prepararDBTemporalCmd(t)

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

	req := httptest.NewRequest(http.MethodGet, "/sesiones?agente=Codex1&proyecto=orquestador&activa=true&lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerSesiones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sesiones status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Sessions") || !strings.Contains(body, "sess-123") || !strings.Contains(body, "/sesiones/"+itoa(sesion.ID)) {
		t.Fatalf("listado de sesiones incompleto: %s", body)
	}
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("lang html inesperado: %s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/sesiones/"+itoa(sesion.ID)+"?lang=en", nil)
	webRouterSesiones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle sesion status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Read-only operational inspection detail.") || !strings.Contains(body, "continuidad de prueba") || !strings.Contains(body, "resume") {
		t.Fatalf("detalle de sesión incompleto: %s", body)
	}
}
