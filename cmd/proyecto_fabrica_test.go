package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/fabricaapp"
)

type fakeProjectAppFactory struct {
	result fabricaapp.GenerationResult
	err    error
}

func (f fakeProjectAppFactory) Generate(spec fabricaapp.AppSpec) (fabricaapp.GenerationResult, error) {
	if f.err != nil {
		return fabricaapp.GenerationResult{}, f.err
	}
	return f.result, nil
}

func TestProyectoFabricarAppCreaBacklogYResuelveDependencias(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "mi-app",
		Nombre:  "Mi App",
		RutaAbs: filepath.Join(tmp, "mi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	prevFactory := newProjectAppFactory
	newProjectAppFactory = func() projectAppFactory {
		return fakeProjectAppFactory{
			result: fabricaapp.GenerationResult{
				Tasks: []fabricaapp.BlueprintTask{
					{
						Key:       "arquitectura",
						Titulo:    "Arquitectura base",
						Modulo:    "arquitectura",
						Prioridad: db.PrioridadAlta,
					},
					{
						Key:              "frontend",
						Titulo:           "Frontend principal",
						Modulo:           "frontend",
						Prioridad:        db.PrioridadAlta,
						Dependencias:     []string{"arquitectura"},
						ContratoDefinido: true,
					},
				},
			},
		}
	}
	t.Cleanup(func() {
		newProjectAppFactory = prevFactory
	})

	resetCommandFlags(proyectoFabricarAppCmd)
	if err := proyectoFabricarAppCmd.Flags().Set("tipo", "web"); err != nil {
		t.Fatalf("set tipo: %v", err)
	}
	if err := proyectoFabricarAppCmd.Flags().Set("nombre", "Mi App"); err != nil {
		t.Fatalf("set nombre: %v", err)
	}
	if err := proyectoFabricarAppCmd.Flags().Set("por", "Codex1"); err != nil {
		t.Fatalf("set por: %v", err)
	}

	out := capturarStdout(t, func() {
		if err := proyectoFabricarAppCmd.RunE(proyectoFabricarAppCmd, []string{"mi-app"}); err != nil {
			t.Fatalf("proyecto fabricar-app: %v", err)
		}
	})
	if !strings.Contains(out, "Backlog de app generado para mi-app") || !strings.Contains(out, "Tareas creadas: 2") {
		t.Fatalf("salida inesperada:\n%s", out)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 2 {
		t.Fatalf("se esperaban 2 tareas, hay %d", len(tareas))
	}

	var arquitectura, frontend *db.Tarea
	for _, tarea := range tareas {
		switch tarea.Modulo {
		case "arquitectura":
			arquitectura = tarea
		case "frontend":
			frontend = tarea
		}
	}
	if arquitectura == nil || frontend == nil {
		t.Fatalf("tareas creadas inesperadas: %+v", tareas)
	}
	if frontend.Estado != db.TareaBacklog {
		t.Fatalf("la tarea dependiente deberia quedar en backlog: %+v", frontend)
	}
	if len(frontend.Dependencias) != 1 || frontend.Dependencias[0] != arquitectura.ID {
		t.Fatalf("dependencias no resueltas: %+v", frontend)
	}
	if !frontend.ContratoDefinido {
		t.Fatalf("contrato_definido no persistido: %+v", frontend)
	}
}

func TestProyectoFabricarAppValidaFlagsMinimos(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "mi-app",
		Nombre:  "Mi App",
		RutaAbs: filepath.Join(tmp, "mi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	prevFactory := newProjectAppFactory
	newProjectAppFactory = func() projectAppFactory {
		t.Fatalf("la fabrica no deberia ejecutarse cuando faltan flags obligatorios")
		return nil
	}
	t.Cleanup(func() {
		newProjectAppFactory = prevFactory
	})

	resetCommandFlags(proyectoFabricarAppCmd)
	err := proyectoFabricarAppCmd.RunE(proyectoFabricarAppCmd, []string{"mi-app"})
	if err == nil {
		t.Fatalf("se esperaba error")
	}
	if !strings.Contains(err.Error(), "--tipo es obligatorio") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestProyectoFabricarAppPermiteArranqueDeCuadrillaNueva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "mi-app",
		Nombre:  "Mi App",
		RutaAbs: filepath.Join(tmp, "mi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexA", "CodexB"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "cuadrilla app"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}

	prevFactory := newProjectAppFactory
	newProjectAppFactory = func() projectAppFactory {
		return fakeProjectAppFactory{
			result: fabricaapp.GenerationResult{
				Tasks: []fabricaapp.BlueprintTask{
					{
						Key:       "briefing",
						Titulo:    "Briefing",
						Modulo:    "producto",
						Prioridad: db.PrioridadAlta,
					},
					{
						Key:       "investigacion",
						Titulo:    "Investigacion",
						Modulo:    "analisis",
						Prioridad: db.PrioridadAlta,
					},
					{
						Key:          "arquitectura",
						Titulo:       "Arquitectura",
						Modulo:       "arquitectura",
						Prioridad:    db.PrioridadAlta,
						Dependencias: []string{"briefing", "investigacion"},
					},
				},
			},
		}
	}
	t.Cleanup(func() {
		newProjectAppFactory = prevFactory
	})

	resetCommandFlags(proyectoFabricarAppCmd)
	if err := proyectoFabricarAppCmd.Flags().Set("tipo", "web_api"); err != nil {
		t.Fatalf("set tipo: %v", err)
	}
	if err := proyectoFabricarAppCmd.Flags().Set("nombre", "Mi App"); err != nil {
		t.Fatalf("set nombre: %v", err)
	}
	if err := proyectoFabricarAppCmd.Flags().Set("por", "Codex1"); err != nil {
		t.Fatalf("set por: %v", err)
	}
	if err := proyectoFabricarAppCmd.RunE(proyectoFabricarAppCmd, []string{"mi-app"}); err != nil {
		t.Fatalf("fabricar app: %v", err)
	}

	if err := db.PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	asignadas := map[string]int{}
	for _, tarea := range tareas {
		if tarea.Agente != nil && (*tarea.Agente == "CodexA" || *tarea.Agente == "CodexB") {
			asignadas[*tarea.Agente]++
		}
	}
	if asignadas["CodexA"] != 1 || asignadas["CodexB"] != 1 {
		t.Fatalf("la cuadrilla deberia arrancar con dos tareas iniciales paralelas: %+v", asignadas)
	}

	estado := "pendiente"
	for _, agente := range []string{"CodexA", "CodexB"} {
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
		if err != nil {
			t.Fatalf("listar runtime orders %s: %v", agente, err)
		}
		if len(orders) != 1 || orders[0].Tipo != "start" {
			t.Fatalf("start no encolada para %s: %+v", agente, orders)
		}
	}
}

func TestProyectoFabricarAppUsaAPICuandoHayServidor(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/proyectos/mi-app/fabricar-app", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("metodo inesperado: %s", r.Method)
		}
		var req apiProyectoFabricarAppRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode fabricar-app: %v", err)
		}
		if req.Tipo != "web_api" || req.Nombre != "Mi App" || req.Por != "Codex1" {
			t.Fatalf("payload fabricar-app inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiProyectoFabricarAppResponse{
			OK:      true,
			Slug:    "mi-app",
			Tipo:    req.Tipo,
			Created: 8,
			Backlog: 3,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	resetCommandFlags(proyectoFabricarAppCmd)
	_ = proyectoFabricarAppCmd.Flags().Set("tipo", "web_api")
	_ = proyectoFabricarAppCmd.Flags().Set("nombre", "Mi App")
	_ = proyectoFabricarAppCmd.Flags().Set("por", "Codex1")

	out := capturarStdout(t, func() {
		if err := proyectoFabricarAppCmd.RunE(proyectoFabricarAppCmd, []string{"mi-app"}); err != nil {
			t.Fatalf("proyecto fabricar-app via api: %v", err)
		}
	})
	if !strings.Contains(out, "Backlog de app generado para mi-app") || !strings.Contains(out, "Tareas creadas: 8") {
		t.Fatalf("salida fabricar-app via api inesperada:\n%s", out)
	}
}
