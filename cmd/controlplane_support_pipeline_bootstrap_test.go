package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestProcesarRuntimeMailboxBatchConsumePipelineLocalBootstrapOnlyConProgresoReal(t *testing.T) {
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	workerDir := filepath.Join(tmp, "worker-gemini-bootstrap-progress")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	startedAt := now.Add(-25 * time.Minute)
	progressAt := now.Add(-1 * time.Minute)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-gemini1-bootstrap-progress",
		"tmux_pane_id":          "%8",
		"mailbox_delivery_mode": "bootstrap_only",
		"can_send_input":        false,
		"started_at":            startedAt.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            progressAt.Format(time.RFC3339Nano),
		"last_output_at":        progressAt.Format(time.RFC3339Nano),
		"last_progress_at":      progressAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": "bootstrap_only",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     progressAt.Format(time.RFC3339Nano),
		"started_at":       startedAt.Format(time.RFC3339Nano),
		"last_output_at":   progressAt.Format(time.RFC3339Nano),
		"last_progress_at": progressAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-bootstrap-progress","tmux_pane_id":"%%8","can_send_input":false,"mailbox_delivery_mode":"bootstrap_only","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', estado='activo', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane: runtime mailbox/session_resume",
		Descripcion: "Frente premium acotado.",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
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

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente: "server",
		ToAgente:   "Gemini1",
		ProyectoID: &proyectoID,
		Kind:       "pipeline_local",
		PayloadJSON: fmt.Sprintf(
			`{"accion":"implementar","kind":"pipeline_local","source":"pipeline_local","tarea_objetivo_id":%d,"texto":"continua slice actual"}`,
			tareaID,
		),
	})
	if err != nil {
		t.Fatalf("mailbox pipeline_local: %v", err)
	}
	msgCreatedAt := now.Add(-5 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, msgCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at mailbox: %v", err)
	}

	payloadJSON := fmt.Sprintf(`{"mailbox_id":%d,"mailbox_kind":"pipeline_local","to_agente":"Gemini1","delivery_attempt_signature":"bootstrap_only|handle:%d|session:"}`, msgID, handle.ID)
	resultadoJSON := `{"deferred":true,"deferred_reason":"runtime_handle_bootstrap_only_mailbox_only:pipeline_local","delivery_state":"queued","dispatch_state":"pending","mailbox_only":true}`
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Gemini1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payloadJSON,
		ResultadoJSON: resultadoJSON,
		Estado:        "pendiente",
		AvailableAt:   now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction pendiente: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente una pipeline_local bootstrap-only con progreso real, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("reload mailbox: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("la mailbox deberia quedar consumida, got=%+v", msg)
	}

	order, err := db.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("reload runtime order: %+v err=%v", order, err)
	}
	if !strings.EqualFold(strings.TrimSpace(order.Estado), "completada") {
		t.Fatalf("la send_instruction deberia quedar completada, got=%+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"receipt_source":"tmux_worker_progress"`) {
		t.Fatalf("faltaba receipt_source de progreso tmux: %s", order.ResultadoJSON)
	}
	if !strings.Contains(order.ResultadoJSON, `"delivery_state":"delivered"`) {
		t.Fatalf("faltaba delivery_state delivered: %s", order.ResultadoJSON)
	}
}

