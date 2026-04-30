package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
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

func TestProcesarReactivacionAgentesSinRuntimeBatchReanimaSupervisorReservadoConContinuidadPendienteAunqueFigureTrabajando(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
		SupervisorAgente:     "Codex2",
		ReserveSupervisor:    true,
		ReviewerAgente:       "",
		ReserveReviewer:      false,
		ObjetivoGeneral:      "cerrar app",
		DefinitionOfDoneJSON: "{}",
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Codex2", proyectoID, "supervision_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","texto":"reanudar supervision reservada"}`,
	}); err != nil {
		t.Fatalf("mailbox autonomia: %v", err)
	}

	rows := []agentesapp.Row{{
		Agente:                   &db.Agente{Nombre: "Codex2", Habilitado: true},
		Asignacion:               &db.Asignacion{Agente: "Codex2", ProyectoID: proyectoID, Estado: db.AsignacionActiva, Nota: "supervision_automatica"},
		EstadoOperativo:          "trabajando",
		DetalleOperativo:         "worker ready con runtime stale",
		WorkerAlive:              true,
		MailboxPending:           1,
		MailboxContinuityPending: 1,
	}}

	n, err := procesarReactivacionAgentesSinRuntimeBatch(rows, nil, nil)
	if err != nil {
		t.Fatalf("procesar reactivacion sin runtime reservado: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reactivar supervisor reservado sin runtime operativo, got=%d", n)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0] == nil || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para supervisor reservado, got=%+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"agente_sin_runtime_activo"`) {
		t.Fatalf("faltaba motivo esperado en payload: %s", orders[0].PayloadJSON)
	}
}

func TestRowTieneRuntimeOHandleOperativoParaReactivacionToleraSnapshotResidualActivoConContinuidadAccionable(t *testing.T) {
	now := time.Now().UTC()
	proyectoID := int64(1)
	row := agentesapp.Row{
		Agente:                   &db.Agente{Nombre: "Codex2", Habilitado: true},
		Asignacion:               &db.Asignacion{Agente: "Codex2", ProyectoID: proyectoID, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "reactivacion_automatica"},
		EstadoOperativo:          "bloqueado_por_runtime",
		DetalleOperativo:         "runtime principal stale",
		WorkerAlive:              true,
		WorkerState:              "ready",
		Handle:                   &db.RuntimeHandle{Estado: "activo", Transporte: "tmux", ProyectoID: &proyectoID},
		Runtime:                  &db.RuntimeInstance{LogicalState: "running", ProyectoID: &proyectoID},
		MailboxActionablePending: 1,
		MailboxContinuityPending: 1,
	}

	if rowTieneRuntimeOHandleOperativoParaReactivacion(row, now) {
		t.Fatal("no deberia tratar snapshot residual activo como runtime operativo para reactivacion cuando ya esta bloqueado_por_runtime y hay continuidad accionable")
	}
}

func TestRowTieneTrabajoReactivableSinRuntimeCuentaContinuidadAccionable(t *testing.T) {
	now := time.Now().UTC()
	row := agentesapp.Row{
		Agente:                   &db.Agente{Nombre: "Codex2", Habilitado: true},
		MailboxActionablePending: 1,
		MailboxContinuityPending: 1,
	}

	if !rowTieneTrabajoReactivableSinRuntime(row, nil, nil, nil, now) {
		t.Fatal("deberia contar continuidad accionable como trabajo reactivable sin runtime")
	}
}
