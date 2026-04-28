package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

func TestDispatchOrderWorkConfirmedFromHandlesPromotesWeakReceiptWithWorkQueue(t *testing.T) {
	t.Parallel()

	proyectoID := int64(7)
	now := time.Now().UTC()
	tmp := t.TempDir()
	metaJSON, _ := json.Marshal(map[string]any{"trace_dir": tmp})
	if err := controlruntime.RecordWorkQueueFromMetadataJSON(string(metaJSON), controlruntime.WorkQueueRecordInput{
		Kind:            "autonomia",
		Action:          "continuar_trabajo",
		VerificationKey: "vk-7",
		State:           "working",
		RecordedAt:      now,
	}); err != nil {
		t.Fatalf("RecordWorkQueueFromMetadataJSON: %v", err)
	}

	order := &db.RuntimeOrder{
		Agente:        "Codex2",
		ProyectoID:    &proyectoID,
		PayloadJSON:   `{"kind":"autonomia","accion":"continuar_trabajo","verification_key":"vk-7"}`,
		ResultadoJSON: `{"dispatch_state":"delivered","delivery_state":"delivered","receipt_source":"tmux_pane_activity"}`,
	}
	handles := []*db.RuntimeHandle{{
		Agente:       "Codex2",
		ProyectoID:   &proyectoID,
		Estado:       "activo",
		MetadataJSON: string(metaJSON),
		UpdatedAt:    now,
	}}

	if !dispatchOrderWorkConfirmedFromHandles(order, handles) {
		t.Fatalf("deberia confirmar trabajo desde work_queue viva")
	}
}

func TestDispatchOrderWorkConfirmedFromHandlesIgnoresMismatchedHandle(t *testing.T) {
	t.Parallel()

	proyectoID := int64(8)
	now := time.Now().UTC()
	tmp := t.TempDir()
	metaJSON, _ := json.Marshal(map[string]any{"trace_dir": tmp})
	if err := controlruntime.RecordWorkQueueFromMetadataJSON(string(metaJSON), controlruntime.WorkQueueRecordInput{
		Kind:            "autonomia",
		Action:          "continuar_trabajo",
		VerificationKey: "vk-8",
		State:           "working",
		RecordedAt:      now,
	}); err != nil {
		t.Fatalf("RecordWorkQueueFromMetadataJSON: %v", err)
	}

	order := &db.RuntimeOrder{
		Agente:        "Codex2",
		ProyectoID:    &proyectoID,
		PayloadJSON:   `{"kind":"autonomia","accion":"continuar_trabajo","verification_key":"vk-8"}`,
		ResultadoJSON: `{"dispatch_state":"delivered","delivery_state":"delivered","receipt_source":"tmux_pane_activity"}`,
	}
	handles := []*db.RuntimeHandle{{
		Agente:       "Codex3",
		ProyectoID:   &proyectoID,
		Estado:       "activo",
		MetadataJSON: string(metaJSON),
		UpdatedAt:    now,
	}}

	if dispatchOrderWorkConfirmedFromHandles(order, handles) {
		t.Fatalf("no deberia confirmar trabajo con handle de otro agente")
	}
}

func TestDispatchOrderFailureCountsAsDebtSoloSiEsReciente(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 12, 0, 0, 0, time.UTC)
	recent := &db.RuntimeOrder{
		CreatedAt: now.Add(-30 * time.Minute),
		UpdatedAt: now.Add(-20 * time.Minute),
	}
	old := &db.RuntimeOrder{
		CreatedAt: now.Add(-6 * time.Hour),
		UpdatedAt: now.Add(-5 * time.Hour),
	}

	if !dispatchOrderFailureCountsAsDebt(recent, now) {
		t.Fatalf("fallo reciente deberia contar como deuda")
	}
	if dispatchOrderFailureCountsAsDebt(old, now) {
		t.Fatalf("fallo historico no deberia contar como deuda viva")
	}
}

