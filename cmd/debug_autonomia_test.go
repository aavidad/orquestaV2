package cmd

import (
    "encoding/json"
    "fmt"
    "path/filepath"
    "testing"
    "time"

    "orquesta/db"
    "orquesta/runtimeagente"
)

func TestDebugAutonomiaSesionActivaBootstrapOnly(t *testing.T) {
	t.Skip("debug test; behavior validated via unit tests for rowPermiteAutoRecuperacion and pipeline flows")

    tmp := prepararDBTemporalCmd(t)
    if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
        t.Fatalf("registrar agente: %v", err)
    }
    proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: filepath.Join(tmp, "orquestador"), Tipo: db.ProyectoRepo, Activo: true})
    if err != nil {
        t.Fatalf("upsert proyecto: %v", err)
    }

    metaJSON, err := json.Marshal(map[string]any{"driver": "tmux_cli_session", "tmux_session": "orq-gemini1-bootstrap", "mailbox_delivery_mode": string(runtimeagente.MailboxDeliveryBootstrapOnly), "tmux_pane_id": "%1"})
    if err != nil {
        t.Fatalf("marshal metadata: %v", err)
    }

    if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente recuperable"); err != nil {
        t.Fatalf("activar asignacion: %v", err)
    }
    sesion, err := db.IniciarSesionContexto(db.SesionInicio{Agente: "Gemini1", ProyectoID: &proyectoID, CWD: filepath.Join(tmp, "orquestador"), Herramienta: "gemini-cli"})
    if err != nil {
        t.Fatalf("iniciar sesion: %v", err)
    }
    handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
    if err != nil || handle == nil {
        t.Fatalf("handle: %+v err=%v", handle, err)
    }
    if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
        t.Fatalf("update handle: %v", err)
    }
    runtime, err := db.GetRuntimeBySesionID(sesion.ID)
    if err != nil || runtime == nil {
        t.Fatalf("runtime: %+v err=%v", runtime, err)
    }
    if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtime.ID); err != nil {
        t.Fatalf("update runtime: %v", err)
    }

    tareaID, err := db.CrearTarea(&db.Tarea{Titulo: "Recuperar tarea bloqueada desde sesion activa", Descripcion: "test", ProyectoID: &proyectoID, Prioridad: db.PrioridadAlta, CreadoPor: "alberto"})
    if err != nil {
        t.Fatalf("crear tarea: %v", err)
    }
    if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
        t.Fatalf("tomar tarea: %v", err)
    }
    if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
        t.Fatalf("iniciar tarea: %v", err)
    }
    if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_runtime: worker recuperado"); err != nil {
        t.Fatalf("bloquear tarea: %v", err)
    }

    rows, err := agentesService.BuildPanelRows()
    if err != nil {
        t.Fatalf("build rows: %v", err)
    }
    for _, row := range rows {
        if row.Agente == nil || row.Agente.Nombre != "Gemini1" {
            continue
        }
        now := time.Now().UTC()
        fmt.Printf("row operativo=%s open=%d blocked=%d fresh=%v tmuxFresh=%v continuity=%v workerState=%s workerDriver=%s mailbox=%s canSend=%v handles=%v runtime=%v\n",
            row.EstadoOperativo, row.OpenTasks, row.BlockedTasks, row.WorkerFresh(now), row.WorkerTMUXFresh(now), row.WorkerSupportsContinuityRecovery(now), row.WorkerState, row.WorkerDriver, row.WorkerMailboxDeliveryMode, row.WorkerCanSendInput, row.Handle != nil, row.Runtime != nil)
        if row.Asignacion != nil {
            fmt.Printf("asignacion estado=%s nota=%s proyecto=%d\n", row.Asignacion.Estado, row.Asignacion.Nota, row.Asignacion.ProyectoID)
        }
        if row.Runtime != nil {
            fmt.Printf("runtime id=%d state=%s logical=%s process=%s\n", row.Runtime.ID, "", row.Runtime.LogicalState, row.Runtime.ProcessState)
        }
    }

    detail, err := agentesService.BuildDetailCompact("Gemini1")
    if err != nil || detail == nil {
        t.Fatalf("detail: %+v err=%v", detail, err)
    }
    fmt.Printf("detail row operativo=%s blocked=%d open=%d\n", detail.Row.EstadoOperativo, detail.Row.BlockedTasks, detail.Row.OpenTasks)

    bloqueos, err := db.ListarResumenBloqueos()
    if err != nil {
        t.Fatalf("bloqueos: %v", err)
    }
    for _, b := range bloqueos {
        fmt.Printf("bloqueo id=%d agente=%s motivo=%s titulo=%s\n", b.ID, b.Agente, b.Motivo, b.Titulo)
    }

    procesadas, err := procesarAutonomiaSesionActiva(sesion)
    fmt.Printf("procesadas=%d err=%v\n", procesadas, err)
    tarea, err := db.GetTarea(tareaID)
    if err != nil {
        t.Fatalf("get tarea: %v", err)
    }
    fmt.Printf("tarea estado=%s agente=%v\n", tarea.Estado, tarea.Agente)
    if procesadas != 1 {
        t.Fatalf("esperaba 1")
    }
}
