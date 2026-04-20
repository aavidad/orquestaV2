package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/runtimeagente"
)

func TestReconciliarRuntimeOrderSendInstructionMailboxEntregadoAbsorbidaPorTrabajoActivo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	agente := "Codex1"
	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Relevo absorbido por trabajo activo",
		Estado:     TareaEnProgreso,
		Prioridad:  "alta",
		ProyectoID: &proyectoID,
		Agente:     &agente,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET estado='en_progreso', agente=? WHERE id=?`, agente, tareaID); err != nil {
		t.Fatalf("forzar tarea activa: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      agente,
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}

	workerDir := filepath.Join(tmp, "worker-codex1-reassign-receipt")
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
	now := time.Now().UTC()
	previousActivity := now.Add(-2 * time.Minute)
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
		"last_progress_at": previousActivity.Format(time.RFC3339Nano),
		"last_output_at":   previousActivity.Format(time.RFC3339Nano),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339Nano),
		"last_progress_at": previousActivity.Format(time.RFC3339Nano),
		"last_output_at":   previousActivity.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-reassign","tmux_pane_id":"%%81","mailbox_delivery_mode":"%s","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`,
		runtimeagente.MailboxDeliverySessionResume, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=?, updated_at=?, last_seen_at=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"session_resume"}`, metaJSON, now, now, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente: "server",
		ToAgente:   agente,
		ProyectoID: &proyectoID,
		Kind:       "autonomia",
		PayloadJSON: fmt.Sprintf(`{"accion":"continuar_trabajo","kind":"autonomia","post_remediation":true,"remediation_kind":"reassign","tarea_id":%d,"to_agente":"%s","verification_key":"reassign:%d:CodexBudget:%s"}`,
			tareaID, agente, tareaID, agente),
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}

	payloadJSON := fmt.Sprintf(`{"from_agente":"server","to_agente":"%s","kind":"autonomia","accion":"continuar_trabajo","mailbox_id":%d,"mailbox_kind":"autonomia","post_remediation":true,"remediation_kind":"reassign","origin_agent":"CodexBudget","tarea_id":%d,"verification_key":"reassign:%d:CodexBudget:%s"}`,
		agente, mailboxID, tareaID, tareaID, agente)
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      agente,
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		RuntimeID:   handle.RuntimeID,
		Tipo:        "send_instruction",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get order: %+v err=%v", order, err)
	}
	payload := mapFromJSON(payloadJSON)
	if err := retenerRuntimeOrderSendInstructionNotificada(order, payload, "mailbox notificado pendiente de recibo", msg.DeliveredAt.UTC()); err != nil {
		t.Fatalf("retener notified: %v", err)
	}
	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}

	handled, err := reconciliarRuntimeOrderSendInstructionMailboxEntregado(order, payload, msg, now)
	if err != nil {
		t.Fatalf("reconciliar mailbox entregado: %v", err)
	}
	if !handled {
		t.Fatalf("la reconciliacion deberia absorber la send_instruction por trabajo activo")
	}

	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload final order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("estado final inesperado: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"superseded":true`) {
		t.Fatalf("resultado sin superseded: %s", order.ResultadoJSON)
	}
	if !strings.Contains(order.ResultadoJSON, `"superseded_reason":"mailbox_instruction_absorbed_by_active_task:`) {
		t.Fatalf("resultado sin razon de absorcion: %s", order.ResultadoJSON)
	}
	msg, err = GetRuntimeMailbox(mailboxID)
	if err != nil {
		t.Fatalf("reload mailbox: %v", err)
	}
	if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("mailbox deberia quedar consumida: %+v", msg)
	}
}
