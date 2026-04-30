package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/runtimeagente"
)

func TestProcesarWorkerAtascadoAutonomiaRowNoReinyectaContinuacionLocalReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaContinueNudgeGate.Reset()
	if err := db.ConfigSet("runtime_shared_account_active_ceiling", "4"); err != nil {
		t.Fatalf("config ceiling: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-worker-ready-stale-retry",
		Nombre:  "Orquestador Worker Ready Stale Retry",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("CodexReadyRetry", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReadyRetry", proyectoID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReadyRetry",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Slice listo para continuidad local sin duplicado",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReadyRetry"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReadyRetry"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	now := time.Now().UTC()
	row := agentesapp.Row{
		Agente:                    &db.Agente{Nombre: "CodexReadyRetry"},
		Asignacion:                &db.Asignacion{Agente: "CodexReadyRetry", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
		Sesion:                    sesion,
		Handle:                    handle,
		WorkerDriver:              "tmux_cli_session",
		WorkerState:               "ready",
		WorkerAlive:               true,
		WorkerCanSendInput:        true,
		WorkerMailboxDeliveryMode: runtimeagente.MailboxDeliverySessionResume,
		WorkerHeartbeat:           ptrTime(now),
		WorkerUpdatedAt:           ptrTime(now),
		WorkerReadyAt:             ptrTime(now.Add(-25 * time.Minute)),
		OpenTasks:                 1,
		EstadoOperativo:           "atascado",
		DetalleOperativo:          "sin progreso desde listo",
	}

	procesadas, err := procesarWorkerAtascadoAutonomiaRow(row, []agentesapp.Row{row}, now)
	if err != nil {
		t.Fatalf("primer intento worker atascado: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("primer intento procesadas=%d, want 1", procesadas)
	}
	agente := "CodexReadyRetry"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders primer intento: %v", err)
	}
	var firstNudgeID int64
	for _, order := range orders {
		if order != nil && strings.TrimSpace(order.Tipo) == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"continuar_trabajo"`) {
			firstNudgeID = order.ID
			break
		}
	}
	if firstNudgeID <= 0 {
		t.Fatalf("faltaba nudge inicial de continuidad: %+v", orders)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='tomada', updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), firstNudgeID); err != nil {
		t.Fatalf("marcar nudge tomada: %v", err)
	}

	procesadas, err = procesarWorkerAtascadoAutonomiaRow(row, []agentesapp.Row{row}, now.Add(15*time.Second))
	if err != nil {
		t.Fatalf("segundo intento worker atascado: %v", err)
	}
	if procesadas != 0 {
		t.Fatalf("no deberia reinyectar continuidad reciente, got=%d", procesadas)
	}
	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders segundo intento: %v", err)
	}
	nudgeCount := 0
	for _, order := range orders {
		if order != nil && strings.TrimSpace(order.Tipo) == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"continuar_trabajo"`) {
			nudgeCount++
		}
	}
	if nudgeCount != 1 {
		t.Fatalf("no deberia duplicar nudge de continuidad reciente, got=%d orders=%+v", nudgeCount, orders)
	}
}
