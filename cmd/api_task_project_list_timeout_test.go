package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestAPIHandlerTareasListarReturns503WhenListHangs(t *testing.T) {
	prevFn := apiListTasksFn
	prevTimeout := apiTaskListTimeout
	t.Cleanup(func() {
		apiListTasksFn = prevFn
		apiTaskListTimeout = prevTimeout
	})

	apiTaskListTimeout = 20 * time.Millisecond
	apiListTasksFn = func(filter db.FiltroTareas) ([]*db.Tarea, error) {
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tareas?estado=libre", nil)
	rec := httptest.NewRecorder()
	apiHandlerTareasListar(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "tareas temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestAPIHandlerProyectosReturns503WhenListHangs(t *testing.T) {
	prevFn := apiListProjectsFn
	prevTimeout := apiProjectListTimeout
	t.Cleanup(func() {
		apiListProjectsFn = prevFn
		apiProjectListTimeout = prevTimeout
	})

	apiProjectListTimeout = 20 * time.Millisecond
	apiListProjectsFn = func(filtro db.FiltroProyectos, cwdHint string) ([]*db.Proyecto, error) {
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/proyectos", nil)
	rec := httptest.NewRecorder()
	apiHandlerProyectos(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "proyectos temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestAPIHandlerTareasListarResumenOmiteNotasPesadas(t *testing.T) {
	prevFn := apiListTasksFn
	t.Cleanup(func() {
		apiListTasksFn = prevFn
	})

	apiListTasksFn = func(filter db.FiltroTareas) ([]*db.Tarea, error) {
		return []*db.Tarea{{
			ID:          9,
			Titulo:      "Lista compacta",
			Descripcion: "detalle que no debe viajar",
			Notas:       "bloque enorme",
			Modulo:      "controlplane",
			Estado:      db.EstadoLibre,
			Prioridad:   db.PrioridadAlta,
		}}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tareas?resumen=1", nil)
	rec := httptest.NewRecorder()
	apiHandlerTareasListar(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiTareasResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Tareas) != 1 {
		t.Fatalf("len(resp.Tareas)=%d want 1", len(resp.Tareas))
	}
	if resp.Tareas[0].Descripcion != "" || resp.Tareas[0].Notas != "" {
		t.Fatalf("la respuesta resumen no deberia incluir descripcion/notas: %+v", resp.Tareas[0])
	}
}