func TestListarDeudaDispatchEstadoPromueveSendInstructionDirectaAbsorbida(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	prevNow := statusNowFunc
	now := time.Now().UTC()
	statusNowFunc = func() time.Time { return now }
	defer func() { statusNowFunc = prevNow }()

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: workingDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	agente := "Codex1"
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Continuar frente activo",
		Estado:     db.TareaEnProgreso,
		Prioridad:  "media",
		ProyectoID: &proyectoID,
		Agente:     &agente,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente=? WHERE id=?`, agente, tareaID); err != nil {
		t.Fatalf("forzar tarea activa: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      agente,
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-codex1-status-dispatch")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker dir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        true,
	})
	writeJSON(statusPath, map[string]any{
		"state":            "running",
		"alive":            true,
		"updated_at":       now.Format(time.RFC3339Nano),
		"last_progress_at": now.Format(time.RFC3339Nano),
		"last_output_at":   now.Format(time.RFC3339Nano),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339Nano),
		"last_progress_at": now.Format(time.RFC3339Nano),
		"last_output_at":   now.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-status","tmux_pane_id":"%%77","mailbox_delivery_mode":"%s","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`,
		runtimeagente.MailboxDeliverySessionResume, manifestPath, statusPath, heartbeatPath)
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=?, updated_at=?, last_seen_at=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"session_resume"}`, metaJSON, now, now, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"from_agente":"orquesta","to_agente":"Codex1","kind":"instruction","classification":"blocked","transcript_id":10128,"texto":"continua de forma autonoma"}`,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if err := db.MarcarRuntimeOrderEstado(orderID, "fallida", `{"ok":false}`, "session_resume timeout"); err != nil {
		t.Fatalf("marcar fallida: %v", err)
	}

	got, err := listarDeudaDispatchEstado()
	if err != nil {
		t.Fatalf("listarDeudaDispatchEstado: %v", err)
	}
	if got.Fallidas != 0 || got.Pendientes != 0 || got.Notificadas != 0 || got.WorkConfirmed != 0 || got.Total != 0 {
		t.Fatalf("la send_instruction directa sin mailbox no deberia contar como deuda durable: %+v", got)
	}
}

func TestNormalizeDispatchDebtTotalSumaSoloCategoriasVivas(t *testing.T) {
	t.Parallel()

	got := normalizeDispatchDebtTotal(deudaDispatchResumen{
		Total:         99,
		Pendientes:    1,
		Notificadas:   2,
		Fallidas:      0,
		WorkConfirmed: 3,
	})
	if got.Total != 6 {
		t.Fatalf("total normalizado inesperado: %+v", got)
	}
}

func TestResumirAutonomiaRowsNoCuentaResumePayloadAbsorbidoComoPendiente(t *testing.T) {
	t.Parallel()

	prevConfigGet := statusConfigGet
	statusConfigGet = func(string) (string, error) { return "Codex1", nil }
	defer func() { statusConfigGet = prevConfigGet }()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			Agente:                   &db.Agente{Nombre: "Codex1"},
			Asignacion:               &db.Asignacion{Estado: db.AsignacionActiva, Nota: "server_autobootstrap"},
			LastAutonomyAction:       "supervisar_proyecto",
			LastAutonomySource:       "resume_payload_mailbox",
			MailboxContinuityPending: 1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "resume_payload_mailbox",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Supervisando != 1 || got.Continuando != 2 {
		t.Fatalf("resumen autonomia inesperado: %+v", got)
	}
	if got.WorkConfirmed != 0 {
		t.Fatalf("no deberia contar work_confirmed en este caso: %+v", got)
	}
	if got.ContinuidadPendiente != 1 {
		t.Fatalf("continuidad pendiente deberia contar solo deuda viva: %+v", got)
	}
}

