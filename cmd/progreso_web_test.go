package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxProgresoWeb() *http.ServeMux {
	mux := http.NewServeMux()
	registrarRutasServe(mux)
	return mux
}

func TestWebProgresoResumenYMutacionesPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	faseID, err := db.RegistrarFaseProyecto(&db.FaseProyecto{
		Proyecto: "orquestador",
		Nombre:   "Arquitectura",
		Orden:    10,
		Peso:     2,
		Estado:   "activa",
	})
	if err != nil {
		t.Fatalf("registrar fase: %v", err)
	}
	tareaFaseID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Diseñar backend",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "Codex1",
		Descripcion: "Diseño inicial",
	})
	if err != nil {
		t.Fatalf("crear tarea con fase: %v", err)
	}
	tareaLibreID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Preparar CI",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadMedia,
		CreadoPor:   "Codex1",
		Descripcion: "Pipeline base",
	})
	if err != nil {
		t.Fatalf("crear tarea sin fase: %v", err)
	}
	if err := db.RegistrarAvanceTarea(&db.AvanceTarea{
		TareaID:        tareaFaseID,
		Proyecto:       "orquestador",
		FaseID:         &faseID,
		ProgresoPct:    55,
		ActualizadoPor: "Codex1",
	}); err != nil {
		t.Fatalf("registrar avance inicial: %v", err)
	}
	if err := db.RegistrarAvanceTarea(&db.AvanceTarea{
		TareaID:        tareaLibreID,
		Proyecto:       "orquestador",
		ProgresoPct:    0,
		ActualizadoPor: "Codex1",
	}); err != nil {
		t.Fatalf("registrar avance inicial sin fase: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/progreso?proyecto=orquestador&lang=en", nil)
	testMuxProgresoWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("progreso status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"<html lang=\"en\">", "Arquitectura", "Diseñar backend", "Preparar CI"} {
		if !strings.Contains(body, token) {
			t.Fatalf("vista de progreso incompleta, falta %q:\n%s", token, body)
		}
	}

	formFase := url.Values{
		"proyecto":    {"orquestador"},
		"nombre":      {"Frontend"},
		"descripcion": {"UI principal"},
		"orden":       {"20"},
		"peso":        {"1.5"},
		"estado":      {"pendiente"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/progreso/fases", strings.NewReader(formFase.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxProgresoWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("registrar fase web status=%d body=%s", rec.Code, rec.Body.String())
	}
	fases, err := db.ListarFasesProyecto("orquestador")
	if err != nil {
		t.Fatalf("listar fases: %v", err)
	}
	if len(fases) != 2 {
		t.Fatalf("fases inesperadas: %+v", fases)
	}

	formAvance := url.Values{
		"proyecto": {"orquestador"},
		"fase":     {strconv.FormatInt(faseID, 10)},
		"pct":      {"80"},
		"por":      {"Codex2"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/progreso/tareas/"+strconv.FormatInt(tareaLibreID, 10)+"/registrar", strings.NewReader(formAvance.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxProgresoWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("registrar avance web status=%d body=%s", rec.Code, rec.Body.String())
	}
	avances, err := db.ListarAvanceTareasProyecto("orquestador")
	if err != nil {
		t.Fatalf("listar avances: %v", err)
	}
	var encontrado bool
	for _, item := range avances {
		if item != nil && item.Tarea != nil && item.Tarea.ID == tareaLibreID {
			encontrado = true
			if item.ProgresoPct != 80 || item.FaseID == nil || *item.FaseID != faseID {
				t.Fatalf("avance de tarea inesperado: %+v", item)
			}
		}
	}
	if !encontrado {
		t.Fatalf("no se encontró avance para tarea %d: %+v", tareaLibreID, avances)
	}
}
