package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarReactivacionAgentesSinRuntimeBatchResuelveProyectoDesdeTareaActiva(t *testing.T) {
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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar frente premium sin runtime vivo",
		Descripcion: "El batch debe resolver el proyecto desde la tarea activa canónica.",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil || tarea == nil {
		t.Fatalf("get tarea: %+v err=%v", tarea, err)
	}

	rows := []agentesapp.Row{{
		Agente:           &db.Agente{Nombre: "Gemini1", Habilitado: true},
		EstadoOperativo:  "bloqueado_por_runtime",
		DetalleOperativo: "tmux pane finalizado",
		OpenTasks:        1,
	}}
	tareasActivas := map[string][]*db.Tarea{
		"Gemini1": {tarea},
	}

	n, err := procesarReactivacionAgentesSinRuntimeBatch(rows, tareasActivas, nil)
	if err != nil {
		t.Fatalf("procesar reactivacion sin runtime: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia encolar una reactivacion usando el proyecto resuelto desde la tarea activa, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("deberia quedar una runtime order pendiente, got=%d", len(orders))
	}
	if orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start al no haber runtime vivo, got=%+v", orders[0])
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"agente_sin_runtime_activo"`) {
		t.Fatalf("faltaba el motivo esperado en payload: %s", orders[0].PayloadJSON)
	}
}
