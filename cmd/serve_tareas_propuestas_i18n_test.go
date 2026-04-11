package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestWebTareasYPropuestasRespetanIdiomaDelRequest(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "documentador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Cerrar i18n web",
		Descripcion: "Barrido de literales",
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "Codex1",
		ProyectoID:  &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.RegistrarAvanceTarea(&db.AvanceTarea{
		TareaID:        tareaID,
		Proyecto:       "orquestador",
		ProgresoPct:    72,
		ActualizadoPor: "Codex1",
	}); err != nil {
		t.Fatalf("registrar avance tarea: %v", err)
	}

	propuestaID, err := db.CrearPropuesta(&db.Propuesta{
		Codigo:       "OP-401",
		Titulo:       "Completar i18n web",
		Descripcion:  "Cerrar textos duros en tareas y propuestas",
		Tipo:         "arquitectura",
		PropuestoPor: "Codex1",
		ProyectoID:   &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	if _, err := db.Votar(propuestaID, "Codex2", db.VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("votar propuesta: %v", err)
	}

	mux := testMuxAgentesWeb()

	assertPage := func(path string, needles ...string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		for _, needle := range needles {
			if !strings.Contains(body, needle) {
				t.Fatalf("respuesta %s sin %q:\n%s", path, needle, body)
			}
		}
		if !strings.Contains(body, `<html lang="en">`) {
			t.Fatalf("html lang inesperado en %s: %s", path, body)
		}
		if got := rec.Header().Get("Content-Language"); got != "en" {
			t.Fatalf("Content-Language %s = %q", path, got)
		}
		return rec
	}

	assertPage(
		"/tareas?lang=en",
		"Tasks",
		"New task",
		"Manage",
		"high",
		"Cerrar i18n web",
		"72%",
	)
	assertPage(
		"/tareas/"+itoa(tareaID)+"?lang=en",
		"back to tasks",
		"Complete task",
		"Reassign to another agent",
		"Add note",
	)
	assertPage(
		"/propuestas?lang=en",
		"Proposals (OPs)",
		"New proposal",
		"Manage",
		"Completar i18n web",
	)
	assertPage(
		"/propuestas/OP-401?lang=en",
		"back to proposals",
		"Vote",
		"Close proposal (Alberto)",
		"Recorded votes",
		"Proposed by Codex1",
	)
}