func TestProcesarRuntimeMailboxBatchConsumePipelineLocalBootstrapOnlyConOutputPosterior(t *testing.T) {
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	workerDir := filepath.Join(tmp, "worker-gemini-bootstrap-output")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	startedAt := now.Add(-25 * time.Minute)
	progressAt := now.Add(-10 * time.Minute)
	outputAt := now.Add(-1 * time.Minute)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-gemini1-bootstrap-output",
		"tmux_pane_id":          "%9",
		"mailbox_delivery_mode": "bootstrap_only",
		"can_send_input":        false,
		"started_at":            startedAt.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            outputAt.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"last_progress_at":      progressAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": "bootstrap_only",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     outputAt.Format(time.RFC3339Nano),
		"started_at":       startedAt.Format(time.RFC3339Nano),
		"last_output_at":   outputAt.Format(time.RFC3339Nano),
		"last_progress_at": progressAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-bootstrap-output","tmux_pane_id":"%%9","can_send_input":false,"mailbox_delivery_mode":"bootstrap_only","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', estado='activo', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane: runtime mailbox/session_resume",
		Descripcion: "Frente premium acotado.",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
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

	msgCreatedAt := now.Add(-5 * time.Minute)
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente: "server",
		ToAgente:   "Gemini1",
		ProyectoID: &proyectoID,
		Kind:       "pipeline_local",
		PayloadJSON: fmt.Sprintf(
			`{"accion":"implementar","kind":"pipeline_local","source":"pipeline_local","tarea_objetivo_id":%d,"texto":"continua slice actual"}`,
			tareaID,
		),
	})
	if err != nil {
		t.Fatalf("mailbox pipeline_local: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, msgCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at mailbox: %v", err)
	}

	payloadJSON := fmt.Sprintf(`{"mailbox_id":%d,"mailbox_kind":"pipeline_local","to_agente":"Gemini1","delivery_attempt_signature":"bootstrap_only|handle:%d|session:"}`, msgID, handle.ID)
	resultadoJSON := `{"deferred":true,"deferred_reason":"runtime_handle_bootstrap_only_mailbox_only:pipeline_local","delivery_state":"queued","dispatch_state":"pending","mailbox_only":true}`
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Gemini1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payloadJSON,
		ResultadoJSON: resultadoJSON,
		Estado:        "pendiente",
		AvailableAt:   now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction pendiente: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar pipeline_local bootstrap-only con output posterior, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("reload mailbox: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("la mailbox deberia quedar consumida, got=%+v", msg)
	}

	order, err := db.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("reload runtime order: %+v err=%v", order, err)
	}
	if !strings.EqualFold(strings.TrimSpace(order.Estado), "completada") {
		t.Fatalf("la send_instruction deberia quedar completada, got=%+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"receipt_source":"tmux_worker_progress"`) {
		t.Fatalf("faltaba receipt_source de worker tmux: %s", order.ResultadoJSON)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeAutonomiaConOutputPosteriorEnTareaActiva(t *testing.T) {
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	workerDir := filepath.Join(tmp, "worker-gemini-autonomia-output")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	startedAt := now.Add(-25 * time.Minute)
	progressAt := now.Add(-10 * time.Minute)
	outputAt := now.Add(-1 * time.Minute)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-gemini1-autonomia-output",
		"tmux_pane_id":          "%10",
		"mailbox_delivery_mode": "bootstrap_only",
		"can_send_input":        false,
		"started_at":            startedAt.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            outputAt.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"last_progress_at":      progressAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": "bootstrap_only",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     outputAt.Format(time.RFC3339Nano),
		"started_at":       startedAt.Format(time.RFC3339Nano),
		"last_output_at":   outputAt.Format(time.RFC3339Nano),
		"last_progress_at": progressAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-autonomia-output","tmux_pane_id":"%%10","can_send_input":false,"mailbox_delivery_mode":"bootstrap_only","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', estado='activo', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane: runtime mailbox/session_resume",
		Descripcion: "Frente premium acotado.",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
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

	msgCreatedAt := now.Add(-5 * time.Minute)
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente: "server",
		ToAgente:   "Gemini1",
		ProyectoID: &proyectoID,
		Kind:       "autonomia",
		PayloadJSON: fmt.Sprintf(
			`{"accion":"continuar_trabajo","kind":"autonomia","tarea_id":%d,"texto":"sigue"}`,
			tareaID,
		),
	})
	if err != nil {
		t.Fatalf("mailbox autonomia: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, msgCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia consumir autonomia con output posterior en tarea activa, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("reload mailbox: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("la mailbox autonomia deberia quedar consumida, got=%+v", msg)
	}
}
