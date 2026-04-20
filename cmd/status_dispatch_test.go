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

func TestResumirAutonomiaRowsNoCuentaResumePayloadAbsorbidoComoPendiente(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	rows := []agentesapp.Row{
		{
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
	if got.ContinuidadPendiente != 1 {
		t.Fatalf("continuidad pendiente deberia contar solo deuda viva: %+v", got)
	}
}

func TestResumirAutonomiaRowsNoCuentaWorkQueueYaEnRunningComoPendiente(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
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
			MailboxContinuityPending: 1,
			OpenTasks:                1,
			WorkerAlive:              true,
			WorkerHeartbeat:          ptrTimeStatusDispatch(now),
		},
	}

	got := resumirAutonomiaRows(rows, now)
	if got.Continuando != 2 {
		t.Fatalf("resumen autonomia inesperado: %+v", got)
	}
	if got.ContinuidadPendiente != 1 {
		t.Fatalf("solo la work_queue no absorbida deberia contar como pendiente: %+v", got)
	}
}

func ptrTimeStatusDispatch(t time.Time) *time.Time { return &t }