func TestResumirAutonomiaRowsCuentaSupervisorPorRolAunqueUltimaAccionSeaContinuar(t *testing.T) {
	t.Parallel()

	prevConfigGet := statusConfigGet
	statusConfigGet = func(string) (string, error) { return "Codex1", nil }
	defer func() { statusConfigGet = prevConfigGet }()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			Agente:             &db.Agente{Nombre: "Codex1"},
			Asignacion:         &db.Asignacion{Estado: db.AsignacionActiva, Nota: "server_autobootstrap"},
			LastAutonomyAction: "continuar_trabajo",
			LastAutonomySource: "work_queue",
			LastAutonomyState:  "work_confirmed",
			OpenTasks:          1,
			WorkerAlive:        true,
			WorkerHeartbeat:    ptrTimeStatusDispatch(now),
		},
		{
			Agente:             &db.Agente{Nombre: "Codex2"},
			LastAutonomyAction: "continuar_trabajo",
			LastAutonomySource: "work_queue",
			LastAutonomyState:  "work_confirmed",
			OpenTasks:          1,
			WorkerAlive:        true,
			WorkerHeartbeat:    ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Supervisando != 1 || got.Continuando != 0 {
		t.Fatalf("deberia distinguir supervisor operativo del resto: %+v", got)
	}
	if got.WorkConfirmed != 2 {
		t.Fatalf("deberia contar work_confirmed desde los rows: %+v", got)
	}
}

func TestResumirAutonomiaRowsNoCuentaWorkQueueYaEnRunningComoPendiente(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	absorbed := now.Add(-2 * time.Minute)
	progress := now.Add(-time.Minute)
	rows := []agentesapp.Row{
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "delivered",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "pending",
			LastAutonomyMoment:       &absorbed,
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
			WorkerLastProgress:       &progress,
		},
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "pending",
			LastAutonomyMoment:       ptrTimeStatusDispatch(now),
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Continuando != 3 {
		t.Fatalf("resumen autonomia inesperado: %+v", got)
	}
	if got.WorkConfirmed != 0 {
		t.Fatalf("ninguna work_queue deberia estar confirmada aqui: %+v", got)
	}
	if got.ContinuidadPendiente != 1 {
		t.Fatalf("solo la work_queue no absorbida deberia contar como pendiente: %+v", got)
	}
}

