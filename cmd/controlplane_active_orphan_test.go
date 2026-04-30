package cmd

import (
	"path/filepath"
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarReactivacionAgentesSinRuntimeBatchUsaIdleSinTareaComoRelevoParaActivaHuerfana(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-activa-huerfana-idle",
		Nombre:  "Orquestador Activa Huerfana Idle",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexCaido", "CodexIdle"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente activo"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente activo huerfano",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexCaido"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexCaido"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil || tarea == nil {
		t.Fatalf("get tarea: %+v err=%v", tarea, err)
	}

	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexCaido", Habilitado: true},
			Asignacion:      &db.Asignacion{Agente: "CodexCaido", ProyectoID: proyectoID, ProyectoSlug: "orquestador-activa-huerfana-idle", Estado: db.AsignacionActiva},
			EstadoOperativo: "caido",
			OpenTasks:       1,
			WorkerAlive:     false,
		},
		{
			Agente:          &db.Agente{Nombre: "CodexIdle", Habilitado: true},
			Asignacion:      &db.Asignacion{Agente: "CodexIdle", ProyectoID: proyectoID, ProyectoSlug: "orquestador-activa-huerfana-idle", Estado: db.AsignacionActiva},
			EstadoOperativo: "sin_tarea",
			OpenTasks:       0,
			BlockedTasks:    0,
			OrdersOpen:      0,
			WorkerAlive:     false,
		},
	}

	n, err := procesarReactivacionAgentesSinRuntimeBatch(rows, map[string][]*db.Tarea{
		"CodexCaido": {tarea},
	}, nil)
	if err != nil {
		t.Fatalf("procesar reactivacion sin runtime: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia relevar una tarea activa huerfana a un idle sin_tarea, got=%d", n)
	}

	actual, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea reasignada: %v", err)
	}
	if actual == nil || actual.Agente == nil || *actual.Agente != "CodexIdle" {
		t.Fatalf("la tarea deberia quedar reasignada a CodexIdle: %+v", actual)
	}
	if actual.Estado != db.EstadoAsignada && actual.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia quedar reabierta para el relevo: %+v", actual)
	}

	agente := "CodexIdle"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders relevo: %v", err)
	}
	if len(orders) == 0 {
		t.Fatalf("deberia encolar continuidad o handoff para el relevo")
	}
}
