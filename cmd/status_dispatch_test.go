package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/internal/controlruntime"
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
			Asignacion:                &db.Asignacion{Estado: db.AsignacionActiva, Nota: "server_autobootstrap"},
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
			Agente:              &db.Agente{Nombre: "Codex1"},
			Asignacion:          &db.Asignacion{Estado: db.AsignacionActiva, Nota: "server_autobootstrap"},
			LastAutonomyAction:  "continuar_trabajo",
			LastAutonomySource:  "work_queue",
			LastAutonomyState:   "work_confirmed",
			OpenTasks:           1,
			WorkerAlive:         true,
			WorkerHeartbeat:     ptrTimeStatusDispatch(now),
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
	if got.Supervisando != 1 || got.Continuando != 1 {
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

func ptrTimeStatusDispatch(t time.Time) *time.Time { return &t }