func TestResumirAutonomiaRowsCuentaWorkQueueConActividadYaStaleComoPendiente(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	absorbed := now.Add(-20 * time.Minute)
	progress := now.Add(-11 * time.Minute)
	rows := []agentesapp.Row{
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "pending",
			LastAutonomyMoment:       &absorbed,
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
			WorkerLastProgress:       &progress,
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.ContinuidadPendiente != 1 {
		t.Fatalf("la actividad vieja no deberia absorber continuidad: %+v", got)
	}
}

func TestResumirAutonomiaRowsNoCuentaWorkConfirmedComoContinuidadPendiente(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "work_confirmed",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.WorkConfirmed != 1 {
		t.Fatalf("deberia contar work_confirmed: %+v", got)
	}
	if got.ContinuidadPendiente != 0 {
		t.Fatalf("work_confirmed no deberia seguir contando continuidad: %+v", got)
	}
}

func TestResumirAutonomiaRowsOmiteAgentesBloqueadosPorCuota(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			EstadoOperativo:          "bloqueado_por_cuota",
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "work_confirmed",
			MailboxContinuityPending: 1,
			OpenTasks:                2,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "work_confirmed",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Continuando != 0 || got.WorkConfirmed != 1 || got.ContinuidadPendiente != 0 {
		t.Fatalf("los agentes bloqueados por cuota no deberian contaminar autonomia activa: %+v", got)
	}
}

func TestResumirAutonomiaRowsOmiteAgentesBloqueadosPorRuntimeYAtascados(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			EstadoOperativo:          "bloqueado_por_runtime",
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "work_confirmed",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
		{
			EstadoOperativo:          "atascado",
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "assignment_handoff",
			LastAutonomyState:        "handoff",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
		{
			EstadoOperativo:          "trabajando",
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "work_queue",
			LastAutonomyState:        "work_confirmed",
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Continuando != 0 || got.WorkConfirmed != 1 || got.ContinuidadPendiente != 0 || got.Handoffs != 0 {
		t.Fatalf("los agentes bloqueados por runtime o atascados no deberian contaminar autonomia activa: %+v", got)
	}
}

func TestResumirAutonomiaRowsCuentaHandoffVivoDesdeAssignment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			LastAutonomyAction: "continuar_trabajo",
			LastAutonomySource: "assignment_handoff",
			LastAutonomyState:  "handoff",
			OpenTasks:          1,
			WorkerAlive:        true,
			WorkerHeartbeat:    ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Handoffs != 1 {
		t.Fatalf("deberia contar handoff vivo desde assignment: %+v", got)
	}
	if got.Continuando != 1 {
		t.Fatalf("deberia seguir contando continuidad viva: %+v", got)
	}
}

func TestResumirAutonomiaRowsNoCuentaHandoffConfirmadoDesdeAssignment(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			LastAutonomyAction: "continuar_trabajo",
			LastAutonomySource: "assignment_handoff",
			LastAutonomyState:  "work_confirmed",
			OpenTasks:          1,
			WorkerAlive:        true,
			WorkerHeartbeat:    ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Handoffs != 0 {
		t.Fatalf("no deberia contar handoff ya confirmado: %+v", got)
	}
	if got.Continuando != 0 || got.WorkConfirmed != 1 {
		t.Fatalf("deberia seguir contando continuidad confirmada: %+v", got)
	}
}

func TestResumirAutonomiaRowsNoCuentaContinuandoSinTareaAbierta(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
			LastAutonomyAction:       "continuar_trabajo",
			LastAutonomySource:       "assignment_handoff",
			LastAutonomyState:        "work_confirmed",
			MailboxContinuityPending: 1,
			OpenTasks:                0,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Continuando != 0 {
		t.Fatalf("no deberia contar continuidad sin tarea abierta: %+v", got)
	}
	if got.ContinuidadPendiente != 0 {
		t.Fatalf("la continuidad confirmada no deberia quedar pendiente: %+v", got)
	}
}

func TestListarHandoffsAutonomiaEstadoCuentaPendientesYEjecutando(t *testing.T) {
	t.Parallel()

	prepararDBTemporalCmd(t)
	for _, agente := range []string{"Codex1", "Codex2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{Agente: "Codex2", Tipo: "handoff", Estado: "pendiente"}); err != nil {
		t.Fatalf("encolar handoff pendiente: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{Agente: "Codex2", Tipo: "handoff", Estado: "ejecutando"}); err != nil {
		t.Fatalf("encolar handoff ejecutando: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{Agente: "Codex2", Tipo: "nudge", Estado: "pendiente"}); err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}
	got, err := listarHandoffsAutonomiaEstado()
	if err != nil {
		t.Fatalf("listarHandoffsAutonomiaEstado: %v", err)
	}
	if got != 2 {
		t.Fatalf("deberia contar solo handoffs abiertos: %d", got)
	}
}

func TestStatusSnapshotNeedsImmediateRefreshConAutonomiaEnTransicion(t *testing.T) {
	t.Parallel()

	if !statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		Autonomia: autonomiaResumen{ContinuidadPendiente: 1},
	}) {
		t.Fatal("deberia refrescar enseguida con continuidad pendiente")
	}
	if !statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		Autonomia: autonomiaResumen{Handoffs: 1},
	}) {
		t.Fatal("deberia refrescar enseguida con handoff vivo")
	}
	if statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		Autonomia:      autonomiaResumen{WorkConfirmed: 1},
		AgentesActivos: []*db.Agente{{Nombre: "Codex1"}},
	}) {
		t.Fatal("no deberia forzar refresh inmediato solo por autonomia confirmada")
	}
	if statusSnapshotNeedsImmediateRefresh(apiStatusResponse{
		AgentesQuotaBlocked: []*db.Agente{{Nombre: "Codex1"}},
		TareasEnProgreso:    []tareaLite{{ID: 1, Estado: db.TareaEnProgreso}},
	}) {
		t.Fatal("no deberia forzar refresh inmediato si lo unico vivo es cuota bloqueada")
	}
}

func ptrTimeStatusDispatch(t time.Time) *time.Time { return &t }
