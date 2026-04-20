package cmd

import (
	"encoding/json"
	"testing"
	"time"

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
