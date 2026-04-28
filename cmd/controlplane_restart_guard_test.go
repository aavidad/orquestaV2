package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarWorkerAtascadoAutonomiaRowNoReiniciaConSendInstructionPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	if err := db.ConfigSet("runtime_shared_account_active_ceiling", "4"); err != nil {
		t.Fatalf("config ceiling: %v", err)
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
	if err := db.RegistrarAgente("Codex2Row", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex2Row", proyectoID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex2Row",
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
	handle, err := db.GetRuntimeHandleActivoAgenteProyecto("Codex2Row", &proyectoID)
	if err != nil || handle == nil {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex2Row",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		Estado:      "pendiente",
		PayloadJSON: `{"texto":"continua"}`,
	}); err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	now := time.Now().UTC()
	row := agentesapp.Row{
		Agente:           &db.Agente{Nombre: "Codex2Row"},
		Asignacion:       &db.Asignacion{Agente: "Codex2Row", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
		Handle:           handle,
		WorkerDriver:     "tmux_cli_session",
		WorkerState:      "running",
		WorkerAlive:      true,
		WorkerHeartbeat:  ptrTime(now),
		WorkerUpdatedAt:  ptrTime(now),
		OpenTasks:        1,
		EstadoOperativo:  "atascado",
		DetalleOperativo: "worker sin progreso reciente",
	}

	procesadas, err := procesarWorkerAtascadoAutonomiaRow(row, []agentesapp.Row{row}, now)
	if err != nil {
		t.Fatalf("procesar worker atascado row: %v", err)
	}
	if procesadas != 0 {
		t.Fatalf("no deberia reencolar restart con send_instruction viva, got=%d", procesadas)
	}
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: ptrString("Codex2Row")})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0] == nil || orders[0].Tipo != "send_instruction" {
		t.Fatalf("no deberia crear stop/start extra, got=%+v", orders)
	}
}
