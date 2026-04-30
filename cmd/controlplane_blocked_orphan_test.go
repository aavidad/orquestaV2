package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarIntervencionTareasBloqueadasDegradadasBatchReasignaBloqueadoHuerfanoSinRuntimeUtil(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-bloqueado-huerfano",
		Nombre:  "Orquestador Bloqueado Huerfano",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexBloq", "CodexLibre"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente bloqueado"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente bloqueado huerfano",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexBloq"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexBloq"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "orquesta", "Agente CodexBloq en estado bloqueado: runtime stale sin worker"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	actual, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea bloqueada: %v", err)
	}
	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexBloq", Rol: "programador"},
			Asignacion:      &db.Asignacion{Agente: "CodexBloq", ProyectoID: proyectoID, ProyectoSlug: "orquestador-bloqueado-huerfano", Estado: db.AsignacionActiva},
			EstadoOperativo: "bloqueado",
			BlockedTasks:    1,
			OpenTasks:       0,
			WorkerAlive:     false,
		},
		{
			Agente:          &db.Agente{Nombre: "CodexLibre", Rol: "programador"},
			Asignacion:      &db.Asignacion{Agente: "CodexLibre", ProyectoID: proyectoID, ProyectoSlug: "orquestador-bloqueado-huerfano", Estado: db.AsignacionActiva},
			EstadoOperativo: "disponible",
			OpenTasks:       0,
			WorkerAlive:     true,
			WorkerState:     "ready",
			WorkerHeartbeat: timePtr(now),
			WorkerUpdatedAt: timePtr(now),
		},
	}
	tareasBloqueadas := map[string][]*db.Tarea{
		"CodexBloq": {actual},
	}
	bloqueos := map[int64]db.ResumenBloqueo{
		tareaID: {ID: tareaID, Agente: "CodexBloq", Motivo: "Agente CodexBloq en estado bloqueado: runtime stale sin worker"},
	}

	procesadas, err := procesarIntervencionTareasBloqueadasDegradadasBatch(rows, tareasBloqueadas, bloqueos, openTasksProjectedFromRows(rows), now)
	if err != nil {
		t.Fatalf("procesarIntervencionTareasBloqueadasDegradadasBatch: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reasignar una tarea bloqueada huerfana, got=%d", procesadas)
	}

	reasignada, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea reasignada: %v", err)
	}
	if reasignada == nil || reasignada.Agente == nil || *reasignada.Agente != "CodexLibre" {
		t.Fatalf("la tarea deberia quedar reasignada a CodexLibre: %+v", reasignada)
	}
	if reasignada.Estado != db.EstadoAsignada && reasignada.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia quedar reabierta para el relevo: %+v", reasignada)
	}

	agente := "CodexLibre"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders relevo: %v", err)
	}
	if len(orders) == 0 {
		t.Fatalf("deberia encolar handoff o continuidad para el relevo")
	}
}
