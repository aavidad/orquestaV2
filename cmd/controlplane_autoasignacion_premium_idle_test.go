package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarAutoasignacionPremiumSinSesionBatchReutilizaProyectoPausadoReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoID, "sin_trabajo_reactivacion_automatica"); err != nil {
		t.Fatalf("pausar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		EstadoOperativo: "sin_tarea",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia autoasignar premium idle sin sesion desde proyecto pausado, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Gemini1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea libre deberia quedar tomada y arrancada en Gemini1: %+v", tarea)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para premium idle sin sesion: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"premium_idle_autoassigned"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutoasignacionPremiumSinSesionBatchUsaAsignacionActivaSinRuntime(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Runtime mailbox/session_resume",
		Descripcion: "Write-set exclusivo: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:microrefactor_loop",
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}
	asignacion, err := db.GetAsignacionActivaAgente("Gemini1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		Asignacion:      asignacion,
		EstadoOperativo: "sin_tarea",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia autoasignar premium idle desde asignacion activa sin runtime, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Gemini1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea libre deberia quedar tomada y arrancada en Gemini1: %+v", tarea)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para premium idle desde asignacion activa: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"premium_idle_autoassigned"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestResolverProyectoAutoasignacionPremiumSinSesionPrefiereProyectoPremiumSobrePausaMasReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoPremiumID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto premium: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoPremiumID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion premium: %v", err)
	}
	proyectoAPIID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "api",
		Nombre:  "API",
		RutaAbs: filepath.Join(tmp, "api"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto api: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoPremiumID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion premium: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoPremiumID, "sin_trabajo_reactivacion_automatica"); err != nil {
		t.Fatalf("pausar asignacion premium: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoAPIID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion api: %v", err)
	}
	if err := db.PausarAsignacion("Gemini1", proyectoAPIID, "sin_trabajo_espera_automatica"); err != nil {
		t.Fatalf("pausar asignacion api: %v", err)
	}

	proyecto, err := resolverProyectoAutoasignacionPremiumSinSesion("Gemini1", agentesapp.Row{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		EstadoOperativo: "sin_tarea",
	})
	if err != nil {
		t.Fatalf("resolverProyectoAutoasignacionPremiumSinSesion: %v", err)
	}
	if proyecto == nil || proyecto.ID != proyectoPremiumID {
		t.Fatalf("deberia preferir el proyecto premium con continuidad frente a la pausa mas reciente: %+v", proyecto)
	}
}
